# go/platform/passstore

Deploy-time secret retrieval from the local [pass](https://www.passwordstore.org/)
store, for tools that run on an operator's workstation (Pulumi programs,
provisioning scripts) and need credentials without a separate secrets
pipeline.

```go
apiKey := passstore.MustShow("vendors/example/api-key")

value, err := passstore.Show("vendors/example/base-url")
```

- `Show` shells out to `pass show <path>` with a 5-second timeout and
  returns the trimmed value.
- `MustShow` panics on failure — use it in Pulumi programs where a
  missing credential should abort the deploy before any resources
  change.

Zero dependencies. Requires the `pass` binary and an initialized store
on the machine running the deploy; this is an operator-workstation tool,
not something to run in CI.

## Consumers

- Pulumi composition roots that seed connector configuration at deploy
  time.
