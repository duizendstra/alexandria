# go/iac/pulumi/gcpinfra

Pulumi building blocks for Google Cloud infrastructure. Each package is a
thin, opinionated adapter: it validates its input, then creates the GCP
resources.

## Packages

| Package | Provisions | Input |
|---|---|---|
| `lifecycle` | Nothing — carries the permanent-or-disposable decision every other block reads | `lifecycle.Option` |
| `folders` | Organizational folder hierarchies | [`governance/hierarchy`](../../../governance/hierarchy/) |
| `tagkeys` | Classification dimensions as org-level tag keys | [`governance/classification`](../../../governance/classification/) |
| `projects` | GCP projects with API enablement | `projects.Config` |
| `secrets` | Secret Manager secrets seeded with caller-supplied values | `secrets.Secret` |
| `serviceaccounts` | Service accounts in a project | `serviceaccounts.Account` |
| `iambindings` | Project-level and service-account-level IAM member bindings | `iambindings.Binding` / `DynamicBinding` / `SAIamBinding` |
| `budgets` | Billing budgets with threshold alerts and email notification channels | `budgets.Config` |
| `datasets` | BigQuery datasets | `datasets.Config` |
| `logsinks` | Org-level log sinks to BigQuery | `logsinks.Config` |
| `connections` | Cloud Build v2 connections to Git hosting providers, with repo links | `connections.Config` / `RepoLink` |
| `registries` | Artifact Registry repositories with reader/writer IAM grants | `registries.Config` |
| `triggers` | Cloud Build triggers firing on tag pushes | `triggers.Config` |
| `cloudrun` | Cloud Run v2 services and jobs (image changes ignored — deploys via CI/CD), invoker grants | `cloudrun.ServiceConfig` / `JobConfig` |
| `scheduler` | Cloud Scheduler jobs with HTTP targets and OAuth authentication | `scheduler.Config` |
| `firestore` | Firestore databases and seeded documents (field changes ignored after creation) | `firestore.DatabaseConfig` / `DocumentConfig` |
| `tables` | BigQuery tables, native (schema, optional DAY partitioning) and external (e.g. Google Sheets) | `tables.Config` / `ExternalConfig` |
| `dataform` | Dataform repositories with Git remotes, release and workflow configs, P4SA enablement | `dataform.RepositoryConfig` / `ReleaseConfig` / `WorkflowConfig` |
| `uptimechecks` | HTTPS uptime checks with a failure alert policy (IAP-aware; caller-supplied notification channels) | `uptimechecks.Config` |

More building blocks (networking, …) will be added as they are
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

Protection sits in two independent layers, and both matter.

**Provider level** — `deletionProtection` / `deletionPolicy` on the GCP
resource itself. It is enforced by the API, so it survives a lost stack
state, but it fails the operation mid-apply, after earlier resources in the
same update have already changed.

**State level** — `pulumi.Protect`. The engine refuses the delete at
*preview*, before anything is touched, which is where you want to find out.
It only knows what the stack state says, so it is the layer to clear when a
replacement is genuinely wanted.

Every block that creates a data-bearing resource sets both by default:

| Block | Resource | Provider | `pulumi.Protect` |
|---|---|---|---|
| `folders` | folder | `deletionProtection: true` | yes |
| `tagkeys` | tag key | — | yes |
| `projects` | project | `deletionPolicy: PREVENT` | yes |
| `datasets` | dataset | `deleteContentsOnDestroy: false` | yes |
| `tables` | native table | `deletionProtection: true` | yes |
| `firestore` | database | `deleteProtectionState: ENABLED`, `deletionPolicy: PREVENT` | yes |
| `firestore` | seeded document | — | yes |
| `registries` | repository | `deletionPolicy: PREVENT` | yes |
| `secrets` | secret | `deletionPolicy: PREVENT`, `deletionProtection: true` | yes |

Deliberately unprotected: `tables` external tables (the rows live in the
source, not the table), `secrets` versions (rotation *is* replacing one),
`serviceaccounts` and `iambindings` (config is the source of truth), and
`projects` API enablements.

`cloudrun` ignores container image changes and `firestore` ignores document
field changes after creation — both are managed outside the stack at runtime
(CI/CD deploys, application config writes). That is also why a seeded document
is protected even though it holds no rows of its own: its live contents come
from the application, so `Fields` cannot put them back.

### Opting out

Test and preview stacks pass `lifecycle.Ephemeral()` to any block, which
clears both layers so `pulumi destroy` works without a manual
`pulumi state unprotect` first:

```go
_, err := datasets.Apply(ctx, projectID, cfg, deps, lifecycle.Ephemeral())
```

`tables.Config.DeletionProtection` and `firestore.DatabaseConfig.DeleteProtection`
force provider-level protection back on for a single resource inside an
otherwise ephemeral stack.

### Renaming a protected resource

A Pulumi logical-name change is a delete plus a create, not an update — on a
data-bearing resource that is data loss. Two ways through:

- To keep the resource, add `pulumi.Aliases` for the old name. The engine
  matches the existing resource and updates it in place.
- To genuinely replace it, `pulumi state unprotect <urn>` first, then rename.

**`Protect` cannot guard the update that introduces it.** It lives in stack
state and is written by an `up`, so an `up` that both adds protection and
renames still deletes the old resource — the state the engine consults was
written before the protection existed. Deploy the protection on its own,
then rename.
