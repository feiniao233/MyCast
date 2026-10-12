//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestChangingRootsCancelsAndSerializesLatestWorker(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic\Old`}
	started := make(chan []string, 4)
	completed := make(chan func(), 4)
	release := make(chan struct{})
	var workers, maxWorkers atomic.Int32
	s.uiUpdate = func(fn func()) { completed <- fn }
	s.scanWork = func(ctx context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		count := workers.Add(1)
		defer workers.Add(-1)
		for {
			old := maxWorkers.Load()
			if old >= count || maxWorkers.CompareAndSwap(old, count) {
				break
			}
		}
		started <- roots
		if roots[0] == `C:\Synthetic\Old` {
			<-ctx.Done()
			<-release
		}
		return []core.Application{{Name: roots[0], Path: roots[0] + `\Fixture.exe`}}, nil, nil
	}
	awaitRoots := func() []string {
		t.Helper()
		select {
		case roots := <-started:
			return roots
		case <-time.After(3 * time.Second):
			t.Fatal("worker did not start")
			return nil
		}
	}
	awaitCompletion := func() {
		t.Helper()
		select {
		case fn := <-completed:
			fn()
		case <-time.After(3 * time.Second):
			t.Fatal("worker did not finish")
		}
	}
	s.scan()
	awaitRoots()
	for i := 0; i < 50; i++ {
		s.config.IndexRoots = []string{`C:\Synthetic\Intermediate`}
		s.scan()
	}
	s.config.IndexRoots = []string{`C:\Synthetic\Latest`}
	s.scan()
	select {
	case <-started:
		t.Fatal("new worker started before cancelled old native call returned")
	default:
	}
	close(release)
	awaitCompletion()
	if roots := awaitRoots(); len(roots) != 1 || roots[0] != `C:\Synthetic\Latest` {
		t.Fatal("worker used stale roots")
	}
	awaitCompletion()
	if maxWorkers.Load() != 1 || s.scanning || len(s.apps) != 1 || s.apps[0].Name != `C:\Synthetic\Latest` {
		t.Fatal("workers overlapped or stale result replaced latest")
	}
}
