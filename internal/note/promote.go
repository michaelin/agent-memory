package note

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// PromoteResult describes the outcome of a Promote call.
type PromoteResult struct {
	Status        string `json:"status"`                   // "promoted" or "error"
	Slug          string `json:"slug"`
	From          string `json:"from,omitempty"`
	To            string `json:"to,omitempty"`
	EpistemicType string `json:"epistemic_type,omitempty"`
	Confirmed     bool   `json:"confirmed"`
	Error         string `json:"error,omitempty"`
}

// FindBySlug scans dir for a note file matching slug.
//
// For inbox-style directories (those whose path contains "_inbox"), it matches
// files whose name ends with "-{slug}.md". For all other directories it
// matches files named exactly "{slug}.md".
//
// Returns the full path on a single match, an error containing "not found"
// when no file matches, and an error containing "ambiguous" when more than one
// file matches.
func FindBySlug(dir, slug string) (string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", fmt.Errorf("read dir %s: %w", dir, err)
	}

	isInbox := strings.Contains(dir, "_inbox")

	var matches []string
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		var matched bool
		if isInbox {
			matched = strings.HasSuffix(e.Name(), "-"+slug+".md")
		} else {
			matched = e.Name() == slug+".md"
		}
		if matched {
			matches = append(matches, filepath.Join(dir, e.Name()))
		}
	}

	switch len(matches) {
	case 0:
		return "", fmt.Errorf("slug %q not found in %s", slug, dir)
	case 1:
		return matches[0], nil
	default:
		return "", fmt.Errorf("slug %q is ambiguous in %s: %d files match", slug, dir, len(matches))
	}
}

// Promote moves a note from _inbox/ to notes/, updating its status to
// "verified". It enforces lint, wikilink resolution, and human-review
// confirmation gates before writing.
//
// confirmed must be true when the note's RequiresHumanReview field is set;
// for all other notes the confirmed flag is silently ignored.
func Promote(vaultPath, slug string, confirmed bool) (PromoteResult, error) {
	errResult := func(err error) (PromoteResult, error) {
		return PromoteResult{Status: "error", Slug: slug, Error: err.Error()}, err
	}

	// Step 1: locate the inbox file.
	inboxDir := filepath.Join(vaultPath, "_inbox")
	inboxPath, err := FindBySlug(inboxDir, slug)
	if err != nil {
		return errResult(err)
	}

	// Step 2: read and parse.
	data, err := os.ReadFile(inboxPath)
	if err != nil {
		return errResult(fmt.Errorf("read inbox file: %w", err))
	}
	n, err := Parse(data)
	if err != nil {
		return errResult(fmt.Errorf("parse inbox file: %w", err))
	}

	// Step 3: lint gate.
	lr := Lint(n)
	if !lr.Valid {
		msgs := make([]string, len(lr.Errors))
		for i, e := range lr.Errors {
			msgs[i] = e.Rule + ": " + e.Message
		}
		return errResult(fmt.Errorf("lint errors: %s", strings.Join(msgs, "; ")))
	}

	// Step 4: wikilink resolution gate.
	warnings := ResolveWikilinks(vaultPath, n.Body)
	if len(warnings) > 0 {
		return errResult(fmt.Errorf("unresolved wikilinks: %s", strings.Join(warnings, "; ")))
	}

	// Step 5 & 6: human-review confirmation gate.
	if n.Frontmatter.RequiresHumanReview && !confirmed {
		return errResult(fmt.Errorf("confirmation required for constraint/decision notes; pass --confirmed"))
	}

	notesPath := filepath.Join(vaultPath, "notes", slug+".md")

	// Step 8: update frontmatter.
	n.Frontmatter.Status = "verified"
	n.Frontmatter.Updated = time.Now().Format("2006-01-02")

	// Step 9: serialize and write to notes/.
	serialized, err := Serialize(n)
	if err != nil {
		return errResult(fmt.Errorf("serialize note: %w", err))
	}
	if err := os.MkdirAll(filepath.Join(vaultPath, "notes"), 0o755); err != nil {
		return errResult(fmt.Errorf("create notes dir: %w", err))
	}
	f, err := os.OpenFile(notesPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o644)
	if err != nil {
		if os.IsExist(err) {
			return errResult(fmt.Errorf("collision: %s already exists in notes/", slug))
		}
		return errResult(fmt.Errorf("write notes file: %w", err))
	}
	if _, err := f.Write(serialized); err != nil {
		f.Close()
		_ = os.Remove(notesPath)
		return errResult(fmt.Errorf("write notes file: %w", err))
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(notesPath)
		return errResult(fmt.Errorf("write notes file: %w", err))
	}

	// Step 10: remove inbox file.
	if err := os.Remove(inboxPath); err != nil {
		// Rollback: remove the notes file we just wrote.
		_ = os.Remove(notesPath)
		return errResult(fmt.Errorf("remove inbox file: %w", err))
	}

	// Step 11: append log entry (best-effort — move already succeeded).
	if err := AppendLog(vaultPath, "promote", n.Frontmatter.EpistemicType, slug, "librarian"); err != nil {
		fmt.Fprintf(os.Stderr, "warn: append log failed for promote %s: %v\n", slug, err)
	}

	// Step 12: return success.
	return PromoteResult{
		Status:        "promoted",
		Slug:          slug,
		From:          inboxPath,
		To:            notesPath,
		EpistemicType: n.Frontmatter.EpistemicType,
		Confirmed:     confirmed,
	}, nil
}
