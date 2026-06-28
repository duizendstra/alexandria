---
name: dialectical-review
description: >
  Adversarial expert review using thesis/antithesis/mediator pattern.
  Launch N expert roles, each with a thesis and antithesis agent, then
  synthesize a mediator verdict. Fix blockers and re-run until clear.
  Trigger: "dialectical review", "thesis antithesis", "fire up experts"
---

# Dialectical Review

An adversarial review pattern that surfaces hidden issues through
structured debate. Each expert domain gets a thesis agent (argues FOR)
and an antithesis agent (argues AGAINST), followed by a mediator
synthesis.

## When to Use

- Before tagging a release
- Before merging significant PRs
- When the team needs high-confidence quality assessment
- When a single reviewer might miss issues due to confirmation bias

## Pattern

### 1. Define Expert Roles

Choose 2-4 expert domains relevant to the review. Examples:

| Domain | Thesis position | Antithesis position |
|---|---|---|
| Go Expert | "This is production-quality Go" | "This code has quality issues" |
| Release Expert | "Ready for release tag" | "Not ready to ship" |
| Open Source Expert | "Meets OS standards" | "Has OS gaps" |
| Security Expert | "No vulnerabilities" | "Has security concerns" |

### 2. Launch Agents

Launch 2N `research` subagents in parallel (thesis + antithesis per
role). Each agent gets:

- **Explicit position framing**: "Your position: **This code is
  production-quality Go**" or "Your position: **Find every problem**"
- **Full file list**: Tell each agent to read ALL relevant files
- **Structured output requirement**: Ask for PASS/WARN/FAIL verdict
  with line-number citations
- **Previous findings** (for round 2+): Feed back previous blockers
  for verification

### 3. Handle Failures

Rate limits may cause agent failures. Retry failed agents while
processing completed ones. Do not block on a single failure.

### 4. Synthesize Mediator Verdict

After all agents report, synthesize a mediator verdict per domain:

| Category | Criteria |
|---|---|
| BLOCKER | Both sides agree, or antithesis proves with evidence |
| WARN | Antithesis raises valid concern, thesis acknowledges |
| INFO | Antithesis raises concern, thesis refutes convincingly |

### 5. Fix and Re-Run

If blockers exist:
1. Fix all blockers
2. Re-run the full review (round 2)
3. Round 2 agents verify previous fixes AND search for new issues
4. Repeat until no blockers remain

## Example Agent Prompt (Thesis)

> You are a **Go Expert — THESIS** reviewing the module at {PATH}.
>
> Your position: **This code is production-quality Go, ready to ship.**
>
> Read ALL files. Cover: API design, error handling, concurrency,
> tests, docs, deps, performance.
>
> Be specific — cite line numbers. Return structured report with
> PASS/WARN/FAIL verdict.

## Example Agent Prompt (Antithesis)

> You are a **Go Expert — ANTITHESIS** reviewing the module at {PATH}.
>
> Your position: **Find every remaining problem.**
>
> Read ALL files. Find: API footguns, untested paths, missing features,
> code smells, documentation gaps. Be ruthless.
>
> Return structured report with PASS/WARN/FAIL verdict.

## Output Format

The mediator report should include:

1. **Agent results table** — verdict per agent
2. **Blocker list** — issues both sides agree on
3. **Debatable items** — thesis vs antithesis vs mediator verdict
4. **Action plan** — must-fix before release + post-release follow-ups
5. **Pre-tag checklist** — concrete steps remaining
