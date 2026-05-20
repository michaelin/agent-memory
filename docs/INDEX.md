# Agent Memory: Design Review & Implementation Planning

## Overview

This directory contains the complete design review and incremental implementation roadmap for the Agent Memory system.

## Documents

### Primary Documents

1. **ROADMAP.md** ⭐ START HERE
   - Complete incremental implementation roadmap
   - 12 increments organized into 5 phases
   - Each increment is independently valuable
   - Includes scope, verification tests, and what becomes possible

2. **ARCHITECTURAL-DECISION-SKILL-VS-SUBAGENT.md**
   - Detailed analysis of key architectural decisions
   - Answers: "Why skills instead of subagents?"
   - Explains the hybrid skill + tool approach
   - Applies to both note writing and note search

### Reference Documents

3. **AGENT_MEMORY_DESIGN.md** (canonical design)
   - Complete design document
   - Theoretical foundations
   - Prior art review
   - Core concepts
   - Resolved design decisions

## Quick Start

### If you want to understand the roadmap:
1. Read **ROADMAP.md** (complete incremental plan)

### If you want to understand the architectural decisions:
1. Read **ARCHITECTURAL-DECISION-SKILL-VS-SUBAGENT.md** (skill vs subagent)
2. Reference **AGENT_MEMORY_DESIGN.md** (original design context)

### If you want to start implementing:
1. Read **ROADMAP.md** (Increment 1 section)
2. Use QRSPI process for Increment 1
3. Each increment is a complete vertical slice

## Key Decisions

### Idempotency
All tools must be safe to run multiple times. This is a first-class requirement, not an afterthought.

### Vault Discovery (git-init model)
The vault is self-describing at `.agent-memory/` — analogous to `.git/`. No config file is required. Discovery order:
1. `AGENT_MEMORY_VAULT` env var (highest priority)
2. Walk up directories looking for `.agent-memory/` (project-local vault)
3. `~/.local/share/agent-memory` (user-global fallback)

### Skills + Tools Architecture
- **Skills** encapsulate agent-facing logic and teach calling agents
- **Tools** handle mechanical work and are deterministic/testable
- Skills delegate to tools for mechanical parts
- This gives flexibility of agents + determinism of tools

### Writing Protocol Evolution
- Increment 1: Minimal (writing not yet enabled)
- Increment 2: Agent instructions (how to write notes)
- Increment 3: Skill instructions (how to use the skill)

## Incremental Roadmap

### Phase 1: Foundation (Increments 1-2)
- Vault scaffolding & configuration
- Note format & frontmatter parsing

### Phase 2: Core Functionality (Increments 3-6)
- Note writing (skill + tool)
- Note search (skill + tool)
- Promotion & lifecycle
- Session initialization

### Phase 3: Maintenance (Increments 7-8)
- Index & constraint maintenance
- Linting & integrity

### Phase 4: Intelligence (Increment 9)
- Librarian agent

### Phase 5: Operations (Increments 10-12)
- Backup & recovery
- Semantic search
- Vault archival

## Each Increment Is Independently Valuable

After Increment 1: Agents can discover vault, humans can browse in Obsidian
After Increment 2: Notes can be validated for structural correctness
After Increment 3: Agents can write notes with duplicate detection
After Increment 4: Agents can search for existing notes
After Increment 5: Notes automatically graduate from inbox to verified
After Increment 6: Agents can load relevant context at session start
After Increment 7: Indices stay current automatically
After Increment 8: Vault integrity is verifiable
After Increment 9: High-stakes decisions can be reviewed
After Increment 10: Vault data is durable
After Increment 11: Better note discovery via semantic search
After Increment 12: Vault performance is maintained as it grows

## Verification Strategy

Each increment has comprehensive integration tests covering:
- Happy path (normal usage works)
- Edge cases (boundary conditions handled correctly)
- Error handling (invalid input rejected with clear errors)
- Idempotency (running twice produces same result)
- Concurrency (multiple agents writing simultaneously don't corrupt vault)
- Backward compatibility (new increments don't break previous ones)

## Next Steps

1. **Review ROADMAP.md** — Understand the complete incremental plan
2. **Review ARCHITECTURAL-DECISION-SKILL-VS-SUBAGENT.md** — Understand key decisions
3. **Start Increment 1 with QRSPI** — Use the full QRSPI process for the first increment
4. **Each increment is a complete vertical slice** — Design, research, structure, plan, work, review

## Questions Answered

✅ How to split the setup into small, individually valuable increments?
✅ How to build incrementally with partial intermediate implementations?
✅ How to verify each increment with comprehensive integration tests?
✅ Q1: What is the benefit of a subagent? Could a skill be sufficient?
✅ Q2: Is a tool the best approach for search, or should it be a skill?

## Status

**Increment 1 (Vault Scaffolding) complete.** The vault discovery and initialization model is implemented. Each remaining increment can be tackled as a complete vertical slice with design, research, structure, planning, work organization, implementation, and review phases.
