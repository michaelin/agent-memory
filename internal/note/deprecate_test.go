package note

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// validNoteContent returns a minimal but fully-valid note file for testing.
// The slug is used only as a label; the actual file name is controlled by the
// caller.
func validNoteContent(title string) []byte {
	return []byte(`---
title: ` + title + `
created: "2026-01-01"
updated: "2026-01-01"
review-by: "2027-01-01"
status: verified
confidence: high
type: observation
scope: project
project: agent-memory
source-agent: test-agent
source-artifact: test-artifact
---
# ` + title + `

## Evidence

some evidence

## Implications

some implications

## Related

none
`)
}

var _ = Describe("Deprecate", func() {
	var vaultPath string

	BeforeEach(func() {
		var err error
		vaultPath, err = os.MkdirTemp("", "vault-deprecate-*")
		Expect(err).NotTo(HaveOccurred())

		for _, dir := range []string{"notes", "_meta"} {
			Expect(os.MkdirAll(filepath.Join(vaultPath, dir), 0o755)).To(Succeed())
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(vaultPath)).To(Succeed())
	})

	Context("happy path", func() {
		It("deprecates the note, moves it, and logs the action", func() {
			slug := "my-test-note"
			notePath := filepath.Join(vaultPath, "notes", slug+".md")
			Expect(os.WriteFile(notePath, validNoteContent("My Test Note"), 0o644)).To(Succeed())

			result, err := Deprecate(vaultPath, slug, "new-slug")

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("deprecated"))
			Expect(result.Slug).To(Equal(slug))
			Expect(result.SupersededBy).To(Equal("new-slug"))

			// Deprecated file must exist.
			deprecatedPath := filepath.Join(vaultPath, "_deprecated", slug+".md")
			Expect(deprecatedPath).To(BeAnExistingFile())
			Expect(result.MovedTo).To(Equal(deprecatedPath))

			// Original must be gone.
			Expect(notePath).NotTo(BeAnExistingFile())

			// Deprecated file must have updated frontmatter.
			data, readErr := os.ReadFile(deprecatedPath)
			Expect(readErr).NotTo(HaveOccurred())
			parsed, parseErr := Parse(data)
			Expect(parseErr).NotTo(HaveOccurred())
			Expect(parsed.Frontmatter.Status).To(Equal("deprecated"))
			Expect(parsed.Frontmatter.SupersededBy).To(Equal("new-slug"))

			// Log must contain a "deprecate" entry.
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			logData, logErr := os.ReadFile(logPath)
			Expect(logErr).NotTo(HaveOccurred())
			Expect(string(logData)).To(ContainSubstring("deprecate"))
			Expect(string(logData)).To(ContainSubstring(slug))
		})
	})

	Context("note not found", func() {
		It("returns an error when the slug does not exist in notes/", func() {
			result, err := Deprecate(vaultPath, "ghost-note", "")

			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("not found"))
		})
	})

	Context("_deprecated/ auto-created", func() {
		It("creates _deprecated/ if it does not exist and places the file there", func() {
			// Confirm _deprecated/ does not exist before the call.
			deprecatedDir := filepath.Join(vaultPath, "_deprecated")
			_, statErr := os.Stat(deprecatedDir)
			Expect(os.IsNotExist(statErr)).To(BeTrue())

			slug := "auto-create-test"
			notePath := filepath.Join(vaultPath, "notes", slug+".md")
			Expect(os.WriteFile(notePath, validNoteContent("Auto Create Test"), 0o644)).To(Succeed())

			result, err := Deprecate(vaultPath, slug, "")
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("deprecated"))

			// _deprecated/ must now exist.
			Expect(deprecatedDir).To(BeADirectory())

			// File must be present inside it.
			Expect(filepath.Join(deprecatedDir, slug+".md")).To(BeAnExistingFile())
		})
	})
})
