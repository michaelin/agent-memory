<plan_artifact feature="librarian-promotion">

<objective>

## Objective

Implement the Librarian promotion pipeline: `promote` and `deprecate` subcommands, exported helper functions, frontmatter consolidation, vault scaffold updates, Librarian deployment templates, and updated instructions. This gives agents the ability to promote inbox notes to verified status and deprecate superseded notes.

**Constrained by:**
- Design decisions in `.qrspi/librarian-promotion/design-draft.md` (16 decisions)
- Structure outline in `.qrspi/librarian-promotion/structure.md` (9 slices)

</objective>

<slices>

<slice name="slice-1-export-helpers">

## Slice 1: Export Helpers

**Goal**: Export private helpers as public functions, rewrite wikilink resolution to Obsidian-style, replace regex with goldmark-wikilink extension.

<tasks>

<task>
**Name**: Task 1.1 — Add goldmark-wikilink dependency
**Files**: `go.mod`, `go.sum`
**Action**: `go get go.abhg.dev/goldmark/wikilink@latest`
**Verify**: `go mod tidy && go build ./...`
**Done**: Dependency added, project compiles
</task>

<task>
**Name**: Task 1.2 — Rewrite ExtractWikilinks with goldmark-wikilink
**Files**: `internal/note/wikilinks.go`
**Action**: Per design decision #14, replace the hybrid regex+goldmark approach. Remove `wikilinkRe` regex, `codeRange` type, and manual code-range tracking. Configure goldmark with `&wikilink.Extender{}` (parser only, no renderer). Walk AST for `wikilink.Kind` nodes, collect `string(node.Target)`, handle `|alias` by taking target only. Deduplicate, preserve first-appearance order.
**Verify**: `go test ./internal/note/... -count=1 -run Wikilink`
**Done**: All existing wikilink extraction tests pass. No `regexp` import in wikilinks.go.
</task>

<task>
**Name**: Task 1.3 — Export AppendLog with action parameter
**Files**: `internal/note/write.go`
**Action**: Per design decision #2, rename `appendLog` → `AppendLog`. Add `action string` parameter (second position). Update format string from hardcoded `"write"` to `action`. Update the call in `Write()` to pass `"write"` as action.
**Verify**: `go test ./internal/note/... -count=1 -run Log`
**Done**: `AppendLog` is exported. `Write()` still produces log entries with `write` action. New test verifies custom action strings appear in log.
</task>

<task>
**Name**: Task 1.4 — Rewrite and export ResolveWikilinks
**Files**: `internal/note/write.go`
**Action**: Per design decisions #2 and #6, rename `resolveWikilinks` → `ResolveWikilinks`. Rewrite implementation: scan all `.md` files in `_inbox/`, `notes/`, `_deprecated/` (exclude `_meta/`). For each wikilink slug, match against filenames — inbox files match if filename ends with `-{slug}.md`, notes/deprecated files match if filename is `{slug}.md`. Return warning strings for unresolved links. Update `Write()` to call `ResolveWikilinks`.
**Verify**: `go test ./internal/note/... -count=1 -run Resolve`
**Done**: Resolution finds notes across all three directories. `_meta/` excluded. Existing write tests pass.
</task>

<task>
**Name**: Task 1.5 — Export FindSimilarNotes
**Files**: `internal/note/write.go`
**Action**: Per design decision #2, rename `findSimilarNotes` → `FindSimilarNotes`. No behavior change. Update `Write()` to call `FindSimilarNotes`. `SimilarNote` type is already exported.
**Verify**: `go test ./internal/note/... -count=1 -run Similar`
**Done**: `FindSimilarNotes` is exported. All existing similarity tests pass.
</task>

</tasks>

<checkpoint>
**Slice 1 Checkpoint**:
- [ ] `go test ./internal/note/... -count=1` — all pass
- [ ] `go test ./test/integration/... -tags=integration -count=1` — no regressions
- [ ] `go vet ./...` — clean
- [ ] No `regexp` import in `wikilinks.go`
- [ ] `AppendLog`, `ResolveWikilinks`, `FindSimilarNotes` are exported
</checkpoint>

</slice>

<slice name="slice-2-frontmatter-changes">

## Slice 2: Frontmatter Changes

**Goal**: Consolidate type field, add SupersededBy, remove UpdateType.

<tasks>

<task>
**Name**: Task 2.1 — Consolidate EpistemicType yaml tag and remove UpdateType
**Files**: `internal/note/note.go`
**Action**: Per design decision #15. Change `EpistemicType` tag from `yaml:"epistemic-type"` to `yaml:"type"`. Remove the `UpdateType string \`yaml:"update-type"\`` field entirely. Add `SupersededBy string \`yaml:"superseded-by"\`` field (per decision #4).
**Verify**: `go build ./...`
**Done**: Frontmatter has `EpistemicType` with `yaml:"type"`, `SupersededBy` with `yaml:"superseded-by"`, no `UpdateType`.
</task>

<task>
**Name**: Task 2.2 — Update test fixtures
**Files**: All test files with YAML fixtures containing `update-type:` or `epistemic-type:`
**Action**: Find all test fixtures using `update-type:` or `epistemic-type:` and change to `type:`. Update any assertions referencing `UpdateType`.
**Verify**: `go test ./internal/note/... -count=1`
**Done**: All tests pass with `type:` yaml key.
</task>

<task>
**Name**: Task 2.3 — Verify lint and serialize
**Files**: `internal/note/lint.go`, `internal/note/serialize.go`
**Action**: Check that NF003 and any other lint rules reference `EpistemicType` (not `UpdateType`). Verify `Serialize()` outputs `type:` in YAML. Update if needed.
**Verify**: `go test ./internal/note/... -count=1 -run "Lint|Serialize"`
**Done**: Lint validates `type:` field. Serialize round-trips correctly. `superseded-by:` appears in serialized output when set.
</task>

</tasks>

<checkpoint>
**Slice 2 Checkpoint**:
- [ ] `go test ./internal/note/... -count=1` — all pass
- [ ] `go test ./test/integration/... -tags=integration -count=1` — no regressions
- [ ] `go vet ./...` — clean
- [ ] No references to `UpdateType` or `update-type` in Go source (except backward compat if added)
</checkpoint>

</slice>

<slice name="slice-5-vault-scaffold">

## Slice 5: Vault Scaffold

**Goal**: Add `_deprecated/` and `_meta/templates/` with Librarian templates to vault structure.

<tasks>

<task>
**Name**: Task 5.1 — Add vault structure entries
**Files**: `internal/vault/structure.go`
**Action**: Per design decisions #12 and #13. Add entries to `VaultStructure()`: `{Path: "_deprecated", IsDir: true}`, `{Path: "_meta/templates", IsDir: true}`, `{Path: "_meta/templates/librarian-agent.md", IsDir: false, Template: "templates/librarian-agent.md"}`, `{Path: "_meta/templates/librarian-skill.md", IsDir: false, Template: "templates/librarian-skill.md"}`. Ensure directory entries come before their children.
**Verify**: `go build ./...`
**Done**: `VaultStructure()` returns entries for `_deprecated/`, `_meta/templates/`, and both template files.
</task>

<task>
**Name**: Task 5.2 — Create embedded Librarian templates
**Files**: `internal/vault/templates/librarian-agent.md`, `internal/vault/templates/librarian-skill.md`
**Action**: Per design decision #13. Write OpenCode-format agent definition template (model, permissions, description) and skill definition template (YAML frontmatter + workflow instructions for the Librarian). These are reference templates users copy to their harness config.
**Verify**: `go build ./...` (embed.FS picks them up)
**Done**: Templates compile into binary. Content describes Librarian workflow, references `promote`/`deprecate`/`lint-note` subcommands.
</task>

<task>
**Name**: Task 5.3 — Update vault tests
**Files**: `internal/vault/structure_test.go`, `internal/vault/init_test.go`
**Action**: Add assertions that `VaultStructure()` includes the new entries. Verify `Init()` creates `_deprecated/` and `_meta/templates/` with template files. Verify repair mode fixes missing entries.
**Verify**: `go test ./internal/vault/... -count=1`
**Done**: All vault tests pass including new entries.
</task>

</tasks>

<checkpoint>
**Slice 5 Checkpoint**:
- [ ] `go test ./internal/vault/... -count=1` — all pass
- [ ] `go test ./test/integration/... -tags=integration -count=1` — no regressions (init tests)
- [ ] New vault created with `agent-memory init` contains `_deprecated/` and `_meta/templates/`
</checkpoint>

</slice>

<slice name="slice-3-promote-core">

## Slice 3: Promote Core

**Goal**: Implement `Promote()` and `FindBySlug()` with all validation gates.

<tasks>

<task>
**Name**: Task 3.1 — Implement FindBySlug
**Files**: `internal/note/promote.go`
**Action**: New file. `FindBySlug(dir, slug string) (string, error)` scans `dir` for files matching the slug. For inbox-style dirs: match `*-{slug}.md`. For notes-style dirs: match `{slug}.md`. Error on zero matches (not found) or multiple matches (ambiguous). Per design decision #9.
**Verify**: `go test ./internal/note/... -count=1 -run FindBySlug`
**Done**: Finds single match, errors on ambiguity, errors on not found.
</task>

<task>
**Name**: Task 3.2 — Implement Promote
**Files**: `internal/note/promote.go`
**Action**: `Promote(vaultPath, slug string, confirmed bool) (PromoteResult, error)`. Per design decisions #1, #3, #5, #7, #16: (1) `FindBySlug` in `_inbox/`, (2) parse note, (3) `Lint()` — refuse if invalid, (4) `ResolveWikilinks()` — refuse if any unresolved (hard gate), (5) if `RequiresHumanReview && !confirmed` → error, (6) if `!RequiresHumanReview` → ignore `confirmed` silently, (7) check `notes/{slug}.md` doesn't exist (collision error), (8) update frontmatter: `Status = "verified"`, `Updated = today`, (9) `Serialize()` and write to `notes/{slug}.md`, (10) remove inbox file, (11) `AppendLog("promote", ...)`. Define `PromoteResult` struct in same file.
**Verify**: `go test ./internal/note/... -count=1 -run Promote`
**Done**: All cases covered: happy path, lint fail, unresolved wikilinks, missing confirmed, confirmed on non-review note, slug collision, ambiguous slug.
</task>

<task>
**Name**: Task 3.3 — Write promote unit tests
**Files**: `internal/note/promote_test.go`
**Action**: New file. Ginkgo/gomega BDD tests. Describe("Promote") with contexts for: happy path observation, happy path constraint with --confirmed, lint failure, unresolved wikilink blocks promotion, missing --confirmed on constraint, --confirmed on observation (ignored), slug collision in notes/, slug not found, ambiguous slug match. Each test sets up a temp vault with `_inbox/`, `notes/`, writes test notes, calls `Promote()`, asserts result and filesystem state.
**Verify**: `go test ./internal/note/... -count=1 -run Promote`
**Done**: All test cases pass. Coverage of all error paths.
</task>

</tasks>

<checkpoint>
**Slice 3 Checkpoint**:
- [ ] `go test ./internal/note/... -count=1` — all pass including promote tests
- [ ] `go vet ./...` — clean
- [ ] Promoted note appears in `notes/` with `status: verified`
- [ ] Log entry written with `promote` action
</checkpoint>

</slice>

<slice name="slice-4-deprecate-core">

## Slice 4: Deprecate Core

**Goal**: Implement `Deprecate()` function.

<tasks>

<task>
**Name**: Task 4.1 — Implement Deprecate
**Files**: `internal/note/deprecate.go`
**Action**: New file. `Deprecate(vaultPath, slug, supersededBy string) (DeprecateResult, error)`. Per design decision #12: (1) find `notes/{slug}.md`, (2) parse, (3) set `Status = "deprecated"`, `SupersededBy = supersededBy`, `Updated = today`, (4) `Serialize()`, (5) create `_deprecated/` if missing, (6) write to `_deprecated/{slug}.md`, (7) remove `notes/{slug}.md`, (8) `AppendLog("deprecate", ...)`. Define `DeprecateResult` struct in same file.
**Verify**: `go test ./internal/note/... -count=1 -run Deprecate`
**Done**: Happy path works. Note not found errors correctly.
</task>

<task>
**Name**: Task 4.2 — Write deprecate unit tests
**Files**: `internal/note/deprecate_test.go`
**Action**: New file. Ginkgo/gomega BDD tests. Describe("Deprecate") with contexts for: happy path, note not found in notes/, `_deprecated/` auto-created. Each test sets up temp vault, writes a note to `notes/`, calls `Deprecate()`, asserts result and filesystem state (file moved, frontmatter updated, log written).
**Verify**: `go test ./internal/note/... -count=1 -run Deprecate`
**Done**: All test cases pass.
</task>

</tasks>

<checkpoint>
**Slice 4 Checkpoint**:
- [ ] `go test ./internal/note/... -count=1` — all pass including deprecate tests
- [ ] `go vet ./...` — clean
- [ ] Deprecated note appears in `_deprecated/` with `status: deprecated` and `superseded-by` set
- [ ] Log entry written with `deprecate` action
</checkpoint>

</slice>

<slice name="slice-6-cli-promote">

## Slice 6: CLI Promote

**Goal**: Wire `promote` subcommand into CLI.

<tasks>

<task>
**Name**: Task 6.1 — Implement promote subcommand
**Files**: `internal/cli/promote.go`, `internal/cli/root.go`
**Action**: New file following `write_note.go` pattern. `newPromoteCmd() *cobra.Command` with local flags: `--slug` (required string), `--confirmed` (bool), `--vault` (string). RunE: discover vault, call `note.Promote()`, output result as JSON or human-readable based on `jsonOutput`. Per design decision #11 for JSON shape. Register in `root.go` with `cmd.AddCommand(newPromoteCmd())`.
**Verify**: `go build ./cmd/agent-memory && ./agent-memory promote --help`
**Done**: Help shows all flags. Binary compiles.
</task>

<task>
**Name**: Task 6.2 — Promote integration test
**Files**: `test/integration/promote_test.go`
**Action**: New file. Integration test using `testutil.BuildBinary`/`RunBinary`. Write a note via `write-note`, then promote via `promote --slug=... --json`. Assert JSON output matches `PromoteResult`. Assert file moved. Assert log entry. Test error case (missing --confirmed on constraint).
**Verify**: `go test ./test/integration/... -tags=integration -count=1 -run Promote`
**Done**: Integration tests pass for happy path and error cases.
</task>

</tasks>

<checkpoint>
**Slice 6 Checkpoint**:
- [ ] `agent-memory promote --slug=test --json` works end-to-end
- [ ] `go test ./test/integration/... -tags=integration -count=1 -run Promote` — pass
- [ ] No regressions: `go test ./... -count=1`
</checkpoint>

</slice>

<slice name="slice-7-cli-deprecate">

## Slice 7: CLI Deprecate

**Goal**: Wire `deprecate` subcommand into CLI.

<tasks>

<task>
**Name**: Task 7.1 — Implement deprecate subcommand
**Files**: `internal/cli/deprecate.go`, `internal/cli/root.go`
**Action**: New file following same pattern. `newDeprecateCmd() *cobra.Command` with local flags: `--slug` (required string), `--superseded-by` (required string), `--vault` (string). RunE: discover vault, call `note.Deprecate()`, output result. Register in `root.go`.
**Verify**: `go build ./cmd/agent-memory && ./agent-memory deprecate --help`
**Done**: Help shows all flags. Binary compiles.
</task>

<task>
**Name**: Task 7.2 — Deprecate integration test
**Files**: `test/integration/deprecate_test.go`
**Action**: New file. Write a note, promote it, then deprecate it via `deprecate --slug=... --superseded-by=... --json`. Assert JSON output. Assert file in `_deprecated/`. Assert log entry.
**Verify**: `go test ./test/integration/... -tags=integration -count=1 -run Deprecate`
**Done**: Integration tests pass.
</task>

</tasks>

<checkpoint>
**Slice 7 Checkpoint**:
- [ ] `agent-memory deprecate --slug=test --superseded-by=new --json` works end-to-end
- [ ] `go test ./test/integration/... -tags=integration -count=1 -run Deprecate` — pass
- [ ] No regressions
</checkpoint>

</slice>

<slice name="slice-8-instructions-update">

## Slice 8: Instructions Update

**Goal**: Update instructions output with Librarian workflow and deployment guidance.

<tasks>

<task>
**Name**: Task 8.1 — Update instructions text
**Files**: `internal/cli/instructions.go`
**Action**: Per design decision #10. Add sections to the hardcoded instructions string covering: Librarian subagent workflow (scan inbox → lint → promote/deprecate), `promote` and `deprecate` subcommand usage, human confirmation flow for constraint/decision notes, deployment template locations (`_meta/templates/librarian-agent.md`, `_meta/templates/librarian-skill.md`), setup instructions for OpenCode.
**Verify**: `go build ./cmd/agent-memory && ./agent-memory instructions`
**Done**: Output mentions promote, deprecate, Librarian, templates, confirmation flow.
</task>

<task>
**Name**: Task 8.2 — Update instructions integration test
**Files**: `test/integration/instructions_test.go` (existing file, if present)
**Action**: Add assertions that instructions output contains key terms: "promote", "deprecate", "Librarian", "templates", "confirmed".
**Verify**: `go test ./test/integration/... -tags=integration -count=1 -run Instructions`
**Done**: Test passes with new content assertions.
</task>

</tasks>

<checkpoint>
**Slice 8 Checkpoint**:
- [ ] `agent-memory instructions` output includes Librarian workflow
- [ ] Integration test passes
</checkpoint>

</slice>

<slice name="slice-9-integration-tests">

## Slice 9: Full Workflow Integration Tests

**Goal**: End-to-end tests covering the complete Librarian workflow.

<tasks>

<task>
**Name**: Task 9.1 — Full lifecycle integration test
**Files**: `test/integration/promote_test.go` or `test/integration/deprecate_test.go` (extend existing)
**Action**: Test the full flow: (1) init vault, (2) write-note an observation, (3) promote it, (4) verify in `notes/`, (5) write-note a replacement, (6) deprecate old note with superseded-by, (7) promote replacement, (8) verify old in `_deprecated/`, new in `notes/`, (9) verify wikilinks resolve across all directories. Test with `--json` output throughout.
**Verify**: `go test ./test/integration/... -tags=integration -count=1`
**Done**: Full lifecycle test passes.
</task>

</tasks>

<checkpoint>
**Slice 9 Checkpoint**:
- [ ] `go test ./test/integration/... -tags=integration -count=1` — all pass
- [ ] `go test ./internal/... -count=1` — all pass
- [ ] `go vet ./...` — clean
</checkpoint>

</slice>

</slices>

<verification>

## Final Verification

Before declaring implementation complete:
- [ ] All unit tests pass: `go test ./internal/... -count=1`
- [ ] All integration tests pass: `go test ./test/integration/... -tags=integration -count=1`
- [ ] `go vet ./...` — clean
- [ ] `agent-memory promote --help` and `agent-memory deprecate --help` show correct flags
- [ ] `agent-memory instructions` includes Librarian workflow
- [ ] `agent-memory init` creates `_deprecated/` and `_meta/templates/`
- [ ] No `regexp` import in `wikilinks.go`
- [ ] No references to `UpdateType` or `update-type` in Go source

</verification>

<success_criteria>

## Success Criteria

- [ ] `agent-memory promote` moves notes from `_inbox/` to `notes/` with validation
- [ ] `agent-memory deprecate` moves notes from `notes/` to `_deprecated/` with forward link
- [ ] Unresolved wikilinks block promotion
- [ ] `--confirmed` required for constraint/decision notes, silently ignored for others
- [ ] Wikilink resolution uses Obsidian-style filename-based lookup across all vault directories
- [ ] Vault scaffold includes `_deprecated/` and Librarian deployment templates
- [ ] Instructions output describes Librarian workflow
- [ ] All slice checkpoints passed

</success_criteria>

</plan_artifact>
