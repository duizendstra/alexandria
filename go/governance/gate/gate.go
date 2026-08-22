package gate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/duizendstra/alexandria/go/governance/invariant"
)

// ErrGateBlocked is returned by Enforce when one or more rules violate the gate policy.
var ErrGateBlocked = errors.New("governance: gate verification failed")

// ErrUnknownPolicy is returned by Policy.Validate and Gate.Validate when the
// policy is not one of PolicyStrict, PolicyStandard or PolicyPermissive.
// A Gate with an unknown policy always evaluates to VerdictBlocked.
var ErrUnknownPolicy = errors.New("governance: unknown gate policy")

// Status is the evaluation outcome of an individual rule.
type Status string

const (
	// StatusPass indicates the rule requirements were completely met.
	StatusPass Status = "PASS"

	// StatusFail indicates the rule was violated (a defect or hard blocker).
	StatusFail Status = "FAIL"

	// StatusAnomaly indicates an unusual condition requiring operator attention.
	StatusAnomaly Status = "ANOMALY"

	// StatusSkipped indicates the rule was intentionally bypassed.
	StatusSkipped Status = "SKIPPED"
)

// Verdict is the overall gate evaluation decision.
type Verdict string

const (
	// VerdictPass indicates the gate permits execution under the configured policy.
	VerdictPass Verdict = "PASS"

	// VerdictBlocked indicates the gate rejects execution.
	VerdictBlocked Verdict = "BLOCKED"
)

// Policy defines how rule statuses are interpreted to determine the final Verdict.
type Policy string

const (
	// PolicyStrict blocks execution if any rule returns StatusFail or StatusAnomaly.
	PolicyStrict Policy = "STRICT"

	// PolicyStandard blocks execution if any rule returns StatusFail, but permits StatusAnomaly.
	PolicyStandard Policy = "STANDARD"

	// PolicyPermissive never blocks execution, recording all findings for audit.
	PolicyPermissive Policy = "PERMISSIVE"
)

// Validate reports ErrUnknownPolicy unless p is one of the defined policies.
// The empty Policy (the zero value) is unknown.
func (p Policy) Validate() error {
	switch p {
	case PolicyStrict, PolicyStandard, PolicyPermissive:
		return nil
	default:
		return fmt.Errorf("%w: %q (expected one of %s, %s, %s)",
			ErrUnknownPolicy, string(p), PolicyStrict, PolicyStandard, PolicyPermissive)
	}
}

// Result captures the evaluation output of a single rule.
type Result struct {
	RuleName string        `json:"rule_name"`
	Status   Status        `json:"status"`
	Reason   string        `json:"reason,omitempty"`
	Evidence []string      `json:"evidence,omitempty"`
	Duration time.Duration `json:"duration_ms"`
}

// Rule defines the interface for an evaluatable policy or invariant check.
type Rule interface {
	Name() string
	Evaluate(ctx context.Context) Result
}

// RuleFunc is an adapter allowing a plain function to act as a Rule.
type RuleFunc struct {
	name string
	fn   func(ctx context.Context) Result
}

// NewRule creates a Rule from a name and evaluation function.
func NewRule(name string, fn func(ctx context.Context) Result) *RuleFunc {
	return &RuleFunc{name: name, fn: fn}
}

// Name returns the rule name.
func (r *RuleFunc) Name() string {
	return r.name
}

// Evaluate runs the underlying rule function.
func (r *RuleFunc) Evaluate(ctx context.Context) Result {
	if r.fn == nil {
		return Result{
			RuleName: r.name,
			Status:   StatusPass,
		}
	}

	return r.fn(ctx)
}

// FromCheck converts an invariant.Check domain model into a gate Rule.
func FromCheck(c invariant.Check) *RuleFunc {
	return NewRule(c.Name, func(_ context.Context) Result {
		var status Status
		switch c.Status {
		case invariant.Pass:
			status = StatusPass
		case invariant.Fail:
			status = StatusFail
		case invariant.Anomaly:
			status = StatusAnomaly
		default:
			status = StatusFail
		}

		return Result{
			RuleName: c.Name,
			Status:   status,
			Evidence: c.Evidence,
		}
	})
}

// FromBuilder evaluates an invariant.Builder into a gate Rule.
func FromBuilder(b *invariant.Builder) *RuleFunc {
	if b == nil {
		return NewRule("nil_invariant", func(_ context.Context) Result {
			return Result{RuleName: "nil_invariant", Status: StatusPass}
		})
	}

	return FromCheck(b.Done())
}

// Report holds the complete results of evaluating a Gate.
type Report struct {
	GateName     string        `json:"gate_name"`
	EvaluatedAt  time.Time     `json:"evaluated_at"`
	Policy       Policy        `json:"policy"`
	Verdict      Verdict       `json:"verdict"`
	TotalRules   int           `json:"total_rules"`
	PassedRules  int           `json:"passed_rules"`
	FailedRules  int           `json:"failed_rules"`
	AnomalyRules int           `json:"anomaly_rules"`
	SkippedRules int           `json:"skipped_rules"`
	Results      []Result      `json:"results"`
	Duration     time.Duration `json:"duration_ms"`
}

// reportJSON provides formatted serialization for Report.
type reportJSON struct {
	GateName     string   `json:"gate_name"`
	EvaluatedAt  string   `json:"evaluated_at"`
	Policy       string   `json:"policy"`
	Verdict      string   `json:"verdict"`
	TotalRules   int      `json:"total_rules"`
	PassedRules  int      `json:"passed_rules"`
	FailedRules  int      `json:"failed_rules"`
	AnomalyRules int      `json:"anomaly_rules"`
	SkippedRules int      `json:"skipped_rules"`
	Results      []Result `json:"results"`
	DurationMs   int64    `json:"duration_ms"`
}

// MarshalJSON encodes Report into formatted JSON.
func (r *Report) MarshalJSON() ([]byte, error) {
	wire := reportJSON{
		GateName:     r.GateName,
		EvaluatedAt:  r.EvaluatedAt.Format(time.RFC3339),
		Policy:       string(r.Policy),
		Verdict:      string(r.Verdict),
		TotalRules:   r.TotalRules,
		PassedRules:  r.PassedRules,
		FailedRules:  r.FailedRules,
		AnomalyRules: r.AnomalyRules,
		SkippedRules: r.SkippedRules,
		Results:      r.Results,
		DurationMs:   r.Duration.Milliseconds(),
	}

	data, err := json.Marshal(wire)
	if err != nil {
		return nil, fmt.Errorf("gate: marshal report: %w", err)
	}

	return data, nil
}

// Summary returns a concise human-readable summary of the gate report.
func (r *Report) Summary() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Gate %q: %s (%d total, %d passed, %d failed, %d anomalies, %d skipped in %v)\n",
		r.GateName, r.Verdict, r.TotalRules, r.PassedRules, r.FailedRules, r.AnomalyRules, r.SkippedRules, r.Duration)

	for _, res := range r.Results {
		fmt.Fprintf(&sb, "  [%s] %s", res.Status, res.RuleName)
		if res.Reason != "" {
			sb.WriteString(" - " + res.Reason)
		}
		sb.WriteString("\n")
		for _, ev := range res.Evidence {
			fmt.Fprintf(&sb, "      %s\n", ev)
		}
	}

	return sb.String()
}

// Option configures a Gate instance.
type Option func(*Gate)

// WithPolicy sets the evaluation policy.
func WithPolicy(p Policy) Option {
	return func(g *Gate) {
		g.policy = p
	}
}

// WithRules adds rules to the gate.
func WithRules(rules ...Rule) Option {
	return func(g *Gate) {
		g.rules = append(g.rules, rules...)
	}
}

// Gate orchestrates the execution and enforcement of policy rules.
type Gate struct {
	name   string
	policy Policy
	rules  []Rule
}

// New creates a new Gate with default StandardPolicy.
func New(name string, opts ...Option) *Gate {
	g := &Gate{
		name:   name,
		policy: PolicyStandard,
	}

	for _, opt := range opts {
		opt(g)
	}

	return g
}

// Validate reports ErrUnknownPolicy when the gate's policy is not one of the
// defined policies. Call it at construction time to reject a misconfigured
// gate before it is evaluated; Evaluate itself never fails open on an unknown
// policy, it blocks.
func (g *Gate) Validate() error {
	return g.policy.Validate()
}

// AddRule appends a single rule to the gate.
func (g *Gate) AddRule(r Rule) *Gate {
	if r != nil {
		g.rules = append(g.rules, r)
	}

	return g
}

// AddRules appends multiple rules to the gate.
func (g *Gate) AddRules(rules ...Rule) *Gate {
	for _, r := range rules {
		if r != nil {
			g.rules = append(g.rules, r)
		}
	}

	return g
}

// Evaluate runs all rules against the provided context and produces a Report without returning an error.
func (g *Gate) Evaluate(ctx context.Context) *Report {
	start := time.Now()
	report := &Report{
		GateName:    g.name,
		EvaluatedAt: start,
		Policy:      g.policy,
		Verdict:     VerdictPass,
		TotalRules:  len(g.rules),
		Results:     make([]Result, 0, len(g.rules)),
	}

	for _, rule := range g.rules {
		if err := ctx.Err(); err != nil {
			report.Results = append(report.Results, Result{
				RuleName: rule.Name(),
				Status:   StatusFail,
				Reason:   fmt.Sprintf("evaluation aborted: %v", err),
			})
			report.FailedRules++

			continue
		}

		rStart := time.Now()
		res := rule.Evaluate(ctx)
		res.Duration = time.Since(rStart)
		if res.RuleName == "" {
			res.RuleName = rule.Name()
		}

		switch res.Status {
		case StatusPass:
			report.PassedRules++
		case StatusFail:
			report.FailedRules++
		case StatusAnomaly:
			report.AnomalyRules++
		case StatusSkipped:
			report.SkippedRules++
		default:
			res.Status = StatusFail
			report.FailedRules++
		}

		report.Results = append(report.Results, res)
	}

	report.Duration = time.Since(start)
	g.applyVerdict(report)

	return report
}

// Enforce evaluates all rules and returns ErrGateBlocked if the final Verdict is VerdictBlocked.
func (g *Gate) Enforce(ctx context.Context) (*Report, error) {
	report := g.Evaluate(ctx)
	if report.Verdict == VerdictBlocked {
		return report, fmt.Errorf("%w: gate %q failed policy %s (%d failed, %d anomalies)",
			ErrGateBlocked, g.name, g.policy, report.FailedRules, report.AnomalyRules)
	}

	return report, nil
}

// applyVerdict sets the report's overall Verdict from its rule counts under
// the gate's policy.
func (g *Gate) applyVerdict(report *Report) {
	switch g.policy {
	case PolicyStrict:
		if report.FailedRules > 0 || report.AnomalyRules > 0 {
			report.Verdict = VerdictBlocked
		}
	case PolicyStandard:
		if report.FailedRules > 0 {
			report.Verdict = VerdictBlocked
		}
	case PolicyPermissive:
		report.Verdict = VerdictPass
	default:
		// An unknown or empty policy (including a zero-valued Gate) must not
		// fail open: block, and record why as a failed result so the report
		// and its Summary explain the verdict.
		report.Results = append(report.Results, Result{
			RuleName: "gate-policy",
			Status:   StatusFail,
			Reason:   g.policy.Validate().Error(),
		})
		report.TotalRules++
		report.FailedRules++
		report.Verdict = VerdictBlocked
	}
}
