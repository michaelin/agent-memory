<worktree feature="vault-scaffolding">

<slice name="slice-1-project-skeleton">

## Slice 1: Project Skeleton

<task id="s1-t1" status="pending">
**Description**: Initialize Go module
**Files**: `go.mod`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s1-t2" status="pending">
**Description**: Create mise config
**Files**: `.mise.toml`
**Context cost**: Small
**Depends on**: none
</task>

<task id="s1-t3" status="pending">
**Description**: Create root cobra command
**Files**: `internal/cli/root.go`
**Context cost**: Small
**Depends on**: s1-t1
</task>

<task id="s1-t4" status="pending">
**Description**: Create main entry point
**Files**: `cmd/agent-memory/main.go`
**Context cost**: Small
**Depends on**: s1-t3
</task>

<task id="s1-t5" status="pending">
**Description**: Fetch dependencies
**Files**: `go.sum`
**Context cost**: Small
**Depends on**: s1-t4
</task>

</slice>

<session_boundary reason="End of Slice 1 — verify binary builds and runs before continuing"/>

<slice name="slice-2-vault-structure">

## Slice 2: Vault Structure Definition

<task id="s2-t1" status="pending">
**Description**: Define vault structure types and entries with embedded templates
**Files**: `internal/vault/structure.go`
**Context cost**: Small
**Depends on**: s1-t5
</task>

<task id="s2-t2" status="pending">
**Description**: Create seed template files
**Files**: `internal/vault/templates/writing-protocol.md`, `internal/vault/templates/tag-taxonomy.md`, `internal/vault/templates/constraints-summary.md`, `internal/vault/templates/status-lifecycle.md`, `internal/vault/templates/log.md`
**Context cost**: Medium
**Depends on**: s2-t1
</task>

<task id="s2-t3" status="pending">
**Description**: Unit tests for vault structure
**Files**: `internal/vault/structure_test.go`
**Context cost**: Small
**Depends on**: s2-t2
</task>

</slice>

<slice name="slice-3-init-core">

## Slice 3: Init Core

<task id="s3-t1" status="pending">
**Description**: Implement vault.Init for fresh creation
**Files**: `internal/vault/init.go`
**Context cost**: Medium
**Depends on**: s2-t3
</task>

<task id="s3-t2" status="pending">
**Description**: Wire init subcommand to CLI
**Files**: `internal/cli/init.go`
**Context cost**: Small
**Depends on**: s3-t1
</task>

<task id="s3-t3" status="pending">
**Description**: Unit tests for init core
**Files**: `internal/vault/init_test.go`
**Context cost**: Small
**Depends on**: s3-t2
</task>

</slice>

<session_boundary reason="End of Slice 3 — verify fresh init works end-to-end before adding idempotency"/>

<slice name="slice-4-idempotency-and-repair">

## Slice 4: Idempotency and Repair

<task id="s4-t1" status="pending">
**Description**: Add idempotency and repair logic to Init
**Files**: `internal/vault/init.go`
**Context cost**: Small
**Depends on**: s3-t3
</task>

<task id="s4-t2" status="pending">
**Description**: Unit tests for idempotency and repair
**Files**: `internal/vault/init_test.go`
**Context cost**: Small
**Depends on**: s4-t1
</task>

</slice>

<slice name="slice-5-force-and-clean">

## Slice 5: Force and Clean Flags

<task id="s5-t1" status="pending">
**Description**: Add force and clean logic to Init
**Files**: `internal/vault/init.go`
**Context cost**: Small
**Depends on**: s4-t2
</task>

<task id="s5-t2" status="pending">
**Description**: Wire --force and --clean flags to CLI
**Files**: `internal/cli/init.go`
**Context cost**: Small
**Depends on**: s5-t1
</task>

<task id="s5-t3" status="pending">
**Description**: Unit tests for force and clean
**Files**: `internal/vault/init_test.go`
**Context cost**: Small
**Depends on**: s5-t2
</task>

</slice>

<session_boundary reason="End of Slice 5 — all init logic complete, verify before integration tests"/>

<slice name="slice-6-instructions">

## Slice 6: Instructions Subcommand

<task id="s6-t1" status="pending">
**Description**: Implement instructions subcommand
**Files**: `internal/cli/instructions.go`
**Context cost**: Small
**Depends on**: s1-t5
</task>

</slice>

<slice name="slice-7-integration-tests">

## Slice 7: Integration Tests

<task id="s7-t1" status="pending">
**Description**: Set up ginkgo test suite
**Files**: `test/integration/integration_suite_test.go`
**Context cost**: Small
**Depends on**: s5-t3, s6-t1
</task>

<task id="s7-t2" status="pending">
**Description**: Create test helpers
**Files**: `internal/testutil/helpers.go`
**Context cost**: Small
**Depends on**: s7-t1
</task>

<task id="s7-t3" status="pending">
**Description**: Write all 10 integration test cases
**Files**: `test/integration/init_test.go`
**Context cost**: Medium
**Depends on**: s7-t2
</task>

</slice>

</worktree>

## Git Worktrees

| Task Group | Worktree Path | Branch | Slices |
|---|---|---|---|
| impl | .worktrees/vault-scaffolding-impl | vault-scaffolding-impl | All slices (sequential) |

<execution_order>

## Execution Order

1. **s1-t1** — Initialize Go module
2. **s1-t2** — Create mise config (parallel with s1-t1)
3. **s1-t3** — Create root cobra command
4. **s1-t4** — Create main entry point
5. **s1-t5** — Fetch dependencies
6. *[Session boundary — verify binary builds]*
7. **s2-t1** — Define vault structure types
8. **s2-t2** — Create seed template files
9. **s2-t3** — Unit tests for vault structure
10. **s3-t1** — Implement vault.Init
11. **s3-t2** — Wire init subcommand
12. **s3-t3** — Unit tests for init core
13. *[Session boundary — verify fresh init works]*
14. **s4-t1** — Add idempotency and repair logic
15. **s4-t2** — Unit tests for idempotency/repair
16. **s5-t1** — Add force and clean logic
17. **s5-t2** — Wire flags to CLI
18. **s5-t3** — Unit tests for force/clean
19. *[Session boundary — all init logic complete]*
20. **s6-t1** — Implement instructions subcommand
21. **s7-t1** — Set up ginkgo test suite
22. **s7-t2** — Create test helpers
23. **s7-t3** — Write all 10 integration test cases

**Parallel opportunities**: s1-t1 and s1-t2 are independent. s6-t1 can run any time after s1-t5. Single worktree means sequential execution in practice.

</execution_order>

<progress>

## Progress Tracking

| Task ID | Description | Status | Completed |
|---------|-------------|--------|-----------|
| s1-t1 | Initialize Go module | pending | — |
| s1-t2 | Create mise config | pending | — |
| s1-t3 | Create root cobra command | pending | — |
| s1-t4 | Create main entry point | pending | — |
| s1-t5 | Fetch dependencies | pending | — |
| s2-t1 | Define vault structure types | pending | — |
| s2-t2 | Create seed template files | pending | — |
| s2-t3 | Unit tests for vault structure | pending | — |
| s3-t1 | Implement vault.Init | pending | — |
| s3-t2 | Wire init subcommand | pending | — |
| s3-t3 | Unit tests for init core | pending | — |
| s4-t1 | Add idempotency/repair logic | pending | — |
| s4-t2 | Unit tests for idempotency/repair | pending | — |
| s5-t1 | Add force/clean logic | pending | — |
| s5-t2 | Wire flags to CLI | pending | — |
| s5-t3 | Unit tests for force/clean | pending | — |
| s6-t1 | Implement instructions subcommand | pending | — |
| s7-t1 | Set up ginkgo test suite | pending | — |
| s7-t2 | Create test helpers | pending | — |
| s7-t3 | Write all 10 integration tests | pending | — |

**Overall**: 0 / 20 tasks complete

</progress>
