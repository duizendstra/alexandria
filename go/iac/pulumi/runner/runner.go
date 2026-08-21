package runner

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/duizendstra/alexandria/go/platform/procrun"
)

const logDirPerm = 0o700

// Sentinel errors.
var (
	// ErrEmptyWorkDir is returned when Runner is initialized with an empty WorkDir.
	ErrEmptyWorkDir = errors.New("pulumi workDir cannot be empty")

	// ErrEmptyStackName is returned when a stack operation receives an empty stack name.
	ErrEmptyStackName = errors.New("stack name cannot be empty")

	// ErrNilDestination is returned when GetOutputs receives a nil destination pointer.
	ErrNilDestination = errors.New("destination cannot be nil")
)

// Option configures a Runner.
type Option func(*options)

type options struct {
	binPath string
	env     map[string]string
	scrub   []string
	logDir  string
}

// WithBinPath sets a custom path or name for the Pulumi executable.
func WithBinPath(path string) Option {
	return func(o *options) {
		o.binPath = path
	}
}

// WithEnv configures environment variables passed to all Pulumi CLI executions.
func WithEnv(env map[string]string) Option {
	return func(o *options) {
		o.env = env
	}
}

// WithScrub specifies variable name prefixes to scrub from the parent environment.
func WithScrub(prefixes ...string) Option {
	return func(o *options) {
		o.scrub = append(o.scrub, prefixes...)
	}
}

// WithLogDir sets the directory where CLI execution logs are stored.
func WithLogDir(dir string) Option {
	return func(o *options) {
		o.logDir = dir
	}
}

// Runner executes Pulumi CLI commands within a designated project directory.
type Runner struct {
	workDir string
	binPath string
	logDir  string
	proc    *procrun.Runner
	env     map[string]string
}

// New creates a new Pulumi Runner for the given working directory.
func New(workDir string, opts ...Option) (*Runner, error) {
	if strings.TrimSpace(workDir) == "" {
		return nil, ErrEmptyWorkDir
	}

	cfg := options{
		binPath: "pulumi",
		env:     make(map[string]string),
	}
	for _, opt := range opts {
		opt(&cfg)
	}

	logDir := cfg.logDir
	if logDir == "" {
		logDir = filepath.Join(os.TempDir(), "alexandria-pulumi-logs")
	}

	if err := os.MkdirAll(logDir, logDirPerm); err != nil {
		return nil, fmt.Errorf("create log directory %s: %w", logDir, err)
	}

	pr := &procrun.Runner{
		Env:   cfg.env,
		Scrub: cfg.scrub,
	}

	return &Runner{
		workDir: workDir,
		binPath: cfg.binPath,
		logDir:  logDir,
		proc:    pr,
		env:     cfg.env,
	}, nil
}

// WorkDir returns the configured working directory of the Runner.
func (r *Runner) WorkDir() string {
	return r.workDir
}

// SelectOrCreateStack initializes and selects a Pulumi stack with `--create`.
func (r *Runner) SelectOrCreateStack(ctx context.Context, stackName string) error {
	if strings.TrimSpace(stackName) == "" {
		return ErrEmptyStackName
	}

	return r.exec(ctx, "stack", "select", stackName, "--create", "--cwd", r.workDir)
}

// SelectStack selects an existing Pulumi stack.
func (r *Runner) SelectStack(ctx context.Context, stackName string) error {
	if strings.TrimSpace(stackName) == "" {
		return ErrEmptyStackName
	}

	return r.exec(ctx, "stack", "select", stackName, "--cwd", r.workDir)
}

// SetConfig sets a plaintext key-value configuration variable on the active stack.
func (r *Runner) SetConfig(ctx context.Context, key, val string) error {
	return r.exec(ctx, "config", "set", key, val, "--cwd", r.workDir)
}

// SetSecret sets an encrypted secret key-value configuration variable on the active stack.
func (r *Runner) SetSecret(ctx context.Context, key, val string) error {
	return r.exec(ctx, "config", "set", "--secret", key, val, "--cwd", r.workDir)
}

// SetConfigs sets multiple configuration key-values in deterministic alphabetical order.
func (r *Runner) SetConfigs(ctx context.Context, configs map[string]string) error {
	keys := make([]string, 0, len(configs))
	for k := range configs {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		if err := r.SetConfig(ctx, k, configs[k]); err != nil {
			return err
		}
	}

	return nil
}

// UpOptions configures a `pulumi up` execution.
type UpOptions struct {
	Stack       string
	SkipPreview bool
}

// UpOption modifies UpOptions.
type UpOption func(*UpOptions)

// WithUpStack targets a specific stack for the deployment.
func WithUpStack(stack string) UpOption {
	return func(o *UpOptions) {
		o.Stack = stack
	}
}

// WithSkipPreview passes `--skip-preview` to `pulumi up`.
func WithSkipPreview(skip bool) UpOption {
	return func(o *UpOptions) {
		o.SkipPreview = skip
	}
}

// Up executes `pulumi up --yes` on the stack.
func (r *Runner) Up(ctx context.Context, opts ...UpOption) error {
	var upOpts UpOptions
	for _, opt := range opts {
		opt(&upOpts)
	}

	args := []string{"up", "--yes", "--cwd", r.workDir}
	if upOpts.Stack != "" {
		args = append(args, "--stack", upOpts.Stack)
	}
	if upOpts.SkipPreview {
		args = append(args, "--skip-preview")
	}

	return r.exec(ctx, args...)
}

// Destroy executes `pulumi destroy --yes` on the stack.
func (r *Runner) Destroy(ctx context.Context) error {
	return r.exec(ctx, "destroy", "--yes", "--cwd", r.workDir)
}

// GetRawOutputs retrieves the outputs from the current Pulumi stack as raw JSON bytes.
func (r *Runner) GetRawOutputs(ctx context.Context) ([]byte, error) {
	logFile := filepath.Join(r.logDir, "pulumi-stack-output.json")
	call := procrun.Call{
		Name:   r.binPath,
		Args:   []string{"stack", "output", "--json", "--cwd", r.workDir},
		Output: logFile,
	}

	if err := r.proc.Run(ctx, &call); err != nil {
		return nil, fmt.Errorf("pulumi stack output failed: %w", err)
	}

	data, err := os.ReadFile(logFile) //nolint:gosec // logFile is constructed securely inside logDir
	if err != nil {
		return nil, fmt.Errorf("read stack output %s: %w", logFile, err)
	}

	return data, nil
}

// GetOutputs unmarshals the JSON outputs of the active stack into the provided destination pointer.
func (r *Runner) GetOutputs(ctx context.Context, dest any) error {
	if dest == nil {
		return ErrNilDestination
	}

	raw, err := r.GetRawOutputs(ctx)
	if err != nil {
		return err
	}

	if err := json.Unmarshal(raw, dest); err != nil {
		return fmt.Errorf("unmarshal pulumi outputs: %w", err)
	}

	return nil
}

func (r *Runner) exec(ctx context.Context, args ...string) error {
	logFile := filepath.Join(r.logDir, fmt.Sprintf("pulumi-%s.log", sanitizeCommand(args)))
	call := procrun.Call{
		Name:   r.binPath,
		Args:   args,
		Output: logFile,
	}

	if err := r.proc.Run(ctx, &call); err != nil {
		return fmt.Errorf("pulumi %s failed: %w", strings.Join(args, " "), err)
	}

	return nil
}

func sanitizeCommand(args []string) string {
	if len(args) == 0 {
		return "run"
	}
	cleaned := make([]string, 0, len(args))
	for _, arg := range args {
		clean := strings.TrimPrefix(arg, "--")
		clean = strings.ReplaceAll(clean, "/", "_")
		cleaned = append(cleaned, clean)
	}

	return strings.Join(cleaned, "-")
}
