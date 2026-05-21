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
	candidates, err := findSimilarNotes(opts.VaultPath, incomingTokens)
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
	warnings := resolveWikilinks(opts.VaultPath, opts.Body)

	// Step 6: Generate path with collision handling.
	slug := Slug(opts.Title)
	path, err := resolveInboxPath(opts.VaultPath, today, slug)
	if err != nil {
		return WriteResult{}, fmt.Errorf("resolve inbox path: %w", err)
	}

	// Step 7: Serialize and write.
	data, err := Serialize(n)
	if err != nil {
		return WriteResult{}, fmt.Errorf("serialize note: %w", err)
	}

	inboxDir := filepath.Join(opts.VaultPath, "_inbox")
	if err := os.MkdirAll(inboxDir, 0o755); err != nil {
		return WriteResult{}, fmt.Errorf("create _inbox dir: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return WriteResult{}, fmt.Errorf("write note file: %w", err)
	}

	// Step 8: Append log entry.
	if err := appendLog(opts.VaultPath, opts.EpistemicType, slug, sourceAgent); err != nil {
		return WriteResult{}, fmt.Errorf("append log: %w", err)
	}

	return WriteResult{
		Status:   "written",
		Path:     path,
		Warnings: warnings,
	}, nil
}

// findSimilarNotes scans _inbox/ and notes/ for .md files and returns any
// whose title Jaccard similarity against incomingTokens is ≥ 0.7.
func findSimilarNotes(vaultPath string, incomingTokens []string) ([]SimilarNote, error) {
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

// resolveWikilinks extracts wikilinks from body and returns a warning for each
// that cannot be resolved to an existing vault file.
func resolveWikilinks(vaultPath, body string) []string {
	links := ExtractWikilinks(body)
	var warnings []string

	for _, link := range links {
		slug := Slug(link)

		// Check notes/{slug}.md
		notesPath := filepath.Join(vaultPath, "notes", slug+".md")
		if _, err := os.Stat(notesPath); err == nil {
			continue
		}

		// Check _inbox/*-{slug}.md (any file ending with -{slug}.md)
		inboxDir := filepath.Join(vaultPath, "_inbox")
		entries, err := os.ReadDir(inboxDir)
		found := false
		if err == nil {
			for _, e := range entries {
				if strings.HasSuffix(e.Name(), "-"+slug+".md") {
					found = true
					break
				}
			}
		}
		if found {
			continue
		}

		warnings = append(warnings, fmt.Sprintf("unresolved wikilink: [[%s]]", link))
	}

	return warnings
}

// resolveInboxPath returns the full path for a new inbox file, appending -2,
// -3, etc. if the base name already exists.
func resolveInboxPath(vaultPath, date, slug string) (string, error) {
	inboxDir := filepath.Join(vaultPath, "_inbox")
	base := date + "-" + slug
	candidate := filepath.Join(inboxDir, base+".md")

	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate, nil
	}

	for i := 2; i <= 999; i++ {
		candidate = filepath.Join(inboxDir, fmt.Sprintf("%s-%d.md", base, i))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate, nil
		}
	}

	return "", fmt.Errorf("could not find available path for slug %q", slug)
}

// appendLog appends a structured log entry to _meta/log.md, creating the file
// and directory if they do not exist.
func appendLog(vaultPath, epistemicType, slug, sourceAgent string) error {
	metaDir := filepath.Join(vaultPath, "_meta")
	if err := os.MkdirAll(metaDir, 0o755); err != nil {
		return fmt.Errorf("create _meta dir: %w", err)
	}

	logPath := filepath.Join(metaDir, "log.md")
	entry := fmt.Sprintf("\n## %s write | %s | %s | by:%s\n",
		time.Now().Format(time.RFC3339),
		epistemicType,
		slug,
		sourceAgent,
	)

	f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}
	defer f.Close() //nolint:errcheck // best-effort close on append-only file

	if _, err := f.WriteString(entry); err != nil {
		return fmt.Errorf("write log entry: %w", err)
	}

	return nil
}
