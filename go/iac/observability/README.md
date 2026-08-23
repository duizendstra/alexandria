# go/iac/observability

Configuration-driven Pulumi blueprint for the **Observability** bounded
context: a dedicated project with a BigQuery log-analytics dataset and
an org-level audit-log sink routed into it.

## What it provisions

1. A GCP project (BigQuery, Logging, Monitoring APIs enabled, deletion
   policy `PREVENT`, no default VPC) in a governance-managed folder
2. An `org_logs` BigQuery dataset for aggregated audit/activity logs
3. An org-level log sink (`org-audit-to-bigquery`) filtering audit logs
   (`cloudaudit.googleapis.com`), extended by any configured log names,
   from the whole organization into the dataset (`includeChildren: true`)
4. *(optional)* An ops-email notification channel and, per configured
   `uptimeTargets` entry, an HTTPS uptime check with a failure alert
   routed to that channel

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
| `alertEmail` | no | Ops recipient; when set, an email notification channel is created and uptime alerts route to it |
| `uptimeTargets` | no | JSON array of HTTPS endpoints to monitor (see below) |
| `sinkExtraLogNames` | no | JSON array of log names to add to the org sink's allowlist, alongside audit logs |

### `sinkExtraLogNames`

A JSON array of strings. The sink always captures audit logs
(`cloudaudit.googleapis.com`); each entry here is *added* to that
allowlist, not a replacement for it — a caller can't accidentally drop
audit capture by supplying an incomplete list. Each entry is matched
against a log's `logName` with Cloud Logging's `:` (has/substring)
operator — the same test the sink's original hardcoded filter used — and
all entries are OR'd together.

**The sink is org-scoped with `includeChildren: true`, so entries are
validated, not just documented:** each one must match
`^[A-Za-z0-9._/%-]+$` (letters, digits, `.`, `_`, `/`, `%`, `-`) or `Apply`
fails the deploy. This rejects an empty string, which would otherwise
match every log in every project, and rejects any character — starting
with `"` — that could break out of the filter's quoted clause and inject
arbitrary Cloud Logging filter syntax.

```json
["example.googleapis.com"]
```

### `uptimeTargets`

A JSON array; each target provisions an HTTPS uptime check and a failure alert.
The probed host is read from a stack reference's URL output (commonly a Cloud
Run URI) and stripped to its host.

| Field | Required | Meaning |
|---|---|---|
| `displayName` | yes | Uptime check / alert display name |
| `stackRef` | yes | Fully-qualified stack reference (`org/project/stack`) exporting the URL |
| `urlOutputKey` | no | Stack-reference output key holding the URL (default `frontendUrl`) |
| `statusClasses` | no | Accepted response classes (default `["2xx","3xx"]` — 3xx covers the IAP sign-in redirect) |

```json
[{ "displayName": "example dev frontend", "stackRef": "organization/example-gcp-frontend/example-dev" }]
```

## Exports

`projectId`, `logDatasetId`, `sinkWriterIdentity`

## Deployment modes

- **Enterprise** (standalone stack): `func main() { observability.Observability() }`
- **Collapsed** (alongside other BCs): `observability.Apply(ctx, &observability.Params{...})`

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **Pulumi composition roots**: stacks wiring monitoring and log routing across
  an estate.
- **Uptime and alerting owners**: configuration naming targets that live in
  other stacks.

### Load-Bearing Promises
1. **Log Names Fail Closed**: an extra sink log name that is empty, that breaks
   out of its quoting, or that tries the same break percent-encoded, fails the
   apply. A filter is never assembled from a value that could change its
   meaning.
2. **Omitting Extras Still Yields A Sink**: with no extra log names supplied
   the default sink is still produced, and caller-supplied names are *added to*
   that default rather than replacing it.
3. **A Stack Is Read Once**: uptime targets that share a stack read it a single
   time, including when the governance stack is itself a target. Adding targets
   does not multiply cross-stack reads.
