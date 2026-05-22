# Research: context-and-reindex

## Q1: How are new CLI subcommands registered?
**Finding:** `internal/cli/root.go:11-24` — `NewRootCmd()` creates the root cobra command and calls `cmd.AddCommand(newXxxCmd())` for each subcommand. Each subcommand is a `newXxxCmd()` factory function in its own file (e.g. `newPromoteCmd()` in `promote.go`). New `context` and `reindex` commands should follow this exact pattern: create `context.go` and `reindex.go` in `internal/cli/`, each exporting a `newContextCmd()` / `newReindexCmd()` factory, and add `cmd.AddCommand(newContextCmd())` / `cmd.AddCommand(newReindexCmd())` to `root.go`.

## Q2: Where should business logic live?
**Finding:** The established pattern splits cobra wiring (`internal/cli/`) from domain logic (`internal/note/`). Promote logic is in `internal/note/promote.go`, deprecate in `internal/note/deprecate.go`, write in `internal/note/write.go`. There are no other `internal/` packages beyond `cli/`, `note/`, `vault/`, `testutil/`, and `jsonout/` (which doesn't exist — see Q4). No `internal/context/` or `internal/reindex/` packages exist. The convention would place reindex and context logic in `internal/note/` or new sibling packages.

## Q3: How does the `--vault` flag work?
**Finding:** Duplicated per command. Each command declares a local `var vaultFlag string`, adds it with `cmd.Flags().StringVar(&vaultFlag, "vault", ...)`, and calls `vault.Discover(vaultFlag)` in its `RunE`. See `promote.go:14,75`, `deprecate.go` (same pattern), `write_note.go` (same). There is no shared mechanism — it's copy-pasted. `vault.Discover()` (`internal/vault/discover.go:18`) resolves: (1) explicit path, (2) `AGENT_MEMORY_VAULT` env var, (3) walk up from cwd looking for `.agent-memory/`.

## Q4: How does `--json` work globally?
**Finding:** `root.go:8` declares `var jsonOutput bool` as a package-level variable. `root.go:17` registers it as a persistent flag: `cmd.PersistentFlags().BoolVar(&jsonOutput, "json", false, ...)`. All commands in the `cli` package reference this same `jsonOutput` var directly (e.g. `promote.go:38,60`). There is no `internal/jsonout` package — JSON marshaling is done inline with `encoding/json` in each command's `RunE`.

## Q5: What is the existing vault directory layout?
**Finding:** `internal/vault/structure.go:17-32` defines: `_meta/`, `_inbox/`, `_contested/`, `notes/`, `_deprecated/`, `_meta/templates/`, plus seed files in `_meta/`. Design doc (`AGENT_MEMORY_DESIGN.md:675`) shows index files at `notes/_index-{scope}.md`. They live in `notes/`, not `_meta/` or vault root.

## Q6: Expected format of `_index-{project}.md` files?
**Finding:** Design doc lines 425-435 show the format:
```
---
title: Golang Domain Index
type: index
---
## Constraints
- [[slug]] — one-line summary
## Patterns
- [[slug]] — one-line summary
## Observations
- [[slug]] — one-line summary
```
Grouped by epistemic type with wikilinks. Budget: ~300 tokens per index (line 437, 770).

## Q7: Expected format of `_index-{domain}.md` files?
**Finding:** The design doc uses the same format for both project and domain indices (line 675: `_index-{scope}.md` — the `{scope}` can be project or domain). No structural difference specified.

## Q8: How should `_meta/constraints-summary.md` be regenerated?
**Finding:** Design doc line 1588-1590: reindex "regenerates `_meta/constraints-summary.md` from all `status: verified` `type: constraint` notes." Current template (`constraints-summary.md`) is a 3-line placeholder: `# Constraints Summary\n\n<!-- Placeholder: document key constraints and decisions here -->`. Design doc line 442 mentions reindex "preserves human additions in a marked section" for indices, but constraints-summary is described as fully tool-regenerated.

## Q9: How does reindex discover all notes?
**Finding:** No reindex implementation exists yet. But the pattern is established: `internal/note/write.go` scans `notes/` and `_inbox/` with `filepath.Glob` for `*.md` files. `note.Parse()` (`internal/note/note.go:90`) parses any note file into `*Note` with `Frontmatter` containing `Project string` (line 32) and `Domain []string` (line 33). So: glob `notes/*.md`, parse each, extract `Project` and `Domain` from frontmatter.

## Q10: How should orphan detection work?
**Finding:** Design doc line 444-447: "Notes in `notes/` that are not referenced from any index are reported by `lint-vault --orphans`. `agent-memory reindex` adds them to the appropriate index automatically when the domain/project tags make the target unambiguous; otherwise they are surfaced for human triage." So reindex should auto-add orphans where possible and report ambiguous ones.

## Q11: Should reindex write log entries?
**Finding:** `AppendLog` signature is `AppendLog(vaultPath, action, epistemicType, slug, sourceAgent string)` at `internal/note/write.go:337`. It writes `## [timestamp] action | epistemicType | slug | by:sourceAgent`. For reindex (not per-note), sensible values: action=`reindex`, epistemicType could be empty or `index`, slug could be empty or a summary like `reindex-complete`, sourceAgent=`cli`. The function would need to accommodate empty fields or a new log format entry. Alternatively, a single summary log entry like `## [timestamp] reindex | index | all | by:cli`.

## Q12: How should idempotency be verified?
**Finding:** Design doc line 1590: "Idempotent. Replaces the entire..." — full overwrite on each run. No incremental updates. Running twice produces identical output.

## Q13: Expected JSON output for `context`?
**Finding:** Design doc line 769-770 defines two tiers:
- **Core**: `_meta/writing-protocol.md` + `_meta/constraints-summary.md` + tail of `_meta/log.md` (target <600 tokens)
- **Index**: `notes/_index-{project}.md` + `notes/_index-{domain}.md` (target ~300 per index, total <1500)
- Total target: <2000 tokens

No specific JSON field names are defined in the design doc. The existing JSON patterns use simple maps (e.g. promote returns `{"status":"promoted","slug":"...","to":"..."}`). Field names need to be designed.

## Q14: How should token counting work?
**Finding:** No token counting utility exists anywhere in the codebase. The design doc mentions budget targets (600 core, 300 per index, 2000 total) but doesn't specify a counting method. No tokenizer library is imported. `len(text)/4` heuristic would need to be implemented from scratch.

## Q15: How should `--project` and `--domain` flags map to index files?
**Finding:** Design doc line 770: "for current project + declared domains." Index files live at `notes/_index-{project}.md` and `notes/_index-{domain}.md` (line 675). If the index doesn't exist, `context` would need to handle that gracefully — no existing code addresses this case.

## Q16: How should staleness detection work?
**Finding:** `Frontmatter.ReviewBy` is a string field (`note.go:27`, yaml key `review-by`). No staleness checking code exists. The `review-by` field is set per-type TTL during write (`write.go`, referenced at design doc line 818). Checking would mean parsing all notes in `notes/`, comparing `review-by` to today's date.

## Q17: How should the log tail work?
**Finding:** `_meta/log.md` format (line 344 of `write.go`): entries are `\n## [timestamp] action | type | slug | by:agent\n`. Each entry is a `## ` line. The init entry format is slightly different: `## [timestamp] init | vault initialized`. Tail = last ~20 `## ` lines. Current vault log has 2 entries.

## Q18: What happens when total tokens exceed 2000 budget?
**Finding:** Design doc line 770: Index tier budget is "~300 tokens per index; total target < 1500 tokens." The questions reference "defers domain indices" but the design doc doesn't specify truncation behavior explicitly. The tier table shows Core is loaded "unconditionally" and Index is loaded "for current project + declared domains." Implication: Core is always included; domain indices are dropped if budget is exceeded. No implementation exists to verify.

## Q19: What is the integration test pattern?
**Finding:** `test/integration/integration_suite_test.go` — uses Ginkgo v2 + Gomega, build tag `//go:build integration`. Each test file declares a package-level `var binPath string` and calls `binPath = testutil.BuildBinary(GinkgoTB())` in a `BeforeSuite` or top-level `BeforeEach`. `testutil.BuildBinary()` (`internal/testutil/helpers.go:13`) runs `go build -o binPath ./cmd/agent-memory` and returns the binary path. Tests then call `testutil.RunBinary(binPath, args...)` which returns stdout, stderr, exitCode.

## Q20: How do integration tests create test vaults and seed notes?
**Finding:** `promote_test.go:22-29`: `tmpDir = GinkgoT().TempDir()`, `vaultPath = filepath.Join(tmpDir, "vault")`, then `testutil.RunBinary(binPath, "--json", "init", vaultPath)` to create vault, `testutil.WriteTestNote()` for body files, then `RunBinary` with `write-note` args to seed notes. Same pattern in all integration tests. Context/reindex tests should follow this: init vault, write-note to seed, then test the command.

## Q21: Should there be unit tests separate from integration tests?
**Finding:** Yes — the codebase has both. `internal/note/` has extensive `_test.go` files (e.g. `promote_test.go`, `deprecate_test.go`, `write_test.go`, `lint_test.go`, `similarity_test.go`). These test domain logic directly without invoking the binary. Integration tests in `test/integration/` test the full CLI invocation. Both layers are expected.

## Q22: Where should generated index files be written?
**Finding:** Design doc line 675 explicitly places them at `notes/_index-{scope}.md`. They live in `notes/`, which means reindex will encounter them on subsequent runs. They should be identifiable by filename pattern `_index-*.md` so reindex can skip or overwrite them.

## Q23: Should `context` fail gracefully when indices don't exist?
**Finding:** No existing code addresses this. Design doc line 769 shows Core tier is loaded "unconditionally" — it doesn't depend on indices. Index tier is loaded "for current project + declared domains." Graceful degradation (return core-only when indices missing) aligns with the tiered architecture.

## Q24: Should `context` without flags auto-detect project?
**Finding:** No auto-detection mechanism exists. The `Frontmatter.Project` field is a simple string. No code maps vault location or git context to a project name. The design doc doesn't specify auto-detection behavior for `context`.

## Q25: Should V1 preserve human additions in index files?
**Finding:** Design doc line 442: "agent-memory reindex preserves human additions in a marked section." This is stated as a feature of reindex, not a future consideration. However, no implementation exists. A marked-section approach (e.g. `<!-- BEGIN auto -->...<!-- END auto -->`) would be needed. Full overwrite is simpler for V1 but contradicts the design doc.
