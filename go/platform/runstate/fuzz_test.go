package runstate

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func FuzzCheckSubject(f *testing.F) {
	// Seed corpus.
	f.Add("prod-db")
	f.Add("user-12345")
	f.Add("service.worker")
	f.Add("..")
	f.Add(".")
	f.Add("")
	f.Add("foo/bar")
	f.Add("foo\\bar")
	f.Add("foo/../bar")
	f.Add("../../../etc/passwd")

	f.Fuzz(func(t *testing.T, subject string) {
		err := checkSubject(subject)
		if err == nil {
			// If checkSubject passed, verify it does not contain path separators or parent refs.
			if subject == "" || subject == "." || subject == ".." {
				t.Errorf("checkSubject accepted invalid subject %q", subject)
			}
			if strings.Contains(subject, "/") || strings.Contains(subject, "\\") || strings.Contains(subject, "..") {
				t.Errorf("checkSubject allowed separator or .. in %q", subject)
			}
		}
	})
}

func FuzzLeaseValid(f *testing.F) {
	nowUnix := time.Now().Unix()
	f.Add(nowUnix, nowUnix, "subject-a", "subject-a", "fp-123", "fp-123", int64(300))
	f.Add(nowUnix, nowUnix-100, "subject-a", "subject-a", "fp-123", "fp-123", int64(300))
	f.Add(nowUnix, nowUnix+100, "subject-a", "subject-a", "fp-123", "fp-123", int64(300)) // future clock skew.

	f.Fuzz(func(t *testing.T, nowSec, issuedSec int64, leaseSub, reqSub, leaseFP, reqFP string, windowSec int64) {
		now := time.Unix(nowSec, 0)
		issued := time.Unix(issuedSec, 0)
		window := time.Duration(windowSec) * time.Second

		l := Lease{
			Subject:     leaseSub,
			Fingerprint: leaseFP,
			IssuedAt:    issued,
		}

		valid := l.Valid(now, reqSub, reqFP, window)
		if valid {
			if l.Subject != reqSub || l.Fingerprint != reqFP || l.Fingerprint == "" {
				t.Errorf("Valid returned true for mismatched subject or empty/mismatched fingerprint")
			}
			age := now.Sub(issued)
			if age < 0 || age >= window {
				t.Errorf("Valid returned true for expired or future lease with age %v and window %v", age, window)
			}
		}
	})
}

func FuzzLeaseJSON(f *testing.F) {
	f.Add([]byte(`{"subject":"test","fingerprint":"fp","issued_at":"2026-08-21T11:00:00Z"}`))
	f.Add([]byte(`{}`))
	f.Add([]byte(`{"subject":null}`))
	f.Add([]byte(`malformed json`))

	f.Fuzz(func(t *testing.T, data []byte) {
		var l Lease
		_ = json.Unmarshal(data, &l)
	})
}
