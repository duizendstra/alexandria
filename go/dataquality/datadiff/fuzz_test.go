// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package datadiff_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/duizendstra/alexandria/go/dataquality/datadiff"
)

// FuzzParseTarget pins the parsing contract for untrusted target strings:
// success iff the input contains at least two dots (SplitN keeps any further
// dots inside the table name), and a successful parse must reassemble to the
// exact input.
func FuzzParseTarget(f *testing.F) {
	f.Add("proj.dataset.table")
	f.Add("proj.dataset.table.with.dots")
	f.Add("..")
	f.Add("a.b")
	f.Add("")
	f.Add("proj..table")

	f.Fuzz(func(t *testing.T, s string) {
		target, err := datadiff.ParseTarget(s)

		if strings.Count(s, ".") < 2 {
			if !errors.Is(err, datadiff.ErrInvalidTarget) {
				t.Fatalf("ParseTarget(%q) error = %v, want ErrInvalidTarget", s, err)
			}

			return
		}

		if err != nil {
			t.Fatalf("ParseTarget(%q): %v", s, err)
		}

		joined := target.Project + "." + target.Dataset + "." + target.Table
		if joined != s {
			t.Fatalf("ParseTarget(%q) round-trip = %q", s, joined)
		}
	})
}
