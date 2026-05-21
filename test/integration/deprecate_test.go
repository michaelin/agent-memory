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

var _ = Describe("deprecate", func() {
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

		// Write and promote the note so it's in notes/.
		args := append([]string{"--json"}, writeNoteArgs(vaultPath, bodyFile)...)
		_, _, exitCode = testutil.RunBinary(binPath, args...)
		Expect(exitCode).To(Equal(0))

		_, _, exitCode = testutil.RunBinary(binPath,
			"--vault", vaultPath, "--json", "promote", "--slug", "test-note",
		)
		Expect(exitCode).To(Equal(0))
	})

	Context("happy path", func() {
		It("deprecates a note from notes/ to _deprecated/", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath,
				"--vault", vaultPath, "--json", "deprecate",
				"--slug", "test-note",
				"--superseded-by", "new-note",
			)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("deprecated"))
			Expect(result["moved_to"]).To(ContainSubstring("_deprecated/"))

			// File should exist in _deprecated/.
			Expect(filepath.Join(vaultPath, "_deprecated", "test-note.md")).To(BeAnExistingFile())

			// File should no longer be in notes/.
			Expect(filepath.Join(vaultPath, "notes", "test-note.md")).NotTo(BeAnExistingFile())

			// Log should contain "deprecate".
			logData, err := os.ReadFile(filepath.Join(vaultPath, "_meta", "log.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(string(logData)).To(ContainSubstring("deprecate"))
		})
	})
})
