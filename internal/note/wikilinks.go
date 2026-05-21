package note

import (
	"regexp"
	"strings"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ExtractWikilinks returns all unique wikilink slugs found in text.
// Matches [[slug]] and [[slug|alias]] — extracts slug only.
// Results are deduplicated and returned in order of first appearance.
func ExtractWikilinks(text string) []string {
	matches := wikilinkRe.FindAllStringSubmatch(text, -1)
	seen := make(map[string]struct{})
	result := make([]string, 0, len(matches))
	for _, m := range matches {
		slug := strings.SplitN(m[1], "|", 2)[0]
		if _, ok := seen[slug]; ok {
			continue
		}
		seen[slug] = struct{}{}
		result = append(result, slug)
	}
	return result
}
