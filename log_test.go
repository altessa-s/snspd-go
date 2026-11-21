// Copyright 2026 ALTESSA SOLUTIONS INC.
// Use of this source code is governed by an MIT-style license that can be
// found in the LICENSE file.

package snspd_test

import (
	"errors"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/altessa-s/snspd-go"
)

func TestLogger(t *testing.T) {
	var calls atomic.Int64
	snspd.SetLogger(func(level snspd.LogLevel, message string) {
		if level != snspd.LogError || message == "" {
			t.Errorf("diagnostic: %v %q", level, message)
		}
		calls.Add(1)
	})
	defer snspd.SetLogger(nil)
	if _, err := snspd.Open(nil, nil); !errors.Is(err, snspd.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
	if calls.Load() != 1 {
		t.Fatal("logger calls", calls.Load())
	}
	snspd.SetLogger(nil)
	_, _ = snspd.Open(nil, nil)
	if calls.Load() != 1 {
		t.Fatal("silenced logger called")
	}
	snspd.SetLogger(func(snspd.LogLevel, string) { calls.Add(10) })
	_, _ = snspd.Open(nil, nil)
	if calls.Load() != 11 {
		t.Fatal("replacement logger not used")
	}
	if snspd.LogDebug.String() != "debug" || snspd.LogError.String() != "error" || snspd.LogLevel(9).String() != "log_level(9)" {
		t.Fatal("log level names")
	}
}
func TestLoggerPanicAndGoexit(t *testing.T) {
	defer snspd.SetLogger(nil)
	for _, handler := range []func(snspd.LogLevel, string){
		func(snspd.LogLevel, string) { panic("handler") },
		func(snspd.LogLevel, string) { runtime.Goexit() },
	} {
		snspd.SetLogger(handler)
		done := make(chan error, 1)
		go func() { _, err := snspd.Open(nil, nil); done <- err }()
		select {
		case err := <-done:
			if !errors.Is(err, snspd.ErrUnexpectedEOF) {
				t.Fatal(err)
			}
		case <-time.After(5 * time.Second):
			t.Fatal("logger stranded C call")
		}
	}
}
func TestLoggerCanReenter(t *testing.T) {
	data := read(t, "valid")
	var calls atomic.Int64
	snspd.SetLogger(func(snspd.LogLevel, string) {
		// Silence before reentry to avoid recursive diagnostics.
		snspd.SetLogger(nil)
		d, err := snspd.Open(data, nil)
		if err != nil || d == nil {
			t.Errorf("reentrant Open: %v", err)
		}
		calls.Add(1)
	})
	defer snspd.SetLogger(nil)
	// Non-PNG diagnostics are emitted after SQLite shutdown/unlock.
	open(t, "non-png")
	if calls.Load() != 1 {
		t.Fatal("callback never ran")
	}
}
func TestConcurrentLoggerReplacement(t *testing.T) {
	var calls atomic.Int64
	handler := func(snspd.LogLevel, string) { calls.Add(1) }
	snspd.SetLogger(handler)
	defer snspd.SetLogger(nil)
	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 20 {
				if _, err := snspd.Open(nil, nil); !errors.Is(err, snspd.ErrUnexpectedEOF) {
					t.Error(err)
				}
			}
		}()
	}
	for range 20 {
		snspd.SetLogger(nil)
		snspd.SetLogger(handler)
	}
	wg.Wait()
}
func TestInFlightLogSurvivesSilencing(t *testing.T) {
	started, release := make(chan struct{}), make(chan struct{})
	snspd.SetLogger(func(snspd.LogLevel, string) { close(started); <-release })
	defer snspd.SetLogger(nil)
	done := make(chan error, 1)
	go func() { _, err := snspd.Open(nil, nil); done <- err }()
	<-started
	snspd.SetLogger(nil)
	close(release)
	if err := <-done; !errors.Is(err, snspd.ErrUnexpectedEOF) {
		t.Fatal(err)
	}
}
