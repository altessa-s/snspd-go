// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"errors"
	"math"
	"testing"

	"github.com/altessa-s/snspd-go"
)

func TestDefaultOptionsAndLimits(t *testing.T) {
	want := snspd.Limits{MaxFileSize: 256 << 20, MaxLayers: 256, MaxTilesPerLayer: 262144, MaxTileBytes: 4 << 20, MaxTotalTileBytes: 256 << 20, MaxConfigEntries: 4096, MaxConfigBytes: 16 << 20, MaxMemoryBytes: 768 << 20, MaxWork: 1 << 30}
	defaults := snspd.DefaultOptions()
	if defaults.Limits != want || defaults.Strict || defaults.RequirePNG128 {
		t.Fatalf("defaults: %+v", defaults)
	}
	data := read(t, "valid")
	for _, tc := range []struct {
		name string
		set  func(*snspd.Limits, uint64)
	}{
		{"file", func(l *snspd.Limits, n uint64) { l.MaxFileSize = n }},
		{"layers", func(l *snspd.Limits, n uint64) { l.MaxLayers = n }},
		{"tiles", func(l *snspd.Limits, n uint64) { l.MaxTilesPerLayer = n }},
		{"tile bytes", func(l *snspd.Limits, n uint64) { l.MaxTileBytes = n }},
		{"total tile bytes", func(l *snspd.Limits, n uint64) { l.MaxTotalTileBytes = n }},
		{"config entries", func(l *snspd.Limits, n uint64) { l.MaxConfigEntries = n }},
		{"config bytes", func(l *snspd.Limits, n uint64) { l.MaxConfigBytes = n }},
		{"memory", func(l *snspd.Limits, n uint64) { l.MaxMemoryBytes = n }},
		{"work", func(l *snspd.Limits, n uint64) { l.MaxWork = n }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, n := range []uint64{0, 1} {
				o := snspd.DefaultOptions()
				tc.set(&o.Limits, n)
				if d, err := snspd.Open(data, &o); d != nil || !errors.Is(err, snspd.ErrLimit) {
					t.Fatalf("limit=%d: drawing=%v err=%v", n, d, err)
				}
			}
		})
	}
	if _, err := snspd.Open(data, &snspd.Options{}); !errors.Is(err, snspd.ErrLimit) {
		t.Fatal("zero options", err)
	}
	for _, name := range []string{"dag-config-4", "dag-config-19", "dag-surface_1-4", "dag-surface_1-19"} {
		o := snspd.DefaultOptions()
		o.Limits.MaxConfigEntries = math.MaxUint32
		o.Limits.MaxTilesPerLayer = math.MaxUint32
		o.Limits.MaxWork = 200000
		if _, err := snspd.Open(read(t, name), &o); !errors.Is(err, snspd.ErrLimit) {
			t.Fatalf("%s: %v", name, err)
		}
	}
}
func TestBudgetFailureCleanup(t *testing.T) {
	data := read(t, "valid")
	for n := uint64(0); n < 180000; n += 5000 {
		for _, memory := range []bool{false, true} {
			o := snspd.DefaultOptions()
			if memory {
				o.Limits.MaxMemoryBytes = n
			} else {
				o.Limits.MaxWork = n
			}
			d, err := snspd.Open(data, &o)
			if err != nil && !errors.Is(err, snspd.ErrLimit) {
				t.Fatalf("budget %d memory=%v: %v", n, memory, err)
			}
			if (d == nil) != (err != nil) {
				t.Fatal("output/error mismatch")
			}
		}
	}
	open(t, "valid") // Later opens must survive failures inside private SQLite.
}
