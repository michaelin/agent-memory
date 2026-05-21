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
//
// CRLF line endings are normalized to LF before parsing. Delimiter lines are
// matched exactly (after trimming trailing whitespace) so that "---notyaml"
// is not treated as a valid delimiter.
func Parse(content []byte) (*Note, error) {
	// Normalize CRLF to LF so Windows-style line endings parse identically.
	s := strings.ReplaceAll(string(content), "\r\n", "\n")

	lines := strings.Split(s, "\n")

	// The first line must be exactly "---" (after trimming trailing whitespace).
	if len(lines) == 0 || strings.TrimRight(lines[0], " \t") != delimiter {
		return nil, ErrNoFrontmatter
	}

	// Find the closing delimiter: a line that is exactly "---" (after trimming
	// trailing whitespace), starting from line index 1.
	closingIdx := -1
	for i := 1; i < len(lines); i++ {
		if strings.TrimRight(lines[i], " \t") == delimiter {
			closingIdx = i
			break
		}
	}
	if closingIdx == -1 {
		return nil, ErrNoFrontmatter
	}

	yamlBlock := strings.Join(lines[1:closingIdx], "\n")
	body := strings.Join(lines[closingIdx+1:], "\n")

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
