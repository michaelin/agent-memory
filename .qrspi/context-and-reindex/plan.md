# Plan: context-and-reindex

**Status:** Ready for implementation.

Constrained by: design-draft.md (APPROVED), structure-draft.md.

---

## Slice 1: reindex

### Task 1.1: `SlugFromFilename` utility

- **File:** `internal/note/slug.go`
- **What:** Implement `SlugFromFilename(filename string) string`. Strip `.md` suffix. If filename matches `YYYY-MM-DD-*` pattern (inbox format), strip the date prefix. Use `strings.TrimSuffix` and a regex or manual prefix check.
- **Test:** `internal/note/slug_test.go` — table-driven: `"my-slug.md" -> "my-slug"`, `"2026-05-22-my-slug.md" -> "my-slug"`, `"no-extension" -> "no-extension"`, `"2026-05-22-.md" -> ""` (edge case).

### Task 1.2: `Reindex` core logic

- **File:** `internal/reindex/reindex.go`
- **What:**
  1. Glob `notes/*.md` and `_inbox/*.md`, skip `_index-*.md`.
  2. Parse each with `note.Parse()`, extract frontmatter fields.
  3. Build maps: `projectNotes map[string][]noteEntry`, `domainNotes map[string][]noteEntry` where `noteEntry` is `{Slug, Title, EpistemicType, Status}`.
  4. Only index notes with `Status == "verified"` (i.e., in `notes/`).
  5. For each project key, render `notes/_index-{project}.md` using the format from design 1.2. Group entries by epistemic type. Omit empty sections.
  6. Same for each domain key.
  7. Collect all `constraint` + `verified` notes, sort by slug, write `_meta/constraints-summary.md`.
  8. Detect orphans: notes in `notes/` with no `Project` AND empty `Domain`.
  9. Call `note.AppendLog(vaultPath, "reindex", "index", "all", "cli")`.
  10. Return `*ReindexResult`.
- **Internal type:** `type noteEntry struct { Slug, Title, EpistemicType string }` (unexported).
- **Test:** `internal/reindex/reindex_test.go` — create temp vault dirs manually (no binary needed), write `.md` files with frontmatter, call `Reindex()`, assert files written and result fields. See structure-draft for test list.

### Task 1.3: CLI wiring for reindex

- **File:** `internal/cli/reindex.go`
- **What:** `newReindexCmd()` factory. Flags: `--vault`. RunE: discover vault, call `reindex.Reindex(vaultPath)`, output JSON or human-readable. Follow `promote.go` error handling pattern exactly (json error wrapping, `SilenceUsage`, `SilenceErrors`).
- **Human output:** Print each index file written, orphan count, "Done." line.
- **Test:** Covered by integration tests.

### Task 1.4: Register reindex command

- **File:** `internal/cli/root.go`
- **What:** Add `cmd.AddCommand(newReindexCmd())` after existing commands.
- **Test:** Covered by integration tests.

### Task 1.5: Integration tests for reindex

- **File:** `test/integration/reindex_test.go`
- **What:** Follow `init_test.go` pattern. Uses `binPath` from `init_test.go`'s `BeforeSuite`. Setup helper: init vault, write-note + promote several notes with varying project/domain/type. Test cases per structure-draft. Assert JSON output fields and file existence.
- **Seed data needed:**
  - 2 notes with `project=myproject`, different domains, different types
  - 1 note with `type=constraint`, `status=verified`
  - 1 note with no project and no domain (orphan)
  - 1 note in `_inbox/` (should not appear in indices)

---

## Slice 2: context

### Task 2.1: `EstimateTokens` utility

- **File:** `internal/context/tokens.go`
- **What:** `func EstimateTokens(text string) int` — `int(float64(len(strings.Fields(text))) * 1.3)`. Handle empty string (return 0).
- **Test:** `internal/context/tokens_test.go` — table-driven: empty string, single word, multi-word text, verify reasonable estimates.

### Task 2.2: `BuildContext` core logic

- **File:** `internal/context/context.go`
- **What:**
  1. Define `ContextOptions`, `CoreContent`, `StaleNote`, `ContextResult` per structure-draft.
  2. Default `Budget` to 2000 if zero.
  3. **Core tier:** Read `_meta/writing-protocol.md`, `_meta/constraints-summary.md`, `_meta/log.md`. For log: split on lines starting `## `, take last 20, rejoin. If files missing, use empty string (no error).
  4. **Token tracking:** Sum core tier tokens.
  5. **Index tier:** If `Project` set, try reading `notes/_index-{project}.md`. If exists, estimate tokens; if fits in remaining budget, add to `Indices` map. Else add to `Deferred`. If file missing, add to `Warnings`.
  6. Same for each domain in `Domains`, in order. First-come first-served.
  7. **Staleness:** Glob `notes/*.md`, skip `_index-*.md`. Parse each, compare `ReviewBy` to `time.Now().Format("2006-01-02")`. If `ReviewBy < today` and non-empty, add to `StaleNotes`.
  8. Set `TokenEstimate`, `Budget`, `Status = "ok"`.
- **Test:** `internal/context/context_test.go` — create temp vault manually, seed files. See structure-draft test list. For budget gating test: create large index content that exceeds remaining budget.

### Task 2.3: CLI wiring for context

- **File:** `internal/cli/context.go`
- **What:** `newContextCmd()` factory. Flags: `--vault`, `--project` (string), `--domain` (string slice, repeatable). RunE: discover vault, build `ContextOptions`, call `context.BuildContext(opts)`, output JSON or human-readable.
- **Human output:** List core files loaded, indices loaded, stale count, token estimate, deferred list.
- **Test:** Covered by integration tests.

### Task 2.4: Register context command

- **File:** `internal/cli/root.go`
- **What:** Add `cmd.AddCommand(newContextCmd())` after reindex registration.
- **Test:** Covered by integration tests.

### Task 2.5: Integration tests for context

- **File:** `test/integration/context_test.go`
- **What:** Follow same pattern. Setup: init vault, write+promote notes, run reindex first to generate indices, then test context. Test cases per structure-draft.
- **Seed data needed:**
  - Same notes as reindex tests (can share setup helper)
  - One note with `review-by` in the past (stale)
  - Run `reindex` before `context` tests to ensure indices exist

---

## Implementation Order

1. Task 1.1 (slug utility) — no dependencies
2. Task 1.2 (reindex logic) — depends on 1.1
3. Task 1.3 + 1.4 (CLI wiring) — depends on 1.2
4. Task 1.5 (integration tests) — depends on 1.3
5. Task 2.1 (token utility) — no dependencies, can parallelize with slice 1
6. Task 2.2 (context logic) — depends on 2.1
7. Task 2.3 + 2.4 (CLI wiring) — depends on 2.2
8. Task 2.5 (integration tests) — depends on 2.3

Tasks 1.1 and 2.1 are independent and can be implemented in parallel.

---

## Risks and Notes

- **Log tail parsing:** The `## ` split approach assumes all log entries start with `## [`. Verify against `_meta/log.md` format in init. The init entry uses this format already.
- **Date comparison for staleness:** Simple string comparison works because `review-by` uses `YYYY-MM-DD` format (lexicographic order matches chronological order).
- **Package initialization:** New packages `internal/reindex/` and `internal/context/` need no `init()` — they're imported by CLI layer only.
