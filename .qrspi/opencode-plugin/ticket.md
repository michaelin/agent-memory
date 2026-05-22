# Increment 6a: OpenCode Plugin (`opencode-plugin-agent-memory`)

## Summary

Build an OpenCode plugin (npm/TypeScript package) that makes the memory system dynamically discoverable by all agents. No agent definition needs modification — the plugin injects context, registers tools, and triggers promotion automatically.

## What it does

### Context Injection (`experimental.chat.system.transform` hook)
- Runs on every LLM call
- Calls `agent-memory context --json` to get vault state
- Appends to system prompt: memory overview, constraints summary, stale notes, epistemic type guide, tool usage instructions
- Target: < 600 tokens total injection
- Caches per session, invalidated on write

### Tool Registration (`tool` hook)
- **`memory-recall`**: query, project, domain, type filters → JSON array of matching note metadata
- **`memory-write`**: title, body, type, confidence, domain, project, scope, source_artifact → writes note to inbox, returns slug/path/warnings

### Session-Idle Promotion (`event` hook)
- Listens for `session.idle` events
- Debounces (default 30s) before triggering
- Checks `_inbox/` for unprocessed notes
- Invokes Librarian via `librarian-workflow` skill

### Environment Wiring (`shell.env` hook)
- Sets `AGENT_MEMORY_VAULT` in all shell invocations

## Prerequisites
- Increment 6 (`agent-memory context`) and Increment 7 (`agent-memory reindex`) must be complete
- Plugin calls these CLI commands internally

## Design references
- `docs/AGENT_MEMORY_DESIGN.md` §5.9 (OpenCode plugin design)
- `docs/ROADMAP.md` Increment 6a
- Plugin API types: `~/.config/opencode/node_modules/@opencode-ai/plugin/dist/index.d.ts`
- Existing plugin examples: `opencode-plugin-openspec`, `@mohak34/opencode-notifier`
