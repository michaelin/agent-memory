//go:build integration

package integration_test

import (
	"encoding/json"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/michaelin/agent-memory/internal/testutil"
)

var _ = Describe("promote", func() {
	var (
		tmpDir    string
		vaultPath string
		bodyFile  string
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		vaultPath = filepath.Join(tmpDir, "vault")
		_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
		Expect(exitCode).To(Equal(0))

		bodyFile = testutil.WriteTestNote(GinkgoTB(), tmpDir, "body.md", noteBody)
	})

	Context("happy path", func() {
		It("promotes a note from _inbox/ to notes/", func() {
			// Write the note first.
			args := append([]string{"--json"}, writeNoteArgs(vaultPath, bodyFile)...)
			stdout, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(0))

			var writeResult map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &writeResult)).To(Succeed())
			Expect(writeResult["status"]).To(Equal("written"))

			// Promote it.
			stdout, _, exitCode = testutil.RunBinary(binPath,
				"--vault", vaultPath, "--json", "promote", "--slug", "test-note",
			)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("promoted"))
			Expect(result["slug"]).To(Equal("test-note"))
			Expect(result["to"]).To(ContainSubstring("notes/"))

			// File should exist in notes/.
			Expect(filepath.Join(vaultPath, "notes", "test-note.md")).To(BeAnExistingFile())

			// File should no longer be in _inbox/.
			matches, err := filepath.Glob(filepath.Join(vaultPath, "_inbox", "*test-note.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(matches).To(BeEmpty())
		})
	})

	Context("constraint note without --confirmed", func() {
		It("returns exit code 1 and JSON error", func() {
			// Write a constraint note.
			constraintBody := testutil.WriteTestNote(GinkgoTB(), tmpDir, "constraint.md", `# Constraint Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`)
			constraintArgs := []string{
				"--vault", vaultPath,
				"write-note",
				"--type", "constraint",
				"--title", "Constraint Note",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				constraintBody,
			}
			_, _, exitCode := testutil.RunBinary(binPath, constraintArgs...)
			Expect(exitCode).To(Equal(0))

			// Attempt to promote without --confirmed.
			stdout, _, exitCode := testutil.RunBinary(binPath,
				"--vault", vaultPath, "--json", "promote", "--slug", "constraint-note",
			)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("error"))
			Expect(result["error"]).NotTo(BeEmpty())
		})
	})
})
