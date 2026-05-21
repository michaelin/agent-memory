package note

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteOptions holds all inputs required to write a new note into the vault.
type WriteOptions struct {
	VaultPath      string
	Title          string
	EpistemicType  string
	Body           string
	Project        string
	Scope          string
	SourceArtifact string
	SourceAgent    string
	Confidence     string
	Domain         []string
	Tags           []string
	Force          bool
}

// WriteResult describes the outcome of a Write call.
type WriteResult struct {
	Status     string      `json:"status"` // "written" or "refused"
	Path       string      `json:"path,omitempty"`
	Reason     string      `json:"reason,omitempty"`
	Warnings   []string    `json:"warnings,omitempty"`
	Candidates []SimilarNote `json:"candidates,omitempty"`
	Errors     []LintError `json:"errors,omitempty"`
}

// SimilarNote is a vault note that scored above the similarity threshold.
type SimilarNote struct {
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Similarity float64 `json:"similarity"`
}

// ttlDays returns the review-by TTL in days for the given epistemic type.
func ttlDays(epistemicType string) int {
	switch epistemicType {
	case "observation":
		return 90
	case "pattern":
		return 180
	case "assumption":
		return 30
	case "constraint", "decision":
		return 365
	default:
		return 180
	}
}

// requiresHumanReview reports whether the epistemic type mandates human review.
func requiresHumanReview(epistemicType string) bool {
	return epistemicType == "constraint" || epistemicType == "decision"
}

// Write assembles a Note from opts, validates it, checks for similar existing
// notes, resolves wikilinks, writes the file to _inbox/, and appends a log
// entry to _meta/log.md.
//
// Returns WriteResult{Status:"refused"} when lint fails or a similar note
// exists (unless opts.Force is true).
func Write(opts WriteOptions) (WriteResult, error) {
	today := time.Now().Format("2006-01-02")

	// Step 1 & 2: Assemble note with auto-populated fields.
	sourceAgent := opts.SourceAgent
	if sourceAgent == "" {
		sourceAgent = DetectSourceAgent()
	}

	confidence := opts.Confidence
	if confidence == "" {
		confidence = "medium"
	}

	reviewBy := time.Now().AddDate(0, 0, ttlDays(opts.EpistemicType)).Format("2006-01-02")

	fm := Frontmatter{
		Title:               opts.Title,
		Created:             today,
		Updated:             today,
		Status:              "inbox",
		Confidence:          confidence,
		EpistemicType:       opts.EpistemicType,
		Scope:               opts.Scope,
		Project:             opts.Project,
		Domain:              opts.Domain,
		Tags:                opts.Tags,
		SourceAgent:         sourceAgent,
		SourceArtifact:      opts.SourceArtifact,
		ReviewBy:            reviewBy,
		RequiresHumanReview: requiresHumanReview(opts.EpistemicType),
	}

	n := &Note{
		Frontmatter: fm,
		Body:        opts.Body,
	}

	// Step 3: Lint.
	lintResult := Lint(n)
	if !lintResult.Valid {
		return WriteResult{
			Status: "refused",
			Reason: "lint errors",
			Errors: lintResult.Errors,
		}, nil
	}

	// Step 4: Similarity scan.
	incomingTokens := NormalizeTokens(opts.Title)
	candidates, err := FindSimilarNotes(opts.VaultPath, incomingTokens)
	if err != nil {
		return WriteResult{}, fmt.Errorf("similarity scan: %w", err)
	}
	if len(candidates) > 0 && !opts.Force {
		return WriteResult{
			Status:     "refused",
			Reason:     "similar note exists",
			Candidates: candidates,
		}, nil
	}

	// Step 5: Wikilink resolution warnings.
	warnings := ResolveWikilinks(opts.VaultPath, opts.Body)

	// Step 6: Generate path with collision handling.
	slug := Slug(opts.Title)

	// Ensure _inbox/ exists before resolveInboxPath attempts O_EXCL creation.
	inboxDir := filepath.Join(opts.VaultPath, "_inbox")
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("create _inbox dir: %w", err)
	}

	path, err := resolveInboxPath(opts.VaultPath, today, slug)
	if err != nil {
		return WriteResult{}, fmt.Errorf("resolve inbox path: %w", err)
	}

	// Clean up placeholder if write fails.
	writeSucceeded := false
	defer func() {
		if !writeSucceeded {
			_ = os.Remove(path)
		}
	}()

	// Step 7: Serialize and write.
	data, err := Serialize(n)
	if err != nil {
		return WriteResult{}, fmt.Errorf("serialize note: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return WriteResult{}, fmt.Errorf("write note file: %w", err)
	}

	writeSucceeded = true

	// AppendLog is best-effort; failure is logged but does not abort the operation.
	if err := AppendLog(opts.VaultPath, "write", opts.EpistemicType, slug, sourceAgent); err != nil {
		fmt.Fprintf(os.Stderr, "warn: append log failed for write %s: %v\n", slug, err)
	}

	return WriteResult{
		Status:   "written",
		Path:     path,
		Warnings: warnings,
	}, nil
}

// FindSimilarNotes scans _inbox/ and notes/ for .md files and returns any
// whose title Jaccard similarity against incomingTokens is ≥ 0.7.
func FindSimilarNotes(vaultPath string, incomingTokens []string) ([]SimilarNote, error) {
	var candidates []SimilarNote

	dirs := []string{
		filepath.Join(vaultPath, "_inbox"),
		filepath.Join(vaultPath, "notes"),
	}

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read dir %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.Type()&os.ModeSymlink != 0 {
				continue
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
				continue
			}

			fullPath := filepath.Join(dir, entry.Name())
			data, err := os.ReadFile(fullPath)
			if err != nil {
				continue
			}

			parsed, err := Parse(data)
			if err != nil || parsed == nil {
				continue
			}

			existingTokens := NormalizeTokens(parsed.Frontmatter.Title)
			sim := Jaccard(incomingTokens, existingTokens)
			if sim >= 0.7 {
				candidates = append(candidates, SimilarNote{
					Path:       fullPath,
					Title:      parsed.Frontmatter.Title,
					Similarity: sim,
				})
			}
		}
	}

	return candidates, nil
}

// ResolveWikilinks extracts wikilinks from body and returns a warning for each
// that cannot be resolved to an existing vault file. It scans _inbox/,
// notes/, and _deprecated/ (excluding _meta/).
//
// Inbox files match if the filename ends with -{slug}.md.
// Notes and deprecated files match if the filename is exactly {slug}.md.
func ResolveWikilinks(vaultPath, body string) []string {
	links := ExtractWikilinks(body)
	var warnings []string

	// Read _inbox/ entries once, outside the per-link loop.
	inboxDir := filepath.Join(vaultPath, "_inbox")
	inboxEntries, err := os.ReadDir(inboxDir)
	if err != nil && !os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("warning: could not read _inbox dir: %v", err))
		return warnings
	}

	// Read _deprecated/ entries once.
	deprecatedDir := filepath.Join(vaultPath, "_deprecated")
	deprecatedEntries, err := os.ReadDir(deprecatedDir)
	if err != nil && !os.IsNotExist(err) {
		warnings = append(warnings, fmt.Sprintf("warning: could not read _deprecated dir: %v", err))
		return warnings
	}

	for _, link := range links {
		slug := Slug(link)

		// Check notes/{slug}.md
		notesPath := filepath.Join(vaultPath, "notes", slug+".md")
		if _, err := os.Stat(notesPath); err == nil {
			continue
		}

		// Check _deprecated/{slug}.md
		found := false
		for _, e := range deprecatedEntries {
			if e.Type()&os.ModeSymlink != 0 {
				continue
			}
			if e.Name() == slug+".md" {
				found = true
				break
			}
		}
		if found {
			continue
		}

		// Check _inbox/*-{slug}.md (any file ending with -{slug}.md)
		for _, e := range inboxEntries {
			if e.Type()&os.ModeSymlink != 0 {
				continue
			}
			if strings.HasSuffix(e.Name(), "-"+slug+".md") {
				found = true
				break
			}
		}
		if found {
			continue
		}

		warnings = append(warnings, fmt.Sprintf("unresolved wikilink: [[%s]]", link))
	}

	return warnings
}

// resolveInboxPath returns the full path for a new inbox file by atomically
// creating it with O_EXCL to avoid TOCTOU races. It appends -2, -3, etc. on
// collision, up to 100 attempts.
func resolveInboxPath(vaultPath, date, slug string) (string, error) {
	inboxDir := filepath.Join(vaultPath, "_inbox")
	base := date + "-" + slug

	for i := 0; i <= 100; i++ {
		var candidate string
		if i == 0 {
			candidate = filepath.Join(inboxDir, base+".md")
		} else {
			candidate = filepath.Join(inboxDir, fmt.Sprintf("%s-%d.md", base, i+1))
		}

		f, err := os.OpenFile(candidate, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			// Successfully claimed the file; close it — Write will overwrite via WriteFile.
			_ = f.Close()
			return candidate, nil
		}
		if !os.IsExist(err) {
			return "", fmt.Errorf("probe inbox path: %w", err)
		}
	}

	return "", fmt.Errorf("could not find available path for slug %q", slug)
}

// AppendLog appends a structured log entry to _meta/log.md, creating the file
// and directory if they do not exist. The action parameter describes the
// operation being logged (e.g. "write", "promote", "deprecate").
func AppendLog(vaultPath, action, epistemicType, slug, sourceAgent string) error {
	metaDir := filepath.Join(vaultPath, "_meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return fmt.Errorf("create _meta dir: %w", err)
	}

	logPath := filepath.Join(metaDir, "log.md")
	entry := fmt.Sprintf("\n## %s %s | %s | %s | by:%s\n",
		time.Now().Format(time.RFC3339),
		action,
		epistemicType,
		slug,
		sourceAgent,
	)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	if _, err := f.WriteString(entry); err != nil {
		_ = f.Close()
		return fmt.Errorf("write log entry: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close log file: %w", err)
	}
	return nil
}
