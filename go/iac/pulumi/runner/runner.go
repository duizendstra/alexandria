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

const (
	logDirPerm = 0o700

	// flagCwd points the CLI at the project directory on every call.
	flagCwd = "--cwd"
)

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

	return r.exec(ctx, "stack", "select", stackName, "--create", flagCwd, r.workDir)
}

// SelectStack selects an existing Pulumi stack.
func (r *Runner) SelectStack(ctx context.Context, stackName string) error {
	if strings.TrimSpace(stackName) == "" {
		return ErrEmptyStackName
	}

	return r.exec(ctx, "stack", "select", stackName, flagCwd, r.workDir)
}

// SetConfig sets a plaintext key-value configuration variable on the active stack.
func (r *Runner) SetConfig(ctx context.Context, key, val string) error {
	return r.exec(ctx, "config", "set", key, val, flagCwd, r.workDir)
}

// SetSecret sets an encrypted secret key-value configuration variable on the active stack.
func (r *Runner) SetSecret(ctx context.Context, key, val string) error {
	return r.exec(ctx, "config", "set", "--secret", key, val, flagCwd, r.workDir)
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

	args := []string{"up", "--yes", flagCwd, r.workDir}
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
	return r.exec(ctx, "destroy", "--yes", flagCwd, r.workDir)
}

// GetRawOutputs retrieves the outputs from the current Pulumi stack as raw JSON bytes.
func (r *Runner) GetRawOutputs(ctx context.Context) ([]byte, error) {
	logFile := filepath.Join(r.logDir, "pulumi-stack-output.json")
	call := procrun.Call{
		Name:   r.binPath,
		Args:   []string{"stack", "output", "--json", flagCwd, r.workDir},
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
	logFile := filepath.Join(r.logDir, fmt.Sprintf("pulumi-%s.log", logName(args)))
	call := procrun.Call{
		Name:   r.binPath,
		Args:   args,
		Output: logFile,
	}

	if err := r.proc.Run(ctx, &call); err != nil {
		return fmt.Errorf("pulumi %s failed: %w", strings.Join(redactArgs(args), " "), err)
	}

	return nil
}

const (
	// maxLogWords is how many leading subcommand words name a log file:
	// "config set", "stack select", "up". Values never reach the name.
	maxLogWords = 2

	// maxLogWordLen bounds each word so the file name stays well inside
	// NAME_MAX whatever the caller passes.
	maxLogWordLen = 32

	// redacted replaces a configuration value wherever an error would
	// otherwise print it.
	redacted = "[redacted]"
)

// keepsValue reports whether flag takes a following argument that the runner
// itself supplies and that may stay readable in an error.
func keepsValue(flag string) bool {
	switch flag {
	case flagCwd, "--stack":
		return true
	default:
		return false
	}
}

// logName derives a log file name from the leading subcommand words of args —
// the words before the first flag, capped in number and length and reduced to
// a portable character set. Positional values such as a configuration value
// or a stack name are never part of the name, so a secret cannot land on disk
// as a file name and an oversized value cannot push it past NAME_MAX.
func logName(args []string) string {
	words := make([]string, 0, maxLogWords)
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") || len(words) == maxLogWords {
			break
		}
		if w := safeWord(arg); w != "" {
			words = append(words, w)
		}
	}
	if len(words) == 0 {
		return "run"
	}

	return strings.Join(words, "-")
}

// safeWord keeps the ASCII letters, digits, '-' and '_' of s, up to
// maxLogWordLen bytes.
func safeWord(s string) string {
	var b strings.Builder
	for _, c := range s {
		if b.Len() == maxLogWordLen {
			break
		}
		switch {
		case c >= 'a' && c <= 'z', c >= 'A' && c <= 'Z', c >= '0' && c <= '9', c == '-', c == '_':
			b.WriteRune(c)
		}
	}

	return b.String()
}

// redactArgs returns a copy of args fit for an error message. For
// `config set` the configuration value — every positional argument after the
// key — is replaced by a placeholder whether or not the value is a secret; the
// key, the flags, and the values of flags for which keepsValue holds stay
// readable. Other commands are returned unchanged.
func redactArgs(args []string) []string {
	out := make([]string, len(args))
	copy(out, args)
	if len(args) < 2 || args[0] != "config" || args[1] != "set" {
		return out
	}

	positionals := 0
	for i := 2; i < len(out); i++ {
		if strings.HasPrefix(out[i], "-") {
			if keepsValue(out[i]) {
				i++ // Keep the flag's own value.
			}

			continue
		}
		positionals++
		if positionals > 1 {
			out[i] = redacted
		}
	}

	return out
}
