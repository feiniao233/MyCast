//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"github.com/feiniao233/MyCast/internal/settings"
	"sync"
	"time"
)

// The worker, rather than a UI completion callback, drains the one pending job.
// Clear takes priority over queued saves. Disk I/O remains serial and bounded.
type snapshotWorker struct {
	store            *settings.SnapshotStore
	notify           func(snapshotJob, error)
	save             func(snapshotJob) error // controlled slow-write fixture
	mu               sync.Mutex
	running, closing bool
	pending          *snapshotJob
	clearDone        chan struct{}
	clearErr         error
}

func (w *snapshotWorker) submit(job snapshotJob) bool {
	w.mu.Lock()
	if w.closing {
		w.mu.Unlock()
		return false
	}
	if job.clear {
		if w.clearDone == nil {
			w.clearDone = make(chan struct{})
		} else {
			select {
			case <-w.clearDone:
				w.clearDone = make(chan struct{})
			default:
				w.mu.Unlock()
				return true
			}
		}
		w.clearErr = nil
	}
	if w.running {
		if job.clear || w.pending == nil || !w.pending.clear {
			w.pending = &job
		}
		w.mu.Unlock()
		return true
	}
	w.running = true
	w.mu.Unlock()
	go w.run(job)
	return true
}
func (w *snapshotWorker) run(job snapshotJob) {
	for {
		var err error
		if job.clear {
			err = w.store.Clear()
		} else if w.save != nil {
			err = w.save(job)
		} else {
			err = w.store.Save(job.value, true, job.epoch, time.Now())
		}
		w.mu.Lock()
		if job.clear {
			w.clearErr = err
			close(w.clearDone)
		}
		next := w.pending
		w.pending = nil
		if next == nil {
			w.running = false
		}
		notify := next == nil && !w.closing
		w.mu.Unlock()
		if next != nil {
			job = *next
			continue
		} // No UI callback can delay the queued clear.
		if notify && w.notify != nil {
			w.notify(job, err)
		}
		return
	}
}
func (w *snapshotWorker) beginShutdown() {
	w.mu.Lock()
	if !w.closing {
		w.closing = true
		w.store.Invalidate() // Cancel any save which has not committed.
		if w.pending != nil && !w.pending.clear {
			w.pending = nil
		}
	}
	w.mu.Unlock()
}
func (w *snapshotWorker) waitForClear(budget time.Duration) error {
	w.mu.Lock()
	done := w.clearDone
	w.mu.Unlock()
	if done == nil {
		return nil
	}
	timer := time.NewTimer(budget)
	defer timer.Stop()
	select {
	case <-done:
		w.mu.Lock()
		err := w.clearErr
		w.mu.Unlock()
		if err != nil {
			return fmt.Errorf("启动快照清除失败；原缓存可能保留，请重启后显式清除")
		}
		return nil
	case <-timer.C:
		return fmt.Errorf("启动快照清除超过退出等待上限；原缓存可能保留，请重启后显式清除")
	}
}
