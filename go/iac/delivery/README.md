# go/iac/delivery

Configuration-driven Pulumi blueprint for the **Delivery** bounded
context: a dedicated CI/CD project with an Artifact Registry, a Git
hosting connection, per-repo Cloud Build triggers, and cross-project
reader grants for consumer workloads.

## What it provisions

1. A GCP project (Cloud Build, Artifact Registry, Secret Manager, IAM,
   Compute APIs enabled, deletion policy `PREVENT`, no default VPC) in
   a governance-managed folder
2. A `DOCKER` Artifact Registry repository, with write access for the
   Cloud Build default SA
3. A Cloud Build v2 connection to GitHub (app installation +
   OAuth-token secret version); until both are configured, the stack
   exports a `nextStep` hint and stops after the registry. Once
   configured, the Compute default SA is granted
   `secretmanager.secretAccessor` on the OAuth token secret (Cloud
   Build v2 triggers run as that SA and read the authorizer
   credential)
4. Repository links and tag-push build triggers, all from config
5. `artifactregistry.reader` for each consumer workload stack's Cloud
   Run service agent (project numbers read via stack references)

## Configuration contract

| Key | Required | Meaning |
|---|---|---|
| `projectName` | yes | Project ID and display name |
| `registryId` | yes | Artifact Registry repository ID |
| `registryDescription` | no | Registry description (default "Container images") |
| `region` | no | Deployment region (default `europe-west4`) |
| `governanceStack` | no | Fully-qualified stack reference (`org/project/stack`) to read placement from |
| `governanceFolder` | no | Key into the governance stack's folder-ID export map (default `shared`) |
| `folderID` | fallback | Parent folder ID; required unless resolved via params or governance stack |
| `billingAccount` | fallback | Billing account ID; required unless resolved via params or governance stack |
| `githubConnectionName` | no | Connection resource name (default `github`) |
| `githubAppInstallationId` | for CI | GitHub app installation ID |
| `githubOAuthSecretVersion` | for CI | Full secret version path for the OAuth token |
| `repositories` | no | List of `{name, remoteURI, triggers[]}`; triggers are `{name, tagPattern, configFile, requireApproval, substitutions}` |
| `consumerWorkloadStacks` | no | Workload stack references whose Cloud Run agents get registry read access |

## Exports

`projectId`, `dockerRepoId` (plus `nextStep` while the GitHub
connection is unconfigured)

## Deployment modes

- **Enterprise** (standalone stack): `func main() { delivery.Delivery() }`
- **Collapsed** (alongside other BCs): `delivery.Apply(ctx, &delivery.Params{...})`
