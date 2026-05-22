# Increments 6+7: Session Context & Index Maintenance

## Summary

Implement two CLI subcommands that ship together:

1. `agent-memory context` — bundles vault knowledge into a session-start payload for agents
2. `agent-memory reindex` — scans all notes and regenerates project/domain indices and constraints summary

These are co-dependent: `context` loads indices that `reindex` generates.

## Increment 6: `agent-memory context`

### What it does
- Bundles Core tier: writing-protocol, constraints-summary, log tail (~20 entries)
- Bundles Index tier: `_index-{project}.md`, `_index-{domain}.md` for requested project/domains
- Detects stale notes (review-by < today)
- Tracks token budget, defers domain indices if total > 2000 tokens
- Returns JSON on stdout

### CLI interface
```
agent-memory context [--project=<slug>] [--domain=<domain>...] [--json]
```

### Token targets
- Core tier: < 600 tokens combined
- Index tier: < 300 tokens per index
- Total budget gate: 2000 tokens

## Increment 7: `agent-memory reindex`

### What it does
- Scans all notes in `notes/` and `_inbox/`
- Generates `_index-{project}.md` from frontmatter scan
- Generates `_index-{domain}.md` from frontmatter scan
- Generates `_meta/constraints-summary.md` from all `constraint` notes
- Detects orphaned notes (not referenced from any index)
- Writes log entry for reindex operations
- Idempotent: safe to run multiple times

### CLI interface
```
agent-memory reindex [--json]
```

## Verification (from roadmap)

### Context tests
- Core tier loading
- Index tier loading with project/domain filters
- Staleness detection
- Budget tracking and gating
- JSON output format

### Reindex tests
- Index generation from notes with different projects/domains
- Constraints summary generation
- Idempotency
- Orphan detection
- Index updates after new notes

## Design references
- `docs/AGENT_MEMORY_DESIGN.md` §5.8 (session initialization), §9.6 (reindex)
- `docs/ROADMAP.md` Increments 6 and 7
