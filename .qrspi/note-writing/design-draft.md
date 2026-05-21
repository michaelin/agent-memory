<design_artifact feature="note-writing" status="draft-v2">

<current_state>

## Current State

- **Vault scaffolding**: `internal/vault/` has `Init()`, `VaultStructure()`, embedded templates. Vault creates `_meta/`, `_inbox/`, `_contested/`, `notes/` directories and seeds `_meta/*.md` files.
- **Note parsing**: `internal/note/` has `Parse()`, `Lint()`, `Rules()`, 7 lint rules (NF001–NF007). Parse takes `[]byte`, returns `*Note`. Lint takes `*Note`, returns `*LintResult`. Both are pure functions — no I/O.
- **CLI**: `agent-memory` binary with `init`, `instructions`, `lint-note` subcommands. `--json` persistent flag. Dual output pattern established.
- **No note serialization**: `Parse()` exists but no function to produce a complete note file from a `Note` struct.
- **No vault discovery**: Vault path is always explicit. No walk-up or env var resolution.
- **No slug generation**: No utility to convert titles to filesystem-safe slugs.
- **No similarity check**: No Jaccard or text comparison utilities.
- **No wikilink parsing**: No `[[slug]]` extraction.
- **Writing protocol v2 bug**: Template has wrong enum values (draft/active/archived instead of inbox/verified/deprecated/contested/superseded). Needs fixing.
- **Log format**: Init uses `## [RFC3339] init | vault initialized`. Design doc specifies write format as `## [YYYY-MM-DD HH:MM] write | <type> | <slug> | by:<agent>`.

</current_state>

<desired_state>

## Desired End State

- **`agent-memory write-note` subcommand** that accepts body content (file path or stdin) and frontmatter fields as CLI flags, assembles a complete note, validates it, checks for duplicates, and writes it to `_inbox/{YYYY-MM-DD}-{slug}.md`.
- **Note serialization** function that produces a complete note file from a `Note` struct (frontmatter + body).
- **Vault discovery** function to resolve vault path from: explicit argument → env var → walk-up → error.
- **Slug generation** from title using markdown anchor-style slugification.
- **Similarity check** using Jaccard on normalized word tokens against existing notes in `_inbox/` and `notes/`. Use an existing library if a good fit exists; otherwise implement (small surface area).
- **Wikilink extraction** from note body, with warnings for unresolved links.
- **Frontmatter assembly**: tool assembles all frontmatter from CLI flags and defaults. Agent never writes raw YAML.
- **Log entry** appended to `_meta/log.md` on successful write.
- **Writing protocol v2 fix**: correct enum values in template.
- **Writing protocol v3**: updated with `write-note` usage instructions.

</desired_state>

<design_decisions>

## Design Decisions

<decision id="1">
**Decision**: Scope this increment to the deterministic write tool only. No `--new-claim`/`--update`/`--contest` flags. No correction protocol. No contest protocol.

**Rationale**: The design doc §9 describes a full write protocol with three modes (new-claim, update, contest). That's a lot of surface area. For increment 3, we implement only the "new note" path: validate, similarity-check, and write to inbox. The `--update` and `--contest` modes can be added in a future increment.

This means: no `_contested/` directory manipulation, no staged correction notes, no `update-type`/`targets` field population. The tool writes new notes to `_inbox/` only.
</decision>

<decision id="2">
**Decision**: `write-note` takes the body as a file argument (or stdin) and frontmatter fields as CLI flags. The tool assembles the complete note.

**Rationale**: The static-tooling principle (§5.0) says "tool owns form, agent owns content." Having the agent compose full frontmatter YAML is error-prone and violates this principle. The agent provides content (body text, evidence) and content-judgment fields (epistemic-type, title, domain, project, scope, source-artifact, confidence) as flags. The tool assembles the frontmatter, fills defaults, and serializes.

CLI interface:
```
agent-memory write-note <body-file> \
  --vault <path> \
  --type <epistemic-type> \
  --title "Concise claim" \
  [--project <name>] \
  [--domain <a,b>] \
  [--scope project|cross-project] \
  [--source-artifact <path-or-url>] \
  [--source-agent <identifier>] \
  [--confidence low|medium|high] \
  [--tags <a,b>] \
  [--force]
```

If `<body-file>` is `-`, read from stdin. This supports both file-based and piped workflows.
</decision>

<decision id="3">
**Decision**: Place write logic in `internal/note/` package, not `internal/vault/`.

**Rationale**: The write operation is note-centric: serialize a note, check similarity against other notes, validate. The vault is just the destination. `internal/vault/` handles vault structure; `internal/note/` handles note operations. Add `Serialize()`, `Slug()`, `Jaccard()` to `internal/note/`.

Vault discovery goes in `internal/vault/` as `Discover(explicitPath string) (string, error)` since it's vault-centric.

The CLI command in `internal/cli/write_note.go` bridges the two: discovers vault, assembles note from flags + body, calls note operations, writes to vault.
</decision>

<decision id="4">
**Decision**: Vault discovery follows the design doc §5.1 priority: explicit `--vault` flag → `AGENT_MEMORY_VAULT` env var → walk up directories looking for `.agent-memory/` → error (no global fallback yet).

**Rationale**: The global fallback (`~/.local/share/agent-memory`) is deferred to increment 13. For now, the tool needs to find the vault. The walk-up pattern is the most common case (agent runs from within a project). The `--vault` flag and env var provide explicit overrides.

Function: `Discover(explicitPath string) (string, error)` in `internal/vault/`.
</decision>

<decision id="5">
**Decision**: Slug generation uses markdown heading anchor-style slugification: lowercase, replace spaces with hyphens, strip non-alphanumeric (except hyphens), collapse consecutive hyphens, trim leading/trailing hyphens, truncate to 60 characters.

**Rationale**: This is a well-known format (GitHub, Goldmark, etc.). Look for an existing Go library during implementation; if nothing fits, the logic is small and well-specified.

Function: `Slug(title string) string` in `internal/note/`.
</decision>

<decision id="6">
**Decision**: Similarity check uses Jaccard index on normalized word tokens. Threshold: 0.7 (TBD — may adjust during testing). Comparison scope: titles of all `.md` files in `_inbox/` and `notes/`.

**Rationale**: Design doc §9 specifies Jaccard on normalized word tokens. Normalization: lowercase, split on whitespace, remove punctuation, remove stop words. Jaccard = |intersection| / |union| of token sets.

The check scans titles only (from frontmatter), not full body content. This keeps it fast and focused.

If a similar note is found and no `--force` flag is passed, the tool refuses the write and returns the candidates as JSON.

Survey existing Go libraries for tokenization/similarity during implementation. The Jaccard calculation itself is ~10 lines, but normalization benefits from a proven implementation.

Functions: `NormalizeTokens(text string) []string`, `Jaccard(a, b []string) float64` in `internal/note/`.
</decision>

<decision id="7">
**Decision**: Wikilink extraction uses a simple regex `\[\[([^\]]+)\]\]` to find all `[[slug]]` references in the note body. Unresolved links produce warnings but do not block the write.

**Rationale**: The agent has context from `memory-search` results and knows which notes exist and are related. The `## Related` section with `[[links]]` is part of the note format spec (§5.3). The agent decides *which* notes are related (content judgment); the tool validates that the links resolve (form). Unresolved links may be intentional (referencing a note the agent plans to write next) — promotion is the hard gate, not the write.

**On Related links as arguments vs. body content**: The `## Related` section includes descriptions (`- [[slug]] — relationship description`) which are content judgment. Making this a flag would require complex structured input. Instead, the agent writes `## Related` as part of the body. The tool doesn't own this section's form.

**On inline links vs. Related section**: Both are valid. Inline `[[links]]` in prose reference notes contextually. The `## Related` section declares explicit relationships. They serve different purposes and coexist naturally (standard Obsidian practice). The tool validates *all* `[[links]]` in the body regardless of location.

Function: `ExtractWikilinks(body string) []string` in `internal/note/`.
</decision>

<decision id="8">
**Decision**: Note serialization produces `---\n{yaml}\n---\n{body}` format.

**Rationale**: Standard frontmatter format. Use `yaml.Marshal()` for the frontmatter block. The body is appended as-is after the closing delimiter.

Function: `Serialize(note *Note) ([]byte, error)` in `internal/note/`.
</decision>

<decision id="9">
**Decision**: Auto-populated fields and their defaults:
- `created`: today's date (YYYY-MM-DD) if empty
- `updated`: always set to today's date on write
- `status`: always set to `inbox`
- `confidence`: from `--confidence` flag; defaults to `medium` if not provided
- `review-by`: calculated from TTL table (observation: 90d, pattern: 180d, assumption: 30d, constraint: 365d, decision: 365d, per design doc §5.6)
- `requires-human-review`: `true` if epistemic-type is `constraint` or `decision`. Kept as a frontmatter field (not purely tooling-driven) because it serves as a visible signal when humans browse in Obsidian, and can be manually set by humans on any note type.
- `source-agent`: Auto-detected from harness environment, optional. Detection order:
  1. `OPENCODE_RUN_ID` → `opencode:<run-id>`
  2. `CLAUDE_CODE_SESSION_ID` (or equivalent) → `claude-code:<session-id>`
  3. `COPILOT_SESSION_ID` (or equivalent) → `copilot:<session-id>`
  4. `VSCODE_SESSION_ID` (or equivalent) → `vscode:<session-id>`
  5. `AGENT_MEMORY_SOURCE_AGENT` env var → value as-is (manual override)
  6. `--source-agent` flag → value as-is (per-invocation override)
  7. Fallback: empty string (no error)
  Flag and env var override auto-detection. The identifier is session-scoped, not role-scoped — "which session produced this note" is more useful than "which agent type."
- `verified-by`, `verified-date`: cleared (empty string). Populated by `memory-promote` (increment 5). Will use the same auto-detection pattern as `source-agent`.
- `tags`: optional `--tags` flag accepts agent-suggested tags. Written to frontmatter as-is. The Librarian's maintenance pass can add, remove, or correct them later. This is strictly better than starting from zero — agents often know the relevant domain.

**Rationale**: Design doc §9 step 2 and §5.6 specify these. TTL values match the design doc table exactly.
</decision>

<decision id="10">
**Decision**: Write result JSON shape:
```json
{"status": "written", "path": "_inbox/2026-05-21-slug.md", "warnings": ["unresolved link: [[foo]]"]}
{"status": "refused", "reason": "similar note exists", "candidates": [{"path": "notes/existing.md", "title": "...", "similarity": 0.85}]}
{"status": "refused", "reason": "validation failed", "errors": [{"rule": "NF001", "message": "..."}]}
```

**Rationale**: Three possible outcomes: written (success with optional warnings), refused due to similarity, refused due to validation failure. Human-readable output follows the same pattern with ✓/✗ formatting.
</decision>

<decision id="11">
**Decision**: Log entry format: `## [YYYY-MM-DDTHH:MM:SS±HH:MM] write | <epistemic-type> | <slug> | by:<source-agent>`

**Rationale**: Design doc says `## [YYYY-MM-DD HH:MM] write | <type> | <slug> | by:<agent>`. Init already uses RFC3339. Using RFC3339 for consistency with init. Minor deviation from design doc — the log is append-only and human-readable either way. Note: Go's `time.RFC3339` produces `Z` for UTC or `±HH:MM` for other timezones; `time.Parse` accepts both forms, so no special handling needed.
</decision>

<decision id="12">
**Decision**: Fix writing protocol v2 enum values as part of this increment. Update to v3 with write-note instructions.

**Rationale**: The wrong enum values in writing-protocol.md are a bug from increment 2. Fix them while updating to v3. The v3 content adds write-note usage instructions per the roadmap.
</decision>

<decision id="13">
**Decision**: File collision handling: if `_inbox/{YYYY-MM-DD}-{slug}.md` already exists, append a numeric suffix: `-2`, `-3`, etc. No `--force` overwrite.

**Rationale**: Collision differs from similarity: similarity checks title words (fuzzy), collision checks slugified filenames (exact). Two notes with different titles can produce the same slug (e.g. "Go Error Handling" vs "Go: Error Handling" → both `go-error-handling`). Similarity would likely catch these too, but not guaranteed. The collision suffix is a safety net for this edge case — should rarely trigger in practice.
</decision>

</design_decisions>

<data_flow>

## Data Flow

```
CLI flags + body file → Assemble Note struct (frontmatter from flags/defaults, body from file)
  → Auto-populate fields (created, updated, status, confidence, review-by, requires-human-review, source-agent)
  → Lint (validate assembled note)
  → If lint fails → refuse with validation errors
  → Similarity check (scan _inbox/ and notes/ titles, Jaccard comparison)
  → If similar found and no --force → refuse with candidates
  → Extract wikilinks from body → check resolution → collect warnings
  → Generate slug from title
  → Serialize note (frontmatter + body)
  → Write to _inbox/{YYYY-MM-DD}-{slug}.md (with collision suffix if needed)
  → Append log entry to _meta/log.md
  → Return result (path + warnings)
```

CLI layer:
```
agent-memory write-note <body-file> \
  --vault <path> --type <type> --title "..." \
  [--project ...] [--domain ...] [--scope ...] \
  [--source-artifact ...] [--source-agent ...] \
  [--confidence ...] [--tags ...] [--force]
  → Discover vault path
  → Read body file (or stdin if "-")
  → Assemble Note from flags + body
  → Call note write pipeline
  → Format output (human-readable or JSON)
  → Exit 0 on success, 1 on refusal/error
```

</data_flow>

</design_artifact>
