# Agent Memory: Incremental Implementation Roadmap

**Status:** Increments 1–3 complete. Increment 4 (Note Search) deferred. Next: Increment 5 (Librarian & Promotion).

Each increment is a complete vertical slice: design → research → structure → plan → work → review. Each increment is independently valuable and individually verifiable via comprehensive integration tests.

---

## Increment 1: Vault Scaffolding & Initialization

**Goal:** Establish the vault structure and CLI entry point so all subsequent increments have a place to write.

**What becomes possible after this increment:**
- A vault can be created at any path with a single command
- The vault structure is initialized and idempotent
- The writing protocol document exists (minimal content at this stage)
- Agents can be given instructions for working with the vault via `agent-memory instructions`
- Humans can browse the vault in Obsidian

**Scope:**

### Vault Initialization (`agent-memory init [path]`)
- Uses a **git-init model**: creates a vault at the given path, or at `.agent-memory/` in the current directory if no path is provided
- No configuration file is written; the vault's presence at a path IS the configuration
- **Idempotency:** Running `init` multiple times on the same vault path is safe
  - If vault already exists and is intact, verify structure and exit 0
  - If vault exists but is missing required directories, repair it
  - `--force` flag: overwrite conflicting files
  - `--clean --force` flags: delete and reinitialize the vault
- Create vault directory structure:
  ```
  .agent-memory/
  ├── _meta/
  │   ├── writing-protocol.md      ← Rules agents MUST follow (minimal in v1)
  │   ├── tag-taxonomy.md          ← Empty in v1
  │   ├── constraints-summary.md   ← Empty in v1
  │   ├── status-lifecycle.md      ← Lifecycle stages (static template)
  │   └── log.md                   ← Append-only log (empty at init)
  ├── _inbox/                      ← All agent writes land here first
  ├── _contested/                  ← Notes with unresolved contradictions (empty at init)
  └── notes/                       ← All promoted notes, flat
  ```
- Seed `_meta/` files from embedded templates (no network, no external files)
- Log initialization to `_meta/log.md`: `## [2026-05-11T14:30:00+03:00] init | vault initialized` (full ISO 8601 with timezone)
- All commands emit JSON on stdout; exit 0 on success, exit 1 on error

### Agent Instructions (`agent-memory instructions`)
- Outputs an agent configuration blurb to stdout
- Intended for piping into a repo's agent instruction file, e.g.:
  ```
  agent-memory instructions >> .claude/AGENTS.md
  ```
- `init` does not touch the user's repo; no `AGENTS.md` is written inside the vault

### Vault Discovery (for future subcommands — not active in v1)
- In v1, vault path is always explicit (via argument or current directory default)
- Future subcommands will resolve vault location via:
  1. `AGENT_MEMORY_VAULT` environment variable (highest priority)
  2. Walk up directories looking for `.agent-memory/`
  3. `~/.local/share/agent-memory` global fallback

### Writing Protocol (v1 — Minimal)
- Document exists at `_meta/writing-protocol.md`
- Content: "This vault is for agent memory. Writing is not yet enabled. See the output of `agent-memory instructions` for current capabilities."
- Will be expanded in Increment 2

### Tooling
- Go version managed via **mise**
- Task runner: **mise** tasks
- Tests: BDD with **Ginkgo/Gomega**; unit tests co-located in packages, integration tests in `test/integration/`

### Verification (Integration Tests)
- **Idempotency:** Run `agent-memory init` twice on same path, verify no errors and vault state unchanged
- **Explicit path:** Run `agent-memory init /some/path`, verify vault is created at that path
- **Default path:** Run `agent-memory init` in a directory, verify `.agent-memory/` is created
- **Vault structure:** Verify all required directories exist after init
- **Seed files:** Verify all `_meta/` files exist and contain expected content
- **Log entry:** Verify init is logged to `_meta/log.md` with full ISO 8601 timestamp
- **Repair:** Corrupt vault (delete a directory), run init again, verify repair
- **Force flag:** Verify `--force` overwrites conflicting files without error
- **Clean reinit:** Verify `--clean --force` deletes and reinitializes the vault
- **JSON output:** Verify all commands emit valid JSON on stdout
- **Instructions output:** Run `agent-memory instructions`, verify output is a non-empty agent blurb
- **No repo mutation:** Verify `init` does not create or modify any files outside the vault path

---

## Increment 2: Note Format & Frontmatter Parsing

**Goal:** Define and validate the note structure so agents can write notes that tools can parse.

**What becomes possible after this increment:**
- Agents understand the note format and frontmatter requirements
- Notes can be validated for structural correctness
- The writing protocol document explains what agents should do

**Scope:**

### Note Format Specification
- Frontmatter (YAML) + body (markdown)
- Required frontmatter fields:
  ```yaml
  title: "Concise factual claim"
  created: 2026-04-24
  updated: 2026-04-24
  status: inbox
  epistemic-type: observation
  confidence: medium
  scope: project
  project: project-slug
  domain: [golang]
  source-agent: research/analyst-a
  source-artifact: "<repo-relative path or URL>"
  ```
- Optional fields (populated by tools, not agents):
  ```yaml
  review-by: ""
  verified-by: ""
  verified-date: ""
  requires-human-review: false
  update-type: ""
  targets: []
  tags: []
  ```
- Body sections (required):
  - `# Title` (repeats frontmatter title)
  - `## Evidence` (where this was observed)
  - `## Implications` (what other agents need to know)
  - `## Related` (wikilinks to related notes)

### Frontmatter Parsing
- YAML parsing (`gopkg.in/yaml.v3`)
- Field validation rules:
  - `title`: non-empty string
  - `created`, `updated`: ISO date format (YYYY-MM-DD)
  - `status`: one of `inbox`, `verified`, `deprecated`, `contested`, `superseded`
  - `epistemic-type`: one of `observation`, `pattern`, `constraint`, `decision`, `assumption`, `synthesis`
  - `confidence`: one of `low`, `medium`, `high`
  - `scope`: one of `project`, `cross-project`
  - `project`: non-empty string (required if `scope: project`)
  - `domain`: list of strings
  - `source-agent`: non-empty string
  - `source-artifact`: non-empty string
- Malformed YAML is rejected with clear error message
- Missing required fields are rejected with clear error message

### `lint-note` Binary
- Single-file validation
- Checks all frontmatter fields are present and valid
- Checks all required body sections are present
- Checks no placeholder values remain
- Exit 0 on pass, 1 on failure
- JSON output: `{"valid": true}` or `{"valid": false, "errors": ["...", "..."]}`

### Writing Protocol (v2 — Agent Instructions)
- Document at `_meta/writing-protocol.md`
- Content:
  ```markdown
  # Writing Protocol for Agent Memory

  When writing a note to this vault, follow these rules:

  1. **Frontmatter:** Every note must have YAML frontmatter with these fields:
     - `title`: Concise factual claim (one sentence)
     - `created`, `updated`: ISO dates (YYYY-MM-DD)
     - `status`: Always `inbox` for new notes
     - `epistemic-type`: One of: observation, pattern, constraint, decision, assumption
     - `confidence`: One of: low, medium, high
     - `scope`: One of: project, cross-project
     - `project`: Project slug (required if scope is project)
     - `domain`: List of technology domains (e.g., [golang, auth])
     - `source-agent`: Your agent name
     - `source-artifact`: Path or URL where this claim came from

  2. **Body:** Write the note body with these sections:
     - `# Title` (repeat the frontmatter title)
     - `## Evidence` (where you observed this; specific line/section references)
     - `## Implications` (what other agents need to know)
     - `## Related` (wikilinks to related notes: [[note-slug]])

  3. **Validation:** Your note will be validated by `lint-note`. If it fails, you'll get specific error messages.

  4. **What happens next:** Your note lands in `_inbox/`. It will be promoted to `notes/` based on its epistemic type and age.

  See `_meta/status-lifecycle.md` for the full lifecycle.
  ```

### Verification (Integration Tests)
- **Valid note:** Create a note with all required fields, run `lint-note`, verify pass
- **Missing field:** Create a note missing a required field, run `lint-note`, verify fail with specific error
- **Invalid value:** Create a note with invalid `epistemic-type`, run `lint-note`, verify fail
- **Malformed YAML:** Create a note with broken YAML, run `lint-note`, verify fail
- **Body sections:** Create a note missing `## Evidence`, run `lint-note`, verify fail
- **Placeholder values:** Create a note with placeholder text, run `lint-note`, verify fail
- **Agent verification test:** Have an agent read the writing protocol and write a valid note, verify it passes `lint-note`

---

## Increment 3: Note Writing (Write Protocol)

**Goal:** Implement the skill and static tool so agents can write notes to the vault.

**What becomes possible after this increment:**
- Agents can write notes to the vault using the `write-memory` skill
- Notes are validated and placed in `_inbox/`
- The similarity check prevents obvious duplicates
- Agents understand how to compose notes with proper evidence and implications

**Scope:**

### Static Tool: `memory-write-mechanics`
- Similarity check (Jaccard on normalized word tokens, threshold TBD during QRSPI)
- Frontmatter assembly (created, updated, status, TTL assignment)
- Inbox placement (`_inbox/{YYYY-MM-DD}-{slug}.md`)
- Log entry writing (`_meta/log.md`)
- Refusal behavior on similarity hit (return candidates as JSON)
- Wikilink validation (warn on unresolved links, but don't block write)

### Skill: `write-memory`
- Takes calling agent's claim, evidence, and metadata
- Composes note body (with LLM judgment)
- Calls `memory-write-mechanics` for similarity check
- Reasons about false positives (is the similarity hit a real duplicate?)
- Calls `memory-write-mechanics` to write the note
- Returns success/failure to calling agent
- Teaches the calling agent how to write notes (via skill instructions)

### Writing Protocol (v3 — Skill Instructions)
- Updated `_meta/writing-protocol.md` with skill usage instructions
- Explains how to invoke `write-memory` skill
- Explains what the skill expects (claim, evidence, metadata)
- Explains what the skill returns (success/failure, warnings)

### Verification (Integration Tests)
- **Write a note:** Invoke `write-memory` skill, verify note lands in `_inbox/`
- **Duplicate detection:** Write a similar note, verify skill detects it and asks for clarification
- **False positive handling:** Write a note that's similar but different, verify skill reasons about it correctly
- **Frontmatter assembly:** Verify frontmatter is correctly assembled (created, updated, status, TTL)
- **Log entry:** Verify log entry is written
- **Wikilink warnings:** Write a note with unresolved wikilinks, verify skill warns but doesn't block
- **Agent verification test:** Have an agent use the skill to write a note, verify it passes `lint-note`

---

## Increment 4: Note Search (Read Protocol) — DEFERRED

> **Deferred:** Postponed until the vault has enough data to validate whether a search command is the right interface vs. direct read access. The two-phase retrieval model may be over-engineered for small vaults. Revisit after real usage.

**Goal:** Implement the skill and static tool so agents can find relevant notes without loading everything.

**What becomes possible after this increment:**
- Agents can search for notes using the `search-memory` skill
- Two-phase retrieval is hidden from the agent (agent just says "find notes about X")
- Agents can load relevant context at session start
- Agents can find existing notes before writing new ones

**Scope:**

### Static Tool: `memory-search-mechanics`
- Phase 1: Frontmatter discovery (grep + YAML parsing)
  - Returns titles, frontmatter, and metadata only (no body content)
  - Filters to `status: verified` only
  - Supports filtering by project, domain, epistemic-type
  - Tag alias expansion (read from `_meta/tag-taxonomy.md`)
- Phase 2: Selective body read
  - Takes a list of slugs
  - Returns full note bodies
  - Capped at 10 notes per invocation
  - Returns as JSON

### Skill: `search-memory`
- Takes a query and optional filters (project, domain, type)
- Calls `memory-search-mechanics` Phase 1 for discovery
- Reasons about which notes to read in full (LLM judgment)
- Calls `memory-search-mechanics` Phase 2 to get bodies
- Returns structured results to calling agent
- Hides the two-phase logic from the agent

### Verification (Integration Tests)
- **Simple search:** Search for a note by title, verify results
- **Filtered search:** Search with project/domain filters, verify filtering works
- **Tag alias expansion:** Search for "auth", verify "authentication" notes are found
- **Status filtering:** Verify only `status: verified` notes are returned
- **Body capping:** Search for many notes, verify at most 10 bodies are returned
- **Agent verification test:** Have an agent use the skill to search for notes, verify results are relevant

---

## Increment 5: Librarian Agent & Promotion Pipeline

**Goal:** Implement the Librarian as a skill-triggered agent that validates, classifies, promotes, deduplicates, and maintains inbox notes. This is the critical path — without the Librarian, all notes stay in `_inbox/` forever.

**What becomes possible after this increment:**
- Notes move from `_inbox/` to `notes/` via Librarian promotion
- Constraints and decisions require human confirmation before promotion
- Duplicate notes are detected and handled
- Outdated assumptions are removed when contradicted by new notes
- Pattern notes require 2+ corroborating references
- Synthesis notes can be created by the Librarian

**Scope:**

### `agent-memory promote` subcommand
- Moves a validated note from `_inbox/` to `notes/`, sets `status: verified`
- Called by the Librarian after validation, not directly by user agents
- `--confirmed` flag required for `constraint` and `decision` notes
- Refuses to promote notes with unresolved `[[wiki-links]]`
- Writes promotion line to `_meta/log.md`

### Librarian Skill Definition
- Skill invoked by user agents as last step of workflow
- Processes all pending inbox notes in a single pass
- Core responsibilities per invocation:
  - Lint check (call `agent-memory lint-note` on each inbox note)
  - Similarity/deduplication check against existing `notes/`
  - Promotion by epistemic type rules (see design doc §5.5)
  - Human confirmation flow for constraints/decisions (via agent tool call)
  - Deprecation of replaced notes (forward link)
  - Assumption outdating when contradicted by new notes

### Verification (Integration Tests)
- **Observation promotion:** Write observation, run Librarian, verify promotion to `notes/`
- **Pattern promotion:** Write pattern referencing 2+ notes, verify promotion
- **Pattern rejection:** Write pattern with <2 references, verify stays in inbox
- **Constraint gating:** Write constraint, verify Librarian requests human confirmation
- **Decision gating:** Write decision, verify same confirmation flow
- **Assumption promotion:** Write assumption, verify promotion (stays as assumption type)
- **Duplicate detection:** Write duplicate note, verify Librarian detects and handles it
- **Wikilink blocking:** Write note with unresolved links, verify promotion blocked
- **Replacement flow:** Write replacement note, verify old note deprecated with forward link
- **Lint failure:** Write malformed note, verify Librarian rejects it
- **Idempotency:** Run Librarian twice, verify same result

---

## Increment 6: Session Initialization (Agent Integration)

**Goal:** Implement the context bundle so agents can load relevant knowledge at session start.

**What becomes possible after this increment:**
- Agents can load Core tier (writing protocol + constraints summary + log tail) at session start
- Agents can load Index tier (project and domain indices) at session start
- Agents are informed of stale notes before work begins
- Token budget is tracked and enforced

**Scope:**

### Static Tool: `memory-context`
- Bundles Core tier:
  - `_meta/writing-protocol.md`
  - `_meta/constraints-summary.md`
  - Last ~20 entries from `_meta/log.md`
  - Target: < 600 tokens combined
- Bundles Index tier:
  - `_index-{project}.md` for current project
  - `_index-{domain}.md` for declared domains
  - Target: < 300 tokens per index
- Staleness detection:
  - Notes where `review-by < today`
  - Returned as list with review-by dates
- Budget tracking:
  - Tracks total tokens loaded
  - Defers domain indices if total > 2000 tokens
  - Returns budget info as JSON

### Skill: `load-memory-context`
- Takes optional project and domain list
- Calls `memory-context` tool
- Returns bundled context to calling agent
- Teaches agent what context is available and how to use it

### Verification (Integration Tests)
- **Core tier loading:** Call `memory-context`, verify Core tier is returned
- **Index tier loading:** Call with project/domains, verify indices are returned
- **Staleness detection:** Create stale notes, call `memory-context`, verify they're identified
- **Budget tracking:** Verify budget info is returned
- **Budget gating:** Load many indices, verify budget gate defers when needed
- **Agent verification test:** Have an agent load context at session start, verify it's available

---

## Increment 7: Index & Constraint Maintenance (Deterministic Maintenance)

**Goal:** Implement the maintenance tools so indices and constraints stay current without manual work.

**What becomes possible after this increment:**
- Indices are automatically regenerated from frontmatter scans
- Constraints summary is automatically regenerated
- Orphaned notes are detected
- Vault integrity is verifiable

**Scope:**

### Static Tool: `memory-reindex`
- Scans all notes in `notes/` and `_inbox/`
- Generates `_index-{project}.md` from frontmatter scan
- Generates `_index-{domain}.md` from frontmatter scan
- Generates `_meta/constraints-summary.md` from all `constraint` notes
- Idempotency: safe to run multiple times
- Orphan detection: identifies notes not referenced from any index
- Log entry writing for reindex operations

### Verification (Integration Tests)
- **Index generation:** Write notes with different projects/domains, run reindex, verify indices are generated
- **Constraints summary:** Write constraint notes, run reindex, verify summary is generated
- **Idempotency:** Run reindex twice, verify same output
- **Orphan detection:** Write orphaned note, run reindex, verify it's detected
- **Index updates:** Add new note, run reindex, verify index is updated

---

## Increment 8: Linting & Integrity (Quality Assurance)

**Goal:** Implement comprehensive vault linting so integrity issues are caught early.

**What becomes possible after this increment:**
- Vault integrity is verifiable
- Broken links are detected
- Orphaned notes are identified
- Stale notes are flagged
- Deprecated notes without forward links are caught
- Dense areas (many notes without synthesis) are identified

**Scope:**

### Static Tool: `lint-vault`
- Unresolved wikilinks (notes/ and _inbox/)
- Orphaned notes (not referenced from any index)
- Stale notes (review-by < today)
- Deprecated notes without forward links
- Untagged notes (in notes/, no tags field)
- Contested notes (any files in _contested/)
- Dense areas (tag+domain combos with 5+ notes and no synthesis)
- Source-artifact resolution (file exists or URL is valid)
- Returns findings as JSON
- Exit 0 if clean, 1 if findings

### Skill: `lint-memory`
- Calls `lint-vault` tool
- Presents findings to agent in human-readable format
- Suggests fixes where possible
- Teaches agent how to interpret findings

### Verification (Integration Tests)
- **Unresolved links:** Create notes with broken links, run lint, verify detection
- **Orphaned notes:** Create orphaned note, run lint, verify detection
- **Stale notes:** Create stale notes, run lint, verify detection
- **Deprecated without link:** Create deprecated note without forward link, run lint, verify detection
- **Untagged notes:** Create untagged note, run lint, verify detection
- **Contested notes:** Create contested notes, run lint, verify detection
- **Dense areas:** Create many notes in same domain, run lint, verify detection
- **Clean vault:** Verify clean vault returns no findings

---

## Increment 9: Librarian Enhancements (Tagging, Pattern Detection, Synthesis)

> **Prerequisite:** Increment 5 (Librarian core). This increment adds advanced Librarian capabilities on top of the core promotion pipeline.

**Goal:** Extend the Librarian with tag management, pattern detection from observations, and synthesis page drafting.

**What becomes possible after this increment:**
- Untagged notes are tagged by the Librarian
- Pattern candidates are detected from clusters of observations
- Synthesis pages are drafted for dense topic areas

**Scope:**

### Tag Management (`agent-memory tag`)
- Librarian assigns tags to untagged promoted notes
- Tag taxonomy management (accept, reject, alias)
- Updates `_meta/tag-taxonomy.md`

### Pattern Detection
- Librarian reads recent observations, identifies clusters
- Drafts pattern notes for human review

### Synthesis Drafting (`agent-memory synthesize`)
- Gathers contributing notes for an entity/tag
- Generates scaffold with deterministic parts
- Librarian fills prose section

### Contested Resolution
- Reviews notes in `_contested/`
- Recommends resolution to human
- Human confirms, Librarian executes

### Verification (Integration Tests)
- **Tagging:** Create untagged promoted notes, verify Librarian assigns tags
- **Tag taxonomy:** Verify accept/reject/alias operations update taxonomy file
- **Pattern detection:** Create observation cluster, verify Librarian drafts pattern note
- **Synthesis drafting:** Create notes for entity, verify Librarian drafts synthesis page
- **Contested resolution:** Create conflicting notes, verify Librarian recommends resolution

---

## Increment 10: Backup & Recovery (Durability)

**Goal:** Implement backup and recovery so vault data is durable.

**What becomes possible after this increment:**
- Vault can be backed up to timestamped archives
- Vault can be recovered from backups
- Backup integrity is verifiable
- Backup retention policy is enforced

**Scope:**

### Static Tool: `agent-memory backup`
- Runs `lint-vault` as gate (abort if findings)
- Creates timestamped `tar.gz` of vault
- Stores backup in `~/.local/share/agent-memory/backups/` (or XDG equivalent)
- Retention policy: keep last 30 backups (configurable)
- Backup manifest: records vault state at backup time
- Idempotency: safe to run multiple times

### Static Tool: `agent-memory restore`
- Lists available backups
- Restores vault from selected backup
- Verifies backup integrity before restore
- Preserves recent writes (doesn't overwrite newer notes)

### Verification (Integration Tests)
- **Backup creation:** Create vault, run backup, verify archive is created
- **Backup integrity:** Verify backup contains all vault files
- **Retention policy:** Create many backups, verify old ones are pruned
- **Restore:** Backup vault, delete a note, restore, verify note is recovered
- **Idempotency:** Run backup twice, verify no duplicate archives

---

## Increment 11: Semantic Search (Enhancement)

**Goal:** Add semantic search capabilities for better note discovery.

**What becomes possible after this increment:**
- Agents can search for notes by semantic similarity
- Search results are ranked by relevance
- Agents can find related notes even if titles don't match

**Scope:**

### Enhancement to `memory-search-mechanics`
- Add semantic search phase (optional, after Phase 1)
- Use local embeddings (no API required)
- Rank results by relevance
- Combine with tag-based search for better results

### Skill Enhancement: `search-memory`
- Optionally use semantic search
- Combine semantic and tag-based results
- Rank by relevance

### Verification (Integration Tests)
- **Semantic search:** Search for concept, verify semantically related notes are found
- **Ranking:** Verify most relevant notes are ranked first
- **Hybrid search:** Combine semantic and tag-based search, verify results are good

---

## Increment 12: Vault Archival (Enhancement)

**Goal:** Archive old notes so vault doesn't grow unbounded.

**What becomes possible after this increment:**
- Old notes can be archived to separate storage
- Archived notes are still searchable
- Vault performance is maintained as it grows

**Scope:**

### Static Tool: `memory-archive`
- Identifies notes older than configurable threshold
- Moves notes to `_archive/` directory
- Updates indices to reference archived notes
- Maintains searchability of archived notes
- Idempotency: safe to run multiple times

### Verification (Integration Tests)
- **Archive creation:** Create old notes, run archive, verify they're moved
- **Searchability:** Search for archived notes, verify they're found
- **Index updates:** Verify indices reference archived notes
- **Idempotency:** Run archive twice, verify no duplicate moves

---

## Increment 13: Global Vault Init (Stabilization)

**Goal:** Support `agent-memory init --global` to create vaults under `$XDG_DATA_HOME/agent-memory` or `$HOME/.agent-memory`.

**What becomes possible after this increment:**
- Users can create a global vault without specifying a full path
- Optional folder name argument replaces the default folder name under the global path
- Vault discovery already handles the global fallback; this makes creation ergonomic

**Scope:**

### CLI Enhancement: `agent-memory init --global [name]`
- `--global` / `-g` flag on `init` command
- Resolves target: `$XDG_DATA_HOME/agent-memory` → `$HOME/.local/share/agent-memory` → `$HOME/.agent-memory`
- Optional `name` argument replaces `agent-memory` in the resolved path (e.g. `init -g work` → `~/.local/share/work/`)
- All other init behavior unchanged (idempotent, `--force`, embedded templates)

### Verification (Integration Tests)
- **Default global path:** `init -g` creates vault at XDG-compliant path
- **Custom name:** `init -g work` creates vault at expected path
- **XDG override:** `XDG_DATA_HOME` env var is respected
- **Conflict with positional arg:** `init -g /some/path` errors clearly

---

## Increment 14: Template Customization (Stabilization)

**Goal:** Allow users to customize vault templates via a config directory, with git-config-style inheritance.

**What becomes possible after this increment:**
- Users can override default templates without forking the binary
- Per-vault and global config coexist with clear precedence
- Linting remains correct regardless of template customization

**Scope:**

### Config Directory Structure
- Global config: `$XDG_CONFIG_HOME/agent-memory/` (or `$HOME/.config/agent-memory/`)
- Per-vault config: `.agent-memory/_config/`
- Resolution order: per-vault → global → embedded defaults (git-config model)
- `agent-memory init` populates config with embedded defaults on first run

### Template Override
- `templates/` subdirectory in config holds user-editable templates
- `agent-memory init` uses config templates instead of embedded when present
- Template changes do not retroactively affect existing vault files

### Lint Compatibility
- Lint rules that validate enum values must either read valid values from config or remain template-agnostic
- Design decision needed: hardcoded rules vs. config-driven rules (to be resolved during QRSPI D phase)

### Verification (Integration Tests)
- **Default behavior:** Without config, embedded templates are used (no regression)
- **Global override:** Custom template in global config is used by init
- **Per-vault override:** Per-vault config takes precedence over global
- **Lint correctness:** Lint passes/fails correctly with customized templates

---

## Next Steps

Each increment will be the basis of a full QRSPI process:

1. **Q (Questions):** What do we need to know about the codebase/requirements?
2. **R (Research):** Gather objective facts
3. **D (Design):** Discuss architectural options
4. **S (Structure):** Define implementation structure
5. **P (Plan):** Create tactical implementation plan
6. **W (Work Tree):** Organize work into manageable pieces
7. **I (Implement):** Write code
8. **PR (Review):** Prepare for human review

After each increment is complete, we'll have a working, tested feature that's individually valuable.

---

## Implementation Sequence

**Phase 1: Foundation (Increments 1–3) ✅**
- Vault scaffolding and initialization
- Note format and frontmatter parsing
- Note writing (write protocol)
- Agents can write validated notes to `_inbox/`

**Phase 2: Librarian & Promotion (Increment 5)**
- Librarian agent (skill-triggered)
- Promotion pipeline (all epistemic types)
- Human confirmation for constraints/decisions
- Deduplication and assumption outdating
- Notes can now move from `_inbox/` to `notes/`

**Phase 3: Search & Context (Increments 4, 6)**
- Note search — simple frontmatter/tag filter tool (Increment 4, revised scope)
- Session initialization and context loading (Increment 6)
- Agents can find and load relevant notes

**Phase 4: Maintenance (Increments 7–8)**
- Index and constraint maintenance
- Linting and integrity checks
- Vault stays healthy as it grows

**Phase 5: Operations (Increments 10–12)**
- Backup and recovery
- Semantic search
- Vault archival
- Vault is production-ready

**Phase 6: Stabilization (Increments 13–14)**
- Global vault init (`--global` flag)
- Template customization with config directory
- Vault is user-customizable
