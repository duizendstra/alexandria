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

## Consumers

- Pulumi composition roots that wire bounded contexts together via
  stack references.
