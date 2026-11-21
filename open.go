// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

/*
#include "snspd/snspd.h"
*/
import "C"

import (
	"math"
	"sync"
	"unsafe"
)

// Open reads a complete self-contained SQLite snapshot and copies all model
// data, strings, PNGs, thumbnail and config into Go memory before freeing C
// storage. The input must remain unchanged until Open returns. No connection or
// pointer to the input survives. opts == nil selects DefaultOptions.
// Parsing and conversion are serialized; waiting calls park in Go before any
// C input allocation. Logging runs after C storage and the gate are released.
func Open(data []byte, opts *Options) (*Drawing, error) {
	options, err := cOptions(opts)
	if err != nil {
		return nil, err
	}
	var out *Drawing
	var logs []logLine
	func() {
		openMu.Lock()
		defer openMu.Unlock()
		defer func() {
			logs, pendingLogs = pendingLogs, nil
		}()
		out, err = openDrawing(data, &options)
	}()
	for _, line := range logs {
		line.deliver()
	}
	return out, err
}

// openMu bounds live C input/model storage and parks waiters outside cgo.
// It also protects pendingLogs, written by the synchronous C log sink.
var openMu sync.Mutex

func openDrawing(data []byte, options *C.spd_options) (*Drawing, error) {
	var drawing *C.spd_drawing
	var detail C.spd_error
	status := C.spd_open(bytesPointer(data), C.size_t(len(data)), options, &drawing, &detail)
	if err := errorOf(status, &detail); err != nil {
		return nil, err
	}
	defer C.spd_drawing_free(drawing)
	var c converter
	out := c.drawing(drawing)
	if c.err != nil {
		return nil, c.err
	}
	return out, nil
}

// Empty input reaches the format checker as a valid, zero-length buffer.
var emptyInput [1]byte

func bytesPointer(data []byte) unsafe.Pointer {
	if len(data) == 0 {
		return unsafe.Pointer(&emptyInput[0])
	}
	return unsafe.Pointer(&data[0])
}

type converter struct{ err error }

func (c *converter) length(n C.size_t, what string) int {
	if c.err != nil {
		return 0
	}
	if uint64(n) > uint64(math.MaxInt) {
		c.err = &Error{Status: StatusOutOfMemory, Message: what + " too large for a Go int"}
		return 0
	}
	return int(n)
}

// Views are used only during conversion, while the C drawing is still alive.
func view[T any](c *converter, p *T, count C.size_t) []T {
	n := c.length(count, "array length")
	if c.err != nil || n == 0 {
		return nil
	}
	if p == nil || uintptr(n) > uintptr(math.MaxInt)/unsafe.Sizeof(*p) {
		c.err = &Error{Status: StatusMalformed, Message: "invalid model storage"}
		return nil
	}
	return unsafe.Slice(p, n)
}
func (c *converter) bytes(v C.spd_bytes) []byte {
	data := view(c, (*byte)(unsafe.Pointer(v.data)), v.size)
	if v.data == nil || c.err != nil {
		return nil
	}
	out := make([]byte, len(data))
	copy(out, data)
	return out
}
func (c *converter) text(v C.spd_string) *string {
	data := view(c, (*byte)(unsafe.Pointer(v.data)), v.length)
	if v.data == nil || c.err != nil {
		return nil
	}
	s := string(data)
	return &s
}
func (c *converter) drawing(v *C.spd_drawing) *Drawing {
	if v == nil {
		c.err = &Error{Status: StatusMalformed, Message: "missing drawing"}
		return nil
	}
	a := v.canvas
	d := &Drawing{Canvas: Canvas{
		Width: float64(a.width), Height: float64(a.height), PPI: float64(a.ppi),
		ViewportX: float64(a.viewport_x), ViewportY: float64(a.viewport_y), ViewportScale: float64(a.viewport_scale),
		HasWidth: bool(a.has_width), HasHeight: bool(a.has_height), HasPPI: bool(a.has_ppi),
		HasViewportX: bool(a.has_viewport_x), HasViewportY: bool(a.has_viewport_y), HasViewportScale: bool(a.has_viewport_scale),
		FormatVersion: c.text(a.fmt_ver), PaperMode: c.text(a.paper_mode), TemplateName: c.text(a.template_name),
		TemplateInfo: c.text(a.template_info), AppVersion: c.text(a.app_version), Thumbnail: c.bytes(a.thumbnail), TemplateData: c.bytes(a.template_data),
	}, ExtraTableCount: c.length(v.extra_table_count, "extra table count"),
		UnknownLayerFieldCount: c.length(v.unknown_layer_field_count, "unknown field count"),
		InvalidTileCount:       c.length(v.invalid_tile_count, "invalid tile count"),
		UnknownFormat:          bool(v.unknown_format), HadWALHeader: bool(v.had_wal_header)}
	layers := view(c, v.layers, v.layer_count)
	d.Layers = make([]Layer, len(layers))
	for i, l := range layers {
		out := &d.Layers[i]
		*out = Layer{ID: uint64(l.id), Field3: uint64(l.field3), Field4: uint64(l.field4),
			HasField3: bool(l.has_field3), HasField4: bool(l.has_field4), UnknownFieldCount: c.length(l.unknown_field_count, "unknown field count"), Raw: c.bytes(l.raw)}
		if s := c.text(l.name); s != nil {
			out.Name = *s
		}
		tiles := view(c, l.tiles, l.tile_count)
		out.Tiles = make([]Tile, len(tiles))
		for j, t := range tiles {
			out.Tiles[j] = Tile{TID: int64(t.tid), TX: uint64(t.tx), TY: uint32(t.ty), PNG: c.bytes(t.png), HasExpectedIHDR: bool(t.has_expected_ihdr)}
		}
	}
	entries := view(c, v.config, v.config_count)
	d.Config = make([]ConfigEntry, len(entries))
	for i, e := range entries {
		out := &d.Config[i]
		*out = ConfigEntry{IsNull: bool(e.is_null), IsText: bool(e.is_text)}
		if !out.IsNull {
			out.Value = c.bytes(e.value)
		}
		if s := c.text(e.key); s != nil {
			out.Key = *s
		}
	}
	return d
}
