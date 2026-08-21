package runner_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/runner"
)

func createFakePulumiScript(t *testing.T, binDir string) string {
	t.Helper()
	fakeBin := filepath.Join(binDir, "fake-pulumi")
	script := `#!/bin/sh
cmd="$1"
subcmd="$2"

if [ "$cmd" = "stack" ] && [ "$subcmd" = "output" ]; then
    echo '{"bucketName":"my-test-bucket","projectID":"test-proj-123","serviceUrl":"https://my-service.run.app"}'
    exit 0
fi

if [ "$cmd" = "stack" ] && [ "$subcmd" = "select" ]; then
    exit 0
fi

if [ "$cmd" = "config" ] && [ "$subcmd" = "set" ]; then
    exit 0
fi

if [ "$cmd" = "up" ]; then
    echo "Updating (dev)..."
    echo "Resources: + 2 created"
    exit 0
fi

if [ "$cmd" = "destroy" ]; then
    echo "Destroying (dev)..."
    echo "Resources: - 2 deleted"
    exit 0
fi

if [ "$cmd" = "fail" ]; then
    echo "error: failed to update stack" >&2
    exit 1
fi

echo "unhandled command: $@" >&2
exit 0
`
	if err := os.WriteFile(fakeBin, []byte(script), 0o755); err != nil {
		t.Fatalf("failed to write fake pulumi script: %v", err)
	}

	return fakeBin
}

func TestNew_Validation(t *testing.T) {
	t.Parallel()

	_, err := runner.New("")
	if !errors.Is(err, runner.ErrEmptyWorkDir) {
		t.Fatalf("expected ErrEmptyWorkDir, got: %v", err)
	}

	_, err = runner.New("   ")
	if !errors.Is(err, runner.ErrEmptyWorkDir) {
		t.Fatalf("expected ErrEmptyWorkDir for whitespace, got: %v", err)
	}
}

func TestRunner_Operations(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	fakeBin := createFakePulumiScript(t, tmpDir)
	logDir := filepath.Join(tmpDir, "logs")

	r, err := runner.New(
		tmpDir,
		runner.WithBinPath(fakeBin),
		runner.WithLogDir(logDir),
		runner.WithEnv(map[string]string{"PULUMI_ACCESS_TOKEN": "test-token"}),
		runner.WithScrub("CLOUDSDK_"),
	)
	if err != nil {
		t.Fatalf("failed to initialize runner: %v", err)
	}

	if r.WorkDir() != tmpDir {
		t.Errorf("workDir: got %s, want %s", r.WorkDir(), tmpDir)
	}

	ctx := context.Background()

	// 1. Stack select / create.
	if err := r.SelectOrCreateStack(ctx, "dev"); err != nil {
		t.Errorf("SelectOrCreateStack failed: %v", err)
	}
	if err := r.SelectStack(ctx, "dev"); err != nil {
		t.Errorf("SelectStack failed: %v", err)
	}
	if err := r.SelectOrCreateStack(ctx, ""); !errors.Is(err, runner.ErrEmptyStackName) {
		t.Errorf("expected ErrEmptyStackName, got: %v", err)
	}
	if err := r.SelectStack(ctx, ""); !errors.Is(err, runner.ErrEmptyStackName) {
		t.Errorf("expected ErrEmptyStackName, got: %v", err)
	}

	// 2. Config set.
	if err := r.SetConfig(ctx, "gcp:region", "europe-west4"); err != nil {
		t.Errorf("SetConfig failed: %v", err)
	}
	if err := r.SetSecret(ctx, "auth:secret", "super-secret"); err != nil {
		t.Errorf("SetSecret failed: %v", err)
	}
	if err := r.SetConfigs(ctx, map[string]string{
		"a": "1",
		"b": "2",
	}); err != nil {
		t.Errorf("SetConfigs failed: %v", err)
	}

	// 3. Up.
	if err := r.Up(ctx, runner.WithUpStack("dev"), runner.WithSkipPreview(true)); err != nil {
		t.Errorf("Up failed: %v", err)
	}

	// 4. Destroy.
	if err := r.Destroy(ctx); err != nil {
		t.Errorf("Destroy failed: %v", err)
	}

	// 5. Outputs.
	raw, err := r.GetRawOutputs(ctx)
	if err != nil {
		t.Fatalf("GetRawOutputs failed: %v", err)
	}
	if len(raw) == 0 {
		t.Fatal("expected non-empty raw outputs")
	}

	type StackOutputs struct {
		BucketName string `json:"bucketName"`
		ProjectID  string `json:"projectID"`
		ServiceURL string `json:"serviceUrl"`
	}

	var outputs StackOutputs
	if err := r.GetOutputs(ctx, &outputs); err != nil {
		t.Fatalf("GetOutputs failed: %v", err)
	}

	if outputs.BucketName != "my-test-bucket" {
		t.Errorf("BucketName: got %s, want my-test-bucket", outputs.BucketName)
	}
	if outputs.ProjectID != "test-proj-123" {
		t.Errorf("ProjectID: got %s, want test-proj-123", outputs.ProjectID)
	}
	if outputs.ServiceURL != "https://my-service.run.app" {
		t.Errorf("ServiceURL: got %s, want https://my-service.run.app", outputs.ServiceURL)
	}

	if err := r.GetOutputs(ctx, nil); !errors.Is(err, runner.ErrNilDestination) {
		t.Errorf("expected ErrNilDestination, got: %v", err)
	}
}
