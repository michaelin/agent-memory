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

var binPath string

var _ = BeforeSuite(func() {
	binPath = testutil.BuildBinary(GinkgoTB())
})

var _ = Describe("agent-memory init", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
	})

	Context("with no existing vault", func() {
		It("creates vault in cwd when no path given (--json)", func() {
			origDir, _ := os.Getwd()
			defer os.Chdir(origDir) //nolint:errcheck
			os.Chdir(tmpDir)        //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init")
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("created"))

			Expect(filepath.Join(tmpDir, ".agent-memory", "_meta")).To(BeADirectory())
			Expect(filepath.Join(tmpDir, ".agent-memory", "_inbox")).To(BeADirectory())
			Expect(filepath.Join(tmpDir, ".agent-memory", "_contested")).To(BeADirectory())
			Expect(filepath.Join(tmpDir, ".agent-memory", "notes")).To(BeADirectory())
			Expect(filepath.Join(tmpDir, ".agent-memory", "_meta", "writing-protocol.md")).To(BeAnExistingFile())
		})

		It("creates vault at specified path (--json)", func() {
			vaultPath := filepath.Join(tmpDir, "my-vault")
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("created"))
			Expect(vaultPath).To(BeADirectory())
		})

		It("creates vault with human-readable output (no --json)", func() {
			vaultPath := filepath.Join(tmpDir, "my-vault")
			stdout, _, exitCode := testutil.RunBinary(binPath, "init", vaultPath)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("Vault created at"))
			Expect(stdout).To(ContainSubstring(vaultPath))
		})
	})

	Context("with existing vault", func() {
		var vaultPath string

		BeforeEach(func() {
			vaultPath = filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))
		})

		It("is idempotent on double init (--json)", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("ok"))
		})

		It("is idempotent on double init (human-readable)", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath, "init", vaultPath)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("Vault already up to date"))
		})

		It("repairs missing directories (--json)", func() {
			os.RemoveAll(filepath.Join(vaultPath, "_inbox")) //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("repaired"))
			Expect(filepath.Join(vaultPath, "_inbox")).To(BeADirectory())
		})

		It("repairs missing directories (human-readable)", func() {
			os.RemoveAll(filepath.Join(vaultPath, "_inbox")) //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "init", vaultPath)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("Repaired:"))
			Expect(stdout).To(ContainSubstring("_inbox"))
		})
	})

	Context("with incompatible path", func() {
		It("errors when vault path is a file (--json)", func() {
			filePath := filepath.Join(tmpDir, "not-a-dir")
			os.WriteFile(filePath, []byte("conflict"), 0600) //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", filePath)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["error"]).NotTo(BeEmpty())
		})

		It("errors when vault path is a file (human-readable)", func() {
			filePath := filepath.Join(tmpDir, "not-a-dir")
			os.WriteFile(filePath, []byte("conflict"), 0600) //nolint:errcheck

			_, stderr, exitCode := testutil.RunBinary(binPath, "init", filePath)
			Expect(exitCode).To(Equal(1))
			Expect(stderr).NotTo(BeEmpty())
		})

		It("errors when vault path is a symlink", func() {
			target := filepath.Join(tmpDir, "real-dir")
			Expect(os.MkdirAll(target, 0700)).To(Succeed())
			linkPath := filepath.Join(tmpDir, "link-vault")
			Expect(os.Symlink(target, linkPath)).To(Succeed())

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", linkPath)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["error"]).To(ContainSubstring("symlink"))
		})

		It("errors on empty string path", func() {
			_, stderr, exitCode := testutil.RunBinary(binPath, "init", "")
			Expect(exitCode).To(Equal(1))
			Expect(stderr).To(ContainSubstring("empty"))
		})

		It("errors when a child entry is a symlink (no --force)", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			// Replace _meta/log.md with a symlink
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			target := filepath.Join(tmpDir, "target.md")
			Expect(os.WriteFile(target, []byte("target"), 0600)).To(Succeed())
			Expect(os.Remove(logPath)).To(Succeed())
			Expect(os.Symlink(target, logPath)).To(Succeed())

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["error"]).To(ContainSubstring("symlink"))
		})

		It("resolves child symlink with --force", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			// Replace _meta/log.md with a symlink
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			target := filepath.Join(tmpDir, "target.md")
			Expect(os.WriteFile(target, []byte("target"), 0600)).To(Succeed())
			Expect(os.Remove(logPath)).To(Succeed())
			Expect(os.Symlink(target, logPath)).To(Succeed())

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", "--force", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).NotTo(BeEmpty())

			// log.md should now be a real file
			info, err := os.Lstat(logPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModeSymlink).To(Equal(os.FileMode(0)))
		})

		It("does not write to stderr when --json is set and an error occurs", func() {
			filePath := filepath.Join(tmpDir, "not-a-dir-stderr")
			os.WriteFile(filePath, []byte("conflict"), 0600) //nolint:errcheck

			stdout, stderr, exitCode := testutil.RunBinary(binPath, "--json", "init", filePath)
			Expect(exitCode).To(Equal(1))
			Expect(stderr).To(BeEmpty())

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["error"]).NotTo(BeEmpty())
		})
	})

	Context("with --force", func() {
		It("overwrites file-as-directory conflicts", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			os.RemoveAll(filepath.Join(vaultPath, "_meta"))                           //nolint:errcheck
			os.WriteFile(filepath.Join(vaultPath, "_meta"), []byte("conflict"), 0600) //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", "--force", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("repaired"))
			Expect(filepath.Join(vaultPath, "_meta")).To(BeADirectory())
		})

		It("removes file at vault root and creates vault", func() {
			vaultPath := filepath.Join(tmpDir, "vault-file")
			Expect(os.WriteFile(vaultPath, []byte("not a dir"), 0600)).To(Succeed())

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", "--force", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("created"))
			Expect(vaultPath).To(BeADirectory())
		})
	})

	Context("with --clean --force", func() {
		It("resets the vault", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "--json", "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			os.WriteFile(filepath.Join(vaultPath, "custom.txt"), []byte("custom"), 0600) //nolint:errcheck

			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", "--clean", "--force", vaultPath)
			Expect(exitCode).To(Equal(0))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["status"]).To(Equal("created"))

			_, err := os.Stat(filepath.Join(vaultPath, "custom.txt"))
			Expect(os.IsNotExist(err)).To(BeTrue())
		})
	})

	Context("with --clean without --force", func() {
		It("errors", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "init", "--clean", vaultPath)
			Expect(exitCode).To(Equal(1))

			var result map[string]interface{}
			Expect(json.Unmarshal([]byte(stdout), &result)).To(Succeed())
			Expect(result["error"]).To(ContainSubstring("--clean requires --force"))
		})
	})

	Describe("log entry", func() {
		It("writes init log with ISO 8601 timestamp", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, _, exitCode := testutil.RunBinary(binPath, "init", vaultPath)
			Expect(exitCode).To(Equal(0))

			logData, err := os.ReadFile(filepath.Join(vaultPath, "_meta", "log.md"))
			Expect(err).NotTo(HaveOccurred())
			logContent := string(logData)
			Expect(logContent).To(ContainSubstring("init | vault initialized"))
			Expect(logContent).To(MatchRegexp(`\[\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}`))
		})
	})
})

var _ = Describe("agent-memory instructions", func() {
	It("outputs agent blurb to stdout", func() {
		stdout, _, exitCode := testutil.RunBinary(binPath, "instructions")
		Expect(exitCode).To(Equal(0))
		Expect(stdout).To(HavePrefix("# Agent Memory"))
		Expect(stdout).To(ContainSubstring("write-note"))
		Expect(stdout).To(ContainSubstring("--json"))
	})
})
