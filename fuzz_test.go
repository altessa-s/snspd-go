// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"errors"
	"reflect"
	"testing"

	"github.com/altessa-s/snspd-go"
)

func FuzzOpen(f *testing.F) {
	// Larger budgets belong to normal tests. Bound hostile mutation costs here.
	f.Fuzz(func(t *testing.T, data []byte) {
		if len(data) > 1<<20 {
			t.Skip()
		}
		o := snspd.DefaultOptions()
		o.Limits.MaxFileSize = 1 << 20
		o.Limits.MaxMemoryBytes = 16 << 20
		o.Limits.MaxWork = 1 << 20
		o.Limits.MaxLayers = 64
		o.Limits.MaxTilesPerLayer = 4096
		d, err := snspd.Open(data, &o)
		again, other := snspd.Open(data, &o)
		if err != nil {
			var detail *snspd.Error
			if d != nil || !errors.As(err, &detail) || !errors.Is(other, &snspd.Error{Status: detail.Status}) {
				t.Fatalf("unstable failure: %v / %v", err, other)
			}
			return
		}
		if other != nil || !reflect.DeepEqual(d, again) {
			t.Fatal("nondeterministic model")
		}
		for i, e := range d.Config {
			if i > 0 && d.Config[i-1].Key >= e.Key || d.ConfigFind(e.Key) == nil {
				t.Fatal("config ordering/lookup")
			}
		}
		for _, l := range d.Layers {
			for i, tile := range l.Tiles {
				if tile.TID < 0 || tile.TX != uint64(tile.TID/4096) || tile.TY != uint32(tile.TID%4096) || i > 0 && l.Tiles[i-1].TID >= tile.TID {
					t.Fatal("tile invariant")
				}
				r := d.TileRect(&tile)
				if r.IsNull || r.Width != 128 || r.Height != 128 {
					t.Fatal("tile geometry")
				}
			}
			if d.PaintedBounds(&l).IsNull != (len(l.Tiles) == 0) {
				t.Fatal("layer bounds")
			}
		}
	})
}
