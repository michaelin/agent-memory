package vault

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// InitOptions controls vault initialization behavior.
type InitOptions struct {
	Force bool
	Clean bool
}

// InitResult is returned by Init, serialized to JSON for CLI output.
type InitResult struct {
	Status   string   `json:"status"`
	Vault    string   `json:"vault"`
	Repaired []string `json:"repaired,omitempty"`
}

// InitError is the error result, serialized to JSON for CLI output.
type InitError struct {
	Error string `json:"error"`
}

// Init scaffolds or repairs a vault at the given path.
func Init(vaultPath string, opts InitOptions) (*InitResult, error) {
	if strings.TrimSpace(vaultPath) == "" {
		return nil, fmt.Errorf("vault path must not be empty")
	}
	absPath, err := filepath.Abs(vaultPath)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	if opts.Clean && !opts.Force {
		return nil, fmt.Errorf("--clean requires --force")
	}
	if opts.Clean && opts.Force {
		// TODO: This heuristic is Unix-centric. Harden for Windows drive roots
		// and UNC paths if cross-platform support is added.
		cleanPath := filepath.Clean(absPath)
		if cleanPath == "" || filepath.Dir(cleanPath) == cleanPath {
			return nil, fmt.Errorf("refusing to remove root filesystem path")
		}
		if err := os.RemoveAll(absPath); err != nil {
			return nil, fmt.Errorf("removing vault: %w", err)
		}
	}

	info, err := os.Lstat(absPath)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("checking vault path: %w", err)
	}

	if info != nil {
		// Reject symlinks at vault root — caller must resolve them first.
		if info.Mode()&os.ModeSymlink != 0 {
			return nil, fmt.Errorf("vault path %s is a symlink; resolve it before running init", absPath)
		}
		if !info.IsDir() {
			if opts.Force {
				if err := os.Remove(absPath); err != nil {
					return nil, fmt.Errorf("removing file at vault root: %w", err)
				}
				info = nil
			} else {
				return nil, fmt.Errorf("vault path %s exists as a file, not a directory", absPath)
			}
		}
	}

	if err := os.MkdirAll(absPath, 0700); err != nil {
		return nil, fmt.Errorf("creating vault root: %w", err)
	}

	var repaired []string
	isExisting := info != nil

	for _, entry := range VaultStructure() {
		entryPath := filepath.Join(absPath, entry.Path)
		entryInfo, err := os.Lstat(entryPath)

		if err != nil && !os.IsNotExist(err) {
			return nil, fmt.Errorf("checking %s: %w", entry.Path, err)
		}

		if entryInfo != nil {
			// Reject or remove symlinks at child paths.
			if entryInfo.Mode()&os.ModeSymlink != 0 {
				if opts.Force {
					if err := os.Remove(entryPath); err != nil {
						return nil, fmt.Errorf("removing symlink %s: %w", entry.Path, err)
					}
					entryInfo = nil
				} else {
					return nil, fmt.Errorf("vault path %s is a symlink; resolve it or use --force", entry.Path)
				}
			}
		}
		if entryInfo != nil {
			if entry.IsDir && !entryInfo.IsDir() {
				if opts.Force {
					if err := os.Remove(entryPath); err != nil {
						return nil, fmt.Errorf("removing conflicting file %s: %w", entry.Path, err)
					}
					if err := os.MkdirAll(entryPath, 0700); err != nil {
						return nil, fmt.Errorf("creating dir %s: %w", entry.Path, err)
					}
					repaired = append(repaired, entry.Path)
				} else {
					return nil, fmt.Errorf("vault path %s exists as a file, not a directory", entry.Path)
				}
			} else if !entry.IsDir && entryInfo.IsDir() {
				if opts.Force {
					if err := os.RemoveAll(entryPath); err != nil {
						return nil, fmt.Errorf("removing conflicting dir %s: %w", entry.Path, err)
					}
					if err := writeTemplate(entryPath, entry.Template); err != nil {
						return nil, fmt.Errorf("writing %s: %w", entry.Path, err)
					}
					repaired = append(repaired, entry.Path)
				} else {
					return nil, fmt.Errorf("vault path %s exists as a directory, not a file", entry.Path)
				}
			}
			// Entry exists and is correct type — skip (idempotent)
			continue
		}

		// Entry missing — create it
		if entry.IsDir {
			if err := os.MkdirAll(entryPath, 0700); err != nil {
				return nil, fmt.Errorf("creating dir %s: %w", entry.Path, err)
			}
		} else {
			if err := writeTemplate(entryPath, entry.Template); err != nil {
				return nil, fmt.Errorf("writing %s: %w", entry.Path, err)
			}
		}
		if isExisting {
			repaired = append(repaired, entry.Path)
		}
	}

	status := "created"
	if isExisting {
		if len(repaired) > 0 {
			status = "repaired"
		} else {
			status = "ok"
		}
	}

	// Only append log entry on actual changes (not idempotent no-ops).
	if status != "ok" {
		logPath := filepath.Join(absPath, "_meta", "log.md")
		timestamp := time.Now().Format(time.RFC3339)
		logEntry := fmt.Sprintf("\n## [%s] init | vault initialized\n", timestamp)
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_WRONLY|os.O_CREATE, 0600)
		if err != nil {
			return nil, fmt.Errorf("opening log: %w", err)
		}
		defer f.Close()
		if _, err := f.WriteString(logEntry); err != nil {
			return nil, fmt.Errorf("writing log: %w", err)
		}
	}

	return &InitResult{
		Status:   status,
		Vault:    absPath,
		Repaired: repaired,
	}, nil
}

func writeTemplate(destPath, templateName string) error {
	data, err := templateFS.ReadFile(templateName)
	if err != nil {
		return fmt.Errorf("reading template %s: %w", templateName, err)
	}
	return os.WriteFile(destPath, data, 0600)
}

// MarshalError marshals an error to JSON InitError format.
func MarshalError(err error) []byte {
	b, _ := json.Marshal(InitError{Error: err.Error()})
	return b
}
