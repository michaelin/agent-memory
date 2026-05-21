package vault

import (
	"os"
	"path/filepath"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

var _ = Describe("Discover", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
	})

	Context("explicit path exists", func() {
		It("returns the absolute path of the given directory", func() {
			result, err := Discover(tmpDir)
			Expect(err).NotTo(HaveOccurred())
			abs, _ := filepath.Abs(tmpDir)
			Expect(result).To(Equal(abs))
		})
	})

	Context("explicit path does not exist", func() {
		It("returns an error", func() {
			_, err := Discover(filepath.Join(tmpDir, "nonexistent"))
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("does not exist"))
		})
	})

	Context("env var set to existing dir", func() {
		It("returns the absolute path from the env var", func() {
			GinkgoT().Setenv("AGENT_MEMORY_VAULT", tmpDir)
			result, err := Discover("")
			Expect(err).NotTo(HaveOccurred())
			abs, _ := filepath.Abs(tmpDir)
			Expect(result).To(Equal(abs))
		})
	})

	Context("env var set to nonexistent dir", func() {
		It("returns an error", func() {
			GinkgoT().Setenv("AGENT_MEMORY_VAULT", filepath.Join(tmpDir, "missing"))
			_, err := Discover("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("AGENT_MEMORY_VAULT"))
		})
	})

	Context("walk-up finds vault in parent", func() {
		It("returns the .agent-memory absolute path", func() {
			// Create: tmpDir/.agent-memory/ and tmpDir/child/
			vaultDir := filepath.Join(tmpDir, ".agent-memory")
			Expect(os.Mkdir(vaultDir, 0700)).To(Succeed())
			childDir := filepath.Join(tmpDir, "child")
			Expect(os.Mkdir(childDir, 0700)).To(Succeed())

			original, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(original) }()

			Expect(os.Chdir(childDir)).To(Succeed())

			result, err := Discover("")
			Expect(err).NotTo(HaveOccurred())
			// Resolve symlinks on both sides to handle macOS /var → /private/var.
			resolvedResult, err := filepath.EvalSymlinks(result)
			Expect(err).NotTo(HaveOccurred())
			resolvedVault, err := filepath.EvalSymlinks(vaultDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolvedResult).To(Equal(resolvedVault))
		})
	})

	Context("walk-up finds vault in grandparent", func() {
		It("returns the .agent-memory absolute path two levels up", func() {
			vaultDir := filepath.Join(tmpDir, ".agent-memory")
			Expect(os.Mkdir(vaultDir, 0700)).To(Succeed())
			nested := filepath.Join(tmpDir, "a", "b")
			Expect(os.MkdirAll(nested, 0700)).To(Succeed())

			original, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(original) }()

			Expect(os.Chdir(nested)).To(Succeed())

			result, err := Discover("")
			Expect(err).NotTo(HaveOccurred())
			resolvedResult, err := filepath.EvalSymlinks(result)
			Expect(err).NotTo(HaveOccurred())
			resolvedVault, err := filepath.EvalSymlinks(vaultDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolvedResult).To(Equal(resolvedVault))
		})
	})

	Context("no vault found", func() {
		It("returns an error when no .agent-memory exists in the tree", func() {
			original, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(original) }()

			Expect(os.Chdir(tmpDir)).To(Succeed())

			_, err = Discover("")
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("no agent-memory vault found"))
		})
	})

	Context("cwd is inside the vault dir itself", func() {
		It("returns the vault dir when cwd IS .agent-memory", func() {
			vaultDir := filepath.Join(tmpDir, ".agent-memory")
			Expect(os.Mkdir(vaultDir, 0700)).To(Succeed())

			original, err := os.Getwd()
			Expect(err).NotTo(HaveOccurred())
			defer func() { _ = os.Chdir(original) }()

			Expect(os.Chdir(vaultDir)).To(Succeed())

			result, err := Discover("")
			Expect(err).NotTo(HaveOccurred())
			resolvedResult, err := filepath.EvalSymlinks(result)
			Expect(err).NotTo(HaveOccurred())
			resolvedVault, err := filepath.EvalSymlinks(vaultDir)
			Expect(err).NotTo(HaveOccurred())
			Expect(resolvedResult).To(Equal(resolvedVault))
		})
	})
})
