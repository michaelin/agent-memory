# Agent Memory Design: Persistent Obsidian-Based Knowledge Store

*Analysis of requirements, prior art, and recommended design for an
out-of-repo persistent memory layer shared across coding agents and
frameworks. The system is intentionally agent-agnostic: any framework or
role can read and write the vault, with the sole exception of the
Librarian, which is the one role this design defines.*

**Status:** Design complete. All open questions resolved. Ready for implementation.

---

## 1. Why This Matters — Context from Your Current Setup

Coding agents are typically stateless between sessions. What an agent
learns in one session dies when the session ends. Whatever per-repo
persistence exists — process artifacts, specs, decision records — is
bounded by the repo it lives in. None of it survives across **projects**,
and none of it records the informal, hard-won knowledge that accumulates
during development: "that
pattern turned out to be a bad idea", "the security team always flags X",
"this codebase's `context.go` is the entry point for everything".

That gap is what persistent agent memory addresses.

A related failure mode is **context rot**: as more context is loaded into a
session, LLMs become progressively less reliable at recalling information from
earlier in that context. The solution is not to load less — it is to load
*selectively*. The memory system described here is designed around that
constraint: small, targeted injections of the right knowledge at the right
time, not a full dump of everything the agent might ever need.

---

## 2. What You Actually Need — Requirements Analysis

Working from your architecture, here is what a memory system must satisfy:

### 2.1 Functional requirements

| Requirement | Why |
|---|---|
| **Out-of-repo storage** | Memory must survive repo deletion, project archival, and new repo creation. It spans projects. |
| **Obsidian-compatible markdown** | Enables human browsing, editing, and curation without any special tooling. Wiki-links and tags let you navigate the graph. |
| **Atomic short notes** | Each note encodes one fact, one pattern, or one decision. Atomic notes are easier to validate, update, and deprecate. |
| **Machine-readable frontmatter** | Agents need to filter by project, status, confidence, and date without reading every file. |
| **Multi-agent write safety** | Multiple agents (often running in parallel) must not create conflicting or redundant notes without a reconciliation step. |
| **Cross-project patterns** | A pattern discovered in project A should be retrievable when starting project B. |
| **Human curatable** | You must be able to browse, dispute, correct, and prune the vault in Obsidian directly. |
| **Tool-agnostic accessible** | Any agent framework on the same machine (OpenCode, pi, future tools) must be able to discover and use the same vault without per-tool configuration. |

### 2.2 Knowledge hygiene requirements (the hard part)

This is where most agent memory systems fail. The specific failure modes
to defend against:

1. **Stale facts** — something that was true in April is still in memory in October
2. **Assumption poisoning** — an agent writes a guess as a fact; future agents treat it as ground truth
3. **Conflicting writes** — two agents write opposite conclusions about the same thing; later agents pick whichever they find first
4. **Citation collapse** — a note loses track of where the claim came from; it becomes impossible to verify
5. **Hallucination laundering** — an agent writes something it hallucinated into the vault; it gets retrieved as verified knowledge in future sessions

---

## 3. Theoretical Foundations

The design rests on four ideas borrowed from different fields and composed
together. Understanding them explains why the system is structured the way
it is — and why simpler approaches fail.

---

### 3.1 Epistemology — not all knowledge is equal

The system distinguishes six **epistemic types**: observation, pattern,
constraint, decision, assumption, synthesis. This is not taxonomy for its
own sake. Each type carries a different claim about how the knowledge was
produced and how much it should be trusted.

An **observation** is a direct read of the codebase — something an agent
saw. A **pattern** is an inference from multiple observations — something
an agent concluded. A **constraint** is an external imposition — something
the system must respect regardless of what agents prefer. A **decision** is
a deliberate choice made by a human — it has authority, not just evidence.
An **assumption** is a belief held without verification — it is explicitly
marked as unreliable. A **synthesis** is a living entity or concept page
that compounds many atomic notes into a single navigable surface — it is
derived, not ground truth, and is owned by the Librarian.

The governance rules (TTL, promotion path, human confirmation requirements)
flow directly from these distinctions. Assumptions expire in 30 days because
they are unreliable by definition. Decisions require human confirmation
because they cannot be derived by agents. Observations auto-promote because
the codebase is the ground truth — a wrong observation fails fast. Synthesis
pages have no TTL because they are regenerable from their contributing notes.

This is applied epistemology: the system encodes *how you know what you
know*, not just *what you know*.

---

### 3.2 Cognitive science — the context window is working memory

LLMs have a fixed context window. Loading everything into it at session
start is the equivalent of trying to hold an entire project in working
memory simultaneously — humans cannot do it either, and neither can models.
Performance degrades as the window fills; this is the context rot problem.

The solution mirrors how human experts actually work: they do not memorise
everything. They maintain a small set of always-active knowledge
(constraints, current project state) and retrieve specific details on demand
when a task requires them.

The three-tier architecture (Core → Index → Archival) is a direct
translation of this:

- **Core** is always in context — like the things an expert never has to
  look up
- **Index** is loaded at session start — like scanning a table of contents
  before starting work
- **Archival** is fetched on demand — like pulling a reference book off the
  shelf when you need it

The hard token budgets (600 for Core, 300 per index) are not arbitrary.
They are sized to leave the majority of the context window free for actual
work. The simple frontmatter/tag filter model (discover via frontmatter fields and tags, then read files directly) enforces this discipline mechanically — agents cannot accidentally load more than they need. A two-phase retrieval pattern (Phase 1 = frontmatter-only discovery, Phase 2 = selective body read with a hard cap) is a planned enhancement for when vault size makes it necessary; see §4.3 and §9.3.

---

### 3.3 Zettelkasten — atomic notes as the unit of knowledge

Niklas Luhmann's Zettelkasten method (1960s–90s) established that knowledge
compounds faster when stored as atomic, linked notes rather than long
documents. One note, one claim. Notes link to each other; the graph of
links is itself knowledge.

The failure mode of long documents under agent writes is well-documented: a
single file becomes a 500-line blob, different agents append contradicting
claims to the same section, and the file becomes untrustworthy as a whole
even if parts of it are correct. You cannot deprecate a paragraph.

Atomic notes solve this. Each note has a single claim that can be
individually verified, updated, or deprecated without touching anything
else. The wikilink graph surfaces relationships. Obsidian makes the graph
navigable by humans without any special tooling.

---

### 3.4 Information hygiene — knowledge has a half-life

The fourth idea is the least glamorous and the most important in practice:
**knowledge goes stale**. A fact that was true in April may be false in
October. An assumption that was reasonable at project start may be
invalidated by a decision made in week three.

Most agent memory systems ignore this. They write facts and never revisit
them. The vault gradually fills with outdated beliefs that future agents
treat as current truth — this is knowledge poisoning, and it compounds
silently.

The TTL system (`review-by` dates), the staleness scan at session start,
and the supersession protocol (never delete, only deprecate with a link
forward) are all responses to this. They encode the assumption that
knowledge decays and build the maintenance obligation into the structure of
every note from the moment it is written.

---

### 3.5 How the four ideas compose

Epistemology tells you *what kind* of knowledge you have. Cognitive science
tells you *how much* to load and *when*. Zettelkasten tells you *how to
structure* it so it stays maintainable. Information hygiene tells you *how
to keep it honest* over time.

None of the four is sufficient alone:

- A system with good structure but no epistemics treats a guess the same as
  a verified fact.
- A system with good epistemics but no hygiene accumulates stale truths.
- A system with good hygiene but no cognitive architecture floods the
  context window and degrades model performance.
- A system with good cognitive architecture but no atomic structure cannot
  deprecate individual claims without corrupting the whole document.

The design is the composition of all four. Every mechanism in §6 traces
back to one or more of these foundations.

---

## 4. What Others Have Built — Prior Art Review

### 3.1 Basic Memory ⭐ Vault format reference

**Repo:** `basicmachines-co/basic-memory`  
**What it is:** An open-source MCP server that gives AI assistants persistent
memory through Obsidian-compatible markdown files. The vault format and note
structure are the direct inspiration for this design.

**How it works:**
- Stores notes as markdown with wiki-links, tags, and frontmatter
- Exposes MCP tools: `write_note`, `read_note`, `search_notes`, `find_related`
- The vault is a standard directory openable in Obsidian simultaneously
- Hybrid semantic search (BM25 + local vector embeddings, no API required)
- Multiple projects supported via `--project` flag
- Write governance: **fully autonomous** — no approval gate; governance is
  "human can edit files locally"

**What makes it proven:**
- Actively maintained (v0.20.3), 2.9k stars, substantial community adoption
- Designed exactly for the "AI + human in same vault" use case
- Notes written by agents are indistinguishable from human-written notes in Obsidian
- Local-first by default; cloud sync is opt-in

**Gap analysis — what Basic Memory does NOT cover:**

| This design's requirement | Gap in Basic Memory |
|---|---|
| Epistemic typing with per-type promotion rules | ❌ Has `[category]` tags on observations but no first-class epistemic type field |
| TTL / `review-by` dates with staleness scanning | ❌ No concept of note expiry or staleness |
| `_inbox/` staging with auto-promotion logic | ❌ Writes directly to vault; no staging layer |
| `_contested/` area for contradictions | ❌ No conflict detection or contested staging |
| `confidence` field tied to upstream quorum | ❌ No confidence scoring |
| MCP access from OpenCode | ❌ Blocked until OpenCode gains native MCP support |

**Bottom line:** Use Basic Memory's vault format and `NOTE-FORMAT.md` as the
canonical reference for note structure and wiki-link conventions. The governance
layer (epistemic types, TTL, staging, contested area) does not exist off the
shelf anywhere — it is genuinely new design. When OpenCode gains MCP support,
the Basic Memory server becomes a drop-in search and graph backend with no
vault migration required.

---

### 3.2 Memory Bank (Cline / Cursor pattern)

**Origin:** Popularized by a widely-shared blog post, now documented in Cline's
official docs as "Memory Bank".

**How it works:**
- A set of 6 structured markdown files at `.memory/` or `docs/memory/`
- Files: `projectbrief.md`, `productContext.md`, `systemPatterns.md`,
  `techContext.md`, `activeContext.md`, `progress.md`
- Agent reads all files at session start; updates relevant files after each session
- Write governance: **fully autonomous** — users almost never review writes;
  implicit review happens at PR time via git diff

**What makes it proven:**
- Extremely simple to implement — no tooling, just file reads/writes
- Widely adopted; many people report it solving their "agent amnesia" problem
- The file categories map well to the kinds of context agents actually need

**Key limitations:**
- Monolithic per-file, not atomic per-fact. One file becomes a 500-line blob.
- Single-project, not cross-project
- No staleness detection or confidence tracking
- Doesn't use Obsidian's graph/linking features
- Race conditions when multiple agents write simultaneously

**Bottom line:** Good starting point for a single-agent, single-project setup
but doesn't scale to your multi-team, multi-project architecture. The file
taxonomy is worth borrowing; the format isn't.

---

### 3.3 MemGPT / Letta

**What it is:** A research project (now productized as Letta) that models agent
memory explicitly: core memory (always in context), archival memory (retrieved
on demand), and recall memory (recent history).

**What makes it proven:**
- Rigorous theoretical model of memory tiers
- Handles the context window overflow problem explicitly (your agents hit this)
- The core/archival distinction maps directly to "always load this" vs. "retrieve if relevant"

**Key limitations:**
- Requires running a Letta server
- Memory is in a database, not Obsidian-compatible markdown
- Not human-browsable without custom tooling
- Tight coupling to the Letta agent framework

**Bottom line:** The **memory tier model** (core vs. archival vs. recall) is the
right mental model to steal. The implementation doesn't fit your setup.

---

### 3.4 mem0

**What it is:** An open-source Python library providing a memory layer for AI
agents, backed by vector search + optional graph storage.

**What makes it proven:**
- Multi-agent memory: different agents have separate memory spaces
- Automatic conflict detection via embedding similarity
- Tracks memory evolution over time (update vs. append vs. contradict)
- Write governance: **fully autonomous** — `memory.add()` with no approval step;
  conflict detection is embedding-similarity-based, not human-gated

**Key limitations:**
- Vector database required (Qdrant, Chroma, etc.)
- Not human-readable without a separate UI
- Not Obsidian-compatible
- Python-only SDK

**Bottom line:** The **conflict detection approach** is worth studying.
The implementation is wrong for your use case.

---

### 3.5 Obsidian-native AI workflows (community)

Several practitioners have published workflows for using Obsidian as the
sole memory store for AI agents, operating purely on the filesystem:

**Common patterns observed across these setups:**

1. **Zettelkasten-style atomic notes** — one fact per file, linked via `[[wiki-link]]`
2. **Frontmatter as structured metadata** — agents filter by `status`, `project`, `tags` via grep/find before reading content
3. **Append-only writes for new information** — agents never overwrite; they create new notes or link to the old one and flag it `deprecated`
4. **`_inbox/` staging area** — new agent writes land in `_inbox/` as a visibility buffer; auto-promoted on TTL expiry unless challenged
5. **`_contested/` area for conflicts** — when an agent finds contradicting information, it writes to `_contested/` and links both claims
6. **Git-backed vault** — the vault is a git repo, giving a full audit trail of every write

**The Zettelkasten discipline is key.** Short, atomic, linked notes are
much more maintainable under agent writes than long summary documents, because:
- Each note has a single claim that can be individually verified or deprecated
- Links naturally surface when two pieces of knowledge are related
- Obsidian's graph view makes knowledge structure visible at a glance

---

## 5. Core Concepts

This section explains the key mechanisms the design relies on. Each concept
is described independently so it can be understood, evaluated, and implemented
without reference to any specific agent framework.

---

### 4.1 Context injection and session initialisation

**The problem:** An agent that starts a session without memory context is
effectively amnesiac. It will re-derive facts it already knows, miss
constraints it should respect, and repeat mistakes it has made before. The
solution is to inject a small, targeted summary of relevant knowledge at the
start of every session — before the agent does any work.

**The concept:** At session start, the agent makes a single tool call
(`agent-memory context`) that returns four things in one bundled payload:
1. **Core knowledge** — the writing protocol and a one-line-per-constraint
   summary of all active constraints. Always loaded, unconditionally.
   Target size: under 600 tokens combined.
2. **Index notes** — lightweight navigation files listing what knowledge
   exists for the current project and relevant domains. Loaded by name,
   not by search. Target size: under 300 tokens each.
3. **Staleness list** — notes where `review-by < today`, surfaced before
   work begins so the agent knows what may no longer be trustworthy.
4. **Recent log tail** — the last ~20 entries from `_meta/log.md`, giving
   the agent a cheap window onto what has changed in the vault since the
   last session.

The agent does not load full note bodies at session start. It loads the
bundle, then fetches specific notes on demand during work. This is the
critical discipline that prevents context rot: the session starts lean and
grows only as needed.

**Harness-agnostic implementation (baseline):**
The agent's definition file (system prompt) instructs it to call the
`agent-memory context` tool as the first action of every session, before any
other tool call. This relies on the model following instructions reliably
— which well-prompted models do consistently. It is the correct default
for any framework.

**Harness-specific enhancements:**
Some frameworks provide hook mechanisms that fire automatically at defined
points in the session lifecycle, independent of model instruction-following.
These are enhancements, not requirements:

| Framework | Hook mechanism | How to use it |
|---|---|---|
| **Claude Code** | `pre-tool-use` hook in `settings.json` | Run a shell script that injects the Core tier and project index before the first tool call |
| **Claude Code** | `user-prompt-submit` hook | Inject top semantic matches from the vault into every prompt (requires a local search process) |
| **Claude Code** | `session-end` hook | Trigger the Librarian's auto-promotion pass after each session |
| **OpenCode** | No hook API currently | Rely on agent instruction baseline |
| **Cursor / Windsurf** | Rules files (`.cursorrules`, etc.) | Include a `agent-memory context` instruction in the rules file |
| **Copilot CLI** | No hook API | Rely on agent instruction baseline |

**Recommendation:** Implement the agent-instruction baseline first. It works
everywhere and requires no framework-specific configuration. Add Claude Code
hooks as an optional enhancement if you find the baseline is being skipped
under context pressure. Do not build the system's correctness guarantees on
hooks — they are an optimisation, not a foundation.

---

### 4.2 The index file pattern

**The problem:** Loading all memory files into context at session start causes
context rot. But if agents don't know what notes exist, they can't search for
them. The index file pattern resolves this tension.

**The concept:** Each domain and project has a lightweight index note in
`notes/`. An index note lists what knowledge exists — titles, one-line
summaries, and epistemic types — without including any note body content.
The agent loads the index at session start (cheap, bounded), then fetches
specific note bodies on demand (selective, controlled).

The index is the table of contents. The notes are the chapters. You load the
table of contents first; you only open a chapter when you need it.

**Index note structure:**
```markdown
---
title: "Index: Golang domain"
epistemic-type: index
domain: [golang]
status: verified
maintained-by: librarian
updated: 2026-04-24
---

# Golang Domain Index

## Constraints
- [[golang-error-handling]] — errors must be wrapped with context at domain boundaries

## Patterns
- [[golang-service-layer]] — service layer owns business logic; handlers own HTTP

## Observations
- [[gokit-middleware-order]] — GoKit middleware order affects auth context availability
```

**Size discipline:** An index note for a mature domain should fit in 300
tokens. Indices are regenerated from frontmatter scans by `agent-memory reindex`
(§9); they are not hand-curated. If an index grows beyond budget, the split
into sub-indices is configured in `agent-memory reindex` rules, not done
manually. The human can also create or edit index notes directly in
Obsidian — `agent-memory reindex` preserves human additions in a marked section.

**Orphan detection:** Notes in `notes/` that are not referenced from any
index are reported by `lint-vault --orphans`. `agent-memory reindex` adds them
to the appropriate index automatically when the domain/project tags make
the target unambiguous; otherwise they are surfaced for human triage.

**Harness-agnostic:** Index files are plain markdown. Any agent that can
read a file can use them. No framework-specific features required.

---

### 4.3 Two-phase retrieval

> **Deferred.** The current implementation of `agent-memory search` uses a simple
> frontmatter/tag filter model: agents filter by project, domain, type, and tag,
> then read note files directly for full content. Two-phase retrieval is deferred
> until the vault has enough data to validate whether the additional complexity is
> warranted. See §9.3 for the current implementation.

**The problem:** Fetching full note bodies for every potentially relevant note
is expensive and causes context overflow. Fetching nothing until explicitly
asked means the agent misses relevant knowledge. Two-phase retrieval is the
middle path.

**The concept:**
- **Phase 1 — Discovery:** grep frontmatter fields (title, tags, domain,
  epistemic-type) across `notes/`. Returns titles and frontmatter only —
  never body content. Cheap; proportional to vault size. Always filters to
  `status: verified`.
- **Phase 2 — Selective read:** the agent reasons about Phase 1 results and
  selects which notes to read in full. Hard cap: 10 note bodies per
  invocation. If more are needed, the agent issues a second narrower query.

Phase 1 answers "what exists?" Phase 2 answers "what does it say?" The agent
decides which notes are worth reading based on Phase 1 metadata before
committing to Phase 2 cost.

**Tag alias expansion:** Phase 1 expands query terms against
`_meta/tag-taxonomy.md` before searching. A query for `auth` also searches
`authentication` and `authz`. This prevents missed results from inconsistent
tagging across notes written by different agents.

**Harness-agnostic:** Both phases run inside the `agent-memory search` subcommand,
which an agent invokes as a single tool call. The subcommand uses standard
filesystem operations internally; the agent never runs raw `grep` and
never has to compose Phase 1 / Phase 2 logic in prompt.

---

### 4.4 Memory consolidation

**The problem:** Over time, the vault accumulates redundancy. Multiple
observation notes may describe the same fact from slightly different angles.
Index notes grow stale as notes are added without updating them. Empty or
trivial notes from early sessions persist and add noise. The vault gradually
becomes harder to navigate and more expensive to search.

**What is already covered by static tooling (§9):**

| Consolidation concern | Tool that handles it |
|---|---|
| Update index notes; remove deprecated/superseded references | `agent-memory reindex` |
| Promote validated `_inbox/` items to `notes/` | `agent-memory promote` (called by the Librarian; no TTL-based auto-promote — `review-by` is a staleness indicator only) and `lint-vault --stale` |
| Identify pattern candidates from corroborating observations | Librarian maintenance pass (LLM judgment; no binary) |
| Detect notes not referenced from any index | `lint-vault --orphans` |
| Detect deprecated notes lacking a forward link | `lint-vault --deprecated-no-link` |

**What remains as future development:**

1. **Remove empty and trivial notes** — detection of notes whose entire
   content is already captured in another note. Requires semantic
   judgment beyond what the current similarity check provides.
2. **Merge duplicate observations** — two notes making the same claim
   merged into one with both source references preserved. Requires the
   same semantic judgment plus a deterministic merge format.
3. **Add cross-references** — suggest `[[links]]` between notes that
   reference the same entities but do not currently link.

These remaining items are best designed after observing how the vault
grows in practice. They will likely become additional flags on existing
subcommands (e.g. `agent-memory reindex --suggest-merges`, `agent-memory lint-vault
--missing-links`) rather than a new top-level subcommand.
---

### 4.5 Pattern promotion

**The problem:** Individual observations accumulate in the vault but no
single agent has visibility across sessions to notice when multiple
observations are corroborating the same underlying pattern.

**The concept:** Pattern detection is a Librarian task, not a deterministic
binary. On its maintenance pass (triggered by log-size threshold or human
request), the Librarian reads recent `observation` notes, applies LLM
judgment to spot clusters of corroborating claims, and drafts `pattern`
notes via `memory-write --type=pattern` for clusters it considers genuine.
No threshold tuning required — the Librarian uses the same judgment a
human reviewer would.

This is intentionally simpler than a deterministic clustering binary: a
cheap model (Haiku-class) reading ten recent observations and asking
"do any of these corroborate each other?" is more accurate and easier to
adjust than tuned Jaccard thresholds, and does not require a separate
binary with its own maintenance burden.

---

## 6. Recommended Design for Your System

This design borrows from Basic Memory (vault format), MemGPT (memory tiers),
and the Obsidian community (hygiene patterns). The governance layer is original.

### 5.0 Static-tooling principle

The single most important factoring rule of this design:

> **The tool owns the *form*. The agent owns the *content*.**

Anything structural, mechanical, or rule-based — frontmatter assembly, TTL
assignment, log writing, index regeneration, inbox promotion,
similarity-based search-before-write, source-artifact
verification, staleness scanning — lives in the Go tooling layer (§9).
These run as deterministic subcommands with JSON output, invoked by agents
as tool calls or by the host (cron, hooks, humans) outside any session.

The agent's per-session token budget is reserved for the work only an LLM
can do:

- Composing the claim itself (the body text).
- Picking the `epistemic-type` when ambiguous.
- Assessing evidence quality and confidence.
- Deciding whether two notes are *semantically* the same claim when titles
  differ.
- Drafting synthesis prose on top of a tool-assembled scaffold.
- Resolving genuine contradictions in `_contested/`.

Every feature in this design must pass the test: *can this be done
deterministically?* If yes, it goes in a binary, not in an agent prompt.
This principle is the lens for evaluating the rest of §6 and §9.

---

### 5.1 Vault location and configuration

The vault is created with `agent-memory init`, which follows a git-init
model: the vault is a self-describing directory whose presence at a path
*is* the configuration. No config file is written anywhere. The vault
directory is named `.agent-memory/`.

```bash
agent-memory init              # creates .agent-memory/ in the current directory
agent-memory init ~/vaults/work  # creates the vault at an explicit path
```

This mirrors how `.git/` works: the directory is the artifact. Any tool
that can find the directory can use the vault — no per-tool or per-project
config file to maintain.

**Global vault init (deferred — Increment 13):** The `init` command will
gain a `--global` / `-g` flag to create the vault under
`$XDG_DATA_HOME/agent-memory` (falling back to `$HOME/.local/share/agent-memory`
or `$HOME/.agent-memory`). An optional folder name argument replaces the
default name in the resolved path (e.g. `agent-memory init -g work` →
`~/.local/share/work/`). This is a stabilization feature, not on the
critical path.

**Template customization (deferred — Increment 14):** A config directory
(global at `$XDG_CONFIG_HOME/agent-memory/`, per-vault at
`.agent-memory/_config/`) will hold user-editable templates with
git-config-style inheritance (per-vault → global → embedded defaults).
`agent-memory init` will populate config with embedded defaults on first
run. Lint rules will need to either read valid values from config or
remain template-agnostic — that design decision is deferred to the
increment's QRSPI cycle. This is a stabilization feature, not on the
critical path.

**Vault discovery (for subcommands that need to locate an existing vault):**
The resolution order is:

1. `AGENT_MEMORY_VAULT` environment variable — explicit path; takes
   precedence unconditionally. Supports CI/CD contexts, headless agents,
   and cases where the home directory is not writable.
2. Walk up the directory tree from the current working directory, looking
   for a `.agent-memory/` directory — analogous to how `git` finds `.git/`.
3. `~/.local/share/agent-memory` — global fallback for vaults not
   associated with any particular working directory.

For v1, the vault path is always explicit: either the `AGENT_MEMORY_VAULT`
env var or the positional argument to `agent-memory init`. Directory
walking is the intended discovery mechanism for future subcommands that
operate on a vault without requiring the user to specify its path.

**`agent-memory instructions`:** Rather than placing an `AGENTS.md` file
inside the vault (which would couple the vault to a specific repo's
conventions), the `instructions` subcommand prints a ready-to-paste blurb
to stdout. Pipe it into whatever instruction file a given repo or framework
uses:

```bash
agent-memory instructions >> AGENTS.md
agent-memory instructions >> .cursor/rules
```

This keeps the vault self-contained and repo-instruction wiring explicit
and per-project.

**On confidentiality:** The vault is a plain local directory — no server, no
network, no sync unless you explicitly opt in. Obsidian's local vault mode
never touches the network. The vault is not a git repository — notes may
contain observations about client codebases and should not be committed to
any remote. Durability is handled by `agent-memory backup`.

---

### 5.2 Vault structure

```
.agent-memory/
├── _meta/
│   ├── writing-protocol.md      ← The rules agents MUST follow when writing
│   ├── tag-taxonomy.md          ← Canonical tags + aliases (auth → authentication, authz…)
│   ├── constraints-summary.md   ← Tool-regenerated synthesis of all active constraints
│   ├── status-lifecycle.md      ← Lifecycle stages + edit vs. supersession rules
│   └── log.md                   ← Append-only chronological log; written by tools, never by agents
│
├── _inbox/                      ← All agent writes land here first
│   └── {YYYY-MM-DD}-{slug}.md
│
├── _contested/                  ← Notes with unresolved contradictions
│   └── {slug}-CONTESTED.md
│
└── notes/                       ← All promoted notes, flat
    ├── {slug}.md                ← knowledge note (any epistemic type incl. synthesis)
    └── _index-{scope}.md        ← Tool-regenerated navigation index (per domain / project)
```

There is no `AGENTS.md` at the vault root. Repo instruction files are
wired separately, per project, using `agent-memory instructions` (see
§5.1). This keeps the vault portable and free of repo-specific coupling.

The folder structure is minimal and reflects **lifecycle stage only**, not
content type. Classification — domain, project, scope, epistemic type — lives
entirely in frontmatter. A note that is simultaneously a `constraint`, a
`golang` concern, and scoped to a particular project carries all three axes
as tags; no folder hierarchy can represent that without an arbitrary choice.

Consequently, there are no rename operations after a note's first promotion.
All subsequent re-classifications (adding a domain tag, changing epistemic
type, updating scope) are frontmatter edits on a file that never moves. Git
sees only content diffs.

---

### 5.3 Note format

Based on Basic Memory's `NOTE-FORMAT.md`, extended with governance frontmatter.
Every note written by an agent MUST follow this structure:

```markdown
---
title: "Concise factual claim or pattern name"
created: 2026-04-24
updated: 2026-04-24
review-by: 2026-07-24          # TTL by epistemic type (see §5.6); blank for synthesis
status: inbox                  # inbox | verified | deprecated | contested | superseded
confidence: medium             # low | medium | high
epistemic-type: observation    # observation | pattern | constraint | decision | assumption | synthesis
scope: project                 # project | cross-project
project: project-slug          # source project; cross-project notes leave this blank
domain: [golang, auth]         # technology domains this note applies to
source-agent: research/analyst-a
source-artifact: "<repo-relative path or URL of the artifact this claim came from>"
verified-by: ""                # agent or human who promoted to verified
verified-date: ""
requires-human-review: false   # set true by memory-write for constraint/decision; gates memory-promote
update-type: ""                # only set by memory-write on staged correction notes: supersedes
targets: []                    # only set on correction notes: [[note-slug]] being amended
tags: []                       # Librarian-assigned on maintenance pass; agents leave blank
---

# Title

One short paragraph. One claim, one pattern, or one decision. No padding.

## Evidence
- Where this was observed (file path, artifact, or external source)
- Specific line/section references where possible

## Implications
- What other agents need to know when acting on this
- Any conditions under which this might not apply

## Related
- [[note-slug]] — short description of relationship
```

**On `update-type` and `targets`:** these fields are blank on new
knowledge notes. They are populated automatically by `agent-memory write-note` only
when `--update=<slug>` targets a high-stakes note (`constraint`,
`decision`, `assumption`) and supersession staging is required. In-place
edits to `observation`/`pattern`/`synthesis` targets do not produce a
correction note at all — the tool edits the target directly and writes a
`## Change log` line. See §5.5.

**On `requires-human-review`:** set to `true` by `agent-memory write-note` for
`constraint` and `decision` writes. `agent-memory promote` will not promote
any note with this flag set; clearing it requires `memory-promote
--slug=<x> --confirmed` (issued by the human after `agent-memory curate`
review).

**On synthesis pages:** synthesis notes use a different body layout
because they are derived, not source. Required sections are
`## Synthesis` (the prose) and `## Contributing notes` (deterministically
assembled by `agent-memory synthesize`). The `## Evidence`/`## Implications`
sections are not required for synthesis; `agent-memory lint-note` exempts them.

**On `Related`:** agents write live `[[wiki-links]]` directly. All links
must resolve to existing notes — `agent-memory write-note` flags unresolved links in
the `warnings` array of its JSON response (write still proceeds, but the
agent is informed immediately). `agent-memory promote` will not promote any note
with unresolved links. `lint-vault --links` reports all unresolved links
across `notes/` and `_inbox/` on demand.

---

### 5.4 Memory tiers and retrieval architecture

See §4.1 (context injection) and §4.3 (two-phase retrieval) for the
concepts behind this section.

| Tier | Contents | When loaded | Budget |
|---|---|---|---|
| **Core** | `_meta/writing-protocol.md` + `_meta/constraints-summary.md` + tail of `_meta/log.md` | Every session start, unconditionally, via `agent-memory context` | Target: < 600 tokens combined |
| **Index** | `notes/_index-{project}.md` + `notes/_index-{domain}.md` for active domains | Session start, for current project + declared domains, via `agent-memory context` | ~300 tokens per index; total target < 1500 tokens |
| **Archival** | All other notes in `notes/` (including `synthesis` pages) | On demand during work, via `agent-memory search` | Two-phase retrieval; cap 10 bodies per query |

**Core tier is size-capped, not category-capped.** It never loads "all
constraint notes" — it loads `constraints-summary.md`, a single
tool-regenerated file that summarises every active constraint in one
line each. The recent `log.md` tail (last ~20 entries) gives the agent a
cheap window onto what has changed since the last session. If an agent
needs the full body of a specific note, it fetches that note via
`agent-memory search` Phase 2.

---

### 5.5 Write protocol (how agents write without poisoning the vault)

The write protocol is enforced by the `agent-memory write-note` tool (§9). The agent
is responsible only for choosing the epistemic type, providing the claim
body, declaring evidence and source, and selecting one of three flags:
`--new-claim`, `--update=<slug>`, or `--contest=<slug>`. Everything else —
frontmatter assembly, TTL assignment, timestamping, similarity check,
inbox placement, log writing, lint gating — happens inside the tool.

**What the agent does**

1. Decide the `epistemic-type`. If unsure, default to `observation`.
2. Write the claim body and the `## Evidence` section.
3. Declare `--source-artifact=<stable path or URL>`.
4. Choose one of:
   - `--new-claim` — the agent has searched (via `agent-memory search`) and is
     confident this is a new claim with no existing equivalent.
   - `--update=<slug>` — amends an existing note. The tool routes the
     correction by the *target's* epistemic type (see correction table
     below).
   - `--contest=<slug>` — contradicts an existing note. The tool moves
     both notes to `_contested/` and links them.

**What the tool does (every invocation of `agent-memory write-note`)**

1. **Similarity check.** Scan `notes/` and `_inbox/` for notes with overlapping
   project/domain and a high title-similarity score (Jaccard on normalised
   word tokens) against the incoming title. If a candidate is found and the
   agent did not pass `--update`/`--contest`/`--new-claim`, **refuse the
   write** and return the
   candidate list as JSON. The agent must reissue with an explicit flag.
2. **Frontmatter assembly.** Fill `created`, `updated`, `status: inbox`,
   `confidence` (defaults to `medium` if unset), `source-agent` (from
   environment), `verified-by`/`verified-date` (blank), `tags` (blank —
   the Librarian assigns tags on its maintenance pass), and `review-by`
   from the per-type TTL table (§5.6).
3. **Lint.** Invoke `agent-memory lint-note` against the assembled note. Refuse on failure.
4. **Constraint and decision gating.** For `epistemic-type: constraint` or
   `decision`, mark the note `requires-human-review: true` in frontmatter so
   `agent-memory promote` will not auto-promote it.
5. **Inbox write.** Write `_inbox/{YYYY-MM-DD}-{slug}.md`.
6. **Log entry.** Append one line to `_meta/log.md`:
   `## [YYYY-MM-DD HH:MM] write | <type> | <slug> | by:<agent>`.

**Inbox lifecycle (handled by the Librarian agent)**

The agent's responsibility ends at the inbox write. Promotion is handled
by the Librarian agent, invoked by the user agent as a skill at the end
of its workflow. The Librarian checks every inbox note for completeness,
uniqueness, and lint validity before promoting.

| Epistemic type | Promotion rule |
|---|---|
| `observation` | Librarian checks completeness, uniqueness, lint. Moves to `notes/`, `status: verified`. |
| `pattern` | Librarian checks completeness, uniqueness, lint. Promotes only if the note references 2+ corroborating notes. Moves to `notes/`, `status: verified`. |
| `assumption` | Librarian checks completeness, uniqueness, lint. Moves to `notes/`, `status: verified`. Stays as `epistemic-type: assumption` — it is not reclassified. User agents can later write an observation or other note that verifies or disproves the assumption; the Librarian removes outdated assumptions. |
| `constraint` | Librarian checks completeness, uniqueness, lint. Requires human confirmation (via agent tool call based on human response). Moves to `notes/`, `status: verified` only after confirmation. |
| `decision` | Same as constraint: Librarian checks, human confirms via agent tool call. |
| `synthesis` | Created by the Librarian (proactively during maintenance, or on agent request). Auto-promoted on creation. |

**Trigger model:** The user agent invokes the Librarian skill as the
last step of its workflow, after writing one or more notes. The
`agent-memory instructions` output tells agents about this flow. The
Librarian processes all pending inbox notes in a single pass.

**Future: decoupled service model.** The skill-invocation trigger is the
initial implementation. A future iteration may run the Librarian as a
background service (triggered by filesystem watch, session-end hook, or
cron) so that promotion does not block the user agent's workflow. The
Librarian's logic is the same in both models; only the trigger mechanism
changes.

**Decisions are vault-native.** Earlier drafts of this design defined an
ADR-conflict reconciliation protocol against a separate `docs/adr/` layer.
That layer is removed: the `decision` note in the vault, optionally backed by
an associated `synthesis` note for narrative context, is the single source
of truth. Projects that maintain their own decision records continue to do
so — those records are simply external sources that an agent may cite as a
`source-artifact` when writing a `decision` note. There is no special
reconciliation flow.

**Deprecation and supersession invariant**
When knowledge changes, the old note gets `status: deprecated` and a link
to the replacement. Deprecation without a replacement link is forbidden
and caught by `agent-memory lint-vault`. This preserves the audit trail of what agents
believed and when.

---

**Correction and amendment protocol — replacement only**

Notes are never edited in place. When knowledge changes, the user agent
writes a new note that replaces the old one. The new note references the
old note via `[[old-slug]]` in its `## Related` section. The Librarian
handles the lifecycle transition:

1. User agent writes a new note to `_inbox/` with updated content.
2. Librarian detects that the new note replaces an existing note (via
   similarity check, explicit reference, or agent-supplied metadata).
3. Librarian promotes the new note to `notes/` with `status: verified`.
4. Librarian sets the old note to `status: deprecated` with a forward
   link to the replacement.
5. For `constraint` and `decision` notes, the human confirms the
   replacement before the Librarian acts (via agent tool call).

The deprecated note is never deleted — future agents follow the link
forward. This preserves the full audit trail of what agents believed
and when.

**Assumption outdating:** When a user agent writes an observation or
other note that verifies or disproves an assumption, the assumption
becomes outdated. The Librarian detects this (via reference or
similarity) and removes the outdated assumption from `notes/`. An
archive mechanism may be introduced in a future increment.

---

### 5.6 Staleness prevention

Every note has a `review-by` date assigned automatically by `agent-memory write-note`
based on `epistemic-type`. This date is a **staleness indicator only** — it
signals when a note should be re-verified, not when it gets promoted.
Promotion is handled by the Librarian (§5.5), not by TTL expiry.

The default TTL values (configurable in a future increment):

| Epistemic type | Default TTL | Reasoning |
|---|---|---|
| `observation` | 90 days | Code changes; observations go stale |
| `pattern` | 180 days | Patterns are more stable but still evolve |
| `constraint` | 365 days | Constraints rarely change; if they do, it's a deliberate decision |
| `decision` | 365 days | Decisions are durable; revisited annually |
| `assumption` | 30 days | Assumptions must be challenged frequently |
| `synthesis` | none | Regenerable from contributing notes; no fixed expiry |

Search results include a `days_to_stale` field for each note: positive
values indicate days remaining until `review-by`, negative values indicate
days overdue. This lets agents assess trustworthiness without parsing dates.

The staleness scan runs deterministically inside `agent-memory context` at
session start (and as `agent-memory lint-vault --stale` on demand). Stale notes are
surfaced to the calling agent before work begins. No agent reasoning is
involved in detecting staleness.

---

### 5.7 Integration with agent roles

This design is deliberately agent-agnostic. The only role it defines is
the Librarian (§5.8). Every other agent in any framework participates
through the same tools:

- **Writing agents** — any agent that produces durable findings. Calls
  `agent-memory write-note` with the chosen epistemic type, claim body, evidence,
  and source-artifact. The tool handles everything else. As the last step
  of its workflow, the agent invokes the Librarian skill to process
  pending inbox notes.
- **Reading agents** — any agent that needs context. Calls `agent-memory context`
  once at session start to load Core + indices + staleness list + recent
  log tail in a single tool call. Can use `agent-memory search` for
  frontmatter/tag filtering to discover relevant notes, then read files
  directly for full content.
- **The Librarian** — invoked by user agents as a skill after writing
  notes. Handles promotion (all types), deduplication, assumption
  outdating, human confirmation for constraints/decisions, tagging,
  pattern detection, synthesis, and optional maintenance.

**Mapping to a multi-agent setup.** A typical hierarchy might map roles as:

- A Research role (potentially with a quorum of analysts) is the primary
  source of `observation` notes. When a quorum is used, divergence maps
  directly to `confidence`: unanimous → high, majority → medium, divergent
  → low. Unresolved contradictions are written with `--contest`.
- An Engineering or Implementation role writes `pattern` and `observation`
  notes when it discovers non-obvious implementation facts.
- A Security or Review role writes `constraint` notes (which are gated
  for human confirmation by the tool, not by the role).
- A Planner or Orchestrator role is primarily a reading agent: it loads
  context and consults search results to avoid contradicting verified
  constraints.

None of these mappings are required by the memory system. The vault does
not care which agent wrote a note; it cares about the epistemic type,
source-artifact, and confidence.

---

### 5.8 The Librarian agent

The Librarian is the one role this design defines. It is invoked by user
agents as a skill at the end of their workflow, after writing one or more
notes to `_inbox/`. The Librarian processes all pending inbox notes in a
single pass: validating, classifying, promoting, deduplicating, and
optionally running maintenance.

**Core responsibilities (every invocation):**

1. **Validate** — check each inbox note for completeness, uniqueness, and
   lint validity. Notes that fail are flagged with specific errors.
2. **Promote** — move validated notes from `_inbox/` to `notes/` with
   `status: verified`. For patterns, verify that 2+ corroborating notes
   exist. For constraints and decisions, request human confirmation via
   agent tool call before promoting.
3. **Deduplicate** — detect notes that duplicate or replace existing notes.
   Deprecate the old note with a forward link to the replacement.
4. **Outdate assumptions** — detect assumptions in `notes/` that have been
   verified or disproved by newly promoted notes. Remove outdated
   assumptions (archive mechanism deferred to a future increment).

**Maintenance responsibilities (Librarian decides when):**

The Librarian has instructions to assess whether maintenance is needed
based on vault state. It may choose to:

- Assign tags to untagged promoted notes using `agent-memory tag`.
- Detect pattern candidates by reading recent observation notes with
  LLM judgment; draft accepted candidates as pattern notes.
- Review synthesis gaps (many notes on the same topic without a synthesis
  page); create synthesis pages proactively or on agent request.
- Resolve notes in `_contested/` when the contradiction is non-trivial.
- Surface stale notes (past `review-by` date) for attention.

**Trigger model:**

The initial implementation uses skill invocation: the user agent calls
the Librarian skill as the last step of its workflow. The
`agent-memory instructions` output tells agents about this flow.

**Future: decoupled service model.** A future iteration may run the
Librarian as a background service triggered by filesystem watch,
session-end hook, or cron. This avoids blocking the user agent's
workflow when the vault is large. The Librarian's logic is identical
in both models; only the trigger mechanism changes. Statically
detectable triggers for the service model:
- New files in `_inbox/` (filesystem watch)
- `_contested/` directory non-empty
- `_meta/log.md` exceeds configured line-count threshold
- Human request

**Why no quorum on the Librarian:** The remaining judgments are either
bounded prose drafting or a single recommendation to the human. A second
Librarian instance adds latency without changing the outcome. Upstream
quorum (where used) has already established the underlying claim.

**Placement:** The Librarian is implemented as a Claude Code skill that user
agents invoke as the last step of their workflow. This is the primary interface:
skill invocation, not a standalone agent. There is no standalone agent definition
file and no team manifest entry for the Librarian (see §10: "No standalone agent
or team manifest"). If a backing agent definition file is introduced in a future
increment (e.g., to support non-skill invocation paths in other frameworks), this
section will be updated to name it explicitly. For now, the skill definition is
the sole artifact.

**Permissions:**
- `read`: allow (vault path)
- `write`: allow (vault path only)
- `edit`: allow (vault path only)
- `bash`: `agent-memory` subcommands on vault directory = allow;
  `git`/`grep`/`find` on vault directory = allow; `*` = deny
- `task`: deny (leaf subagent; does not spawn sub-agents)

**Tools the Librarian invokes:**

The Librarian uses the same agent-facing tools as everyone else
(`agent-memory context`, `agent-memory search`, `agent-memory write-note`), plus tools it has
exclusive access to:

- `agent-memory tag` — tag taxonomy management. Used to assign tags to untagged
  notes (`--assign --slug=<x> --tags=<a,b>`), accept new tags into the
  taxonomy (`--accept <tag> [--alias=<x,y>]`), and reject proposals
  (`--reject <tag> [--canonical=<existing>]`). Updates `_meta/tag-taxonomy.md`
  and appends a log entry.
- `agent-memory curate` — for each high-stakes inbox item (`constraint`,
  `decision`), presents the note alongside relevant existing notes and
  produces a structured promote/reject recommendation. The human's
  confirmation is relayed back via agent tool call.
- `agent-memory synthesize <entity>` (in draft mode) — generates a scaffold the
  Librarian then fills with prose. The non-draft mode of the same tool
  performs deterministic refresh and does not need the Librarian.

All other operations the Librarian might appear to do (writing the log,
updating an index, computing TTL, scanning staleness) are tool
side-effects, not Librarian work.

---

## 7. Preventing Knowledge Poisoning — Summary of Controls

| Threat | Control |
|---|---|
| Stale facts | TTL (`review-by`) + staleness scan at session start |
| Assumption treated as fact | Mandatory `epistemic-type: assumption` + 30-day TTL + agents forbidden from acting on assumptions without re-verifying |
| Conflicting writes | Search-before-write + `_contested/` staging for contradictions |
| Hallucination laundering | `_inbox/` visibility buffer + epistemic typing; `constraint`/`decision` types require Librarian review and human confirmation |
| Citation collapse | Mandatory `source-agent`, `source-artifact` fields |
| Agent overconfidence | `confidence: low/medium/high` — upstream quorum (where used) maps directly: unanimous = high, majority = medium, divergent = low |
| Silent deprecation | Never delete, only deprecate or supersede with a link to the replacement |
| Scope creep | Short atomic notes + tag taxonomy prevents sprawl |
| Context rot | Index-first loading; two-phase retrieval; Core tier hard-capped at 600 tokens |
| No human visibility | Vault is a standard Obsidian vault — human can browse, edit, and dispute at any time |

**Note on HITL scope:** Human confirmation is only required for `constraint`
and `decision` notes — the two types where a wrong memory can silently shape
agent behavior for months without triggering an obvious failure. For
`observation` and `pattern` notes, the codebase is the ground truth: wrong
memories fail fast (grep, tests, obvious errors). The community consensus
across all major coding agent memory systems (Basic Memory, Cursor/Cline Memory
Bank, mem0) is fully autonomous writes as the default. Requiring human review
for routine observations is over-engineering.

---

## 8. Implementation Sequence

The order is dictated by the static-tooling principle (§5.0): tools land
before the Librarian agent, because most of what earlier drafts assigned
to the Librarian is now a tool side-effect.

1. **Write `_meta/writing-protocol.md` first** — the constitution of the
   memory system. Reflects the static-tooling factoring: it tells agents
   what they are responsible for (claim, evidence, type, flag) and points
   at the tools that handle everything else.

2. **Build the Go tooling core** — `agent-memory init`, `agent-memory lint-note`,
   `agent-memory lint-vault`. Init scaffolds the vault (`_meta/` templates, `_meta/log.md`,
   directory structure). No config file is written; the vault is self-describing.

3. **Build the agent-facing trio** — `agent-memory context`, `agent-memory write-note`,
   `agent-memory search`. These three commands are the entire agent-side surface
   for normal operation. Until these exist, no agent can use the vault.

4. **Build the maintenance commands** — `agent-memory promote`, `agent-memory reindex`.
   With these in place, the inbox lifecycle and index maintenance run
   without any LLM involvement, on a `session-end` hook or cron.

5. **Define the Librarian agent** — thin escalation handler (§5.8). The
   definition is short because the responsibilities are short. Standalone
   global agent; not added to any team manifest.

6. **Build `agent-memory curate`** — Librarian-only; structures high-stakes
   inbox items (`constraint`, `decision`) into a promote/reject
   recommendation for the human.

7. **Seed `_meta/constraints-summary.md` and first index notes** — the
   first run of `agent-memory reindex` produces empty indices; seed them with
   any known constraints from existing project records. The Core and
   Index tiers become useful immediately.

8. **Wire reading agents to `agent-memory context`** — add a single
   session-start tool call (or framework-equivalent hook) to every agent
   definition that needs context. No further per-agent integration is
   required for reads.

9. **Wire writing agents to `agent-memory write-note`** — enable the tool for any
   agent that produces durable findings. The tool's refusal behaviour
   (`--new-claim`/`--update`/`--contest` required on similarity hit)
   teaches the agent the discipline; no per-agent prompt engineering
   needed.

10. **Build `agent-memory synthesize` and `agent-memory tag`** —
    deferrable until the vault has accumulated content. `agent-memory synthesize`
    produces deterministic scaffolds; the Librarian fills in prose only when
    invoked. `agent-memory tag` enables the Librarian's tagging maintenance pass.

11. **Backfill (optional)** — run reading agents over existing project
    artifacts to seed the vault with current knowledge. Use `agent-memory write-note`
    like any other agent.

---

## 9. Go CLI Tooling

The memory system ships as a single Go binary (`agent-memory`) with
subcommands, per [ADR-0001](adr/adr-0001-single-binary-with-subcommands.md).
The original design proposed separate binaries per tool; that was rejected
in favour of the single-binary pattern for simpler discovery, installation,
and version consistency.

All subcommands support `--json` for machine-readable output (JSON to
stdout, exit codes for pass/fail). Without `--json`, output is
human-readable. This dual-output pattern is inherited from the root
command.

### 9.1 Subcommand overview

| Subcommand | Audience | Purpose |
|---|---|---|
| `agent-memory init` | Human | Vault setup: scaffold directory, seed `_meta/` from embedded templates |
| `agent-memory instructions` | Human / agent setup | Print agent configuration blurb to stdout |
| `agent-memory write-note` | Agent | Single entry point for agent writes; enforces protocol deterministically |
| `agent-memory context` | Agent | Bundled session-start context (replaces multi-step session-init workflow) |
| `agent-memory search` | Agent | Simple frontmatter/tag filter for note discovery; returns metadata for matching notes (two-phase deferred — see §9.3) |
| `agent-memory promote` | Host (cron / hook / human) | Auto-promote inbox notes past TTL or with corroboration; no LLM |
| `agent-memory reindex` | Host | Regenerate `_index-*.md` and `_meta/constraints-summary.md` from frontmatter scan |
| `agent-memory synthesize` | Host or Librarian | Build/refresh a synthesis page scaffold for an entity |
| `agent-memory tag` | Librarian | Tag taxonomy management: assign tags, accept/reject proposals, update `tag-taxonomy.md` |
| `agent-memory curate` | Librarian | Structures a high-stakes inbox item for human confirmation |
| `agent-memory lint-note` | Agent / tool | Single-file validation; called as a hard gate by `write-note` |
| `agent-memory lint-vault` | Agent / host | Cross-file integrity (links, orphans, stale, source-artifact resolution, untagged, dense) |
| `agent-memory backup` | Human | Run `agent-memory lint-vault`; abort on findings; create timestamped `tar.gz` of vault |
| `agent-memory lint` | Human | Human-readable wrapper: runs `agent-memory lint-note` on all files + `agent-memory lint-vault`; pretty-prints findings |

The agent-facing trio (`context`, `write-note`, `search`) is the entire
interface a normal agent needs. Maintenance subcommands are invoked
outside sessions; the Librarian subcommands are invoked only on tool
escalations.

**Implemented so far:** `init`, `instructions`, `write-note`. Remaining
subcommands are planned for future increments.

### 9.2 Repository structure

```
agent-memory/
├── cmd/
│   └── agent-memory/            # single binary entry point (main.go)
├── internal/
│   ├── cli/                     # cobra command factories (root, init, write-note, etc.)
│   ├── note/                    # Note struct, frontmatter, Parse(), Serialize(), Slug(),
│   │                            #   Lint(), Write(), Jaccard similarity, wikilinks, source-agent
│   ├── vault/                   # vault discovery (Discover()), Init(), embedded templates
│   └── testutil/                # shared test helpers
├── test/
│   └── integration/             # integration tests (//go:build integration)
├── docs/
│   ├── adr/                     # architectural decision records
│   ├── AGENT_MEMORY_DESIGN.md   # this file
│   ├── ROADMAP.md               # increment roadmap
│   └── INDEX.md                 # document index
└── go.mod                       # module: github.com/michaelin/agent-memory
```

### 9.3 Agent-facing subcommands

#### `agent-memory context`

One call replaces the entire session-init workflow. Returns a single JSON
payload bundling everything an agent needs at session start.

```bash
agent-memory context [--project=<name>] [--domains=<a,b>] [--log-tail=20]
```

Returns:
```json
{
  "core": {
    "writing_protocol": "...",
    "constraints_summary": "..."
  },
  "indices": {
    "project": "...",
    "domains": {"golang": "...", "auth": "..."}
  },
  "stale_notes": [{"slug": "...", "review_by": "..."}],
  "recent_log": ["## [2026-04-24 10:13] write | observation | foo | by:research/analyst-a", ...],
  "budget": {"loaded_tokens": 1842, "deferred": []}
}
```

The budget gate (defer domain indices when total > 2000 tokens) runs
inside the subcommand; the agent does not need to reason about it.

#### `agent-memory write-note`

Single entry point for all agent writes. The agent supplies the content;
the tool handles the form.

```bash
agent-memory write-note \
  --type=<observation|pattern|constraint|decision|assumption> \
  --title=<...> \
  [--force] \
  [--project=<name>] [--domain=<a,b>] \
  [--source-artifact=<path-or-url>] \
  [--confidence=<low|medium|high>] \
  [--tags=<a,b>] \
  [<body-file> | - (stdin)]
```

The `--force` flag bypasses the similarity check. Body is provided as a
positional file argument or piped via stdin (use `-` explicitly). Frontmatter
is assembled from flags; the agent never writes raw frontmatter.

Returns:
```json
{
  "written": "_inbox/2026-04-24-foo.md",
  "log_entry": "## [2026-04-24 10:13] write | observation | foo | by:research/analyst-a",
  "requires_human_review": false,
  "warnings": [
    {"type": "unresolved-link", "link": "[[bar]]", "message": "target note does not exist; will block promotion until resolved"}
  ]
}
```

Or, on similarity hit without a flag:
```json
{
  "refused": true,
  "reason": "similar_existing_note",
  "candidates": [
    {"slug": "foo", "title": "...", "score": 0.87, "path": "notes/foo.md"}
  ],
  "hint": "reissue with --update=<slug>, --contest=<slug>, or --new-claim"
}
```

Side effects (always): assemble frontmatter, assign TTL, run lint checks,
write inbox file, append to `_meta/log.md`. Unresolved `[[wiki-links]]` in
the body are reported in the `warnings` array but do not block the write;
they block promotion.

**Update and contest modes (planned):** Notes are never edited in place.
When knowledge changes, the user agent writes a new replacement note to
`_inbox/`. The Librarian handles deprecation of the old note and
promotion of the replacement. The `--contest=<slug>` flag (staging a
competing claim in `_contested/`) is also planned.

#### `agent-memory search`

Simple frontmatter and tag filter tool for note discovery. Returns
metadata for matching notes; agents can then read files directly for
full content.

```bash
agent-memory search [<query>] [--type=<...>] [--project=<...>] [--domain=<...>] [--tag=<...>] [--status=<...>]
```

Returns a JSON array of matching notes with frontmatter fields and a
`days_to_stale` indicator (positive = days remaining until `review-by`,
negative = days overdue). The query, if provided, is a case-insensitive
substring match on the note title. All filter flags are optional and
combined with AND logic.

Agents are free to use this tool for discovery or to search the vault
directly via filesystem access. The tool is a convenience, not a gate.

**Deferred: two-phase retrieval model.** The original design proposed a
two-phase model (Phase 1 = frontmatter only, Phase 2 = body read with
10-note cap) to prevent context overflow in large vaults. This is
deferred until the vault has enough data to validate whether the
complexity is warranted. The simple filter model is sufficient for now.

### 9.4 Maintenance subcommands (no LLM in the loop)

#### `agent-memory promote`

Moves a validated note from `_inbox/` to `notes/` and sets
`status: verified`. Called by the Librarian after validation, not
directly by user agents.

```bash
agent-memory promote --slug=<slug> [--confirmed]
```

The `--confirmed` flag is required for `constraint` and `decision` notes
(human confirmation relayed via agent tool call). For all other types,
the Librarian calls `promote` after its own validation pass.

For all types: **refuses to promote any note with unresolved `[[wiki-links]]`
in its body**. The agent is notified at write time via the `warnings` array;
promotion is the hard gate.

Writes a promotion line to `_meta/log.md` for every action.

**Future: decoupled service model.** In the initial implementation, the
Librarian is invoked as a skill by the user agent and calls `promote`
directly. A future iteration may run the Librarian as a background
service, but the `promote` subcommand interface remains the same.

#### `agent-memory reindex`

Deterministically regenerates `_index-{project}.md` and
`_index-{domain}.md` files from a full frontmatter scan, plus regenerates
`_meta/constraints-summary.md` from all `status: verified`
`epistemic-type: constraint` notes. Idempotent. Replaces the entire
"Librarian maintains the indices" loop with a binary that runs in milliseconds.

#### `agent-memory tag` *(Librarian-only)*

Tag taxonomy management. The Librarian is the only agent that calls this.
Agents do not supply tags when writing notes; tags are assigned after promotion.

```bash
agent-memory tag --assign --slug=<x> --tags=<a,b>          # assign tags to an existing note
agent-memory tag --accept <tag> [--alias=<x,y>]            # add tag to taxonomy
agent-memory tag --reject <tag> [--canonical=<existing>]   # reject; optionally redirect to existing tag
agent-memory tag --list-untagged                           # list promoted notes with no tags
```

All operations update `_meta/tag-taxonomy.md` and append a log entry to
`_meta/log.md`. The Librarian runs `--list-untagged` on its maintenance pass
to find notes needing tags, then calls `--assign` for each one.

#### `agent-memory synthesize <entity>`

Given an entity slug or tag, gathers all `status: verified` notes that
reference it and emits a synthesis-page scaffold: contributing-notes
section (deterministically composed), backlink graph, gap markers. In
`--draft` mode it leaves a `## Synthesis` section blank for the Librarian
to fill with prose. In `--refresh` mode it updates only the
deterministic sections of an existing synthesis page, leaving the prose
untouched.

#### `agent-memory curate` *(Librarian-only)*

Processes one high-stakes inbox item (`constraint` or `decision`) at a
time. Reads the staged note, runs `agent-memory search` for related existing
notes, and emits a structured promote/reject recommendation as JSON for
the human:

```json
{
  "slug": "auth-tokens-rotate-90d",
  "type": "constraint",
  "recommendation": "promote",
  "reasoning": "...",
  "related_notes": [{"slug": "...", "relationship": "reinforces"}],
  "conflicts": []
}
```

The human's confirmation is the trigger for `agent-memory promote --slug=<x>
--confirmed`, which clears `requires-human-review` and moves the note to
`notes/`. `agent-memory curate` appends its own line to `_meta/log.md` on
every invocation (`curate | <type> | <slug> | rec:<promote|reject>`) so
the Librarian never writes the log directly.

### 9.5 Lint subcommands

#### `agent-memory lint-note`

```bash
agent-memory lint-note _inbox/2026-04-24-foo.md
# pass  → {"valid": true}
# fail  → {"valid": false, "errors": ["missing field: confidence", "unresolved link: [[foo]]"]}
```

Exit 0 on pass, 1 on failure. Called as a hard gate by `write-note`
and `promote`.

**Checks performed:**
- All required frontmatter fields present and non-empty
- `epistemic-type` is one of the allowed values (incl. `synthesis`)
- `status` is one of the allowed values
- `confidence` is one of the allowed values
- `created` and `updated` are valid ISO dates
- `review-by` is a valid ISO date and is in the future (warn if < 7 days);
  not required for `synthesis`
- `## Evidence`, `## Implications`, `## Related` sections present (not
  required for `synthesis`, which uses `## Synthesis` + `## Contributing notes`)
- No frontmatter fields with placeholder values

#### `agent-memory lint-vault`

```bash
agent-memory lint-vault                       # all checks
agent-memory lint-vault --links               # unresolved wikilinks (notes/ and _inbox/)
agent-memory lint-vault --orphans             # notes not referenced from any index
agent-memory lint-vault --stale               # notes where review-by < today
agent-memory lint-vault --source-artifacts    # source-artifact paths/URLs that no longer resolve
agent-memory lint-vault --deprecated-no-link  # status: deprecated without a forward link
agent-memory lint-vault --untagged            # promoted notes in notes/ with no tags
agent-memory lint-vault --contested           # any files present in _contested/
agent-memory lint-vault --dense               # tag+domain combos with 5+ notes and no synthesis page
```

Output: JSON array of findings. Exit 0 if clean, 1 if any findings.
`agent-memory backup` runs `agent-memory lint-vault` as a gate.

**Why Go, not grep:** Wikilink extraction requires markdown-context-aware
parsing. Grep fails on multiple links per line, `[[slug|alias]]` syntax,
links inside code blocks, and frontmatter boundary detection. A proper
parser handles all of these correctly.

### 9.6 Human-facing subcommands

| Command | What it does |
|---|---|
| `agent-memory init [path]` | Create vault at `path` or `.agent-memory/` in cwd; seed `_meta/` from embedded templates; write `_meta/log.md`; JSON output to stdout |
| `agent-memory instructions` | Output agent configuration blurb to stdout for piping into agent definitions or harness configs |
| `agent-memory write-note` | Agent write entry point: assemble frontmatter from flags, lint, similarity check, atomic file write |
| `agent-memory backup` | Run `agent-memory lint-vault`; abort on findings; create timestamped `tar.gz` of vault |
| `agent-memory lint` | Human-readable wrapper: runs `agent-memory lint-note` on all files + `agent-memory lint-vault`; pretty-prints findings |
| `agent-memory promote` | Human-readable wrapper around the promotion logic |
| `agent-memory reindex` | Human-readable wrapper around the reindex logic |

**`init` detail:**
- Path is a positional argument; defaults to `.agent-memory/` in the current working directory (git-init model)
- Seeds `_meta/` from Go `embed.FS` templates (no network, no external files)
- No `AGENTS.md` created in the vault — use `agent-memory instructions` to obtain the agent configuration blurb
- `--force` overwrites existing files; `--clean --force` removes and recreates the vault directory
- Idempotent without flags: re-running on an existing vault skips existing files
- JSON output to stdout on success

### 9.7 Configuration resolution

Priority order (highest to lowest):

1. `AGENT_MEMORY_VAULT` environment variable
2. Walk up from the current directory looking for `.agent-memory/` — the same
   discovery model git uses to find `.git/`.
3. *(Deferred — Increment 13)* `~/.local/share/agent-memory` — global
   fallback for vaults not associated with any particular working directory.

No config file is read or written. The vault is located entirely through the
environment and the filesystem. This means `agent-memory init` in a project
directory creates a project-scoped vault that all tools discover automatically
when run from within that directory tree, while a global vault at the fallback
path serves as the catch-all for invocations outside any project.

### 9.8 Session lifecycle (when each subcommand runs)

| When | What runs | Who triggers it |
|---|---|---|
| Session start | `agent-memory context` | Agent (one tool call) |
| During work | `agent-memory search`, `agent-memory write-note` | Agent, on demand |
| Session end (current model) | Librarian skill invocation: validate, promote, deduplicate inbox notes | User agent (last step of workflow) |
| Session end (future: decoupled model) | `agent-memory promote`, `agent-memory reindex` | Host hook / cron / human — **no agent involvement** |
| Log threshold crossed | Librarian maintenance pass: tag untagged notes, detect patterns, review synthesis gaps | Host/cron (log line-count check) |
| Daily / weekly | `agent-memory lint-vault` | Cron or human |
| On escalation | `agent-memory curate`, Librarian invocation | `requires-human-review: true` in `_inbox/`, `_contested/` non-empty, or human |
| On demand | `agent-memory synthesize`, `agent-memory backup` | Human |

---

## 10. Resolved Design Decisions

All questions from the initial design phase are closed.

| Question | Decision |
|---|---|
| Vault location | git-init model: `agent-memory init [path]` creates vault at `path` or `.agent-memory/` in cwd. Discovery: env var → walk up directory tree for `.agent-memory/` → `~/.local/share/agent-memory` fallback. No config file. |
| Cross-project scope | Cross-project and cross-agent from the start; all classification in frontmatter |
| Wikilinks | Live `[[wiki-links]]` written directly by agents; unresolved links reported as warnings by `agent-memory write-note`, hard-blocked at promotion by `agent-memory promote`; `lint-vault --links` covers both `notes/` and `_inbox/` |
| Epistemic types | Six types: observation, pattern, constraint, decision, assumption, synthesis |
| HITL scope | `constraint` and `decision` require human confirmation (via agent tool call) before Librarian promotes. All other types promoted by Librarian after completeness/uniqueness/lint checks — no TTL-based auto-promote. |
| Decision records | Vault-native; no separate ADR layer or reconciliation protocol. External decision records (if a project keeps them) are cited as `source-artifact`. |
| Static-tooling principle | Tool owns form, agent owns content; all deterministic work lives in Go subcommands |
| Tag assignment | Agents do not supply tags; tags are assigned by the Librarian on its maintenance pass using `agent-memory tag` |
| Log file | `_meta/log.md`, append-only, written by tools (`agent-memory write-note`, `agent-memory promote`, `agent-memory tag`), never by agents |
| Index maintenance | `agent-memory reindex` regenerates from frontmatter scan; not Librarian work |
| Promotion | Librarian-driven. User agent invokes Librarian skill after writing notes. Librarian checks completeness, uniqueness, lint, then promotes. No TTL-based auto-promote; `review-by` is a staleness indicator only. |
| Librarian role | Validation, classification, promotion (all types), deduplication, assumption outdating, human confirmation for constraints/decisions, tagging, pattern detection, synthesis, optional maintenance. |
| Librarian triggers | User agent invokes Librarian skill as last step of workflow. Future: decoupled background service (filesystem watch, session-end hook, cron). |
| Correction/amendment | Replacement only — no in-place edits. New note replaces old; Librarian deprecates old note with forward link. Full audit trail preserved. |
| Pattern detection | Librarian task using LLM judgment on its maintenance pass; no deterministic binary |
| Synthesis triggers | Librarian creates proactively during maintenance or on agent request; `lint-vault --dense` surfaces areas needing synthesis |
| Librarian placement | Skill invoked by user agents; defined as a Claude Code skill. No standalone agent or team manifest. |
| Agent-framework coupling | Design is framework-agnostic; only the Librarian role is defined here |
| Harness hooks | Agent-instruction baseline only; hooks documented as optional enhancements |
| Tooling | Go module `github.com/michaelin/agent-memory`; single binary with subcommands ([ADR-0001](adr/adr-0001-single-binary-with-subcommands.md)); agent-facing subcommands (`context`, `write-note`, `search`) + maintenance (`promote`, `reindex`, `tag`, `synthesize`, `curate`) + lint subcommands |
| Git in vault | No git; vault is plain files; durability via `agent-memory backup` |
| `git init` during init | Dropped; `init` is a plain file/folder scaffold from embedded templates |
| Module path | `github.com/michaelin/agent-memory` (public repo) |
| Memory consolidation | `agent-memory reindex` covers deterministic parts; richer consolidation deferred |
| Dreaming / pattern promotion | Librarian task during promotion pass; detects pattern candidates from recent observations using LLM judgment |

---

## Appendix: Key References

| Resource | What to take from it |
|---|---|
| `basicmachines-co/basic-memory` | Vault format, note structure, wiki-link conventions (`NOTE-FORMAT.md`) |
| MemGPT/Letta paper (2023) | Memory tier model (core/archival/recall) |
| Zettelkasten method (Luhmann) | Atomic note discipline — one claim per note |
| `mem0` conflict detection | How to detect and stage contradicting claims |
| Memory Bank (Cline docs) | Practical file taxonomy for agent context; autonomous write precedent |
| OpenClaw memory architecture | Daily notes + long-term memory + dreaming promotion pattern |
| Obsidian community "AI vault" threads | Real-world patterns from practitioners |
