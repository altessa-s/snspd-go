// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#include "snspd/snspd.h"
*/
import "C"

// Two directions ensure any drift in the mirrored C enums fails to compile.
var (
	_ [int(StatusOK) - int(C.SPD_OK)]struct{}
	_ [int(C.SPD_OK) - int(StatusOK)]struct{}
	_ [int(StatusInvalidArgument) - int(C.SPD_ERR_INVALID_ARGUMENT)]struct{}
	_ [int(C.SPD_ERR_INVALID_ARGUMENT) - int(StatusInvalidArgument)]struct{}
	_ [int(StatusOutOfMemory) - int(C.SPD_ERR_OUT_OF_MEMORY)]struct{}
	_ [int(C.SPD_ERR_OUT_OF_MEMORY) - int(StatusOutOfMemory)]struct{}
	_ [int(StatusUnsupportedFormat) - int(C.SPD_ERR_UNSUPPORTED_FORMAT)]struct{}
	_ [int(C.SPD_ERR_UNSUPPORTED_FORMAT) - int(StatusUnsupportedFormat)]struct{}
	_ [int(StatusUnexpectedEOF) - int(C.SPD_ERR_UNEXPECTED_EOF)]struct{}
	_ [int(C.SPD_ERR_UNEXPECTED_EOF) - int(StatusUnexpectedEOF)]struct{}
	_ [int(StatusMalformed) - int(C.SPD_ERR_MALFORMED)]struct{}
	_ [int(C.SPD_ERR_MALFORMED) - int(StatusMalformed)]struct{}
	_ [int(StatusLimit) - int(C.SPD_ERR_LIMIT)]struct{}
	_ [int(C.SPD_ERR_LIMIT) - int(StatusLimit)]struct{}
	_ [int(LogDebug) - int(C.SPD_LOG_DEBUG)]struct{}
	_ [int(C.SPD_LOG_DEBUG) - int(LogDebug)]struct{}
	_ [int(LogError) - int(C.SPD_LOG_ERROR)]struct{}
	_ [int(C.SPD_LOG_ERROR) - int(LogError)]struct{}
)
