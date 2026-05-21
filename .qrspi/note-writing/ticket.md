# Increment 3: Note Writing (Write Protocol)

**Goal:** Implement the static tool so agents can write notes to the vault.

**What becomes possible after this increment:**
- Agents can write notes to the vault via `agent-memory write-note`
- Notes are validated and placed in `_inbox/`
- The similarity check prevents obvious duplicates
- Agents understand how to compose notes with proper evidence and implications

## Scope

### Subcommand: `agent-memory write-note`
Per ADR-0001, this is a subcommand of the single binary, not a separate tool.

- Similarity check (Jaccard on normalized word tokens, threshold TBD during QRSPI)
- Frontmatter assembly (created, updated, status, TTL assignment)
- Inbox placement (`_inbox/{YYYY-MM-DD}-{slug}.md`)
- Log entry writing (`_meta/log.md`)
- Refusal behavior on similarity hit (return candidates as JSON)
- Wikilink validation (warn on unresolved links, but don't block write)

### Writing Protocol (v3 — Updated Instructions)
- Updated `_meta/writing-protocol.md` with write-note usage instructions
- Explains how to invoke `agent-memory write-note`
- Explains what the tool expects (frontmatter + body via stdin or file)
- Explains what the tool returns (success/failure, warnings)

### Verification (Integration Tests)
- **Write a note:** Run `write-note`, verify note lands in `_inbox/`
- **Duplicate detection:** Write a similar note, verify tool detects it and returns candidates
- **Frontmatter assembly:** Verify frontmatter is correctly assembled (created, updated, status)
- **Log entry:** Verify log entry is written
- **Wikilink warnings:** Write a note with unresolved wikilinks, verify tool warns but doesn't block
- **Validation:** Verify written note passes `lint-note`
- **Slug generation:** Verify filename follows `{YYYY-MM-DD}-{slug}.md` pattern

## Notes
- The roadmap mentions a `write-memory` skill (LLM-powered composition). That's a separate concern — this increment focuses on the deterministic tool that accepts a complete note and writes it to the vault. The skill can be built on top later.
- ADR-0001 applies: `memory-write-mechanics` becomes `agent-memory write-note`
