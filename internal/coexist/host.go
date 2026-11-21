// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// Package coexist is a test host with an ordinary, independent SQLite engine.
// It intentionally exports normal sqlite3_* C symbols to catch interposition.
package coexist

/*
#cgo CFLAGS: -I${SRCDIR}/../clib/sqlite -DSQLITE_THREADSAFE=1
#cgo linux LDFLAGS: -lm -ldl -lpthread
#include "sqlite3.h"
#include <stdlib.h>
*/
import "C"

import (
	"fmt"
	"unsafe"
)

// Host keeps a real SQLite connection alive while the binding is exercised.
type Host struct{ db *C.sqlite3 }

// OpenHost opens the ordinary SQLite engine with its normal allocator and VFS.
func OpenHost() (*Host, error) {
	name := C.CString(":memory:")
	defer C.free(unsafe.Pointer(name))
	h := new(Host)
	if rc := C.sqlite3_open(name, &h.db); rc != C.SQLITE_OK {
		h.Close()
		return nil, fmt.Errorf("host sqlite open: %d", int(rc))
	}
	return h, nil
}

// Exec runs SQL against the independent engine.
func (h *Host) Exec(sql string) error {
	text := C.CString(sql)
	defer C.free(unsafe.Pointer(text))
	if rc := C.sqlite3_exec(h.db, text, nil, nil, nil); rc != C.SQLITE_OK {
		return fmt.Errorf("host sqlite exec: %d", int(rc))
	}
	return nil
}

// Value reads a single integer, verifying that host state survives Open calls.
func (h *Host) Value() (int, error) {
	text := C.CString("SELECT n FROM retained")
	defer C.free(unsafe.Pointer(text))
	var statement *C.sqlite3_stmt
	if rc := C.sqlite3_prepare_v2(h.db, text, -1, &statement, nil); rc != C.SQLITE_OK {
		return 0, fmt.Errorf("host prepare: %d", int(rc))
	}
	defer C.sqlite3_finalize(statement)
	if rc := C.sqlite3_step(statement); rc != C.SQLITE_ROW {
		return 0, fmt.Errorf("host step: %d", int(rc))
	}
	return int(C.sqlite3_column_int(statement, 0)), nil
}

// Close releases the ordinary engine's connection.
func (h *Host) Close() {
	if h.db != nil {
		C.sqlite3_close(h.db)
		h.db = nil
	}
}
