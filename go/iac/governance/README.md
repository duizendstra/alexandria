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

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Pulumi composition roots**: stacks realising governance over an
  organisation or a folder.
- **Platform teams adopting a tier**: callers choosing how much governance to
  switch on without rewriting their stack.

### Load-Bearing Promises
1. **Standard Is The Default Tier**: a stack that names no tier gets Standard.
   The default is explicit and pinned, not whichever tier happens to be first.
2. **Tiers Apply At Both Scopes**: the Starter tier applies at folder scope and
   at organisation scope, so scope is a choice rather than a constraint.
3. **An Invalid Parent Fails**: a parent that does not resolve fails the apply
   instead of provisioning against the wrong place in the hierarchy.
4. **Enterprise Degrades Explicitly**: the Enterprise tier applies without tag
   keys, and fails when tag keys are supplied but malformed.
