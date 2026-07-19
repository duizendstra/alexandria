# go/iac/identity

Configuration-driven Pulumi blueprint for the **identity** bounded
context: a dedicated project holding Secret Manager secrets and service
accounts, with IAM for consumers and impersonators.

## What it provisions

1. A GCP project (Secret Manager + IAM APIs enabled, deletion policy
   `PREVENT`, no default VPC) in a governance-managed folder
2. Secret Manager secrets, seeded at deploy time via a `SecretResolver`
   (default: the local `pass` store)
3. Service accounts, each exported as `<id>-email`
4. IAM: `secretAccessor` for consumer SAs; `serviceAccountUser`,
   `serviceAccountTokenCreator`, and `serviceAccountOpenIdTokenCreator`
   on every SA for each impersonator

## Configuration contract

| Key | Required | Meaning |
|---|---|---|
| `projectName` | yes | Project ID and display name |
| `consumerSAs` | yes | SA emails granted `secretmanager.secretAccessor` |
| `governanceStack` | no | Fully-qualified stack reference (`org/project/stack`) to read placement from |
| `governanceFolder` | no | Key into the governance stack's folder-ID export map (default `shared`) |
| `folderID` | fallback | Parent folder ID; required unless resolved via params or governance stack |
| `billingAccount` | fallback | Billing account ID; required unless resolved via params or governance stack |
| `secrets` | no | List of `{name, ref}`; `ref` is passed to the `SecretResolver` |
| `serviceAccounts` | no | List of `{id, displayName}` |
| `impersonators` | no | IAM members granted impersonation on all SAs |

Placement (folder ID and billing account) resolves in order: `Params`
(collapsed mode) → governance stack reference → explicit config.

## Deployment modes

```go
// Enterprise: identity as its own Pulumi program.
func main() { identity.Identity() }

// Collapsed: alongside other bounded contexts in one program.
identity.Apply(ctx, &identity.Params{FolderID: f, BillingAccount: b})
```

## Exports

| Name | Value |
|---|---|
| `projectId` | The identity project ID |
| `<sa-id>-email` | Email of each provisioned service account |
