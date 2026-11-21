// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

// Package snspd reads Supernote Atelier .spd drawings through a vendored copy
// of snspd-lib. It exposes canvas metadata, protobuf-ordered layers, sorted
// undecoded PNG tiles, thumbnail bytes, raw config and viewport-relative bounds.
//
// Open copies the complete model into Go memory and releases every C allocation
// before returning. No Close is needed. Strings preserve exact stored bytes,
// including embedded NULs and invalid UTF-8. Optional strings use *string so nil
// differs from stored empty. Config preserves SQLite NULL/TEXT/BLOB distinctions.
//
// Pass nil options for defaults or modify DefaultOptions. Zero ceilings are
// literal zero; they never select defaults. C memory and work ceilings do not
// bound Go output allocations or wall-clock time. Errors support errors.Is with
// the Err* sentinels and errors.As to *Error for status and SQLite result code.
//
// Tiles sort column-major by TID = TX*4096+TY, with TX uint64 and TY uint32.
// Drawing geometry subtracts the saved viewport from stored tile coordinates;
// its page anchoring is unconfirmed. Bounds include transparent tiles, without
// decoding, clipping, a half-page offset or applying viewport scale. Layer flags and
// paper modes retain raw values; their semantics are not established. The PNG
// policy checks only signature/IHDR dimensions, not decodability or pixels.
//
// Independent Open calls are safe concurrently; parsing and conversion are
// serialized before entering C. Queued calls hold no C copy or cgo thread.
// Logging runs after C storage is freed and the serialization gate is released.
// Model mutation requires caller synchronization. SetLogger installs
// an optional process-wide, concurrency-safe diagnostic bridge. After fork in a
// multithreaded process, the child must exec before calling the C library.
//
// Builds need cgo, Go 1.24+ and a C23 compiler (clang 19+ or GCC 13+). SQLite is
// vendored, static inside the parser, with no system SQLite dependency, file I/O
// or extensions. Every external snspd symbol has a package-specific prefix.
// Supply a complete snapshot: WAL sidecars are neither read nor detected.
package snspd
