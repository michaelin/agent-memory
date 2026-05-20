package vault

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VaultStructure", func() {
	It("returns 9 entries", func() {
		entries := VaultStructure()
		Expect(entries).To(HaveLen(9))
	})

	It("lists directories before files", func() {
		entries := VaultStructure()
		seenFiles := false
		for _, e := range entries {
			if !e.IsDir {
				seenFiles = true
			}
			if seenFiles && e.IsDir {
				Fail("directory " + e.Path + " appears after a file entry")
			}
		}
	})

	It("has readable non-empty templates for all file entries", func() {
		entries := VaultStructure()
		for _, e := range entries {
			if e.IsDir {
				continue
			}
			data, err := templateFS.ReadFile(e.Template)
			Expect(err).NotTo(HaveOccurred(), "template %s not readable", e.Template)
			Expect(data).NotTo(BeEmpty(), "template %s is empty", e.Template)
		}
	})
})
