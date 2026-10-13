//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"syscall"
	"time"
	"unsafe"
)

var (
	firstChangeNotification = kernel32.NewProc("FindFirstChangeNotificationW")
	nextChangeNotification  = kernel32.NewProc("FindNextChangeNotification")
	closeChangeNotification = kernel32.NewProc("FindCloseChangeNotification")
	waitChangeNotification  = kernel32.NewProc("WaitForSingleObject")
)

type sourceSubscription struct {
	path   string
	handle uintptr
}

func subscribeApplicationRoot(path string) uintptr {
	// Even subscription creation must fail before native access to network or
	// reparse ancestors. Files/missing roots use scoped periodic refresh instead
	// of expanding watches to parents.
	if LocalAppPath(path) != nil {
		return 0
	}
	wide, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0
	}
	attrs, _, _ := getFileAttributes.Call(uintptr(unsafe.Pointer(wide)))
	if uint32(attrs) == 0xffffffff || uint32(attrs)&0x10 == 0 {
		return 0
	}
	handle, _, _ := firstChangeNotification.Call(uintptr(unsafe.Pointer(wide)), 1, 0x1f)
	if handle == ^uintptr(0) {
		return 0
	}
	return handle
}

// WatchApplicationSources owns synchronous notification handles on one worker.
// Zero-time waits never block shutdown; no OVERLAPPED allocation/cancel race.
// A fixed 500-ms merge window caps continuous-event latency. Every 30 seconds
// a scoped refresh/retry also covers files, unavailable roots, and AppsFolder
// package changes that filesystem notifications do not represent.
func WatchApplicationSources(ctx context.Context, roots []string, notify func()) {
	subscriptions := make([]sourceSubscription, 0, min(len(roots), 32))
	for i, path := range roots {
		if i >= 32 || ctx.Err() != nil {
			break
		}
		subscriptions = append(subscriptions, sourceSubscription{path: path, handle: subscribeApplicationRoot(path)})
	}
	defer func() {
		for _, entry := range subscriptions {
			if entry.handle != 0 {
				closeChangeNotification.Call(entry.handle)
			}
		}
	}()
	tick := time.NewTicker(250 * time.Millisecond)
	defer tick.Stop()
	retry := time.NewTicker(30 * time.Second)
	defer retry.Stop()
	var due time.Time
	for {
		select {
		case <-ctx.Done():
			return
		case now := <-tick.C:
			for i := range subscriptions {
				entry := &subscriptions[i]
				if entry.handle == 0 {
					continue
				}
				state, _, _ := waitChangeNotification.Call(entry.handle, 0)
				if state == 0 {
					if due.IsZero() {
						due = now.Add(500 * time.Millisecond)
					}
					ok, _, _ := nextChangeNotification.Call(entry.handle)
					if ok == 0 {
						closeChangeNotification.Call(entry.handle)
						entry.handle = 0
					}
				} else if state == 0xffffffff {
					closeChangeNotification.Call(entry.handle)
					entry.handle = 0
					if due.IsZero() {
						due = now.Add(500 * time.Millisecond)
					}
				}
			}
			if !due.IsZero() && !now.Before(due) {
				due = time.Time{}
				if ctx.Err() == nil {
					notify()
				}
			}
		case now := <-retry.C:
			for i := range subscriptions {
				entry := &subscriptions[i]
				if entry.handle != 0 && LocalAppPath(entry.path) != nil {
					closeChangeNotification.Call(entry.handle)
					entry.handle = 0
				}
				if entry.handle == 0 && ctx.Err() == nil {
					entry.handle = subscribeApplicationRoot(entry.path)
				}
			}
			if due.IsZero() {
				due = now.Add(500 * time.Millisecond)
			}
		}
	}
}
