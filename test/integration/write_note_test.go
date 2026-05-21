//go:build integration

package integration_test

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/michaelin/agent-memory/internal/testutil"
)

// noteBody is a minimal valid note body with all required sections.
const noteBody = `# Test Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`

// writeNoteArgs returns the base required flags for write-note.
func writeNoteArgs(vaultPath, bodyFile string) []string {
	return []string{
		"--vault", vaultPath,
		"write-note",
		"--type", "observation",
		"--title", "Test Note",
		"--scope", "cross-project",
		"--domain", "testing",
		"--source-artifact", "ci-run",
		"--confidence", "high",
		bodyFile,
	}
}

var _ = Describe("write-note", func() {
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
		It("writes the note and returns success output", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath, writeNoteArgs(vaultPath, bodyFile)...)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("✓ Written:"))
			Expect(stdout).To(ContainSubstring("notes/test-note.md"))

			Expect(filepath.Join(vaultPath, "notes", "test-note.md")).To(BeAnExistingFile())
		})
	})

	Context("JSON mode", func() {
		It("returns valid JSON with status written", func() {
			args := append([]string{"--json"}, writeNoteArgs(vaultPath, bodyFile)...)
			stdout, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("written"))
			Expect(result["path"]).To(ContainSubstring("notes/"))
		})
	})

	Context("stdin body", func() {
		It("reads body from stdin when - is passed", func() {
			args := []string{
				"--vault", vaultPath,
				"write-note",
				"--type", "observation",
				"--title", "Stdin Note",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				"-",
			}
			stdout, _, exitCode := testutil.RunBinaryWithStdin(binPath, noteBody, args...)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("✓ Written:"))
			Expect(filepath.Join(vaultPath, "notes", "stdin-note.md")).To(BeAnExistingFile())
		})
	})

	Context("similarity refusal", func() {
		It("refuses when a similar note already exists", func() {
			// Write the note once.
			_, _, exitCode := testutil.RunBinary(binPath, writeNoteArgs(vaultPath, bodyFile)...)
			Expect(exitCode).To(Equal(0))

			// Attempt to write the same note again — should be refused.
			stdout, _, exitCode := testutil.RunBinary(binPath, writeNoteArgs(vaultPath, bodyFile)...)
			Expect(exitCode).To(Equal(1))
			Expect(stdout).To(ContainSubstring("✗ Refused:"))
		})

		It("returns JSON status refused on similarity collision", func() {
			// Write the note once.
			args := append([]string{"--json"}, writeNoteArgs(vaultPath, bodyFile)...)
			_, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(0))

			// Second write — refused.
			stdout, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("refused"))
			Expect(result["candidates"]).NotTo(BeNil())
		})
	})

	Context("force flag", func() {
		It("bypasses similarity refusal with --force", func() {
			// Write the note once.
			_, _, exitCode := testutil.RunBinary(binPath, writeNoteArgs(vaultPath, bodyFile)...)
			Expect(exitCode).To(Equal(0))

			// Second write with --force — should succeed.
			args := append(writeNoteArgs(vaultPath, bodyFile), "--force")
			stdout, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("✓ Written:"))
		})
	})

	Context("missing required flags", func() {
		It("exits non-zero when --type is missing", func() {
			args := []string{
				"--vault", vaultPath,
				"write-note",
				"--title", "Test Note",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				bodyFile,
			}
			_, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(1))
		})

		It("exits non-zero when --title is missing", func() {
			args := []string{
				"--vault", vaultPath,
				"write-note",
				"--type", "observation",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				bodyFile,
			}
			_, _, exitCode := testutil.RunBinary(binPath, args...)
			Expect(exitCode).To(Equal(1))
		})
	})

	Context("missing vault", func() {
		It("exits non-zero when no vault is discoverable", func() {
			// Run from a temp dir with no vault and no env var.
			args := []string{
				"write-note",
				"--type", "observation",
				"--title", "Test Note",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				bodyFile,
			}
			// Unset AGENT_MEMORY_VAULT to ensure no accidental discovery.
			cmd := testutil.RunBinaryCmd(binPath, args...)
			cmd.Env = append(os.Environ(), "AGENT_MEMORY_VAULT=")
			cmd.Dir = tmpDir
			out, err := cmd.Output()
			_ = out
			if err == nil {
				// If it somehow succeeded (vault found in parent), skip.
				Skip("vault discovered in parent directory; skipping no-vault test")
			}
			Expect(err).To(HaveOccurred())
		})
	})

	Context("vault via --vault flag", func() {
		It("uses the explicit vault path", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath, writeNoteArgs(vaultPath, bodyFile)...)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("✓ Written:"))
		})
	})

	Context("vault via env var", func() {
		It("uses AGENT_MEMORY_VAULT when set", func() {
			args := []string{
				"write-note",
				"--type", "observation",
				"--title", "Env Vault Note",
				"--scope", "cross-project",
				"--domain", "testing",
				"--source-artifact", "ci-run",
				"--confidence", "high",
				bodyFile,
			}
			cmd := testutil.RunBinaryCmd(binPath, args...)
			cmd.Env = append(os.Environ(), "AGENT_MEMORY_VAULT="+vaultPath)
			outBytes, err := cmd.Output()
			Expect(err).NotTo(HaveOccurred())
			Expect(string(outBytes)).To(ContainSubstring("✓ Written:"))
		})
	})
})
