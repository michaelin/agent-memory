package note

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("ExtractWikilinks", func() {
	Context("single plain link [[slug]]", func() {
		It("returns the slug", func() {
			Expect(ExtractWikilinks("see [[my-note]] for details")).To(Equal([]string{"my-note"}))
		})
	})

	Context("alias form [[slug|alias]]", func() {
		It("returns only the slug", func() {
			Expect(ExtractWikilinks("see [[my-note|My Note]] for details")).To(Equal([]string{"my-note"}))
		})
	})

	Context("multiple links on one line", func() {
		It("returns all slugs", func() {
			Expect(ExtractWikilinks("[[alpha]] and [[beta]] and [[gamma]]")).To(Equal([]string{"alpha", "beta", "gamma"}))
		})
	})

	Context("duplicate links", func() {
		It("deduplicates, preserving first-occurrence order", func() {
			Expect(ExtractWikilinks("[[foo]] then [[bar]] then [[foo]]")).To(Equal([]string{"foo", "bar"}))
		})
	})

	Context("no links", func() {
		It("returns empty slice", func() {
			Expect(ExtractWikilinks("no links here")).To(BeEmpty())
		})
	})

	Context("links in fenced code blocks", func() {
		It("skips wikilinks inside fenced code blocks", func() {
			text := "```\n[[code-slug]]\n```"
			Expect(ExtractWikilinks(text)).To(BeEmpty())
		})
	})

	Context("links in inline code spans", func() {
		It("skips wikilinks inside inline code", func() {
			Expect(ExtractWikilinks("see `[[not-a-link]]` here")).To(BeEmpty())
		})
	})

	Context("links in indented code blocks", func() {
		It("skips wikilinks in 4-space indented code", func() {
			text := "prose\n\n    [[indented-code-link]]\n"
			Expect(ExtractWikilinks(text)).To(BeEmpty())
		})
	})

	Context("mix of prose and fenced code block", func() {
		It("returns only the prose link, skips the code link", func() {
			text := "[[real-link]]\n\n```\n[[code-link]]\n```\n"
			Expect(ExtractWikilinks(text)).To(Equal([]string{"real-link"}))
		})
	})

	Context("link in prose after a fenced code block", func() {
		It("still extracts the link", func() {
			text := "```\n[[code-link]]\n```\n\nsee [[after-code]] for details"
			Expect(ExtractWikilinks(text)).To(Equal([]string{"after-code"}))
		})
	})
})
