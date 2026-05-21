<worktree feature="librarian-promotion">

## Git Worktrees

| Task Group | Worktree Path | Branch | Slices |
|---|---|---|---|
| wave1 | .worktrees/librarian-promotion-wave1 | librarian-promotion-wave1 | slice-1, slice-2, slice-5 |
| wave2 | .worktrees/librarian-promotion-wave2 | librarian-promotion-wave2 | slice-3, slice-4 |
| wave3 | .worktrees/librarian-promotion-wave3 | librarian-promotion-wave3 | slice-6, slice-7, slice-8, slice-9 |

**Wave dependencies**: wave2 depends on wave1 completion. wave3 depends on wave2 completion.

---

<slice name="slice-1-export-helpers">

## Slice 1: Export Helpers (Wave 1 — parallel)

<task id="s1-t1" status="pending">
**Description**: Add goldmark-wikilink dependency
**Files**: `go.mod`, `go.sum`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s1-t2" status="pending">
**Description**: Rewrite ExtractWikilinks with goldmark-wikilink extension
**Files**: `internal/note/wikilinks.go`
**Context cost**: Small
**Depends on**: s1-t1
</task>

<task id="s1-t3" status="pending">
**Description**: Export AppendLog with action parameter
**Files**: `internal/note/write.go`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s1-t4" status="pending">
**Description**: Rewrite and export ResolveWikilinks (Obsidian-style)
**Files**: `internal/note/write.go`
**Context cost**: Medium
**Depends on**: s1-t2
</task>

<task id="s1-t5" status="pending">
**Description**: Export FindSimilarNotes
**Files**: `internal/note/write.go`
**Context cost**: Small
**Depends on**: none
</task>

</slice>

<slice name="slice-2-frontmatter-changes">

## Slice 2: Frontmatter Changes (Wave 1 — parallel)

<task id="s2-t1" status="pending">
**Description**: Consolidate EpistemicType yaml tag, remove UpdateType, add SupersededBy
**Files**: `internal/note/note.go`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s2-t2" status="pending">
**Description**: Update test fixtures for type: yaml key
**Files**: All test files with YAML fixtures
**Context cost**: Medium
**Depends on**: s2-t1
</task>

<task id="s2-t3" status="pending">
**Description**: Verify lint and serialize handle new fields
**Files**: `internal/note/lint.go`, `internal/note/serialize.go`
**Context cost**: Small
**Depends on**: s2-t1
</task>

</slice>

<slice name="slice-5-vault-scaffold">

## Slice 5: Vault Scaffold (Wave 1 — parallel)

<task id="s5-t1" status="pending">
**Description**: Add _deprecated/ and _meta/templates/ vault structure entries
**Files**: `internal/vault/structure.go`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s5-t2" status="pending">
**Description**: Create embedded Librarian templates
**Files**: `internal/vault/templates/librarian-agent.md`, `internal/vault/templates/librarian-skill.md`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s5-t3" status="pending">
**Description**: Update vault tests for new entries
**Files**: `internal/vault/structure_test.go`, `internal/vault/init_test.go`
**Context cost**: Small
**Depends on**: s5-t1, s5-t2
</task>

</slice>

<session_boundary reason="End of Wave 1 — merge wave1 worktree, verify all 3 slices pass before Wave 2"/>

<slice name="slice-3-promote-core">

## Slice 3: Promote Core (Wave 2 — parallel)

<task id="s3-t1" status="pending">
**Description**: Implement FindBySlug
**Files**: `internal/note/promote.go`
**Context cost**: Small
**Depends on**: s1-t4 (ResolveWikilinks), s2-t1 (frontmatter)
</task>

<task id="s3-t2" status="pending">
**Description**: Implement Promote function
**Files**: `internal/note/promote.go`
**Context cost**: Medium
**Depends on**: s3-t1, s1-t3 (AppendLog), s1-t4 (ResolveWikilinks)
</task>

<task id="s3-t3" status="pending">
**Description**: Write promote unit tests
**Files**: `internal/note/promote_test.go`
**Context cost**: Medium
**Depends on**: s3-t2
</task>

</slice>

<slice name="slice-4-deprecate-core">

## Slice 4: Deprecate Core (Wave 2 — parallel)

<task id="s4-t1" status="pending">
**Description**: Implement Deprecate function
**Files**: `internal/note/deprecate.go`
**Context cost**: Small
**Depends on**: s1-t3 (AppendLog), s2-t1 (SupersededBy field)
</task>

<task id="s4-t2" status="pending">
**Description**: Write deprecate unit tests
**Files**: `internal/note/deprecate_test.go`
**Context cost**: Small
**Depends on**: s4-t1
</task>

</slice>

<session_boundary reason="End of Wave 2 — merge wave2 worktree, verify promote+deprecate work before CLI layer"/>

<slice name="slice-6-cli-promote">

## Slice 6: CLI Promote (Wave 3 — sequential)

<task id="s6-t1" status="pending">
**Description**: Implement promote subcommand
**Files**: `internal/cli/promote.go`, `internal/cli/root.go`
**Context cost**: Small
**Depends on**: s3-t2 (Promote function)
</task>

<task id="s6-t2" status="pending">
**Description**: Promote integration test
**Files**: `test/integration/promote_test.go`
**Context cost**: Medium
**Depends on**: s6-t1
</task>

</slice>

<slice name="slice-7-cli-deprecate">

## Slice 7: CLI Deprecate (Wave 3 — sequential)

<task id="s7-t1" status="pending">
**Description**: Implement deprecate subcommand
**Files**: `internal/cli/deprecate.go`, `internal/cli/root.go`
**Context cost**: Small
**Depends on**: s4-t1 (Deprecate function)
</task>

<task id="s7-t2" status="pending">
**Description**: Deprecate integration test
**Files**: `test/integration/deprecate_test.go`
**Context cost**: Medium
**Depends on**: s7-t1
</task>

</slice>

<slice name="slice-8-instructions-update">

## Slice 8: Instructions Update (Wave 3 — sequential)

<task id="s8-t1" status="pending">
**Description**: Update instructions text with Librarian workflow
**Files**: `internal/cli/instructions.go`
**Context cost**: Small
**Depends on**: s6-t1, s7-t1
</task>

<task id="s8-t2" status="pending">
**Description**: Update instructions integration test
**Files**: `test/integration/instructions_test.go`
**Context cost**: Small
**Depends on**: s8-t1
</task>

</slice>

<slice name="slice-9-integration-tests">

## Slice 9: Full Workflow Integration Tests (Wave 3 — sequential)

<task id="s9-t1" status="pending">
**Description**: Full lifecycle integration test (init → write → promote → deprecate → promote replacement)
**Files**: `test/integration/promote_test.go` or `test/integration/deprecate_test.go`
**Context cost**: Medium
**Depends on**: s6-t2, s7-t2
</task>

</slice>

<session_boundary reason="End of Wave 3 — all implementation complete, run final verification"/>

</worktree>

<execution_order>

## Execution Order

### Wave 1 (parallel — all in wave1 worktree)
Slices 1, 2, 5 are independent and can be implemented in any order within the same worktree session.

1. **s1-t1** → **s1-t2** → **s1-t4** (wikilink chain)
2. **s1-t3** (AppendLog — independent)
3. **s1-t5** (FindSimilarNotes — independent)
4. **s2-t1** → **s2-t2** → **s2-t3** (frontmatter chain)
5. **s5-t1**, **s5-t2** → **s5-t3** (vault chain)

### Wave 2 (parallel — in wave2 worktree, after wave1 merged)
Slices 3, 4 are independent of each other but depend on wave1.

6. **s3-t1** → **s3-t2** → **s3-t3** (promote chain)
7. **s4-t1** → **s4-t2** (deprecate chain)

### Wave 3 (sequential — in wave3 worktree, after wave2 merged)
Slices 6–9 are sequential.

8. **s6-t1** → **s6-t2** (CLI promote)
9. **s7-t1** → **s7-t2** (CLI deprecate)
10. **s8-t1** → **s8-t2** (instructions)
11. **s9-t1** (full lifecycle test)

</execution_order>

<progress>

## Progress Tracking

| Task ID | Description | Status | Completed |
|---------|-------------|--------|-----------|
| s1-t1 | Add goldmark-wikilink dependency | pending | — |
| s1-t2 | Rewrite ExtractWikilinks | pending | — |
| s1-t3 | Export AppendLog | pending | — |
| s1-t4 | Rewrite and export ResolveWikilinks | pending | — |
| s1-t5 | Export FindSimilarNotes | pending | — |
| s2-t1 | Consolidate EpistemicType, add SupersededBy | pending | — |
| s2-t2 | Update test fixtures | pending | — |
| s2-t3 | Verify lint and serialize | pending | — |
| s5-t1 | Add vault structure entries | pending | — |
| s5-t2 | Create Librarian templates | pending | — |
| s5-t3 | Update vault tests | pending | — |
| s3-t1 | Implement FindBySlug | pending | — |
| s3-t2 | Implement Promote | pending | — |
| s3-t3 | Write promote unit tests | pending | — |
| s4-t1 | Implement Deprecate | pending | — |
| s4-t2 | Write deprecate unit tests | pending | — |
| s6-t1 | Implement promote subcommand | pending | — |
| s6-t2 | Promote integration test | pending | — |
| s7-t1 | Implement deprecate subcommand | pending | — |
| s7-t2 | Deprecate integration test | pending | — |
| s8-t1 | Update instructions text | pending | — |
| s8-t2 | Update instructions integration test | pending | — |
| s9-t1 | Full lifecycle integration test | pending | — |

**Overall**: 0 / 23 tasks complete

</progress>
