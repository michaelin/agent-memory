<design_artifact feature="note-search">

<current_state>

## Current State

- **Note type + Parse()**: `internal/note/note.go` — `Note{Frontmatter, Body string}`. All frontmatter fields exported as plain strings/slices. `Parse([]byte) (*Note, error)` splits on `---` delimiters, returns body as trimmed string. Unknown YAML fields silently ignored; malformed YAML returns error.

- **File scanning**: `findSimilarNotes()` in `write.go` is the only code that walks vault directories. Uses `os.ReadDir()` → filter `.md` → skip symlinks/dirs → `ReadFile()` → `Parse()`. Scans both `_inbox/` and `notes/`. Silently skips parse failures. Not extracted into a reusable function.

- **Similarity**: `NormalizeTokens()` + `Jaccard()` in `similarity.go`. Title-only, threshold-based (≥ 0.7). Not a general search mechanism.

- **Vault discovery**: `vault.Discover(explicitPath string) (string, error)` returns absolute path. Used by `write-note` via `--vault` flag.

- **CLI pattern**: Subcommand factories in `internal/cli/`. `--json` persistent flag on root, checked as package-level `jsonOutput` bool. Each command does its own `json.Marshal()` + `fmt.Println()`. Human output uses `fmt.Printf()` with ✓/✗ symbols.

- **Tag taxonomy**: Placeholder file. No format, no parser, no aliases.

- **Vault state**: `notes/` is always empty (promotion not implemented). All written notes are in `_inbox/` with `status: inbox`.

</current_state>

<desired_state>

## Desired End State

- `agent-memory search <query>` subcommand exists with two phases
- Phase 1: scan vault notes, filter by frontmatter fields, match query against titles, return frontmatter-only results
- Phase 2: given a list of slugs, return full note content (frontmatter + body), capped at 10
- Both JSON and human-readable output modes
- Vault discovery via `--vault` flag (same pattern as write-note)

</desired_state>

<design_decisions>

## Design Decisions

<decision id="1">
**Decision**: Extract a shared note-scanning function from `findSimilarNotes()` into a reusable utility.

**Rationale**: Both `findSimilarNotes()` and the new search need the same walk: ReadDir → filter .md → ReadFile → Parse. Duplicating this is wasteful and error-prone. The extracted function would return `[]Note` (or a struct with path + parsed note) for a given list of directories.

**Alternatives considered**: Leave scanning inline in each caller. Rejected because search adds a third caller (Phase 1 scan, Phase 2 slug lookup) on top of the existing similarity scan.

**Assumption**: The extracted function lives in `internal/note/` since it depends on `Parse()`.
</decision>

<decision id="2">
**Decision**: Query matching uses case-insensitive substring match on the note title for Phase 1.

**Rationale**: The design doc says "grep + YAML parsing." Substring on title is the simplest useful search. Body search is expensive (requires reading full files) and the design doc's two-phase model implies Phase 1 is lightweight — frontmatter only. Jaccard is wrong here because it's threshold-based similarity, not keyword search.

**Alternatives considered**:
- Token-based matching (all query words must appear in title). More precise but harder to use — agents would need exact wording.
- Body search in Phase 1. Rejected — contradicts the "frontmatter discovery" description and makes Phase 1 as expensive as Phase 2.
- Regex matching. Overkill for agent use.

**Uncertainty**: Should the query also match against `domain`, `project`, `tags` fields, or just title? I'm leaning title-only for the query, with separate `--project`, `--domain`, `--type` flags for field filtering.
</decision>

<decision id="3">
**Decision**: Search scans both `notes/` and `_inbox/` by default, with a `--status` flag to filter.

**Rationale**: The design doc says "status: verified only" but `notes/` is empty until increment 5 (promotion). If we only search `notes/`, the command is useless until then. Searching both directories and defaulting to no status filter makes the command immediately useful. Agents that want only verified notes can pass `--status=verified`.

**Alternatives considered**:
- Default to `--status=verified` per the design doc. Rejected — returns nothing until promotion exists. We can change the default later.
- Always search both, no status filter. Simpler but loses the design doc's intent.

**Uncertainty**: This diverges from the design doc. The design doc's "verified only" default makes sense once promotion exists. We could flip the default in increment 5.
</decision>

<decision id="4">
**Decision**: Phase 2 slug matching is exact match on the slug derived from the note's title (via `Slug()`), not filename-based.

**Rationale**: Inbox files are `{date}-{slug}.md`, notes files would be `{slug}.md`. Matching on filename requires stripping the date prefix for inbox files. Matching on `Slug(note.Frontmatter.Title)` is consistent regardless of which directory the note lives in.

**Alternatives considered**:
- Filename-stem matching. Fragile because inbox and notes use different naming conventions.
- Return a unique ID in Phase 1 results that Phase 2 uses. More robust but adds complexity. The slug derived from title is already unique enough (collision handling in write adds suffixes, but the title-derived slug is the canonical identifier).

**Uncertainty**: What if two notes have the same title-derived slug but different date prefixes in `_inbox/`? `Slug()` is deterministic for a given title, so same title = same slug. The collision suffix in the filename (`-2`, `-3`) is not reflected in the slug. Phase 2 would need to return all notes matching the slug, or we need a different identifier.

<!-- REVIEW: This is a real problem. The slug is not unique in _inbox/ because collision suffixes exist. Should Phase 1 return the file path as the identifier instead of the slug? Or should Phase 2 accept paths? -->
</decision>

<decision id="5">
**Decision**: Phase 2 cap at 10 slugs — hard error if more than 10 provided.

**Rationale**: The design doc says "capped at 10." A hard error is clearer than silent truncation. The agent gets an explicit message and can reduce its request. Warnings-with-truncation risk the agent not noticing it got partial results.

**Alternatives considered**:
- Silent truncation (return first 10). Agent might not realize it's missing results.
- Warning + truncation. Better than silent but still returns partial results without the agent explicitly opting in.
</decision>

<decision id="6">
**Decision**: Defer tag alias expansion to a future increment.

**Rationale**: The tag-taxonomy.md template is a placeholder with no structured format. No alias syntax is defined. No parser exists. Implementing alias expansion requires first defining the taxonomy format, which is a design decision that affects increment 9 (Librarian) and increment 7 (reindex). Doing it here would be premature.

**Alternatives considered**:
- Define a simple alias format now (e.g., `auth: [authentication, authn]`) and implement expansion. Risks locking in a format before the Librarian increment clarifies requirements.
- Skip tag expansion but add a `--tag` filter flag that does exact matching. This is useful without aliases.

**Assumption**: We add `--tag` as an exact-match filter flag. Alias expansion comes later.
</decision>

<decision id="7">
**Decision**: Malformed notes are silently skipped during search, matching the existing `findSimilarNotes()` pattern.

**Rationale**: Consistency with existing behavior. A search should not fail because one note in the vault has bad YAML. The user can run `agent-memory lint-note` to find malformed files.

**Alternatives considered**:
- Emit warnings for skipped files. Adds noise to search results. Could be a `--verbose` flag but that's scope creep.
</decision>

<decision id="8">
**Decision**: Search logic lives in `internal/note/` as a `Search()` function, with the CLI in `internal/cli/search.go`.

**Rationale**: Follows the write-note pattern: business logic in `internal/note/write.go`, CLI wiring in `internal/cli/write_note.go`. The search function takes a vault path + options and returns results. The CLI handles flag parsing, vault discovery, and output formatting.

**Alternatives considered**: Put search logic in `internal/vault/`. Rejected — `internal/note/` already has the scanning pattern and depends on `Parse()`.
</decision>

<decision id="9">
**Decision**: Phase 1 and Phase 2 are separate modes of the same subcommand, selected by `--phase` flag (default: 1).

**Rationale**: The design doc shows `agent-memory search <query> [--phase=1|2] [--slugs=<a,b>]`. Phase 2 requires `--slugs` and ignores the query. Phase 1 requires a query and ignores `--slugs`. These are mutually exclusive input sets controlled by the `--phase` flag.

**Alternatives considered**:
- Two separate subcommands (`search-discover`, `search-read`). Adds CLI surface area for no benefit.
- Auto-detect phase from flags (if `--slugs` present → Phase 2, else Phase 1). Implicit behavior is harder to document and debug.
</decision>

</design_decisions>

<data_flow>

## Data Flow

### Phase 1 (Discovery)
```
CLI args (query, filters) → vault.Discover() → scan notes/ + _inbox/ → Parse() each .md → filter by status/project/domain/type → substring match query on title → return []SearchResult{slug, title, frontmatter}
```

### Phase 2 (Body Read)
```
CLI args (slugs) → vault.Discover() → scan notes/ + _inbox/ → Parse() each .md → match slug → cap at 10 → return []NoteResult{slug, frontmatter, body}
```

### Output
```
SearchResult/NoteResult → json.Marshal (--json) or fmt.Printf (human)
```

</data_flow>

<api_surface>

## CLI Surface

```
agent-memory search <query> [flags]

Flags:
  --phase int        Search phase: 1 (discovery) or 2 (body read) (default 1)
  --slugs strings    Slugs to read (Phase 2 only, comma-separated)
  --status string    Filter by status (default: all)
  --type string      Filter by epistemic-type
  --project string   Filter by project
  --domain strings   Filter by domain (comma-separated)
  --tag strings      Filter by tag (exact match, comma-separated)
  --vault string     Path to vault (overrides auto-discovery)
  --json             Output JSON (inherited from root)
```

### Phase 1 JSON output
```json
{
  "results": [
    {
      "slug": "auth-tokens-rotate-90d",
      "path": "_inbox/2026-05-21-auth-tokens-rotate-90d.md",
      "title": "Auth tokens should rotate every 90 days",
      "status": "inbox",
      "epistemic_type": "constraint",
      "confidence": "high",
      "project": "my-project",
      "domain": ["auth", "security"],
      "tags": [],
      "created": "2026-05-21",
      "updated": "2026-05-21"
    }
  ],
  "count": 1
}
```

### Phase 2 JSON output
```json
{
  "notes": [
    {
      "slug": "auth-tokens-rotate-90d",
      "path": "_inbox/2026-05-21-auth-tokens-rotate-90d.md",
      "frontmatter": { ... },
      "body": "# Auth tokens should rotate every 90 days\n\n## Evidence\n..."
    }
  ],
  "count": 1
}
```

</api_surface>

<open_questions>

## Open Questions

1. **Slug uniqueness problem**: In `_inbox/`, collision suffixes (`-2`, `-3`) make filenames unique but the title-derived slug is the same for all collisions. Phase 2 slug lookup could match multiple files. Should we use file path as the identifier instead? Or return all matches?

2. **Default status filter**: The design doc says "verified only" but that returns nothing until promotion exists. Should we default to "all" now and flip to "verified" in increment 5? Or default to "verified" and accept empty results?

3. **Query as positional arg**: The design doc shows `<query>` as positional. In Phase 2, the query is irrelevant. Should Phase 2 require no positional arg, or accept and ignore it?

4. **Empty query**: Should `agent-memory search` with no query and no filters return all notes? Or require at least one filter criterion?

</open_questions>

</design_artifact>
