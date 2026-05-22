# Research: opencode-plugin

## Q1: Does `experimental.chat.system.transform` append or replace?
**Finding:** The hook signature is `(input: { sessionID?: string; model: Model }, output: { system: string[] }) => Promise<void>`. The `output` parameter is mutable — the plugin receives the existing `system` array and mutates it (e.g., `output.system.push(...)`). This is the standard pattern: all hooks receive `output` as a mutable object. No reassignment needed; push to the array.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:261-266`

## Q2: What `Event` type shapes exist? Is there a `session.idle`?
**Finding:** Yes. `Event` is a union of ~30 event types defined in `@opencode-ai/sdk`. `EventSessionIdle` exists:
```typescript
type EventSessionIdle = {
  type: "session.idle";
  properties: {
    sessionID: string;
  };
};
```
Other relevant events: `session.error` (with `properties.sessionID?` and `error`), `session.status` (with `properties.sessionID` and `status: SessionStatus`), `session.compacted`, `session.created`, `session.updated`, `session.deleted`. No timestamp field on `session.idle` — only `sessionID`.
- File: `~/.cache/opencode/packages/@mohak34/opencode-notifier@latest/node_modules/@opencode-ai/sdk/dist/gen/types.gen.d.ts:413-418,602`

## Q3: Does `tool.schema` expose `z` (zod), and should the plugin use its own?
**Finding:** Yes. `tool.schema` is literally `typeof z` from `zod`. The `tool.d.ts` imports `z` from `"zod"` and exports `tool.schema = z`. The plugin should use `tool.schema` (i.e., `import { tool } from "@opencode-ai/plugin"; const z = tool.schema;`) — this guarantees version compatibility with the host. No separate zod dependency needed.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/tool.d.ts:1,49-51`

## Q4: Can `shell.env` read existing env vars from `output.env`?
**Finding:** The hook signature is `(input: { cwd: string; sessionID?: string; callID?: string }, output: { env: Record<string, string> }) => Promise<void>`. The `output.env` object is mutable and may contain vars set by other plugins or the system. The plugin can read and write it. It should check for existing `AGENT_MEMORY_VAULT` before overwriting if desired.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:238-244`

## Q5: What is `BunShell` and can it execute CLI commands?
**Finding:** `BunShell` is a tagged template literal shell interface. Usage: `` const result = await $`agent-memory context --json`.json() ``. It supports `.text()`, `.json()`, `.quiet()`, `.nothrow()`, `.cwd()`, `.env()`. It throws on non-zero exit by default (use `.nothrow()` to suppress). This is the preferred way to run CLI commands — no need for `child_process`. The notifier plugin uses `child_process` but it's an older pattern; `$` is provided specifically for plugin use.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/shell.d.ts:7-108`

## Q6: How should plugin options be validated?
**Finding:** `PluginOptions` is typed as `Record<string, unknown>`. No built-in validation. The openspec plugin does not validate options at all (it ignores them). The notifier plugin loads its own config from a separate JSON file. For our plugin, manual validation or a simple zod schema (using `tool.schema`) at plugin init would work. No framework-level validation mechanism exists.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:47,51`

## Q7: `directory` vs `worktree` for vault path resolution?
**Finding:** `PluginInput` has `directory: string` and `worktree: string`. `ToolContext` also has both, with doc comments: `directory` = "Current project directory for this session. Prefer this over process.cwd()", `worktree` = "Project worktree root for this session. Useful for generating stable relative paths." For vault resolution, `directory` is the project dir where `.agent-memory/` lives. Use `directory` for vault discovery. `worktree` is for git worktree scenarios where the working dir differs from the repo root.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/tool.d.ts:9-15`, `index.d.ts:39-40`

## Q8: Does `PluginModule` need an `id` field?
**Finding:** `id` is optional in `PluginModule`: `{ id?: string; server: Plugin; tui?: never }`. The openspec plugin does not set `id`. The notifier plugin does not set `id`. Neither has a `PluginModule` wrapper — both export a default function directly (the `Plugin` function itself, not `PluginModule`). OpenCode apparently accepts both a bare `Plugin` function as default export and a `PluginModule` object.
- File: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts:52-56`

## Q9: Can the `config` hook register an agent (like the Librarian)?
**Finding:** Yes. The openspec plugin demonstrates this exactly. The `config` hook receives a mutable `Config` object. It sets `config.agent["openspec-plan"] = { name, mode, description, prompt, permission, color }`. The agent definition includes full permission objects. Our plugin could register a Librarian agent the same way. The Librarian is currently a skill-based agent defined in vault templates, but the plugin could also register it via `config`.
- File: `~/.cache/opencode/packages/opencode-plugin-openspec@latest/node_modules/opencode-plugin-openspec/dist/index.js:43-106`

## Q10: Plugin lifecycle — `server()` called once or per-session?
**Finding:** `server` is the `Plugin` function: `(input: PluginInput, options?) => Promise<Hooks>`. Based on the notifier plugin's behavior (it uses in-memory Maps for state tracking across sessions, sets up `setInterval` for cleanup), `server()` is called **once at startup** and the returned `Hooks` persist across all sessions. Hooks receive `sessionID` in their input parameters to distinguish sessions. The plugin is a long-lived singleton.
- File: notifier plugin `dist/index.js:783-1091` (module-level state, setInterval at line 813)

## Q11: `agent-memory context --json` — does it exist?
**Finding:** **No.** The `context` command does not exist yet. `root.go` registers: `init`, `instructions`, `lint-note`, `write-note`, `promote`, `deprecate`. The design doc (§8) lists it as part of Increment 3 ("Build the agent-facing trio — context, write-note, search"). Context injection will need to be built before the plugin can use it, or the plugin must assemble context itself by reading vault files directly.
- File: `internal/cli/root.go:18-24`

## Q12: `agent-memory search` — does it exist?
**Finding:** **No.** Not registered in `root.go`. Listed as Increment 3 in the design doc. The `memory-recall` tool will need a fallback strategy (grep-based search or direct file reading) until this command is built.
- File: `internal/cli/root.go:18-24`

## Q13: `agent-memory write-note --json` output format?
**Finding:** Returns a `WriteResult` struct serialized as JSON:
```json
{
  "status": "written" | "refused",
  "path": "/path/to/note.md",         // omitempty, present when written
  "reason": "...",                      // omitempty, present when refused
  "warnings": ["..."],                  // omitempty
  "candidates": [                       // omitempty, present when refused due to similarity
    {"path": "...", "title": "...", "similarity": 0.85}
  ],
  "errors": [...]                       // omitempty, lint errors
}
```
On error (file not readable, vault not found, internal error), returns `{"status": "error", "error": "..."}`.
Required flags: `--type`, `--title`. Other flags: `--domain`, `--scope`, `--source-artifact`, `--source-agent`, `--confidence`, `--tags`, `--project`, `--vault`, `--force`. Body is read from a file arg (or `-` for stdin).
- File: `internal/note/write.go:28-35`, `internal/cli/write_note.go:30-148`

## Q14: Should the plugin assume `agent-memory` is on PATH?
**Finding:** The plugin receives `$: BunShell` which inherits the process environment. The `shell.env` hook can set `PATH` additions. The design doc says the plugin sets `AGENT_MEMORY_VAULT` via `shell.env` but doesn't mention PATH resolution. The plugin should assume `agent-memory` is on PATH (the standard Go install pattern puts it in `$GOPATH/bin` or `$HOME/go/bin`). If not found, `$` will throw — the plugin should catch and warn.
- File: design doc §5.9, `shell.d.ts:33` (throws by default)

## Q15: Error handling for CLI calls?
**Finding:** `BunShell` throws on non-zero exit by default. The plugin can use `.nothrow()` to get the result without throwing and check `exitCode`. The notifier plugin doesn't shell out to external CLIs. The openspec plugin doesn't either. For our plugin: use `.nothrow()` on `$`, check `exitCode`, and if non-zero, log a warning but don't crash the plugin. The `event` hook returns `Promise<void>` so throwing would be swallowed anyway, but tool hooks should return meaningful error messages.
- File: `shell.d.ts:29-33,84-88`

## Q16: Canonical plugin export pattern?
**Finding:** Both openspec and notifier export a **default function** that matches the `Plugin` signature: `async (ctx: PluginInput) => Hooks`. They do NOT use the `PluginModule = { server: Plugin }` wrapper. The `PluginModule` type exists but neither real plugin uses it. The canonical pattern is: `export default async (ctx) => ({ ...hooks })`.
- File: openspec `dist/index.js:110-122`, notifier `dist/index.js:1031-1091`

## Q17: Should the plugin detect `.agent-memory/` and return `{}` if absent?
**Finding:** Yes, this is the established pattern. The openspec plugin checks `existsSync(configYamlPath) || existsSync(agentsMdPath)` and returns `{}` if the project doesn't use openspec. Our plugin should similarly check for `.agent-memory/` (or run vault discovery) and return `{}` if no vault is found, making the plugin a no-op for projects without agent memory.
- File: openspec `dist/index.js:4-9,110-114`

## Q18: Are there examples of `tool`, `event`, `shell.env`, or `system.transform` hooks?
**Finding:** The **notifier plugin** demonstrates:
- `event` hook: handles `session.idle`, `session.error`, `session.status`, `permission.asked` events (line 1041-1071)
- `tool.execute.before` hook: intercepts `question` tool calls (line 1078-1083)
- `permission.ask` hook (line 1072-1077)

No existing plugin demonstrates `tool` (registration), `shell.env`, or `experimental.chat.system.transform`. These will be first-of-kind usages.
- File: notifier `dist/index.js:1031-1085`

## Q19: Build setup?
**Finding:** The openspec plugin's `package.json` would show the build script, but the built output is a single `dist/index.js` file. The notifier plugin is similarly a single bundled `dist/index.js`. Both are ESM (`export default`). The openspec README mentions `bun build ./src/index.ts --outfile ./dist/index.js --target node`. This is the pattern to follow.
- File: openspec `dist/index.js` (122 lines, single file), notifier `dist/index.js` (1091 lines, single file)

## Q20: How does OpenCode resolve and install plugin packages?
**Finding:** Plugins are resolved to `~/.cache/opencode/packages/<name>@latest/node_modules/<name>/`. The cache directory structure shows packages installed with their full dependency trees (e.g., the notifier package has its own `node_modules` with `zod`, `@opencode-ai/sdk`, etc.). Registration is in `opencode.json` as plain strings or `[name, options]` tuples. OpenCode likely uses `bun install` or `npm install` to resolve packages into this cache.
- File: `~/.config/opencode/opencode.json:3-6`, `~/.cache/opencode/packages/` directory structure

## Q21: How to enforce the 600-token context limit?
**Finding:** The design doc says "Content is kept under 600 tokens to minimise context window impact" (§5.9). No mechanism is specified. Character-count approximation (1 token ≈ 4 chars, so ~2400 chars) is the practical approach. The `system.transform` hook just pushes strings to an array — no built-in token counting.
- File: design doc line 1213-1214

## Q22: How to trigger the Librarian from session-idle?
**Finding:** The `event` hook is `Promise<void>` — fire-and-forget. The plugin has access to `client: ReturnType<typeof createOpencodeClient>` from `PluginInput`. The notifier plugin uses `client.session.messages()` and `client.session.get()` to query session state. The plugin could potentially use the client SDK to create a new session or send a message to trigger the Librarian. However, there is **no API to invoke a skill programmatically** visible in the SDK types. The design doc says the plugin "invokes the Librarian via the `librarian-workflow` skill" but the mechanism is unspecified. Options: (1) shell out to `agent-memory promote` directly for each inbox note, (2) use the SDK client to create a session with instructions to run the librarian workflow. A plugin **cannot** invoke a skill directly.
- File: `index.d.ts:37,171-173`, notifier usage of `client` at lines 949-979

## Q23: How should the plugin cache context?
**Finding:** The plugin's `server()` is called once; hooks persist across sessions. An in-memory `Map<string, { content: string; timestamp: number }>` is viable for caching. The `system.transform` hook receives `sessionID` in `input`. Invalidation: the `memory-write` tool's `execute` callback knows when a write happens — it can clear the cache. The `event` hook can also watch for `file.edited` events on vault files (event type exists: `EventFileEdited = { type: "file.edited"; properties: { file: string } }`). No framework-provided caching.
- File: `index.d.ts:261-266` (system.transform input has sessionID), types.gen.d.ts:425-429 (file.edited event)

## Q24: Should the plugin live in this repo or a separate package?
**Finding:** This is a design question, not a codebase fact. Observable facts: the Go CLI is built with `go build`. The plugin would be TypeScript built with `bun`. The openspec plugin is a separate npm package. The notifier plugin is a separate npm package. The vault templates (`_meta/templates/`) are seeded by `agent-memory init` and live in this repo. The design doc (§8, item 6) lists the plugin as a separate build step. No `plugin/` directory exists in the repo currently.
- File: design doc line 1351, repo structure

## Q25: Does the notifier plugin show additional hook patterns?
**Finding:** Yes, extensively. Key patterns observed:
1. **Event hook with debouncing**: Uses `setTimeout` with sequence numbers to debounce `session.idle` events (350ms delay). Tracks per-session state with Maps.
2. **Accessing SDK client**: Uses `client.session.messages()` and `client.session.get()` for session introspection.
3. **Feature detection at init**: Checks `process.env.OPENCODE_CLIENT` to conditionally activate.
4. **Config from external file**: Loads config from `~/.config/opencode/opencode-notifier.json`, separate from plugin options.
5. **Default export pattern**: `export default NotifierPlugin` where `NotifierPlugin` matches `Plugin` signature.
6. **Multiple hooks**: Returns object with `event`, `permission.ask`, and `tool.execute.before` hooks simultaneously.
- File: notifier `dist/index.js:783-1091`
