package vault

import (
	"fmt"
	"os"
	"path/filepath"
)

// Discover resolves the vault path using the following priority order:
//
//  1. explicitPath — if non-empty, verify it exists as a directory and return
//     its absolute path.
//  2. AGENT_MEMORY_VAULT environment variable — if set, verify it exists and
//     return its absolute path.
//  3. Walk up from the current working directory looking for a child directory
//     named ".agent-memory". Return the absolute path of that child when found.
//  4. Return an error indicating no vault was found.
func Discover(explicitPath string) (string, error) {
	if explicitPath != "" {
		abs, err := filepath.Abs(explicitPath)
		if err != nil {
			return "", fmt.Errorf("resolving explicit path: %w", err)
		}
		if err := requireDir(abs); err != nil {
			return "", fmt.Errorf("explicit vault path: %w", err)
		}
		return abs, nil
	}

	if envPath := os.Getenv("AGENT_MEMORY_VAULT"); envPath != "" {
		abs, err := filepath.Abs(envPath)
		if err != nil {
			return "", fmt.Errorf("resolving AGENT_MEMORY_VAULT: %w", err)
		}
		if err := requireDir(abs); err != nil {
			return "", fmt.Errorf("AGENT_MEMORY_VAULT: %w", err)
		}
		return abs, nil
	}

	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("getting working directory: %w", err)
	}

	// Walk up the directory tree looking for a ".agent-memory" child directory.
	dir := cwd
	for {
		candidate := filepath.Join(dir, ".agent-memory")
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			return candidate, nil
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding a vault.
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("no agent-memory vault found: set AGENT_MEMORY_VAULT, pass --vault, or run from inside a vault tree")
}

// requireDir returns an error if path does not exist or is not a directory.
func requireDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("path does not exist: %s", path)
		}
		return fmt.Errorf("checking path %s: %w", path, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", path)
	}
	return nil
}
