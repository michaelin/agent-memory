<worktree feature="note-format-frontmatter">

<slice name="slice-1-parse">

## Slice 1: Note Parsing

<task id="s1-t1" status="pending">
**Description**: Create note package with `Note`, `Frontmatter` types
**Files**: `internal/note/note.go`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s1-t2" status="pending">
**Description**: Implement `Parse(content []byte) (*Note, error)`
**Files**: `internal/note/note.go`
**Context cost**: Small
**Depends on**: s1-t1
</task>

<task id="s1-t3" status="pending">
**Description**: Create ginkgo test suite and Parse tests
**Files**: `internal/note/note_suite_test.go`, `internal/note/note_test.go`
**Context cost**: Small
**Depends on**: s1-t2
</task>

</slice>

<session_boundary reason="End of Slice 1 — verify Parse works before building rules on top"/>

<slice name="slice-2-rules-engine">

## Slice 2: Rule Engine + NF001

<task id="s2-t1" status="pending">
**Description**: Define rule engine types (`Rule`, `LintError`, `LintResult`, `Lint()`)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s1-t3
</task>

<task id="s2-t2" status="pending">
**Description**: Implement NF001 (required fields present)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t1
</task>

<task id="s2-t3" status="pending">
**Description**: Write NF001 tests
**Files**: `internal/note/lint_test.go`
**Context cost**: Small
**Depends on**: s2-t2
</task>

</slice>

<session_boundary reason="End of Slice 2 — verify rule engine + NF001 before adding remaining rules"/>

<slice name="slice-3-validation-rules">

## Slice 3: Validation Rules NF002–NF007

<task id="s3-t1" status="pending">
**Description**: Implement NF002 (date format validation)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t2" status="pending">
**Description**: Implement NF003 (enum values)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t3" status="pending">
**Description**: Implement NF004 (conditional project requirement)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t4" status="pending">
**Description**: Implement NF005 (body sections)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t5" status="pending">
**Description**: Implement NF006 (placeholder detection)
**Files**: `internal/note/placeholder.go`, `internal/note/placeholder_test.go`, `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t6" status="pending">
**Description**: Implement NF007 (domain list validation)
**Files**: `internal/note/lint.go`
**Context cost**: Small
**Depends on**: s2-t3
</task>

<task id="s3-t7" status="pending">
**Description**: Write comprehensive rule tests for NF002–NF007
**Files**: `internal/note/lint_test.go`
**Context cost**: Medium
**Depends on**: s3-t1, s3-t2, s3-t3, s3-t4, s3-t5, s3-t6
</task>

</slice>

<session_boundary reason="End of Slice 3 — all rules implemented, verify before CLI wiring"/>

<slice name="slice-4-cli-subcommand">

## Slice 4: CLI Subcommand

<task id="s4-t1" status="pending">
**Description**: Implement `lint-note` subcommand with dual output
**Files**: `internal/cli/lint_note.go`
**Context cost**: Small
**Depends on**: s3-t7
</task>

<task id="s4-t2" status="pending">
**Description**: Register subcommand in root.go
**Files**: `internal/cli/root.go`
**Context cost**: Small
**Depends on**: s4-t1
</task>

<task id="s4-t3" status="pending">
**Description**: Manual smoke test
**Files**: none
**Context cost**: Small
**Depends on**: s4-t2
</task>

</slice>

<session_boundary reason="End of Slice 4 — CLI works end-to-end, verify before integration tests"/>

<slice name="slice-5-integration-tests">

## Slice 5: Integration Tests

<task id="s5-t1" status="pending">
**Description**: Add `WriteTestNote` helper to testutil
**Files**: `internal/testutil/helpers.go`
**Context cost**: Small
**Depends on**: s4-t3
</task>

<task id="s5-t2" status="pending">
**Description**: Write lint-note integration tests
**Files**: `test/integration/lint_note_test.go`
**Context cost**: Medium
**Depends on**: s5-t1
</task>

<task id="s5-t3" status="pending">
**Description**: Verify existing integration tests still pass
**Files**: `test/integration/init_test.go`
**Context cost**: Small
**Depends on**: s5-t2
</task>

</slice>

<session_boundary reason="End of Slice 5 — all tests pass, verify before writing protocol changes"/>

<slice name="slice-6-writing-protocol-v2">

## Slice 6: Writing Protocol v2

<task id="s6-t1" status="pending">
**Description**: Write v2 writing protocol template
**Files**: `internal/vault/templates/writing-protocol.md`
**Context cost**: Small
**Depends on**: s5-t3
</task>

<task id="s6-t2" status="pending">
**Description**: Update instructions subcommand
**Files**: `internal/cli/instructions.go`
**Context cost**: Small
**Depends on**: s6-t1
</task>

<task id="s6-t3" status="pending">
**Description**: Update affected tests
**Files**: `test/integration/init_test.go`, `internal/vault/structure_test.go`
**Context cost**: Small
**Depends on**: s6-t2
</task>

</slice>

</worktree>

<execution_order>

## Execution Order

All slices are sequential — each builds on the previous.

1. **s1-t1** → **s1-t2** → **s1-t3** (Slice 1: Parse)
2. *[Session boundary — verify Parse]*
3. **s2-t1** → **s2-t2** → **s2-t3** (Slice 2: Rules engine + NF001)
4. *[Session boundary — verify rule engine]*
5. **s3-t1** through **s3-t6** (parallel within slice) → **s3-t7** (Slice 3: Rules NF002–NF007)
6. *[Session boundary — verify all rules]*
7. **s4-t1** → **s4-t2** → **s4-t3** (Slice 4: CLI)
8. *[Session boundary — verify CLI]*
9. **s5-t1** → **s5-t2** → **s5-t3** (Slice 5: Integration tests)
10. *[Session boundary — verify integration tests]*
11. **s6-t1** → **s6-t2** → **s6-t3** (Slice 6: Writing protocol v2)

**Parallel opportunities**: Tasks s3-t1 through s3-t6 are independent of each other (all depend only on s2-t3). In practice, they share the same file so sequential is simpler.

</execution_order>

## Git Worktrees

| Task Group | Worktree Path | Branch | Slices |
|---|---|---|---|
| impl | .worktrees/note-format-frontmatter-impl | note-format-frontmatter-impl | slice-1 through slice-6 |

<progress>

## Progress Tracking

| Task ID | Description | Status | Completed |
|---------|-------------|--------|-----------|
| s1-t1 | Create note package with types | pending | — |
| s1-t2 | Implement Parse function | pending | — |
| s1-t3 | Create ginkgo test suite and Parse tests | pending | — |
| s2-t1 | Define rule engine types | pending | — |
| s2-t2 | Implement NF001 | pending | — |
| s2-t3 | Write NF001 tests | pending | — |
| s3-t1 | Implement NF002 (date format) | pending | — |
| s3-t2 | Implement NF003 (enum values) | pending | — |
| s3-t3 | Implement NF004 (conditional project) | pending | — |
| s3-t4 | Implement NF005 (body sections) | pending | — |
| s3-t5 | Implement NF006 (placeholder detection) | pending | — |
| s3-t6 | Implement NF007 (domain list) | pending | — |
| s3-t7 | Comprehensive rule tests NF002–NF007 | pending | — |
| s4-t1 | Implement lint-note subcommand | pending | — |
| s4-t2 | Register subcommand | pending | — |
| s4-t3 | Manual smoke test | pending | — |
| s5-t1 | Add WriteTestNote helper | pending | — |
| s5-t2 | Write lint-note integration tests | pending | — |
| s5-t3 | Verify existing integration tests | pending | — |
| s6-t1 | Write v2 writing protocol template | pending | — |
| s6-t2 | Update instructions subcommand | pending | — |
| s6-t3 | Update affected tests | pending | — |

**Overall**: 0 / 22 tasks complete

</progress>
