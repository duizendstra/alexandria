# go/iac/pulumi/gcpinfra

Pulumi building blocks for Google Cloud infrastructure. Each package is a
thin, opinionated adapter: it validates input via a domain package, then
creates the GCP resources with deletion protection and Pulumi `Protect`
enabled.

## Packages

| Package | Provisions | Domain input |
|---|---|---|
| `folders` | Organizational folder hierarchies | [`governance/hierarchy`](../../../governance/hierarchy/) |
| `tagkeys` | Classification dimensions as org-level tag keys | [`governance/classification`](../../../governance/classification/) |

More building blocks (projects, datasets, service accounts, …) will be
added as they are generalized.

## Usage

```go
import (
    "github.com/duizendstra/alexandria/go/governance/hierarchy"
    "github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/folders"
)

outputs, err := folders.Apply(ctx, hierarchy.Config{
    Parent:   "organizations/123456789",
    RootName: "example",
    Children: []string{"dev", "prod"},
})
```

All resources are created with `DeletionProtection` (GCP-level) and
`pulumi.Protect` (state-level). Downgrade both deliberately before
destroying anything.
