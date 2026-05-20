# Agent Memory: Design Review & Roadmap Summary

## What Was Done

You asked for a thorough review of the AGENT_MEMORY_DESIGN.md to identify:
1. How to split the setup into small, individually valuable increments
2. How to build incrementally with partial intermediate implementations
3. How to verify each increment with comprehensive integration tests

## Key Findings

### Architectural Holes in Original Design
The design was intentionally vague on several points (as you noted). The most critical areas for QRSPI exploration:

1. **Similarity check algorithm** — Threshold value, domain scoping, false positive handling
2. **Frontmatter parsing** — Malformed YAML handling, field validation rules
3. **TTL and staleness** — Timezone handling, date comparison logic
4. **Promotion logic** — Race conditions, concurrent writes, atomic operations
5. **Librarian invocation** — Specific trigger thresholds, invocation mechanism
6. **Wikilink resolution** — Circular links, deprecated note links, contested note links
7. **Vault backup/recovery** — Format, retention policy, recovery procedure
8. **Synthesis page ownership** — Conflict handling, versioning, refresh logic
9. **Vault size limits** — Search performance degradation, archival strategy
10. **Idempotency** — All tools must be safe to run multiple times

### Architectural Decisions Made

**Q1: Subagent vs Skill for Note Writing?**
- **Decision:** Skill is sufficient and better
- **Reasoning:** A skill encapsulates the logic, is reusable across agents, and can delegate mechanical work to static tools. A subagent adds unnecessary latency and complexity.
- **Implementation:** `write-memory` skill calls `memory-write-mechanics` tool for similarity check and frontmatter assembly

**Q2: Tool vs Skill vs Direct Access for Note Search?**
- **Decision:** Skill that wraps a tool
- **Reasoning:** The skill hides the two-phase retrieval logic from the agent. Agent just says "find notes about X" and gets results. The skill can evolve independently and be ported to different frameworks.
- **Implementation:** `search-memory` skill calls `memory-search-mechanics` tool internally

**Idempotency as First-Class Requirement**
- All tools must be safe to run multiple times
- `agent-memory init` verifies and repairs vault structure
- `memory-promote`, `memory-reindex`, `agent-memory backup` are all idempotent

**XDG Folder Structure Support**
- Respects `XDG_CONFIG_HOME` and `XDG_DATA_HOME`
- Falls back to `$HOME/.config` and `$HOME/.local/share`
- Environment variable override: `AGENT_MEMORY_VAULT`

**Writing Protocol Evolution**
- Increment 1: Minimal (writing not yet enabled)
- Increment 2: Agent instructions (how to write notes)
- Increment 3: Skill instructions (how to use the skill)

## Incremental Implementation Roadmap

The design has been split into **12 increments**, organized into **5 phases**:

### Phase 1: Foundation (Increments 1-2)
1. **Vault Scaffolding & Configuration** — Idempotent init, XDG support, minimal writing protocol
2. **Note Format & Frontmatter Parsing** — YAML validation, `lint-note` binary, agent instructions

### Phase 2: Core Functionality (Increments 3-6)
3. **Note Writing** — `write-memory` skill + `memory-write-mechanics` tool
4. **Note Search** — `search-memory` skill + `memory-search-mechanics` tool
5. **Promotion & Lifecycle** — `memory-promote` tool, TTL-based and corroboration-based promotion
6. **Session Initialization** — `memory-context` tool + `load-memory-context` skill

### Phase 3: Maintenance (Increments 7-8)
7. **Index & Constraint Maintenance** — `memory-reindex` tool, orphan detection
8. **Linting & Integrity** — `lint-vault` tool, comprehensive vault checks

### Phase 4: Intelligence (Increment 9)
9. **Librarian Agent** — High-stakes review, contested resolution, tagging, pattern detection, synthesis

### Phase 5: Operations (Increments 10-12)
10. **Backup & Recovery** — `agent-memory backup` and `agent-memory restore`
11. **Semantic Search** — Enhancement to `search-memory` skill
12. **Vault Archival** — `memory-archive` tool for old notes

## Each Increment Is Independently Valuable

- **Increment 1** alone: Agents can discover vault, humans can browse it in Obsidian
- **Increment 2** alone: Notes can be validated for structural correctness
- **Increment 3** alone: Agents can write notes with duplicate detection
- **Increment 4** alone: Agents can search for existing notes
- **Increment 5** alone: Notes automatically graduate from inbox to verified
- **Increment 6** alone: Agents can load relevant context at session start
- **Increment 7** alone: Indices stay current automatically
- **Increment 8** alone: Vault integrity is verifiable
- **Increment 9** alone: High-stakes decisions can be reviewed
- **Increment 10** alone: Vault data is durable
- **Increment 11** alone: Better note discovery via semantic search
- **Increment 12** alone: Vault performance is maintained as it grows

## Verification Strategy

Each increment has comprehensive integration tests covering:
- **Happy path:** Normal usage works
- **Edge cases:** Boundary conditions handled correctly
- **Error handling:** Invalid input rejected with clear errors
- **Idempotency:** Running twice produces same result
- **Concurrency:** Multiple agents writing simultaneously don't corrupt vault
- **Backward compatibility:** New increments don't break previous ones

## Next Steps

Each increment will be the basis of a full **QRSPI process**:

1. **Q (Questions):** What do we need to know?
2. **R (Research):** Gather objective facts
3. **D (Design):** Discuss architectural options
4. **S (Structure):** Define implementation structure
5. **P (Plan):** Create tactical implementation plan
6. **W (Work Tree):** Organize work into manageable pieces
7. **I (Implement):** Write code
8. **PR (Review):** Prepare for human review

## Files Created

- **ROADMAP.md** — Complete incremental implementation roadmap with all 12 increments
- **This summary** — Overview of decisions and findings

## Recommended First Step

Start with **Increment 1 (Vault Scaffolding & Configuration)** using the QRSPI process:
- Q phase will identify what we need to know about Go module structure, XDG conventions, etc.
- R phase will gather facts about existing implementations
- D phase will discuss architectural options for idempotency, config resolution, etc.
- S phase will define the Go module structure and package layout
- P phase will create the implementation plan
- W phase will organize work into manageable pieces
- I phase will write the code
- PR phase will prepare for review

This gives you a solid foundation for all subsequent increments.
