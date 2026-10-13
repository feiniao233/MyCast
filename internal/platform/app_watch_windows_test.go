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
		WatchApplicationSources(ctx, []string{root}, false, func(bool, bool) { t.Error("callback after cancellation") })
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

func TestRunningWatchIdleMergeReplacementRetryAndHandleCleanup(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "Configured")
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	ready := make(chan []sourceSubscription, 1)
	events := make(chan [2]bool, 16)
	finished := make(chan struct{})
	options := sourceWatchOptions{tick: 10 * time.Millisecond, merge: 40 * time.Millisecond, retry: 80 * time.Millisecond, ready: func(v []sourceSubscription) { ready <- v }}
	go func() {
		watchApplicationSources(ctx, []string{root}, false, func(files, shell bool) { events <- [2]bool{files, shell} }, options)
		close(finished)
	}()
	var initial []sourceSubscription
	select {
	case initial = <-ready:
	case <-time.After(time.Second):
		t.Fatal("watch registration timeout")
	}
	if len(initial) != 1 || initial[0].handle == 0 {
		t.Fatal("watch did not subscribe fixture")
	}
	// Multiple accelerated retry periods must not scan an unchanged healthy tree.
	select {
	case <-events:
		t.Fatal("idle healthy source requested full scan")
	case <-time.After(210 * time.Millisecond):
	}
	for n := 0; n < 16; n++ {
		if err := os.WriteFile(filepath.Join(root, "Fixture.exe"), []byte{byte(n)}, 0600); err != nil {
			t.Fatal(err)
		}
	}
	select {
	case event := <-events:
		if event != [2]bool{true, false} {
			t.Fatal("file event expanded to Shell")
		}
	case <-time.After(time.Second):
		t.Fatal("missing merged callback")
	}
	select {
	case <-events:
		t.Fatal("burst failed to coalesce")
	case <-time.After(100 * time.Millisecond):
	}
	// Replace the path with a new directory identity while the old handle still
	// names the old inode. Retry must compare identity, retire/rebind, and scan.
	if err := os.Rename(root, root+"Old"); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(root, 0700); err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("replacement did not request refresh")
	}
	// Give the retry time to rebind before creating inside the replacement.
	time.Sleep(120 * time.Millisecond)
	for {
		select {
		case <-events:
			continue
		default:
			goto drained
		}
	}
drained:
	if err := os.WriteFile(filepath.Join(root, "Replacement.exe"), []byte("inert"), 0600); err != nil {
		t.Fatal(err)
	}
	select {
	case <-events:
	case <-time.After(time.Second):
		t.Fatal("replacement directory was not watched")
	}
	cancel()
	select {
	case <-finished:
	case <-time.After(time.Second):
		t.Fatal("running cancellation stuck")
	}
	if state, _, _ := waitChangeNotification.Call(initial[0].handle, 0); state != 0xffffffff {
		t.Fatal("retired native handle still open")
	}
	select {
	case <-events:
		t.Fatal("callback after running cancellation")
	case <-time.After(60 * time.Millisecond):
	}
}

func TestWatchShellTimerDoesNotRequestHealthyFileScan(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	events := make(chan [2]bool, 8)
	done := make(chan struct{})
	go func() {
		watchApplicationSources(ctx, []string{t.TempDir()}, true, func(f, s bool) { events <- [2]bool{f, s} }, sourceWatchOptions{tick: 5 * time.Millisecond, merge: 10 * time.Millisecond, retry: 50 * time.Millisecond})
		close(done)
	}()
	for n := 0; n < 2; n++ {
		select {
		case event := <-events:
			if event != [2]bool{false, true} {
				t.Fatal("Shell timer expanded to healthy files")
			}
		case <-time.After(time.Second):
			t.Fatal("Shell timer absent")
		}
	}
	cancel()
	<-done
}
