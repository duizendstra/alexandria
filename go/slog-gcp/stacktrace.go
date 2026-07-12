// Copyright 2026 Jasper Duizendstra. All rights reserved.
// Licensed under the Apache License, Version 2.0.
// SPDX-License-Identifier: Apache-2.0.

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
	var pcs [128]uintptr
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
		b.WriteString("()")
		b.WriteByte('\n')
		b.WriteString("\t")
		b.WriteString(frame.File)
		b.WriteByte(':')
		var lineBuf [20]byte
		b.Write(strconv.AppendInt(lineBuf[:0], int64(frame.Line), 10)) //nolint:mnd // Base-10 is the only valid radix for line numbers.
		b.WriteByte('\n')

		if !more {
			break
		}
	}

	return b.String()
}
