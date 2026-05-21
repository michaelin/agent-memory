package note

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)


// DeprecateResult describes the outcome of a Deprecate call.
type DeprecateResult struct {
	Status       string `json:"status"`                  // "deprecated" or "error"
	Slug         string `json:"slug"`
	SupersededBy string `json:"superseded_by,omitempty"`
	MovedTo      string `json:"moved_to,omitempty"`
	Error        string `json:"error,omitempty"`
}

// Deprecate marks the note identified by slug as deprecated, moves it from
// notes/{slug}.md to _deprecated/{slug}.md, updates its frontmatter, and
// appends a log entry to _meta/log.md.
//
// supersededBy is the slug of the note that replaces this one; it may be empty
// if there is no direct replacement.
//
// Returns DeprecateResult{Status:"error"} together with the error on any
// failure.
func Deprecate(vaultPath, slug, supersededBy string) (DeprecateResult, error) {
	notePath := filepath.Join(vaultPath, "notes", slug+".md")

	// Step 1: Verify the source note exists.
	if _, err := os.Stat(notePath); os.IsNotExist(err) {
		err := fmt.Errorf("note not found: %s", slug)
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()}, err
	}

	// Step 2: Read and parse.
	data, err := os.ReadFile(notePath)
	if err != nil {
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()},
			fmt.Errorf("read note: %w", err)
	}

	n, err := Parse(data)
	if err != nil {
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()},
			fmt.Errorf("parse note: %w", err)
	}

	// Step 3: Update frontmatter.
	n.Frontmatter.Status = "deprecated"
	n.Frontmatter.SupersededBy = supersededBy
	n.Frontmatter.Updated = time.Now().Format("2006-01-02")

	// Step 4: Serialize.
	serialized, err := Serialize(n)
	if err != nil {
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()},
			fmt.Errorf("serialize note: %w", err)
	}

	// Step 5: Ensure _deprecated/ exists.
	deprecatedDir := filepath.Join(vaultPath, "_deprecated")
	if err := os.MkdirAll(deprecatedDir, 0o755); err != nil {
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()},
			fmt.Errorf("create _deprecated dir: %w", err)
	}

	// Step 6: Write to _deprecated/{slug}.md.
	destPath := filepath.Join(deprecatedDir, slug+".md")
	if err := os.WriteFile(destPath, serialized, 0o644); err != nil {
		return DeprecateResult{Status: "error", Slug: slug, Error: err.Error()},
			fmt.Errorf("write deprecated note: %w", err)
	}

	// Step 7: Remove the original from notes/.
	if err := os.Remove(notePath); err != nil {
		// Rollback: remove the _deprecated/ file we just wrote.
		_ = os.Remove(destPath)
		wrapped := fmt.Errorf("remove original note: %w", err)
		return DeprecateResult{Status: "error", Slug: slug, Error: wrapped.Error()},
			wrapped
	}

	// Step 8: Append log entry (best-effort — move already succeeded).
	if err := AppendLog(vaultPath, "deprecate", n.Frontmatter.EpistemicType, slug, "librarian"); err != nil {
		fmt.Fprintf(os.Stderr, "warn: append log failed for deprecate %s: %v\n", slug, err)
	}

	// Step 9: Return success.
	return DeprecateResult{
		Status:       "deprecated",
		Slug:         slug,
		SupersededBy: supersededBy,
		MovedTo:      filepath.Join(vaultPath, "_deprecated", slug+".md"),
	}, nil
}
