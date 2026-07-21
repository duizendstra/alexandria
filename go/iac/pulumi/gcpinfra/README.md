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

- `folders` and `tagkeys` create resources with `DeletionProtection`
  (GCP-level) **and** `pulumi.Protect` (state-level). Downgrade both
  deliberately before destroying anything.
- `projects` sets `DeletionPolicy: PREVENT` (GCP-level) and disables the
  auto-created default VPC.
- `secrets`, `serviceaccounts`, and `iambindings` create unprotected
  resources — they are cheap to recreate and their sources of truth live
  outside the stack (secret values with the caller, IAM in config).
- `cloudrun` ignores container image changes and `firestore` ignores
  document field changes after creation — both are managed outside the
  stack at runtime (CI/CD deploys, application config writes).
- `tables` and `firestore` databases expose explicit deletion-protection
  knobs (`DeletionProtection` / `DeleteProtection`) — set them per
  environment.
