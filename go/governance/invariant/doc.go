// Package invariant builds named, evidence-carrying verification results.
//
// BC:      Governance
// Concern: Did this rule hold, and what is the evidence either way?
//
// A verification suite that answers only "pass" or "fail" is hard to act on:
// when it fails you still have to reconstruct why, and there is no way to say
// "this looks wrong but I am not certain". This package models the shape that
// mechanical verification actually needs:
//
//   - every rule has a NAME and a number, so a failure names one rule instead
//     of one long boolean expression;
//   - every rule carries EVIDENCE — the observations it made — so the result
//     is readable without rerunning it;
//   - a rule can report an ANOMALY as well as a failure. An anomaly means
//     "this needs a human look", not "this is broken". Collapsing the two
//     forces a choice between crying wolf and staying silent.
//
// A Builder starts optimistic (Pass) and degrades: an anomaly never overrides
// a failure, and neither is ever reset. Evidence accumulates in the order it
// was recorded.
//
// Labels are configurable because the evidence lines are often read by people
// in their own language, and because existing suites have prefixes already in
// use that must not change. Zero dependencies outside the standard library.
package invariant
