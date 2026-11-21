// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// The C side of SetLogger: a sink with the signature the library calls, which
// hands each line to the exported Go function. Kept apart from log.go because
// a Go file that exports to C may only declare, not define, in its preamble.

#include "_cgo_export.h"
#include "snspd/log.h"

static void spdgo_log_sink(spd_log_level level, const char *message, void *context) {
    (void)context;
    spdgoLog((int)level, (char *)message);
}

void spdgo_log_install(void) {
    spd_log_install(spdgo_log_sink, nullptr);
}
