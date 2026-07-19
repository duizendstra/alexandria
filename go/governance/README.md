# Governance

Organizational structure, classification, and resource controls.

## What is governance?

Governance defines **how cloud resources are organized and controlled**.
It answers: what folders exist, what environments are available, how resources
are classified, and who pays.

## Scope

Scope determines **where** governance operates — the organizational boundary.

| Scope | Meaning | GCP parent |
|---|---|---|
| **Folder** | Sub-org governance under an existing folder | `folders/123456` |
| **Organization** | Full org-level governance | `organizations/123456` |

Scope constrains which tiers are available.

## Tiers

Tiers describe **what** governance features are active — the maturity level.
Following Google Cloud naming: Starter / Standard / Enterprise.

### Tier × Scope matrix

| Feature | Starter | Standard | Enterprise |
|---|---|---|---|
| Folder hierarchy | Single root folder | Environment folders (dev/prod) | Full organizational tree |
| Classification | — | — | Tag keys and dimensions |
| Billing export | — | — | Billing account exported |
| Org-level exports | — | — | OrgID exported downstream |
| **Available at Folder scope** | ✅ | ✅ | ❌ |
| **Available at Organization scope** | ✅ | ✅ | ✅ |

Enterprise requires Organization scope. Classification and org-level exports
are GCP org-level resources — they cannot be created at folder level.

### Moving between tiers

Going **up** is additive and safe — Pulumi creates new resources:
- Starter → Standard: add environment children
- Standard → Enterprise: change scope to Organization, add dimensions

Going **down** is destructive — Pulumi deletes resources:
- Enterprise → Standard: removes tag keys, org exports
- Standard → Starter: removes environment folders

## Domain model

```
plan.Plan                    Desired governance state (pure data)
scope.Scope                  Organization or Folder — determines capabilities
hierarchy.Config             Folder tree definition
classification.Dimension     Tag key / label axis
```

The `Plan` is the port. This module is pure Go with zero external
dependencies — provisioning adapters (Pulumi, Terraform, any cloud)
consume a validated `Plan` and create the actual resources.
