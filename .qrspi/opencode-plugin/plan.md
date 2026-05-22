# Plan: opencode-plugin-agent-memory

**Constrained by:** structure-draft.md, design-draft.md (APPROVED)

---

## Slice 1: Plugin Scaffold + Vault Discovery + shell.env

### Task 1.1: package.json + tsconfig.json
- **File:** `opencode-plugin-agent-memory/package.json`
- **What:** Create package.json with `name: "opencode-plugin-agent-memory"`, `type: "module"`, `main: "dist/index.js"`, `scripts.build: "bun build ./src/index.ts --outfile ./dist/index.js --target node"`, peer dep on `@opencode-ai/plugin`, dev dep on `@opencode-ai/plugin` + `typescript`
- **File:** `opencode-plugin-agent-memory/tsconfig.json`
- **What:** Standard TS config: `target: "esnext"`, `module: "esnext"`, `moduleResolution: "bundler"`, `types: ["bun-types"]`, strict mode
- **Verify:** `cd opencode-plugin-agent-memory && bun install` succeeds

### Task 1.2: vault.ts — resolveVault
- **File:** `opencode-plugin-agent-memory/src/vault.ts`
- **What:** Implement `resolveVault(directory, options)`:
  1. If `options.vault` set, resolve relative to `directory`, check `existsSync`
  2. Walk up from `directory` checking for `.agent-memory/` directory
  3. Check `~/.local/share/agent-memory`
  4. Return absolute path or `null`
- **Verify:** Manual test — create a temp dir with `.agent-memory/`, confirm discovery. Also confirm `null` return for empty dir.

### Task 1.3: index.ts — entry point + shell.env hook
- **File:** `opencode-plugin-agent-memory/src/index.ts`
- **What:** Default export matching `Plugin` signature. Call `resolveVault()`. If null, return `{}`. Parse options via `parseOptions()`. Return hooks object with `"shell.env"` that sets `AGENT_MEMORY_VAULT = vaultPath` if not already present.
- **Verify:** `bun build` succeeds. Plugin can be loaded by OpenCode (add to `opencode.json` with local path, start OpenCode, check env in a shell tool call).

---

## Slice 2: memory-write Tool

### Task 2.1: memory-write.ts — tool definition
- **File:** `opencode-plugin-agent-memory/src/tools/memory-write.ts`
- **What:** Export `createMemoryWriteTool($, vaultPath, onWrite)` returning a `ToolDefinition`. Schema uses `tool.schema` (z) for params matching `MemoryWriteParams`. Execute fn:
  1. Try stdin approach: pipe `args.body` via `$\`echo ${args.body} | agent-memory write-note --json --type ${args.type} --title ${args.title} ... -\``
  2. If stdin fails, fall back to temp file: write body to `/tmp/agent-memory-write-${Date.now()}.md`, pass as positional arg, clean up after
  3. Build flags array from args (--confidence, --scope, --domain, --project, --source-artifact)
  4. Use `.nothrow().quiet()`, check exitCode
  5. On success with `status: "written"`, call `onWrite()` to invalidate context cache
  6. Return parsed JSON result
- **Verify:** With `agent-memory` on PATH, call the tool from OpenCode with a test note. Confirm note appears in `_inbox/`. Confirm similarity refusal returns candidates JSON.

### Task 2.2: Wire memory-write into index.ts
- **File:** `opencode-plugin-agent-memory/src/index.ts`
- **What:** Import `createMemoryWriteTool`, add to returned hooks as `tool: { "memory-write": createMemoryWriteTool($, vaultPath, () => cache.invalidateAll()) }`. (Cache reference comes from Slice 4; for now use a no-op callback.)
- **Verify:** `bun build` succeeds. Tool visible in OpenCode tool list.

---

## Slice 3: memory-recall Tool

### Task 3.1: memory-recall.ts — tool definition with fallback
- **File:** `opencode-plugin-agent-memory/src/tools/memory-recall.ts`
- **What:** Export `createMemoryRecallTool($, vaultPath)` returning a `ToolDefinition`. Schema uses z for `MemoryRecallParams`. Execute fn:
  1. Build flags: `--json --query ${args.query}` + optional `--project`, `--domain`, `--type`
  2. Run `$\`agent-memory search ${flags}\`.nothrow().quiet()`
  3. If exitCode !== 0, return `{ error: "memory-recall is not yet available. The 'agent-memory search' command is not installed. Use direct file reading in the .agent-memory/notes/ directory instead.", vault_path: vaultPath }`
  4. On success, return parsed JSON
- **Verify:** Without `agent-memory search` command, confirm tool returns the fallback error message. Tool is visible in OpenCode's tool list.

### Task 3.2: Wire memory-recall into index.ts
- **File:** `opencode-plugin-agent-memory/src/index.ts`
- **What:** Import `createMemoryRecallTool`, add to `tool` object alongside memory-write.
- **Verify:** `bun build` succeeds. Both tools visible.

---

## Slice 4: Context Injection

### Task 4.1: context.ts — cache + system.transform
- **File:** `opencode-plugin-agent-memory/src/context.ts`
- **What:**
  1. `createContextCache($, vaultPath)`: Returns object with `get(sessionID)` and `invalidateAll()`. Internal `Map<string, { content: string; ts: number }>`. `get()` checks 60s TTL; if expired or missing, synchronously returns undefined (caller must refresh). Add `refresh(sessionID)` that runs `$\`agent-memory context --json\`.nothrow().quiet()`, parses result, formats into the context template from design §3 (constraints, stale notes, usage guide), enforces ~2400 char cap via truncation, stores in map.
  2. `createSystemTransformHook(cache)`: Returns the hook function. On call: get cached content for `input.sessionID ?? "__global"`. If cache miss, call `cache.refresh()`. If content available, `output.system.push(content)`.
  3. Format function: takes context JSON, produces markdown block with `<!-- agent-memory context -->` wrapper matching design §3.
- **Verify:** With `agent-memory context --json` unavailable, confirm hook silently skips (no error). With mock/future command, confirm context appears in system prompt.

### Task 4.2: Wire context into index.ts
- **File:** `opencode-plugin-agent-memory/src/index.ts`
- **What:** Import context functions. Create cache instance. Pass `cache.invalidateAll` as `onWrite` callback to memory-write tool. Add `"experimental.chat.system.transform": createSystemTransformHook(cache)` to hooks. Gate behind `opts.inject_context`.
- **Verify:** `bun build` succeeds. With `inject_context: false` in options, hook is not registered.

---

## Slice 5: Session-Idle Promotion

### Task 5.1: promotion.ts — event hook
- **File:** `opencode-plugin-agent-memory/src/promotion.ts`
- **What:** Export `createEventHook($, vaultPath, debounceMs, enabled)`. Implementation:
  1. If `!enabled`, return async no-op
  2. Track `pendingTimers: Map<string, Timer>` and `idleSequence: Map<string, number>` (module-level or closure-scoped)
  3. On `session.status` with `status.type === "busy"`: increment sequence, clear pending timer
  4. On `session.idle`: increment sequence, capture seq, clear existing timer, set new timeout
  5. On timeout fire: check seq still matches, then call `runPromotion()`
  6. `runPromotion()`: check `existsSync(join(vaultPath, "_inbox"))`, list `.md` files, extract slugs (strip date prefix + `.md`), run `$\`agent-memory promote --slug=${slug} --json\`.nothrow().quiet()` for each, log results
- **Verify:** Trigger a `session.idle` event (write a note, wait for debounce). Confirm notes in `_inbox/` get promoted to `notes/`. Confirm constraint/decision notes stay (promote refuses without `--confirmed`).

### Task 5.2: Wire event hook into index.ts
- **File:** `opencode-plugin-agent-memory/src/index.ts`
- **What:** Import `createEventHook`, add `event: createEventHook($, vaultPath, opts.idle_debounce_ms, opts.auto_promote)` to hooks.
- **Verify:** `bun build` succeeds. Full plugin loads in OpenCode with all hooks.

---

## Final Integration

### Task 6.1: Build and smoke test
- **File:** `opencode-plugin-agent-memory/dist/index.js`
- **What:** Run `bun build`. Register plugin in `opencode.json` via local path. Start OpenCode. Verify:
  1. `AGENT_MEMORY_VAULT` is set in shell env
  2. `memory-write` tool is listed and callable
  3. `memory-recall` tool is listed and returns fallback error
  4. System prompt contains agent-memory context block (or silently skips if `context` command unavailable)
  5. Session idle triggers promotion attempt
- **Verify:** All 5 checks pass.

---

## Task Summary

| # | Task | File(s) | Slice |
|---|------|---------|-------|
| 1.1 | package.json + tsconfig | package.json, tsconfig.json | 1 |
| 1.2 | resolveVault | src/vault.ts | 1 |
| 1.3 | Entry point + shell.env | src/index.ts | 1 |
| 2.1 | memory-write tool | src/tools/memory-write.ts | 2 |
| 2.2 | Wire memory-write | src/index.ts | 2 |
| 3.1 | memory-recall tool | src/tools/memory-recall.ts | 3 |
| 3.2 | Wire memory-recall | src/index.ts | 3 |
| 4.1 | Context cache + transform | src/context.ts | 4 |
| 4.2 | Wire context | src/index.ts | 4 |
| 5.1 | Event hook (promotion) | src/promotion.ts | 5 |
| 5.2 | Wire event hook | src/index.ts | 5 |
| 6.1 | Build + smoke test | dist/index.js | all |
