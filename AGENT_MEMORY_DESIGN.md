# Agent Memory Design: Persistent Obsidian-Based Knowledge Store

*Analysis of requirements, prior art, and recommended design for extending the
OpenCode multi-agent system with out-of-repo persistent memory.*

**Status:** Design complete. All open questions resolved. Ready for implementation.

---

## 1. Why This Matters — Context from Your Current Setup

Your system is designed to be stateless between sessions. Every agent file
has a "Context reset recovery" section precisely because continuity is
currently a problem. What agents learn in one session dies when the session
ends. The only persistence today is:

- `.qrspi/{feature-name}/` artifacts (per-feature, per-repo)
- `openspec/specs/` canonical specs (per-repo)
- `docs/adr/` accepted decisions (per-repo)

None of these survive across **projects**, and none of them record the
informal, hard-won knowledge that accumulates during development: "that
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
| **Multi-agent write safety** | Your Research Lead runs 3 analysts in parallel. They must not create conflicting or redundant notes without a reconciliation step. |
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

The system distinguishes five **epistemic types**: observation, pattern,
constraint, decision, assumption. This is not taxonomy for its own sake.
Each type carries a different claim about how the knowledge was produced
and how much it should be trusted.

An **observation** is a direct read of the codebase — something an agent
saw. A **pattern** is an inference from multiple observations — something
an agent concluded. A **constraint** is an external imposition — something
the system must respect regardless of what agents prefer. A **decision** is
a deliberate choice made by a human — it has authority, not just evidence.
An **assumption** is a belief held without verification — it is explicitly
marked as unreliable.

The governance rules (TTL, promotion path, human confirmation requirements)
flow directly from these distinctions. Assumptions expire in 30 days because
they are unreliable by definition. Decisions require human confirmation
because they cannot be derived by agents. Observations auto-promote because
the codebase is the ground truth — a wrong observation fails fast.

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
work. The two-phase retrieval pattern (discover via frontmatter grep, then
selectively read bodies) enforces this discipline mechanically — agents
cannot accidentally load more than they need.

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
| `confidence` field tied to Research Lead quorum | ❌ No confidence scoring |
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

**The concept:** At session start, the agent loads two things:
1. **Core knowledge** — the writing protocol and a one-line-per-constraint
   summary of all active constraints. This is always loaded, unconditionally.
   Target size: under 600 tokens combined.
2. **Index notes** — lightweight navigation files listing what knowledge exists
   for the current project and relevant domains. These are loaded by name, not
   by search. Target size: under 300 tokens each.

The agent does not load full note bodies at session start. It loads the index,
then fetches specific notes on demand during work. This is the critical
discipline that prevents context rot: the session starts lean and grows only
as needed.

**Harness-agnostic implementation (baseline):**
The agent's definition file (system prompt) instructs it to run the
`memory-read` `session-init` workflow as the first action of every session,
before any other tool call. This relies on the model following instructions
reliably — which well-prompted models do consistently. It is the correct
default for any framework.

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
| **Cursor / Windsurf** | Rules files (`.cursorrules`, etc.) | Include a memory-read instruction in the rules file |
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
tokens. If it grows beyond that, the Librarian splits it into sub-indices by
subtopic. The human can also create or edit index notes directly in Obsidian.

**Orphan detection:** Notes in `notes/` that are not referenced from any index
are invisible to session-start loading. The Librarian periodically checks for
orphans and adds them to the appropriate index, or flags them to the human if
the right index is unclear.

**Harness-agnostic:** Index files are plain markdown. Any agent that can read
a file can use them. No framework-specific features required.

---

### 4.3 Two-phase retrieval

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

**Harness-agnostic:** Both phases use `grep` and file reads — standard
filesystem operations available in any agent framework with bash access.

---

### 4.4 Memory consolidation *(future development)*

**The problem:** Over time, the vault accumulates redundancy. Multiple
observation notes may describe the same fact from slightly different angles.
Index notes grow stale as notes are added without updating them. Empty or
trivial notes from early sessions persist and add noise. The vault gradually
becomes harder to navigate and more expensive to search.

**The concept:** Periodic consolidation is a housekeeping pass over the entire
vault, distinct from the Librarian's per-note curation work. Where
`memory-curate` handles individual high-stakes notes at promotion time,
consolidation handles the vault as a whole on a scheduled basis.

A consolidation pass would:
1. **Remove empty and trivial notes** — notes with no body content, or notes
   whose entire content is already captured in another note
2. **Merge duplicate observations** — two notes making the same claim are
   merged into one, with both source references preserved
3. **Identify pattern candidates** — scan `_inbox/` and `notes/` for
   `observation` notes that share tags and make related claims; propose
   `pattern` note creation when 2+ corroborating observations are found
4. **Update index notes** — ensure every promoted note appears in the
   appropriate index; remove references to deprecated or superseded notes
5. **Resolve stale open threads** — find notes with `status: inbox` whose
   `review-by` date has passed and either auto-promote or flag for human review
6. **Add cross-references** — identify notes that reference the same entities
   and suggest `[[links]]` between them

**Why this is future development, not initial scope:**
Consolidation requires the vault to have accumulated enough content to make
the housekeeping meaningful. Running it on an empty or sparse vault adds
complexity with no benefit. The correct sequence is: build the vault, use it
for several months, then add consolidation once the noise problem is real
rather than theoretical.

Consolidation also requires careful design of the Librarian's judgment rules
for merging and pattern promotion — rules that are best derived from observing
how the vault actually grows in practice, not specified upfront.

**Harness-agnostic:** Consolidation is a Librarian workflow using grep, file
reads, and file writes. No framework-specific features required. It could also
be triggered by a cron job or a scheduled task outside any agent framework,
invoking the Librarian with a `consolidate` instruction.

---

### 4.5 The dreaming / pattern promotion mechanism *(future development)*

**The problem:** Individual observations accumulate in the vault but the
system has no automatic way to notice when multiple observations are
corroborating the same underlying pattern. A human reviewing the vault might
spot this, but agents working on individual tasks will not.

**The concept:** Inspired by OpenClaw's "dreaming" process, this is a
background pass that reads recent observations, scores them for recurrence,
and proposes `pattern` note creation when the evidence threshold is met.

The process:
1. Scan `notes/` and `_inbox/` for `observation` notes added in the last N
   days (configurable; default 30)
2. Group observations by shared tags and domain
3. Within each group, identify observations that make related or overlapping
   claims (by title similarity and tag overlap — no vector search required
   for a first implementation)
4. For any group with 2+ observations that appear to corroborate the same
   claim, draft a proposed `pattern` note and write it to `_inbox/` with
   `epistemic-type: pattern`, linking the source observations as evidence
5. Surface the proposed patterns to the human or Librarian for review before
   promotion

**Why this is future development, not initial scope:**
Pattern promotion requires enough observations to be meaningful. It also
requires tuning the similarity threshold — too aggressive and it creates
spurious patterns; too conservative and it never fires. Like consolidation,
this is best designed after observing how the vault grows in practice.

The initial design already handles the manual version of this: agents are
instructed to write `pattern` notes when they observe 2+ corroborating
instances. The dreaming mechanism automates the detection step for cases
where the corroborating observations were written by different agents in
different sessions and no single agent had visibility of both.

**Harness-agnostic:** The dreaming pass is a Librarian workflow. It could
run on demand, on a schedule, or be triggered by the consolidation pass.
No framework-specific features required.

---

## 6. Recommended Design for Your System

This design borrows from Basic Memory (vault format), MemGPT (memory tiers),
and the Obsidian community (hygiene patterns). The governance layer is original.

### 5.1 Vault location and configuration

The vault path is declared once in a tool-agnostic config file at a
well-known home-directory location:

```
~/.agent-memory.json
```
```json
{
  "vault": "~/Documents/Obsidian/AgentMemory"
}
```

This file is owned by no specific tool. Any agent framework — OpenCode,
pi, or anything else — reads it to discover the vault path. Skills in any
framework reference `~/.agent-memory.json` rather than a tool-specific
config directory. This means the same vault is shared automatically across
all frameworks on the machine with zero per-tool or per-project configuration.

**Environment variable override:** If `AGENT_MEMORY_VAULT` is set in the
environment, it takes precedence over `~/.agent-memory.json`. This supports
CI/CD contexts, headless agents, and cases where the home directory is not
writable.

```bash
export AGENT_MEMORY_VAULT="/path/to/vault"
```

Priority order: `AGENT_MEMORY_VAULT` env var → `~/.agent-memory.json` → error.

**On confidentiality:** The vault is a plain local directory — no server, no
network, no sync unless you explicitly opt in. Obsidian's local vault mode
never touches the network. The vault is not a git repository — notes may
contain observations about client codebases and should not be committed to
any remote. Durability is handled by `agent-memory backup`.

---

### 5.2 Vault structure

```
AgentMemory/
├── _meta/
│   ├── writing-protocol.md      ← The rules agents MUST follow when writing
│   ├── tag-taxonomy.md          ← Canonical tags + aliases (auth → authentication, authz…)
│   ├── constraints-summary.md   ← Librarian-maintained synthesis of all active constraints
│   └── status-lifecycle.md      ← Lifecycle stages + edit vs. supersession rules
│
├── _inbox/                      ← All agent writes land here first
│   └── {YYYY-MM-DD}-{slug}.md
│
├── _contested/                  ← Notes with unresolved contradictions
│   └── {slug}-CONTESTED.md
│
└── notes/                       ← All promoted notes, flat
    ├── {slug}.md                ← knowledge note
    └── _index-{scope}.md        ← Librarian-maintained navigation index (per domain / project)
```

The folder structure is minimal and reflects **lifecycle stage only**, not
content type. Classification — domain, project, scope, epistemic type — lives
entirely in frontmatter. A note that is simultaneously a `constraint`, a
`golang` concern, and scoped to `opencode-orchestrator` carries all three axes
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
review-by: 2026-07-24          # TTL by epistemic type (see §6.6)
status: inbox                  # inbox | verified | deprecated | contested | superseded
confidence: medium             # low | medium | high
epistemic-type: observation    # observation | pattern | constraint | decision | assumption
scope: project                 # project | cross-project
project: opencode-orchestrator # source project; cross-project notes leave this blank
domain: [golang, auth]         # technology domains this note applies to
source-agent: research/analyst-a
source-artifact: ".qrspi/auth-refresh/research.md"
verified-by: ""                # agent or human who promoted to verified
verified-date: ""
update-type: ""                # only set on correction notes: edit | supersedes
targets: []                    # only set on correction notes: [[note-slug]] being amended
tags: [golang, auth, middleware]
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

#tag1 #tag2
```

**On `update-type` and `targets`:** these fields are blank on new knowledge
notes. They are only populated when an agent is writing a *correction* to an
existing note (see §6.5 correction protocol). The distinction is intentional —
a correction note is not new knowledge; it is an instruction to the Librarian
to modify or supersede a specific existing note.

**On `Related`:** agents write live `[[wiki-links]]` directly. Unresolved
links (where the target note does not yet exist) are acceptable — Obsidian
renders them visually as broken links, making them easy to spot. The Librarian
flags accumulating unresolved links during curation passes.

---

### 5.4 Memory tiers and retrieval architecture

See §5.1 (context injection) and §5.3 (two-phase retrieval) for the concepts
behind this section.

| Tier | Contents | When loaded | Budget |
|---|---|---|---|
| **Core** | `_meta/writing-protocol.md` + `_meta/constraints-summary.md` | Every session start, unconditionally | Target: < 600 tokens combined |
| **Index** | `notes/_index-{project}.md` + `notes/_index-{domain}.md` for active domains | Session start, for current project + declared domains | ~300 tokens per index; total target < 1500 tokens |
| **Archival** | All other notes in `notes/` | On demand during R-phase, Q-phase, planning, implementation | Two-phase retrieval; cap 10 bodies per query |

**Core tier is size-capped, not category-capped.** It never loads "all
constraint notes" — it loads `constraints-summary.md`, a single
Librarian-maintained file that summarises every active constraint in one
line each. If an agent needs the full body of a specific constraint, it
fetches that note via Phase 2. This is what keeps the Core tier bounded
as the vault grows.

---

### 5.5 Write protocol (how agents write without poisoning the vault)

**Step 1 — Search before write**
Before writing any note, the agent MUST search the vault for existing notes
on the same topic (by tag + keyword). If a note already exists:
- Same claim → update the `updated` date; don't duplicate
- Related claim → add a suggestion in the `## Related` section body
- Contradicting claim → write to `_contested/`, link both, set both to `status: contested`

**Step 2 — Land in `_inbox/`, not directly in the hierarchy**
All agent writes go to `_inbox/`. This is a visibility buffer, not a mandatory
human gate. Notes auto-promote to their correct vault location when the
`review-by` TTL expires without challenge. You can intervene at any time,
but you are not required to.

**Step 3 — Epistemic type determines the promotion path**
Every note must declare its `epistemic-type`. The type controls how the note
is promoted from `_inbox/`:

| Epistemic type | Promotion action | Rationale |
|---|---|---|
| `observation` | Move to `notes/`, `status: verified`, on TTL expiry | Machine-verifiable; fails fast if wrong |
| `pattern` | Move to `notes/`, `status: verified`, after 2+ corroborating observations | Corroboration is the gate, not human review |
| `assumption` | Stays in `_inbox/`; agents MUST re-verify before acting | Unverified belief; short TTL forces re-examination |
| `constraint` | Move to `notes/`, `status: verified`, after Librarian review + human confirmation | Wrong constraint propagates silently everywhere |
| `decision` | Move to `notes/`, `status: verified`, after human confirmation; ADR conflict protocol applies | Not agent-derivable; expensive to reverse if wrong |

**Decision vs. ADR conflict protocol**

When the Librarian curates a `decision` note during promotion, it MUST check
`docs/adr/` in the relevant project repo for any ADR that covers the same
decision. When a conflict is found:

1. **The ADR wins immediately.** The vault note does not promote — it is
   deprecated with a link to the conflicting ADR.
2. **The conflict surfaces to the human as a distinct question**, separate
   from the deprecation: *"This vault decision contradicts ADR-{id}. Does
   this conflict suggest ADR-{id} needs revision?"*
3. **Human decides:**
   - *Yes, revise* → human initiates a new ADR via the existing `create-adr`
     process. The new ADR, once accepted, supersedes the old one. The vault
     note is re-evaluated against the new ADR and promoted normally if no
     longer conflicting.
   - *No, dismiss* → the deprecated vault note is archived in `_contested/`
     as a record of the disagreement, not deleted. Future agents can see
     that this question was raised and resolved.

**The human is the only party who can initiate an ADR revision.** Agents
must not attempt to draft or propose ADR revisions autonomously, even when
they have high confidence the ADR is outdated. Surface the conflict, wait.

**Step 4 — Never delete, only deprecate or supersede**
When knowledge changes, the old note gets `status: deprecated` and a link
to the replacement. Deprecation without a replacement link is forbidden.
This preserves the audit trail of what agents believed and when.

---

**Correction and amendment protocol**

When an agent discovers an existing note needs to be amended, it writes a
correction note to `_inbox/` with `update-type` and `targets` populated.
The Librarian applies the correction using a method determined by the
*target* note's epistemic type:

| Target epistemic type | Update method | Rationale |
|---|---|---|
| `observation` | In-place edit | Low stakes; `git log` on the file is the audit trail |
| `pattern` | In-place edit | Patterns evolve incrementally; history tracks that |
| `assumption` | Supersession | Preserves the chain of wrong beliefs — informative about agent reasoning |
| `constraint` | Supersession | A corrected constraint should remain visible as a cautionary record |
| `decision` | Supersession | Decisions have formal history; the old decision informs the new one |

**In-place edit:** The Librarian edits the target note directly, updates the
`updated` date, appends a one-line entry to a `## Change log` section at the
bottom of the note recording the date and nature of the change, then deletes
the inbox correction note. Git sees a single-file content diff — no renames.

**Supersession:** The Librarian creates a new note in `notes/` with
`status: verified` (bypassing normal inbox staging — the Librarian is the
authority here), sets the original note to `status: superseded` with a link
to the new note, and deletes the inbox correction note. The superseded note
is never deleted — future agents encountering it follow the link forward.
Git sees: one new file, one edit to the original, one deletion from `_inbox/`.

---

### 5.6 Staleness prevention

Every note has a `review-by` date. The logic:

| Epistemic type | Default TTL | Reasoning |
|---|---|---|
| `observation` | 90 days | Code changes; observations go stale |
| `pattern` | 180 days | Patterns are more stable but still evolve |
| `constraint` | 365 days | Constraints rarely change; if they do, it's a deliberate decision |
| `decision` | 365 days | Links to ADR; ADR is the canonical source |
| `assumption` | 30 days | Assumptions must be challenged frequently |

At session start, the Orchestrator (via the `memory-read` skill) checks for
notes where `review-by < today` in the Core and Index tiers and surfaces
them before starting work.

---

### 5.7 Integration with your agent hierarchy

**Orchestrator** — on every context reset, runs the `memory-read`
`session-init` workflow: loads Core tier (`writing-protocol.md` +
`constraints-summary.md`) and the project index note for the active
project. Surfaces any stale notes flagged by the staleness scan before
starting work. Uses `memory-read` search workflow when it needs to
investigate a specific area during triage or conflict checking.

**Research Lead / Analysts** — primary writers to the vault. After the
R-phase, analysts write their factual findings as `observation` notes to
`_inbox/` via `memory-write`. The Research Lead reconciles outputs using the
existing quorum model before writing — divergence between analysts maps
directly to confidence level (unanimous = high, majority = medium, divergent =
low) and unresolved contradictions go to `_contested/`.

**Engineering Lead / Specialists** — read from vault to inform decisions;
write `pattern` and `observation` notes when they discover non-obvious
implementation facts. Security Reviewer writes `constraint` notes (these
require Librarian review and human confirmation before taking effect).

**Librarian** — dedicated agent responsible for the vault. Owns the write
protocol, manages promotion and deprecation, surfaces stale notes, and
resolves contested claims. See §6.8 for full definition.

**Planner** — at session start, runs `memory-read` `session-init` to load
`constraints-summary.md` and the active project index. Uses `memory-read`
search to pull specific `decision` notes that are relevant to the work
being planned. Planning should never contradict a verified constraint;
`constraints-summary.md` is the authoritative checklist for that.

---

### 5.8 The Librarian agent

The Librarian is a dedicated leaf subagent with a single coherent
responsibility: the vault is its domain. It is separate from the Enabler
by design — the Enabler owns system infrastructure (agent definitions, skills,
configuration); the Librarian owns knowledge governance. The separation keeps
both roles clean and allows any team lead to invoke the Librarian directly
without routing through the Enabler.

**Why no quorum on the Librarian:** The quorum pattern exists for ambiguous
analytical judgments where independent perspectives add value. The Librarian's
decisions are mostly deterministic — does this note already exist? does this
frontmatter have all required fields? is this `review-by` date past today?
The one judgment call (promote or reject a `constraint` note) is gated on
human confirmation anyway, so a second Librarian instance adds cost and latency
for no meaningful gain. The quorum already ran upstream in the Research Lead;
by the time findings reach the Librarian, consensus has been established.

**Placement:** The Librarian is a standalone global agent defined at
`~/.config/opencode/agent/librarian.md`. It is not tied to any team manifest.
Any team lead or the Orchestrator can invoke it directly. This keeps the
Librarian available to all agent setups on the machine without requiring
per-team registration.

**Permissions:**
- `read`: allow (vault path)
- `write`: allow (vault path only)
- `edit`: allow (vault path only)
- `bash`: `grep`, `find`, `git` on vault directory = allow; `*` = deny
- `task`: deny (leaf subagent; does not spawn sub-agents)

**Responsibilities and skills:**

**`memory-read`** — router skill with two workflows; used by all agents
and accessible from any framework that can run skills

*`session-init` workflow* — runs once at session start (see §5.1):
1. Load `_meta/writing-protocol.md` and `_meta/constraints-summary.md`
   unconditionally. Combined target: < 600 tokens.
2. Load `notes/_index-{project}.md` for the active project and
   `notes/_index-{domain}.md` for each domain declared relevant to the
   session.
3. Staleness scan: grep for notes where `review-by < today` across
   `notes/`; surface the list to the calling agent before work begins.
4. Context budget gate: if total loaded content exceeds 2000 tokens,
   load the project index only and defer domain indices to on-demand
   search. Log the deferral so the agent knows to search explicitly.

*`search` workflow* — runs on demand during work (see §5.3):
1. **Phase 1 — Discovery:** expand query terms against
   `_meta/tag-taxonomy.md` aliases. Grep frontmatter fields across
   `notes/`. Filter: `status: verified` only. Return titles + frontmatter.
   Include match count.
2. **Phase 2 — Selective read:** calling agent selects candidates from
   Phase 1 results. Librarian reads those note bodies. Hard cap: 10
   notes per invocation.

Used by: all agents; Orchestrator and Planner invoke `session-init` at
session start; all others invoke `search` during work phases.

**`memory-write`** — writes a note to `_inbox/` following the protocol  
Input: note content + frontmatter fields  
Enforces: search-before-write, epistemic typing, TTL assignment by type  
Used by: Research analysts, Engineering specialists, Security Reviewer

**`memory-curate`** *(Librarian-only)* — reviews `_inbox/` for high-stakes notes  
Scope: only processes `constraint` and `decision` notes; `observation` and
`pattern` notes auto-promote without curation  
Output: recommendation to promote or reject, with reasoning, surfaced to human  
Triggered by: Orchestrator or any lead when a new `constraint` or `decision`
note appears in `_inbox/`

All three skills use direct filesystem operations — `find`, `grep` on
frontmatter, `read`, `write` — which agents already have permission for.
No external server required.

---

## 7. Preventing Knowledge Poisoning — Summary of Controls

| Threat | Control |
|---|---|
| Stale facts | TTL (`review-by`) + staleness scan at session start |
| Assumption treated as fact | Mandatory `epistemic-type: assumption` + 30-day TTL + agents forbidden from acting on assumptions without re-verifying |
| Conflicting writes | Search-before-write + `_contested/` staging for contradictions |
| Hallucination laundering | `_inbox/` visibility buffer + epistemic typing; `constraint`/`decision` types require Librarian review and human confirmation |
| Citation collapse | Mandatory `source-agent`, `source-artifact` fields |
| Agent overconfidence | `confidence: low/medium/high` — Research Lead quorum maps directly to this (unanimous = high, majority = medium, divergent = low) |
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

1. **Write `_meta/writing-protocol.md` first** — this is the constitution of
   the memory system. Get explicit approval on it before any agent touches the
   vault. If the protocol is wrong, everything built on top of it is wrong.

2. **Create the vault** — run `agent-memory init`: prompts for vault path,
   creates `_meta/`, `_inbox/`, `_contested/`, and `notes/` directories,
   seeds `_meta/` from embedded templates, writes `~/.agent-memory.json`.
   Open the vault in Obsidian.

3. **Write the `memory-read` skill** — router skill with `session-init`
   and `search` workflows (see §5.1, §5.2, §5.3). Start with the
   `session-init` workflow first — it unblocks all agents immediately.
   Add the `search` workflow second.

4. **Write the `memory-write` skill** — implements the §6.5 protocol:
   search-before-write, land in `_inbox/`, enforce required frontmatter,
   assign TTL by epistemic type.

5. **Define the Librarian agent** — leaf subagent definition at
   `~/.config/opencode/agent/librarian.md`. Standalone global agent; not
   added to any team manifest.

6. **Write the `memory-curate` skill** — Librarian-only; reviews `_inbox/`
   for `constraint` and `decision` notes; produces promote/reject
   recommendations for human confirmation.

7. **Seed `_meta/constraints-summary.md` and first index notes** — Librarian
   creates these files from any existing ADRs and known constraints. These
   two files make the Core and Index tiers immediately useful before any
   knowledge notes exist.

8. **Update the Orchestrator** — add `memory-read` `session-init` to the
   context reset recovery sequence. Add staleness surfacing before any
   work begins.

9. **Update the Research analysts** — after producing research artifacts, write
   key findings as `observation` notes via `memory-write`.

10. **Backfill** — optionally, run the Research team over existing QRSPI
    artifacts and ADRs to seed the vault with your current knowledge base.

---

## 9. Go CLI Tooling

The memory system ships as a Go module (`github.com/michaelin/agent-memory`)
with three binaries. The CLI is for human use; the linters are for agent use.

### 8.1 Audience separation

| Binary | Audience | Output format | Purpose |
|---|---|---|---|
| `agent-memory` | Human | Pretty, interactive | Vault setup, backup, lint wrapper |
| `lint-note` | Agent | JSON | Single-file validation before write/promote |
| `lint-vault` | Agent | JSON | Cross-file integrity checks |

Agents call `lint-note` and `lint-vault` directly as tools. The CLI's
`agent-memory lint` subcommand is a human-readable wrapper over the same
underlying logic.

### 8.2 Repository structure

```
agent-memory/
├── cmd/
│   ├── agent-memory/    # human CLI: init, backup, lint (pretty wrapper)
│   ├── lint-note/       # agent tool: single-file validation, JSON output
│   └── lint-vault/      # agent tool: cross-file checks, JSON output
├── internal/
│   ├── config/          # XDG resolution, env var override, ~/.agent-memory.json
│   ├── vault/           # note struct, frontmatter parsing (gopkg.in/yaml.v3)
│   ├── lint/            # shared lint logic used by all three binaries
│   ├── init/            # vault scaffolding, _meta/ seeding, gum prompts
│   └── backup/          # lint gate, timestamped tar.gz
├── testdata/
└── go.mod               # module: github.com/michaelin/agent-memory
```

### 8.3 `lint-note` — single-file validation

```bash
lint-note notes/golang-error-handling.md
# pass  → {"valid": true}
# fail  → {"valid": false, "errors": ["missing field: confidence", "unresolved link: [[foo]]"]}
```

Exit 0 on pass, 1 on failure. Agents call this before promoting any note.
The `memory-write` and `memory-curate` skills call it as a hard gate —
a note that fails lint is not written or promoted.

**Checks performed:**
- All required frontmatter fields present and non-empty
- `epistemic-type` is one of the allowed values
- `status` is one of the allowed values
- `confidence` is one of the allowed values
- `created` and `updated` are valid ISO dates
- `review-by` is a valid ISO date and is in the future (warn if < 7 days)
- `## Evidence`, `## Implications`, `## Related` sections present
- No frontmatter fields with placeholder values (e.g. `""` on required fields)

### 8.4 `lint-vault` — cross-file integrity

```bash
lint-vault                    # all checks
lint-vault --links            # unresolved wikilinks only
lint-vault --orphans          # notes not referenced from any index
lint-vault --stale            # notes where review-by < today
```

Output: JSON array of findings, one object per issue.

```json
[
  {"check": "unresolved-link", "file": "notes/foo.md", "link": "[[bar]]"},
  {"check": "orphan", "file": "notes/baz.md"},
  {"check": "stale", "file": "notes/qux.md", "review-by": "2026-01-01"}
]
```

Exit 0 if no findings, 1 if any findings. The `agent-memory backup` command
runs `lint-vault` as a gate — backup aborts if any findings are returned.

**Why Go, not grep:** Wikilink extraction requires markdown-context-aware
parsing. Grep fails on: multiple links per line, `[[slug|alias]]` syntax,
links inside code blocks, and frontmatter boundary detection. Goldmark
(`github.com/yuin/goldmark`) handles all of these correctly.

### 8.5 `agent-memory` CLI commands

| Command | What it does |
|---|---|
| `agent-memory init` | Interactive vault setup: prompt for path, scaffold directories, seed `_meta/` from embedded templates, write `~/.agent-memory.json` |
| `agent-memory backup` | Run `lint-vault`; abort on findings; create timestamped `tar.gz` of vault |
| `agent-memory lint` | Human-readable wrapper: runs `lint-note` on all files + `lint-vault`; pretty-prints findings |

**`init` detail:**
- Prompts for vault path interactively; default suggestion: `~/.local/share/agent-memory`
- Seeds `_meta/` from Go `embed.FS` templates (no network, no external files)
- Idempotent: re-running on an existing vault skips existing files, reports what was skipped

### 8.6 Configuration resolution

Priority order (highest to lowest):

1. `AGENT_MEMORY_VAULT` environment variable
2. `vault` field in `~/.agent-memory.json`
3. Error — no vault configured

`~/.agent-memory.json` format:
```json
{
  "vault": "/Users/michaelin/.local/share/agent-memory"
}
```

---

## 10. Resolved Design Decisions

All questions from the initial design phase are closed.

| Question | Decision |
|---|---|
| Vault location | User-prompted during `agent-memory init`; default `~/.local/share/agent-memory` |
| Cross-project scope | Cross-project and cross-agent from the start; all classification in frontmatter |
| Wikilinks | Live `[[wiki-links]]` written directly by agents; unresolved links acceptable |
| HITL scope | `constraint` and `decision` only; all others auto-promote on TTL expiry |
| Librarian placement | Standalone global agent at `~/.config/opencode/agent/librarian.md`; no team manifest |
| Harness hooks | Agent-instruction baseline only; hooks documented as optional enhancements |
| Tooling | Go CLI (`github.com/michaelin/agent-memory`); three binaries; lint is a hard gate |
| Git in vault | No git; vault is plain files; durability via `agent-memory backup` |
| `git init` during init | Dropped; `init` is a plain file/folder scaffold from embedded templates |
| Module path | `github.com/michaelin/agent-memory` (public repo) |
| Memory consolidation | Future development; deferred until vault has real content |
| Dreaming / pattern promotion | Future development; deferred |

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
