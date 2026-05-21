# Increment 5: Librarian Agent & Promotion Pipeline

## Goal

Implement the Librarian as a skill-triggered agent that validates, classifies, promotes, deduplicates, and maintains inbox notes. This is the critical path -- without the Librarian, all notes stay in `_inbox/` forever.

## What becomes possible after this increment

- Notes move from `_inbox/` to `notes/` via Librarian promotion
- Constraints and decisions require human confirmation before promotion
- Duplicate notes are detected and handled
- Outdated assumptions are removed when contradicted by new notes
- Pattern notes require 2+ corroborating references
- Synthesis notes can be created by the Librarian

## Scope

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
  - Promotion by epistemic type rules (see design doc section 5.5)
  - Human confirmation flow for constraints/decisions (via agent tool call)
  - Deprecation of replaced notes (forward link)
  - Assumption outdating when contradicted by new notes

### Verification (Integration Tests)
- Observation promotion: Write observation, run Librarian, verify promotion to `notes/`
- Pattern promotion: Write pattern referencing 2+ notes, verify promotion
- Pattern rejection: Write pattern with <2 references, verify stays in inbox
- Constraint gating: Write constraint, verify Librarian requests human confirmation
- Decision gating: Write decision, verify same confirmation flow
- Assumption promotion: Write assumption, verify promotion (stays as assumption type)
- Duplicate detection: Write duplicate note, verify Librarian detects and handles it
- Wikilink blocking: Write note with unresolved links, verify promotion blocked
- Replacement flow: Write replacement note, verify old note deprecated with forward link
- Lint failure: Write malformed note, verify Librarian rejects it
- Idempotency: Run Librarian twice, verify same result

## Key design references
- `docs/AGENT_MEMORY_DESIGN.md` section 5.5 (promotion rules), 5.6 (staleness), 5.8 (Librarian agent)
- `docs/AGENT_MEMORY_DESIGN.md` section 9.4 (`agent-memory promote` subcommand)
- `docs/ROADMAP.md` Increment 5
