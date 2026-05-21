package note

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// validNote returns a Note that satisfies NF001–NF007 for a non-synthesis note.
func validNote() *Note {
	fm := validFrontmatter()
	return &Note{Frontmatter: fm, Body: validBody()}
}

var _ = Describe("Helpers", func() {
	Describe("IsPlaceholder", func() {
		DescribeTable("returns true for placeholder values",
			func(v string) { Expect(IsPlaceholder(v)).To(BeTrue()) },
			Entry("empty string", ""),
			Entry("angle bracket pattern", "<your-title-here>"),
			Entry("bare angle brackets", "<>"),
			Entry("TODO uppercase", "TODO"),
			Entry("todo lowercase", "todo"),
			Entry("TBD", "TBD"),
			Entry("tbd", "tbd"),
			Entry("FIXME", "FIXME"),
			Entry("fixme", "fixme"),
			Entry("xxx", "xxx"),
			Entry("XXX", "XXX"),
			Entry("placeholder", "placeholder"),
			Entry("PLACEHOLDER", "PLACEHOLDER"),
		)

		DescribeTable("returns false for real values",
			func(v string) { Expect(IsPlaceholder(v)).To(BeFalse()) },
			Entry("normal title", "My Note Title"),
			Entry("date", "2026-01-01"),
			Entry("status", "active"),
			Entry("agent name", "claude-3-5-sonnet"),
		)
	})

	Describe("IsValidDate", func() {
		DescribeTable("returns true for valid YYYY-MM-DD dates",
			func(v string) { Expect(IsValidDate(v)).To(BeTrue()) },
			Entry("2026-01-01", "2026-01-01"),
			Entry("2024-12-31", "2024-12-31"),
			Entry("2000-02-29 (leap year)", "2000-02-29"),
		)

		DescribeTable("returns false for invalid dates",
			func(v string) { Expect(IsValidDate(v)).To(BeFalse()) },
			Entry("slash separator", "2024/01/15"),
			Entry("not a date", "not-a-date"),
			Entry("invalid month", "2024-13-01"),
			Entry("invalid day", "2024-01-32"),
			Entry("partial date", "2024-01"),
			Entry("empty string", ""),
		)
	})
})

var _ = Describe("Lint / NF002", func() {
	Context("valid dates on created and updated", func() {
		It("returns no NF002 errors", func() {
			n := validNote()
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF002"))
			}
		})
	})

	Context("invalid created date", func() {
		It("returns an NF002 error", func() {
			n := validNote()
			n.Frontmatter.Created = "2024/01/15"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF002",
				Message: "field created has invalid date format (expected YYYY-MM-DD): 2024/01/15",
			}))
		})
	})

	Context("invalid updated date", func() {
		It("returns an NF002 error", func() {
			n := validNote()
			n.Frontmatter.Updated = "not-a-date"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF002",
				Message: "field updated has invalid date format (expected YYYY-MM-DD): not-a-date",
			}))
		})
	})

	Context("review-by is empty", func() {
		It("does not produce an NF002 error", func() {
			n := validNote()
			n.Frontmatter.ReviewBy = ""
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF002"))
			}
		})
	})

	Context("review-by has invalid date", func() {
		It("returns an NF002 error", func() {
			n := validNote()
			n.Frontmatter.ReviewBy = "31-12-2026"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF002",
				Message: "field review-by has invalid date format (expected YYYY-MM-DD): 31-12-2026",
			}))
		})
	})
})

var _ = Describe("Lint / NF003", func() {
	Context("all enum fields are valid", func() {
		It("returns no NF003 errors", func() {
			n := validNote()
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF003"))
			}
		})
	})

	Context("invalid status", func() {
		It("returns an NF003 error listing valid options", func() {
			n := validNote()
			n.Frontmatter.Status = "unknown"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF003",
				Message: "field status has invalid value 'unknown': must be one of [draft, active, archived, deprecated]",
			}))
		})
	})

	Context("invalid epistemic-type", func() {
		It("returns an NF003 error", func() {
			n := validNote()
			n.Frontmatter.EpistemicType = "opinion"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF003",
				Message: "field epistemic-type has invalid value 'opinion': must be one of [observation, inference, synthesis, hypothesis, procedure]",
			}))
		})
	})

	Context("invalid confidence", func() {
		It("returns an NF003 error", func() {
			n := validNote()
			n.Frontmatter.Confidence = "very-high"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF003",
				Message: "field confidence has invalid value 'very-high': must be one of [low, medium, high]",
			}))
		})
	})

	Context("invalid scope", func() {
		It("returns an NF003 error", func() {
			n := validNote()
			n.Frontmatter.Scope = "team"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF003",
				Message: "field scope has invalid value 'team': must be one of [project, global]",
			}))
		})
	})

	Context("empty enum field", func() {
		It("does not produce an NF003 error (NF001 handles empty)", func() {
			n := validNote()
			n.Frontmatter.Status = ""
			result := Lint(n)
			for _, e := range result.Errors {
				if e.Rule == "NF003" {
					Expect(e.Message).NotTo(ContainSubstring("status"))
				}
			}
		})
	})
})

var _ = Describe("Lint / NF004", func() {
	Context("scope=project with project set", func() {
		It("returns no NF004 error", func() {
			n := validNote()
			n.Frontmatter.Scope = "project"
			n.Frontmatter.Project = "agent-memory"
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF004"))
			}
		})
	})

	Context("scope=project with empty project", func() {
		It("returns an NF004 error", func() {
			n := validNote()
			n.Frontmatter.Scope = "project"
			n.Frontmatter.Project = ""
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF004",
				Message: "field project is required when scope is 'project'",
			}))
		})
	})

	Context("scope=global with empty project", func() {
		It("returns no NF004 error", func() {
			n := validNote()
			n.Frontmatter.Scope = "global"
			n.Frontmatter.Project = ""
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF004"))
			}
		})
	})
})

var _ = Describe("Lint / NF005", func() {
	Context("non-synthesis note with all required sections", func() {
		It("returns no NF005 errors", func() {
			n := validNote()
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF005"))
			}
		})
	})

	Context("non-synthesis note missing ## Evidence", func() {
		It("returns an NF005 error", func() {
			n := validNote()
			n.Body = "# My Title\n\n## Implications\n\ntext\n\n## Related\n\nnone"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF005",
				Message: "body is missing required section: ## Evidence",
			}))
		})
	})

	Context("non-synthesis note missing ## Implications", func() {
		It("returns an NF005 error", func() {
			n := validNote()
			n.Body = "# My Title\n\n## Evidence\n\ntext\n\n## Related\n\nnone"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF005",
				Message: "body is missing required section: ## Implications",
			}))
		})
	})

	Context("note missing ## Related", func() {
		It("returns an NF005 error", func() {
			n := validNote()
			n.Body = "# My Title\n\n## Evidence\n\ntext\n\n## Implications\n\ntext"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF005",
				Message: "body is missing required section: ## Related",
			}))
		})
	})

	Context("note missing h1 title", func() {
		It("returns an NF005 error", func() {
			n := validNote()
			n.Body = "## Evidence\n\ntext\n\n## Implications\n\ntext\n\n## Related\n\nnone"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF005",
				Message: "body is missing required section: # ",
			}))
		})
	})

	Context("synthesis note with all required sections", func() {
		It("returns no NF005 errors", func() {
			n := validNote()
			n.Frontmatter.EpistemicType = "synthesis"
			n.Body = "# My Title\n\n## Synthesis\n\ntext\n\n## Contributing notes\n\nnotes\n\n## Related\n\nnone"
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF005"))
			}
		})
	})

	Context("synthesis note missing ## Synthesis", func() {
		It("returns an NF005 error", func() {
			n := validNote()
			n.Frontmatter.EpistemicType = "synthesis"
			n.Body = "# My Title\n\n## Contributing notes\n\nnotes\n\n## Related\n\nnone"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF005",
				Message: "body is missing required section: ## Synthesis",
			}))
		})
	})

	Context("synthesis note should not require ## Evidence or ## Implications", func() {
		It("returns no NF005 error for missing Evidence/Implications", func() {
			n := validNote()
			n.Frontmatter.EpistemicType = "synthesis"
			n.Body = "# My Title\n\n## Synthesis\n\ntext\n\n## Contributing notes\n\nnotes\n\n## Related\n\nnone"
			result := Lint(n)
			for _, e := range result.Errors {
				if e.Rule == "NF005" {
					Expect(e.Message).NotTo(ContainSubstring("Evidence"))
					Expect(e.Message).NotTo(ContainSubstring("Implications"))
				}
			}
		})
	})
})

var _ = Describe("Lint / NF006", func() {
	Context("field with angle-bracket placeholder", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.Title = "<your-title-here>"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field title contains a placeholder value: '<your-title-here>'",
			}))
		})
	})

	Context("field with TODO", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.SourceAgent = "TODO"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field source-agent contains a placeholder value: 'TODO'",
			}))
		})
	})

	Context("field with tbd (case-insensitive)", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.SourceArtifact = "tbd"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field source-artifact contains a placeholder value: 'tbd'",
			}))
		})
	})

	Context("field with FIXME", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.Title = "FIXME"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field title contains a placeholder value: 'FIXME'",
			}))
		})
	})

	Context("field with placeholder keyword", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.Title = "placeholder"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field title contains a placeholder value: 'placeholder'",
			}))
		})
	})

	Context("field with xxx", func() {
		It("returns an NF006 error", func() {
			n := validNote()
			n.Frontmatter.Title = "xxx"
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF006",
				Message: "field title contains a placeholder value: 'xxx'",
			}))
		})
	})

	Context("empty field", func() {
		It("does not produce an NF006 error (NF001 handles it)", func() {
			n := validNote()
			n.Frontmatter.Title = ""
			result := Lint(n)
			for _, e := range result.Errors {
				if e.Rule == "NF006" {
					Expect(e.Message).NotTo(ContainSubstring("title"))
				}
			}
		})
	})

	Context("valid field value", func() {
		It("returns no NF006 error", func() {
			n := validNote()
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF006"))
			}
		})
	})
})

var _ = Describe("Lint / NF007", func() {
	Context("domain is nil", func() {
		It("returns an NF007 error", func() {
			n := validNote()
			n.Frontmatter.Domain = nil
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF007",
				Message: "field domain must be a non-empty list",
			}))
		})
	})

	Context("domain is empty slice", func() {
		It("returns an NF007 error", func() {
			n := validNote()
			n.Frontmatter.Domain = []string{}
			result := Lint(n)
			Expect(result.Errors).To(ContainElement(LintError{
				Rule:    "NF007",
				Message: "field domain must be a non-empty list",
			}))
		})
	})

	Context("domain has one item", func() {
		It("returns no NF007 error", func() {
			n := validNote()
			n.Frontmatter.Domain = []string{"go"}
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF007"))
			}
		})
	})

	Context("domain has multiple items", func() {
		It("returns no NF007 error", func() {
			n := validNote()
			n.Frontmatter.Domain = []string{"go", "testing", "memory"}
			result := Lint(n)
			for _, e := range result.Errors {
				Expect(e.Rule).NotTo(Equal("NF007"))
			}
		})
	})
})

var _ = Describe("Rules()", func() {
	It("includes NF001 through NF007 in order", func() {
		ids := make([]string, 0, len(Rules()))
		for _, r := range Rules() {
			ids = append(ids, r.ID)
		}
		Expect(ids).To(Equal([]string{"NF001", "NF002", "NF003", "NF004", "NF005", "NF006", "NF007"}))
	})
})
