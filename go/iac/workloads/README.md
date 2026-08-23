# go/iac/workloads

Configuration-driven Pulumi blueprint for the **workloads** bounded
context: one or more GCP projects per environment, each serving one or
more concerns (e.g. compute, data, reports), with per-concern exports
for downstream stacks.

## What it provisions

1. A GCP project per `projects` entry (deletion policy `PREVENT`, no
   default VPC) in a governance-managed folder, with the listed APIs
   enabled
2. Per-concern exports for each project: `{concern}ProjectId` and
   `{concern}ProjectNumber` — a collapsed project serving several
   concerns exports under each of them
3. Optionally, `roles/run.developer` for the delivery trigger compute
   SA on every project (when `deliveryProjectNumber` is set)

## Configuration contract

| Key | Required | Meaning |
|---|---|---|
| `environmentFolder` | yes | Key into the governance stack's folder-ID export map (e.g. `shared`, `acceptance`, `production`) |
| `projects` | yes | List of `{name, concerns, apis}` |
| `governanceStack` | no | Fully-qualified stack reference (`org/project/stack`) to read placement from |
| `folderID` | fallback | Parent folder ID; required unless resolved via params or governance stack |
| `billingAccount` | fallback | Billing account ID; required unless resolved via params or governance stack |
| `deliveryProjectNumber` | no | Project number whose default compute SA gets `roles/run.developer` on each project |

## Deployment modes

- **Enterprise** (standalone stack): `func main() { workloads.Workloads() }`
- **Collapsed** (alongside other BCs): `workloads.Apply(ctx, &workloads.Params{FolderID: …, BillingAccount: …})`

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Pulumi composition roots**: stacks provisioning multi-project workload
  environments.

### Load-Bearing Promises
1. **`Apply` Is The Entry Point**: the blueprint is consumed through `Apply`;
   the types it reads are the published surface and anything else is internal.
2. **Configuration Is Read, Not Inferred**: the blueprint provisions what the
   Pulumi configuration states, so a stack's shape is readable from its config
   rather than from this package's defaults.

> This module currently carries no tests of its own. Until it does, the
> promises above are structural rather than pinned — treat changes here as
> needing review against consuming stacks rather than against a suite.
