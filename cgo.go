// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#cgo CFLAGS: -std=c2x -I${SRCDIR}/internal/clib/include -I${SRCDIR}/internal/clib/src -I${SRCDIR}/internal/clib/sqlite -include ${SRCDIR}/clib_symbols.h
#cgo linux LDFLAGS: -lm -lpthread
#include "snspd/snspd.h"
#include "clib_digest.h"
*/
import "C"

// Version is the vendored library version declared by its public header.
// internal/clib/VERSION and COMMIT identify the exact upstream snapshot.
const Version = "1.0.0"

const (
	vendoredVersion = C.SNSPD_GO_CLIB_VERSION
	vendoredDigest  = C.SNSPD_GO_CLIB_DIGEST
	headerVersion   = C.SNSPD_VERSION_STRING
)
