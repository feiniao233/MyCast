//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"os"
	"runtime"
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
	lastFallback time.Time
	path         string
	handle       uintptr
	info         os.FileInfo
}
type sourceWatchOptions struct {
	fallback           time.Duration
	subscribe          func(string) uintptr
	tick, merge, retry time.Duration
	ready              func([]sourceSubscription)
}

func subscribeApplicationRoot(path string) uintptr {
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
	runtime.KeepAlive(wide)
	if handle == ^uintptr(0) {
		return 0
	}
	return handle
}

func applicationSourceInfo(path string) os.FileInfo {
	if LocalAppPath(path) != nil {
		return nil
	}
	info, err := os.Stat(path)
	if err != nil {
		return nil
	}
	return info
}
func sameSourceInfo(a, b os.FileInfo) bool {
	if a == nil || b == nil {
		return a == nil && b == nil
	}
	if !os.SameFile(a, b) {
		return false
	}
	if a.IsDir() && b.IsDir() {
		return true
	}
	return a.Size() == b.Size() && a.ModTime().Equal(b.ModTime())
}

// Healthy directory roots are event-driven. Retry polls only root identity/file
// metadata and attempts failed subscriptions, never a complete healthy tree.
// Shell fallback is separate, enabled only for a configured AppsFolder source.
func WatchApplicationSources(ctx context.Context, roots []string, shell bool, notify func(files, shell bool, roots []string)) {
	watchApplicationSources(ctx, roots, shell, notify, sourceWatchOptions{tick: 250 * time.Millisecond, merge: 500 * time.Millisecond, retry: 30 * time.Second})
}

func watchApplicationSources(ctx context.Context, roots []string, shell bool, notify func(bool, bool, []string), options sourceWatchOptions) {
	if options.fallback <= 0 {
		options.fallback = 2 * time.Minute
	}
	if options.subscribe == nil {
		options.subscribe = subscribeApplicationRoot
	}
	subscriptions := make([]sourceSubscription, 0, min(len(roots), 32))
	for i, path := range roots {
		if i >= 32 || ctx.Err() != nil {
			break
		}
		subscriptions = append(subscriptions, sourceSubscription{path: path, info: applicationSourceInfo(path), handle: options.subscribe(path), lastFallback: time.Now()})
	}
	defer func() {
		for _, entry := range subscriptions {
			if entry.handle != 0 {
				closeChangeNotification.Call(entry.handle)
			}
		}
	}()
	if options.ready != nil {
		options.ready(append([]sourceSubscription(nil), subscriptions...))
	}
	tick := time.NewTicker(options.tick)
	defer tick.Stop()
	retry := time.NewTicker(options.retry)
	defer retry.Stop()
	var due time.Time
	dirtyFiles, dirtyShell := false, false
	changedRoots := map[string]bool{}
	mark := func(now time.Time, files, shell bool, root string) {
		if files && root != "" {
			changedRoots[root] = true
		}
		dirtyFiles = dirtyFiles || files
		dirtyShell = dirtyShell || shell
		if due.IsZero() {
			due = now.Add(options.merge)
		}
	}
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
					mark(now, true, false, entry.path)
					ok, _, _ := nextChangeNotification.Call(entry.handle)
					if ok == 0 {
						closeChangeNotification.Call(entry.handle)
						entry.handle = 0
					}
				} else if state == 0xffffffff {
					closeChangeNotification.Call(entry.handle)
					entry.handle = 0
					mark(now, true, false, entry.path)
				}
			}
			if !due.IsZero() && !now.Before(due) {
				files, shell := dirtyFiles, dirtyShell
				roots := make([]string, 0, len(changedRoots))
				for root := range changedRoots {
					roots = append(roots, root)
				}
				changedRoots = map[string]bool{}
				dirtyFiles, dirtyShell = false, false
				due = time.Time{}
				if ctx.Err() == nil {
					notify(files, shell, roots)
				}
			}
		case now := <-retry.C:
			for i := range subscriptions {
				if ctx.Err() != nil {
					return
				}
				entry := &subscriptions[i]
				info := applicationSourceInfo(entry.path)
				replaced := !sameSourceInfo(entry.info, info)
				if replaced {
					if entry.handle != 0 {
						closeChangeNotification.Call(entry.handle)
						entry.handle = 0
					}
					entry.info = info
					mark(now, true, false, entry.path)
				}
				if entry.handle == 0 && ctx.Err() == nil && info != nil && info.IsDir() {
					entry.handle = options.subscribe(entry.path)
					if entry.handle != 0 {
						mark(now, true, false, entry.path)
					}
					if entry.handle == 0 && now.Sub(entry.lastFallback) >= options.fallback {
						entry.lastFallback = now
						mark(now, true, false, entry.path)
					}
				}
			}
			if shell {
				mark(now, false, true, "")
			}
		}
	}
}
