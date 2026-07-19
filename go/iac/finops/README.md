# go/iac/finops

Configuration-driven Pulumi blueprint for the **FinOps** bounded
context: a dedicated project with a BigQuery billing-export dataset and
an org-level budget with threshold alerts.

## What it provisions

1. A GCP project (BigQuery, Monitoring, Billing Budgets APIs enabled,
   deletion policy `PREVENT`, no default VPC) in a governance-managed
   folder
2. A `billing_export` BigQuery dataset (billing export itself is enabled
   once in the Cloud Console — the dataset is its destination)
3. An org-scoped billing budget with email notification channels and
   alert thresholds (default 50/75/90/100%)

## Configuration contract

| Key | Required | Meaning |
|---|---|---|
| `projectName` | yes | Project ID and display name |
| `monthlyBudget` | yes | Monthly budget amount (integer, in `currency` units) |
| `alertEmails` | yes | Notification recipients for budget alerts |
| `currency` | no | ISO 4217 code (default `EUR`) |
| `thresholds` | no | Alert fractions (default `[0.50, 0.75, 0.90, 1.00]`) |
| `region` | no | Dataset region (default `europe-west4`) |
| `governanceStack` | no | Fully-qualified stack reference (`org/project/stack`) to read placement from |
| `folderID` | fallback | Parent folder ID; required unless resolved via params or governance stack |
| `billingAccount` | fallback | Billing account ID; required unless resolved via params or governance stack |
| `orgID` | fallback | Organization ID (budget scope); required unless resolved via params or governance stack |

## Exports

`projectId`, `billingDatasetId`

## Deployment modes

- **Enterprise** (standalone stack): `func main() { finops.FinOps() }`
- **Collapsed** (alongside other BCs): `finops.Apply(ctx, &finops.Params{...})`
