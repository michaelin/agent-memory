# Questions: context-and-reindex

## Codebase Architecture

1. How are new CLI subcommands registered? (root.go `AddCommand` pattern — do `context` and `reindex` follow the same `newXxxCmd()` factory pattern?)
2. Where should the business logic for `context` and `reindex` live? Existing commands split logic between `internal/cli/` (cobra wiring) and `internal/note/` (domain logic). Should these get new packages (e.g. `internal/context/`, `internal/reindex/`) or live in `internal/note/`?
3. How does the `--vault` flag work? Promote/deprecate declare a local `vaultFlag` string and call `vault.Discover(vaultFlag)`. Is this duplicated per command or is there a shared mechanism?
4. How does `--json` work globally? It's a persistent flag on root (`jsonOutput` package var). Do all commands use the same `jsonOutput` var for conditional output?
5. What is the existing vault directory layout? Where do index files (`_index-*.md`) belong — in vault root, `notes/`, or `_meta/`? The design doc shows them in `notes/` with frontmatter.

## Implementation Details — reindex

6. What is the expected format/content of `_index-{project}.md` files? The design doc (§4.2) shows frontmatter with `type: index`, title, and sections grouped by epistemic type (Constraints, Patterns, Observations) with wikilinks. Is this the exact format to generate?
7. What is the expected format of `_index-{domain}.md` files? Same as project indices or different structure?
8. How should `_meta/constraints-summary.md` be regenerated? The template is a placeholder. Should reindex replace its entire content, or preserve a marked section for human additions (as design doc mentions for indices)?
9. How does reindex discover all notes? Scan `notes/` and `_inbox/` directories for `*.md` files, parse each with `note.Parse()`, and extract frontmatter fields `project` and `domain`?
10. How should orphan detection work? A note is orphaned if it exists in `notes/` but is not referenced from any index. Should orphans be reported in JSON output, added to indices automatically, or both?
11. Should reindex write log entries via the existing `AppendLog` function? The current signature is `AppendLog(vaultPath, action, epistemicType, slug, sourceAgent)` — what values make sense for a reindex operation that isn't per-note?
12. How should idempotency be verified? Overwrite index files entirely on each run, so re-running produces identical output?

## Implementation Details — context

13. What is the expected JSON output structure for `agent-memory context`? The ticket mentions Core tier (writing-protocol, constraints-summary, log tail) and Index tier (project/domain indices). What are the JSON field names?
14. How should token counting work? Is there an existing token counting utility, or do we need to implement one? The design doc mentions targets (600 tokens core, 300 per index, 2000 total) — is a simple `len(text)/4` heuristic sufficient or do we need a proper tokenizer?
15. How should the `--project` and `--domain` flags map to index file loading? `--project=foo` loads `_index-foo.md`, `--domain=bar` loads `_index-bar.md`? What if the index doesn't exist (reindex hasn't been run)?
16. How should staleness detection work? Parse all notes in `notes/`, compare `review-by` field to today's date, return list of stale slugs? Or only check notes relevant to the requested project/domains?
17. How should the log tail work? Read `_meta/log.md`, extract last ~20 entries, include in the context payload? What's an "entry" — each `## timestamp ...` block?
18. What happens when total tokens exceed the 2000 budget? The ticket says "defers domain indices." Does that mean domain indices are dropped from the response, or truncated, or returned with a warning?

## Integration & Testing

19. What is the integration test pattern? Tests use `testutil.RunBinary(binPath, args...)` to invoke the compiled binary. How is `binPath` set up? (Check `integration_suite_test.go` for the build step.)
20. How do existing integration tests create test vaults and seed notes? The promote test calls `init` then `write-note` via the binary. Should context/reindex tests follow the same pattern?
21. Should there be unit tests for the index generation logic separate from integration tests? Existing packages have both `_test.go` files in `internal/note/` and integration tests in `test/integration/`.

## Open Design Questions

22. Where should generated index files be written — vault root or `notes/`? The design doc shows indices as notes in `notes/` with frontmatter, but they're auto-generated, not manually written. Writing to `notes/` means they'd be scanned by reindex itself on subsequent runs.
23. Should `context` fail gracefully when indices don't exist (reindex never run), or should it return whatever is available (core tier only)?
24. Should `context` without `--project` or `--domain` flags return only core tier, or auto-detect project from vault location / git context?
25. The design doc mentions `agent-memory reindex` preserving human additions in a "marked section." Should V1 implement this preservation, or is full overwrite acceptable for the initial increment?
