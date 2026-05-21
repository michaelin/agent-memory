# Increment 1: Vault Scaffolding & Configuration

**Goal:** Establish the vault structure and configuration layer so all subsequent increments have a place to write.

**What becomes possible after this increment:**
- Agents can discover the vault location
- The vault structure is initialized and idempotent
- The writing protocol document exists (though it's mostly empty at this stage)
- Humans can browse the vault in Obsidian

## Configuration Resolution
- `AGENT_MEMORY_VAULT` environment variable (highest priority)
- `~/.config/agent-memory/config.json` (XDG_CONFIG_HOME if set, else `~/.config`)
- Fallback to `$HOME/.local/share/agent-memory` (XDG_DATA_HOME if set, else `~/.local/share`)
- Config file format:
  ```json
  {
    "vault": "/path/to/vault"
  }
  ```

## Vault Initialization (`agent-memory init`)
- **Idempotency:** Running `init` multiple times on the same vault path is safe
  - If vault already exists, verify structure is correct and exit 0
  - If vault exists but is corrupted (missing required directories), repair it
  - If vault exists but is from an older version, upgrade it (no-op for v1)
- Create vault directory structure:
  ```
  AgentMemory/
  ├── AGENTS.md                    <- Points to writing protocol
  ├── _meta/
  │   ├── writing-protocol.md      <- Rules agents MUST follow (mostly empty in v1)
  │   ├── tag-taxonomy.md          <- Empty in v1
  │   ├── constraints-summary.md   <- Empty in v1
  │   ├── status-lifecycle.md      <- Lifecycle stages (static template)
  │   └── log.md                   <- Append-only log (empty at init)
  ├── _inbox/                      <- All agent writes land here first
  ├── _contested/                  <- Notes with unresolved contradictions (empty at init)
  └── notes/                       <- All promoted notes, flat
  ```
- Seed `_meta/` files from embedded templates (no network, no external files)
- Create `AGENTS.md` at vault root pointing at `_meta/writing-protocol.md`
- Write configuration file to `~/.config/agent-memory/config.json` (or env var location)
- Log initialization to `_meta/log.md`: `## [YYYY-MM-DD HH:MM] init | vault initialized`

## Writing Protocol (v1 -- Minimal)
- Document exists at `_meta/writing-protocol.md`
- Content: "This vault is for agent memory. Writing is not yet enabled. See AGENTS.md for current capabilities."
- Will be expanded in Increment 2

## Verification (Integration Tests)
- **Idempotency:** Run `agent-memory init` twice on same path, verify no errors and vault state unchanged
- **Config resolution:** Test all three config sources (env var, config file, default)
- **XDG support:** Test with `XDG_CONFIG_HOME` and `XDG_DATA_HOME` set
- **Fallback:** Test that default path is used when no config exists
- **Vault structure:** Verify all required directories exist after init
- **Seed files:** Verify all `_meta/` files exist and contain expected content
- **AGENTS.md:** Verify it points to `_meta/writing-protocol.md`
- **Log entry:** Verify init is logged to `_meta/log.md`
- **Repair:** Corrupt vault (delete a directory), run init again, verify repair

## Design Reference
- See AGENT_MEMORY_DESIGN.md sections 5.1 (Vault location and configuration) and 5.2 (Vault structure)
- Config resolution in ROADMAP.md differs from design doc (ROADMAP uses XDG conventions; design doc uses `~/.agent-memory.json`). ROADMAP takes precedence.
