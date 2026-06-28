// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0

package sloggcp

import (
	"runtime"
	"strconv"
	"strings"
)

// stackTrace captures a stack trace starting from the caller's frame,
// skipping the specified number of frames. Returns a formatted string
// compatible with Cloud Error Reporting's expected stack trace format.
func stackTrace(skip int) string {
	var pcs [32]uintptr
	n := runtime.Callers(skip, pcs[:])
	if n == 0 {
		return ""
	}

	frames := runtime.CallersFrames(pcs[:n])

	var b strings.Builder

	for {
		frame, more := frames.Next()
		if frame.Function == "" {
			break
		}

		b.WriteString(frame.Function)
		b.WriteByte('\n')
		b.WriteString("\t")
		b.WriteString(frame.File)
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(frame.Line))
		b.WriteByte('\n')

		if !more {
			break
		}
	}

	return b.String()
}
