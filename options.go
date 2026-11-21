// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#include "snspd/snspd.h"
*/
import "C"

// Limits bounds independent resources used by the C reader. Every zero is an
// actual zero ceiling. Memory excludes Go input/output, host allocator overhead,
// stack, and process-lifetime log bindings. Work is metered operations, not time.
// Raising byte limits may also require raising MaxMemoryBytes and MaxWork.
type Limits struct {
	MaxFileSize       uint64 // Input bytes.
	MaxLayers         uint64
	MaxTilesPerLayer  uint64
	MaxTileBytes      uint64
	MaxTotalTileBytes uint64
	MaxConfigEntries  uint64
	MaxConfigBytes    uint64 // Combined key and value bytes.
	MaxMemoryBytes    uint64 // Live C allocation bytes, including SQLite.
	MaxWork           uint64
}

// Options controls format checks and resource ceilings. Start with DefaultOptions
// before changing individual fields. A nil *Options uses the defaults.
type Options struct {
	Limits        Limits
	Strict        bool // Require fmt_ver exactly "2"; unknown fields remain accepted.
	RequirePNG128 bool // Check signature and IHDR dimensions only, not CRCs or pixels.
}

// DefaultOptions returns the vendored reader's defaults.
func DefaultOptions() Options {
	o := C.spd_options_default()
	l := o.limits
	return Options{Limits: Limits{
		MaxFileSize: uint64(l.max_file_size), MaxLayers: uint64(l.max_layers),
		MaxTilesPerLayer: uint64(l.max_tiles_per_layer), MaxTileBytes: uint64(l.max_tile_bytes),
		MaxTotalTileBytes: uint64(l.max_total_tile_bytes), MaxConfigEntries: uint64(l.max_config_entries),
		MaxConfigBytes: uint64(l.max_config_bytes), MaxMemoryBytes: uint64(l.max_memory_bytes), MaxWork: uint64(l.max_work),
	}, Strict: bool(o.strict), RequirePNG128: bool(o.require_png_128)}
}

func cOptions(o *Options) (C.spd_options, error) {
	if o == nil {
		return C.spd_options_default(), nil
	}
	l := o.Limits
	for _, n := range []uint64{l.MaxFileSize, l.MaxLayers, l.MaxTilesPerLayer, l.MaxTileBytes,
		l.MaxTotalTileBytes, l.MaxConfigEntries, l.MaxConfigBytes, l.MaxMemoryBytes} {
		if uint64(C.size_t(n)) != n {
			return C.spd_options{}, &Error{Status: StatusLimit, Message: "limit exceeds platform size_t"}
		}
	}
	return C.spd_options{limits: C.spd_limits{
		max_file_size: C.size_t(l.MaxFileSize), max_layers: C.size_t(l.MaxLayers),
		max_tiles_per_layer: C.size_t(l.MaxTilesPerLayer), max_tile_bytes: C.size_t(l.MaxTileBytes),
		max_total_tile_bytes: C.size_t(l.MaxTotalTileBytes), max_config_entries: C.size_t(l.MaxConfigEntries),
		max_config_bytes: C.size_t(l.MaxConfigBytes), max_memory_bytes: C.size_t(l.MaxMemoryBytes), max_work: C.uint64_t(l.MaxWork),
	}, strict: C.bool(o.Strict), require_png_128: C.bool(o.RequirePNG128)}, nil
}
