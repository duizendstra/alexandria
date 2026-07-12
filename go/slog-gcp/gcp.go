// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

package sloggcp

import (
	"log/slog"
)

// GCPReplaceAttr maps Go slog field names and values to GCP Cloud
// Logging equivalents: msg → message, time → timestamp,
// source → logging.googleapis.com/sourceLocation, and level → severity
// with value mapping (DEBUG/INFO/NOTICE/WARNING/ERROR/CRITICAL/ALERT/EMERGENCY).
func GCPReplaceAttr(_ []string, a slog.Attr) slog.Attr {
	switch a.Key {
	case slog.MessageKey:
		a.Key = "message"

	case slog.TimeKey:
		a.Key = "timestamp"

	case slog.SourceKey:
		a.Key = FieldSourceLocation

		if source, ok := a.Value.Any().(*slog.Source); ok && source != nil {
			a.Value = slog.GroupValue(
				slog.String("file", source.File),
				slog.Int("line", source.Line),
				slog.String("function", source.Function),
			)
		}

	case slog.LevelKey:
		a.Key = "severity"

		if level, ok := a.Value.Any().(slog.Level); ok {
			a.Value = slog.StringValue(gcpSeverity(level))
		}
	}

	return a
}

// gcpSeverity maps a slog.Level to the corresponding GCP Cloud Logging
// severity string.
func gcpSeverity(level slog.Level) string {
	switch {
	case level < slog.LevelInfo:
		return "DEBUG"
	case level < LevelNotice:
		return "INFO"
	case level < slog.LevelWarn:
		return "NOTICE"
	case level < slog.LevelError:
		return "WARNING"
	case level < LevelCritical:
		return "ERROR"
	case level < LevelAlert:
		return "CRITICAL"
	case level < LevelEmergency:
		return "ALERT"
	default:
		return "EMERGENCY"
	}
}

