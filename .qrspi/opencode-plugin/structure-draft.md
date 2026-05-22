# Structure Draft: opencode-plugin-agent-memory

**Based on:** design-draft.md (APPROVED), research.md, plugin API types

---

## Vertical Slices

### Slice 1: Plugin Scaffold + Vault Discovery + shell.env Hook

**Goal:** Minimal plugin that discovers the vault, returns `{}` if absent, and sets `AGENT_MEMORY_VAULT` in shell environments.

**New files:**

```
opencode-plugin-agent-memory/
├── src/
│   ├── index.ts
│   └── vault.ts
├── package.json
├── tsconfig.json
```

**Type signatures:**

```typescript
// src/vault.ts
export function resolveVault(directory: string, options: PluginOptions): string | null;

// src/index.ts
import type { Plugin, Hooks, PluginInput, PluginOptions } from "@opencode-ai/plugin";

interface PluginOpts {
  vault?: string;
  inject_context: boolean;
  auto_promote: boolean;
  idle_debounce_ms: number;
}

function parseOptions(options?: PluginOptions): PluginOpts;

const plugin: Plugin;
export default plugin;
// Returns: { "shell.env": ... } at minimum, {} if no vault
```

**Hook signatures:**

```typescript
// shell.env hook
"shell.env": (
  input: { cwd: string; sessionID?: string; callID?: string },
  output: { env: Record<string, string> }
) => Promise<void>;
// Sets output.env.AGENT_MEMORY_VAULT = vaultPath if not already set
```

**Dependencies:** None (foundation slice).

---

### Slice 2: memory-write Tool

**Goal:** Register `memory-write` tool that shells out to `agent-memory write-note --json`.

**New files:**

```
opencode-plugin-agent-memory/src/tools/
└── memory-write.ts
```

**Type signatures:**

```typescript
// src/tools/memory-write.ts
import type { ToolDefinition } from "@opencode-ai/plugin";
import type { BunShell } from "@opencode-ai/plugin/dist/shell";

// Parameters schema (using tool.schema / z)
interface MemoryWriteParams {
  title: string;
  body: string;
  type: "observation" | "pattern" | "constraint" | "decision" | "assumption";
  confidence: "low" | "medium" | "high";  // default "medium"
  domain?: string[];
  project?: string;
  scope: "project" | "cross-project";     // default "project"
  source_artifact?: string;
}

// Return type mirrors Go WriteResult
interface WriteResult {
  status: "written" | "refused" | "error";
  path?: string;
  reason?: string;
  warnings?: string[];
  candidates?: Array<{ path: string; title: string; similarity: number }>;
  error?: string;
}

export function createMemoryWriteTool(
  $: BunShell,
  vaultPath: string,
  onWrite: () => void   // cache invalidation callback
): ToolDefinition;
```

**Dependencies:** Slice 1 (needs `vaultPath`, `$` from plugin init).

---

### Slice 3: memory-recall Tool (Fallback Error)

**Goal:** Register `memory-recall` tool. Returns graceful error until `agent-memory search` CLI exists.

**New files:**

```
opencode-plugin-agent-memory/src/tools/
└── memory-recall.ts
```

**Type signatures:**

```typescript
// src/tools/memory-recall.ts
import type { ToolDefinition } from "@opencode-ai/plugin";
import type { BunShell } from "@opencode-ai/plugin/dist/shell";

interface MemoryRecallParams {
  query: string;
  project?: string;
  domain?: string;
  type?: "observation" | "pattern" | "constraint" | "decision" | "assumption" | "synthesis";
}

export function createMemoryRecallTool(
  $: BunShell,
  vaultPath: string
): ToolDefinition;
```

**Dependencies:** Slice 1.

---

### Slice 4: system.transform Context Injection

**Goal:** Inject vault context into system prompt on every LLM call. Cached with 60s TTL. Invalidated on write.

**New files:**

```
opencode-plugin-agent-memory/src/
└── context.ts
```

**Type signatures:**

```typescript
// src/context.ts
import type { BunShell } from "@opencode-ai/plugin/dist/shell";

interface ContextCache {
  get(sessionID: string): string | undefined;
  invalidateAll(): void;
}

export function createContextCache($: BunShell, vaultPath: string): ContextCache;

export function createSystemTransformHook(
  cache: ContextCache
): (
  input: { sessionID?: string; model: unknown },
  output: { system: string[] }
) => Promise<void>;
```

**Dependencies:** Slice 1 (vault path, shell). Slice 2 calls `cache.invalidateAll()` on successful write.

---

### Slice 5: Event Hook — Session-Idle Promotion

**Goal:** On `session.idle`, debounce then run `agent-memory promote` for each inbox note.

**New files:**

```
opencode-plugin-agent-memory/src/
└── promotion.ts
```

**Type signatures:**

```typescript
// src/promotion.ts
import type { Event } from "@opencode-ai/sdk";
import type { BunShell } from "@opencode-ai/plugin/dist/shell";

export function createEventHook(
  $: BunShell,
  vaultPath: string,
  debouncMs: number,
  enabled: boolean
): (input: { event: Event }) => Promise<void>;
```

**Dependencies:** Slice 1.

---

## Final File Tree

```
opencode-plugin-agent-memory/
├── src/
│   ├── index.ts           # entry point, assembles hooks
│   ├── vault.ts           # resolveVault()
│   ├── context.ts         # cache + system.transform hook
│   ├── promotion.ts       # event hook (idle promotion)
│   └── tools/
│       ├── memory-write.ts
│       └── memory-recall.ts
├── dist/
│   └── index.js           # bun build output
├── package.json
└── tsconfig.json
```

## Dependency Graph

```
Slice 1 (scaffold)
  ├── Slice 2 (memory-write)  ──invalidates──▶ Slice 4
  ├── Slice 3 (memory-recall)
  ├── Slice 4 (context injection)
  └── Slice 5 (idle promotion)
```

All slices depend on Slice 1. Slice 4 has a runtime coupling with Slice 2 (cache invalidation callback). Slices 2, 3, 5 are independent of each other.
