---
name: diffract-review
description: >
  Structured code review using the Diffract protocol with 9 parallel lens
  agents and a CHECK mediator. Each lens uses deterministic tools (grep,
  find) before reasoning. Based on contextvibes/diffract v0.1.0. Trigger:
  "diffract review", "structured review", "9 lens review", "run diffract".
---

# Diffract Review

A structured review protocol that launches 9 parallel lens agents, each
with tool access, feeding findings into a CHECK mediator that vets them
through governors. Based on [contextvibes/diffract](https://github.com/contextvibes/diffract).

> **Tools first, reasoning second.** Every lens agent runs deterministic
> commands (grep, find, wc) before making judgment calls. A grep for
> duplicated strings is more reliable than intuition.

## When to Use

- Before merging significant PRs
- Before tagging a release
- When reviewing documentation, blueprints, or config templates
- When a single reviewer might miss issues due to confirmation bias

## Governor Presets

Pick a preset or define custom governors.

| Preset | Compass | Cobra |
|---|---|---|
| **production** | "Is this ready for production?" | Cautious — fix more, skip less |
| **prototype** | "Does this work for a first iteration?" | Aggressive — skip more, ship faster |
| **documentation** | "Is this ready for public consumption?" | Cautious — clarity and accuracy matter |
| **pr-review** | "Does this PR meet the project's merge standards?" | Balanced — fix blockers, note improvements |

## Execution

### Phase 1: PLAN — Set Governors

Determine the review scope and governors:

```
Diffract: v0.1.0
🧭 Compass: [from preset or user-specified]
🐍 Cobra:   [from preset or user-specified]
⚖️ Integrity: file:line evidence per lens. Cognitive anchoring required.
```

For PR reviews, identify changed files:
```bash
git diff --name-only main...HEAD
```

For full-repo reviews, identify all relevant files:
```bash
find . -name '*.go' -o -name '*.md' -o -name '*.yaml' | grep -v vendor | grep -v .git
```

### Phase 2: DO — Launch 9 Parallel Lens Agents

Launch 9 `research` subagents in parallel using `invoke_subagent`. Each
agent gets a lens-specific prompt from the `prompts.md` file in this
skill directory.

Read `prompts.md` to get the prompt template for each lens. Substitute:
- `{WORKSPACE_ROOT}` — the workspace root path
- `{CHANGED_FILES}` — newline-separated list of changed files
- `{COMPASS}` — the compass governor from PLAN
- `{COBRA}` — the cobra governor from PLAN

Each lens agent:
1. Runs the prescribed tool commands first
2. Reads all changed files
3. Applies the lens-specific analysis
4. Returns findings in the required format (table or cognitive anchoring)

### Phase 3: CHECK — Mediator Vets Findings

After all 9 lens agents report, launch one final `research` subagent as
the CHECK mediator. The mediator receives all findings and:

1. Deduplicates findings that appear across multiple lenses
2. Vets each finding through all three governors:
   - ⚖️ **Integrity**: Is the evidence real? Is it testable?
   - 🧭 **Compass**: Is it relevant to the review goal?
   - 🐍 **Cobra**: Does fixing it cause more harm than the finding?
3. Assigns verdicts: 🔴 Fix / 🟡 Suggest / 🟢 Accept
4. Produces the unified CHECK table

Use the CHECK mediator prompt from `prompts.md`.

### Phase 4: LEARN — Scorecard and Report

After the mediator reports, compile the final review artifact:

1. **Governor summary** — compass, cobra, integrity settings
2. **CHECK table** — all findings with verdicts
3. **Scorecard** — grade per lens, overall grade
4. **Gap analysis** — what wasn't covered and why
5. **Action items** — prioritized fix list (🔴 must-fix, 🟡 should-fix, 🟢 nice-to-have)

If 🔴 Fix items exist, offer to apply them and re-run.

## Handling Failures

Rate limits may cause agent failures. When an agent fails:
1. Continue processing completed agents
2. Retry failed agents after a brief pause
3. If a lens agent cannot complete after retry, note it in the gap analysis

## Output Format

The final report should follow this structure:

```markdown
# Diffract Review — [artifact description]

## Governors
🧭 Compass: ...
🐍 Cobra: ...
⚖️ Integrity: ...

## CHECK Table
| ID | Lens | Finding | ⚖️ Integrity | 🧭 Compass | 🐍 Cobra | Verdict |

## Scorecard
| Lens | Grade | Rationale |

## Gap Analysis
| Gap | Reason | Recommendation |

## Action Items
| Priority | ID | Action Required |
```

## Reference

- [Diffract protocol](https://github.com/contextvibes/diffract)
- [Diffract PROMPT.md](https://github.com/contextvibes/diffract/blob/main/PROMPT.md)
- [9 Lenses documentation](https://github.com/contextvibes/diffract/blob/main/docs/lenses.md)
