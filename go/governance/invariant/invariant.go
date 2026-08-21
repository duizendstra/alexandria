package invariant

import (
	"fmt"
	"sort"
)

// Status is the outcome of one check, or of a whole suite.
type Status string

// The three outcomes. They are ordered by severity: Fail outranks Anomaly,
// which outranks Pass.
const (
	// Pass means the rule held.
	Pass Status = "PASS"

	// Fail means the rule was violated. This is a defect.
	Fail Status = "FAIL"

	// Anomaly means the rule could not be confirmed, or held in a way that
	// deserves a human look — an unreadable input, an unexpected-but-not-
	// forbidden shape. It is deliberately distinct from Fail: treating
	// "unsure" as "broken" trains people to ignore failures.
	Anomaly Status = "ANOMALY"
)

// Check is the result of evaluating one named rule.
type Check struct {
	// Name identifies the rule in reports, e.g. "source-residue".
	Name string `json:"name"`

	// Invariant is the rule's number in the specification it implements.
	// Zero when the suite does not number its rules.
	Invariant int `json:"invariant"`

	// Status is the outcome.
	Status Status `json:"status"`

	// Evidence lists what the rule observed, in the order recorded. It is
	// populated on success as well as on failure: a passing check that
	// explains what it compared is auditable, one that says nothing is not.
	Evidence []string `json:"evidence"`
}

// Labels are the prefixes put in front of evidence lines. Failure and anomaly
// lines are prefixed so the severity survives when the evidence is read on its
// own; notes are not.
type Labels struct {
	Fail    string
	Anomaly string
}

// Default evidence prefixes.
const (
	DefaultFailLabel    = "FAIL: "
	DefaultAnomalyLabel = "ANOMALY: "
)

// DefaultLabels returns the prefixes used when no others are given.
func DefaultLabels() Labels {
	return Labels{Fail: DefaultFailLabel, Anomaly: DefaultAnomalyLabel}
}

// Option configures a Builder.
type Option func(*Builder)

// WithLabels overrides the evidence prefixes — for suites that report in
// another language, or that have prefixes already in use whose wording must
// stay byte-for-byte identical.
func WithLabels(l Labels) Option {
	return func(b *Builder) { b.labels = l }
}

// Builder accumulates the outcome of one rule.
//
// It starts at Pass and only ever degrades: Failf sets Fail permanently,
// Anomalyf raises Pass to Anomaly but never lowers Fail. A rule can therefore
// record everything it finds without ordering its observations carefully.
//
// A Builder is not safe for concurrent use; build one per rule evaluation.
type Builder struct {
	check  Check
	labels Labels
}

// New starts a check for the named rule. invariant is the rule's number in the
// specification, or 0 when the suite does not number its rules.
func New(name string, invariant int, opts ...Option) *Builder {
	b := &Builder{
		check:  Check{Name: name, Invariant: invariant, Status: Pass},
		labels: DefaultLabels(),
	}
	for _, opt := range opts {
		opt(b)
	}

	return b
}

// Failf records a violation and sets the status to Fail permanently.
func (b *Builder) Failf(format string, a ...any) {
	b.check.Status = Fail
	b.check.Evidence = append(b.check.Evidence, b.labels.Fail+fmt.Sprintf(format, a...))
}

// Anomalyf records something that needs a human look. It raises Pass to
// Anomaly but leaves an existing Fail untouched — a rule that already found a
// defect stays failed.
func (b *Builder) Anomalyf(format string, a ...any) {
	if b.check.Status != Fail {
		b.check.Status = Anomaly
	}
	b.check.Evidence = append(b.check.Evidence, b.labels.Anomaly+fmt.Sprintf(format, a...))
}

// Notef records a neutral observation without changing the status. Use it for
// the counts and comparisons that make a passing check auditable.
func (b *Builder) Notef(format string, a ...any) {
	b.check.Evidence = append(b.check.Evidence, fmt.Sprintf(format, a...))
}

// Status reports the outcome so far, so a rule can skip work that only makes
// sense while it is still passing.
func (b *Builder) Status() Status { return b.check.Status }

// Done returns the finished check.
func (b *Builder) Done() Check { return b.check }

// Rule is one named invariant together with the function that evaluates it
// against T. Collecting rules in a slice makes the suite enumerable: it can be
// listed, counted, and — importantly — checked for whether every rule has a
// test of its own.
type Rule[T any] struct {
	Name      string
	Invariant int
	Eval      func(T) Check
}

// Evaluate applies every rule in order and returns the checks. The order is
// preserved because reports are read in it.
func Evaluate[T any](subject T, rules []Rule[T]) []Check {
	checks := make([]Check, 0, len(rules))
	for _, r := range rules {
		checks = append(checks, r.Eval(subject))
	}

	return checks
}

// Verdict is the worst status among the checks: Fail if any failed, else
// Anomaly if any was anomalous, else Pass. An empty slice yields Pass — a
// suite with no rules has found nothing wrong, which is why counting the rules
// is worth a test of its own.
func Verdict(checks []Check) Status {
	worst := Pass
	for _, c := range checks {
		if severity(c.Status) > severity(worst) {
			worst = c.Status
		}
	}

	return worst
}

// severity ranks statuses so a suite verdict can be computed by taking the
// worst one. Fail outranks Anomaly, which outranks Pass.
func severity(s Status) int {
	const (
		rankPass    = 0
		rankAnomaly = 1
		rankFail    = 2
	)

	switch s {
	case Fail:
		return rankFail
	case Anomaly:
		return rankAnomaly
	case Pass:
		return rankPass
	}

	return rankPass
}

// Failed returns the names of the checks that failed, sorted, for a compact
// summary line.
func Failed(checks []Check) []string {
	return namesWithStatus(checks, Fail)
}

// Anomalous returns the names of the checks that reported an anomaly, sorted.
func Anomalous(checks []Check) []string {
	return namesWithStatus(checks, Anomaly)
}

func namesWithStatus(checks []Check, want Status) []string {
	var out []string
	for _, c := range checks {
		if c.Status == want {
			out = append(out, c.Name)
		}
	}
	sort.Strings(out)

	return out
}
