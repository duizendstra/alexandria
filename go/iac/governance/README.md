# go/iac/governance

The governance blueprint: a complete, configuration-driven Pulumi program
that provisions organizational governance on Google Cloud.

It reads Pulumi stack configuration, builds a validated
[`governance/plan`](../../governance/plan/), deploys it through the
[`gcpinfra`](../pulumi/gcpinfra/) building blocks, and exports the results
under the [`governance/exports`](../../governance/exports/) contract for
downstream stacks.

## Configuration

| Key | Required | Description |
|---|---|---|
| `parent` | ✅ | GCP parent: `organizations/<id>` or `folders/<id>` |
| `rootFolder` | ✅ | Display name of the root folder |
| `tier` | — | `starter`, `standard` (default), or `enterprise` |
| `environments` | tier ≥ standard | Child folder names, e.g. `["dev", "prod"]` |
| `tagKeys` | — | Enterprise only: classification dimensions |
| `billingAccount` | — | Enterprise only: billing account ID to export |

## Usage

As a standalone program:

```go
package main

import "github.com/duizendstra/alexandria/go/iac/governance"

func main() { governance.Governance() }
```

Composed with other stacks in one program:

```go
pulumi.Run(func(ctx *pulumi.Context) error {
    return governance.Apply(ctx)
})
```

## Exports

| Name | When |
|---|---|
| `rootFolderId` | always |
| `folderIds` | always |
| `orgId` | Organization scope |
| `billingAccount` | when configured |
| `<dimension>TagKeyId` | per configured tag key |
