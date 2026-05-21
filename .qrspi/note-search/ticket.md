# Increment 4: Note Search (Read Protocol)

## Goal

Implement the `agent-memory search` subcommand so agents can find relevant notes without loading everything.

## What becomes possible after this increment

- Agents can search for notes by frontmatter fields (project, domain, epistemic-type, status)
- Two-phase retrieval: Phase 1 returns titles + frontmatter only; Phase 2 returns full bodies for selected slugs
- Phase 2 is capped at 10 notes per invocation
- Only `status: verified` notes are returned by default
- Tag alias expansion reads from `_meta/tag-taxonomy.md`
- Agents can find existing notes before writing new ones (search-before-write)

## Constraints

- Single binary with subcommands (ADR-0001)
- Dual output: human-readable default, JSON with `--json`
- Ginkgo/gomega BDD tests
- Design doc defines behaviour/strategy; package layout is implementation detail
- Vault discovery reuses `internal/vault/discover.go` (already implemented in increment 3)

## From the design doc (AGENT_MEMORY_DESIGN.md)

### Two-phase retrieval (§4.3)

Phase 1 returns titles + frontmatter for all matches (status: verified only, tag aliases auto-expanded). Phase 2 takes a list of slugs and returns bodies, capped at 10. The two-phase split is enforced by the subcommand; an agent cannot accidentally request all bodies at once.

### CLI interface

```bash
agent-memory search <query> [--type=<...>] [--project=<...>] [--phase=1|2] [--slugs=<a,b>]
```

### From the roadmap

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
