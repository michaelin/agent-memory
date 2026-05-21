package note

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Serialize", func() {
	Context("complete note", func() {
		It("serializes all frontmatter fields and body", func() {
			n := &Note{
				Frontmatter: validFrontmatter(),
				Body:        validBody(),
			}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(HavePrefix("---\n"))
			Expect(string(data)).To(ContainSubstring("title: Test Note"))
			Expect(string(data)).To(ContainSubstring(validBody()))
		})

		It("round-trips through Parse", func() {
			n := &Note{
				Frontmatter: validFrontmatter(),
				Body:        validBody(),
			}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())

			parsed, err := Parse(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Frontmatter.Title).To(Equal(n.Frontmatter.Title))
			Expect(parsed.Frontmatter.Created).To(Equal(n.Frontmatter.Created))
			Expect(parsed.Frontmatter.Status).To(Equal(n.Frontmatter.Status))
			Expect(parsed.Frontmatter.Domain).To(Equal(n.Frontmatter.Domain))
			Expect(parsed.Body).To(Equal(n.Body))
		})
	})

	Context("minimal note", func() {
		It("serializes with empty frontmatter fields and body only", func() {
			n := &Note{Body: "just a body"}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(HavePrefix("---\n"))
			Expect(string(data)).To(ContainSubstring("just a body"))
		})
	})

	Context("nil note", func() {
		It("returns an error", func() {
			data, err := Serialize(nil)
			Expect(err).To(HaveOccurred())
			Expect(data).To(BeNil())
		})
	})

	Context("empty body", func() {
		It("produces valid frontmatter with empty body section", func() {
			n := &Note{Frontmatter: validFrontmatter()}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(HavePrefix("---\n"))
			Expect(string(data)).To(ContainSubstring("title: Test Note"))

			parsed, err := Parse(data)
			Expect(err).NotTo(HaveOccurred())
			Expect(parsed.Body).To(BeEmpty())
		})
	})

	Context("superseded-by field", func() {
		It("includes superseded-by in serialized output when set", func() {
			fm := validFrontmatter()
			fm.SupersededBy = "some-other-note"
			n := &Note{Frontmatter: fm, Body: validBody()}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).To(ContainSubstring("superseded-by: some-other-note"))
		})

		It("omits superseded-by when empty", func() {
			n := &Note{Frontmatter: validFrontmatter(), Body: validBody()}
			data, err := Serialize(n)
			Expect(err).NotTo(HaveOccurred())
			Expect(string(data)).NotTo(ContainSubstring("superseded-by"))
		})
	})
})

var _ = Describe("Slug", func() {
	It("converts a normal title to a slug", func() {
		Expect(Slug("Hello World")).To(Equal("hello-world"))
	})

	It("strips special characters", func() {
		Expect(Slug("Hello, World!")).To(Equal("hello-world"))
	})

	It("collapses consecutive hyphens", func() {
		Expect(Slug("foo--bar")).To(Equal("foo-bar"))
	})

	It("truncates long titles to 60 chars with no trailing hyphen", func() {
		// 80-char title: "aaaa...a" (80 a's separated by spaces every 4)
		title := "This is a very long title that exceeds sixty characters in total length here"
		result := Slug(title)
		Expect(len(result)).To(BeNumerically("<=", 60))
		Expect(result).NotTo(HaveSuffix("-"))
	})

	It("returns empty string for empty input", func() {
		Expect(Slug("")).To(Equal(""))
	})

	It("strips non-ASCII unicode characters", func() {
		result := Slug("Héllo Wörld")
		Expect(len(result)).To(BeNumerically(">", 0))
		// Result must only contain lowercase alphanumeric and hyphens
		for _, r := range result {
			Expect(r >= 'a' && r <= 'z' || r >= '0' && r <= '9' || r == '-').To(BeTrue(),
				"unexpected character %q in slug %q", r, result)
		}
	})
})
