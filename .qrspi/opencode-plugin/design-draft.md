# Design Draft: opencode-plugin-agent-memory

**Status:** APPROVED — decisions resolved 2026-05-22.

---

## 1. Project Setup

**Location:** Subdirectory of this repo at `opencode-plugin-agent-memory/`. Named after the npm package to leave room for future harness plugins (e.g. `cline-plugin-agent-memory/`).
<!-- review: should it be `opencode-plugin-agent-memory`? RESOLVED: yes, directory matches package name -->

**Package structure:**
```
opencode-plugin-agent-memory/
├── src/
│   └── index.ts          # single entry point
├── dist/
│   └── index.js          # bundled output
├── package.json
└── tsconfig.json
```

**Build:** `bun build ./src/index.ts --outfile ./dist/index.js --target node` — matches the openspec plugin pattern. Single-file ESM output. No tsup/esbuild needed; bun's bundler handles it.

**Dependencies:**
- `@opencode-ai/plugin` — peer dependency (types only; no runtime import needed beyond `tool`)
- No `zod` dependency — use `tool.schema` which is the host's zod instance

**Registration in opencode.json:**
```json
{
  "plugin": [
    ["opencode-plugin-agent-memory", {
      "vault": ".agent-memory",
      "inject_context": true,
      "auto_promote": true,
      "idle_debounce_ms": 30000
    }]
  ]
}
```

During development, use a local path or `file:` reference. For distribution, publish to npm.

**Package name:** `opencode-plugin-agent-memory`

---

## 2. Plugin Entry Point

```typescript
// src/index.ts
import { existsSync } from "node:fs";
import { join } from "node:path";
import { tool } from "@opencode-ai/plugin";

const z = tool.schema;

export default async (ctx, options = {}) => {
  const vaultPath = resolveVault(ctx.directory, options);
  if (!vaultPath) return {};  // no vault found → no-op

  // ... return hooks
};
```

**Vault discovery** (`resolveVault`):
1. If `options.vault` is set → resolve relative to `ctx.directory`
2. Walk up from `ctx.directory` looking for `.agent-memory/`
3. Check `~/.local/share/agent-memory`
4. Return `null` if nothing found

This mirrors the Go CLI's discovery order (minus the env var, which the plugin itself sets).

---

## 3. `experimental.chat.system.transform` Hook — Context Injection

**What gets injected:** A single string pushed to `output.system[]` containing:

```
<!-- agent-memory context -->
## Agent Memory

This project has a persistent knowledge vault at {vaultPath}.

### Active Constraints
{constraints-summary content, or "No constraints recorded."}

### Stale Notes
{list of notes past review-by, or "None."}

### How to Use
- At the end of every task, use `memory-write` to record durable findings
- Types: observation (saw it), pattern (2+ observations), constraint (external rule), decision (human choice), assumption (unverified belief)
- Write things that would be valuable to a different agent on a different task
<!-- END agent-memory context -->
```

This matches the `AGENTS.md` end-of-task write directive pattern. All agents see this, not just the Librarian.

**Data source:** Shell out to `agent-memory context --json` via `ctx.$`.

**Problem:** `agent-memory context` doesn't exist yet (Increment 6 prerequisite). Two options:

- **Option A (preferred):** Build the plugin against the planned `context --json` interface. Ship the plugin after `context` is implemented. The plugin returns `{}` for the `system.transform` hook if the command fails.
- **Option B:** Assemble context directly by reading vault files (`_meta/constraints-summary.md`, scanning `notes/` for stale notes). This duplicates logic that belongs in the CLI.

**Decision to flag:** Which option? I lean toward A — the plugin should not duplicate CLI logic. The plugin can ship with a graceful fallback: if `agent-memory context --json` fails (command not found or error), skip context injection and log a warning. The other hooks (tools, env) still work independently.
<!-- RESOLVED: Option A -->

**Caching:**
```typescript
const contextCache = new Map<string, { content: string; ts: number }>();

"experimental.chat.system.transform": async (input, output) => {
  const key = input.sessionID ?? "__global";
  let cached = contextCache.get(key);
  
  if (!cached || Date.now() - cached.ts > 60_000) { // refresh every 60s
    const result = await ctx.$`agent-memory context --json`.nothrow().quiet();
    if (result.exitCode === 0) {
      cached = { content: formatContext(result.json()), ts: Date.now() };
      contextCache.set(key, cached);
    }
  }
  
  if (cached) {
    output.system.push(cached.content);
  }
}
```

**Invalidation:** The `memory-write` tool's execute callback clears the cache after a successful write. This ensures the next LLM call sees updated context.
<!-- RESOLVED: caching justified — system.transform fires on every LLM call -->

**Token budget:** Approximate enforcement via character count (~2400 chars ≈ 600 tokens). Truncate the constraints summary and stale list if they exceed budget. The `context --json` command should handle this server-side, but the plugin adds a safety cap.

---

## 4. `tool` Hook — memory-recall and memory-write

### memory-recall

```typescript
tool({
  name: "memory-recall",
  description: "Search the agent memory vault for relevant knowledge notes. Returns note metadata (title, type, domain, confidence, staleness). Read specific note files for full content.",
  parameters: z.object({
    query: z.string().describe("Search query — keywords, concepts, or questions"),
    project: z.string().optional().describe("Filter by project slug"),
    domain: z.string().optional().describe("Filter by domain (e.g. 'golang', 'auth')"),
    type: z.enum(["observation", "pattern", "constraint", "decision", "assumption", "synthesis"]).optional().describe("Filter by epistemic type"),
  }),
  async execute(args, ctx) {
    // Shell out to: agent-memory search --json --query "..." [--project X] [--domain Y] [--type Z]
    const flags = [`--json`, `--query`, args.query];
    if (args.project) flags.push(`--project`, args.project);
    if (args.domain) flags.push(`--domain`, args.domain);
    if (args.type) flags.push(`--type`, args.type);
    
    const result = await ctx.$`agent-memory search ${flags}`.nothrow().quiet();
    if (result.exitCode !== 0) {
      return { error: "agent-memory search failed", stderr: result.stderr.toString() };
    }
    return result.json();
  }
})
```

**Problem:** `agent-memory search` doesn't exist yet (Increment 7 prerequisite). Same fallback strategy as context: return an error message explaining the command is not available. The tool registration still happens — the LLM sees it in the tool list — but calls return a graceful error until the CLI catches up.

**Decision to flag:** Should the tool be omitted entirely when `search` isn't available, or registered with a fallback error? I lean toward registering always — removing tools between sessions would confuse the model, and the error message guides the agent to use direct file reads instead.

### memory-write

```typescript
tool({
  name: "memory-write",
  description: "Write a knowledge note to the agent memory vault. The note lands in _inbox/ for Librarian review. Use at the end of tasks to record durable findings.",
  parameters: z.object({
    title: z.string().describe("Concise factual claim or pattern name"),
    body: z.string().describe("Note body in markdown. One claim per note. Include ## Evidence and ## Implications sections."),
    type: z.enum(["observation", "pattern", "constraint", "decision", "assumption"]).describe("Epistemic type"),
    confidence: z.enum(["low", "medium", "high"]).default("medium").describe("Confidence level"),
    domain: z.array(z.string()).optional().describe("Technology domains (e.g. ['golang', 'auth'])"),
    project: z.string().optional().describe("Source project slug"),
    scope: z.enum(["project", "cross-project"]).default("project").describe("Scope of applicability"),
    source_artifact: z.string().optional().describe("Repo-relative path or URL of the source"),
  }),
  async execute(args, ctx) {
    // Write body to a temp file, then call agent-memory write-note
    const bodyFile = `/tmp/agent-memory-write-${Date.now()}.md`;
    await Bun.write(bodyFile, args.body);
    
    const flags = [
      `--json`,
      `--type`, args.type,
      `--title`, args.title,
      `--confidence`, args.confidence,
      `--scope`, args.scope,
    ];
    if (args.domain) flags.push(`--domain`, args.domain.join(","));
    if (args.project) flags.push(`--project`, args.project);
    if (args.source_artifact) flags.push(`--source-artifact`, args.source_artifact);
    
    const result = await ctx.$`agent-memory write-note ${flags} ${bodyFile}`.nothrow().quiet();
    
    // Clean up temp file
    try { await ctx.$`rm ${bodyFile}`.quiet(); } catch {}
    
    if (result.exitCode !== 0) {
      return { error: "write-note failed", stderr: result.stderr.toString() };
    }
    
    const writeResult = result.json();
    
    // Invalidate context cache on successful write
    if (writeResult.status === "written") {
      contextCache.clear();
    }
    
    return writeResult;
  }
})
```

**On similarity refusal:** When `agent-memory write-note` returns `{"status": "refused", "candidates": [...]}`, the tool returns this directly to the LLM. The LLM sees the similar notes and can decide to either use `--force` (not exposed — deliberate) or adjust its note title/content. The refusal is informative, not a hard error.

**Decision to flag:** Should `memory-write` expose a `force` parameter to bypass similarity checks? The design doc doesn't mention it. I lean toward no — if the similarity check fires, the agent should be forced to reconsider, not bypass. The human can always write directly via CLI with `--force`.
<!-- RESOLVED: no force flag -->

**Temp file for body:** BunShell doesn't have a clean way to pipe multi-line stdin. Writing to a temp file and passing it as the body file arg is the simplest approach. Alternative: use `-` for stdin with BunShell's `.stdin()` if available.

**Decision to flag:** How to pass the body to `write-note`. Try stdin first via BunShell `.stdin()`, fall back to temp file for complex content.
<!-- RESOLVED: both — stdin preferred, temp file fallback -->
---

## 5. `event` Hook — Session-Idle Promotion

```typescript
const IDLE_DEBOUNCE_MS = options.idle_debounce_ms ?? 30_000;
const pendingTimers = new Map<string, ReturnType<typeof setTimeout>>();
const idleSequence = new Map<string, number>();

event: async ({ event }) => {
  if (!options.auto_promote) return;
  
  if (event.type === "session.status" && event.properties.status.type === "busy") {
    // Cancel pending promotion — session is active again
    const sid = event.properties.sessionID;
    const seq = (idleSequence.get(sid) ?? 0) + 1;
    idleSequence.set(sid, seq);
    const timer = pendingTimers.get(sid);
    if (timer) { clearTimeout(timer); pendingTimers.delete(sid); }
    return;
  }
  
  if (event.type === "session.idle") {
    const sid = event.properties.sessionID;
    const seq = (idleSequence.get(sid) ?? 0) + 1;
    idleSequence.set(sid, seq);
    
    // Clear any existing timer
    const existing = pendingTimers.get(sid);
    if (existing) clearTimeout(existing);
    
    const timer = setTimeout(async () => {
      pendingTimers.delete(sid);
      if (idleSequence.get(sid) !== seq) return; // superseded
      
      await runLibrarianPromotion();
    }, IDLE_DEBOUNCE_MS);
    
    pendingTimers.set(sid, timer);
  }
}
```

**How `runLibrarianPromotion` works:**

The plugin cannot invoke a skill programmatically (no API for it). Two options:

- **Option A:** Shell out to `agent-memory promote` for each inbox note directly. This handles the mechanical promotion (lint, move file, update status) but skips human confirmation for constraint/decision notes, and skips the Librarian's judgment (supersession detection, pattern detection).

- **Option B:** List inbox notes, then for each one run `agent-memory promote --slug=<slug> --json`. If the note `requires-human-review: true`, skip it (leave for manual Librarian invocation).

I lean toward **Option B** — it's the simplest thing that works:
<!-- RESOLVED: Option B -->

```typescript
async function runLibrarianPromotion() {
  // Check if inbox has notes
  const inboxPath = join(vaultPath, "_inbox");
  if (!existsSync(inboxPath)) return;
  
  const result = await ctx.$`ls ${inboxPath}/*.md 2>/dev/null`.nothrow().quiet();
  if (result.exitCode !== 0) return; // no files
  
  const files = result.text().trim().split("\n").filter(Boolean);
  for (const file of files) {
    const filename = file.split("/").pop()!;
    // Extract slug: {date}-{slug}.md → slug
    const slug = filename.replace(/^\d{4}-\d{2}-\d{2}-/, "").replace(/\.md$/, "");
    
    const promoteResult = await ctx.$`agent-memory promote --slug=${slug} --json`.nothrow().quiet();
    // Log result but don't throw — fire-and-forget
  }
}
```

**Decision to flag:** This is the biggest open design question. The Librarian skill includes judgment steps (supersession detection, human confirmation flow) that `agent-memory promote` alone doesn't replicate. Options:

1. **CLI-only promotion (Option B above)** — handles 80% of cases (observations, patterns, assumptions auto-promote). Constraint/decision notes stay in inbox until manually handled. Misses supersession detection.
2. **Create a new SDK session** — use `ctx.client` to create a session with Librarian instructions. Heavy-weight, but gets full Librarian judgment. Unclear if the SDK supports creating sessions with custom prompts.
3. **Register a Librarian agent via config hook** — like openspec registers its agent. Then trigger it somehow. But there's no API to start an agent session programmatically from a plugin.

I recommend Option 1 for v1. The Librarian skill remains available for manual invocation when judgment is needed.

---

## 6. `shell.env` Hook — Vault Path

```typescript
"shell.env": async (input, output) => {
  if (!output.env.AGENT_MEMORY_VAULT) {
    output.env.AGENT_MEMORY_VAULT = vaultPath;
  }
}
```

Simple. Only sets if not already present (respects explicit overrides). `vaultPath` is the absolute path resolved at plugin init time.

---

## 7. Options Validation

At plugin init, validate options with a simple check (not zod — keep it lightweight):

```typescript
const opts = {
  vault: typeof options?.vault === "string" ? options.vault : undefined,
  inject_context: options?.inject_context !== false,    // default true
  auto_promote: options?.auto_promote !== false,         // default true
  idle_debounce_ms: typeof options?.idle_debounce_ms === "number" ? options.idle_debounce_ms : 30_000,
};
```

---

## 8. Error Handling Strategy

| Scenario | Behavior |
|---|---|
| `agent-memory` binary not on PATH | Tools return error message; context injection skipped; env hook still sets vault path |
| `agent-memory context --json` fails | Context injection skipped for this call; cached value used if available |
| `agent-memory write-note` returns non-zero | Tool returns error with stderr to LLM |
| `agent-memory write-note` returns similarity refusal | Tool returns the refusal JSON (with candidates) to LLM — informative, not a crash |
| `agent-memory promote` fails for one note | Log and continue to next note |
| Vault directory doesn't exist | Plugin returns `{}` — complete no-op |

All CLI calls use `.nothrow().quiet()` to prevent BunShell from throwing on non-zero exit.

---

## 9. Resolved Decisions

| # | Question | Resolution |
|---|----------|------------|
| 1 | Plugin directory name | `opencode-plugin-agent-memory/` — matches npm package name, leaves room for other harness plugins |
| 2 | Context source | Depend on `agent-memory context --json` with graceful fallback (Option A) |
| 3 | Librarian trigger on idle | CLI-only `agent-memory promote` per note (Option B). Constraint/decision notes stay in inbox. |
| 4 | `memory-recall` without `search` | Register always with fallback error |
| 5 | Body passing for `memory-write` | Try stdin first, fall back to temp file |
| 6 | `force` flag on `memory-write` | No — agents must reconsider on similarity refusal |
| 7 | Config hook for Librarian agent | Deferred — no API to trigger agent sessions programmatically |
| 8 | Cache invalidation scope | Clear all entries on any write (vault is shared across sessions) |
| 9 | Caching | Justified — system.transform fires on every LLM call, 60s TTL |

---

## 10. Hook Summary

| Hook | Purpose | Depends on CLI |
|---|---|---|
| `experimental.chat.system.transform` | Inject vault context into system prompt | `agent-memory context --json` (Increment 6) |
| `tool.memory-recall` | Search vault | `agent-memory search --json` (Increment 7) |
| `tool.memory-write` | Write notes to inbox | `agent-memory write-note --json` (exists) |
| `event` | Auto-promote on session idle | `agent-memory promote --json` (exists) |
| `shell.env` | Set `AGENT_MEMORY_VAULT` | None |
