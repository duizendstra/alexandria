package coordination_test

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/duizendstra/alexandria/go/platform/coordination"
)

// TestCoordination_SubjectRejectsAnythingThatEscapesTheStore pins the one
// promise every adapter leans on: a subject becomes part of a path, and a
// name that could leave the store is refused with ErrBadSubject rather than
// composed into a path outside it.
func TestCoordination_SubjectRejectsAnythingThatEscapesTheStore(t *testing.T) {
	bad := []coordination.Subject{
		"",
		".",
		"..",
		"../elsewhere",
		"nested/subject",
		"weird..name",
		coordination.Subject(filepath.Join("sep", "arated")),
	}
	for _, s := range bad {
		if err := s.Validate(); !errors.Is(err, coordination.ErrBadSubject) {
			t.Errorf("Validate(%q) = %v, want ErrBadSubject", string(s), err)
		}
	}

	good := []coordination.Subject{"shared-index", "resource-1", "a.b", "UPPER_case-9"}
	for _, s := range good {
		if err := s.Validate(); err != nil {
			t.Errorf("Validate(%q) = %v, want nil", string(s), err)
		}
	}
}

// TestCoordination_SubjectErrorNamesTheOffendingSubject pins that the
// refusal is diagnosable: the rejected name is in the message, so an
// operator reading a log does not have to guess which of several subjects
// was wrong.
func TestCoordination_SubjectErrorNamesTheOffendingSubject(t *testing.T) {
	err := coordination.Subject("nested/subject").Validate()
	if err == nil || !strings.Contains(err.Error(), "nested/subject") {
		t.Fatalf("Validate error = %v, want it to name the subject", err)
	}
}

// TestCoordination_SelfDescribesTheCallingProcess pins what an adapter
// writes into a holder record: this process, this host, now, and the stated
// purpose — with Since in UTC so an age comparison against a record written
// by another process in another zone is meaningful.
func TestCoordination_SelfDescribesTheCallingProcess(t *testing.T) {
	before := time.Now().UTC()
	h := coordination.Self("updating the shared index")
	after := time.Now().UTC()

	if h.PID != os.Getpid() {
		t.Errorf("PID = %d, want %d", h.PID, os.Getpid())
	}
	if h.Host == "" {
		t.Error("Host must never be empty — an unnamed host is not diagnosable")
	}
	if h.Purpose != "updating the shared index" {
		t.Errorf("Purpose = %q, want the stated purpose", h.Purpose)
	}
	if h.Since.Before(before) || h.Since.After(after) {
		t.Errorf("Since = %v, want it between %v and %v", h.Since, before, after)
	}
	if h.Since.Location() != time.UTC {
		t.Errorf("Since location = %v, want UTC", h.Since.Location())
	}
}

// TestCoordination_HolderAgeIsZeroWithoutSince pins the safe reading of an
// incomplete record: with no Since there is no meaningful age, so an
// adapter comparing against a reclaim age can never mistake an unreadable
// claim for an abandoned one.
func TestCoordination_HolderAgeIsZeroWithoutSince(t *testing.T) {
	if got := (coordination.Holder{}).Age(time.Now()); got != 0 {
		t.Fatalf("Age of a holder without Since = %v, want 0", got)
	}

	since := time.Now().UTC().Add(-90 * time.Minute)
	if got := (coordination.Holder{Since: since}).Age(since.Add(time.Hour)); got != time.Hour {
		t.Fatalf("Age = %v, want 1h", got)
	}
}

// TestCoordination_HolderRoundTripsAsJSON pins the record's wire shape: the
// four documented fields, under their documented names, surviving a round
// trip — an adapter writes it, an operator or another process reads it.
func TestCoordination_HolderRoundTripsAsJSON(t *testing.T) {
	want := coordination.Holder{
		PID:     4242,
		Host:    "somewhere",
		Since:   time.Now().UTC().Truncate(time.Second),
		Purpose: "updating the shared index",
	}

	b, err := json.Marshal(want)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var keys map[string]any
	if err := json.Unmarshal(b, &keys); err != nil {
		t.Fatalf("unmarshal into map: %v", err)
	}
	for _, k := range []string{"pid", "host", "since", "purpose"} {
		if _, ok := keys[k]; !ok {
			t.Errorf("holder JSON is missing %q: %s", k, b)
		}
	}

	var got coordination.Holder
	if err := json.Unmarshal(b, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got.PID != want.PID || got.Host != want.Host || got.Purpose != want.Purpose || !got.Since.Equal(want.Since) {
		t.Fatalf("round trip = %+v, want %+v", got, want)
	}
}

// TestCoordination_ErrorsAreDistinctSentinels pins that the three published
// errors can actually be told apart with errors.Is — a caller classifying a
// refusal must never have to match text.
func TestCoordination_ErrorsAreDistinctSentinels(t *testing.T) {
	all := []error{coordination.ErrBadSubject, coordination.ErrLocked, coordination.ErrStaleLock}
	for i, a := range all {
		for j, b := range all {
			if i != j && errors.Is(a, b) {
				t.Errorf("sentinel %v must not match %v", a, b)
			}
		}
	}
}

// TestCoordination_HolderStringIsOneOperatorReadableLine pins the rendering
// an operator meets in a give-up error or a log line: pid, host, the UTC
// instant and the purpose — and an unstated purpose is said out loud rather
// than rendered as empty parentheses.
func TestCoordination_HolderStringIsOneOperatorReadableLine(t *testing.T) {
	since := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	h := coordination.Holder{PID: 4242, Host: "somewhere", Since: since, Purpose: "rebuilding the catalogue"}

	want := "pid 4242 on somewhere since 2026-01-02T03:04:05Z (rebuilding the catalogue)"
	if got := h.String(); got != want {
		t.Fatalf("String() = %q, want %q", got, want)
	}

	if got := (coordination.Holder{PID: 1, Host: "h", Since: since}).String(); !strings.Contains(got, "unstated purpose") {
		t.Fatalf("String() without a purpose = %q, want it to say the purpose is unstated", got)
	}
}
