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

	Context("links in code blocks", func() {
		It("extracts them anyway (code-block awareness deferred)", func() {
			text := "```\n[[code-slug]]\n```"
			Expect(ExtractWikilinks(text)).To(Equal([]string{"code-slug"}))
		})
	})
})
