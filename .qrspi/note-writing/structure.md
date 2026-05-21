<structure_artifact feature="note-writing">

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
	Force          bool
}

// WriteResult is the outcome of a write-note operation.
type WriteResult struct {
	Status     string        `json:"status"`
	Path       string        `json:"path,omitempty"`
	Reason     string        `json:"reason,omitempty"`
	Warnings   []string      `json:"warnings,omitempty"`
	Candidates []SimilarNote `json:"candidates,omitempty"`
	Errors     []LintError   `json:"errors,omitempty"`
}

// SimilarNote represents an existing note similar to the incoming one.
type SimilarNote struct {
	Path       string  `json:"path"`
	Title      string  `json:"title"`
	Similarity float64 `json:"similarity"`
}
```

### New functions in `internal/note/`

```go
func Serialize(n *Note) ([]byte, error)
func Slug(title string) string
func NormalizeTokens(text string) []string
func Jaccard(a, b []string) float64
func ExtractWikilinks(text string) []string
func DetectSourceAgent() string
func Write(opts WriteOptions) (*WriteResult, error)
```

### New functions in `internal/vault/`

```go
func Discover(explicitPath string) (string, error)
```

### Modified logic

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

### Slice 1: Serialize + Slug

**Scope**: `Serialize()` produces valid note files from `Note` structs. `Slug()` converts titles to filesystem-safe slugs. Pure functions, no I/O.

**Files**:
- `internal/note/serialize.go`
- `internal/note/serialize_test.go`

**Verification**: `go test ./internal/note/ -run Serialize` and `-run Slug` pass. Round-trip: `Parse(Serialize(note))` produces equivalent note.

**Dependencies**: None.

### Slice 2: Vault discovery

**Scope**: `Discover()` resolves vault path: explicit → env var → walk-up → error.

**Files**:
- `internal/vault/discover.go`
- `internal/vault/discover_test.go`

**Verification**: `go test ./internal/vault/ -run Discover` passes. Covers: explicit path, env var, walk-up from nested dir, not-found error.

**Dependencies**: None.

### Slice 3: Similarity check + wikilink extraction

**Scope**: `NormalizeTokens()`, `Jaccard()`, `ExtractWikilinks()`. Pure functions.

**Files**:
- `internal/note/similarity.go`
- `internal/note/wikilinks.go`
- `internal/note/similarity_test.go`
- `internal/note/wikilinks_test.go`

**Verification**: `go test ./internal/note/ -run Jaccard` and `-run Wikilink` pass. Covers: identical → 1.0, disjoint → 0.0, threshold boundary, stop words, `[[slug]]` and `[[slug|alias]]` extraction.

**Dependencies**: None.

### Slice 4: Harness detection + NF001 relaxation

**Scope**: `DetectSourceAgent()` reads harness env vars. Relax NF001 to make `source-agent` optional.

**Files**:
- `internal/note/source_agent.go`
- `internal/note/source_agent_test.go`
- `internal/note/lint.go` (modify `ruleNF001`)

**Verification**: `go test ./internal/note/ -run DetectSourceAgent` passes. Existing lint tests pass with relaxed NF001.

**Dependencies**: None.

### Slice 5: Write pipeline

**Scope**: `Write()` integrates all prior slices: assemble note from `WriteOptions`, auto-populate fields, lint, similarity scan, wikilink resolution, slug generation, serialize, write file, append log entry.

**Files**:
- `internal/note/write.go`
- `internal/note/write_test.go`

**Verification**: `go test ./internal/note/ -run Write` passes. Covers: happy path, lint failure, similarity refusal, force override, wikilink warnings, file collision suffix, log entry, auto-populated fields.

**Dependencies**: Slices 1, 2, 3, 4.

### Slice 6: CLI command + integration tests

**Scope**: `newWriteNoteCmd()` wired into root. Dual output. Body from file or stdin. Integration tests via `BuildBinary`/`RunBinary`. Remove "Writing to the vault is not yet enabled" from instructions.

**Files**:
- `internal/cli/write_note.go`
- `internal/cli/root.go` (register subcommand)
- `internal/cli/instructions.go` (update text)
- `test/integration/write_note_test.go`

**Verification**: `go test ./test/integration/ -run WriteNote` passes. Covers: file arg, `--json`, stdin, refusal output, missing flags, missing vault, exit codes.

**Dependencies**: Slice 5.

### Slice 7: Writing protocol v3

**Scope**: Fix wrong enum values in template. Add write-note usage instructions.

**Files**:
- `internal/vault/templates/writing-protocol.md`
- Test files updated if they assert template content.

**Verification**: `go test ./...` passes.

**Dependencies**: None.

## Implementation Order

```
Slice 1 (Serialize+Slug) ──┐
Slice 2 (Vault discovery) ─┤
Slice 3 (Similarity+Wiki) ─┼── Slice 5 (Write pipeline) ── Slice 6 (CLI + integration)
Slice 4 (Harness+NF001) ───┘
Slice 7 (Protocol v3) ──────── (independent)
```

Slices 1–4 are independent and parallelizable.
Slice 5 depends on 1–4.
Slice 6 depends on 5.
Slice 7 is independent.

</structure_artifact>
