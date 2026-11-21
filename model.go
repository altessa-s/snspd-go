// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

import "sort"

// Drawing is a complete Go-owned snapshot. No Close is required. Independent
// Open calls are safe concurrently; callers synchronize mutations of a model.
type Drawing struct {
	Canvas                 Canvas
	Layers                 []Layer       // Protobuf order, including empty layers.
	Config                 []ConfigEntry // Strict binary key order.
	ExtraTableCount        int
	UnknownLayerFieldCount int // Container and child unknown protobuf fields.
	InvalidTileCount       int // Unexpected signature/IHDR, not decoding failures.
	UnknownFormat          bool
	HadWALHeader           bool // Flags normalized in a private copy; no sidecar was read.
}

// Canvas holds stored pixels and metadata. Has flags distinguish missing numeric
// values from stored zero. Nil strings/bytes mean absent or NULL; non-nil empty
// values mean stored empty. Text preserves raw bytes, including invalid UTF-8.
type Canvas struct {
	Width, Height, PPI                                               float64
	ViewportX, ViewportY, ViewportScale                              float64
	HasWidth, HasHeight, HasPPI                                      bool
	HasViewportX, HasViewportY, HasViewportScale                     bool
	FormatVersion, PaperMode, TemplateName, TemplateInfo, AppVersion *string
	Thumbnail, TemplateData                                          []byte // Undecoded, Go-owned image/raw bytes.
}

// Layer retains its raw protobuf message and sorted tiles. Field3 and Field4
// have unknown semantics; no visibility or compositing order is inferred.
type Layer struct {
	ID                   uint64
	Name                 string // Exact stored bytes.
	Field3, Field4       uint64
	HasField3, HasField4 bool
	UnknownFieldCount    int
	Raw                  []byte
	Tiles                []Tile // Ascending (TX, TY), equivalent to ascending TID: column-major.
}

// Tile holds one undecoded PNG, copied into Go memory. HasExpectedIHDR checks
// only the signature and 128×128 IHDR dimensions, not CRCs, compression or pixels.
// Column-major IDs decode as TX = TID/4096 and TY = TID%4096 (0…4095).
type Tile struct {
	TID             int64
	TX              uint64
	TY              uint32
	PNG             []byte
	HasExpectedIHDR bool
}

// ConfigEntry preserves SQLite's NULL, TEXT and BLOB distinctions. Keys and
// values preserve exact bytes. A stored empty value is non-nil; NULL is nil.
type ConfigEntry struct {
	Key    string
	Value  []byte
	IsNull bool
	IsText bool
}

// ConfigFind returns the entry with an exact byte key, including embedded NULs,
// or nil when absent. Keep Config in binary key order when modifying it.
// The pointer refers to the current Config backing array.
func (d *Drawing) ConfigFind(key string) *ConfigEntry {
	if d == nil {
		return nil
	}
	i := sort.Search(len(d.Config), func(i int) bool { return d.Config[i].Key >= key })
	if i < len(d.Config) && d.Config[i].Key == key {
		return &d.Config[i]
	}
	return nil
}

// Rect is a viewport-relative pixel rectangle. IsNull marks an empty extent.
// Large grid coordinates may round to double precision.
type Rect struct {
	X, Y, Width, Height float64
	IsNull              bool
}

// TileRect returns a tile's stored pixel rectangle at TX*128-ViewportX,
// TY*128-ViewportY. The saved viewport's page anchoring is unconfirmed.
// No scale, half-page offset or clipping is applied. Nil arguments yield null.
func (d *Drawing) TileRect(tile *Tile) Rect {
	if d == nil || tile == nil {
		return Rect{IsNull: true}
	}
	return Rect{X: float64(tile.TX)*128 - d.Canvas.ViewportX,
		Y: float64(tile.TY)*128 - d.Canvas.ViewportY, Width: 128, Height: 128}
}

// PaintedBounds returns the extent of stored tile rectangles, including fully
// transparent tiles. A nil layer includes every layer. Empty extents are null.
// Width/height use integer grid spans before conversion to preserve single-tile
// sizes at large coordinates. No image decoding, scaling or clipping is applied.
// Like TileRect, it subtracts the saved viewport without assuming a page origin.
func (d *Drawing) PaintedBounds(layer *Layer) Rect {
	if d == nil {
		return Rect{IsNull: true}
	}
	layers := d.Layers
	if layer != nil {
		layers = []Layer{*layer}
	}
	var minX, maxX uint64
	var minY, maxY uint32
	found := false
	for _, l := range layers {
		for _, t := range l.Tiles {
			if !found {
				minX, maxX, minY, maxY = t.TX, t.TX, t.TY, t.TY
				found = true
			}
			minX, maxX = min(minX, t.TX), max(maxX, t.TX)
			minY, maxY = min(minY, t.TY), max(maxY, t.TY)
		}
	}
	if !found {
		return Rect{IsNull: true}
	}
	return Rect{X: float64(minX)*128 - d.Canvas.ViewportX, Y: float64(minY)*128 - d.Canvas.ViewportY,
		Width: float64(maxX-minX+1) * 128, Height: float64(uint64(maxY)-uint64(minY)+1) * 128}
}
