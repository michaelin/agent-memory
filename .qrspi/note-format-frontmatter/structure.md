<structure_artifact feature="note-format-frontmatter">

<type_definitions>

## Type Definitions

```go
// package: internal/note

// Note represents a parsed markdown note with YAML frontmatter.
type Note struct {
    Frontmatter Frontmatter
    Body        string
}

// Frontmatter contains all YAML frontmatter fields for a note.
type Frontmatter struct {
    Title               string   `yaml:"title"`
    Created             string   `yaml:"created"`
    Updated             string   `yaml:"updated"`
    ReviewBy            string   `yaml:"review-by"`
    Status              string   `yaml:"status"`
    Confidence          string   `yaml:"confidence"`
    EpistemicType       string   `yaml:"epistemic-type"`
    Scope               string   `yaml:"scope"`
    Project             string   `yaml:"project"`
    Domain              []string `yaml:"domain"`
    SourceAgent         string   `yaml:"source-agent"`
    SourceArtifact      string   `yaml:"source-artifact"`
    VerifiedBy          string   `yaml:"verified-by"`
    VerifiedDate        string   `yaml:"verified-date"`
    RequiresHumanReview bool     `yaml:"requires-human-review"`
    UpdateType          string   `yaml:"update-type"`
    Targets             []string `yaml:"targets"`
    Tags                []string `yaml:"tags"`
}

// Rule represents a single lint check with a unique ID.
type Rule struct {
    ID          string
    Description string
    Check       func(note *Note) []string
}

// LintError is a single validation failure tied to a rule.
type LintError struct {
    Rule    string `json:"rule"`
    Message string `json:"message"`
}

// LintResult is the output of running all lint rules on a note.
type LintResult struct {
    Valid  bool        `json:"valid"`
    Errors []LintError `json:"errors,omitempty"`
}
```

</type_definitions>

<signatures>

## Function Signatures

```go
// package: internal/note

// Parse splits a markdown file into frontmatter and body.
// Returns error if frontmatter delimiters are missing or YAML is malformed.
func Parse(content []byte) (*Note, error)

// Lint runs all registered rules against a parsed note.
func Lint(note *Note) *LintResult

// Rules returns the ordered list of all registered lint rules.
func Rules() []Rule

// IsPlaceholder checks if a string value matches known placeholder patterns.
func IsPlaceholder(value string) bool

// IsValidDate checks if a string is a valid YYYY-MM-DD date.
func IsValidDate(value string) bool
```

```go
// package: internal/cli

// newLintNoteCmd creates the lint-note subcommand.
func newLintNoteCmd() *cobra.Command
```

</signatures>

<vertical_slices>

## Vertical Slices

<slice name="slice-1-parse" order="1">

**Scope**: Note parsing — split frontmatter from body, unmarshal YAML into `Frontmatter` struct. No validation yet. This is the foundation everything else builds on.

**Files to create/modify**:
- `internal/note/note.go` — `Note`, `Frontmatter` types and `Parse()` function
- `internal/note/note_suite_test.go` — ginkgo suite bootstrap
- `internal/note/note_test.go` — unit tests for `Parse()`: valid note, malformed YAML, missing delimiters, empty body, extra unknown fields (lenient)

**Verification point**:
```bash
go test ./internal/note/ -v
```
Tests pass for: valid parse, malformed YAML error, missing `---` delimiters error, unknown fields silently ignored.

**Depends on**: none

</slice>

<slice name="slice-2-rules-engine" order="2">

**Scope**: Rule engine and first rule (NF001 — required fields). Establishes the `Rule`, `LintError`, `LintResult` types and the `Lint()` function that iterates rules. Proves the architecture works end-to-end with one real rule.

**Files to create/modify**:
- `internal/note/lint.go` — `Rule`, `LintError`, `LintResult` types, `Lint()` function, `Rules()` function, NF001 rule implementation
- `internal/note/lint_test.go` — unit tests for NF001: all required fields present (pass), each required field missing individually (fail with correct rule ID and message)

**Verification point**:
```bash
go test ./internal/note/ -v -run "Lint"
```
Tests pass for: valid note passes NF001, missing `title` fails with `NF001`, missing `source-agent` fails with `NF001`, etc.

**Depends on**: slice-1-parse

</slice>

<slice name="slice-3-validation-rules" order="3">

**Scope**: Remaining lint rules NF002–NF007. Each rule is independently testable.

- **NF002**: Valid date format for `created`, `updated`, `review-by`
- **NF003**: Valid enum values for `status`, `epistemic-type`, `confidence`, `scope`
- **NF004**: `project` required when `scope: project`
- **NF005**: Required body sections (`# Title`, `## Evidence`, `## Implications`, `## Related`; synthesis exemption)
- **NF006**: No placeholder values in required fields
- **NF007**: `domain` is a non-empty list

**Files to create/modify**:
- `internal/note/lint.go` — add NF002–NF007 rule implementations to `Rules()`
- `internal/note/lint_test.go` — unit tests for each rule: valid values pass, invalid values fail with correct rule ID
- `internal/note/placeholder.go` — `IsPlaceholder()` helper (used by NF006)
- `internal/note/placeholder_test.go` — unit tests for placeholder detection patterns

**Verification point**:
```bash
go test ./internal/note/ -v
```
All rules pass on a valid note. Each rule fails correctly on targeted invalid input. Synthesis exemption works for NF005.

**Depends on**: slice-2-rules-engine

</slice>

<slice name="slice-4-cli-subcommand" order="4">

**Scope**: Wire `lint-note` into the CLI as `agent-memory lint-note <file>`. Reads file, calls `Parse()` and `Lint()`, outputs result in human-readable or JSON format. Exit 0/1.

**Files to create/modify**:
- `internal/cli/lint_note.go` — `newLintNoteCmd()` factory, file reading, dual output formatting
- `internal/cli/root.go` — register `newLintNoteCmd()` via `AddCommand()`
- `internal/cli/lint_note_test.go` — unit tests for output formatting (optional, may defer to integration tests)

**Verification point**:
```bash
go build ./cmd/agent-memory && ./bin/agent-memory lint-note testdata/valid-note.md
go build ./cmd/agent-memory && ./bin/agent-memory --json lint-note testdata/invalid-note.md
```
Human-readable output shows `✓` or `✗` with rule IDs. JSON output matches `LintResult` shape. Exit codes correct.

**Depends on**: slice-3-validation-rules

</slice>

<slice name="slice-5-integration-tests" order="5">

**Scope**: Integration tests exercising the `lint-note` subcommand end-to-end via the compiled binary. Also review and update existing integration tests if needed (e.g., `instructions` output).

**Files to create/modify**:
- `test/integration/lint_note_test.go` — integration tests: valid note passes, missing field fails, invalid enum fails, malformed YAML fails, missing body section fails, placeholder fails; both `--json` and human-readable output
- `test/integration/init_test.go` — review/update if writing-protocol v2 changes affect existing assertions
- `internal/testutil/helpers.go` — add `WriteTestNote()` helper if needed for creating test note files in temp vaults

**Verification point**:
```bash
go test -tags integration ./test/integration/ -v
```
All integration tests pass. Existing init tests still pass.

**Depends on**: slice-4-cli-subcommand

</slice>

<slice name="slice-6-writing-protocol-v2" order="6">

**Scope**: Replace writing-protocol.md v1 template content with v2 containing full agent instructions for note format, frontmatter fields, body sections, and validation. Update `instructions` subcommand output if needed.

**Files to create/modify**:
- `internal/vault/templates/writing-protocol.md` — replace v1 content with v2 (full note format spec, frontmatter fields, body sections, validation instructions)
- `internal/cli/instructions.go` — update hardcoded output if it references "Writing to the vault is not yet enabled" or needs to reflect v2 capabilities
- `internal/vault/structure_test.go` — may need update if test asserts specific entry count or template content
- `test/integration/init_test.go` — update assertions if writing-protocol content is checked

**Verification point**:
```bash
go test ./internal/vault/ -v
go test -tags integration ./test/integration/ -v
```
Vault init seeds v2 writing protocol. Existing tests pass with updated assertions. Instructions output is accurate.

**Depends on**: slice-5-integration-tests

</slice>

</vertical_slices>

<implementation_order>

## Implementation Order

1. **slice-1-parse**: Note types and frontmatter parsing
2. **slice-2-rules-engine**: Rule architecture and NF001 (required fields)
3. **slice-3-validation-rules**: NF002–NF007 (all remaining rules)
4. **slice-4-cli-subcommand**: `agent-memory lint-note` wiring
5. **slice-5-integration-tests**: End-to-end binary tests
6. **slice-6-writing-protocol-v2**: Template content and instructions update

**Parallel opportunities**: Slices 1–3 are pure `internal/note/` work with no CLI dependency. Slice 6 (writing protocol) is independent of slices 1–5 in terms of code, but is ordered last because integration tests in slice 5 should verify the lint-note behavior before we update the protocol that references it.

</implementation_order>

</structure_artifact>
