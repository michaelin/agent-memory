package note

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

// makeVault creates a minimal vault directory structure under root.
func makeVault(root string) {
	GinkgoHelper()
	for _, dir := range []string{"_inbox", "notes", "_meta"} {
		Expect(os.MkdirAll(filepath.Join(root, dir), 0o755)).To(Succeed())
	}
}

// validWriteOpts returns a WriteOptions that passes lint for an observation note.
func validWriteOpts(vaultPath string) WriteOptions {
	return WriteOptions{
		VaultPath:      vaultPath,
		Title:          "Goroutines Are Lightweight",
		EpistemicType:  "observation",
		Body:           "# Goroutines Are Lightweight\n\n## Evidence\n\nsome evidence\n\n## Implications\n\nsome implications\n\n## Related\n\nnone",
		Scope:          "project",
		Project:        "agent-memory",
		SourceArtifact: "test-artifact",
		SourceAgent:    "test-agent",
		Domain:         []string{"go"},
	}
}

var _ = Describe("Write", func() {
	var vaultPath string

	BeforeEach(func() {
		var err error
		vaultPath, err = os.MkdirTemp("", "vault-*")
		Expect(err).NotTo(HaveOccurred())
		makeVault(vaultPath)
	})

	AfterEach(func() {
		Expect(os.RemoveAll(vaultPath)).To(Succeed())
	})

	Context("happy path", func() {
		It("writes the file and returns status written", func() {
			opts := validWriteOpts(vaultPath)
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))
			Expect(result.Path).NotTo(BeEmpty())

			// File must exist on disk.
			_, statErr := os.Stat(result.Path)
			Expect(statErr).NotTo(HaveOccurred())
		})

		It("places the file in _inbox with YYYY-MM-DD-slug.md naming", func() {
			opts := validWriteOpts(vaultPath)
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			today := time.Now().Format("2006-01-02")
			slug := Slug(opts.Title)
			expectedName := today + "-" + slug + ".md"
			Expect(filepath.Base(result.Path)).To(Equal(expectedName))
			Expect(filepath.Dir(result.Path)).To(Equal(filepath.Join(vaultPath, "_inbox")))
		})
	})

	Context("lint failure", func() {
		It("refuses with lint errors when required fields are missing", func() {
			opts := WriteOptions{
				VaultPath: vaultPath,
				// Title, EpistemicType, Scope, SourceArtifact all empty → NF001 errors
			}
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("refused"))
			Expect(result.Reason).To(Equal("lint errors"))
			Expect(result.Errors).NotTo(BeEmpty())
		})
	})

	Context("similarity refusal", func() {
		It("refuses when an existing note has a similar title", func() {
			// Write a note first.
			opts := validWriteOpts(vaultPath)
			first, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(first.Status).To(Equal("written"))

			// Attempt to write a nearly identical note.
			// Tokens for "Goroutines Lightweight" = [goroutines, lightweight]
			// Tokens for "Goroutines Lightweight Go" = [goroutines, lightweight, go]
			// Jaccard = 2/3 ≈ 0.667 — not enough.
			// Use same tokens with one extra to get 2/3; instead use exact same
			// title to guarantee Jaccard = 1.0 (Force=false, different body heading).
			opts2 := validWriteOpts(vaultPath)
			// Same title → Jaccard = 1.0, guaranteed refusal.
			opts2.Body = strings.ReplaceAll(opts2.Body, "Goroutines Are Lightweight", opts2.Title)
			result, err := Write(opts2)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("refused"))
			Expect(result.Reason).To(Equal("similar note exists"))
			Expect(result.Candidates).NotTo(BeEmpty())
		})
	})

	Context("force override", func() {
		It("bypasses similarity refusal when Force is true", func() {
			opts := validWriteOpts(vaultPath)
			_, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			opts2 := validWriteOpts(vaultPath)
			opts2.Title = "Goroutines Are Lightweight Threads"
			opts2.Body = strings.ReplaceAll(opts2.Body, "Goroutines Are Lightweight", "Goroutines Are Lightweight Threads")
			opts2.Force = true
			result, err := Write(opts2)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))
		})
	})

	Context("wikilink warnings", func() {
		It("warns about unresolved wikilinks in the body", func() {
			opts := validWriteOpts(vaultPath)
			opts.Body += "\n\nSee also [[NonExistentNote]]."
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))
			Expect(result.Warnings).To(ContainElement(ContainSubstring("NonExistentNote")))
		})

		It("does not warn when the wikilink resolves to a notes/ file", func() {
			// Create the target note file.
			targetSlug := Slug("ExistingNote")
			targetPath := filepath.Join(vaultPath, "notes", targetSlug+".md")
			Expect(os.WriteFile(targetPath, []byte("---\ntitle: ExistingNote\n---\nbody"), 0o644)).To(Succeed())

			opts := validWriteOpts(vaultPath)
			opts.Body += "\n\nSee also [[ExistingNote]]."
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Warnings).NotTo(ContainElement(ContainSubstring("ExistingNote")))
		})
	})

	Context("file collision", func() {
		It("appends -2 suffix when the slug already exists", func() {
			opts := validWriteOpts(vaultPath)
			first, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(first.Status).To(Equal("written"))

			// Force a second write with the same slug (bypass similarity check).
			opts.Force = true
			second, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(second.Status).To(Equal("written"))

			today := time.Now().Format("2006-01-02")
			slug := Slug(opts.Title)
			Expect(filepath.Base(second.Path)).To(Equal(today + "-" + slug + "-2.md"))
		})
	})

	Context("AppendLog failure is best-effort", func() {
		It("preserves the note file and returns status written even when AppendLog cannot write", func() {
			if os.Getuid() == 0 {
				Skip("skipping: running as root")
			}

			metaDir := filepath.Join(vaultPath, "_meta")
			Expect(os.Chmod(metaDir, 0o555)).To(Succeed())
			defer os.Chmod(metaDir, 0o755) //nolint:errcheck // best-effort restore for cleanup

			opts := validWriteOpts(vaultPath)
			result, err := Write(opts)

			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))
			Expect(result.Path).NotTo(BeEmpty())

			_, statErr := os.Stat(result.Path)
			Expect(statErr).NotTo(HaveOccurred())
		})
	})

	Context("log entry", func() {
		It("appends an entry to _meta/log.md", func() {
			opts := validWriteOpts(vaultPath)
			_, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			data, readErr := os.ReadFile(logPath)
			Expect(readErr).NotTo(HaveOccurred())

			slug := Slug(opts.Title)
			Expect(string(data)).To(ContainSubstring("write | observation | " + slug))
			Expect(string(data)).To(ContainSubstring("by:test-agent"))
		})

		It("creates _meta/log.md if it does not exist", func() {
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			Expect(logPath).NotTo(BeAnExistingFile())

			opts := validWriteOpts(vaultPath)
			_, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			Expect(logPath).To(BeAnExistingFile())
		})
	})

	Context("auto-populated fields", func() {
		It("sets created, updated, status, confidence, review-by, requires-human-review", func() {
			opts := validWriteOpts(vaultPath)
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))

			data, readErr := os.ReadFile(result.Path)
			Expect(readErr).NotTo(HaveOccurred())

			parsed, parseErr := Parse(data)
			Expect(parseErr).NotTo(HaveOccurred())

			today := time.Now().Format("2006-01-02")
			Expect(parsed.Frontmatter.Created).To(Equal(today))
			Expect(parsed.Frontmatter.Updated).To(Equal(today))
			Expect(parsed.Frontmatter.Status).To(Equal("inbox"))
			Expect(parsed.Frontmatter.Confidence).To(Equal("medium"))
			Expect(parsed.Frontmatter.ReviewBy).NotTo(BeEmpty())
			// observation does not require human review
			Expect(parsed.Frontmatter.RequiresHumanReview).To(BeFalse())
		})

		It("defaults confidence to medium when not provided", func() {
			opts := validWriteOpts(vaultPath)
			opts.Confidence = ""
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			data, _ := os.ReadFile(result.Path)
			parsed, _ := Parse(data)
			Expect(parsed.Frontmatter.Confidence).To(Equal("medium"))
		})

		It("respects an explicit confidence value", func() {
			opts := validWriteOpts(vaultPath)
			opts.Confidence = "high"
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			data, _ := os.ReadFile(result.Path)
			parsed, _ := Parse(data)
			Expect(parsed.Frontmatter.Confidence).To(Equal("high"))
		})
	})

	Context("TTL / review-by dates", func() {
		DescribeTable("each epistemic type gets the correct review-by offset",
			func(epistemicType string, days int, humanReview bool) {
				opts := validWriteOpts(vaultPath)
				opts.EpistemicType = epistemicType
				// Adjust body heading to match title (lint NF005 requires h1).
				opts.Body = "# Goroutines Are Lightweight\n\n## Evidence\n\nsome evidence\n\n## Implications\n\nsome implications\n\n## Related\n\nnone"
				if epistemicType == "synthesis" {
					opts.Body = "# Goroutines Are Lightweight\n\n## Synthesis\n\nsome synthesis\n\n## Contributing notes\n\nnone\n\n## Related\n\nnone"
				}
				result, err := Write(opts)
				Expect(err).NotTo(HaveOccurred())
				Expect(result.Status).To(Equal("written"), "epistemic-type=%s result=%+v", epistemicType, result)

				data, _ := os.ReadFile(result.Path)
				parsed, _ := Parse(data)

				expected := time.Now().AddDate(0, 0, days).Format("2006-01-02")
				Expect(parsed.Frontmatter.ReviewBy).To(Equal(expected))
				Expect(parsed.Frontmatter.RequiresHumanReview).To(Equal(humanReview))
			},
			Entry("observation → 90 days", "observation", 90, false),
			Entry("pattern → 180 days", "pattern", 180, false),
			Entry("assumption → 30 days", "assumption", 30, false),
			Entry("constraint → 365 days", "constraint", 365, true),
			Entry("decision → 365 days", "decision", 365, true),
		)
	})

	Context("source-agent detection", func() {
		It("uses DetectSourceAgent() when SourceAgent is empty", func() {
			// Ensure no env var is set so DetectSourceAgent returns "".
			Expect(os.Unsetenv("OPENCODE_RUN_ID")).To(Succeed())

			opts := validWriteOpts(vaultPath)
			opts.SourceAgent = ""
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))

			data, _ := os.ReadFile(result.Path)
			parsed, _ := Parse(data)
			// With no env var, DetectSourceAgent returns ""; source-agent field is empty.
			Expect(parsed.Frontmatter.SourceAgent).To(Equal(""))
		})

		It("uses the OPENCODE_RUN_ID env var via DetectSourceAgent", func() {
			Expect(os.Setenv("OPENCODE_RUN_ID", "run-42")).To(Succeed())
			DeferCleanup(func() { os.Unsetenv("OPENCODE_RUN_ID") })

			opts := validWriteOpts(vaultPath)
			opts.SourceAgent = ""
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("written"))

			data, _ := os.ReadFile(result.Path)
			parsed, _ := Parse(data)
			Expect(parsed.Frontmatter.SourceAgent).To(Equal("opencode:run-42"))
		})

		It("prefers explicit SourceAgent over DetectSourceAgent", func() {
			Expect(os.Setenv("OPENCODE_RUN_ID", "run-99")).To(Succeed())
			DeferCleanup(func() { os.Unsetenv("OPENCODE_RUN_ID") })

			opts := validWriteOpts(vaultPath)
			opts.SourceAgent = "explicit-agent"
			result, err := Write(opts)
			Expect(err).NotTo(HaveOccurred())

			data, _ := os.ReadFile(result.Path)
			parsed, _ := Parse(data)
			Expect(parsed.Frontmatter.SourceAgent).To(Equal("explicit-agent"))
		})
	})
})

var _ = Describe("ResolveWikilinks", func() {
	var vaultPath string

	BeforeEach(func() {
		var err error
		vaultPath, err = os.MkdirTemp("", "vault-resolve-*")
		Expect(err).NotTo(HaveOccurred())
		for _, dir := range []string{"_inbox", "notes", "_deprecated", "_meta"} {
			Expect(os.MkdirAll(filepath.Join(vaultPath, dir), 0o755)).To(Succeed())
		}
	})

	AfterEach(func() {
		Expect(os.RemoveAll(vaultPath)).To(Succeed())
	})

	It("returns a warning for an unresolved wikilink", func() {
		warnings := ResolveWikilinks(vaultPath, "See [[ghost-note]].")
		Expect(warnings).To(ContainElement(ContainSubstring("ghost-note")))
	})

	It("resolves a wikilink to a notes/ file", func() {
		slug := Slug("NotesTarget")
		Expect(os.WriteFile(filepath.Join(vaultPath, "notes", slug+".md"), []byte("body"), 0o644)).To(Succeed())
		warnings := ResolveWikilinks(vaultPath, fmt.Sprintf("See [[%s]].", "NotesTarget"))
		Expect(warnings).To(BeEmpty())
	})

	It("resolves a wikilink to an _inbox/ file (date-slug naming)", func() {
		slug := Slug("InboxTarget")
		name := "2026-01-01-" + slug + ".md"
		Expect(os.WriteFile(filepath.Join(vaultPath, "_inbox", name), []byte("body"), 0o644)).To(Succeed())
		warnings := ResolveWikilinks(vaultPath, fmt.Sprintf("See [[%s]].", "InboxTarget"))
		Expect(warnings).To(BeEmpty())
	})

	It("resolves a wikilink to a _deprecated/ file", func() {
		slug := Slug("DeprecatedTarget")
		Expect(os.WriteFile(filepath.Join(vaultPath, "_deprecated", slug+".md"), []byte("body"), 0o644)).To(Succeed())
		warnings := ResolveWikilinks(vaultPath, fmt.Sprintf("See [[%s]].", "DeprecatedTarget"))
		Expect(warnings).To(BeEmpty())
	})

	It("does not resolve a wikilink to a _meta/ file", func() {
		slug := Slug("MetaTarget")
		Expect(os.WriteFile(filepath.Join(vaultPath, "_meta", slug+".md"), []byte("body"), 0o644)).To(Succeed())
		warnings := ResolveWikilinks(vaultPath, fmt.Sprintf("See [[%s]].", "MetaTarget"))
		Expect(warnings).To(ContainElement(ContainSubstring("MetaTarget")))
	})

	It("returns no warnings for an empty body", func() {
		warnings := ResolveWikilinks(vaultPath, "no links here")
		Expect(warnings).To(BeEmpty())
	})
})
