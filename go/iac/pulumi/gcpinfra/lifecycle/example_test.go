// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package lifecycle_test

import (
	"fmt"

	"github.com/duizendstra/alexandria/go/iac/pulumi/gcpinfra/lifecycle"
)

// A stack that keeps its data passes no option at all: every building block
// protects its data-bearing resources by default, and a destroy has to be
// unlocked deliberately with `pulumi state unprotect`.
func ExampleIsEphemeral() {
	fmt.Println(lifecycle.IsEphemeral())
	// Output: false
}

// A test or preview stack threads lifecycle.Ephemeral through the same
// building blocks it would use in production, which is the point: the stack is
// exercised as written, and only its disposability differs.
//
//	_, err := datasets.Apply(ctx, projectID, cfg, deps, opts...)
func ExampleEphemeral() {
	opts := []lifecycle.Option{lifecycle.Ephemeral()}

	fmt.Println(lifecycle.IsEphemeral(opts...))
	// Output: true
}
