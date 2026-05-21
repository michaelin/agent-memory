# Architectural Decision: Skill vs Subagent for Note Writing

## Your Question

> Q1: What is the benefit of a subagent? Could a skill be sufficient to let the handle what the subagent would do, and also teach it how to use the utility?

## Answer: Yes, a Skill Is Sufficient and Better

### What a Subagent Would Do

A subagent for note writing would:
1. Receive a claim and evidence from the calling agent
2. Compose the note body (with LLM judgment)
3. Call a tool to check for similarity
4. Reason about false positives
5. Call a tool to write the note
6. Return success/failure to the calling agent

### What a Skill Does (Same Thing, Better)

A skill for note writing does:
1. Receive a claim and evidence from the calling agent
2. Compose the note body (with LLM judgment)
3. Call a tool to check for similarity
4. Reason about false positives
5. Call a tool to write the note
6. Return success/failure to the calling agent

**The key difference:** The skill is invoked directly by the calling agent, not as a separate process.

### Why a Skill Is Better

| Aspect | Subagent | Skill |
|--------|----------|-------|
| **Latency** | Spawns separate process; adds round-trip | Runs in-context; no extra latency |
| **Complexity** | Requires agent framework to spawn/manage | Encapsulated in skill definition |
| **Reusability** | Must be explicitly delegated | Available to all agents automatically |
| **Token cost** | Extra tokens for agent invocation overhead | Tokens only for actual work |
| **Testability** | Requires agent framework to test | Can be tested like any skill |
| **Evolution** | Update agent prompt, rebuild binary | Update skill definition, done |
| **Framework coupling** | Tied to specific agent framework | Portable across frameworks |
| **Teaching** | Subagent teaches itself via prompt | Skill teaches calling agent via instructions |

### How the Skill Teaches the Calling Agent

The skill's instructions (embedded in the skill definition) teach the calling agent:

```markdown
# write-memory Skill

Use this skill to write a note to the agent memory vault.

## How to use it

1. **Prepare your claim:** You have a factual claim, pattern, or decision you want to record
2. **Gather evidence:** Identify where this claim came from (file path, URL, etc.)
3. **Invoke the skill:** Call `write-memory` with:
   - `claim`: Your factual claim (one sentence)
   - `evidence`: Where you observed this
   - `type`: One of: observation, pattern, constraint, decision, assumption, synthesis
   - `confidence`: One of: low, medium, high
   - `scope`: One of: project, cross-project
   - `project`: Project slug (if scope is project)
   - `domain`: List of technology domains (e.g., [golang, auth])

## What the skill does

1. Composes the note body with proper structure
2. Checks for similar existing notes
3. If a similar note exists, asks you to clarify if it's a duplicate
4. Writes the note to `_inbox/` with proper frontmatter
5. Returns success/failure and any warnings

## Example

You've discovered that "Go error handling must wrap errors with context at domain boundaries."

```
Call write-memory with:
- claim: "Go error handling must wrap errors with context at domain boundaries"
- evidence: "Observed in pkg/errors/wrap.go lines 10-25"
- type: "observation"
- confidence: "high"
- scope: "project"
- project: "myproject"
- domain: ["golang", "error-handling"]
```

The skill will:
1. Compose the note body
2. Check if a similar note exists
3. Write to `_inbox/2026-04-24-golang-error-wrapping.md`
4. Return success with the note slug
```

### The Hybrid Approach (Skill + Tool)

The skill delegates mechanical work to a static tool:

```
Calling Agent
    ↓
write-memory Skill (LLM reasoning)
    ├→ Composes note body
    ├→ Calls memory-write-mechanics tool
    │   ├→ Similarity check
    │   ├→ Frontmatter assembly
    │   ├→ Inbox placement
    │   └→ Log entry writing
    └→ Returns result to calling agent
```

This gives you:
- **Flexibility** of an LLM (skill can reason about quality, ask clarifying questions)
- **Determinism** of a tool (mechanical parts are reproducible, testable)
- **Reusability** (skill can be used by any agent)
- **Maintainability** (tool and skill can evolve independently)

### Why Not Just a Tool?

A pure static tool (without a skill) would require the calling agent to:
1. Compose the note body itself
2. Understand the frontmatter requirements
3. Call the tool with all the right parameters
4. Handle the tool's response (similarity hits, errors, etc.)

This puts too much burden on the calling agent. The skill abstracts away the complexity.

### Why Not Just a Subagent?

A pure subagent would:
1. Add latency (separate process invocation)
2. Add complexity (requires agent framework)
3. Add token cost (agent invocation overhead)
4. Reduce reusability (must be explicitly delegated)
5. Reduce testability (requires agent framework)

The skill does everything the subagent would do, but better.

## Conclusion

**Use a skill that wraps a tool.**

The skill:
- Teaches the calling agent how to write notes
- Composes the note body with LLM judgment
- Delegates mechanical work to a static tool
- Is reusable across all agents
- Has no latency overhead
- Is easy to test and evolve

This is the sweet spot between flexibility and determinism.

## Same Decision for Note Search

The same reasoning applies to note search:

**Use a skill that wraps a tool.**

The `search-memory` skill:
- Takes a simple query (no `--phase=2` weirdness)
- Internally handles two-phase retrieval
- Returns structured results to the calling agent
- Hides implementation details
- Can be enhanced later (semantic search, ranking, etc.)

The calling agent just says "find notes about auth" and gets back relevant notes. The skill handles the complexity.
