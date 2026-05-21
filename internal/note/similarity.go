package note

import (
	"strings"
	"unicode"
)

// stopWords is the set of tokens to exclude during normalization.
var stopWords = map[string]struct{}{
	"a": {}, "an": {}, "the": {}, "is": {}, "are": {}, "was": {}, "were": {},
	"in": {}, "on": {}, "of": {}, "to": {}, "for": {}, "and": {}, "or": {},
	"but": {}, "with": {}, "by": {}, "at": {}, "from": {},
}

// NormalizeTokens lowercases text, splits on whitespace, strips punctuation
// and non-ASCII characters from each token, and removes stop words.
// Stop words: a, an, the, is, are, was, were, in, on, of, to, for, and, or, but, with, by, at, from
func NormalizeTokens(text string) []string {
	fields := strings.Fields(strings.ToLower(text))
	result := make([]string, 0, len(fields))
	for _, tok := range fields {
		// Keep only ASCII alphanumeric characters (matches Slug behavior).
		cleaned := strings.Map(func(r rune) rune {
			if r > unicode.MaxASCII {
				return -1
			}
			if unicode.IsLetter(r) || unicode.IsDigit(r) {
				return r
			}
			return -1
		}, tok)
		if cleaned == "" {
			continue
		}
		if _, isStop := stopWords[cleaned]; isStop {
			continue
		}
		result = append(result, cleaned)
	}
	return result
}

// Jaccard returns the Jaccard similarity coefficient for two token slices.
// Returns 0.0 when both slices are empty.
func Jaccard(a, b []string) float64 {
	if len(a) == 0 && len(b) == 0 {
		return 0.0
	}
	if len(a) == 0 || len(b) == 0 {
		return 0.0
	}

	setA := make(map[string]struct{}, len(a))
	for _, tok := range a {
		setA[tok] = struct{}{}
	}

	setB := make(map[string]struct{}, len(b))
	for _, tok := range b {
		setB[tok] = struct{}{}
	}

	intersection := 0
	for tok := range setA {
		if _, ok := setB[tok]; ok {
			intersection++
		}
	}

	union := len(setA) + len(setB) - intersection
	return float64(intersection) / float64(union)
}
