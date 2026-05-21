---
title: "ADR-0001: Single binary with subcommands over separate binaries"
status: "Accepted"
date: "2026-05-21"
authors: ["michaelin"]
tags: ["architecture", "cli"]
supersedes: ""
superseded_by: ""
---

# ADR-0001: Single binary with subcommands over separate binaries

## Status

**Accepted**

## Context

The original design document (AGENT_MEMORY_DESIGN.md §8.2) proposed multiple separate binaries under `cmd/` — one per tool. The design listed eight binaries: `memory-context`, `memory-write`, `memory-search`, `memory-promote`, `memory-reindex`, `memory-synthesize`, `lint-note`, and `lint-vault`, plus the `agent-memory` CLI wrapper for human-facing commands.

During increment 1, the project was built with a single `agent-memory` binary using cobra subcommands (`agent-memory init`, `agent-memory instructions`). No separate binaries were created.

As increment 2 introduces `lint-note` functionality, a decision was needed: follow the design doc's multi-binary layout, or continue the single-binary subcommand pattern.

The design doc's multi-binary layout was motivated by a Unix philosophy of small, composable tools. Each binary would be independently callable by agents, scripts, and CI pipelines. The trade-off is operational complexity: nine binaries to build, install, version, and discover.

The project is a personal developer tool, not a platform with independent release cycles per component. All tools share the same vault format, the same frontmatter types, and the same validation logic. There is no team boundary or deployment boundary that would benefit from separate binaries.

## Decision

All tools ship as subcommands of the single `agent-memory` binary. There are no separate binaries.

What the design doc called `lint-note <file>` becomes `agent-memory lint-note <file>`. What it called `memory-write --new-claim ...` becomes `agent-memory write --new-claim ...`. The `memory-` prefix is dropped from subcommand names since the binary name already provides the namespace.

## Consequences

### Positive

- **Single discovery point.** Agents need to find one binary. The `instructions` subcommand already tells agents where to find it. Adding more tools doesn't change the discovery mechanism.
- **Shared flags and conventions.** The `--json` persistent flag, error formatting, and output conventions are defined once on the root command. Every new subcommand inherits them without reimplementation.
- **Self-documenting.** `agent-memory --help` lists every available command. Agents and humans can discover capabilities without knowing the full tool inventory in advance.
- **Simpler build and install.** One `go build` target, one binary to copy or `go install`. No need for a Makefile that builds nine binaries, no risk of version skew between them.
- **Shared internal packages.** All subcommands link against the same `internal/vault`, `internal/lint`, etc. There is no risk of one binary using a stale version of the frontmatter parser while another uses a newer one.
- **Follows established convention.** Tools like `git`, `docker`, `kubectl`, and `gh` use the single-binary-with-subcommands pattern. Agents trained on these tools will find the pattern familiar.

### Negative

- **Binary size grows.** Every subcommand's dependencies are linked into one binary. For a Go CLI tool this is unlikely to matter in practice — the binary will stay under 20MB even with all planned subcommands — but it is a real trade-off compared to building only the tools you need.
- **No independent versioning.** You cannot release `lint-note` v2 while keeping `write` at v1. All subcommands share a single version number. This would matter if different consumers needed different release cadences, but that is not the case here.
- **No independent deployment.** In a scenario where you wanted to install only the lint tools on a CI runner and nothing else, you would still install the full binary. The unused subcommands add no runtime cost, but they do add binary size.
- **Startup cost is shared.** If any subcommand introduces expensive initialization (e.g., loading a large embedded asset), it could affect startup time for all subcommands unless initialization is deferred to the subcommand's `RunE`. Cobra's lazy initialization pattern handles this well, but it requires discipline.
- **Cobra dependency is load-bearing.** The entire tool surface depends on cobra for command registration, flag parsing, and help generation. Replacing cobra would require touching every subcommand. This is a low-probability risk given cobra's maturity and adoption.

## Alternatives Considered

### Separate binaries per tool (design doc approach)

**Description**: Each tool as its own `main.go` under `cmd/` — `cmd/lint-note/main.go`, `cmd/memory-write/main.go`, etc. Each binary is independently buildable and callable. The `agent-memory` binary would be a thin wrapper that delegates to the other binaries or reimplements their logic.

**Rejection reason**: Nine binaries to build, install, and keep in sync. Agents need to discover and locate each binary separately — either via PATH or by convention. Version skew between binaries is possible if they are installed independently. The project has no team or deployment boundary that would benefit from this separation. The thin-wrapper pattern (where `agent-memory lint` shells out to `lint-note`) adds latency and complexity without adding capability.

### Hybrid: single binary with symlink dispatch

**Description**: Build a single binary but install it under multiple names via symlinks (`lint-note` → `agent-memory`). The binary inspects `os.Args[0]` to determine which subcommand to run, similar to BusyBox.

**Rejection reason**: Adds installation complexity (symlink management) for no practical benefit over subcommands. The `os.Args[0]` dispatch pattern is fragile across platforms and confusing to debug. It solves a problem (short command names) that doesn't exist here — `agent-memory lint-note` is already short enough, and agents don't care about typing economy.

## Implementation Notes

- The subcommand naming convention drops the `memory-` prefix from the design doc names: `memory-write` becomes `agent-memory write`, `memory-search` becomes `agent-memory search`, etc. The `lint-` prefix is kept for lint commands: `agent-memory lint-note`, `agent-memory lint-vault`.
- Each subcommand follows the existing pattern: unexported factory function (`newLintNoteCmd()`) in `internal/cli/`, returning `*cobra.Command`, registered via `AddCommand()` in `NewRootCmd()`.
- The `--json` persistent flag on the root command is available to all subcommands. Each subcommand implements dual output (JSON and human-readable) following the pattern established by `init`.

## References

- AGENT_MEMORY_DESIGN.md §8.2 (original multi-binary layout)
- Increment 1 implementation establishing single-binary pattern with `init` and `instructions` subcommands
