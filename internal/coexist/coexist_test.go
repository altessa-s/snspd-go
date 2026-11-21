// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package coexist

import (
	"os"
	"sync"
	"testing"

	"github.com/altessa-s/snspd-go"
)

func TestSQLiteCoexistence(t *testing.T) {
	h, err := OpenHost()
	if err != nil {
		t.Fatal(err)
	}
	defer h.Close()
	if err := h.Exec("CREATE TABLE retained(n); INSERT INTO retained VALUES(42)"); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile("../../testdata/fixtures/valid.spd")
	if err != nil {
		t.Fatal(err)
	}
	var wg sync.WaitGroup
	for range 4 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 10 {
				d, err := snspd.Open(data, nil)
				if err != nil || d == nil || len(d.Layers) != 2 {
					t.Errorf("binding beside live host SQLite: %v", err)
					return
				}
			}
		}()
	}
	for range 10 {
		if n, err := h.Value(); err != nil || n != 42 {
			t.Fatalf("host SQLite state: %d %v", n, err)
		}
	}
	wg.Wait()
	if err := h.Exec("UPDATE retained SET n=43"); err != nil {
		t.Fatal(err)
	}
	if n, err := h.Value(); err != nil || n != 43 {
		t.Fatalf("host after binding: %d %v", n, err)
	}
}
