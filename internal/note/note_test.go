package note

import (
	"errors"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Parse", func() {
	Context("valid note with all fields", func() {
		It("parses frontmatter and body correctly", func() {
			content := []byte(`---
title: My Note
created: "2026-01-01"
updated: "2026-05-21"
review-by: "2026-12-31"
status: verified
confidence: high
epistemic-type: observation
scope: project
project: agent-memory
domain:
  - go
  - testing
source-agent: test-agent
source-artifact: test-artifact
verified-by: human
verified-date: "2026-05-21"
requires-human-review: true
update-type: minor
targets:
  - target-a
  - target-b
tags:
  - tag1
  - tag2
---
This is the body of the note.
`)
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n).NotTo(BeNil())

			fm := n.Frontmatter
			Expect(fm.Title).To(Equal("My Note"))
			Expect(fm.Created).To(Equal("2026-01-01"))
			Expect(fm.Updated).To(Equal("2026-05-21"))
			Expect(fm.ReviewBy).To(Equal("2026-12-31"))
			Expect(fm.Status).To(Equal("verified"))
			Expect(fm.Confidence).To(Equal("high"))
			Expect(fm.EpistemicType).To(Equal("observation"))
			Expect(fm.Scope).To(Equal("project"))
			Expect(fm.Project).To(Equal("agent-memory"))
			Expect(fm.Domain).To(ConsistOf("go", "testing"))
			Expect(fm.SourceAgent).To(Equal("test-agent"))
			Expect(fm.SourceArtifact).To(Equal("test-artifact"))
			Expect(fm.VerifiedBy).To(Equal("human"))
			Expect(fm.VerifiedDate).To(Equal("2026-05-21"))
			Expect(fm.RequiresHumanReview).To(BeTrue())
			Expect(fm.UpdateType).To(Equal("minor"))
			Expect(fm.Targets).To(ConsistOf("target-a", "target-b"))
			Expect(fm.Tags).To(ConsistOf("tag1", "tag2"))
		})
	})

	Context("body extraction", func() {
		It("trims leading and trailing whitespace from the body", func() {
			content := []byte("---\ntitle: Test\n---\n\n  hello world  \n\n")
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n.Body).To(Equal("hello world"))
		})

		It("returns an empty string for an empty body", func() {
			content := []byte("---\ntitle: Test\n---\n")
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n.Body).To(BeEmpty())
		})

		It("preserves multi-line body content", func() {
			content := []byte("---\ntitle: Test\n---\nLine one.\nLine two.\n")
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n.Body).To(Equal("Line one.\nLine two."))
		})
	})

	Context("missing frontmatter", func() {
		It("returns ErrNoFrontmatter when there are no delimiters", func() {
			content := []byte("Just plain text with no frontmatter.\n")
			_, err := Parse(content)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrNoFrontmatter)).To(BeTrue())
		})

		It("returns ErrNoFrontmatter when the opening delimiter is missing", func() {
			content := []byte("title: Test\n---\nbody\n")
			_, err := Parse(content)
			Expect(errors.Is(err, ErrNoFrontmatter)).To(BeTrue())
		})

		It("returns ErrNoFrontmatter when the closing delimiter is missing", func() {
			content := []byte("---\ntitle: Test\nbody without closing delimiter\n")
			_, err := Parse(content)
			Expect(errors.Is(err, ErrNoFrontmatter)).To(BeTrue())
		})
	})

	Context("malformed YAML", func() {
		It("returns an error for invalid YAML in frontmatter", func() {
			content := []byte("---\ntitle: [\nbad yaml\n---\nbody\n")
			_, err := Parse(content)
			Expect(err).To(HaveOccurred())
			Expect(errors.Is(err, ErrNoFrontmatter)).To(BeFalse())
		})
	})

	Context("unknown YAML fields", func() {
		It("silently ignores unknown fields", func() {
			content := []byte("---\ntitle: Known\nunknown-field: some value\nanother-unknown: 42\n---\nbody\n")
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n.Frontmatter.Title).To(Equal("Known"))
		})
	})

	Context("CRLF line endings", func() {
		It("parses correctly — same result as LF", func() {
			lf := []byte("---\ntitle: CRLF Test\n---\nBody text.\n")
			crlf := []byte("---\r\ntitle: CRLF Test\r\n---\r\nBody text.\r\n")

			nLF, err := Parse(lf)
			Expect(err).NotTo(HaveOccurred())

			nCRLF, err := Parse(crlf)
			Expect(err).NotTo(HaveOccurred())

			Expect(nCRLF.Frontmatter.Title).To(Equal(nLF.Frontmatter.Title))
			Expect(nCRLF.Body).To(Equal(nLF.Body))
		})
	})

	Context("delimiter with trailing spaces", func() {
		It("treats '---   ' as a valid delimiter", func() {
			content := []byte("---   \ntitle: Trailing Spaces\n---   \nBody.\n")
			n, err := Parse(content)
			Expect(err).NotTo(HaveOccurred())
			Expect(n.Frontmatter.Title).To(Equal("Trailing Spaces"))
		})
	})

	Context("first line is '---notyaml'", func() {
		It("returns ErrNoFrontmatter", func() {
			content := []byte("---notyaml\ntitle: Test\n---\nbody\n")
			_, err := Parse(content)
			Expect(errors.Is(err, ErrNoFrontmatter)).To(BeTrue())
		})
	})
})
