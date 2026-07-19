# go/iac/pulumi/gcpinfra

Pulumi building blocks for Google Cloud infrastructure. Each package is a
thin, opinionated adapter: it validates its input, then creates the GCP
resources.

## Packages

| Package | Provisions | Input |
|---|---|---|
| `folders` | Organizational folder hierarchies | [`governance/hierarchy`](../../../governance/hierarchy/) |
| `tagkeys` | Classification dimensions as org-level tag keys | [`governance/classification`](../../../governance/classification/) |
| `projects` | GCP projects with API enablement | `projects.Config` |
| `secrets` | Secret Manager secrets seeded with caller-supplied values | `secrets.Secret` |
| `serviceaccounts` | Service accounts in a project | `serviceaccounts.Account` |
| `iambindings` | Project-level and service-account-level IAM member bindings | `iambindings.Binding` / `DynamicBinding` / `SAIamBinding` |

More building blocks (datasets, networking, …) will be added as they are
generalized.

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

## Protection semantics

- `folders` and `tagkeys` create resources with `DeletionProtection`
  (GCP-level) **and** `pulumi.Protect` (state-level). Downgrade both
  deliberately before destroying anything.
- `projects` sets `DeletionPolicy: PREVENT` (GCP-level) and disables the
  auto-created default VPC.
- `secrets`, `serviceaccounts`, and `iambindings` create unprotected
  resources — they are cheap to recreate and their sources of truth live
  outside the stack (secret values with the caller, IAM in config).
