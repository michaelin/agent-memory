# Agent Memory: Incremental Implementation Roadmap

**Status:** Architectural questions resolved. Ready for QRSPI process.

Each increment is a complete vertical slice: design → research → structure → plan → work → review. Each increment is independently valuable and individually verifiable via comprehensive integration tests.

---

## Increment 1: Vault Scaffolding & Configuration

**Goal:** Establish the vault structure and configuration layer so all subsequent increments have a place to write.

**What becomes possible after this increment:**
- Agents can discover the vault location
- The vault structure is initialized and idempotent
- The writing protocol document exists (though it's mostly empty at this stage)
- Humans can browse the vault in Obsidian

**Scope:**

### Configuration Resolution
- `AGENT_MEMORY_VAULT` environment variable (highest priority)
- `~/.config/agent-memory/config.json` (XDG_CONFIG_HOME if set, else `~/.config`)
- Fallback to `$HOME/.local/share/agent-memory` (XDG_DATA_HOME if set, else `~/.local/share`)
- Config file format:
  ```json
  {
    "vault": "/path/to/vault"
  }
  ```

### Vault Initialization (`agent-memory init`)
- **Idempotency:** Running `init` multiple times on the same vault path is safe
  - If vault already exists, verify structure is correct and exit 0
  - If vault exists but is corrupted (missing required directories), repair it
  - If vault exists but is from an older version, upgrade it (no-op for v1)
- Create vault directory structure:
  ```
  AgentMemory/
  ├── AGENTS.md                    ← Points to writing protocol
  ├── _meta/
  │   ├── writing-protocol.md      ← Rules agents MUST follow (mostly empty in v1)
  │   ├── tag-taxonomy.md          ← Empty in v1
  │   ├── constraints-summary.md   ← Empty in v1
  │   ├── status-lifecycle.md      ← Lifecycle stages (static template)
  │   └── log.md                   ← Append-only log (empty at init)
  ├── _inbox/                      ← All agent writes land here first
  ├── _contested/                  ← Notes with unresolved contradictions (empty at init)
  └── notes/                       ← All promoted notes, flat
  ```
- Seed `_meta/` files from embedded templates (no network, no external files)
- Create `AGENTS.md` at vault root pointing at `_meta/writing-protocol.md`
- Write configuration file to `~/.config/agent-memory/config.json` (or env var location)
- Log initialization to `_meta/log.md`: `## [YYYY-MM-DD HH:MM] init | vault initialized`

### Writing Protocol (v1 — Minimal)
- Document exists at `_meta/writing-protocol.md`
- Content: "This vault is for agent memory. Writing is not yet enabled. See AGENTS.md for current capabilities."
- Will be expanded in Increment 2

### Verification (Integration Tests)
- **Idempotency:** Run `agent-memory init` twice on same path, verify no errors and vault state unchanged
- **Config resolution:** Test all three config sources (env var, config file, default)
- **XDG support:** Test with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` set
- **Fallback:** Test that default path is used when no config exists
- **Vault structure:** Verify all required directories exist after init
- **Seed files:** Verify all `_meta/` files exist and contain expected content
- **AGENTS.md:** Verify it points to `_meta/writing-protocol.md`
- **Log entry:** Verify init is logged to `_meta/log.md`
- **Repair:** Corrupt vault (delete a directory), run init again, verify repair

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

## Increment 4: Note Search (Read Protocol)

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

## Increment 5: Promotion & Lifecycle (Inbox Management)

**Goal:** Implement the promotion pipeline so notes move from inbox to verified state automatically.

**What becomes possible after this increment:**
- Notes automatically graduate from `_inbox/` to `notes/` based on their epistemic type
- Observations promote after TTL expiry
- Patterns promote when 2+ observations corroborate them
- Assumptions are flagged for re-verification
- Constraints and decisions are held for human review
- Wikilink validation blocks promotion of broken notes

**Scope:**

### Static Tool: `memory-promote`
- Scans `_inbox/` for notes ready to promote
- Promotion logic by epistemic type:
  - `observation`: promote on TTL expiry (or immediately with `--eager`)
  - `pattern`: promote when 2+ corroborating observations exist (shared tags/domain)
  - `assumption`: never promote; flag on TTL expiry for re-verification
  - `constraint`: hold with `requires-human-review: true`
  - `decision`: hold with `requires-human-review: true`
  - `synthesis`: not produced via inbox; not handled here
- Wikilink validation (block promotion if unresolved links exist)
- Log entry writing for all promotions
- Idempotency: safe to run multiple times

### Verification (Integration Tests)
- **TTL-based promotion:** Write observation, mock time passage, run promote, verify promotion
- **Corroboration-based promotion:** Write 2 observations with shared tags, run promote, verify pattern promotion
- **Assumption flagging:** Write assumption, run promote, verify it stays in inbox with flag
- **Constraint gating:** Write constraint, run promote, verify it stays in inbox with `requires-human-review: true`
- **Decision gating:** Write decision, run promote, verify it stays in inbox with `requires-human-review: true`
- **Wikilink blocking:** Write note with unresolved links, run promote, verify it's blocked
- **Idempotency:** Run promote twice, verify same result

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

## Increment 9: Librarian Agent (Escalation Handler)

**Goal:** Implement the Librarian agent to handle high-stakes decisions and maintenance tasks.

**What becomes possible after this increment:**
- High-stakes notes (constraints, decisions) can be reviewed and promoted
- Contested notes can be resolved
- Untagged notes can be tagged
- Pattern candidates can be detected
- Synthesis pages can be drafted

**Scope:**

### Librarian Agent Definition
- Standalone global agent (e.g., `~/.config/opencode/agent/librarian.md`)
- Permissions:
  - Read: vault path
  - Write: vault path only
  - Bash: `memory-*` and `lint-*` binaries on vault directory
  - Task: deny (leaf subagent)

### Librarian Tasks
- **High-stakes review:** Review notes with `requires-human-review: true`
  - Read the note
  - Search for related existing notes
  - Recommend promote/reject to human
  - Human confirms, Librarian executes
- **Contested resolution:** Review notes in `_contested/`
  - Read both conflicting notes
  - Reason about which is correct (or if both are valid)
  - Recommend resolution to human
  - Human confirms, Librarian executes
- **Tagging:** Assign tags to untagged notes
  - Read untagged notes
  - Assign tags based on content and domain
  - Use `memory-tag` tool to update tags
- **Pattern detection:** Detect pattern candidates from observations
  - Read recent observation notes
  - Identify clusters of corroborating claims
  - Draft pattern notes for human review
- **Synthesis drafting:** Draft synthesis pages
  - Read contributing notes
  - Compose synthesis prose
  - Use `memory-synthesize` tool to create/update page

### Static Tools (Librarian-only)
- `memory-tag`: Tag taxonomy management
  - Assign tags to notes
  - Accept new tags into taxonomy
  - Reject tags with optional redirect
- `memory-curate`: Structure high-stakes inbox items
  - Read note with `requires-human-review: true`
  - Search for related notes
  - Produce promote/reject recommendation
- `memory-synthesize`: Generate synthesis page scaffolds
  - Takes entity slug or tag
  - Gathers contributing notes
  - Generates scaffold (deterministic parts)
  - In `--draft` mode, leaves prose section blank for Librarian

### Verification (Integration Tests)
- **High-stakes review:** Create constraint note, verify Librarian can review and promote
- **Contested resolution:** Create two conflicting notes, verify Librarian can resolve
- **Tagging:** Create untagged notes, verify Librarian can tag them
- **Pattern detection:** Create observation notes, verify Librarian can detect patterns
- **Synthesis drafting:** Create notes for entity, verify Librarian can draft synthesis

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

**Phase 1: Foundation (Increments 1-2)**
- Vault scaffolding and configuration
- Note format and validation
- Ready for agents to write notes

**Phase 2: Core Functionality (Increments 3-6)**
- Note writing (skill + tool)
- Note search (skill + tool)
- Promotion and lifecycle
- Session initialization
- Agents can now use the vault for reading and writing

**Phase 3: Maintenance (Increments 7-8)**
- Index and constraint maintenance
- Linting and integrity checks
- Vault stays healthy as it grows

**Phase 4: Intelligence (Increment 9)**
- Librarian agent
- High-stakes review and resolution
- Pattern detection and synthesis
- Vault becomes smarter over time

**Phase 5: Operations (Increments 10-12)**
- Backup and recovery
- Semantic search
- Vault archival
- Vault is production-ready
