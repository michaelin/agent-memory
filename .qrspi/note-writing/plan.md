<plan_artifact feature="note-writing">

<objective>

## Objective

Implement `agent-memory write-note` — a deterministic tool that accepts a body file and CLI flags, assembles a complete note with auto-populated frontmatter, validates it, checks for duplicates via Jaccard similarity, and writes it to `_inbox/{YYYY-MM-DD}-{slug}.md` with a log entry. Also fixes the writing protocol v2 enum bug and updates to v3.

**Constrained by:**
- Design decisions in `.qrspi/note-writing/design.md`
- Structure outline in `.qrspi/note-writing/structure.md`

</objective>

<slices>

<slice name="slice-1-serialize-slug">

## Slice 1: Serialize + Slug

**Goal**: Pure functions to produce note files from `Note` structs and convert titles to slugs.

<tasks>

<task>
**Name**: Task 1.1 — Implement Serialize
**Files**: `internal/note/serialize.go`
**Action**: Implement `Serialize(n *Note) ([]byte, error)`. Use `yaml.Marshal()` for frontmatter, produce `---\n{yaml}\n---\n{body}\n` format. Handle nil note error. Ensure field order follows struct tag order. (Design decision #8)
**Verify**: `go test ./internal/note/ -run Serialize`
**Done**: Serialize produces valid frontmatter+body. Round-trip `Parse(Serialize(note))` produces equivalent note.
</task>

<task>
**Name**: Task 1.2 — Implement Slug
**Files**: `internal/note/serialize.go`
**Action**: Implement `Slug(title string) string`. Markdown anchor-style: lowercase, replace spaces with hyphens, strip non-alphanumeric (except hyphens), collapse consecutive hyphens, trim leading/trailing hyphens, truncate to 60 chars. (Design decision #5)
**Verify**: `go test ./internal/note/ -run Slug`
**Done**: Slug handles: normal titles, special characters, unicode, empty string, titles exceeding 60 chars.
</task>

<task>
**Name**: Task 1.3 — Write tests
**Files**: `internal/note/serialize_test.go`
**Action**: Ginkgo/gomega BDD tests. Describe("Serialize") with contexts for: complete note, minimal note, nil note, empty body, empty frontmatter fields. Describe("Slug") with contexts for: normal title, special chars, consecutive hyphens, long title truncation, empty string, unicode.
**Verify**: `go test ./internal/note/ -run Serialize -run Slug -v`
**Done**: All tests pass. Round-trip test included.
</task>

</tasks>

<checkpoint>
**Slice 1 Checkpoint**:
- [ ] `Serialize` round-trips with `Parse`
- [ ] `Slug` produces filesystem-safe output for all edge cases
- [ ] `go test ./internal/note/ -run "Serialize|Slug"` passes
</checkpoint>

</slice>

<slice name="slice-2-vault-discovery">

## Slice 2: Vault Discovery

**Goal**: Resolve vault path from explicit argument, env var, or directory walk-up.

<tasks>

<task>
**Name**: Task 2.1 — Implement Discover
**Files**: `internal/vault/discover.go`
**Action**: Implement `Discover(explicitPath string) (string, error)`. Priority: (1) explicitPath if non-empty, verify `.agent-memory` dir exists at path or path is a `.agent-memory` dir; (2) `AGENT_MEMORY_VAULT` env var; (3) walk up from cwd looking for `.agent-memory/` directory; (4) return error. Return absolute path. (Design decision #4)
**Verify**: `go test ./internal/vault/ -run Discover`
**Done**: All four resolution paths tested. Error messages are clear.
</task>

<task>
**Name**: Task 2.2 — Write tests
**Files**: `internal/vault/discover_test.go`
**Action**: Ginkgo/gomega BDD tests. Contexts: explicit path exists, explicit path missing, env var set, env var points to nonexistent dir, walk-up finds vault in parent, walk-up finds vault in grandparent, no vault found, cwd inside vault dir itself.
**Verify**: `go test ./internal/vault/ -run Discover -v`
**Done**: All resolution paths and error cases covered.
</task>

</tasks>

<checkpoint>
**Slice 2 Checkpoint**:
- [ ] Discover finds vault via all three mechanisms
- [ ] Discover returns clear error when no vault found
- [ ] `go test ./internal/vault/ -run Discover` passes
</checkpoint>

</slice>

<slice name="slice-3-similarity-wikilinks">

## Slice 3: Similarity Check + Wikilink Extraction

**Goal**: Pure functions for Jaccard similarity on normalized tokens and wikilink extraction from markdown.

<tasks>

<task>
**Name**: Task 3.1 — Implement NormalizeTokens and Jaccard
**Files**: `internal/note/similarity.go`
**Action**: Implement `NormalizeTokens(text string) []string` — lowercase, split on whitespace, strip punctuation, remove stop words (a, an, the, is, are, was, were, in, on, of, to, for, and, or, but, with, by, at, from). Implement `Jaccard(a, b []string) float64` — |intersection|/|union|, return 0.0 for two empty sets. Survey Go libraries for tokenization; implement directly if nothing fits (small surface). (Design decision #6)
**Verify**: `go test ./internal/note/ -run "NormalizeTokens|Jaccard"`
**Done**: Jaccard(identical) = 1.0, Jaccard(disjoint) = 0.0, stop words removed, punctuation stripped.
</task>

<task>
**Name**: Task 3.2 — Implement ExtractWikilinks
**Files**: `internal/note/wikilinks.go`
**Action**: Implement `ExtractWikilinks(text string) []string`. Regex `\[\[([^\]]+)\]\]`. Handle `[[slug|alias]]` by extracting slug only (split on `|`, take first). Deduplicate results. (Design decision #7)
**Verify**: `go test ./internal/note/ -run ExtractWikilinks`
**Done**: Extracts `[[slug]]`, `[[slug|alias]]`, multiple links per line, deduplicates.
</task>

<task>
**Name**: Task 3.3 — Write tests
**Files**: `internal/note/similarity_test.go`, `internal/note/wikilinks_test.go`
**Action**: Ginkgo/gomega BDD tests. Similarity: identical titles, disjoint, partial overlap, stop-word-heavy titles, empty input. Wikilinks: single link, multiple links, alias form, no links, links in code blocks (extract anyway — code-block awareness deferred to lint-vault).
**Verify**: `go test ./internal/note/ -run "Jaccard|NormalizeTokens|ExtractWikilinks" -v`
**Done**: All edge cases covered.
</task>

</tasks>

<checkpoint>
**Slice 3 Checkpoint**:
- [ ] Jaccard produces correct similarity scores
- [ ] Wikilinks extracted correctly including alias form
- [ ] `go test ./internal/note/ -run "Jaccard|NormalizeTokens|ExtractWikilinks"` passes
</checkpoint>

</slice>

<slice name="slice-4-harness-nf001">

## Slice 4: Harness Detection + NF001 Relaxation

**Goal**: Auto-detect source-agent from harness env vars. Make source-agent optional in lint.

<tasks>

<task>
**Name**: Task 4.1 — Implement DetectSourceAgent
**Files**: `internal/note/source_agent.go`
**Action**: Implement `DetectSourceAgent() string`. Check env vars in order: `OPENCODE_RUN_ID` → `opencode:{value}`. Other harnesses (Claude Code, Copilot, VS Code) — check for known env vars, format as `{harness}:{session-id}`. Fallback: empty string. (Design decision #9)
**Verify**: `go test ./internal/note/ -run DetectSourceAgent`
**Done**: Returns `opencode:<id>` when `OPENCODE_RUN_ID` set. Returns empty when no harness detected.
</task>

<task>
**Name**: Task 4.2 — Relax NF001 for source-agent
**Files**: `internal/note/lint.go`
**Action**: Modify `ruleNF001.Check` to skip the `source-agent` empty check. The field remains in frontmatter but is no longer required to be non-empty. (Design decision #14)
**Verify**: `go test ./internal/note/ -run Lint`
**Done**: Existing lint tests pass. A note with empty `source-agent` no longer triggers NF001.
</task>

<task>
**Name**: Task 4.3 — Write tests
**Files**: `internal/note/source_agent_test.go`
**Action**: Ginkgo/gomega BDD tests. Contexts: OPENCODE_RUN_ID set, no env vars set, AGENT_MEMORY_SOURCE_AGENT override. Use `t.Setenv` for env var manipulation.
**Verify**: `go test ./internal/note/ -run DetectSourceAgent -v`
**Done**: All detection paths tested.
</task>

</tasks>

<checkpoint>
**Slice 4 Checkpoint**:
- [ ] DetectSourceAgent returns correct identifiers
- [ ] NF001 no longer requires source-agent
- [ ] `go test ./internal/note/` passes (no regressions)
</checkpoint>

</slice>

<slice name="slice-5-write-pipeline">

## Slice 5: Write Pipeline

**Goal**: `Write()` integrates all prior slices into a complete write-to-inbox pipeline with I/O.

<tasks>

<task>
**Name**: Task 5.1 — Implement Write function
**Files**: `internal/note/write.go`
**Action**: Implement `Write(opts WriteOptions) (*WriteResult, error)`. Steps:
1. Assemble `Note` from `WriteOptions` — populate `Frontmatter` fields from opts
2. Auto-populate: `created` (today if empty), `updated` (always today), `status` (always `inbox`), `confidence` (default `medium`), `review-by` (TTL table from design decision #9), `requires-human-review` (true for constraint/decision), `source-agent` (from opts, or `DetectSourceAgent()`)
3. Run `Lint()` — if invalid, return refused with errors
4. Scan `_inbox/` and `notes/` for existing notes, parse titles, run Jaccard against incoming title — if similarity ≥ 0.7 and not `Force`, return refused with candidates
5. Extract wikilinks from body, check resolution against `notes/{slug}.md` and `_inbox/*-{slug}.md`, collect warnings
6. Generate slug from title, construct filename `_inbox/{YYYY-MM-DD}-{slug}.md`, handle collision with numeric suffix (design decision #13)
7. Serialize note, write file
8. Append log entry to `_meta/log.md` in RFC3339 format (design decision #11)
9. Return written result with path and warnings
**Verify**: `go test ./internal/note/ -run Write`
**Done**: All paths exercised (see task 5.2).
</task>

<task>
**Name**: Task 5.2 — Write tests
**Files**: `internal/note/write_test.go`
**Action**: Ginkgo/gomega BDD tests using temp directories with vault structure (create `_inbox/`, `notes/`, `_meta/log.md`). Contexts:
- Happy path: valid opts → note in `_inbox/`, log entry appended
- Lint failure: missing title → refused with errors
- Similarity hit: existing note with similar title → refused with candidates
- Force override: similar note + Force=true → written
- Wikilink warning: body has `[[nonexistent]]` → written with warning
- Wikilink resolved: body has `[[existing]]`, `notes/existing.md` exists → no warning
- File collision: pre-create `_inbox/{date}-{slug}.md` → suffix `-2` used
- Auto-populated fields: verify created, updated, status, review-by, requires-human-review values
- TTL table: observation=90d, pattern=180d, assumption=30d, constraint=365d, decision=365d
- Source-agent: opts.SourceAgent used when set, DetectSourceAgent fallback when empty
- Tags: opts.Tags written to frontmatter
**Verify**: `go test ./internal/note/ -run Write -v`
**Done**: All contexts pass. No regressions in existing note tests.
</task>

</tasks>

<checkpoint>
**Slice 5 Checkpoint**:
- [ ] Write produces valid notes in `_inbox/`
- [ ] Lint gating works (refuses invalid notes)
- [ ] Similarity check refuses duplicates, force overrides
- [ ] Wikilink warnings reported correctly
- [ ] Log entry appended
- [ ] `go test ./internal/note/` passes (full package, no regressions)
</checkpoint>

</slice>

<slice name="slice-6-cli-integration">

## Slice 6: CLI Command + Integration Tests

**Goal**: Wire `write-note` subcommand into the binary with dual output and integration tests.

<tasks>

<task>
**Name**: Task 6.1 — Implement CLI command
**Files**: `internal/cli/write_note.go`
**Action**: Implement `newWriteNoteCmd()`. Positional arg: `<body-file>` (`-` for stdin). Flags: `--vault`, `--type` (required), `--title` (required), `--project`, `--domain` (string slice), `--scope`, `--source-artifact`, `--source-agent`, `--confidence`, `--tags` (string slice), `--force`. Read body from file or stdin. Call `vault.Discover()` for vault resolution. Call `note.Write()`. Format output: JSON mode → marshal `WriteResult`; human mode → `✓ Written: path` or `✗ Refused: reason` with details. Follow `lint_note.go` patterns for error handling and `SilenceUsage`/`SilenceErrors`. (Design decision #2, #10)
**Verify**: `go build ./cmd/agent-memory/ && ./agent-memory write-note --help`
**Done**: Help text shows all flags. Command compiles.
</task>

<task>
**Name**: Task 6.2 — Register subcommand
**Files**: `internal/cli/root.go`
**Action**: Add `newWriteNoteCmd()` to root command's `AddCommand` call.
**Verify**: `go build ./cmd/agent-memory/ && ./agent-memory --help`
**Done**: `write-note` appears in subcommand list.
</task>

<task>
**Name**: Task 6.3 — Update instructions text
**Files**: `internal/cli/instructions.go`
**Action**: Remove "Writing to the vault is not yet enabled" text. Add `write-note` usage to the instructions output.
**Verify**: `go build ./cmd/agent-memory/ && ./agent-memory instructions`
**Done**: Instructions mention `write-note`. No "not yet enabled" text.
</task>

<task>
**Name**: Task 6.4 — Integration tests
**Files**: `test/integration/write_note_test.go`
**Action**: Ginkgo/gomega integration tests using `testutil.BuildBinary`/`RunBinary`. Set up temp vault via `agent-memory init`. Contexts:
- Happy path: write body file with required flags → exit 0, note in `_inbox/`
- JSON mode: `--json` → valid JSON output matching `WriteResult`
- Stdin: body-file `-`, pipe body via stdin → written
- Similarity refusal: create existing note, write similar → exit 1, candidates in output
- Force: similar + `--force` → exit 0
- Missing required flags: no `--type` → exit 1, usage shown
- Missing vault: no vault in tree, no env var → exit 1, clear error
- Vault via `--vault` flag: explicit path → works
- Vault via env var: `AGENT_MEMORY_VAULT` set → works
**Verify**: `go test ./test/integration/ -run WriteNote -v`
**Done**: All integration tests pass.
</task>

</tasks>

<checkpoint>
**Slice 6 Checkpoint**:
- [ ] `agent-memory write-note` works end-to-end from CLI
- [ ] JSON and human-readable output correct
- [ ] Stdin input works
- [ ] Vault discovery works via flag, env var, and walk-up
- [ ] `go test ./test/integration/ -run WriteNote` passes
- [ ] `go test ./...` passes (full suite, no regressions)
</checkpoint>

</slice>

<slice name="slice-7-protocol-v3">

## Slice 7: Writing Protocol v3

**Goal**: Fix enum bug in writing-protocol.md template and update to v3 with write-note instructions.

<tasks>

<task>
**Name**: Task 7.1 — Fix enum values
**Files**: `internal/vault/templates/writing-protocol.md`
**Action**: Replace wrong enum values:
- `status`: `draft, active, archived, deprecated` → `inbox, verified, deprecated, contested, superseded`
- `epistemic-type`: `observation, inference, synthesis, hypothesis, procedure` → `observation, pattern, constraint, decision, assumption, synthesis`
- `scope`: `project, global` → `project, cross-project`
(Design decision #12)
**Verify**: `go test ./...`
**Done**: Template contains correct enum values.
</task>

<task>
**Name**: Task 7.2 — Add v3 write-note instructions
**Files**: `internal/vault/templates/writing-protocol.md`
**Action**: Add section documenting `agent-memory write-note` usage: CLI syntax, required/optional flags, body file format, what the tool auto-populates, similarity check behavior, expected output.
**Verify**: Manual review of template content.
**Done**: Protocol v3 documents the write-note workflow.
</task>

<task>
**Name**: Task 7.3 — Update tests if needed
**Files**: `internal/vault/init_test.go`, `test/integration/init_test.go`
**Action**: Check if any tests assert on writing-protocol.md content. Update assertions to match v3 content.
**Verify**: `go test ./...`
**Done**: All tests pass with updated template.
</task>

</tasks>

<checkpoint>
**Slice 7 Checkpoint**:
- [ ] Template has correct enum values
- [ ] v3 content documents write-note usage
- [ ] `go test ./...` passes (full suite)
</checkpoint>

</slice>

</slices>

<verification>

## Final Verification

Before declaring implementation complete:
- [ ] All 7 slice checkpoints passed
- [ ] `go test ./...` passes with no failures
- [ ] `go vet ./...` clean
- [ ] `agent-memory write-note` works end-to-end from a fresh vault
- [ ] JSON output is valid and matches documented shape
- [ ] Human-readable output is clear and useful

</verification>

<success_criteria>

## Success Criteria

- [ ] `agent-memory write-note <body-file> --type observation --title "test"` writes a note to `_inbox/`
- [ ] Auto-populated fields are correct (created, updated, status, review-by, etc.)
- [ ] Similarity check refuses duplicates; `--force` overrides
- [ ] Wikilink warnings reported but don't block writes
- [ ] Vault discovery works via `--vault`, env var, and walk-up
- [ ] `--json` produces valid JSON matching `WriteResult` shape
- [ ] Writing protocol v3 has correct enum values and documents write-note
- [ ] NF001 no longer requires source-agent
- [ ] All slice checkpoints passed
- [ ] No regressions in existing tests

</success_criteria>

</plan_artifact>
