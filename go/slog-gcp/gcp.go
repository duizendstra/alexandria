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
	if a.Key == slog.MessageKey {
		a.Key = "message"
	}

	if a.Key == slog.TimeKey {
		a.Key = "timestamp"
	}

	if a.Key == slog.SourceKey {
		a.Key = FieldSourceLocation

		if source, ok := a.Value.Any().(*slog.Source); ok && source != nil {
			a.Value = slog.GroupValue(
				slog.String("file", source.File),
				slog.Int("line", source.Line),
				slog.String("function", source.Function),
			)
		}
	}

	if a.Key == slog.LevelKey {
		a.Key = "severity"

		if level, ok := a.Value.Any().(slog.Level); ok {
			switch {
			case level < slog.LevelInfo:
				a.Value = slog.StringValue("DEBUG")
			case level < LevelNotice:
				a.Value = slog.StringValue("INFO")
			case level < slog.LevelWarn:
				a.Value = slog.StringValue("NOTICE")
			case level < slog.LevelError:
				a.Value = slog.StringValue("WARNING")
			case level < LevelCritical:
				a.Value = slog.StringValue("ERROR")
			case level < LevelAlert:
				a.Value = slog.StringValue("CRITICAL")
			case level < LevelEmergency:
				a.Value = slog.StringValue("ALERT")
			default:
				a.Value = slog.StringValue("EMERGENCY")
			}
		}
	}

	return a
}
