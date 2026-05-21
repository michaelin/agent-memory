<progress feature="note-writing" started="2026-05-21">

<phases>
<phase name="questions" skill="q-questions" status="complete" artifact="questions.md"/>
<phase name="research" skill="r-research" status="complete" artifact="research.md"/>
<phase name="design" skill="d-design" status="complete" artifact="design.md"/>
<phase name="structure" skill="s-structure" status="complete" artifact="structure.md"/>
<phase name="plan" skill="p-plan" status="complete" artifact="plan.md"/>
<phase name="worktree" skill="w-worktree" status="complete" artifact="worktree.md"/>
<phase name="implement" skill="i-implement" status="pending" artifact="implement-log.md"/>
<phase name="review" skill="pr-review" status="pending" artifact="review.md"/>
</phases>

<current_phase>implement</current_phase>

<notes>
- Increments 1 (vault scaffolding) and 2 (note format & lint) are complete
- ADR-0001: single binary with subcommands — write-note is a subcommand
- internal/note/ package has Parse(), Lint(), Rules() — reuse for validation
- Vault init, config resolution, and structure exist in internal/vault/
- This increment focuses on the deterministic write tool, not the LLM skill
</notes>

</progress>
