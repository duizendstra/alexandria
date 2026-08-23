# go/iac/pulumi/stackref

Typed readers for Pulumi stack reference outputs.

Pulumi's `StackReference.GetOutput` returns an untyped `AnyOutput`;
composition roots that chain stacks together (workload project IDs,
service account emails, …) want strings without repeating the same
type-assertion boilerplate at every call site.

```go
workloadRef, _ := pulumi.NewStackReference(ctx, "org/workloads/prod", nil)
projectID := stackref.RequireString(workloadRef, "computeProjectId")
```

`RequireString` resolves to the empty string when the referenced stack
has not exported the key yet (e.g. first deploy ordering), so downstream
resources fail with a clear GCP validation error instead of a nil panic.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- Pulumi composition roots that wire bounded contexts together via
  stack references.

### Load-Bearing Promises
1. **Present Values Are Returned As-Is**: a required string that exists comes
   back unchanged.
2. **Missing Means Empty, Not Panic**: a missing key yields the empty string,
   so a composition root decides how to handle absence rather than being
   aborted mid-wiring.

