package note

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"go.yaml.in/yaml/v3"
)

// Serialize encodes n into the canonical note file format: a YAML frontmatter
// block delimited by "---" lines followed by the note body.
//
// Returns an error if n is nil.
func Serialize(n *Note) ([]byte, error) {
	if n == nil {
		return nil, fmt.Errorf("serialize: note must not be nil")
	}

	fm, err := yaml.Marshal(n.Frontmatter)
	if err != nil {
		return nil, fmt.Errorf("serialize frontmatter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.Write(fm)
	sb.WriteString("---\n")
	sb.WriteString(n.Body)
	sb.WriteString("\n")

	return []byte(sb.String()), nil
}

var (
	reNonAlphanumHyphen  = regexp.MustCompile(`[^a-z0-9-]`)
	reConsecutiveHyphens = regexp.MustCompile(`-{2,}`)
)

// Slug converts title into a URL- and filesystem-safe slug. The result is
// lowercase, contains only alphanumeric characters and hyphens, has no
// leading or trailing hyphens, and is at most 60 characters long.
//
// Empty input returns an empty string.
func Slug(title string) string {
	if title == "" {
		return ""
	}

	// Strip non-ASCII characters before lowercasing so that accented letters
	// (e.g. é, ö) are removed rather than transliterated.
	var ascii strings.Builder
	for _, r := range title {
		if r <= unicode.MaxASCII {
			ascii.WriteRune(r)
		}
	}

	s := strings.ToLower(ascii.String())
	s = strings.ReplaceAll(s, " ", "-")
	s = reNonAlphanumHyphen.ReplaceAllString(s, "")
	s = reConsecutiveHyphens.ReplaceAllString(s, "-")
	s = strings.Trim(s, "-")

	if s == "" {
		s = "note"
	}

	if len(s) > 60 {
		s = s[:60]
		s = strings.TrimRight(s, "-")
	}

	return s
}
