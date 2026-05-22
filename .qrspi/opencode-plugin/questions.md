# Questions: opencode-plugin

## Plugin API & Hooks

1. The `experimental.chat.system.transform` hook receives `output: { system: string[] }` — does the plugin append to this array, or replace it? Is mutation the pattern (push to array) or reassignment?
2. The `event` hook signature is `(input: { event: Event }) => Promise<void>`. What `Event` type shapes exist? Specifically, is there a `session.idle` event type, and what fields does it carry (sessionID, timestamp, etc.)?
3. The `tool` hook uses `tool()` from `@opencode-ai/plugin` with zod schemas. The `tool.schema` namespace exposes `z` — is this the same `zod` version the plugin should use, or should the plugin declare its own zod dependency?
4. The `shell.env` hook mutates `output.env`. Can it read existing env vars from `output.env`, or only set new ones? Does it need to check for existing `AGENT_MEMORY_VAULT` before setting?
5. The `PluginInput` provides `$: BunShell` — what is `BunShell` and can it be used to execute CLI commands like `agent-memory context --json`? Is this preferred over `child_process.exec`?
6. Plugin options are typed as `Record<string, unknown>`. The design doc specifies `vault`, `inject_context`, `auto_promote`, `idle_debounce_ms` options. How should these be validated — zod schema, manual checks, or trust the config?
7. The `PluginInput` has both `directory` (project dir) and `worktree` fields. Which should be used to resolve the vault path (`.agent-memory/` relative to project)?
8. Does the plugin module need an `id` field (optional in `PluginModule`)? What is it used for and should we set it?
9. The `config` hook receives `Config` and can mutate it. Could the plugin use this to register an agent (like openspec does) for the Librarian, or is the Librarian strictly a skill?
10. How does the plugin lifecycle work — is `server()` called once at startup, or per-session? Do hooks persist across sessions or are they re-registered?

## Integration with CLI

11. The `context` command (`agent-memory context --json`) does not exist yet (Increment 6). What JSON shape should we expect it to return? What fields does the system.transform injection need (vault stats, constraints summary, stale notes, type guide)?
12. The `search` command (for `memory-recall`) does not exist yet (Increment 7 / reindex). What interface should we code against? Will it be `agent-memory search --json --query "..." --domain X`?
13. `agent-memory write-note` exists. What is its exact CLI interface for `--json` output? What does it return on success (slug, path, warnings)?
14. The plugin shells out to `agent-memory` — should it assume the binary is on PATH, or resolve it explicitly? Should the vault path be passed as an env var or CLI flag?
15. Error handling for CLI calls: if `agent-memory` is not installed or returns non-zero, what should the plugin do? Fail silently, log a warning, or throw?

## Existing Plugin Patterns

16. The openspec plugin exports a default function `async (ctx) => Hooks` — is this the canonical pattern? The type says `PluginModule = { server: Plugin }`, but openspec uses a default export directly. Which is correct?
17. The openspec plugin uses `existsSync` for feature detection (checking if openspec dir exists). Should our plugin similarly detect whether `.agent-memory/` exists and return `{}` if absent?
18. The openspec plugin only uses the `config` hook to register an agent. No existing plugin demonstrates `tool`, `event`, `shell.env`, or `system.transform` hooks. Are there any other plugin examples that use these hooks?
19. The openspec plugin builds with `bun build ./src/index.ts --outfile ./dist/index.js --target node`. Should our plugin follow the same build setup, or is there a preferred scaffolding?
20. Plugins are registered in `opencode.json` as strings (package names) or `[name, options]` tuples. How does OpenCode resolve and install these packages — npm, bun, or its own package cache (`~/.cache/opencode/packages/`)?

## Open Design Questions

21. The design doc says context injection should be "< 600 tokens". How should the plugin measure or enforce this? Approximate by character count, or hard-truncate?
22. Session-idle promotion: the `event` hook is a fire-and-forget `Promise<void>`. How should the plugin trigger the Librarian — shell out to `agent-memory promote` for each inbox note, or invoke the `librarian-workflow` skill programmatically? Can a plugin invoke a skill?
23. The design doc mentions caching context "per session, invalidated on write". The `system.transform` hook runs on every LLM call. How should the plugin cache — in-memory Map keyed by sessionID? What invalidates it (the `memory-write` tool call)?
24. Should the plugin package live in this repo (monorepo with `plugin/` directory) or as a separate npm package repo? The Go CLI and TypeScript plugin have different build systems.
25. The `@mohak34/opencode-notifier` plugin is also registered. Is there any documentation or source for it that shows additional hook patterns (especially `event` or `tool`)?
