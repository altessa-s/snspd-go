// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"reflect"
	"strings"
	"sync"
	"testing"

	"github.com/altessa-s/snspd-go"
)

func read(t testing.TB, name string) []byte {
	t.Helper()
	b, err := os.ReadFile("testdata/fixtures/" + name + ".spd")
	if err != nil {
		t.Fatal(err)
	}
	return b
}
func open(t testing.TB, name string) *snspd.Drawing {
	t.Helper()
	d, err := snspd.Open(read(t, name), nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}
func TestModel(t *testing.T) {
	d := open(t, "valid")
	a := d.Canvas
	if len(d.Layers) != 2 || len(d.Config) != 18 || d.ExtraTableCount != 1 || d.UnknownFormat || d.InvalidTileCount != 0 || d.HadWALHeader {
		t.Fatalf("model counts/flags: %+v", d)
	}
	if a.Width != 1920 || a.Height != 2560 || a.PPI != 72 || a.ViewportX != 249984 || a.ViewportY != 249984 || a.ViewportScale != 1 || !a.HasWidth || !a.HasHeight || !a.HasPPI || !a.HasViewportX || !a.HasViewportY || !a.HasViewportScale {
		t.Fatalf("canvas: %+v", a)
	}
	if a.FormatVersion == nil || *a.FormatVersion != "2" || a.PaperMode == nil || *a.PaperMode != "3" || a.TemplateName == nil || *a.TemplateName != "" || a.TemplateInfo == nil || *a.TemplateInfo != "-1,-1,," || a.AppVersion == nil || *a.AppVersion != "synthetic" || !bytes.Equal(a.Thumbnail, []byte{255, 216, 255, 217}) || a.TemplateData == nil || len(a.TemplateData) != 0 {
		t.Fatalf("canvas metadata: %+v", a)
	}
	l := d.Layers[0]
	ref := d.Layers[1]
	if l.ID != 1 || l.Name != "Layer 1" || l.HasField3 || l.HasField4 || len(l.Raw) == 0 || ref.ID != 9999 || ref.Name != "Reference Layer" || !ref.HasField3 || ref.Field3 != 2 || !ref.HasField4 || ref.Field4 != 1 || len(ref.Tiles) != 0 {
		t.Fatalf("layers: %+v", d.Layers)
	}
	if len(l.Tiles) != 3 {
		t.Fatal("tile count", len(l.Tiles))
	}
	for i, want := range [][2]uint64{{1952, 1953}, {1952, 1957}, {1955, 1954}} {
		tile := l.Tiles[i]
		if tile.TX != want[0] || uint64(tile.TY) != want[1] || tile.TID != int64(want[0]*4096+want[1]) || !tile.HasExpectedIHDR || !bytes.HasPrefix(tile.PNG, []byte("\x89PNG\r\n\x1a\n")) {
			t.Fatalf("tile %d: %+v", i, tile)
		}
	}
	for i, want := range []snspd.Rect{
		{X: -128, Y: 0, Width: 128, Height: 128},
		{X: -128, Y: 512, Width: 128, Height: 128},
		{X: 256, Y: 128, Width: 128, Height: 128},
	} {
		if got := d.TileRect(&l.Tiles[i]); got != want {
			t.Fatalf("tile %d rectangle=%+v, want %+v", i, got, want)
		}
	}
	for _, layer := range []*snspd.Layer{nil, &l} {
		if got := d.PaintedBounds(layer); got != (snspd.Rect{X: -128, Width: 512, Height: 640}) {
			t.Fatal(got)
		}
	}
	if !d.PaintedBounds(&ref).IsNull {
		t.Fatal("empty layer bounds")
	}
	for i, e := range d.Config {
		if i > 0 && d.Config[i-1].Key >= e.Key {
			t.Fatal("config order")
		}
		if d.ConfigFind(e.Key) != &d.Config[i] {
			t.Fatal("lookup identity", e.Key)
		}
	}
	if e := d.ConfigFind("unknown.key"); e == nil || !bytes.Equal(e.Value, []byte{0, 255}) || e.IsText || e.IsNull {
		t.Fatalf("raw config: %+v", e)
	}
	if e := d.ConfigFind("nul\x00key"); e == nil || string(e.Value) != "value" {
		t.Fatal("NUL key")
	}
	if e := d.ConfigFind("null.key"); e == nil || !e.IsNull || e.Value != nil {
		t.Fatal("NULL config")
	}
	if e := d.ConfigFind("text.key"); e == nil || !e.IsText || string(e.Value) != "text" {
		t.Fatal("TEXT config")
	}
	if e := d.ConfigFind("template_name"); e == nil || e.Value == nil || len(e.Value) != 0 || e.IsNull {
		t.Fatal("empty config")
	}
	if d.ConfigFind("absent") != nil {
		t.Fatal("absent config found")
	}
}
func TestVariants(t *testing.T) {
	for _, name := range []string{"without-rowid", "nullable-config", "nonnull-config", "collation-index", "trigger", "unordered-config", "large-config"} {
		t.Run(name, func(t *testing.T) {
			if len(open(t, name).Layers[0].Tiles) != 3 {
				t.Fatal("tiles missing")
			}
		})
	}
	d := open(t, "wal")
	if !d.HadWALHeader {
		t.Fatal("WAL flag missing")
	}
	if !open(t, "unknown-version").UnknownFormat {
		t.Fatal("unknown format missing")
	}
	d = open(t, "unknown-fields")
	if d.UnknownLayerFieldCount != 4 || d.Layers[0].UnknownFieldCount != 3 || len(d.Layers[0].Raw) <= 11 {
		t.Fatal("unknown fields lost")
	}
	d = open(t, "null-config")
	if d.Canvas.HasPPI || d.Canvas.PPI != 0 {
		t.Fatal("missing PPI presence")
	}
	d = open(t, "empty-layers")
	if len(d.Layers) != 0 || !d.PaintedBounds(nil).IsNull {
		t.Fatal("empty drawing")
	}
	d = open(t, "huge-tid")
	tile := d.Layers[0].Tiles[0]
	if tile.TID != math.MaxInt64 || tile.TX != uint64(math.MaxInt64)/4096 || tile.TY != 4095 {
		t.Fatal("large tile ID")
	}
	wantRect := snspd.Rect{X: float64(tile.TX)*128 - d.Canvas.ViewportX,
		Y: 4095*128 - d.Canvas.ViewportY, Width: 128, Height: 128}
	if got := d.TileRect(&tile); got != wantRect {
		t.Fatalf("large tile rectangle=%+v, want %+v", got, wantRect)
	}
	r := d.PaintedBounds(nil)
	if math.IsInf(r.X, 0) || r != wantRect {
		t.Fatal(r)
	}
	a := open(t, "decimal-rounding").Canvas
	if a.PPI != math.MaxFloat64 || a.Width != 0x1p-1022 || a.ViewportX != math.SmallestNonzeroFloat64 || !math.Signbit(a.ViewportY) || a.ViewportScale != 0.1 {
		t.Fatalf("decimal rounding: %+v", a)
	}
	if open(t, "decimal-tie").Canvas.ViewportX != 1 || open(t, "decimal-above-tie").Canvas.ViewportX != math.Nextafter(1, 2) {
		t.Fatal("decimal ties")
	}
}
func TestGridBoundary(t *testing.T) {
	d := open(t, "grid-boundary")
	tiles := d.Layers[0].Tiles
	if len(tiles) != 3 {
		t.Fatal("tile count", len(tiles))
	}
	for i, want := range []struct {
		tid int64
		tx  uint64
		ty  uint32
	}{{0, 0, 0}, {4095, 0, 4095}, {4096, 1, 0}} {
		if got := tiles[i]; got.TID != want.tid || got.TX != want.tx || got.TY != want.ty {
			t.Fatalf("tile %d: id=%d grid=(%d,%d), want %+v", i, got.TID, got.TX, got.TY, want)
		}
	}
	want := snspd.Rect{X: -249984, Y: -249984, Width: 256, Height: 524288}
	if got := d.PaintedBounds(nil); got != want {
		t.Fatalf("boundary bounds=%+v, want %+v", got, want)
	}
}
func TestOpenErrors(t *testing.T) {
	malformed := strings.Fields("missing-table wrong-types negative-tid duplicate-layers malformed-protobuf null-tile wrong-tile-type view bad-config-schema bad-decimal missing-ls bad-group overflow-varint missing-id max-id wrong-wire bad-key-type extra-column generated-column duplicate-config corrupt-page virtual-config attach-schema truncated cycle-config cycle-surface_1 cycle-deep-config cycle-deep-surface_1")
	cases := map[string]error{"empty": snspd.ErrUnexpectedEOF, "corrupt-header": snspd.ErrUnsupportedFormat}
	for _, name := range malformed {
		cases[name] = snspd.ErrMalformed
	}
	for name, want := range cases {
		t.Run(name, func(t *testing.T) {
			d, err := snspd.Open(read(t, name), nil)
			var detail *snspd.Error
			if d != nil || !errors.Is(err, want) || !errors.As(err, &detail) || detail.Message == "" {
				t.Fatalf("drawing=%v err=%v want=%v", d, err, want)
			}
			if detail.Message != detail.Status.String() || detail.Error() != "snspd: "+detail.Status.String() {
				t.Fatalf("C diagnostic must retain the status phrase without duplicating it: %+v", detail)
			}
			if name == "corrupt-page" && detail.Value == 0 {
				t.Fatal("SQLite result code lost")
			}
		})
	}
	for _, data := range [][]byte{nil, {}, []byte("SQLite")} {
		if _, err := snspd.Open(data, nil); !errors.Is(err, snspd.ErrUnexpectedEOF) {
			t.Fatal(err)
		}
	}
}
func TestPolicies(t *testing.T) {
	o := snspd.DefaultOptions()
	o.Strict = true
	if _, err := snspd.Open(read(t, "unknown-version"), &o); !errors.Is(err, snspd.ErrUnsupportedFormat) {
		t.Fatal(err)
	}
	if _, err := snspd.Open(read(t, "valid"), &o); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"non-png", "wrong-ihdr"} {
		d := open(t, name)
		if d.InvalidTileCount != 3 || d.Layers[0].Tiles[0].HasExpectedIHDR {
			t.Fatal("invalid tile count")
		}
		o.RequirePNG128 = true
		if _, err := snspd.Open(read(t, name), &o); !errors.Is(err, snspd.ErrMalformed) {
			t.Fatal(err)
		}
	}
}
func TestOwnershipAndConcurrency(t *testing.T) {
	input := read(t, "valid")
	before := bytes.Clone(input)
	want, err := snspd.Open(input, nil)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(input, before) {
		t.Fatal("Open mutated input")
	}
	clear(input)
	if !reflect.DeepEqual(want, open(t, "valid")) {
		t.Fatal("model retains input")
	}
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				got, err := snspd.Open(before, nil)
				if err != nil || !reflect.DeepEqual(got, want) {
					t.Errorf("concurrent result differs: %v", err)
					return
				}
				clear(got.Layers[0].Tiles[0].PNG)
				clear(got.Canvas.Thumbnail)
				clear(got.ConfigFind("ls").Value)
			}
		}()
	}
	wg.Wait()
	if !reflect.DeepEqual(want, open(t, "valid")) {
		t.Fatal("separate results alias")
	}
}
func TestErrorsMatchByStatus(t *testing.T) {
	sentinels := []error{snspd.ErrInvalidArgument, snspd.ErrOutOfMemory, snspd.ErrUnsupportedFormat, snspd.ErrUnexpectedEOF, snspd.ErrMalformed, snspd.ErrLimit}
	for i, sentinel := range sentinels {
		status := snspd.Status(i + 1)
		e := &snspd.Error{Status: status, Value: 17, Message: "detail"}
		if !errors.Is(fmt.Errorf("outer: %w", e), sentinel) || e.Error() != "snspd: "+status.String()+": detail" {
			t.Fatal(e)
		}
		var nilError *snspd.Error
		if e.Is(nilError) || nilError.Is(e) || nilError.Error() != "snspd: <nil>" || e.Is(errors.New(e.Error())) {
			t.Fatal("invalid error match")
		}
		for j, other := range sentinels {
			if errors.Is(e, other) != (i == j) {
				t.Fatal("status conflated")
			}
		}
	}
	if snspd.Status(999).String() != "status(999)" || snspd.StatusOK.String() != "success" {
		t.Fatal("status names")
	}
}
func TestNilGeometry(t *testing.T) {
	var d *snspd.Drawing
	if !d.TileRect(nil).IsNull || !d.PaintedBounds(nil).IsNull || d.ConfigFind("") != nil {
		t.Fatal("nil receiver")
	}
	d = &snspd.Drawing{Canvas: snspd.Canvas{ViewportX: 0.5, ViewportY: -1.25, ViewportScale: 99}}
	if !d.TileRect(nil).IsNull {
		t.Fatal("nil tile")
	}
	if got := d.TileRect(&snspd.Tile{TX: 1, TY: 2}); got != (snspd.Rect{X: 127.5, Y: 257.25, Width: 128, Height: 128}) {
		t.Fatal(got)
	}
}
