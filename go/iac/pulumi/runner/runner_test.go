package runner_test

import (
	"context"
	"errors"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/iac/pulumi/runner"
)

// fakePulumiScript answers the commands the runner issues. failingPulumiScript
// fails every command without echoing its arguments, so any secret appearing in
// an error can only have come from the runner itself.
const (
	fakePulumiScript = `#!/bin/sh
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

	failingPulumiScript = "#!/bin/sh\necho 'error: failed to set config' >&2\nexit 1\n"
)

// The stubs are written once, from TestMain, before any test starts. That is
// deliberate. On Linux, exec fails with ETXTBSY while any process holds the
// file open for writing, and os.OpenFile sets O_CLOEXEC, which closes the
// descriptor at exec but not at fork. A stub written inside a t.Parallel()
// test is therefore exec'd while a child forked by a sibling test may still
// hold the writing descriptor it inherited. Writing the stubs while the
// process is still single-threaded removes the overlap rather than masking
// it. See issue #291.
var (
	fakePulumiBin    string
	failingPulumiBin string
)

func TestMain(m *testing.M) {
	dir, err := os.MkdirTemp("", "pulumi-runner-stubs")
	if err != nil {
		fmt.Fprintf(os.Stderr, "create stub dir: %v\n", err)
		os.Exit(1)
	}

	fakePulumiBin, err = writeStub(dir, "fake-pulumi", fakePulumiScript)
	if err == nil {
		failingPulumiBin, err = writeStub(dir, "failing-pulumi", failingPulumiScript)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "write stub: %v\n", err)
		os.RemoveAll(dir)
		os.Exit(1)
	}

	code := m.Run()
	os.RemoveAll(dir)
	os.Exit(code)
}

func writeStub(dir, name, body string) (string, error) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		return "", err
	}

	return path, nil
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
	logDir := filepath.Join(tmpDir, "logs")

	r, err := runner.New(
		tmpDir,
		runner.WithBinPath(fakePulumiBin),
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

func logNames(t *testing.T, logDir string) []string {
	t.Helper()
	entries, err := os.ReadDir(logDir)
	if err != nil {
		t.Fatalf("read log dir: %v", err)
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		names = append(names, e.Name())
	}

	return names
}

// TestRunner_SecretsNeverNamed proves that a secret value reaches neither the
// log file name nor the error string, and that a value longer than NAME_MAX
// cannot make the log file uncreatable (#252).
func TestRunner_SecretsNeverNamed(t *testing.T) {
	t.Parallel()

	const sentinel = "hunter2-SENTINEL"
	ctx := context.Background()

	t.Run("failure keeps the value out of the error and the file name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		logDir := filepath.Join(tmpDir, "logs")
		r, err := runner.New(tmpDir, runner.WithBinPath(failingPulumiBin), runner.WithLogDir(logDir))
		if err != nil {
			t.Fatalf("failed to initialize runner: %v", err)
		}

		err = r.SetSecret(ctx, "auth:token", sentinel)
		if err == nil {
			t.Fatal("expected SetSecret to fail")
		}
		if strings.Contains(err.Error(), sentinel) {
			t.Errorf("error string leaks the secret value: %v", err)
		}
		if !strings.Contains(err.Error(), "config set --secret auth:token") {
			t.Errorf("error string should still name the command and key, got: %v", err)
		}

		err = r.SetConfig(ctx, "gcp:region", sentinel)
		if err == nil {
			t.Fatal("expected SetConfig to fail")
		}
		if strings.Contains(err.Error(), sentinel) {
			t.Errorf("error string leaks the plaintext config value: %v", err)
		}

		for _, name := range logNames(t, logDir) {
			if strings.Contains(name, sentinel) {
				t.Errorf("log file name leaks the value: %s", name)
			}
		}
	})

	t.Run("oversized value stays inside NAME_MAX", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		logDir := filepath.Join(tmpDir, "logs")
		r, err := runner.New(tmpDir, runner.WithBinPath(fakePulumiBin), runner.WithLogDir(logDir))
		if err != nil {
			t.Fatalf("failed to initialize runner: %v", err)
		}

		long := strings.Repeat("x", 300)
		if err := r.SetSecret(ctx, "auth:token", long); err != nil {
			t.Fatalf("SetSecret with a 300-byte value failed: %v", err)
		}

		names := logNames(t, logDir)
		if len(names) == 0 {
			t.Fatal("expected a log file")
		}
		for _, name := range names {
			if len(name) > 255 {
				t.Errorf("log file name exceeds NAME_MAX (%d bytes): %s", len(name), name)
			}
			if strings.Contains(name, long) {
				t.Errorf("log file name contains the value: %s", name)
			}
		}
	})

	t.Run("ordinary commands keep a readable file name", func(t *testing.T) {
		t.Parallel()

		tmpDir := t.TempDir()
		logDir := filepath.Join(tmpDir, "logs")
		r, err := runner.New(tmpDir, runner.WithBinPath(fakePulumiBin), runner.WithLogDir(logDir))
		if err != nil {
			t.Fatalf("failed to initialize runner: %v", err)
		}

		if err := r.SelectOrCreateStack(ctx, "dev"); err != nil {
			t.Fatalf("SelectOrCreateStack failed: %v", err)
		}
		if err := r.SetSecret(ctx, "auth:token", sentinel); err != nil {
			t.Fatalf("SetSecret failed: %v", err)
		}
		if err := r.Up(ctx, runner.WithUpStack("dev")); err != nil {
			t.Fatalf("Up failed: %v", err)
		}
		if err := r.Destroy(ctx); err != nil {
			t.Fatalf("Destroy failed: %v", err)
		}

		want := map[string]bool{
			"pulumi-stack-select.log": true,
			"pulumi-config-set.log":   true,
			"pulumi-up.log":           true,
			"pulumi-destroy.log":      true,
		}
		got := logNames(t, logDir)
		for _, name := range got {
			if !want[name] {
				t.Errorf("unexpected log file name %q (want one of %v)", name, slices.Sorted(maps.Keys(want)))
			}
		}
		if len(got) != len(want) {
			t.Errorf("log files: got %v, want %d files", got, len(want))
		}
	})
}
