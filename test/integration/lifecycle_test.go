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

var _ = Describe("full lifecycle", func() {
	var (
		tmpDir    string
		vaultPath string
	)

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
		vaultPath = filepath.Join(tmpDir, "vault")
		_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
		Expect(exitCode).To(Equal(0))
	})

	It("write → promote → write replacement → promote → deprecate original", func() {
		// Step 1: Write observation note "Test Note".
		bodyFile := testutil.WriteTestNote(GinkgoTB(), tmpDir, "body.md", noteBody)
		args := append([]string{"--json"}, writeNoteArgs(vaultPath, bodyFile)...)
		_, _, exitCode := testutil.RunBinary(binPath, args...)
		Expect(exitCode).To(Equal(0))

		// Step 2: Promote "test-note".
		stdout, _, exitCode := testutil.RunBinary(binPath,
			"--vault", vaultPath, "--json", "promote", "--slug", "test-note",
		)
		Expect(exitCode).To(Equal(0))
		var promoteResult map[string]interface{}
		Expect(json.Unmarshal([]byte(stdout), &promoteResult)).To(Succeed())
		Expect(promoteResult["status"]).To(Equal("promoted"))
		Expect(filepath.Join(vaultPath, "notes", "test-note.md")).To(BeAnExistingFile())

		// Step 3: Write replacement note "Replacement Note".
		replacementBody := testutil.WriteTestNote(GinkgoTB(), tmpDir, "replacement.md", `# Replacement Note

## Evidence

Replacement evidence here.

## Implications

Replacement implications here.

## Related

None.
`)
		replacementArgs := []string{
			"--vault", vaultPath,
			"write-note",
			"--type", "observation",
			"--title", "Replacement Note",
			"--scope", "cross-project",
			"--domain", "testing",
			"--source-artifact", "ci-run",
			"--confidence", "high",
			replacementBody,
		}
		_, _, exitCode = testutil.RunBinary(binPath, replacementArgs...)
		Expect(exitCode).To(Equal(0))

		// Step 4: Promote "replacement-note".
		stdout, _, exitCode = testutil.RunBinary(binPath,
			"--vault", vaultPath, "--json", "promote", "--slug", "replacement-note",
		)
		Expect(exitCode).To(Equal(0))
		var promoteResult2 map[string]interface{}
		Expect(json.Unmarshal([]byte(stdout), &promoteResult2)).To(Succeed())
		Expect(promoteResult2["status"]).To(Equal("promoted"))
		Expect(filepath.Join(vaultPath, "notes", "replacement-note.md")).To(BeAnExistingFile())

		// Step 5: Deprecate "test-note" superseded by "replacement-note".
		stdout, _, exitCode = testutil.RunBinary(binPath,
			"--vault", vaultPath, "--json", "deprecate",
			"--slug", "test-note",
			"--superseded-by", "replacement-note",
		)
		Expect(exitCode).To(Equal(0))
		var deprecateResult map[string]interface{}
		Expect(json.Unmarshal([]byte(stdout), &deprecateResult)).To(Succeed())
		Expect(deprecateResult["status"]).To(Equal("deprecated"))

		// Step 6: Assert original is in _deprecated/, replacement still in notes/.
		Expect(filepath.Join(vaultPath, "_deprecated", "test-note.md")).To(BeAnExistingFile())
		Expect(filepath.Join(vaultPath, "notes", "test-note.md")).NotTo(BeAnExistingFile())
		Expect(filepath.Join(vaultPath, "notes", "replacement-note.md")).To(BeAnExistingFile())

		// Step 7: Verify _deprecated/test-note.md contains superseded-by in frontmatter.
		deprecatedData, err := os.ReadFile(filepath.Join(vaultPath, "_deprecated", "test-note.md"))
		Expect(err).NotTo(HaveOccurred())
		Expect(string(deprecatedData)).To(ContainSubstring("superseded-by: replacement-note"))

		// Step 8: Verify log has both "promote" and "deprecate" entries.
		logData, err := os.ReadFile(filepath.Join(vaultPath, "_meta", "log.md"))
		Expect(err).NotTo(HaveOccurred())
		logContent := string(logData)
		Expect(logContent).To(ContainSubstring("promote"))
		Expect(logContent).To(ContainSubstring("deprecate"))
	})
})
