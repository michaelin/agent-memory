package vault

import (
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("VaultStructure", func() {
	It("returns 13 entries", func() {
		entries := VaultStructure()
		Expect(entries).To(HaveLen(13))
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

	It("includes _deprecated/ directory", func() {
		paths := make([]string, 0)
		for _, e := range VaultStructure() {
			if e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		Expect(paths).To(ContainElement("_deprecated"))
	})

	It("includes _meta/templates/ directory", func() {
		paths := make([]string, 0)
		for _, e := range VaultStructure() {
			if e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		Expect(paths).To(ContainElement("_meta/templates"))
	})

	It("includes librarian-agent.md template file", func() {
		paths := make([]string, 0)
		for _, e := range VaultStructure() {
			if !e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		Expect(paths).To(ContainElement("_meta/templates/librarian-agent.md"))
	})

	It("includes librarian-skill.md template file", func() {
		paths := make([]string, 0)
		for _, e := range VaultStructure() {
			if !e.IsDir {
				paths = append(paths, e.Path)
			}
		}
		Expect(paths).To(ContainElement("_meta/templates/librarian-skill.md"))
	})
})
