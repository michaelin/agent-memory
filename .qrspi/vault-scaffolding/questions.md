<questions_artifact feature="vault-scaffolding">

<feature>
**Name:** vault-scaffolding
**Description:** Establish the vault structure and configuration layer (agent-memory init, config resolution, vault directory scaffolding, seed templates, idempotency, integration tests)
</feature>

<questions>

<category name="project-setup">
## Project Setup Questions

<question>
What Go module path should this project use? (e.g., `github.com/michaelin/agent-memory`, or a vanity import path?)
<why>Forces decision on: module identity, import paths for all future packages, and whether this will be a public or private module</why>
</question>

<question>
What minimum Go version should the project target, and should it use Go workspaces or a single module?
<why>Forces decision on: go.mod constraints, availability of features like `embed`, `os.UserConfigDir()`, `testing/fstest`, generics</why>
</question>

<question>
What is the intended distribution mechanism for the `agent-memory` binary? (go install, Homebrew, goreleaser, manual build, or just `go build` for now?)
<why>Forces decision on: main package location (cmd/agent-memory/), build tags, release automation, and whether cross-compilation matters for Increment 1</why>
</question>

<question>
Should the project use a Makefile, Taskfile, or just `go` commands for build/test/lint?
<why>Forces decision on: developer workflow, CI expectations, and whether there are existing conventions from other projects in this workspace</why>
</question>
</category>

<category name="data-model">
## Data Model Questions

<question>
What is the exact JSON schema for `~/.config/agent-memory/config.json`? The ROADMAP shows `{"vault": "/path/to/vault"}` — should it support additional fields for future increments (e.g., backup location, log level), or should it be strictly minimal for v1?
<why>Forces decision on: config struct shape, forward compatibility, and whether to use a versioned config format</why>
</question>

<question>
The ROADMAP specifies config at `~/.config/agent-memory/config.json` while the design doc specifies `~/.agent-memory.json`. The ticket notes ROADMAP takes precedence. Should the tool also check the design doc's location as a fallback for migration, or is it a clean break?
<why>Forces decision on: config resolution chain length and whether there's a migration path from the design doc's convention</why>
</question>

<question>
What are the exact contents of each seed template file? The ROADMAP specifies `writing-protocol.md` content but not `tag-taxonomy.md`, `constraints-summary.md`, `status-lifecycle.md`, or `AGENTS.md`. Should these be empty files, files with headers only, or files with placeholder content?
<why>Forces exploration of: AGENT_MEMORY_DESIGN.md sections 5.2-5.6 for the expected structure of each meta file, and whether Increment 1 needs to seed content that Increment 2+ will depend on</why>
</question>

<question>
Should `_meta/log.md` use the format `## [YYYY-MM-DD HH:MM] init | vault initialized` (as in ROADMAP) or a different structured format? Should the log use UTC or local time? Should it include timezone info?
<why>Forces decision on: log format that all future increments will append to, time zone handling across machines</why>
</question>
</category>

<category name="api">
## CLI API Questions

<question>
What is the CLI interface for `agent-memory init`? Does it accept a `--vault-path` flag to override config resolution? Does it accept `--config` to specify a config file path? What flags exist?
<why>Forces decision on: CLI framework (cobra, urfave/cli, bare flag package), command structure for future subcommands (init, lint-note, lint-vault, etc.), and whether all tools share a single binary or are separate binaries</why>
</question>

<question>
Should all agent-memory tools be subcommands of a single `agent-memory` binary (e.g., `agent-memory init`, `agent-memory lint-note`), or separate binaries (`agent-memory`, `lint-note`, `lint-vault`)?
<why>Forces decision on: binary distribution, package layout (cmd/agent-memory/ vs cmd/lint-note/), and how agents invoke tools (single binary path vs multiple)</why>
</question>

<question>
What should `agent-memory init` output on success? On repair? On no-op (already initialized)? Should it use JSON output like `lint-note`, or human-readable text, or both (with a `--json` flag)?
<why>Forces decision on: output format conventions that all future subcommands will follow, and whether agents parse stdout or just check exit codes</why>
</question>

<question>
Should `agent-memory init` write the config file automatically, or should config file creation be a separate step? If the vault path comes from an env var, should init still write a config file?
<why>Forces decision on: whether init has side effects outside the vault directory, and the relationship between vault creation and config persistence</why>
</question>
</category>

<category name="integration">
## Integration Questions

<question>
How will agents discover and invoke the `agent-memory` binary? Is it expected to be on `$PATH`, or will skills/tools reference an absolute path? Does OpenCode have a convention for tool binary discovery?
<why>Forces decision on: installation location, PATH requirements, and whether the binary needs to be discoverable without configuration</why>
</question>

<question>
The design doc mentions `go:embed` for seed templates. Should templates be embedded in the binary, or read from a known filesystem location? Embedding is simpler but means template changes require a rebuild.
<why>Forces decision on: template management strategy, and whether the binary is fully self-contained</why>
</question>

<question>
Should `agent-memory init` create the vault parent directories if they don't exist (e.g., `~/Documents/Obsidian/` doesn't exist yet), or should it fail if the parent doesn't exist?
<why>Forces decision on: how aggressive init is about filesystem side effects, and error messaging for common setup issues</why>
</question>
</category>

<category name="edge-cases">
## Edge Case Questions

<question>
What happens if `agent-memory init` is run with a vault path that points to an existing non-vault directory (e.g., a directory with files but no `_meta/`)? Should it refuse, or should it scaffold around existing files?
<why>Forces decision on: safety behavior when init encounters unexpected filesystem state</why>
</question>

<question>
What happens if the config file exists but contains invalid JSON, or valid JSON with an invalid vault path (e.g., path doesn't exist, path is a file not a directory)?
<why>Forces decision on: config validation behavior and error reporting</why>
</question>

<question>
What filesystem permissions should vault directories and files be created with? Should the tool respect umask, or set explicit permissions (e.g., 0700 for directories, 0600 for files)?
<why>Forces decision on: security posture of the vault, especially since it may contain observations about client codebases</why>
</question>

<question>
How should `agent-memory init` handle symlinks? If the vault path is a symlink, should it follow it? If a subdirectory inside the vault is a symlink, should repair detect that as corruption?
<why>Forces decision on: symlink handling policy, which affects Obsidian compatibility and backup behavior</why>
</question>

<question>
The ROADMAP mentions "upgrade it (no-op for v1)" — should the vault have a version marker file or field? Where does the version live? This affects all future increments that might need migration.
<why>Forces decision on: vault versioning strategy, which must be established in Increment 1 even if v1 is trivial</why>
</question>
</category>

</questions>

</questions_artifact>
