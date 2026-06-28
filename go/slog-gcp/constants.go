// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import "log/slog"

// GCP Cloud Logging field keys.
const (
	// FieldTrace is the Cloud Logging trace field key.
	FieldTrace = "logging.googleapis.com/trace"

	// FieldSpanID is the Cloud Logging span ID field key.
	FieldSpanID = "logging.googleapis.com/spanId"

	// FieldTraceSampled is the Cloud Logging trace sampled field key.
	FieldTraceSampled = "logging.googleapis.com/trace_sampled"

	// FieldSourceLocation is the Cloud Logging source location field key.
	FieldSourceLocation = "logging.googleapis.com/sourceLocation"
)

// GCP severity levels beyond slog's built-in levels.
// Use these with slog to emit Cloud Logging severities not covered
// by the standard slog.LevelDebug/Info/Warn/Error constants.
const (
	// LevelNotice maps to Cloud Logging NOTICE severity.
	LevelNotice = slog.LevelInfo + 2 //nolint:mnd // GCP NOTICE threshold.

	// LevelCritical maps to Cloud Logging CRITICAL severity.
	LevelCritical = slog.LevelError + 4 //nolint:mnd // GCP CRITICAL threshold.

	// LevelAlert maps to Cloud Logging ALERT severity.
	LevelAlert = slog.LevelError + 8 //nolint:mnd // GCP ALERT threshold.

	// LevelEmergency maps to Cloud Logging EMERGENCY severity.
	LevelEmergency = slog.LevelError + 12 //nolint:mnd // GCP EMERGENCY threshold.
)

// ErrorReportingType is the protobuf type URL for Cloud Error Reporting events.
const ErrorReportingType = "type.googleapis.com/google.devtools.clouderrorreporting.v1beta1.ReportedErrorEvent"
