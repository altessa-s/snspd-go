// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"testing"

	"github.com/altessa-s/snspd-go"
)

// BenchmarkOpenParallel measures parsing and conversion under gate contention.
// The shared input is a small, immutable synthetic fixture; loading is untimed.
func BenchmarkOpenParallel(b *testing.B) {
	data := read(b, "valid")
	b.ReportAllocs()
	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			if _, err := snspd.Open(data, nil); err != nil {
				b.Error(err)
				return
			}
		}
	})
}
