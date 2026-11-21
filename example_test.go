// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"fmt"
	"os"

	"github.com/altessa-s/snspd-go"
)

func ExampleOpen() {
	data, err := os.ReadFile("testdata/fixtures/valid.spd")
	if err != nil {
		panic(err)
	}
	opts := snspd.DefaultOptions()
	opts.Strict = true
	opts.RequirePNG128 = true
	drawing, err := snspd.Open(data, &opts)
	if err != nil {
		panic(err)
	}
	bounds := drawing.PaintedBounds(nil)
	fmt.Printf("%.0f×%.0f canvas, %d layers\n", drawing.Canvas.Width, drawing.Canvas.Height, len(drawing.Layers))
	fmt.Printf("stored tile extent: %.0f×%.0f\n", bounds.Width, bounds.Height)
	// Output:
	// 1920×2560 canvas, 2 layers
	// stored tile extent: 512×640
}
