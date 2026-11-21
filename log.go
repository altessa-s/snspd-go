// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#include "snspd/log.h"

void spdgo_log_install(void);
*/
import "C"

import (
	"strconv"
	"sync"
	"sync/atomic"
)

// LogLevel distinguishes retained unfamiliar details from a failed open.
type LogLevel int

// The log levels.
const (
	LogDebug LogLevel = iota
	LogError
)

// String names the level and retains unknown numbers in its fallback.
func (l LogLevel) String() string {
	if l == LogError {
		return "error"
	}
	if l == LogDebug {
		return "debug"
	}
	return "log_level(" + strconv.Itoa(int(l)) + ")"
}

// logHandler is what the C sink calls through. Read on whichever thread is
// opening and swapped by SetLogger, hence atomic; nil drops every line.
var logHandler atomic.Pointer[func(LogLevel, string)]

// logInstall installs the C sink once per process. The C library keeps every
// distinct sink/context pair it is given. Keep one C binding for the lifetime
// of the process; the Go handler decides whether to deliver a line. If the first
// installation fails to allocate, logging stays disabled for the process: the
// C API reports no installation result and sync.Once does not retry.
var logInstall sync.Once

// SetLogger names the function the reader's diagnostics go to, for the whole
// process; nil goes back to silence, which is the default. It may be called
// at any time, from any goroutine.
//
// Each handler runs on a separate goroutine, synchronously with the open that
// produced the line, after C storage is freed and the Open gate is released.
// Concurrent opens may call it concurrently, so it must
// synchronize shared state. A handler that panics or exits with runtime.Goexit
// loses only that line; the open continues normally. The open waits for the
// handler to return, so a handler must not wait for that open. A line already
// in flight may still be delivered after SetLogger(nil) returns.
//
// Once a logger has been set, the C sink stays installed even while the Go
// handler is nil. Diagnostics are fixed, content-free descriptions.
//
// Messages never contain drawing content, paths or names. Reentrant Open calls
// are permitted; a handler must prevent recursive logging (for example, by
// silencing before reentry).
func SetLogger(handler func(level LogLevel, message string)) {
	if handler == nil {
		logHandler.Store(nil)
		return
	}
	logHandler.Store(&handler)
	logInstall.Do(func() {
		C.spdgo_log_install()
	})
}

//export spdgoLog
func spdgoLog(level C.int, message *C.char) {
	handler := logHandler.Load()
	if handler == nil {
		return
	}
	// spd_open is the only diagnostic producer and runs under openMu. Copy
	// both the line and its selected handler before C returns, but invoke user
	// code only after conversion/free and unlock so Open can be reentered.
	pendingLogs = append(pendingLogs, logLine{*handler, LogLevel(level), C.GoString(message)})
}

type logLine struct {
	handler func(LogLevel, string)
	level   LogLevel
	message string
}

var pendingLogs []logLine // guarded by openMu

func (line logLine) deliver() {
	done := make(chan struct{})
	go func() {
		defer close(done)
		defer func() { _ = recover() }()
		line.handler(line.level, line.message)
	}()
	<-done
}
