// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"log/slog"
	"os"
)

// ErrorAttrs returns slog attributes for Cloud Error Reporting.
// Attach these to error-level log lines so GCP Error Reporting
// automatically picks them up. Includes a stack trace for grouping.
func ErrorAttrs(err error) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("@type", ErrorReportingType),
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
		slog.String("stack_trace", stackTrace(3)),
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}

// ErrorAttrsAny returns error reporting attributes as []any for use
// with slog's alternating key-value API (e.g. slog.Error("msg", attrs...)).
func ErrorAttrsAny(err error) []any {
	attrs := []any{
		"@type",
		ErrorReportingType,
		slog.Group("serviceContext",
			slog.String("service", os.Getenv("K_SERVICE")),
			slog.String("version", os.Getenv("K_REVISION")),
		),
		"stack_trace", stackTrace(3),
	}

	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	return attrs
}
