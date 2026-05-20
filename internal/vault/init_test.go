package vault

import (
	"encoding/json"
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("vault.Init", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
	})

	Context("fresh creation", func() {
		It("creates all vault directories and files", func() {
			vaultPath := filepath.Join(tmpDir, "test-vault")
			result, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("created"))

			for _, entry := range VaultStructure() {
				p := filepath.Join(vaultPath, entry.Path)
				info, err := os.Stat(p)
				Expect(err).NotTo(HaveOccurred(), "entry %s missing", entry.Path)
				if entry.IsDir {
					Expect(info.IsDir()).To(BeTrue(), "entry %s should be a directory", entry.Path)
				} else {
					Expect(info.IsDir()).To(BeFalse(), "entry %s should be a file", entry.Path)
				}
			}
		})

		It("creates directories with 0700 permissions", func() {
			vaultPath := filepath.Join(tmpDir, "test-vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(vaultPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0700)))
		})

		It("creates files with 0600 permissions", func() {
			vaultPath := filepath.Join(tmpDir, "test-vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			info, err := os.Stat(filepath.Join(vaultPath, "_meta", "writing-protocol.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode().Perm()).To(Equal(os.FileMode(0600)))
		})

		It("writes a log entry", func() {
			vaultPath := filepath.Join(tmpDir, "test-vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			logData, err := os.ReadFile(filepath.Join(vaultPath, "_meta", "log.md"))
			Expect(err).NotTo(HaveOccurred())
			Expect(logData).NotTo(BeEmpty())
		})

		It("creates vault at default path", func() {
			orig, _ := os.Getwd()
			defer os.Chdir(orig) //nolint:errcheck
			os.Chdir(tmpDir)     //nolint:errcheck

			result, err := Init(".agent-memory", InitOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("created"))
			Expect(filepath.Join(tmpDir, ".agent-memory")).To(BeADirectory())
		})
	})

	Context("idempotency", func() {
		It("returns ok on double init", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			result, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("ok"))
		})
	})

	Context("repair", func() {
		It("repairs a missing file", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			os.Remove(filepath.Join(vaultPath, "_meta", "tag-taxonomy.md")) //nolint:errcheck

			result, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("repaired"))
			Expect(result.Repaired).To(ContainElement("_meta/tag-taxonomy.md"))
		})

		It("repairs a missing directory", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			os.RemoveAll(filepath.Join(vaultPath, "_inbox")) //nolint:errcheck

			result, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("repaired"))
		})
	})

	Context("incompatible paths", func() {
		It("errors on empty path", func() {
			_, err := Init("", InitOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("empty"))
		})

		It("errors when a required directory path is a file", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			os.RemoveAll(filepath.Join(vaultPath, "_meta"))                           //nolint:errcheck
			os.WriteFile(filepath.Join(vaultPath, "_meta"), []byte("conflict"), 0600) //nolint:errcheck

			_, err = Init(vaultPath, InitOptions{})
			Expect(err).To(HaveOccurred())
		})

		It("errors when vault path is a file", func() {
			vaultPath := filepath.Join(tmpDir, "vault-file")
			os.WriteFile(vaultPath, []byte("not a dir"), 0600) //nolint:errcheck

			_, err := Init(vaultPath, InitOptions{})
			Expect(err).To(HaveOccurred())
		})

		It("errors when vault path is a symlink", func() {
			target := filepath.Join(tmpDir, "real-dir")
			Expect(os.MkdirAll(target, 0700)).To(Succeed())
			linkPath := filepath.Join(tmpDir, "link-vault")
			Expect(os.Symlink(target, linkPath)).To(Succeed())

			_, err := Init(linkPath, InitOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("symlink"))
		})
	})

	Context("with --force", func() {
		It("overwrites file-as-directory conflicts", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			os.RemoveAll(filepath.Join(vaultPath, "_meta"))                           //nolint:errcheck
			os.WriteFile(filepath.Join(vaultPath, "_meta"), []byte("conflict"), 0600) //nolint:errcheck

			result, err := Init(vaultPath, InitOptions{Force: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("repaired"))
		})

		It("removes file at vault root and creates vault", func() {
			vaultPath := filepath.Join(tmpDir, "vault-file")
			Expect(os.WriteFile(vaultPath, []byte("not a dir"), 0600)).To(Succeed())

			result, err := Init(vaultPath, InitOptions{Force: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("created"))
			Expect(vaultPath).To(BeADirectory())
		})
	})

	Context("with --clean", func() {
		It("resets vault when combined with --force", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			result, err := Init(vaultPath, InitOptions{Force: true, Clean: true})
			Expect(err).NotTo(HaveOccurred())
			Expect(result.Status).To(Equal("created"))
		})

		It("errors without --force", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{Clean: true})
			Expect(err).To(HaveOccurred())
		})

		It("refuses to remove root path", func() {
			_, err := Init("/", InitOptions{Clean: true, Force: true})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("refusing to remove root"))
		})
	})

	Context("child entry symlinks", func() {
		It("errors when a child entry is a symlink (no --force)", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Replace _meta/log.md with a symlink
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			target := filepath.Join(tmpDir, "target.md")
			Expect(os.WriteFile(target, []byte("target"), 0600)).To(Succeed())
			Expect(os.Remove(logPath)).To(Succeed())
			Expect(os.Symlink(target, logPath)).To(Succeed())

			_, err = Init(vaultPath, InitOptions{})
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("symlink"))
		})

		It("resolves symlink at child entry with --force", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			_, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			// Replace _meta/log.md with a symlink
			logPath := filepath.Join(vaultPath, "_meta", "log.md")
			target := filepath.Join(tmpDir, "target.md")
			Expect(os.WriteFile(target, []byte("target"), 0600)).To(Succeed())
			Expect(os.Remove(logPath)).To(Succeed())
			Expect(os.Symlink(target, logPath)).To(Succeed())

			_, err = Init(vaultPath, InitOptions{Force: true})
			Expect(err).NotTo(HaveOccurred())

			// log.md should now be a real file, not a symlink
			info, err := os.Lstat(logPath)
			Expect(err).NotTo(HaveOccurred())
			Expect(info.Mode() & os.ModeSymlink).To(Equal(os.FileMode(0)))
		})
	})

	Context("JSON serialization", func() {
		It("produces valid JSON with status and vault fields", func() {
			vaultPath := filepath.Join(tmpDir, "vault")
			result, err := Init(vaultPath, InitOptions{})
			Expect(err).NotTo(HaveOccurred())

			b, err := json.Marshal(result)
			Expect(err).NotTo(HaveOccurred())

			var m map[string]interface{}
			Expect(json.Unmarshal(b, &m)).To(Succeed())
			Expect(m["status"]).To(Equal("created"))
			Expect(m["vault"]).NotTo(BeEmpty())
		})
	})
})
