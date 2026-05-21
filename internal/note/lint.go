package note

import (
	"fmt"
	"strings"
)

// Rule describes a single lint rule that can be applied to a Note.
type Rule struct {
	ID          string
	Description string
	Check       func(note *Note) []string
}

// LintError represents a single rule violation found during linting.
type LintError struct {
	Rule    string `json:"rule"`
	Message string `json:"message"`
}

// LintResult is the outcome of running all lint rules against a Note.
type LintResult struct {
	Valid  bool        `json:"valid"`
	Errors []LintError `json:"errors,omitempty"`
}

// Rules returns all lint rules for note frontmatter validation.
// Rule IDs follow the NF### convention (e.g., NF001, NF002).
// Each rule ID must be unique across all rules returned by this function.
func Rules() []Rule {
	return []Rule{
		ruleNF001,
		ruleNF002,
		ruleNF003,
		ruleNF004,
		ruleNF005,
		ruleNF006,
		ruleNF007,
	}
}

// Lint runs all registered rules against note and returns a LintResult.
// Valid is true when no rule violations are found.
func Lint(note *Note) *LintResult {
	if note == nil {
		return &LintResult{
			Valid:  false,
			Errors: []LintError{{Rule: "internal", Message: "note is nil"}},
		}
	}
	var errs []LintError
	for _, r := range Rules() {
		for _, msg := range r.Check(note) {
			errs = append(errs, LintError{Rule: r.ID, Message: msg})
		}
	}
	return &LintResult{
		Valid:  len(errs) == 0,
		Errors: errs,
	}
}

// ruleNF001 checks that all required frontmatter fields are present and
// non-empty.
var ruleNF001 = Rule{
	ID:          "NF001",
	Description: "Required frontmatter fields must be present and non-empty.",
	Check: func(n *Note) []string {
		type field struct {
			name  string
			value string
		}
		required := []field{
			{"title", n.Frontmatter.Title},
			{"created", n.Frontmatter.Created},
			{"updated", n.Frontmatter.Updated},
			{"status", n.Frontmatter.Status},
			{"confidence", n.Frontmatter.Confidence},
			{"epistemic-type", n.Frontmatter.EpistemicType},
			{"scope", n.Frontmatter.Scope},
			{"source-artifact", n.Frontmatter.SourceArtifact},
		}
		var msgs []string
		for _, f := range required {
			if f.value == "" {
				msgs = append(msgs, fmt.Sprintf("required field missing or empty: %s", f.name))
			}
		}
		return msgs
	},
}

// ruleNF002 validates that date fields contain valid YYYY-MM-DD dates.
var ruleNF002 = Rule{
	ID:          "NF002",
	Description: "Date fields must use YYYY-MM-DD format.",
	Check: func(n *Note) []string {
		type dateField struct {
			name  string
			value string
		}
		fields := []dateField{
			{"created", n.Frontmatter.Created},
			{"updated", n.Frontmatter.Updated},
			{"review-by", n.Frontmatter.ReviewBy},
		}
		var msgs []string
		for _, f := range fields {
			if f.value == "" {
				// Optional fields are skipped when empty; required fields are
				// handled by NF001.
				continue
			}
			if !IsValidDate(f.value) {
				msgs = append(msgs, fmt.Sprintf(
					"field %s has invalid date format (expected YYYY-MM-DD): %s",
					f.name, f.value,
				))
			}
		}
		return msgs
	},
}

// ruleNF003 validates enum fields against their allowed value sets.
var ruleNF003 = Rule{
	ID:          "NF003",
	Description: "Enum fields must contain one of their allowed values.",
	Check: func(n *Note) []string {
		type enumField struct {
			name  string
			value string
			valid []string
		}
		fields := []enumField{
			{"status", n.Frontmatter.Status, []string{"inbox", "verified", "deprecated", "contested", "superseded"}},
			{"epistemic-type", n.Frontmatter.EpistemicType, []string{"observation", "pattern", "constraint", "decision", "assumption", "synthesis"}},
			{"confidence", n.Frontmatter.Confidence, []string{"low", "medium", "high"}},
			{"scope", n.Frontmatter.Scope, []string{"project", "cross-project"}},
		}
		var msgs []string
		for _, f := range fields {
			if f.value == "" {
				continue
			}
			if !containsString(f.valid, f.value) {
				msgs = append(msgs, fmt.Sprintf(
					"field %s has invalid value '%s': must be one of [%s]",
					f.name, f.value, strings.Join(f.valid, ", "),
				))
			}
		}
		return msgs
	},
}

// ruleNF004 requires the project field when scope is "project".
var ruleNF004 = Rule{
	ID:          "NF004",
	Description: "Field 'project' is required when scope is 'project'.",
	Check: func(n *Note) []string {
		if n.Frontmatter.Scope == "project" && n.Frontmatter.Project == "" {
			return []string{"field project is required when scope is 'project'"}
		}
		return nil
	},
}

// ruleNF005 checks that the note body contains all required section headings.
var ruleNF005 = Rule{
	ID:          "NF005",
	Description: "Note body must contain required section headings.",
	Check: func(n *Note) []string {
		var msgs []string
		isSynthesis := n.Frontmatter.EpistemicType == "synthesis"

		required := []string{"# ", "## Related"}
		if isSynthesis {
			required = append(required, "## Synthesis", "## Contributing notes")
		} else {
			required = append(required, "## Evidence", "## Implications")
		}

		for _, heading := range required {
			if !bodyHasSection(n.Body, heading) {
				msgs = append(msgs, fmt.Sprintf("body is missing required section: %s", heading))
			}
		}
		return msgs
	},
}

// ruleNF006 detects placeholder values in required fields.
var ruleNF006 = Rule{
	ID:          "NF006",
	Description: "Required fields must not contain placeholder values.",
	Check: func(n *Note) []string {
		type field struct {
			name  string
			value string
		}
		required := []field{
			{"title", n.Frontmatter.Title},
			{"created", n.Frontmatter.Created},
			{"updated", n.Frontmatter.Updated},
			{"status", n.Frontmatter.Status},
			{"confidence", n.Frontmatter.Confidence},
			{"epistemic-type", n.Frontmatter.EpistemicType},
			{"scope", n.Frontmatter.Scope},
			{"source-agent", n.Frontmatter.SourceAgent},
			{"source-artifact", n.Frontmatter.SourceArtifact},
		}
		var msgs []string
		for _, f := range required {
			// Empty values are handled by NF001; only flag non-empty placeholders.
			if f.value != "" && IsPlaceholder(f.value) {
				msgs = append(msgs, fmt.Sprintf(
					"field %s contains a placeholder value: '%s'",
					f.name, f.value,
				))
			}
		}
		return msgs
	},
}

// ruleNF007 requires the domain field to be a non-empty list.
var ruleNF007 = Rule{
	ID:          "NF007",
	Description: "Field 'domain' must be a non-empty list.",
	Check: func(n *Note) []string {
		if len(n.Frontmatter.Domain) == 0 {
			return []string{"field domain must be a non-empty list"}
		}
		return nil
	},
}

// containsString reports whether slice contains s.
func containsString(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// bodyHasSection reports whether body contains a line that starts with heading.
// For "# " (h1 detection) we check for any line starting with "# " (one hash
// followed by a space), which matches any h1 heading. Trailing whitespace on
// each line is trimmed before matching so that "##  Evidence" (double space)
// still matches "## Evidence".
func bodyHasSection(body, heading string) bool {
	for _, line := range strings.Split(body, "\n") {
		// Normalize runs of spaces after the leading hashes so that
		// "##  Evidence" matches the expected heading "## Evidence".
		normalized := normalizeHeadingSpaces(line)
		if strings.HasPrefix(normalized, heading) {
			return true
		}
	}
	return false
}

// normalizeHeadingSpaces collapses multiple spaces between the leading '#'
// characters and the heading text into a single space, matching the canonical
// Markdown heading format.
func normalizeHeadingSpaces(line string) string {
	// Count leading '#' characters.
	i := 0
	for i < len(line) && line[i] == '#' {
		i++
	}
	if i == 0 {
		return line
	}
	// Trim any whitespace between hashes and text, then re-join with one space.
	rest := strings.TrimLeft(line[i:], " \t")
	if rest == "" {
		return line[:i]
	}
	return line[:i] + " " + rest
}
