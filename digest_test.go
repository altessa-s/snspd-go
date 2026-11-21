// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestVendoredDigest(t *testing.T) {
	var files []string
	err := filepath.WalkDir("internal/clib", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	slices.Sort(files)
	h := sha256.New()
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		relative, err := filepath.Rel("internal/clib", path)
		if err != nil {
			t.Fatal(err)
		}
		fmt.Fprintf(h, "%x ./%s\n", sha256.Sum256(data), filepath.ToSlash(relative))
	}
	if got := fmt.Sprintf("%x", h.Sum(nil)); got != vendoredDigest {
		t.Fatalf("vendored digest=%s, header=%s; run scripts/update-clib.sh", got, vendoredDigest)
	}
	data, err := os.ReadFile("internal/clib/VERSION")
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) != vendoredVersion || Version != headerVersion {
		t.Fatalf("version drift: Go %s header %s snapshot %s", Version, headerVersion, vendoredVersion)
	}
	for _, name := range []string{"log", "model", "parser"} {
		data, err := os.ReadFile("clib_" + name + ".c")
		if err != nil {
			t.Fatal(err)
		}
		for _, include := range []string{"clib_digest.h", "clib_symbols.h", "clib_sqlite.h", "internal/clib/src/spd_" + name + ".c"} {
			if !strings.Contains(string(data), `#include "`+include+`"`) {
				t.Errorf("%s wrapper missing %s", name, include)
			}
		}
	}
}
func TestSyntheticSeedDigests(t *testing.T) {
	data, err := os.ReadFile("testdata/seeds.sha256")
	if err != nil {
		t.Fatal(err)
	}
	checked := map[string]bool{}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		sum, path, ok := strings.Cut(line, "  ")
		if !ok {
			t.Fatal("invalid seed manifest")
		}
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if fmt.Sprintf("%x", sha256.Sum256(b)) != sum {
			t.Errorf("seed changed: %s", path)
		}
		checked[path] = true
	}
	err = filepath.WalkDir("testdata", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && (strings.HasSuffix(path, ".spd") || strings.Contains(filepath.ToSlash(path), "/fuzz/FuzzOpen/")) && !checked[filepath.ToSlash(path)] {
			t.Errorf("unrecorded seed: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
