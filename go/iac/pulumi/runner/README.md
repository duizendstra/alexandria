# go/iac/pulumi/runner

`go/iac/pulumi/runner` drives the Pulumi CLI from Go: select or create a stack, set configuration, deploy or destroy it, and read its outputs back as a typed struct.

It shells out rather than embedding Pulumi's automation API, so a provisioning tool gets the CLI it already trusts, running under the environment control of [`go/platform/procrun`](../../../platform/procrun/) — and it takes care of the two things that go wrong when you drive a configuration CLI by hand: **a secret must not survive in a file name, and it must not survive in an error string**.

## Features

- **Typed Stack Outputs**: `GetOutputs` unmarshals `pulumi stack output --json` straight into a caller's struct; `GetRawOutputs` returns the untouched JSON for a caller that would rather decode it itself.
- **Stack And Configuration Operations**: `SelectStack`, `SelectOrCreateStack`, `SetConfig`, `SetSecret`, `SetConfigs`, `Up` (with `WithUpStack` and `WithSkipPreview`), and `Destroy`.
- **Values Never Name A File**: a log file is named from the leading subcommand words only — `pulumi-config-set.log`, `pulumi-up.log`. A positional value cannot land on disk as a file name, and cannot push the name past `NAME_MAX`.
- **Values Never Reach An Error**: for `config set`, every positional after the key is replaced by `[redacted]` — secret or not — while the subcommand and the key stay readable, so a failure is still diagnosable.
- **Inherited Environment Control**: `WithEnv` fixes variables for every call and `WithScrub` drops inherited prefixes; both are passed through to `procrun`, which owns those guarantees.
- **Ordered Bulk Configuration**: `SetConfigs` applies keys in sorted order so a run is reproducible, and stops at the first failure.

## Installation

```bash
go get github.com/duizendstra/alexandria/go/iac/pulumi/runner
```

## Quick Start

### Deploying A Stack And Reading Its Outputs

```go
package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/duizendstra/alexandria/go/iac/pulumi/runner"
)

type stackOutputs struct {
	ServiceURL string `json:"serviceUrl"`
	BucketName string `json:"bucketName"`
}

func main() {
	ctx := context.Background()

	r, err := runner.New(
		"./infra",
		// Fix what the CLI needs; drop what the developer's shell carries.
		runner.WithEnv(map[string]string{
			"PULUMI_ACCESS_TOKEN": os.Getenv("PULUMI_ACCESS_TOKEN"),
		}),
		runner.WithScrub("CLOUDSDK_"),
		runner.WithLogDir("./deploy-logs"),
	)
	if err != nil {
		slog.Error("init runner", slog.Any("err", err))
		os.Exit(1)
	}

	if err := r.SelectOrCreateStack(ctx, "dev"); err != nil {
		slog.Error("select stack", slog.Any("err", err))
		os.Exit(1)
	}

	// Applied in sorted order, so two runs issue the same sequence of calls.
	if err := r.SetConfigs(ctx, map[string]string{
		"gcp:region":  "europe-west4",
		"app:logging": "structured",
	}); err != nil {
		slog.Error("set config", slog.Any("err", err))
		os.Exit(1)
	}

	// The value is redacted from any error this returns, and never appears
	// in the name of the log file the call writes.
	if err := r.SetSecret(ctx, "app:apiToken", os.Getenv("APP_API_TOKEN")); err != nil {
		slog.Error("set secret", slog.Any("err", err))
		os.Exit(1)
	}

	if err := r.Up(ctx, runner.WithUpStack("dev"), runner.WithSkipPreview(true)); err != nil {
		slog.Error("deploy", slog.Any("err", err))
		os.Exit(1)
	}

	var out stackOutputs
	if err := r.GetOutputs(ctx, &out); err != nil {
		slog.Error("read outputs", slog.Any("err", err))
		os.Exit(1)
	}

	slog.Info("deployed", slog.String("url", out.ServiceURL), slog.String("bucket", out.BucketName))
}
```

## SRE & Performance Hardening details

1. **Nothing Positional Reaches The Filesystem**: the log name is built from the subcommand words before the first flag, capped at two words of 32 bytes each and reduced to `[A-Za-z0-9_-]`. A 300-byte configuration value therefore cannot produce a name that the filesystem refuses, and a secret cannot be read off a directory listing.
2. **Redaction Is Not Conditional On Secrecy**: `SetConfig` and `SetSecret` share one redaction path, so a value that turns out to have been sensitive is already protected — the caller does not have to have classified it correctly in advance.
3. **The Log Directory Is Created 0700**: `New` creates the directory if it is missing, owner-only. Logs hold whatever the CLI prints, which is not the runner's to sanitise.
4. **Output Goes To A File, Not A Pipe**: inherited from `procrun` — a long deployment cannot deadlock or be truncated by a reader that stopped early.

## Consumers & Load-Bearing Promises

### Consumer Archetypes
- **CLI provisioners and migrators**: terminal tools that stand up or tear down
  an environment as one step of an ordered flow, and must report which step
  failed.
- **Integration test harnesses**: suites that create a real stack, assert
  against its outputs, and destroy it again.

### Load-Bearing Promises
1. **A Value Never Names A File**: the log file for a call is named from the
   subcommand words alone. Neither a secret nor a 300-byte plaintext value
   contributes to the name, and the name stays inside `NAME_MAX`.
2. **A Configuration Value Never Reaches An Error**: when `config set` fails,
   every positional after the key is `[redacted]` — whether it was set with
   `SetSecret` or `SetConfig`.
3. **A Redacted Error Is Still A Useful Error**: the subcommand and the
   configuration key remain readable in the message, so the failing step is
   identifiable without the value.
4. **Empty Or Nil Input Is Refused Before The CLI Runs**: an empty or
   whitespace-only working directory (`ErrEmptyWorkDir`), an empty stack name
   on either select path (`ErrEmptyStackName`), and a nil destination for
   `GetOutputs` (`ErrNilDestination`) are sentinel errors, not a CLI invocation
   with a missing argument.
5. **Outputs Arrive Typed, Or Raw If You Prefer**: `GetOutputs` decodes the
   stack's JSON outputs into the caller's struct; `GetRawOutputs` returns the
   same bytes undecoded.
6. **Log Names Say Which Step Wrote Them**: `pulumi-stack-select.log`,
   `pulumi-config-set.log`, `pulumi-up.log`, `pulumi-destroy.log` — a failing
   run is diagnosable by opening the file named after the step.
