---
name: release-review
description: Performs a comprehensive pre-release review of a repository using 6 specialized parallel subagents. Activate when the user wants to verify a repo is ready for public release, or asks for a thorough review before shipping.
---

# Release Review Skill

Perform a formal pre-release review of the current repository by dispatching
6 specialized subagents in parallel. Each agent reviews the repo from a
different angle and produces a structured verdict.

## When to Activate

- User asks to "review for release" or "make sure this is ready"
- User mentions "pre-release review" or "release readiness"
- User wants to verify a repo before making it public
- User asks for a "full review" or "comprehensive review"

## Execution

1. Read the repository structure to understand what exists.
2. Launch all 6 review agents in parallel using `invoke_subagent`.
3. Wait for all agents to report back.
4. Consolidate results into a Go/No-Go artifact.
5. If any agent reports FAIL, present the failures and offer to fix them.

## Agent Definitions

Launch each agent as a `research` subagent (read-only). Each agent must read
all relevant files and return a structured report.

### Agent 1: Link & Reference Integrity

**Prompt template:**

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **Link & Reference Integrity**.
>
> Read every `.md` file in the repository. For each markdown link (syntax: text in brackets, URL in parens):
> 1. If `target` is a relative path, verify the file or directory exists.
> 2. If `target` contains an anchor `#heading`, verify the heading exists in the target file.
> 3. If `target` is an external URL (http/https), note it but do not validate.
>
> Also check:
> - No orphaned files that nothing links to (excluding LICENSE, .gitignore, .editorconfig, config files)
> - All directory README/index files link to their children
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

### Agent 2: License & Legal Compliance

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **License & Legal Compliance**.
>
> Read every LICENSE file in the repository. Verify:
> 1. Each LICENSE file contains valid, unmodified license text (compare structure against known licenses).
> 2. Copyright year and holder are correct and consistent across all files.
> 3. README and CONTRIBUTING license sections accurately describe what each license covers.
> 4. Every directory containing content has license coverage (either its own LICENSE or inherits from parent).
> 5. CODE_OF_CONDUCT.md attribution is correct.
> 6. No conflicting license declarations.
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

### Agent 3: OKF Conformance

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **OKF Conformance**.
>
> Read every `.md` file under the `docs/` directory. For each file:
> 1. Verify YAML frontmatter exists and is valid YAML.
> 2. Verify the `type` field is present (OKF required).
> 3. If the repo has an OKF profile document, verify all profile-required fields are present.
> 4. Verify field values match allowed values defined in the profile.
> 5. Verify `domain` values match the folder the file is in.
> 6. Verify reserved filenames (`index.md`, `log.md`) are used correctly.
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

### Agent 4: CI & DevOps Validation

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **CI & DevOps Validation**.
>
> Read all files under `.github/`, `.githooks/`, and config files (`.editorconfig`, `.gitignore`, `dependabot.yml`).
> Verify:
> 1. All YAML files are syntactically valid.
> 2. GitHub Actions workflow logic is sound (conditionals, matrix strategy, permissions).
> 3. Git hook scripts are syntactically correct and the regex patterns are valid.
> 4. Hook header comments match actual patterns scanned.
> 5. Dependabot config references valid directories.
> 6. Issue template forms have all required fields.
> 7. `.editorconfig` syntax is valid.
> 8. `.gitignore` covers expected artifacts.
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

### Agent 5: Content Quality & Consistency

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **Content Quality & Consistency**.
>
> Read every `.md` file in the repository. Check:
> 1. No spelling errors in headings and key terms.
> 2. Consistent terminology throughout (e.g., always "Go module" not sometimes "Go package").
> 3. No TODO/FIXME/HACK/TBD/XXX markers (except intentional seed directory markers).
> 4. No placeholder text (lorem ipsum, example.com in real contexts).
> 5. No broken promises (text claiming something exists when it doesn't).
> 6. Contact information is consistent everywhere it appears.
> 7. Version numbers are consistent (Go version, etc.).
> 8. Professional tone suitable for a public open source project.
> 9. No references to internal/private resources.
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

### Agent 6: Security & Sensitive Data Scan

> You are reviewing the repository at {WORKSPACE_ROOT} for release readiness.
> Your role: **Security & Sensitive Data Scan**.
>
> Read every file in the repository (including non-markdown). Search for:
> 1. API keys, tokens, passwords, or credentials (patterns: ghp_, gho_, AIza, AKIA, -----BEGIN.*KEY).
> 2. Internal/private URLs, IP addresses, or hostnames.
> 3. Hardcoded cloud project IDs (GCP project IDs, AWS account numbers).
> 4. Personal data beyond intentional maintainer attribution.
> 5. Accidentally committed binary files or build artifacts.
> 6. `.env` files or similar secret configuration.
> 7. Any content that would be inappropriate in a public repository.
>
> Also verify git history is clean (check `git log --oneline` for sensitive commits).
>
> Return a structured report with verdict (PASS/WARN/FAIL) and findings.

## Report Format

Consolidate all 6 agent reports into a single artifact:

```markdown
# Release Review Report

## Overall Verdict: GO / NO-GO

| Agent | Verdict | Findings |
|---|---|---|
| Link Integrity | PASS/WARN/FAIL | N pass, N warn, N fail |
| License Compliance | PASS/WARN/FAIL | N pass, N warn, N fail |
| OKF Conformance | PASS/WARN/FAIL | N pass, N warn, N fail |
| CI & DevOps | PASS/WARN/FAIL | N pass, N warn, N fail |
| Content Quality | PASS/WARN/FAIL | N pass, N warn, N fail |
| Security Scan | PASS/WARN/FAIL | N pass, N warn, N fail |

## Detailed Findings

### Agent 1: Link Integrity
- [PASS] description
- [WARN] description
- [FAIL] description

(repeat for each agent)

## Action Items

(list any FAIL items that must be fixed before release)
```

## Post-Review

If the verdict is **GO**:
- Offer to stage and commit all changes.
- Remind the user of post-push actions (GitHub settings).

If the verdict is **NO-GO**:
- Present the FAIL items.
- Offer to fix them automatically.
- Re-run the failed checks after fixing.
