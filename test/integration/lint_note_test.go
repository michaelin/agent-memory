//go:build integration

package integration_test

import (
	"encoding/json"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/michaelin/agent-memory/internal/testutil"
)

const validNoteContent = `---
title: Test Note
created: 2026-01-01
updated: 2026-01-01
status: verified
confidence: high
type: observation
scope: cross-project
source-agent: test-agent
source-artifact: test-artifact
domain:
  - testing
---
# Test Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`

const missingTitleContent = `---
created: 2026-01-01
updated: 2026-01-01
status: verified
confidence: high
type: observation
scope: cross-project
source-agent: test-agent
source-artifact: test-artifact
domain:
  - testing
---
# Test Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`

const invalidEnumContent = `---
title: Test Note
created: 2026-01-01
updated: 2026-01-01
status: invalid-value
confidence: high
type: observation
scope: cross-project
source-agent: test-agent
source-artifact: test-artifact
domain:
  - testing
---
# Test Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`

const malformedYAMLContent = "---\ntitle: [unclosed\n---\n# Title\n"

const missingBodySectionContent = `---
title: Test Note
created: 2026-01-01
updated: 2026-01-01
status: verified
confidence: high
type: observation
scope: cross-project
source-agent: test-agent
source-artifact: test-artifact
domain:
  - testing
---
# Test Note

## Related

None.
`

const placeholderContent = `---
title: <title>
created: 2026-01-01
updated: 2026-01-01
status: verified
confidence: high
type: observation
scope: cross-project
source-agent: test-agent
source-artifact: test-artifact
domain:
  - testing
---
# Test Note

## Evidence

Some evidence here.

## Implications

Some implications here.

## Related

None.
`

// parseLintResult unmarshals JSON lint output into a map.
func parseLintResult(stdout string) map[string]interface{} {
	var result map[string]interface{}
	ExpectWithOffset(1, json.Unmarshal([]byte(stdout), &result)).To(Succeed())
	return result
}

// lintResultHasRule returns true if the errors array in result contains an entry with the given rule.
func lintResultHasRule(result map[string]interface{}, rule string) bool {
	errs, ok := result["errors"].([]interface{})
	if !ok {
		return false
	}
	for _, e := range errs {
		entry, ok := e.(map[string]interface{})
		if !ok {
			continue
		}
		if entry["rule"] == rule {
			return true
		}
	}
	return false
}

var _ = Describe("lint-note", func() {
	var tmpDir string

	BeforeEach(func() {
		tmpDir = GinkgoT().TempDir()
	})

	Context("valid note", func() {
		It("exits 0 and returns JSON valid:true", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "valid.md", validNoteContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(0))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeTrue())
		})
	})

	Context("missing required field", func() {
		It("exits 1 and JSON errors contain NF001", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "missing-title.md", missingTitleContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
			Expect(lintResultHasRule(result, "NF001")).To(BeTrue(), "expected NF001 in errors: %v", result["errors"])
		})
	})

	Context("invalid enum value", func() {
		It("exits 1 and JSON errors contain NF003", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "invalid-enum.md", invalidEnumContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
			Expect(lintResultHasRule(result, "NF003")).To(BeTrue(), "expected NF003 in errors: %v", result["errors"])
		})
	})

	Context("malformed YAML", func() {
		It("exits 1 and JSON valid:false", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "malformed.md", malformedYAMLContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
		})
	})

	Context("missing body section", func() {
		It("exits 1 and JSON errors contain NF005", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "missing-section.md", missingBodySectionContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
			Expect(lintResultHasRule(result, "NF005")).To(BeTrue(), "expected NF005 in errors: %v", result["errors"])
		})
	})

	Context("placeholder value", func() {
		It("exits 1 and JSON errors contain NF006", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "placeholder.md", placeholderContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", path)
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
			Expect(lintResultHasRule(result, "NF006")).To(BeTrue(), "expected NF006 in errors: %v", result["errors"])
		})
	})

	Context("human-readable output", func() {
		It("exits 0 and stdout contains ✓ for valid note", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "valid-hr.md", validNoteContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "lint-note", path)
			Expect(exitCode).To(Equal(0))
			Expect(stdout).To(ContainSubstring("✓"))
		})

		It("exits 1 and stdout contains ✗ for invalid note", func() {
			path := testutil.WriteTestNote(GinkgoTB(), tmpDir, "invalid-hr.md", missingTitleContent)
			stdout, _, exitCode := testutil.RunBinary(binPath, "lint-note", path)
			Expect(exitCode).To(Equal(1))
			Expect(stdout).To(ContainSubstring("✗"))
		})
	})

	Context("nonexistent file", func() {
		It("exits 1 and JSON valid:false", func() {
			stdout, _, exitCode := testutil.RunBinary(binPath, "--json", "lint-note", "/nonexistent/path/note.md")
			Expect(exitCode).To(Equal(1))

			result := parseLintResult(stdout)
			Expect(result["valid"]).To(BeFalse())
		})
	})
})
