package runstate

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// DefaultLeasePrefix and leaseSuffix compose the lease file name.
const (
	DefaultLeasePrefix = "lease."
	leaseSuffix        = ".json"
)

// Lease is proof that a check was passed for one subject, against one
// fingerprint of the thing that was checked.
//
// The fingerprint is what makes a lease safe to keep on disk: a commit, a
// content hash, a configuration digest. A lease issued against one fingerprint
// says nothing about another, so a rebuild or a config change invalidates it
// without anyone having to remember to clean up.
type Lease struct {
	Subject     string    `json:"subject"`
	Fingerprint string    `json:"fingerprint"`
	IssuedAt    time.Time `json:"issued_at"`
}

// Valid reports whether this lease may carry an action now: same subject, same
// fingerprint, and younger than window.
//
// A lease dated in the future is not valid. Clock skew is the one case where a
// stale lease could otherwise outlive its window, and a run that refuses is
// cheaper than one that should not have started.
func (l Lease) Valid(now time.Time, subject, fingerprint string, window time.Duration) bool {
	if l.Subject != subject || l.Fingerprint == "" || l.Fingerprint != fingerprint {
		return false
	}

	age := now.Sub(l.IssuedAt)

	return age >= 0 && age < window
}

// LeaseStore keeps one lease per subject as a JSON file.
type LeaseStore struct {
	// Dir holds the lease files. It is created if missing.
	Dir string

	// Prefix goes in front of the subject in the file name.
	// Empty means DefaultLeasePrefix.
	Prefix string
}

// Load reads the lease of a subject. A missing lease is not an error, and
// neither is an unreadable one: both mean "no lease", which is the answer that
// makes the caller do the work again rather than skip it.
func (s *LeaseStore) Load(subject string) (Lease, bool, error) {
	path, err := s.Path(subject)
	if err != nil {
		return Lease{}, false, err
	}

	b, err := os.ReadFile(path) //nolint:gosec // the caller owns this directory
	if errors.Is(err, os.ErrNotExist) {
		return Lease{}, false, nil
	}

	if err != nil {
		return Lease{}, false, fmt.Errorf("read lease %s: %w", path, err)
	}

	var lease Lease
	if err := json.Unmarshal(b, &lease); err != nil {
		// An unreadable lease is no lease: the answer that makes the caller
		// redo the check is safer than one that makes it stop.
		//nolint:nilerr // deliberate: a broken lease counts as absent
		return Lease{}, false, nil
	}

	return lease, true, nil
}

// Save writes the lease atomically: a temporary file in the same directory,
// then a rename. A reader never sees a half-written lease.
func (s *LeaseStore) Save(lease Lease) error {
	path, err := s.Path(lease.Subject)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(s.Dir, dirPerm); err != nil {
		return fmt.Errorf("lease directory %s: %w", s.Dir, err)
	}

	b, err := json.MarshalIndent(lease, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal lease: %w", err)
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), ".lease-*")
	if err != nil {
		return fmt.Errorf("temporary lease: %w", err)
	}

	name := tmp.Name()
	defer func() { _ = os.Remove(name) }()

	if _, err := tmp.Write(append(b, '\n')); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("write lease: %w", err)
	}

	if err := tmp.Chmod(filePerm); err != nil {
		_ = tmp.Close()

		return fmt.Errorf("lease permissions: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close lease: %w", err)
	}

	if err := os.Rename(name, path); err != nil {
		return fmt.Errorf("place lease: %w", err)
	}

	return nil
}

// Consume removes the lease of a subject. A missing lease is not an error:
// consuming after a success and revoking after a failure are both idempotent.
func (s *LeaseStore) Consume(subject string) error {
	path, err := s.Path(subject)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("remove lease %s: %w", path, err)
	}

	return nil
}

// Path is where the lease of a subject lives.
func (s *LeaseStore) Path(subject string) (string, error) {
	return pathFor(s.Dir, s.prefix(), subject, leaseSuffix)
}

func (s *LeaseStore) prefix() string {
	if s.Prefix == "" {
		return DefaultLeasePrefix
	}

	return s.Prefix
}
