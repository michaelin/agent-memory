# Structure Draft: context-and-reindex

**Status:** Draft for human review.

---

## Vertical Slices

### Slice 1: reindex

Independently testable. No dependency on context.

1. `internal/note/slug.go` — `SlugFromFilename` utility
2. `internal/reindex/reindex.go` — scan, generate indices, generate constraints summary, detect orphans, write log
3. `internal/cli/reindex.go` — cobra wiring
4. `internal/cli/root.go` — register `newReindexCmd()`
5. `test/integration/reindex_test.go` — end-to-end

### Slice 2: context

Depends on slice 1 (reads index files that reindex generates, but degrades gracefully without them).

1. `internal/context/tokens.go` — token estimation
2. `internal/context/context.go` — core tier, index tier, staleness, budget gating
3. `internal/cli/context.go` — cobra wiring
4. `internal/cli/root.go` — register `newContextCmd()`
5. `test/integration/context_test.go` — end-to-end

---

## Type Signatures

### `internal/note/slug.go`

```go
package note

// SlugFromFilename extracts the slug from a note filename.
// Handles both inbox format ("2026-05-22-my-slug.md" -> "my-slug")
// and notes format ("my-slug.md" -> "my-slug").
func SlugFromFilename(filename string) string
```

### `internal/reindex/reindex.go`

```go
package reindex

// ReindexResult holds the outcome of a reindex operation.
type ReindexResult struct {
    Status            string   `json:"status"`              // "ok"
    ProjectIndices    []string `json:"project_indices"`     // project slugs indexed
    DomainIndices     []string `json:"domain_indices"`      // domain slugs indexed
    ConstraintsCount  int      `json:"constraints_count"`   // number of constraints in summary
    NotesScanned      int      `json:"notes_scanned"`       // total notes parsed
    Orphans           []string `json:"orphans"`             // slugs with no project/domain
    IndicesWritten    []string `json:"indices_written"`     // relative paths of written index files
    ConstraintsSummary string  `json:"constraints_summary"` // relative path of constraints file
}

// Reindex scans all notes in the vault, regenerates project/domain index files
// and the constraints summary. Idempotent — full overwrite on each run.
// Writes a single log entry via note.AppendLog.
func Reindex(vaultPath string) (*ReindexResult, error)
```

### `internal/reindex/reindex_test.go`

```go
package reindex

func TestReindex_EmptyVault(t *testing.T)          // no notes -> no indices, no errors
func TestReindex_SingleProject(t *testing.T)        // one project note -> one project index
func TestReindex_MultipleDomains(t *testing.T)      // notes across domains -> multiple domain indices
func TestReindex_ConstraintsSummary(t *testing.T)   // constraint notes -> constraints-summary.md
func TestReindex_SkipsIndexFiles(t *testing.T)      // _index-*.md files not scanned as notes
func TestReindex_OrphanDetection(t *testing.T)      // notes with no project/domain -> orphans list
func TestReindex_Idempotent(t *testing.T)           // two runs produce identical output
func TestReindex_OnlyVerifiedNotes(t *testing.T)    // inbox notes excluded from indices
func TestReindex_OmitsEmptySections(t *testing.T)   // index sections with no entries are omitted
```

### `internal/context/tokens.go`

```go
package context

// EstimateTokens returns a rough token count for the given text.
// Heuristic: word count * 1.3.
func EstimateTokens(text string) int
```

### `internal/context/context.go`

```go
package context

// ContextOptions configures what to include in the context bundle.
type ContextOptions struct {
    VaultPath string   // resolved vault path
    Project   string   // project slug for index loading (optional)
    Domains   []string // domain slugs for index loading (optional)
    Budget    int      // max token budget (default 2000)
}

// StaleNote represents a note past its review-by date.
type StaleNote struct {
    Slug     string `json:"slug"`
    ReviewBy string `json:"review_by"`
}

// CoreContent holds the always-loaded core tier files.
type CoreContent struct {
    WritingProtocol    string `json:"writing_protocol"`
    ConstraintsSummary string `json:"constraints_summary"`
    LogTail            string `json:"log_tail"`
}

// ContextResult is the full context bundle returned to agents.
type ContextResult struct {
    Status        string            `json:"status"`         // "ok"
    Core          CoreContent       `json:"core"`
    Indices       map[string]string `json:"indices"`        // scope -> content
    StaleNotes    []StaleNote       `json:"stale_notes"`
    TokenEstimate int               `json:"token_estimate"`
    Budget        int               `json:"budget"`
    Deferred      []string          `json:"deferred"`       // indices skipped due to budget
    Warnings      []string          `json:"warnings"`       // e.g. missing index files
}

// BuildContext assembles the context bundle for an agent session.
// Core tier is always loaded. Index tier is loaded per options and gated by budget.
// Staleness is detected across all verified notes.
func BuildContext(opts ContextOptions) (*ContextResult, error)
```

### `internal/context/context_test.go`

```go
package context

func TestBuildContext_CoreOnly(t *testing.T)         // no project/domain -> core tier only
func TestBuildContext_WithProject(t *testing.T)       // project flag loads project index
func TestBuildContext_WithDomains(t *testing.T)       // domain flags load domain indices
func TestBuildContext_MissingIndex(t *testing.T)      // missing index -> warning, no error
func TestBuildContext_BudgetGating(t *testing.T)      // domain deferred when budget exceeded
func TestBuildContext_StalenessDetection(t *testing.T)// notes past review-by detected
func TestBuildContext_EmptyVault(t *testing.T)        // missing files -> graceful degradation
func TestEstimateTokens(t *testing.T)                 // token heuristic sanity checks
```

### `internal/cli/reindex.go`

```go
package cli

// newReindexCmd creates the "reindex" cobra subcommand.
// Flags: --vault
// Calls reindex.Reindex(), outputs JSON or human-readable summary.
func newReindexCmd() *cobra.Command
```

### `internal/cli/context.go`

```go
package cli

// newContextCmd creates the "context" cobra subcommand.
// Flags: --vault, --project, --domain (repeatable)
// Calls context.BuildContext(), outputs JSON or human-readable summary.
func newContextCmd() *cobra.Command
```

### `internal/cli/root.go`

Add two lines:

```go
cmd.AddCommand(newReindexCmd())
cmd.AddCommand(newContextCmd())
```

---

## Integration Test Signatures

### `test/integration/reindex_test.go`

```go
// Pattern: init vault, seed notes via write-note + promote, run reindex, assert.

var _ = Describe("reindex command", func() {
    It("generates project index from promoted notes", func() { ... })
    It("generates domain index from promoted notes", func() { ... })
    It("generates constraints summary from constraint notes", func() { ... })
    It("reports orphaned notes with no project/domain", func() { ... })
    It("is idempotent — second run produces identical output", func() { ... })
    It("returns valid JSON with --json flag", func() { ... })
})
```

### `test/integration/context_test.go`

```go
// Pattern: init vault, seed + promote notes, run reindex, then run context.

var _ = Describe("context command", func() {
    It("returns core tier without flags", func() { ... })
    It("includes project index with --project flag", func() { ... })
    It("includes domain indices with --domain flags", func() { ... })
    It("warns when requested index does not exist", func() { ... })
    It("detects stale notes past review-by date", func() { ... })
    It("defers indices when budget is exceeded", func() { ... })
    It("returns valid JSON with --json flag", func() { ... })
})
```

---

## Files Changed Summary

| File | Action |
|------|--------|
| `internal/note/slug.go` | New — `SlugFromFilename` |
| `internal/note/slug_test.go` | New — unit tests |
| `internal/reindex/reindex.go` | New — `Reindex`, `ReindexResult` |
| `internal/reindex/reindex_test.go` | New — unit tests |
| `internal/context/tokens.go` | New — `EstimateTokens` |
| `internal/context/tokens_test.go` | New — unit tests |
| `internal/context/context.go` | New — `BuildContext`, types |
| `internal/context/context_test.go` | New — unit tests |
| `internal/cli/reindex.go` | New — `newReindexCmd` |
| `internal/cli/context.go` | New — `newContextCmd` |
| `internal/cli/root.go` | Modified — two `AddCommand` calls |
| `test/integration/reindex_test.go` | New — integration tests |
| `test/integration/context_test.go` | New — integration tests |
