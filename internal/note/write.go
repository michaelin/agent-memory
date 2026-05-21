package note

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// WriteOptions holds all parameters for writing a note to the vault.
type WriteOptions struct {
	// VaultPath is the absolute path to the vault root directory.
	VaultPath string

	// Body is the raw markdown body content of the note (without frontmatter).
	Body string

	// Frontmatter fields supplied by the caller.
	Type           string   // epistemic-type
	Title          string   // required
	Project        string   // optional
	Domain         []string // required (non-empty)
	Scope          string   // required
	SourceArtifact string   // required
	SourceAgent    string   // optional
	Confidence     string   // required
	Tags           []string // optional

	// Force bypasses similarity refusal.
	Force bool
}

// WriteStatus describes the outcome of a Write call.
type WriteStatus string

const (
	// WriteStatusWritten means the note was successfully written to the vault.
	WriteStatusWritten WriteStatus = "written"
	// WriteStatusRefused means the note was refused due to similarity with an
	// existing note.
	WriteStatusRefused WriteStatus = "refused"
)

// Candidate is a similar existing note that caused a refusal.
type Candidate struct {
	// Path is the vault-relative path of the similar note.
	Path string `json:"path"`
	// Score is the Jaccard similarity score (0.0–1.0).
	Score float64 `json:"score"`
}

// WriteResult is the outcome of a Write call.
type WriteResult struct {
	// Status is "written" or "refused".
	Status WriteStatus `json:"status"`
	// Path is the vault-relative path of the written note (only set when
	// Status is WriteStatusWritten).
	Path string `json:"path,omitempty"`
	// Reason explains why the note was refused (only set when Status is
	// WriteStatusRefused).
	Reason string `json:"reason,omitempty"`
	// Candidates lists similar existing notes that triggered a refusal.
	Candidates []Candidate `json:"candidates,omitempty"`
}

// similarityThreshold is the Jaccard score above which a note is considered
// too similar to an existing note.
const similarityThreshold = 0.7

// Write assembles a Note from opts, lints it, checks for similarity against
// existing notes in the vault, and writes it to <vault>/notes/<slug>.md.
//
// If a similar note exists and opts.Force is false, Write returns a
// WriteStatusRefused result without writing any file.
//
// If opts.Force is true, similarity checking is skipped and the note is
// written unconditionally.
func Write(opts WriteOptions) (*WriteResult, error) {
	today := time.Now().Format("2006-01-02")

	n := &Note{
		Frontmatter: Frontmatter{
			Title:          opts.Title,
			Created:        today,
			Updated:        today,
			Status:         "inbox",
			Confidence:     opts.Confidence,
			EpistemicType:  opts.Type,
			Scope:          opts.Scope,
			Project:        opts.Project,
			Domain:         opts.Domain,
			SourceAgent:    opts.SourceAgent,
			SourceArtifact: opts.SourceArtifact,
			Tags:           opts.Tags,
		},
		Body: opts.Body,
	}

	result := Lint(n)
	if !result.Valid {
		msgs := make([]string, len(result.Errors))
		for i, e := range result.Errors {
			msgs[i] = fmt.Sprintf("%s: %s", e.Rule, e.Message)
		}
		return nil, fmt.Errorf("note failed lint: %s", strings.Join(msgs, "; "))
	}

	if !opts.Force {
		candidates, err := findSimilar(opts.VaultPath, n)
		if err != nil {
			return nil, fmt.Errorf("similarity check: %w", err)
		}
		if len(candidates) > 0 {
			return &WriteResult{
				Status:     WriteStatusRefused,
				Reason:     "note is too similar to existing notes",
				Candidates: candidates,
			}, nil
		}
	}

	data, err := Serialize(n)
	if err != nil {
		return nil, fmt.Errorf("serialize note: %w", err)
	}

	slug := Slug(opts.Title)
	if slug == "" {
		slug = "untitled"
	}
	relPath := filepath.Join("notes", slug+".md")
	absPath := filepath.Join(opts.VaultPath, relPath)

	if err := os.WriteFile(absPath, data, 0600); err != nil {
		return nil, fmt.Errorf("write note file: %w", err)
	}

	return &WriteResult{
		Status: WriteStatusWritten,
		Path:   relPath,
	}, nil
}

// findSimilar scans all .md files under <vault>/notes/ and returns any whose
// title+body Jaccard similarity with n exceeds similarityThreshold.
func findSimilar(vaultPath string, n *Note) ([]Candidate, error) {
	notesDir := filepath.Join(vaultPath, "notes")
	entries, err := os.ReadDir(notesDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("reading notes directory: %w", err)
	}

	incomingText := n.Frontmatter.Title + " " + n.Body
	incomingTokens := NormalizeTokens(incomingText)

	var candidates []Candidate
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") {
			continue
		}

		absPath := filepath.Join(notesDir, entry.Name())
		data, err := os.ReadFile(absPath)
		if err != nil {
			// Skip unreadable files rather than aborting the whole check.
			continue
		}

		existing, err := Parse(data)
		if err != nil {
			continue
		}

		existingText := existing.Frontmatter.Title + " " + existing.Body
		existingTokens := NormalizeTokens(existingText)

		score := Jaccard(incomingTokens, existingTokens)
		if score >= similarityThreshold {
			candidates = append(candidates, Candidate{
				Path:  filepath.Join("notes", entry.Name()),
				Score: score,
			})
		}
	}

	return candidates, nil
}
