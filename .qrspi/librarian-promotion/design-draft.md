<design_artifact feature="librarian-promotion">

<current_state>

## Current State

- **Frontmatter struct** (`internal/note/note.go`): All fields needed for promotion already exist — `Status` (string, not typed enum), `RequiresHumanReview` (bool), `EpistemicType` (string, currently named `UpdateType` in code — rename to `EpistemicType` / `epistemic-type` in this increment), `Targets` ([]string). No `SupersededBy` field exists.

- **Lint** (`internal/note/lint.go`): `Lint(*Note) *LintResult` is exported and callable programmatically. NF003 already validates status enum including `verified`, `deprecated`, `superseded`. Seven rules total (NF001-NF007).

- **Similarity** (`internal/note/write.go`): `findSimilarNotes()` scans both `_inbox/` and `notes/`, uses Jaccard similarity on title tokens with threshold 0.7. Private function — not exported.

- **Wikilink resolution** (`internal/note/write.go`): `resolveWikilinks()` checks both `notes/{slug}.md` and `_inbox/*-{slug}.md`. Private function, returns `[]string` of warning messages. Uses path-based lookup, not Obsidian-style filename-based resolution.

- **Log writing** (`internal/note/write.go`): `appendLog()` is private, hardcodes `write` action. Format: `## {RFC3339} write | {type} | {slug} | by:{agent}`.

- **Filename conventions**: Inbox files are `{date}-{slug}.md`. Notes files are `{slug}.md` (implied by wikilink resolution code, never explicitly produced).

- **CLI pattern**: One file per subcommand in `internal/cli/`. Each defines `newXxxCmd() *cobra.Command`. Vault discovery via per-command `--vault` flag + `vault.Discover()`. Dual output via package-level `jsonOutput` bool.

- **Instructions output**: Hardcoded string mentioning `lint-note` and `write-note`. No mention of promotion or Librarian.

- **Skill format**: YAML frontmatter (`name`, `description`) + markdown body. Skills inherit invoking agent's tool permissions. No explicit permission declarations.

- **Vault templates**: Currently embedded in Go binary (`internal/vault/templates/`). No mechanism for the vault to carry its own template files for agent/skill definitions.

</current_state>

<desired_state>

## Desired End State

- **`agent-memory promote`** subcommand exists and moves notes from `_inbox/` to `notes/`
- **`agent-memory deprecate`** subcommand exists and moves deprecated notes from `notes/` to `_deprecated/` with forward link
- Notes in `notes/` have `status: verified` and filename `{slug}.md`
- Constraint/decision notes require `--confirmed` flag
- Unresolved wikilinks block promotion (hard gate)
- Promotion and deprecation write log entries to `_meta/log.md`
- **Librarian is a subagent** that runs isolated from the calling agent's context. It is invoked by the user agent as the last step of its workflow. The Librarian reads the vault, processes inbox notes, and returns results — the user agent's context is not polluted with vault-scanning work.
- **Harness deployment**: For this increment, we target OpenCode (agent definition + skill). The architecture supports future expansion to VS Code Copilot, GitHub CLI, and other harnesses. The CLI tools are harness-agnostic; only the agent/skill definitions are harness-specific.
- **Vault carries deployment templates**: The vault's `_meta/templates/` directory contains template files for agent and skill definitions that `agent-memory instructions` can reference or emit. This makes deployment guidance vault-local and versionable.
- Deprecated notes live in `_deprecated/` (not `notes/`), with wikilink resolution updated to check all three directories. This follows Obsidian's model where wikilinks resolve by filename regardless of folder.
- Instructions output tells agents about the Librarian workflow

</desired_state>

<design_decisions>

## Design Decisions

<decision id="1">
**Decision**: `promote` does validation as a safety net, not just a file move.

`promote` does: (1) find note by slug in `_inbox/`, (2) parse and lint check, (3) wikilink resolution check (hard gate — refuses on unresolved links), (4) `--confirmed` gate for constraint/decision, (5) move file from `_inbox/{date}-{slug}.md` to `notes/{slug}.md`, (6) update frontmatter (`status: verified`, `updated` date), (7) write log entry.

The Librarian should also check these things before calling promote, but promote is the last line of defense. The design doc explicitly says "refuses to promote any note with unresolved wiki-links."

**Alternatives considered**: Pure file-move with no validation. Rejected because the design doc mandates wikilink checking at promotion time.
</decision>

<decision id="2">
**Decision**: Export the private helpers as package-level functions.

- `appendLog()` → `AppendLog(vaultPath, action, epistemicType, slug, agent string)` — parameterize the action (current code hardcodes `"write"`; the new signature accepts any action string so `promote` and `deprecate` can log their own actions)
- `resolveWikilinks()` → `ResolveWikilinks(vaultPath, body string) []string` — exported with the same signature, but the implementation is rewritten per Decision 6 (Obsidian-style filename-based resolution). This is a single combined change: export + rewrite.
- `findSimilarNotes()` → `FindSimilarNotes(vaultPath string, incomingTokens []string) ([]SimilarNote, error)` — exported with the same signature. `SimilarNote` is already an exported type in `write.go`.

The existing `Write()` function is updated to call the newly-exported versions (passing `"write"` as the action to `AppendLog`).

**Rationale**: These functions already exist and work. Exporting them makes them accessible to the promote subcommand (and future tools) without duplication.
</decision>

<decision id="3">
**Decision**: Promote renames the file from `{date}-{slug}.md` to `{slug}.md` when moving to `notes/`.

The slug is derived from the note's title via `Slug(frontmatter.Title)`, not by string-slicing the filename. This is more robust and handles edge cases like collision suffixes (`-2`).

**Rationale**: Consistent with existing wikilink resolution expectations (`notes/{slug}.md`).
</decision>

<decision id="4">
**Decision**: Add a `SupersededBy` field to the `Frontmatter` struct.

New field: `SupersededBy string \`yaml:"superseded-by"\``

This is machine-readable, queryable, and unambiguous. Used by the `deprecate` subcommand to record which note replaced the deprecated one.

**Alternatives considered**: Using existing `Targets` field (rejected — undefined semantics, list type for a 1:1 relationship). Body-only links in `## Related` (rejected — not machine-readable for future tooling).
</decision>

<decision id="5">
**Decision**: The `--confirmed` gate is a hard error when missing for constraint/decision notes.

When `promote` is called on a note with `requires-human-review: true` and `--confirmed` is not passed, the subcommand exits with error code 1 and a clear message. It does NOT silently skip or set a flag.

**Rationale**: The caller (Librarian subagent) should never call promote without --confirmed on a constraint/decision. A hard error surfaces bugs immediately.
</decision>

<decision id="6">
**Decision**: Wikilink resolution uses Obsidian-style filename-based lookup across all vault directories.

Currently `resolveWikilinks()` does path-based lookup in specific directories (`notes/{slug}.md`, `_inbox/*-{slug}.md`). This is fragile — adding `_deprecated/` means updating the resolution logic every time a new directory is added.

Instead, switch to Obsidian-style resolution: scan all `.md` files in the vault (recursively or across known directories) and resolve `[[slug]]` by matching the slug against filenames. This matches how Obsidian actually resolves wikilinks and means moving files between directories never breaks links.

For this increment, the known directories are: `_inbox/`, `notes/`, `_deprecated/`. The resolution function scans all three. The `_meta/` directory is excluded (those are not notes).

**Rationale**: Obsidian resolves `[[wikilinks]]` by filename, not by path. When you move a file in Obsidian, it auto-updates links — but even without that, `[[slug]]` finds `slug.md` anywhere in the vault. Our resolution should match this behavior. This also makes the `_deprecated/` folder work without breaking links.

**Alternatives considered**: Path-based lookup with hardcoded directory list (current approach — fragile, breaks when directories change). Keeping deprecated notes in `notes/` to avoid the problem (rejected — separating deprecated notes is cleaner for filtering and vault hygiene).
</decision>

<decision id="7">
**Decision**: Slug collision in `notes/` during promotion is an error.

If `notes/{slug}.md` already exists, promote errors. This is either a duplicate (should have been caught by similarity check) or a replacement (old note should have been deprecated first).

**Rationale**: Keep promote simple. Collision resolution is the caller's responsibility.
</decision>

<decision id="8">
**Decision**: The Librarian is a subagent, not just instructions.

The Librarian runs as a subagent invoked by the user agent. This keeps memory-handling work isolated from the user agent's context, preventing context poisoning. The Librarian reads the vault, processes inbox notes, calls CLI tools, and returns a summary to the caller.

For this increment, we target OpenCode:
- **Agent definition**: A Librarian agent definition file that configures the subagent (permissions, tools, model)
- **Skill definition**: A SKILL.md that provides the Librarian's workflow instructions

The CLI tools (`promote`, `deprecate`, `lint-note`) are harness-agnostic. Only the agent/skill definitions are harness-specific. Future increments will add definitions for VS Code Copilot, GitHub CLI, etc.

**Deployment templates in the vault**: The vault carries template files for agent and skill definitions in `_meta/templates/`. `agent-memory instructions` references these templates so users know how to deploy the Librarian in their harness. `agent-memory init` seeds these templates from embedded Go templates.

**Rationale**: Running the Librarian as a subagent isolates vault-scanning, dedup reasoning, and human confirmation flows from the user agent's working context. This is the same pattern used for golang, QA, and other specialist agents — the Librarian is a specialist for memory management.
</decision>

<decision id="9">
**Decision**: The `promote` subcommand takes `--slug` to identify the note, not a file path.

```bash
agent-memory promote --slug=my-observation [--confirmed] [--vault=path]
```

The subcommand finds the note in `_inbox/` by scanning for files ending with `-{slug}.md`. If multiple files match, it errors (ambiguity).

**Rationale**: Slugs are the canonical identifier throughout the system.
</decision>

<decision id="10">
**Decision**: Update `agent-memory instructions` output to include Librarian workflow and deployment guidance.

Add sections explaining:
- After writing notes, agents should invoke the Librarian subagent to process pending inbox notes
- How the Librarian works (lint, type-check, promote, deprecate)
- The human confirmation flow for constraints/decisions
- Where to find deployment templates for the Librarian agent/skill definitions
- How to set up the Librarian in OpenCode (with pointers to templates in the vault)

This is a text change to the hardcoded string in `instructions.go`.
</decision>

<decision id="11">
**Decision**: Dual JSON/human output for promote and deprecate.

JSON output for promote:
```json
{
  "status": "promoted",
  "slug": "my-observation",
  "from": "_inbox/2026-05-21-my-observation.md",
  "to": "notes/my-observation.md",
  "epistemic_type": "observation",
  "confirmed": false
}
```

JSON output for deprecate:
```json
{
  "status": "deprecated",
  "slug": "old-note",
  "superseded_by": "new-note",
  "moved_to": "_deprecated/old-note.md"
}
```

Error cases use `"status": "error"` with an `"error"` field.
</decision>

<decision id="12">
**Decision**: `agent-memory deprecate` moves notes to `_deprecated/` and updates frontmatter.

```bash
agent-memory deprecate --slug=<slug> --superseded-by=<new-slug> [--vault=<path>] [--json]
```

The subcommand: (1) finds the note in `notes/` by slug, (2) reads and parses it, (3) sets `status: deprecated`, `superseded-by: <new-slug>`, updates `updated` date, (4) re-serializes, (5) moves the file from `notes/{slug}.md` to `_deprecated/{slug}.md`, (6) writes log entry.

Deprecated notes live in `_deprecated/`, not `notes/`. This is safe because wikilink resolution now uses Obsidian-style filename-based lookup across all vault directories (Decision 6). `[[old-slug]]` still resolves to `_deprecated/old-slug.md`.

Benefits of a separate folder:
- Clean separation: `notes/` contains only active, verified notes
- Easier filtering for search and context loading (no need to filter by status)
- Vault hygiene: deprecated notes don't clutter the active note set
- Obsidian sidebar shows them in a distinct folder

`_deprecated/` is created by `agent-memory init` (add to vault scaffold).

**Rationale**: Obsidian resolves wikilinks by filename, not path. Moving files between folders does not break links. A separate folder provides cleaner organization than status-based filtering within `notes/`.
</decision>

<decision id="13">
**Decision**: Vault carries deployment templates for agent/skill definitions.

`agent-memory init` seeds template files in the vault for deploying the Librarian:

```
_meta/
├── templates/
│   ├── librarian-agent.md      ← Agent definition template (OpenCode format)
│   └── librarian-skill.md      ← Skill definition template (OpenCode format)
```

These are reference templates, not active agent definitions. `agent-memory instructions` tells users where to find them and how to deploy them to their harness (e.g., copy to `~/.config/opencode/agent/` or `~/.config/opencode/skills/`).

Future increments will add templates for other harnesses (VS Code Copilot, GitHub CLI).

**Rationale**: Keeping templates in the vault makes them versionable with the code and discoverable via `agent-memory instructions`. Users don't need to find external documentation to set up the Librarian.
</decision>

<decision id="14">
**Decision**: Replace the hybrid regex+goldmark wikilink extraction with the `goldmark-wikilink` extension.

Currently `ExtractWikilinks()` uses a hybrid approach: goldmark parses the AST to identify code block/span byte ranges, then a regex (`\[\[([^\]]+)\]\]`) finds wikilink matches outside those ranges. The `goldmark-wikilink` extension (`go.abhg.dev/goldmark/wikilink`) parses `[[...]]` into proper AST nodes natively. By adding it as a goldmark extension, we get:

- Wikilinks as first-class AST nodes (`wikilink.Node` with `Target` field)
- Code block/span exclusion handled automatically by goldmark's parser (inline parsers don't run inside code contexts)
- No regex, no manual `codeRange` tracking
- Proper handling of edge cases the extension already covers

The rewritten `ExtractWikilinks()`: configure goldmark with `&wikilink.Extender{}`, parse the content, walk the AST for `wikilink.Kind` nodes, collect `string(node.Target)` values, deduplicate.

We don't need the renderer — we only use the parser to extract link targets. The `Resolver` interface is also unused (we do our own resolution).

**Rationale**: We already depend on goldmark. The extension is well-maintained (v0.6.0, BSD-3, 34 importers). It eliminates regex and ~30 lines of manual code-range logic.
</decision>

<decision id="15">
**Decision**: Rename `UpdateType` to `EpistemicType` in the Frontmatter struct.

The frontmatter field `update-type` / Go field `UpdateType` is renamed to `epistemic-type` / `EpistemicType`. This aligns the codebase with the terminology used throughout the design doc and eliminates the confusing split where the struct says `UpdateType` but all prose, JSON output, and function parameters say `epistemicType`.

The YAML key changes from `update-type` to `type`. Existing notes using `update-type` will need migration (lint rule or backward-compatible parsing).

**Rationale**: One concept, one name. The term "epistemic type" is what the design doc uses; the Go field is `EpistemicType` for clarity in code, but the YAML key is simply `type` — short, unambiguous in note frontmatter context.
</decision>

<decision id="16">
**Decision**: `--confirmed` is silently ignored when `requires-human-review` is false.

Passing `--confirmed` on a note that doesn't require human review is a no-op. This simplifies the Librarian's logic — it can always pass `--confirmed` after getting human confirmation without needing to branch on note type first.

**Rationale**: Strictness here would add complexity with no safety benefit. The flag only gates promotion of constraint/decision notes; for other types it's irrelevant.
</decision>

</design_decisions>

<data_flow>

## Data Flow

### Promotion flow (happy path — observation)
```
User agent completes its work, invokes Librarian subagent
  → Librarian scans _inbox/ for .md files
  → for each note:
      → agent-memory lint-note --json <path>  (validate)
      → check epistemic type from frontmatter
      → if observation: agent-memory promote --json --slug=<slug>
          → promote reads note, lints, checks wikilinks, moves file, updates frontmatter, writes log
          → returns JSON result
  → Librarian returns summary to user agent
```

### Promotion flow (constraint/decision)
```
Librarian subagent processing inbox
  → for each note:
      → agent-memory lint-note --json <path>
      → if constraint/decision:
          → Librarian presents note to human (via tool call), asks for confirmation
          → human confirms (or rejects)
          → if confirmed: agent-memory promote --json --slug=<slug> --confirmed
          → if rejected: Librarian leaves note in inbox
```

### Deprecation flow (replacement)
```
Librarian detects new note replaces existing note (via similarity or explicit reference)
  → agent-memory deprecate --json --slug=<old-slug> --superseded-by=<new-slug>
      → deprecate reads old note, sets status: deprecated, superseded-by, updates `updated` date
      → re-serializes and moves file from notes/ to _deprecated/
      → writes log
  → agent-memory promote --json --slug=<new-slug>
      → promote moves new note from _inbox/ to notes/
```

</data_flow>

<api_surface>

## CLI Surface

### `agent-memory promote`
```
agent-memory promote --slug=<slug> [--confirmed] [--vault=<path>] [--json]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--slug` | yes | Slug of the note to promote |
| `--confirmed` | no | Required for constraint/decision notes |
| `--vault` | no | Vault path override |
| `--json` | no | JSON output |

Exit codes: 0 = promoted, 1 = error

### `agent-memory deprecate`
```
agent-memory deprecate --slug=<slug> --superseded-by=<new-slug> [--vault=<path>] [--json]
```

| Flag | Required | Description |
|------|----------|-------------|
| `--slug` | yes | Slug of the note to deprecate |
| `--superseded-by` | yes | Slug of the replacement note |
| `--vault` | no | Vault path override |
| `--json` | no | JSON output |

Exit codes: 0 = deprecated, 1 = error

### Vault structure changes

```
.agent-memory/
├── _meta/
│   ├── templates/
│   │   ├── librarian-agent.md      ← NEW: agent definition template
│   │   └── librarian-skill.md      ← NEW: skill definition template
│   └── ... (existing files)
├── _inbox/                          ← existing
├── _deprecated/                     ← NEW: deprecated notes (created by `agent-memory init`)
├── _contested/                      ← existing
└── notes/                           ← existing (only active verified notes)
```

### New/modified types

```go
// Frontmatter — add field:
SupersededBy string `yaml:"superseded-by"`

// PromoteResult — new type for promote output
type PromoteResult struct {
    Status        string `json:"status"`
    Slug          string `json:"slug"`
    From          string `json:"from,omitempty"`
    To            string `json:"to,omitempty"`
    EpistemicType string `json:"epistemic_type,omitempty"`
    Confirmed     bool   `json:"confirmed"`
    Error         string `json:"error,omitempty"`
}

// DeprecateResult — new type for deprecate output
type DeprecateResult struct {
    Status       string `json:"status"`
    Slug         string `json:"slug"`
    SupersededBy string `json:"superseded_by,omitempty"`
    MovedTo      string `json:"moved_to,omitempty"`
    Error        string `json:"error,omitempty"`
}
```

### Exported functions (newly exported from internal/note/)
```go
func AppendLog(vaultPath, action, epistemicType, slug, agent string) error
func ResolveWikilinks(vaultPath, body string) []string
func FindSimilarNotes(vaultPath string, incomingTokens []string) ([]SimilarNote, error)
```

</api_surface>

<open_questions>

## Open Questions

- **Synthesis creation**: Deferred to Increment 9 (Librarian Enhancements). This increment focuses on promotion and deprecation only.

- **Assumption outdating**: This is LLM judgment work — detecting that a new note contradicts an existing assumption. The Librarian skill instructions will include guidance to check for contradicted assumptions during its pass. No new subcommand needed — the Librarian reads notes, reasons about contradictions, and calls `deprecate` if appropriate.

- **Template directory location**: Resolved — `_meta/templates/` (see Decision 13).

- **Wikilink resolution performance**: Scanning all `.md` files across three directories for every wikilink resolution call could be slow in large vaults. For now this is fine (vaults are small). Future optimization: cache the filename→path mapping and invalidate on file changes.

</open_questions>

</design_artifact>
