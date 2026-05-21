<plan_artifact feature="note-format-frontmatter">

<objective>

## Objective

Implement note format parsing, rule-based frontmatter validation, the `agent-memory lint-note` subcommand, and writing protocol v2. After this increment, agents can write structurally valid notes and tools can parse and validate them.

**Constrained by:**
- Design decisions in `.qrspi/note-format-frontmatter/design.md`
- Structure outline in `.qrspi/note-format-frontmatter/structure.md`
- ADR-0001: Single binary with subcommands

</objective>

<slices>

<slice name="slice-1-parse">

## Slice 1: Note Parsing

**Goal**: Parse a markdown file with YAML frontmatter into a structured `Note` type. Foundation for all subsequent slices.

<tasks>

<task>
**Name**: Task 1.1 — Create note package with types
**Files**: `internal/note/note.go`
**Action**: Create `internal/note/` package. Define `Note` struct (containing `Frontmatter` and `Body string`) and `Frontmatter` struct with all fields per design decision #3. Use `go.yaml.in/yaml/v3` per design decision #2.
**Verify**: `go build ./internal/note/`
**Done**: Package compiles. `Note` and `Frontmatter` types are exported.
</task>

<task>
**Name**: Task 1.2 — Implement Parse function
**Files**: `internal/note/note.go`
**Action**: Implement `Parse(content []byte) (*Note, error)` per design decision #4. Split on `---` delimiters to extract YAML frontmatter and markdown body. Unmarshal YAML into `Frontmatter` struct with lenient unmarshaling per design decision #6. Return clear errors for: missing opening `---`, missing closing `---`, malformed YAML.
**Verify**: `go build ./internal/note/`
**Done**: `Parse` compiles and handles the three error cases.
</task>

<task>
**Name**: Task 1.3 — Create ginkgo test suite and Parse tests
**Files**: `internal/note/note_suite_test.go`, `internal/note/note_test.go`
**Action**: Bootstrap ginkgo suite. Write unit tests for `Parse()`: (1) valid note with all fields parses correctly, (2) malformed YAML returns error, (3) missing `---` delimiters returns error, (4) empty body after frontmatter is accepted, (5) extra unknown fields are silently ignored per design decision #6, (6) date-like strings remain as strings (not auto-parsed to time.Time) per design decision #3.
**Verify**: `go test ./internal/note/ -v`
**Done**: All 6 test cases pass.
</task>

</tasks>

<checkpoint>
**Slice 1 Checkpoint**:
- [ ] `Parse()` correctly splits frontmatter from body
- [ ] Malformed YAML produces clear error
- [ ] Missing delimiters produce clear error
- [ ] Unknown fields are silently ignored
- [ ] All tests pass: `go test ./internal/note/ -v`
</checkpoint>

</slice>

<slice name="slice-2-rules-engine">

## Slice 2: Rule Engine + NF001

**Goal**: Establish the rule-based lint architecture and prove it works with the first rule (NF001 — required fields present).

<tasks>

<task>
**Name**: Task 2.1 — Define rule engine types
**Files**: `internal/note/lint.go`
**Action**: Define `Rule`, `LintError`, `LintResult` types per design decision #12. Implement `Rules() []Rule` returning the registered rule slice. Implement `Lint(note *Note) *LintResult` that iterates all rules, collects errors, and sets `Valid` accordingly.
**Verify**: `go build ./internal/note/`
**Done**: Rule engine compiles. `Lint()` returns `LintResult` with `Valid: true` when no rules fail.
</task>

<task>
**Name**: Task 2.2 — Implement NF001 (required fields)
**Files**: `internal/note/lint.go`
**Action**: Implement NF001 rule checking that all required frontmatter fields are present and non-empty: `title`, `created`, `updated`, `status`, `confidence`, `epistemic-type`, `scope`, `source-agent`, `source-artifact`. Note: `project` is conditionally required (handled by NF004), `domain` is a list (handled by NF007), `review-by` is optional. Each missing field produces a separate `LintError` with rule ID `NF001`.
**Verify**: `go test ./internal/note/ -v -run "NF001"`
**Done**: NF001 correctly identifies each missing required field individually.
</task>

<task>
**Name**: Task 2.3 — Write NF001 tests
**Files**: `internal/note/lint_test.go`
**Action**: Write unit tests: (1) valid note with all fields passes NF001, (2) missing `title` fails with `NF001`, (3) missing `source-agent` fails with `NF001`, (4) empty string `title` fails with `NF001`, (5) multiple missing fields produce multiple `NF001` errors. Verify rule ID and message content in each `LintError`.
**Verify**: `go test ./internal/note/ -v -run "NF001"`
**Done**: All NF001 test cases pass. Error messages include field names.
</task>

</tasks>

<checkpoint>
**Slice 2 Checkpoint**:
- [ ] `Lint()` runs all registered rules and returns `LintResult`
- [ ] NF001 detects each missing required field
- [ ] `LintError` contains correct rule ID and descriptive message
- [ ] All tests pass: `go test ./internal/note/ -v`
</checkpoint>

</slice>

<slice name="slice-3-validation-rules">

## Slice 3: Validation Rules NF002–NF007

**Goal**: Implement all remaining lint rules. Each rule is independently testable.

<tasks>

<task>
**Name**: Task 3.1 — Implement NF002 (date format)
**Files**: `internal/note/lint.go`
**Action**: Implement NF002 checking `created`, `updated` are valid `YYYY-MM-DD` dates. `review-by` is checked only if non-empty (it's optional). Add `IsValidDate(value string) bool` helper. Use `time.Parse("2006-01-02", value)` for validation.
**Verify**: `go test ./internal/note/ -v -run "NF002"`
**Done**: Valid dates pass. Invalid formats (`2026/04/24`, `April 24`, `2026-13-01`) fail with `NF002`.
</task>

<task>
**Name**: Task 3.2 — Implement NF003 (enum values)
**Files**: `internal/note/lint.go`
**Action**: Implement NF003 checking: `status` is one of `inbox|verified|deprecated|contested|superseded`; `epistemic-type` is one of `observation|pattern|constraint|decision|assumption|synthesis`; `confidence` is one of `low|medium|high`; `scope` is one of `project|cross-project`. Each invalid value produces a separate `LintError`.
**Verify**: `go test ./internal/note/ -v -run "NF003"`
**Done**: Valid enum values pass. Invalid values fail with `NF003` and include the field name and invalid value in the message.
</task>

<task>
**Name**: Task 3.3 — Implement NF004 (conditional project)
**Files**: `internal/note/lint.go`
**Action**: Implement NF004 per design decision #7: if `scope: project`, then `project` must be non-empty. If `scope: cross-project`, `project` is not checked (no warning either way).
**Verify**: `go test ./internal/note/ -v -run "NF004"`
**Done**: `scope: project` with empty `project` fails. `scope: project` with `project: foo` passes. `scope: cross-project` with or without `project` passes.
</task>

<task>
**Name**: Task 3.4 — Implement NF005 (body sections)
**Files**: `internal/note/lint.go`
**Action**: Implement NF005 per design decision #9. Check body contains lines starting with `# ` (title heading), `## Evidence`, `## Implications`, `## Related`. Synthesis exemption: if `epistemic-type: synthesis`, check for `## Synthesis` and `## Contributing notes` instead of `## Evidence` and `## Implications`.
**Verify**: `go test ./internal/note/ -v -run "NF005"`
**Done**: Valid body passes. Missing `## Evidence` fails. Synthesis note with `## Synthesis` + `## Contributing notes` passes. Synthesis note missing `## Related` fails.
</task>

<task>
**Name**: Task 3.5 — Implement NF006 (placeholder detection)
**Files**: `internal/note/placeholder.go`, `internal/note/placeholder_test.go`, `internal/note/lint.go`
**Action**: Implement `IsPlaceholder(value string) bool` per design decision #8. Checks: empty string, angle-bracket pattern `<.*>`, case-insensitive match for `TODO`, `TBD`, `FIXME`, `xxx`, `placeholder`. Add NF006 rule that checks required string fields only (not optional fields like `verified-by`).
**Verify**: `go test ./internal/note/ -v -run "NF006|Placeholder"`
**Done**: `IsPlaceholder` correctly identifies all patterns. NF006 catches placeholders in required fields. Optional fields with empty strings do not trigger NF006.
</task>

<task>
**Name**: Task 3.6 — Implement NF007 (domain list)
**Files**: `internal/note/lint.go`
**Action**: Implement NF007 per design decision #5. Check that `domain` is a non-empty list. Per design decision #5, if the YAML library silently unmarshals a bare string into `[]string`, we accept it.
**Verify**: `go test ./internal/note/ -v -run "NF007"`
**Done**: `domain: [golang]` passes. Empty `domain: []` fails. Missing `domain` (nil slice) fails.
</task>

<task>
**Name**: Task 3.7 — Write comprehensive rule tests
**Files**: `internal/note/lint_test.go`
**Action**: Add tests for each rule (NF002–NF007) covering pass and fail cases. Add a test that runs all rules on a fully valid note and verifies `Valid: true` with no errors. Add a test with multiple violations across different rules and verify all are reported.
**Verify**: `go test ./internal/note/ -v`
**Done**: All rule tests pass. Valid note produces zero errors. Multi-violation note produces errors from multiple rules.
</task>

</tasks>

<checkpoint>
**Slice 3 Checkpoint**:
- [ ] All 7 rules (NF001–NF007) implemented and tested
- [ ] Valid note passes all rules
- [ ] Each rule fails correctly on targeted invalid input
- [ ] Synthesis exemption works for NF005
- [ ] Placeholder detection works for all patterns
- [ ] All tests pass: `go test ./internal/note/ -v`
</checkpoint>

</slice>

<slice name="slice-4-cli-subcommand">

## Slice 4: CLI Subcommand

**Goal**: Wire `lint-note` into the CLI with dual output (human-readable default, JSON with `--json`).

<tasks>

<task>
**Name**: Task 4.1 — Implement lint-note subcommand
**Files**: `internal/cli/lint_note.go`
**Action**: Create `newLintNoteCmd()` per design decision #10. Takes exactly one file path argument (`cobra.ExactArgs(1)`). Reads file, calls `note.Parse()`, then `note.Lint()`. On parse error, report as a single error. Format output per design decision #10: human-readable shows `✓`/`✗` with rule IDs, JSON outputs `LintResult` struct. Exit 0 on valid, exit 1 on invalid. Follow existing error handling pattern from `init.go` (silence cobra errors when `--json`).
**Verify**: `go build ./cmd/agent-memory/`
**Done**: Subcommand compiles.
</task>

<task>
**Name**: Task 4.2 — Register subcommand
**Files**: `internal/cli/root.go`
**Action**: Add `cmd.AddCommand(newLintNoteCmd())` to `NewRootCmd()`.
**Verify**: `go build ./cmd/agent-memory/ && ./agent-memory lint-note --help`
**Done**: `lint-note` appears in help output. `--help` shows usage with file argument.
</task>

<task>
**Name**: Task 4.3 — Manual smoke test
**Files**: none (use ad-hoc test files)
**Action**: Create a valid test note file and an invalid one. Run `agent-memory lint-note` on both with and without `--json`. Verify output format and exit codes.
**Verify**: `go build -o bin/agent-memory ./cmd/agent-memory/ && bin/agent-memory lint-note <valid-file> && bin/agent-memory --json lint-note <invalid-file>`
**Done**: Human-readable output shows `✓`/`✗` with rule IDs. JSON output is valid JSON matching `LintResult` shape. Exit codes are correct.
</task>

</tasks>

<checkpoint>
**Slice 4 Checkpoint**:
- [ ] `agent-memory lint-note <file>` works end-to-end
- [ ] Human-readable output shows `✓` or `✗` with rule IDs
- [ ] JSON output matches `LintResult` shape
- [ ] Exit 0 on valid note, exit 1 on invalid
- [ ] Parse errors are reported cleanly
- [ ] All unit tests still pass: `go test ./internal/... -v`
</checkpoint>

</slice>

<slice name="slice-5-integration-tests">

## Slice 5: Integration Tests

**Goal**: End-to-end binary tests for `lint-note` subcommand. Add test helpers for creating note files.

<tasks>

<task>
**Name**: Task 5.1 — Add test note helper
**Files**: `internal/testutil/helpers.go`
**Action**: Add `WriteTestNote(t testing.TB, dir, filename, content string) string` helper that writes a note file to a temp directory and returns the full path. This simplifies creating test notes in integration tests.
**Verify**: `go build ./internal/testutil/`
**Done**: Helper compiles and is usable from test code.
</task>

<task>
**Name**: Task 5.2 — Write lint-note integration tests
**Files**: `test/integration/lint_note_test.go`
**Action**: Write integration tests using `testutil.BuildBinary` and `testutil.RunBinary`. Test cases: (1) valid note — exit 0, JSON `{"valid": true}`, (2) missing required field — exit 1, JSON contains `NF001`, (3) invalid enum — exit 1, JSON contains `NF003`, (4) malformed YAML — exit 1, error reported, (5) missing body section — exit 1, JSON contains `NF005`, (6) placeholder value — exit 1, JSON contains `NF006`, (7) human-readable output — exit 0 contains `✓`, exit 1 contains `✗`, (8) nonexistent file — exit 1 with error.
**Verify**: `go test -tags integration ./test/integration/ -v -run "lint.note"`
**Done**: All 8 integration test cases pass.
</task>

<task>
**Name**: Task 5.3 — Review existing integration tests
**Files**: `test/integration/init_test.go`
**Action**: Check if any existing assertions are affected by changes in this increment. The `instructions` test asserts "Writing to the vault is not yet enabled" — this will be updated in slice 6 if the instructions output changes. For now, verify existing tests still pass.
**Verify**: `go test -tags integration ./test/integration/ -v`
**Done**: All existing integration tests pass alongside new lint-note tests.
</task>

</tasks>

<checkpoint>
**Slice 5 Checkpoint**:
- [ ] All lint-note integration tests pass (both JSON and human-readable)
- [ ] Existing init integration tests still pass
- [ ] Test helper `WriteTestNote` works correctly
- [ ] All tests pass: `go test -tags integration ./test/integration/ -v`
</checkpoint>

</slice>

<slice name="slice-6-writing-protocol-v2">

## Slice 6: Writing Protocol v2

**Goal**: Replace writing-protocol.md v1 with v2 containing full agent instructions. Update instructions subcommand if needed.

<tasks>

<task>
**Name**: Task 6.1 — Write v2 writing protocol template
**Files**: `internal/vault/templates/writing-protocol.md`
**Action**: Replace v1 content with v2 per design decision #11 and the ticket's writing protocol spec. Include: note format overview, all required frontmatter fields with descriptions and allowed values, body section requirements (with synthesis exemption), validation instructions referencing `agent-memory lint-note`, and the rule ID table (NF001–NF007).
**Verify**: `go test ./internal/vault/ -v`
**Done**: Template is non-empty. Vault unit tests pass (structure_test.go checks template readability).
</task>

<task>
**Name**: Task 6.2 — Update instructions subcommand
**Files**: `internal/cli/instructions.go`
**Action**: Update the hardcoded instructions output to reflect that note writing format is now defined. Remove or update "Writing to the vault is not yet enabled in this version." to reflect current capabilities (format defined, writing not yet enabled). Reference `_meta/writing-protocol.md` for the full spec and `agent-memory lint-note` for validation.
**Verify**: `go build -o bin/agent-memory ./cmd/agent-memory/ && bin/agent-memory instructions`
**Done**: Instructions output is accurate for current capabilities.
</task>

<task>
**Name**: Task 6.3 — Update affected tests
**Files**: `test/integration/init_test.go`, `internal/vault/structure_test.go`
**Action**: Update any test assertions that check writing-protocol content or instructions output. The integration test for `instructions` asserts "Writing to the vault is not yet enabled" — update to match new output. Verify vault structure tests still pass with v2 template.
**Verify**: `go test ./internal/vault/ -v && go test -tags integration ./test/integration/ -v`
**Done**: All tests pass with updated assertions.
</task>

</tasks>

<checkpoint>
**Slice 6 Checkpoint**:
- [ ] Writing protocol v2 contains full note format spec
- [ ] Instructions output is accurate
- [ ] All vault unit tests pass
- [ ] All integration tests pass (including updated assertions)
- [ ] Full test suite: `go test ./internal/... -v && go test -tags integration ./test/integration/ -v`
</checkpoint>

</slice>

</slices>

<verification>

## Final Verification

Before declaring implementation complete:
- [ ] All 6 slices verified end-to-end
- [ ] All unit tests pass: `go test ./internal/... -v`
- [ ] All integration tests pass: `go test -tags integration ./test/integration/ -v`
- [ ] `agent-memory lint-note` works on valid and invalid notes (both output modes)
- [ ] Writing protocol v2 is complete and accurate
- [ ] No regressions in existing init/instructions functionality

</verification>

<success_criteria>

## Success Criteria

- [ ] `note.Parse()` correctly parses frontmatter + body from markdown files
- [ ] 7 lint rules (NF001–NF007) implemented and independently tested
- [ ] `agent-memory lint-note <file>` produces correct output (human-readable and JSON)
- [ ] Writing protocol v2 provides complete agent instructions for note format
- [ ] All slice checkpoints passed
- [ ] No regressions in existing functionality

</success_criteria>

</plan_artifact>
