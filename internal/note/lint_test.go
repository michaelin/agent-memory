package note

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// validFrontmatter returns a Frontmatter with all NF001-required fields set.
// It also satisfies NF002–NF007 so that tests focused on NF001 are not
// polluted by errors from other rules.
func validFrontmatter() Frontmatter {
	return Frontmatter{
		Title:          "Test Note",
		Created:        "2026-01-01",
		Updated:        "2026-05-21",
		Status:         "verified",
		Confidence:     "high",
		EpistemicType:  "observation",
		Scope:          "project",
		Project:        "agent-memory",
		Domain:         []string{"go"},
		SourceAgent:    "test-agent",
		SourceArtifact: "test-artifact",
	}
}

// validBody returns a note body that satisfies NF005 for a non-synthesis note.
func validBody() string {
	return "# Test Note\n\n## Evidence\n\nsome evidence\n\n## Implications\n\nsome implications\n\n## Related\n\nnone"
}

var _ = Describe("Lint / NF001", func() {
	Context("valid note with all required fields", func() {
		It("returns Valid=true with no errors", func() {
			n := &Note{Frontmatter: validFrontmatter(), Body: validBody()}
			result := Lint(n)
			Expect(result.Valid).To(BeTrue())
			Expect(result.Errors).To(BeEmpty())
		})
	})

	Context("note missing title", func() {
		It("returns an error for 'title'", func() {
			fm := validFrontmatter()
			fm.Title = ""
			result := Lint(&Note{Frontmatter: fm, Body: validBody()})
			Expect(result.Valid).To(BeFalse())
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF001",
				Message: "required field missing or empty: title",
			}))
		})
	})

	Context("note with an empty-string field", func() {
		It("treats empty string as missing", func() {
			fm := validFrontmatter()
			fm.Confidence = ""
			result := Lint(&Note{Frontmatter: fm, Body: validBody()})
			Expect(result.Valid).To(BeFalse())
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF001",
				Message: "required field missing or empty: confidence",
			}))
		})
	})

	Context("note missing multiple required fields", func() {
		It("returns one NF001 error per missing field", func() {
			n := &Note{} // zero-value Frontmatter — all fields empty
			result := Lint(n)
			Expect(result.Valid).To(BeFalse())

			requiredFields := []string{
				"title", "created", "updated", "status", "confidence",
				"epistemic-type", "scope", "source-agent", "source-artifact",
			}
			for _, field := range requiredFields {
				Expect(result.Errors).To(ContainElement(LintError{
					Rule:    "NF001",
					Message: "required field missing or empty: " + field,
				}))
			}
		})
	})

	Context("note with all required fields plus optional/extra fields", func() {
		It("returns Valid=true regardless of optional fields", func() {
			fm := validFrontmatter()
			fm.ReviewBy = "2026-12-31"
			fm.Tags = []string{"tag1"}
			n := &Note{Frontmatter: fm, Body: validBody()}
			result := Lint(n)
			Expect(result.Valid).To(BeTrue())
			Expect(result.Errors).To(BeEmpty())
		})
	})

	Context("Rules()", func() {
		It("includes NF001", func() {
			ids := make([]string, 0, len(Rules()))
			for _, r := range Rules() {
				ids = append(ids, r.ID)
			}
			Expect(ids).To(ContainElement("NF001"))
		})
	})
})

var _ = Describe("Lint / nil note", func() {
	It("returns invalid result when note is nil", func() {
		result := Lint(nil)
		Expect(result.Valid).To(BeFalse())
		Expect(result.Errors).To(HaveLen(1))
		Expect(result.Errors[0].Rule).To(Equal("internal"))
	})
})
