package note

import (
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// codeRange is a half-open byte interval [start, stop) within the source.
type codeRange struct {
	start, stop int
}

func (r codeRange) contains(pos int) bool {
	return pos >= r.start && pos < r.stop
}

// ExtractWikilinks returns all unique wikilink slugs found in text.
// Matches [[slug]] and [[slug|alias]] — extracts slug only.
// Results are deduplicated and returned in order of first appearance.
// Wikilinks inside fenced code blocks, indented code blocks, and inline
// code spans are ignored.
func ExtractWikilinks(content string) []string {
	src := []byte(content)
	reader := text.NewReader(src)
	parser := goldmark.DefaultParser()
	doc := parser.Parse(reader)

	// Collect byte ranges that belong to code contexts.
	var codeRanges []codeRange

	ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch n.Kind() {
		case ast.KindCodeBlock, ast.KindFencedCodeBlock:
			// Collect all line ranges for this block.
			lines := n.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				codeRanges = append(codeRanges, codeRange{seg.Start, seg.Stop})
			}
			return ast.WalkSkipChildren, nil

		case ast.KindCodeSpan:
			// CodeSpan children are Text nodes holding the raw span content.
			for c := n.FirstChild(); c != nil; c = c.NextSibling() {
				if tn, ok := c.(*ast.Text); ok {
					codeRanges = append(codeRanges, codeRange{tn.Segment.Start, tn.Segment.Stop})
				}
			}
			return ast.WalkSkipChildren, nil
		}

		return ast.WalkContinue, nil
	}) //nolint:errcheck // visitor never returns an error

	// Apply the regex to the full source and skip matches inside code ranges.
	locs := wikilinkRe.FindAllSubmatchIndex(src, -1)
	seen := make(map[string]struct{})
	var result []string

	for _, loc := range locs {
		// loc[0]:loc[1] is the full match; loc[2]:loc[3] is capture group 1.
		matchStart := loc[0]
		inCode := false
		for _, cr := range codeRanges {
			if cr.contains(matchStart) {
				inCode = true
				break
			}
		}
		if inCode {
			continue
		}

		inner := string(src[loc[2]:loc[3]])
		slug := strings.SplitN(inner, "|", 2)[0]
		if _, exists := seen[slug]; exists {
			continue
		}
		seen[slug] = struct{}{}
		result = append(result, slug)
	}

	return result
}
