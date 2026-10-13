//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNativeConfiguredRootNotificationsCoalesceAndCancel(t *testing.T) {
	root := t.TempDir()
	// Exercise the exact native ABI with an owned directory; no desktop input or
	// system installation is involved. Wait for registration before mutations.
	handle := subscribeApplicationRoot(root)
	if handle == 0 {
		t.Fatal("fixture subscription failed")
	}
	defer closeChangeNotification.Call(handle)
	for n := 0; n < 8; n++ {
		if err := os.WriteFile(filepath.Join(root, "Fixture.exe"), []byte{byte(n)}, 0600); err != nil {
			t.Fatal(err)
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	signaled := false
	for time.Now().Before(deadline) {
		state, _, _ := waitChangeNotification.Call(handle, 0)
		if state == 0 {
			signaled = true
			break
		}
		time.Sleep(10 * time.Millisecond)
	}
	if !signaled {
		t.Fatal("native change not signaled")
	}
	if ok, _, _ := nextChangeNotification.Call(handle); ok == 0 {
		t.Fatal("notification rearm failed")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	finished := make(chan struct{})
	go func() {
		WatchApplicationSources(ctx, []string{root}, func() { t.Error("callback after cancellation") })
		close(finished)
	}()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("watch did not close promptly")
	}
	if subscribeApplicationRoot(`\\Synthetic\Share`) != 0 {
		t.Fatal("network subscription accepted")
	}
	if subscribeApplicationRoot(filepath.Join(root, "Fixture.exe")) != 0 {
		t.Fatal("file root expanded into parent watch")
	}
}
