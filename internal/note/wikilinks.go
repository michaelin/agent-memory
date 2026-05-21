package note

import (
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
	"go.abhg.dev/goldmark/wikilink"
)

// ExtractWikilinks returns all unique wikilink slugs found in text.
// Matches [[slug]] and [[slug|alias]] — extracts slug only.
// Results are deduplicated and returned in order of first appearance.
// Wikilinks inside fenced code blocks, indented code blocks, and inline
// code spans are ignored.
func ExtractWikilinks(content string) []string {
	src := []byte(content)
	reader := text.NewReader(src)

	md := goldmark.New(goldmark.WithExtensions(&wikilink.Extender{}))
	doc := md.Parser().Parse(reader)

	seen := make(map[string]struct{})
	var result []string

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		wn, ok := n.(*wikilink.Node)
		if !ok {
			return ast.WalkContinue, nil
		}

		target := string(wn.Target)
		// Handle |alias syntax: take only the part before |.
		// (The goldmark-wikilink parser already separates target from label,
		// but guard against any raw pipe that may remain.)
		target = strings.SplitN(target, "|", 2)[0]

		if _, exists := seen[target]; !exists {
			seen[target] = struct{}{}
			result = append(result, target)
		}

		return ast.WalkContinue, nil
	}) //nolint:errcheck // visitor never returns an error

	return result
}
