// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"os"
	"testing"

	"github.com/altessa-s/snspd-go"
)

// Opt-in only: no private path or payload is part of the repository/corpus.
func TestPrivateSampleStructure(t *testing.T) {
	path := os.Getenv("SNSPD_SMOKE_FILE")
	if path == "" {
		t.Skip("set SNSPD_SMOKE_FILE for a read-only structure smoke")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	opts := snspd.DefaultOptions()
	opts.Strict = true
	opts.RequirePNG128 = true
	d, err := snspd.Open(data, &opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(d.Layers) != 2 || d.Layers[0].ID != 1 || d.Layers[1].ID != 9999 || len(d.Layers[1].Tiles) != 0 {
		t.Fatal("unexpected sample layer structure")
	}
	t.Logf("input=%d bytes canvas=%gx%g ppi=%g viewport=(%g,%g) scale=%g layers=%d config=%d extra_tables=%d unknown_layer_fields=%d invalid_tiles=%d unknown_format=%v wal_header=%v thumbnail=%d bytes template_data=%d bytes bounds=%+v",
		len(data), d.Canvas.Width, d.Canvas.Height, d.Canvas.PPI, d.Canvas.ViewportX, d.Canvas.ViewportY, d.Canvas.ViewportScale, len(d.Layers), len(d.Config), d.ExtraTableCount, d.UnknownLayerFieldCount, d.InvalidTileCount, d.UnknownFormat, d.HadWALHeader, len(d.Canvas.Thumbnail), len(d.Canvas.TemplateData), d.PaintedBounds(nil))
	for i, l := range d.Layers {
		total := 0
		var minX, maxX uint64
		var minY, maxY uint32
		for j, tile := range l.Tiles {
			total += len(tile.PNG)
			if j == 0 {
				minX, maxX, minY, maxY = tile.TX, tile.TX, tile.TY, tile.TY
			}
			minX, maxX = min(minX, tile.TX), max(maxX, tile.TX)
			minY, maxY = min(minY, tile.TY), max(maxY, tile.TY)
		}
		t.Logf("layer_index=%d id=%d tiles=%d encoded_tile_bytes=%d raw_message_bytes=%d unknown_fields=%d has_field3=%v has_field4=%v bounds=%+v", i, l.ID, len(l.Tiles), total, len(l.Raw), l.UnknownFieldCount, l.HasField3, l.HasField4, d.PaintedBounds(&l))
		if len(l.Tiles) != 0 {
			t.Logf("layer_index=%d grid_tx=%d..%d grid_ty=%d..%d", i, minX, maxX, minY, maxY)
		}
		if l.ID == 1 {
			if len(l.Tiles) != 143 || minX != 1946 || maxX != 1958 || minY != 1945 || maxY != 1962 {
				t.Fatalf("unexpected sample grid: tiles=%d tx=%d..%d ty=%d..%d", len(l.Tiles), minX, maxX, minY, maxY)
			}
			want := snspd.Rect{X: -896, Y: -1024, Width: 1664, Height: 2304}
			if got := d.PaintedBounds(&l); got != want {
				t.Fatalf("sample bounds=%+v, want %+v", got, want)
			}
		}
	}
}
