// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#include "snspd/snspd.h"
*/
import "C"

import "strconv"

// Status identifies the reason a call failed.
type Status int

// Status values agree with the C library.
const (
	StatusOK Status = iota
	StatusInvalidArgument
	StatusOutOfMemory
	StatusUnsupportedFormat
	StatusUnexpectedEOF
	StatusMalformed
	StatusLimit
)

// String returns the library's English description or status(number) for unknown values.
func (s Status) String() string {
	if s < StatusOK || s > StatusLimit {
		return "status(" + strconv.Itoa(int(s)) + ")"
	}
	return C.GoString(C.spd_status_string(C.spd_status(s)))
}

// Error records a failure, its SQLite result code, and a content-free diagnostic.
type Error struct {
	// Status identifies the failure category used by errors.Is.
	Status Status
	// Value is the SQLite result code for engine failures; zero otherwise.
	Value int64
	// Message is a content-free diagnostic. C reader failures currently use
	// the status phrase verbatim; Go conversion failures may add detail.
	Message string
}

// Sentinels match errors by status through errors.Is.
var (
	ErrInvalidArgument   = &Error{Status: StatusInvalidArgument}
	ErrOutOfMemory       = &Error{Status: StatusOutOfMemory}
	ErrUnsupportedFormat = &Error{Status: StatusUnsupportedFormat}
	ErrUnexpectedEOF     = &Error{Status: StatusUnexpectedEOF}
	ErrMalformed         = &Error{Status: StatusMalformed}
	ErrLimit             = &Error{Status: StatusLimit}
)

// Error spells the status followed by its optional diagnostic.
func (e *Error) Error() string {
	if e == nil {
		return "snspd: <nil>"
	}
	text := "snspd: " + e.Status.String()
	if e.Message != "" && e.Message != e.Status.String() {
		text += ": " + e.Message
	}
	return text
}

// Is compares statuses, so callers can distinguish limits from malformed files.
func (e *Error) Is(target error) bool {
	t, ok := target.(*Error)
	return ok && t != nil && e != nil && t.Status == e.Status
}
func errorOf(status C.spd_status, e *C.spd_error) error {
	if status == C.SPD_OK {
		return nil
	}
	return &Error{Status: Status(status), Value: int64(e.value), Message: C.GoString(&e.message[0])}
}
