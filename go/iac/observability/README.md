# go/iac/observability

Configuration-driven Pulumi blueprint for the **Observability** bounded
context: a dedicated project with a BigQuery log-analytics dataset and
an org-level audit-log sink routed into it.

## What it provisions

1. A GCP project (BigQuery, Logging, Monitoring APIs enabled, deletion
   policy `PREVENT`, no default VPC) in a governance-managed folder
2. An `org_logs` BigQuery dataset for aggregated audit/activity logs
3. An org-level log sink (`org-audit-to-bigquery`) filtering
   `cloudaudit.googleapis.com` logs from the whole organization into
   the dataset (`includeChildren: true`)

The sink's writer identity is exported (`sinkWriterIdentity`) so a
downstream stack can grant it BigQuery access.

## Configuration contract

| Key | Required | Meaning |
|---|---|---|
| `projectName` | yes | Project ID and display name |
| `region` | no | Dataset region (default `europe-west4`) |
| `governanceStack` | no | Fully-qualified stack reference (`org/project/stack`) to read placement from |
| `governanceFolder` | no | Key into the governance stack's folder-ID export map (default `shared`) |
| `folderID` | fallback | Parent folder ID; required unless resolved via params or governance stack |
| `billingAccount` | fallback | Billing account ID; required unless resolved via params or governance stack |
| `orgID` | fallback | Organization ID (sink source scope); required unless resolved via params or governance stack |

## Exports

`projectId`, `logDatasetId`, `sinkWriterIdentity`

## Deployment modes

- **Enterprise** (standalone stack): `func main() { observability.Observability() }`
- **Collapsed** (alongside other BCs): `observability.Apply(ctx, &observability.Params{...})`
