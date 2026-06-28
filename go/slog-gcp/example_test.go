// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp_test

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"time"

	sloggcp "github.com/duizendstra/alexandria/go/slog-gcp"
)

func ExampleSetup() {
	// Setup configures the default slog logger for GCP Cloud Logging.
	// On Cloud Run (K_SERVICE set), outputs JSON; locally, outputs text.
	sloggcp.Setup()

	slog.Info("server started", "port", 8080)
	// Output is environment-dependent.
}

func ExampleNewHandler() {
	inner := slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		ReplaceAttr: sloggcp.GCPReplaceAttr,
	})

	// nil resolver disables trace injection; empty projectID auto-detects.
	handler := sloggcp.NewHandler(inner, nil, "my-project", sloggcp.WithEventID(false))
	logger := slog.New(handler)
	logger.Info("hello")
}

func ExampleErrorAttrs() {
	err := fmt.Errorf("connection refused")
	attrs := sloggcp.ErrorAttrs(err)

	// attrs contains @type, serviceContext, and error fields
	// for Cloud Error Reporting integration.
	fmt.Println(len(attrs))
	// Output:
	// 3
}

func ExampleWithTrace() {
	ctx := sloggcp.WithTrace(context.Background(), "my-project")
	// Use ctx with any slog call — trace ID is injected automatically.
	slog.InfoContext(ctx, "job started")
}

func ExampleHTTPRequestAttr() {
	reqAttr := sloggcp.HTTPRequestAttr(sloggcp.HTTPRequest{
		Method:  "GET",
		URL:     "/api/health",
		Status:  200,
		Latency: 42 * time.Millisecond,
	})
	_ = reqAttr
}
