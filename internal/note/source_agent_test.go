package note

import (
	"os"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("DetectSourceAgent", func() {
	Context("OPENCODE_RUN_ID is set", func() {
		It("returns opencode:{value}", func() {
			Expect(os.Setenv("OPENCODE_RUN_ID", "abc123")).To(Succeed())
			DeferCleanup(func() { os.Unsetenv("OPENCODE_RUN_ID") })
			Expect(DetectSourceAgent()).To(Equal("opencode:abc123"))
		})
	})

	Context("no env vars set", func() {
		It("returns empty string", func() {
			os.Unsetenv("OPENCODE_RUN_ID")
			Expect(DetectSourceAgent()).To(Equal(""))
		})
	})
})
