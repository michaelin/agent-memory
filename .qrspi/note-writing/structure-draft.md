<structure_artifact feature="note-writing" status="draft">

## Type Signatures

### New types in `internal/note/`

```go
// WriteOptions holds the parameters for writing a new note to the vault inbox.
type WriteOptions struct {
	VaultPath      string
	Title          string
	EpistemicType  string
	Body           string
	Project        string
	Domain         []string
	Scope          string
	SourceArtifact string
	SourceAgent    string
	Confidence     string
	Tags           []string
	Force          bool // skip similarity check refusal
}

// WriteResult is the outcome of a write-note operation.
type WriteResult struct {
	Status     string            `json:"status"`               // "written" or "refused"
	Path       string            `json:"path,omitempty"`       // relative path within vault
	Reason     string            `json:"reason,omitempty"`     // refusal reason
	Warnings   []string          `json:"warnings,omitempty"`   // e.g. unresolved wikilinks
	Candidates []SimilarNote     `json:"candidates,omitempty"` // similar notes found
	Errors     []LintError       `json:"errors,omitempty"`     // validation errors
}

// SimilarNote represents an existing note that is similar to the incoming one.
type SimilarNote struct {
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Similarity float64 `json:"similarity"`
}
```

### New functions in `internal/note/`

```go
// Serialize produces a complete note file (frontmatter + body) from a Note.
func Serialize(n *Note) ([]byte, error)

// Slug converts a title to a filesystem-safe slug (markdown anchor style).
func Slug(title string) string

// NormalizeTokens lowercases, splits, removes punctuation and stop words.
func NormalizeTokens(text string) []string

// Jaccard computes the Jaccard similarity index between two token sets.
func Jaccard(a, b []string) float64

// ExtractWikilinks returns all [[slug]] references found in the text.
func ExtractWikilinks(text string) []string

// Write assembles a note from WriteOptions, validates, similarity-checks,
// and writes to _inbox/. Returns WriteResult.
func Write(opts WriteOptions) (*WriteResult, error)
```

### New functions in `internal/vault/`

```go
// Discover resolves a vault path from explicit path, env var, or directory walk-up.
// Returns the absolute path to the vault directory, or an error if not found.
func Discover(explicitPath string) (string, error)
```

### New function in `internal/note/` (harness detection)

```go
// DetectSourceAgent reads harness environment variables to construct
// a session-scoped source-agent identifier. Returns empty string if
// no harness is detected and no override is set.
func DetectSourceAgent() string
```

### Modified types

```go
// Lint rule NF001: relax source-agent from required to optional.
// No struct change — logic change only in ruleNF001.Check.
```

### New file: `internal/cli/write_note.go`

```go
func newWriteNoteCmd() *cobra.Command
// Flags: --vault, --type, --title, --project, --domain, --scope,
//        --source-artifact, --source-agent, --confidence, --tags, --force
// Args: <body-file> (positional, "-" for stdin)
```

## Vertical Slices

### Slice 1: Serialize + Slug (foundation)

**Scope**: `Serialize()` produces valid note files from `Note` structs. `Slug()` converts titles to filesystem-safe slugs. These are pure functions with no I/O — the foundation everything else builds on.

**Files**:
- `internal/note/serialize.go` — `Serialize()`, `Slug()`
- `internal/note/serialize_test.go` — unit tests

**Verification**: `go test ./internal/note/ -run Serialize` and `-run Slug` pass. Round-trip test: `Parse(Serialize(note))` produces equivalent note.

**Dependencies**: None.

### Slice 2: Vault discovery

**Scope**: `Discover()` resolves vault path from explicit → env var → walk-up → error. Pure filesystem logic.

**Files**:
- `internal/vault/discover.go` — `Discover()`
- `internal/vault/discover_test.go` — unit tests (temp dirs with `.agent-memory/`)

**Verification**: `go test ./internal/vault/ -run Discover` passes. Tests cover: explicit path, env var, walk-up from nested dir, no vault found error.

**Dependencies**: None.

### Slice 3: Similarity check + wikilink extraction

**Scope**: `NormalizeTokens()`, `Jaccard()`, `ExtractWikilinks()`. Pure functions, no I/O.

**Files**:
- `internal/note/similarity.go` — `NormalizeTokens()`, `Jaccard()`
- `internal/note/wikilinks.go` — `ExtractWikilinks()`
- `internal/note/similarity_test.go` — unit tests
- `internal/note/wikilinks_test.go` — unit tests

**Verification**: `go test ./internal/note/ -run Jaccard` and `-run Wikilink` pass. Tests cover: identical titles → 1.0, disjoint → 0.0, threshold boundary, stop word removal, `[[slug]]` extraction including `[[slug|alias]]` form.

**Dependencies**: None.

### Slice 4: Harness detection + source-agent

**Scope**: `DetectSourceAgent()` reads env vars and returns session identifier. Relax NF001 to make `source-agent` optional.

**Files**:
- `internal/note/source_agent.go` — `DetectSourceAgent()`
- `internal/note/source_agent_test.go` — unit tests (set/unset env vars)
- `internal/note/lint.go` — modify `ruleNF001` to skip `source-agent` check

**Verification**: `go test ./internal/note/ -run DetectSourceAgent` passes. Existing lint tests still pass with relaxed NF001.

**Dependencies**: None.

### Slice 5: Write pipeline (integration)

**Scope**: `Write()` ties everything together: assemble note from `WriteOptions`, auto-populate fields, lint, similarity scan (reads `_inbox/` and `notes/`), wikilink resolution check, slug generation, serialize, write file, append log. This is the first slice with real I/O.

**Files**:
- `internal/note/write.go` — `Write()`, `WriteOptions`, `WriteResult`, `SimilarNote`
- `internal/note/write_test.go` — unit/integration tests against temp vault dirs

**Verification**: `go test ./internal/note/ -run Write` passes. Tests cover:
- Happy path: valid note written to `_inbox/`
- Lint failure: refused with errors
- Similarity hit: refused with candidates
- Similarity hit + `--force`: written anyway
- Wikilink warning: written with warnings
- File collision: suffix appended
- Log entry appended to `_meta/log.md`
- Auto-populated fields correct (created, updated, status, review-by, etc.)

**Dependencies**: Slices 1, 2, 3, 4.

### Slice 6: CLI command + integration tests

**Scope**: `newWriteNoteCmd()` wired into root command. Dual output (human-readable + JSON). Body from file or stdin. Integration tests using `BuildBinary`/`RunBinary`.

**Files**:
- `internal/cli/write_note.go` — `newWriteNoteCmd()`
- `internal/cli/root.go` — register `write-note` subcommand
- `internal/cli/instructions.go` — remove "Writing to the vault is not yet enabled" text
- `test/integration/write_note_test.go` — integration tests

**Verification**: `go test ./test/integration/ -run WriteNote` passes. Tests cover:
- CLI happy path: file arg, required flags, note appears in `_inbox/`
- CLI with `--json`: JSON output matches `WriteResult` shape
- CLI stdin (`-`): body piped via stdin
- CLI refusal: similar note, JSON output with candidates
- CLI error: missing required flags, missing vault
- Exit codes: 0 on success, 1 on refusal/error

**Dependencies**: Slice 5.

### Slice 7: Writing protocol v3 + template fix

**Scope**: Fix wrong enum values in writing-protocol.md template. Update to v3 with write-note usage instructions. Update existing vault init tests if they assert template content.

**Files**:
- `internal/vault/templates/writing-protocol.md` — fix enums, add v3 content
- `internal/vault/init_test.go` — update assertions if needed
- `test/integration/init_test.go` — update assertions if needed

**Verification**: `go test ./...` passes. Manual review of template content.

**Dependencies**: None (can run in parallel with slices 1–6, but placed last for clean sequencing).

## Implementation Order

```
Slice 1 (Serialize+Slug) ──┐
Slice 2 (Vault discovery) ─┤
Slice 3 (Similarity+Wiki) ─┼── Slice 5 (Write pipeline) ── Slice 6 (CLI + integration)
Slice 4 (Harness+NF001) ───┘
Slice 7 (Protocol v3) ──────── (independent, any time)
```

Slices 1–4 are independent and can be implemented in parallel.
Slice 5 depends on all of 1–4.
Slice 6 depends on 5.
Slice 7 is independent.

</structure_artifact>
