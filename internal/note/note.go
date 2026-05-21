// Package note provides types and parsing logic for agent-memory note files.
// Notes consist of a YAML frontmatter block delimited by "---" lines followed
// by a plain-text body.
package note

import (
	"bytes"
	"errors"
	"fmt"
	"strings"

	"go.yaml.in/yaml/v3"
)

// Note represents a parsed agent-memory note file.
type Note struct {
	Frontmatter Frontmatter
	Body        string
}

// Frontmatter holds the structured metadata stored in the YAML front matter of
// a note file.
type Frontmatter struct {
	Title               string   `yaml:"title"`
	Created             string   `yaml:"created"`
	Updated             string   `yaml:"updated"`
	ReviewBy            string   `yaml:"review-by"`
	Status              string   `yaml:"status"`
	Confidence          string   `yaml:"confidence"`
	EpistemicType       string   `yaml:"epistemic-type"`
	Scope               string   `yaml:"scope"`
	Project             string   `yaml:"project"`
	Domain              []string `yaml:"domain"`
	SourceAgent         string   `yaml:"source-agent"`
	SourceArtifact      string   `yaml:"source-artifact"`
	VerifiedBy          string   `yaml:"verified-by"`
	VerifiedDate        string   `yaml:"verified-date"`
	RequiresHumanReview bool     `yaml:"requires-human-review"`
	UpdateType          string   `yaml:"update-type"`
	Targets             []string `yaml:"targets"`
	Tags                []string `yaml:"tags"`
}

// ErrNoFrontmatter is returned by Parse when the content does not contain the
// opening and closing "---" frontmatter delimiters.
var ErrNoFrontmatter = errors.New("no frontmatter found")

const delimiter = "---"

// Parse splits content into a YAML frontmatter block and a body, unmarshals
// the frontmatter, and returns a *Note. Unknown YAML fields are silently
// ignored. The body is trimmed of leading and trailing whitespace.
//
// Returns ErrNoFrontmatter if the content does not begin with a "---"
// delimiter followed by a closing "---" delimiter. Returns a wrapped error if
// the YAML is malformed.
func Parse(content []byte) (*Note, error) {
	s := string(content)

	// The file must start with "---\n" (or "---" at EOF, though that is
	// degenerate). We look for the opening delimiter on the very first line.
	if !strings.HasPrefix(s, delimiter) {
		return nil, ErrNoFrontmatter
	}

	// Advance past the opening "---".
	rest := s[len(delimiter):]
	// Consume an optional newline immediately after the opening delimiter.
	rest = strings.TrimPrefix(rest, "\n")

	// Find the closing "---".
	idx := strings.Index(rest, "\n"+delimiter)
	if idx == -1 {
		// Also handle the case where the closing delimiter is at the very start
		// (empty frontmatter with no leading newline consumed above).
		if strings.HasPrefix(rest, delimiter) {
			idx = 0
		} else {
			return nil, ErrNoFrontmatter
		}
	}

	var yamlBlock string
	var body string

	if strings.HasPrefix(rest, delimiter) {
		// Opening "---" was immediately followed by closing "---" (empty FM).
		yamlBlock = ""
		body = rest[len(delimiter):]
	} else {
		yamlBlock = rest[:idx]
		afterClose := rest[idx+1+len(delimiter):]
		body = afterClose
	}

	var fm Frontmatter
	dec := yaml.NewDecoder(bytes.NewBufferString(yamlBlock))
	if err := dec.Decode(&fm); err != nil {
		// A nil document (empty frontmatter) is not an error.
		if yamlBlock != "" {
			return nil, fmt.Errorf("parse frontmatter: %w", err)
		}
	}

	return &Note{
		Frontmatter: fm,
		Body:        strings.TrimSpace(body),
	}, nil
}
