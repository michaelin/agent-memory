<worktree_artifact feature="note-writing">

## Feature Branch

- **Branch**: `note-writing`
- **Base**: `main` at `89b30de`

## Worktrees

| Task Group | Path | Branch | Slices | Status |
|---|---|---|---|---|
| foundation | `.worktrees/note-writing-foundation` | `note-writing-foundation` | 1 (Serialize+Slug), 2 (Vault Discovery) | pending |
| similarity | `.worktrees/note-writing-similarity` | `note-writing-similarity` | 3 (Similarity+Wikilinks), 4 (Harness+NF001) | pending |
| protocol | `.worktrees/note-writing-protocol` | `note-writing-protocol` | 7 (Writing Protocol v3) | pending |

## Sequential (on feature branch after merge)

| Slices | Branch | Status |
|---|---|---|
| 5 (Write Pipeline), 6 (CLI + Integration) | `note-writing` (after consolidation) | pending |

## Task Tree

### Session 1 — Parallel (3 worktrees)

#### Worktree: foundation
- [ ] Task 1.1 — Implement Serialize
- [ ] Task 1.2 — Implement Slug
- [ ] Task 1.3 — Write Serialize+Slug tests
- [ ] Slice 1 checkpoint
- [ ] Task 2.1 — Implement Discover
- [ ] Task 2.2 — Write Discover tests
- [ ] Slice 2 checkpoint

#### Worktree: similarity
- [ ] Task 3.1 — Implement NormalizeTokens and Jaccard
- [ ] Task 3.2 — Implement ExtractWikilinks
- [ ] Task 3.3 — Write Similarity+Wikilinks tests
- [ ] Slice 3 checkpoint
- [ ] Task 4.1 — Implement DetectSourceAgent
- [ ] Task 4.2 — Relax NF001 for source-agent
- [ ] Task 4.3 — Write DetectSourceAgent tests
- [ ] Slice 4 checkpoint

#### Worktree: protocol
- [ ] Task 7.1 — Fix enum values
- [ ] Task 7.2 — Add v3 write-note instructions
- [ ] Task 7.3 — Update tests if needed
- [ ] Slice 7 checkpoint

### Session 2 — Sequential (feature branch, after worktree consolidation)

- [ ] Task 5.1 — Implement Write function
- [ ] Task 5.2 — Write Write tests
- [ ] Slice 5 checkpoint
- [ ] Task 6.1 — Implement CLI command
- [ ] Task 6.2 — Register subcommand
- [ ] Task 6.3 — Update instructions text
- [ ] Task 6.4 — Integration tests
- [ ] Slice 6 checkpoint

## Session Boundaries

- **Session 1 end**: After all 3 worktrees complete slices 1–4 + 7. Consolidate worktrees into feature branch.
- **Session 2 end**: After slices 5–6 complete on feature branch. Run final verification.

</worktree_artifact>
