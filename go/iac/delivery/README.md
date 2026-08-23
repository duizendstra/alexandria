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
   identity builds run as: the `buildServiceAccount` when configured,
   otherwise both the Compute default SA (what new projects build as
   since Cloud Build's 2024 default change) and the legacy Cloud Build
   SA (`<number>@cloudbuild.gserviceaccount.com`)
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
| `buildServiceAccount` | no | Email of the service account builds run as; it becomes the sole `artifactregistry.writer` grantee. Unset: both the Compute default SA and the legacy Cloud Build SA are granted |
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

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Pulumi composition roots**: stacks wiring the Delivery bounded context into
  a wider platform.
- **Build and deploy pipelines**: infrastructure granting the identities that
  push images and roll out revisions.

### Load-Bearing Promises
1. **Absent Is Not Malformed**: an omitted optional block applies cleanly and
   creates nothing. A *malformed* block fails the apply — it is never skipped
   silently, because a typo that quietly provisions nothing is the failure
   mode this blueprint exists to prevent.
2. **Well-Formed Means Created**: a valid optional block creates its resources.
   Configuration that parses is configuration that takes effect.
3. **Both Default Build Identities Are Covered**: the writer grant reaches both
   default build service accounts, so a project on either default does not
   silently lack permission.
4. **A Configured Build Account Is Honoured**: when a build service account is
   named, the writer grant lands on that account rather than on a default.
