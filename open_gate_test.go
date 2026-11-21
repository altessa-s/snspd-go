// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd

import (
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestOpenWaitsOutsideC(t *testing.T) {
	data, err := os.ReadFile("testdata/fixtures/valid.spd")
	if err != nil {
		t.Fatal(err)
	}
	opts := DefaultOptions()
	openMu.Lock()
	var finished sync.WaitGroup
	defer func() {
		openMu.Unlock()
		finished.Wait()
	}()
	before := runtime.NumCgoCall()
	for range 64 {
		finished.Add(1)
		go func() {
			defer finished.Done()
			if _, err := Open(data, &opts); err != nil {
				t.Errorf("Open: %v", err)
			}
		}()
	}
	// Require every caller to be blocked in Open at the mutex, rather than
	// assuming a scheduling delay was long enough. Grow the snapshot buffer
	// if needed so unrelated goroutines cannot truncate the evidence.
	deadline := time.Now().Add(10 * time.Second)
	buf := make([]byte, 64*1024)
	for {
		n := runtime.Stack(buf, true)
		if n == len(buf) {
			buf = make([]byte, 2*len(buf))
			continue
		}
		blocked := 0
		for _, stack := range strings.Split(string(buf[:n]), "\n\n") {
			if strings.Contains(stack, "[sync.Mutex.Lock]") &&
				strings.Contains(stack, "github.com/altessa-s/snspd-go.Open(") &&
				strings.Contains(stack, "TestOpenWaitsOutsideC.func") {
				blocked++
			}
		}
		if blocked == 64 {
			break
		}
		if time.Now().After(deadline) {
			t.Fatalf("only %d of 64 Opens blocked at the Go gate", blocked)
		}
		time.Sleep(time.Millisecond)
	}
	// Windows syscalls also increment the global cgo counter. The stack
	// assertion above verifies the gate there without counting those calls.
	if got := runtime.NumCgoCall() - before; runtime.GOOS != "windows" && got != 0 {
		t.Errorf("queued Opens entered C %d times", got)
	}
}
