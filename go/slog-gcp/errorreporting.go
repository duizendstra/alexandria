// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"log/slog"
	"os"
)

// ServiceContext identifies the service for Cloud Error Reporting.
type ServiceContext struct {
	Service string
	Version string
}

// ServiceContextFromEnv reads service context from environment variables.
// It checks K_SERVICE/K_REVISION (Cloud Run), falling back to
// CLOUD_RUN_JOB/CLOUD_RUN_EXECUTION (Cloud Run Jobs), and finally
// SERVICE_NAME/HOSTNAME (GKE/Generic).
func ServiceContextFromEnv() ServiceContext {
	svc := os.Getenv("K_SERVICE")
	if svc == "" {
		svc = os.Getenv("CLOUD_RUN_JOB")
	}
	if svc == "" {
		svc = os.Getenv("SERVICE_NAME")
	}

	ver := os.Getenv("K_REVISION")
	if ver == "" {
		ver = os.Getenv("CLOUD_RUN_EXECUTION")
	}
	if ver == "" {
		ver = os.Getenv("HOSTNAME")
	}

	return ServiceContext{
		Service: svc,
		Version: ver,
	}
}

// ErrorAttrs returns slog attributes for Cloud Error Reporting.
// Attach these to error-level log lines so GCP Error Reporting
// automatically picks them up. Includes a stack trace for grouping.
func ErrorAttrs(err error, sc ServiceContext) []slog.Attr {
	attrs := []slog.Attr{
		slog.String("@type", ErrorReportingType),
		slog.Group("serviceContext", //nolint:sloglint // GCP Error Reporting expects camelCase key.
			slog.String("service", sc.Service),
			slog.String("version", sc.Version),
		),
		slog.String("stack_trace", stackTrace(3)), //nolint:mnd // Skip count for caller stack frames.
	}

	if err != nil {
		attrs = append(attrs, slog.String("error", err.Error()))
	}

	return attrs
}

// ErrorAttrsAny returns error reporting attributes as []any for use
// with slog's alternating key-value API (e.g. slog.Error("msg", attrs...)).
func ErrorAttrsAny(err error, sc ServiceContext) []any {
	attrs := []any{
		"@type",
		ErrorReportingType,
		slog.Group("serviceContext", //nolint:sloglint // GCP Error Reporting expects camelCase key.
			slog.String("service", sc.Service),
			slog.String("version", sc.Version),
		),
		"stack_trace", stackTrace(3), //nolint:mnd // Skip count for caller stack frames.
	}

	if err != nil {
		attrs = append(attrs, "error", err.Error())
	}

	return attrs
}
