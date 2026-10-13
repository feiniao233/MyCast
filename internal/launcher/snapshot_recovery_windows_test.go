//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync/atomic"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestSnapshotPartialThenScopedRecoveryReviewsMissingRows(t *testing.T) {
	a, b := t.TempDir(), t.TempDir()
	store := &settings.SnapshotStore{Path: filepath.Join(t.TempDir(), "snapshot.json")}
	old := core.Application{Name: "Synthetic Removed", Path: filepath.Join(a, "Removed.exe")}
	fp := settings.SourceFingerprint([]string{a, b}, nil, true)
	if err := store.Save(settings.NewSnapshot([]core.Application{old}, fp, time.Now()), true, store.Epoch(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s := &launcher{config: settings.Defaults(), snapshotStore: store}
	s.config.IndexRoots = []string{a, b}
	s.config.StartupSnapshot = true
	callbacks := make(chan func(), 10)
	s.uiUpdate = func(fn func()) { callbacks <- fn }
	fileCalls, shellCalls := 0, 0
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		fileCalls++
		if fileCalls == 1 {
			return nil, nil, []error{errors.New("synthetic partial root batch")}
		}
		return nil, nil, nil
	}
	s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
		shellCalls++
		if shellCalls == 1 {
			return nil, nil, []error{errors.New("synthetic partial Shell batch")}
		}
		return nil, nil, nil
	}
	launches := 0
	s.launchApplication = func(string) error { launches++; return nil }
	s.launchShellApplication = s.launchApplication
	s.scan()
	awaitScanUI(t, callbacks)
	awaitScanUI(t, callbacks)
	if len(s.pendingApps) != 1 {
		t.Fatal("initial partial removed pending entry")
	}
	s.activate(s.applicationResults(), nil, false)
	s.scanSourcesWithRoots(scanFiles, []string{a})
	awaitScanUI(t, callbacks)
	if len(s.pendingApps) != 1 || s.scanning {
		t.Fatal("one root recovery guessed missing/unbounded review")
	}
	s.scanSourcesWithRoots(scanFiles, []string{b})
	awaitScanUI(t, callbacks)
	if len(s.pendingApps) != 1 || s.scanning {
		t.Fatal("unknown Shell source removed pending entry")
	}
	s.activate(s.applicationResults(), nil, false)
	s.scanSources(scanShell)
	awaitScanUI(t, callbacks)
	if !s.scanning || !s.pendingReviewQueued || len(s.pendingApps) != 1 {
		t.Fatal("all scoped sources healthy did not queue bounded full review")
	}
	awaitScanUI(t, callbacks) // Automatically serialized complete-source review.
	awaitScanUI(t, callbacks) // Complete empty snapshot write.
	defer s.stopSourceRetry()
	if len(s.pendingApps) != 0 || len(s.apps) != 0 || s.scanning || launches != 0 {
		t.Fatal("recovery did not converge safely")
	}
	if fileCalls != 4 || shellCalls != 3 {
		t.Fatal("unexpected/unbounded source work", fileCalls, shellCalls)
	}
	value, err := store.Load(fp, time.Now())
	if err != nil || len(value.Entries) != 0 {
		t.Fatal("complete recovery snapshot not updated", value, err)
	}
}

func TestSnapshotRecoveryReviewRequiresNewRecoveryAfterPartial(t *testing.T) {
	s := &launcher{pendingApps: []core.Application{{Name: "Synthetic Pending", DisplayOnlyID: "synthetic-id"}}}
	if s.needsPendingSnapshotReview([]string{"synthetic-root"}, false) {
		t.Fatal("unknown root allowed review")
	}
	s.recordSourceDiagnostics([]string{"synthetic-root"}, scanFiles, false, false, 1)
	if !s.needsPendingSnapshotReview([]string{"synthetic-root"}, false) {
		t.Fatal("recovered source did not allow review")
	}
	s.pendingReviewQueued = true
	if s.needsPendingSnapshotReview([]string{"synthetic-root"}, false) {
		t.Fatal("same recovery repeatedly queued full review")
	}
	s.recordSourceDiagnostics([]string{"synthetic-root"}, scanFiles, true, false, 2)
	if s.needsPendingSnapshotReview([]string{"synthetic-root"}, false) {
		t.Fatal("partial review treated source healthy")
	}
	s.recordSourceDiagnostics([]string{"synthetic-root"}, scanFiles, false, false, 3)
	if !s.needsPendingSnapshotReview([]string{"synthetic-root"}, false) {
		t.Fatal("new recovery transition could not retry")
	}
}

func TestSnapshotSlowSaveClearThenQuitDoesNotNeedUI(t *testing.T) {
	store := &settings.SnapshotStore{Path: filepath.Join(t.TempDir(), "snapshot.json")}
	if err := os.WriteFile(store.Path, []byte("synthetic original"), 0600); err != nil {
		t.Fatal(err)
	}
	started, release := make(chan struct{}), make(chan struct{})
	var callbacks atomic.Int32
	w := &snapshotWorker{store: store, notify: func(snapshotJob, error) { callbacks.Add(1) }, save: func(snapshotJob) error {
		close(started)
		<-release
		return os.WriteFile(store.Path, []byte("synthetic already committing slow write"), 0600)
	}}
	s := &launcher{snapshotStore: store, snapshotWorker: w, uiUpdate: func(func()) { t.Error("quit clear depended on UI dispatch") }}
	s.queueSnapshot(snapshotJob{epoch: store.Epoch()})
	<-started
	s.clearStartupSnapshot()
	s.quitting = true
	w.beginShutdown()
	close(release)
	if err := w.waitForClear(time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.Path); !errors.Is(err, os.ErrNotExist) || callbacks.Load() != 0 {
		t.Fatal("quit left cache or required UI callback", err, callbacks.Load())
	}
	if w.submit(snapshotJob{epoch: store.Epoch()}) {
		t.Fatal("shutdown accepted late save")
	}
}

func TestSnapshotClearQuitWaitHasBoundedFailure(t *testing.T) {
	store := &settings.SnapshotStore{Path: filepath.Join(t.TempDir(), "snapshot.json")}
	started, release := make(chan struct{}), make(chan struct{})
	w := &snapshotWorker{store: store, save: func(snapshotJob) error { close(started); <-release; return nil }}
	w.submit(snapshotJob{})
	<-started
	store.Invalidate()
	w.submit(snapshotJob{clear: true, epoch: store.Epoch()})
	w.beginShutdown()
	start := time.Now()
	err := w.waitForClear(10 * time.Millisecond)
	if err == nil || time.Since(start) > time.Second {
		t.Fatal("unbounded or silently successful quit clear", err)
	}
	close(release)
	if err = w.waitForClear(time.Second); err != nil {
		t.Fatal("controlled cleanup", err)
	}
}
