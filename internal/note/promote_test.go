package note

import (
	"os"
	"path/filepath"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// makePromoteVault creates a minimal vault with _inbox/, notes/, and _meta/.
func makePromoteVault(root string) {
	GinkgoHelper()
	for _, dir := range []string{"_inbox", "notes", "_meta"} {
		Expect(os.MkdirAll(filepath.Join(root, dir), 0o755)).To(Succeed())
	}
}

// validInboxNote returns the raw bytes of a valid observation note suitable
// for writing to _inbox/. The slug is derived from the title "Test Observation
// Note" → "test-observation-note".
func validObservationNoteBytes() []byte {
	today := time.Now().Format("2006-01-02")
	reviewBy := time.Now().AddDate(0, 0, 90).Format("2006-01-02")
	return []byte(`---
title: Test Observation Note
created: ` + today + `
updated: ` + today + `
review-by: ` + reviewBy + `
status: inbox
confidence: medium
type: observation
scope: project
project: agent-memory
domain:
  - go
source-agent: test-agent
source-artifact: test-artifact
requires-human-review: false
---
# Test Observation Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

none
`)
}

// validConstraintNoteBytes returns a valid constraint note (RequiresHumanReview=true).
func validConstraintNoteBytes() []byte {
	today := time.Now().Format("2006-01-02")
	reviewBy := time.Now().AddDate(0, 0, 365).Format("2006-01-02")
	return []byte(`---
title: Test Constraint Note
created: ` + today + `
updated: ` + today + `
review-by: ` + reviewBy + `
status: inbox
confidence: high
type: constraint
scope: project
project: agent-memory
domain:
  - go
source-agent: test-agent
source-artifact: test-artifact
requires-human-review: true
---
# Test Constraint Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

none
`)
}

// inboxFilename returns the standard inbox filename for a given slug.
func inboxFilename(slug string) string {
	today := time.Now().Format("2006-01-02")
	return today + "-" + slug + ".md"
}

var _ = Describe("FindBySlug", func() {
	var vaultPath string

	BeforeEach(func() {
		var err error
		vaultPath, err = os.MkdirTemp("", "vault-findbyslug-*")
		Expect(err).NotTo(HaveOccurred())
		makePromoteVault(vaultPath)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(vaultPath)).To(Succeed())
	})

	Context("inbox directory (date-slug naming)", func() {
		It("finds a file matching *-{slug}.md", func() {
			slug := "my-slug"
			name := inboxFilename(slug)
			inboxDir := filepath.Join(vaultPath, "_inbox")
			Expect(os.WriteFile(filepath.Join(inboxDir, name), []byte("body"), 0o644)).To(Succeed())

			found, err := FindBySlug(inboxDir, slug)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(filepath.Join(inboxDir, name)))
		})

		It("returns not found error when no file matches", func() {
			inboxDir := filepath.Join(vaultPath, "_inbox")
			_, err := FindBySlug(inboxDir, "ghost-slug")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})

		It("returns ambiguous error when multiple files match", func() {
			slug := "shared-slug"
			inboxDir := filepath.Join(vaultPath, "_inbox")
			Expect(os.WriteFile(filepath.Join(inboxDir, "2026-01-01-"+slug+".md"), []byte("a"), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(inboxDir, "2026-02-01-"+slug+".md"), []byte("b"), 0o644)).To(Succeed())

			_, err := FindBySlug(inboxDir, slug)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("ambiguous"))
		})
	})

	Context("notes directory (exact naming)", func() {
		It("finds a file named exactly {slug}.md", func() {
			slug := "exact-slug"
			notesDir := filepath.Join(vaultPath, "notes")
			Expect(os.WriteFile(filepath.Join(notesDir, slug+".md"), []byte("body"), 0o644)).To(Succeed())

			found, err := FindBySlug(notesDir, slug)
			Expect(err).NotTo(HaveOccurred())
			Expect(found).To(Equal(filepath.Join(notesDir, slug+".md")))
		})

		It("does not match a file that merely ends with -{slug}.md", func() {
			slug := "target"
			notesDir := filepath.Join(vaultPath, "notes")
			Expect(os.WriteFile(filepath.Join(notesDir, "prefix-"+slug+".md"), []byte("body"), 0o644)).To(Succeed())

			_, err := FindBySlug(notesDir, slug)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("not found"))
		})
	})
})

var _ = Describe("Promote", func() {
	var vaultPath string

	BeforeEach(func() {
		var err error
		vaultPath, err = os.MkdirTemp("", "vault-promote-*")
		Expect(err).NotTo(HaveOccurred())
		makePromoteVault(vaultPath)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(vaultPath)).To(Succeed())
	})

	Context("happy path — observation note without --confirmed", func() {
		It("promotes the note and removes it from _inbox", func() {
			slug := "test-observation-note"
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, validObservationNoteBytes(), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("promoted"))
			Expect(result.Slug).To(Equal(slug))
			Expect(result.EpistemicType).To(Equal("observation"))

			// File must exist in notes/.
			notesPath := filepath.Join(vaultPath, "notes", slug+".md")
			Expect(notesPath).To(BeAnExistingFile())

			// File must no longer exist in _inbox/.
			Expect(inboxPath).NotTo(BeAnExistingFile())

			// Frontmatter status must be "verified".
			data, readErr := os.ReadFile(notesPath)
			Expect(readErr).NotTo(HaveOccurred())
			parsed, parseErr := Parse(data)
			Expect(parseErr).NotTo(HaveOccurred())
			Expect(parsed.Frontmatter.Status).To(Equal("verified"))
		})
	})

	Context("happy path — constraint note with --confirmed", func() {
		It("promotes the note when confirmed is true", func() {
			slug := "test-constraint-note"
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, validConstraintNoteBytes(), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("promoted"))
			Expect(result.Confirmed).To(BeTrue())

			notesPath := filepath.Join(vaultPath, "notes", slug+".md")
			Expect(notesPath).To(BeAnExistingFile())
			Expect(inboxPath).NotTo(BeAnExistingFile())
		})
	})

	Context("lint failure blocks promotion", func() {
		It("returns an error and leaves the inbox file in place", func() {
			slug := "bad-note"
			// Write a note with missing required fields (no title, type, etc.).
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, []byte("---\ntitle: \n---\nbody\n"), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("lint"))

			// Inbox file must still be present.
			Expect(inboxPath).To(BeAnExistingFile())
			// Notes file must not have been created.
			Expect(filepath.Join(vaultPath, "notes", slug+".md")).NotTo(BeAnExistingFile())
		})
	})

	Context("unresolved wikilink blocks promotion", func() {
		It("returns an error mentioning the wikilink and leaves inbox file in place", func() {
			slug := "test-observation-note"
			// Build a note with an unresolved wikilink in the body.
			today := time.Now().Format("2006-01-02")
			reviewBy := time.Now().AddDate(0, 0, 90).Format("2006-01-02")
			noteContent := []byte(`---
title: Test Observation Note
created: ` + today + `
updated: ` + today + `
review-by: ` + reviewBy + `
status: inbox
confidence: medium
type: observation
scope: project
project: agent-memory
domain:
  - go
source-agent: test-agent
source-artifact: test-artifact
requires-human-review: false
---
# Test Observation Note

## Evidence

See [[nonexistent-note-xyz]].

## Implications

Some implications.

## Related

none
`)
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, noteContent, 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("wikilink"))

			Expect(inboxPath).To(BeAnExistingFile())
			Expect(filepath.Join(vaultPath, "notes", slug+".md")).NotTo(BeAnExistingFile())
		})
	})

	Context("missing --confirmed on constraint note", func() {
		It("returns an error and leaves inbox file in place", func() {
			slug := "test-constraint-note"
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, validConstraintNoteBytes(), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("confirmation required"))

			Expect(inboxPath).To(BeAnExistingFile())
			Expect(filepath.Join(vaultPath, "notes", slug+".md")).NotTo(BeAnExistingFile())
		})
	})

	Context("--confirmed on observation note (silently ignored)", func() {
		It("promotes successfully regardless of confirmed value", func() {
			slug := "test-observation-note"
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, validObservationNoteBytes(), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, true)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("promoted"))

			Expect(filepath.Join(vaultPath, "notes", slug+".md")).To(BeAnExistingFile())
			Expect(inboxPath).NotTo(BeAnExistingFile())
		})
	})

	Context("slug collision in notes/", func() {
		It("returns a collision error and leaves inbox file in place", func() {
			slug := "test-observation-note"
			inboxPath := filepath.Join(vaultPath, "_inbox", inboxFilename(slug))
			Expect(os.WriteFile(inboxPath, validObservationNoteBytes(), 0o644)).To(Succeed())

			// Pre-create the notes/ file to trigger a collision.
			notesPath := filepath.Join(vaultPath, "notes", slug+".md")
			Expect(os.WriteFile(notesPath, []byte("existing"), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("collision"))

			Expect(inboxPath).To(BeAnExistingFile())
		})
	})

	Context("slug not found in _inbox", func() {
		It("returns a not found error", func() {
			result, err := Promote(vaultPath, "ghost-slug", false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("not found"))
		})
	})

	Context("ambiguous slug in _inbox", func() {
		It("returns an ambiguous error", func() {
			slug := "test-observation-note"
			inboxDir := filepath.Join(vaultPath, "_inbox")
			Expect(os.WriteFile(filepath.Join(inboxDir, "2026-01-01-"+slug+".md"), validObservationNoteBytes(), 0o644)).To(Succeed())
			Expect(os.WriteFile(filepath.Join(inboxDir, "2026-02-01-"+slug+".md"), validObservationNoteBytes(), 0o644)).To(Succeed())

			result, err := Promote(vaultPath, slug, false)
			Expect(err).To(HaveOccurred())
			Expect(result.Status).To(Equal("error"))
			Expect(result.Error).To(ContainSubstring("ambiguous"))
		})
	})
})
