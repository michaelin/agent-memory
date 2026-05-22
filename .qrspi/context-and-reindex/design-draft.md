# Design Draft: context-and-reindex

**Status:** APPROVED — decisions resolved 2026-05-22.

---

## 1. reindex command

### 1.1 Note scanning

Scan two directories: `notes/` and `_inbox/`. Glob for `*.md`, skip files matching `_index-*.md` (the files we generate). Parse each with `note.Parse()`, extract `Frontmatter.Project`, `Frontmatter.Domain`, `Frontmatter.EpistemicType`, `Frontmatter.Status`, `Frontmatter.Title`.

The slug is derived from the filename: strip date prefix and `.md` suffix. For `_inbox/` files: `2026-05-22-my-slug.md` -> `my-slug`. For `notes/` files: `my-slug.md` -> `my-slug`.

### 1.2 Index file format

Index files live at `notes/_index-{scope}.md` per design doc line 675. Both project and domain indices use the same format:

```markdown
---
title: "Index: {scope}"
type: index
status: verified
updated: {today}
---

# {Scope} Index

## Constraints
- [[slug]] -- one-line title

## Patterns
- [[slug]] -- one-line title

## Observations
- [[slug]] -- one-line title

## Assumptions
- [[slug]] -- one-line title

## Decisions
- [[slug]] -- one-line title
```

Sections with no entries are omitted entirely. Only notes with `status: verified` (in `notes/`) are included. Inbox notes are scanned for orphan detection but not indexed.

**Project index:** generated for each unique `Frontmatter.Project` value found across scanned notes. File: `notes/_index-{project}.md`.

**Domain index:** generated for each unique domain value found across all `Frontmatter.Domain` slices. File: `notes/_index-{domain}.md`.

### 1.3 Constraints summary

Regenerate `_meta/constraints-summary.md` from all notes where `EpistemicType == "constraint"` and `Status == "verified"`. Full overwrite. Format:

```markdown
# Constraints Summary

- [[slug]] -- title
- [[slug]] -- title
```

One line per constraint. Sorted alphabetically by slug.

### 1.4 Orphan detection

A note in `notes/` is orphaned if:
- It has no `Project` set AND no `Domain` values set
- OR it was not included in any generated index (shouldn't happen if project/domain are set, but catches data issues)

Orphaned notes are reported in output but not auto-fixed in V1. The design doc says reindex should auto-add orphans "when the domain/project tags make the target unambiguous" -- but if project/domain are set, they'd already be indexed. Orphans are genuinely ambiguous notes with missing metadata.

**Decision to flag:** The design doc (line 444) says reindex auto-adds orphans when unambiguous. I read this as: orphans only exist when metadata is missing/ambiguous. V1 reports them; auto-fix is deferred.

### 1.5 Human additions preservation

Indices are fully automated — no human additions to preserve. Full overwrite on every run.

### 1.6 Log entry

Single log entry per reindex run via `note.AppendLog`:
```
## [2026-05-22 14:30] reindex | index | all | by:cli
```

Using action=`reindex`, epistemicType=`index`, slug=`all`, sourceAgent=`cli`.

### 1.7 JSON output

```json
{
  "status": "ok",
  "project_indices": ["myproject"],
  "domain_indices": ["golang", "auth"],
  "constraints_count": 3,
  "notes_scanned": 12,
  "orphans": ["some-untagged-note"],
  "indices_written": [
    "notes/_index-myproject.md",
    "notes/_index-golang.md",
    "notes/_index-auth.md"
  ],
  "constraints_summary": "_meta/constraints-summary.md"
}
```

Human-readable output: print a summary line per index written, orphan count, done.

### 1.8 Idempotency

Full overwrite of all index files and constraints-summary on every run. Running twice with no vault changes produces identical files.

---

## 2. context command

### 2.1 Core tier (always loaded)

Read three files unconditionally:

1. `_meta/writing-protocol.md` -- full content
2. `_meta/constraints-summary.md` -- full content
3. `_meta/log.md` -- tail only (last ~20 `## ` entries)

Log tail extraction: split on lines starting with `## `, take last 20, rejoin.

### 2.2 Index tier (loaded by flags)

- `--project=foo` loads `notes/_index-foo.md`
- `--domain=bar` loads `notes/_index-bar.md` (repeatable flag)

If an index file doesn't exist, skip it silently and include a warning in the `warnings` array. Core tier is always returned even if no indices exist.

### 2.3 Staleness detection

Scan all `*.md` in `notes/` (skip `_index-*.md`). Parse each, compare `Frontmatter.ReviewBy` to today's date. If `review-by < today`, the note is stale. Return list of stale note slugs + their review-by dates.

**Decision to flag:** Should we scan ALL notes or only notes relevant to the requested project/domains? Scanning all is simpler and matches the design doc's "staleness scan at session start" language. Filtering to project/domain would reduce noise. Proposing: scan all, always.

### 2.4 Token counting

Simple heuristic: `words * 1.3` where words = `len(strings.Fields(text))`. No tokenizer library. This is a budget estimate, not an exact count.

Utility function in a shared location (see package structure below):

```go
func EstimateTokens(text string) int {
    return int(float64(len(strings.Fields(text))) * 1.3)
}
```

### 2.5 Budget gating

Budget: 2000 tokens total.
<!-- review: is 2000 tokens sufficient for context? I have no idea what makes sense. What do others do? -->

1. Always include Core tier (target < 600 tokens). No gating on core -- it's always loaded.
2. Add project index if requested and fits within remaining budget.
3. Add domain indices one at a time. If adding the next domain index would exceed 2000 total, skip it and add to `deferred` list in output.

Domain indices are added in the order specified by `--domain` flags. First-come, first-served.

### 2.6 JSON output

```json
{
  "status": "ok",
  "core": {
    "writing_protocol": "... content ...",
    "constraints_summary": "... content ...",
    "log_tail": "... last 20 entries ..."
  },
  "indices": {
    "myproject": "... index content ...",
    "golang": "... index content ..."
  },
  "stale_notes": [
    {"slug": "old-note", "review_by": "2026-04-01"}
  ],
  "token_estimate": 1450,
  "budget": 2000,
  "deferred": ["auth"],
  "warnings": ["index not found: _index-nonexistent.md"]
}
```

Human-readable: print core file names, index names loaded, stale count, token estimate, any deferred indices.

### 2.7 Behavior without --project/--domain

Return core tier only. No auto-detection of project. The `indices` map is empty. This is useful for agents that only need constraints and writing protocol.

---

## 3. Package structure

### 3.1 New domain logic package: `internal/reindex/`

Reason: reindex logic (scanning, index generation, orphan detection) is distinct from note CRUD operations. Putting it in `internal/note/` would make that package a grab-bag. A dedicated package keeps it focused.

Contents:
- `reindex.go` -- `Reindex(vaultPath string) (*ReindexResult, error)`
- `reindex_test.go` -- unit tests

### 3.2 New domain logic package: `internal/context/`

Reason: context bundling (tier assembly, token counting, budget gating, staleness) is its own concern.

Contents:
- `context.go` -- `BuildContext(opts ContextOptions) (*ContextResult, error)`
- `tokens.go` -- `EstimateTokens(text string) int`
- `context_test.go`, `tokens_test.go`

### 3.3 CLI layer: `internal/cli/`

Two new files following existing pattern:
- `reindex.go` -- `newReindexCmd()` factory, cobra wiring, calls `reindex.Reindex()`
- `context.go` -- `newContextCmd()` factory, cobra wiring, calls `context.BuildContext()`

Both registered in `root.go` via `cmd.AddCommand()`.

### 3.4 Shared utilities

`EstimateTokens` lives in `internal/context/` since only `context` uses it. If reindex later needs token counting (for budget-checking index sizes), it can be extracted to a shared package then.

The slug-from-filename extraction is needed by reindex and may already exist. Check `internal/note/` for existing slug utilities -- `note.ValidateSlug` exists but no `SlugFromFilename`. Add `SlugFromFilename(filename string) string` to `internal/note/`.

### 3.5 Integration tests

New file: `test/integration/reindex_test.go` and `test/integration/context_test.go`. Follow the existing pattern: `testutil.BuildBinary()`, init vault, seed notes via `write-note` + `promote`, then run `reindex`/`context` and assert JSON output.

---

## 4. Resolved decisions

| # | Question | Resolution |
|---|----------|------------|
| 1 | Where do index files live? | `notes/_index-{scope}.md` — per design doc, discoverable by agents browsing in Obsidian |
| 2 | Preserve human additions in indices? | No. Indices are fully automated, never hand-edited. Full overwrite is correct behavior. |
| 3 | Scan all notes for staleness? | Yes, scan all. No filtering by project/domain. |
| 4 | New packages vs extending `internal/note/`? | New `internal/reindex/` and `internal/context/` packages. |
| 5 | Token heuristic | `words * 1.3` — no tokenizer library needed for budget estimates. |
| 6 | Orphan auto-fix or report-only? | Report-only. |
| 7 | Include `_inbox/` notes in indices? | No — verified notes only. Librarian promotes immediately after writing (via plugin hook). |
| 8 | AppendLog for reindex? | Use existing signature with `slug="all"` convention. |
