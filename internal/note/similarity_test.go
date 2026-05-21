package note

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("NormalizeTokens", func() {
	Context("normal text", func() {
		It("lowercases tokens and strips punctuation", func() {
			tokens := NormalizeTokens("Hello, World!")
			Expect(tokens).To(ConsistOf("hello", "world"))
		})
	})

	Context("stop words", func() {
		It("removes stop words", func() {
			tokens := NormalizeTokens("the cat sat on a mat")
			Expect(tokens).To(ConsistOf("cat", "sat", "mat"))
		})
	})

	Context("empty string", func() {
		It("returns empty slice", func() {
			tokens := NormalizeTokens("")
			Expect(tokens).To(BeEmpty())
		})
	})

	Context("only stop words", func() {
		It("returns empty slice", func() {
			tokens := NormalizeTokens("a an the is are was were in on of to for and or but with by at from")
			Expect(tokens).To(BeEmpty())
		})
	})
})

var _ = Describe("Jaccard", func() {
	Context("identical token sets", func() {
		It("returns 1.0", func() {
			a := []string{"foo", "bar", "baz"}
			Expect(Jaccard(a, a)).To(Equal(1.0))
		})
	})

	Context("completely disjoint sets", func() {
		It("returns 0.0", func() {
			a := []string{"foo", "bar"}
			b := []string{"baz", "qux"}
			Expect(Jaccard(a, b)).To(Equal(0.0))
		})
	})

	Context("partial overlap", func() {
		It("returns correct ratio", func() {
			a := []string{"foo", "bar", "baz"}
			b := []string{"bar", "baz", "qux"}
			// intersection: {bar, baz} = 2; union: {foo, bar, baz, qux} = 4
			Expect(Jaccard(a, b)).To(BeNumerically("~", 0.5, 1e-9))
		})
	})

	Context("both empty", func() {
		It("returns 0.0", func() {
			Expect(Jaccard(nil, nil)).To(Equal(0.0))
		})
	})

	Context("one empty, one non-empty", func() {
		It("returns 0.0", func() {
			Expect(Jaccard([]string{"foo"}, nil)).To(Equal(0.0))
			Expect(Jaccard(nil, []string{"foo"})).To(Equal(0.0))
		})
	})

	Context("stop-word-heavy titles", func() {
		It("computes similarity on non-stop tokens", func() {
			a := NormalizeTokens("the cat sat on the mat")
			b := NormalizeTokens("a cat sat on a rug")
			// a → [cat, sat, mat]; b → [cat, sat, rug]
			// intersection: {cat, sat} = 2; union: {cat, sat, mat, rug} = 4
			Expect(Jaccard(a, b)).To(BeNumerically("~", 0.5, 1e-9))
		})
	})
})
