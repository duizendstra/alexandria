# Skills

Antigravity AI skills shareable across workspaces.

## Usage

Consumer repos inherit skills via `skills.json`:

```json
{
  "inherits": [
    { "path": "../alexandria/skills" }
  ]
}
```

> **Note**: Use a path relative to the consuming repository, or an absolute path
> if workspaces are not co-located.

## Skill Index

| Skill | Description |
|---|---|
| [dialectical-review](dialectical-review/) | Adversarial expert review using thesis/antithesis/mediator pattern |
| [ko-build](ko-build/) | Ko container build setup for Go services targeting GCP Cloud Run |
| [release-review](release-review/) | Pre-release repository review with 6 specialized parallel agents |

## Creating a Skill

Each skill is a directory with a `SKILL.md` file (YAML frontmatter + markdown
instructions). See the Antigravity documentation for the full skill authoring
guide.
