<structure_artifact feature="librarian-promotion">

<type_definitions>

## Type Definitions

### Modified types

```go
// Frontmatter — in internal/note/note.go
// 1. Rename field: UpdateType → remove (already have EpistemicType with yaml:"epistemic-type")
//    Change EpistemicType yaml tag from "epistemic-type" to "type"
// 2. Add field:
SupersededBy string `yaml:"superseded-by"`

// Note: UpdateType field and yaml:"update-type" tag are removed entirely.
// Existing notes with "update-type:" in frontmatter need backward compat
// handling — accept both "type" and "update-type" during Parse, but
// Serialize always writes "type".
```

### New types

```go
// PromoteResult — in internal/note/promote.go
// Output type for the promote subcommand.
type PromoteResult struct {
    Status        string `json:"status"`          // "promoted" or "error"
    Slug          string `json:"slug"`
    From          string `json:"from,omitempty"`
    To            string `json:"to,omitempty"`
    EpistemicType string `json:"epistemic_type,omitempty"`
    Confirmed     bool   `json:"confirmed"`
    Error         string `json:"error,omitempty"`
}

// DeprecateResult — in internal/note/deprecate.go
// Output type for the deprecate subcommand.
type DeprecateResult struct {
    Status       string `json:"status"`           // "deprecated" or "error"
    Slug         string `json:"slug"`
    SupersededBy string `json:"superseded_by,omitempty"`
    MovedTo      string `json:"moved_to,omitempty"`
    Error        string `json:"error,omitempty"`
}
```

### Exported functions (currently private)

```go
// in internal/note/write.go (or extracted to internal/note/log.go, internal/note/resolve.go)

// AppendLog appends a structured log entry to _meta/log.md.
// action is "write", "promote", or "deprecate".
func AppendLog(vaultPath, action, epistemicType, slug, agent string) error

// ResolveWikilinks extracts wikilinks from body and returns a warning string
// for each that cannot be resolved to an existing vault file.
// Uses Obsidian-style filename-based lookup across _inbox/, notes/, _deprecated/.
func ResolveWikilinks(vaultPath, body string) []string

// FindSimilarNotes scans vault note directories for notes whose title
// Jaccard similarity against incomingTokens is >= 0.7.
func FindSimilarNotes(vaultPath string, incomingTokens []string) ([]SimilarNote, error)
```

### New functions

```go
// in internal/note/promote.go

// Promote validates and moves a note from _inbox/ to notes/.
// Returns PromoteResult describing the outcome.
// Errors: slug not found, ambiguous match, lint failure, unresolved wikilinks,
//         --confirmed required but not passed, slug collision in notes/.
func Promote(vaultPath, slug string, confirmed bool) (PromoteResult, error)

// FindBySlug scans a directory for a file matching the slug pattern.
// In _inbox/: matches *-{slug}.md. In notes/: matches {slug}.md.
// Returns the full path or error (not found / ambiguous).
func FindBySlug(dir, slug string) (string, error)
```

```go
// in internal/note/deprecate.go

// Deprecate moves a note from notes/ to _deprecated/, updating its frontmatter.
// Sets status: deprecated, superseded-by, and updated date.
func Deprecate(vaultPath, slug, supersededBy string) (DeprecateResult, error)
```

</type_definitions>

<vertical_slices>

## Vertical Slices

<slice name="slice-1-export-helpers" order="1">

**Scope**: Export the three private helpers (`appendLog`, `resolveWikilinks`, `findSimilarNotes`) as public functions. Rewrite `resolveWikilinks` to use Obsidian-style filename-based resolution (scan `_inbox/`, `notes/`, `_deprecated/`). Parameterize `AppendLog` with an `action` argument. Update `Write()` to call the exported versions. Replace wikilink regex with `goldmark-wikilink` extension in `ExtractWikilinks`. All existing tests must still pass; add tests for the new resolution behavior.

**Files to create/modify**:
- `internal/note/write.go` — export `findSimilarNotes` → `FindSimilarNotes`, `resolveWikilinks` → `ResolveWikilinks`, `appendLog` → `AppendLog`; update `Write()` call sites
- `internal/note/wikilinks.go` — replace regex+codeRange with `goldmark-wikilink` AST walking
- `go.mod` / `go.sum` — add `go.abhg.dev/goldmark/wikilink` dependency

**Verification point**:
```bash
go test ./internal/note/... -count=1
# All existing tests pass. New tests verify:
# - ResolveWikilinks finds notes across _inbox/, notes/, _deprecated/
# - ResolveWikilinks excludes _meta/
# - AppendLog writes correct action string
# - ExtractWikilinks still works (same behavior, no regex)
```

**Depends on**: none

</slice>

<slice name="slice-2-frontmatter-changes" order="2">

**Scope**: Add `SupersededBy` field to Frontmatter. Rename `UpdateType` yaml tag from `"update-type"` to `"type"` (the Go field `EpistemicType` already exists with `yaml:"epistemic-type"` — consolidate to one field with `yaml:"type"`). Remove the `UpdateType` field. Handle backward compatibility: notes with `update-type:` or `epistemic-type:` in YAML should still parse. Update lint rule NF003 if it references the old field. Update `Serialize` output.

**Files to create/modify**:
- `internal/note/note.go` — add `SupersededBy`, change `EpistemicType` tag to `yaml:"type"`, remove `UpdateType`
- `internal/note/lint.go` — update any references to `UpdateType`; verify NF003 still works
- `internal/note/serialize.go` — verify output uses `type:` not `update-type:` or `epistemic-type:`
- Test files — update fixtures using `update-type:` to use `type:`

**Verification point**:
```bash
go test ./internal/note/... -count=1
# Parse round-trips with "type:" yaml key
# SupersededBy field serializes as "superseded-by:"
# Old "update-type:" notes still parse (backward compat)
```

**Depends on**: none (parallel with slice 1)

</slice>

<slice name="slice-3-promote-core" order="3">

**Scope**: Implement `Promote()` function and `FindBySlug()` in `internal/note/promote.go`. This is the core logic: find note by slug in `_inbox/`, parse, lint, check wikilinks (hard gate), check `--confirmed` gate, update frontmatter (`status: verified`, `updated` date), move file to `notes/{slug}.md`, write log entry. Unit tests cover happy path, lint failure, unresolved wikilinks, missing `--confirmed`, slug collision, and ambiguous match.

**Files to create/modify**:
- `internal/note/promote.go` — new file: `Promote()`, `FindBySlug()`, `PromoteResult`
- `internal/note/promote_test.go` — new file: unit tests

**Verification point**:
```bash
go test ./internal/note/... -count=1 -run Promote
# Happy path: note moves from _inbox/ to notes/, status=verified, log written
# Lint failure: returns error result
# Unresolved wikilink: returns error result
# Missing --confirmed on constraint: returns error result
# --confirmed on observation: silently ignored, promotes normally
# Slug collision: returns error result
```

**Depends on**: slice-1-export-helpers, slice-2-frontmatter-changes

</slice>

<slice name="slice-4-deprecate-core" order="4">

**Scope**: Implement `Deprecate()` function in `internal/note/deprecate.go`. Find note by slug in `notes/`, parse, set `status: deprecated`, `superseded-by`, update `updated` date, re-serialize, move to `_deprecated/{slug}.md`, write log entry. Unit tests cover happy path, note not found, and `_deprecated/` directory creation.

**Files to create/modify**:
- `internal/note/deprecate.go` — new file: `Deprecate()`, `DeprecateResult`
- `internal/note/deprecate_test.go` — new file: unit tests

**Verification point**:
```bash
go test ./internal/note/... -count=1 -run Deprecate
# Happy path: note moves from notes/ to _deprecated/, frontmatter updated, log written
# Note not found: returns error
# _deprecated/ created if missing
```

**Depends on**: slice-1-export-helpers, slice-2-frontmatter-changes

</slice>

<slice name="slice-5-vault-scaffold" order="5">

**Scope**: Add `_deprecated/` directory and `_meta/templates/` directory with Librarian template files to `VaultStructure()`. Add embedded template files for `librarian-agent.md` and `librarian-skill.md`. Existing init tests must still pass; add tests for new entries.

**Files to create/modify**:
- `internal/vault/structure.go` — add `_deprecated` dir entry, `_meta/templates` dir entry, template file entries
- `internal/vault/templates/librarian-agent.md` — new embedded template: OpenCode agent definition for Librarian
- `internal/vault/templates/librarian-skill.md` — new embedded template: OpenCode skill definition for Librarian
- Existing init/structure tests — verify new entries are created

**Verification point**:
```bash
go test ./internal/vault/... -count=1
# VaultStructure() includes _deprecated/, _meta/templates/, and template files
# Init creates all new entries
# Repair mode detects and fixes missing entries
```

**Depends on**: none (parallel with slices 1-4)

</slice>

<slice name="slice-6-cli-promote" order="6">

**Scope**: Add `agent-memory promote` subcommand. Wire up `--slug`, `--confirmed`, `--vault` flags. Call `Promote()` from `internal/note/`. Dual output (human/JSON). Register in `root.go`.

**Files to create/modify**:
- `internal/cli/promote.go` — new file: `newPromoteCmd()`
- `internal/cli/root.go` — add `cmd.AddCommand(newPromoteCmd())`

**Verification point**:
```bash
go build ./cmd/agent-memory && ./agent-memory promote --help
# Shows --slug, --confirmed, --vault, --json flags
go test ./test/integration/... -tags=integration -run Promote
# Integration test: write a note, promote it, verify file moved and output correct
```

**Depends on**: slice-3-promote-core

</slice>

<slice name="slice-7-cli-deprecate" order="7">

**Scope**: Add `agent-memory deprecate` subcommand. Wire up `--slug`, `--superseded-by`, `--vault` flags. Call `Deprecate()` from `internal/note/`. Dual output. Register in `root.go`.

**Files to create/modify**:
- `internal/cli/deprecate.go` — new file: `newDeprecateCmd()`
- `internal/cli/root.go` — add `cmd.AddCommand(newDeprecateCmd())`

**Verification point**:
```bash
go build ./cmd/agent-memory && ./agent-memory deprecate --help
# Shows --slug, --superseded-by, --vault, --json flags
go test ./test/integration/... -tags=integration -run Deprecate
# Integration test: promote a note, deprecate it, verify file moved to _deprecated/
```

**Depends on**: slice-4-deprecate-core

</slice>

<slice name="slice-8-instructions-update" order="8">

**Scope**: Update `agent-memory instructions` output to include Librarian workflow, promotion/deprecation commands, human confirmation flow, and deployment template pointers.

**Files to create/modify**:
- `internal/cli/instructions.go` — update hardcoded instructions string

**Verification point**:
```bash
go build ./cmd/agent-memory && ./agent-memory instructions | grep -q "promote"
go build ./cmd/agent-memory && ./agent-memory instructions | grep -q "Librarian"
go test ./test/integration/... -tags=integration -run Instructions
```

**Depends on**: slice-5-vault-scaffold (references template paths)

</slice>

<slice name="slice-9-integration-tests" order="9">

**Scope**: End-to-end integration tests covering the full Librarian workflow: write note → promote → verify in `notes/` → write replacement → deprecate old → promote new → verify wikilinks resolve across all directories. Tests use the built binary.

**Files to create/modify**:
- `test/integration/promote_test.go` — new file
- `test/integration/deprecate_test.go` — new file

**Verification point**:
```bash
go test ./test/integration/... -tags=integration -count=1
# Full workflow passes end-to-end
```

**Depends on**: slice-6-cli-promote, slice-7-cli-deprecate

</slice>

</vertical_slices>

<implementation_order>

## Implementation Order

1. **slice-1-export-helpers**: Export private helpers, rewrite wikilink resolution, replace regex with goldmark-wikilink
2. **slice-2-frontmatter-changes**: Add SupersededBy, consolidate type field, remove UpdateType
3. **slice-5-vault-scaffold**: Add _deprecated/ and _meta/templates/ to vault structure (parallel with 1+2)
4. **slice-3-promote-core**: Promote() function with validation gates
5. **slice-4-deprecate-core**: Deprecate() function
6. **slice-6-cli-promote**: CLI subcommand for promote
7. **slice-7-cli-deprecate**: CLI subcommand for deprecate
8. **slice-8-instructions-update**: Update instructions output
9. **slice-9-integration-tests**: End-to-end integration tests

**Parallel opportunities**:
- Slices 1, 2, and 5 have no dependencies on each other — can run in parallel
- Slices 3 and 4 can run in parallel (both depend on 1+2, but not on each other)
- Slices 6 and 7 can run in parallel (each depends only on its respective core slice)

</implementation_order>

</structure_artifact>
