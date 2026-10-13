//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"testing"
	"time"
)

func TestPartialFileProviderRetriesWithoutEventAndSkipsHealthyRoot(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic\Failed`, `C:\Synthetic\Healthy`}
	s.retryDelay = 10 * time.Millisecond
	defer s.stopSourceRetry()
	failed := core.Application{Name: "Recovering", Path: `C:\Synthetic\Failed\Fixture.exe`}
	healthy := core.Application{Name: "Healthy", Path: `C:\Synthetic\Healthy\Fixture.exe`}
	s.apps = []core.Application{failed, healthy}
	updates := make(chan func(), 8)
	s.uiUpdate = func(fn func()) { updates <- fn }
	calls := 0
	s.scanWork = func(_ context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		calls++
		if len(roots) != 1 || roots[0] != s.config.IndexRoots[0] {
			t.Error("retry scanned healthy/unconfigured root")
		}
		if calls == 1 {
			return nil, nil, []error{errors.New("synthetic transient permission failure")}
		}
		return []core.Application{failed}, nil, nil
	}
	await := func() {
		t.Helper()
		select {
		case fn := <-updates:
			fn()
		case <-time.After(time.Second):
			t.Fatal("retry UI timeout")
		}
	}
	s.automaticRefreshRoots(scanFiles, []string{s.config.IndexRoots[0]})
	await()
	if len(s.apps) != 2 || s.retryTimer == nil {
		t.Fatal("partial did not preserve or schedule retry")
	}
	await()
	await() // timer callback then the single retry completion
	if calls != 2 || len(s.apps) != 2 || len(s.incompleteRoots) != 0 || s.retryTimer != nil {
		t.Fatal("retry did not recover/stop")
	}
}

func TestRetryEpochCancelsPostedWorkAndDelayBackoff(t *testing.T) {
	s := syntheticLauncher()
	s.retryDelay = 5 * time.Millisecond
	updates := make(chan func(), 4)
	s.uiUpdate = func(fn func()) { updates <- fn }
	s.recordFileScanCompletion([]string{`C:\Synthetic`}, true)
	var posted func()
	select {
	case posted = <-updates:
	case <-time.After(time.Second):
		t.Fatal("retry was not posted")
	}
	s.stopSourceRetry()
	posted()
	if s.scanQueue.Running || s.retryTimer != nil || len(s.incompleteRoots) != 0 {
		t.Fatal("retired retry scheduled worker")
	}
	// A pending timer is reused, not multiplied by rapid failure publications.
	s.recordFileScanCompletion([]string{`C:\Synthetic`}, true)
	timer := s.retryTimer
	for n := 0; n < 50; n++ {
		s.recordFileScanCompletion([]string{`C:\Synthetic`}, true)
	}
	if s.retryTimer != timer || s.retryAttempts != 1 {
		t.Fatal("retry timer proliferation")
	}
	s.stopSourceRetry()
}

func TestFileDeadlineBeforeShellPreservesBothProviders(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic`}
	s.config.AppsFolder = true
	s.scanTimeout = 10 * time.Millisecond
	file := s.apps[0]
	shell := core.Application{Name: "Shell", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App"}
	s.apps = []core.Application{file, shell}
	defer s.stopSourceRetry()
	updates := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { updates <- fn }
	s.scanWork = func(ctx context.Context, _ []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		<-ctx.Done()
		return nil, nil, []error{ctx.Err()}
	}
	s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
		t.Error("Shell ran after file deadline")
		return nil, nil, nil
	}
	s.scan()
	select {
	case fn := <-updates:
		fn()
	case <-time.After(time.Second):
		t.Fatal("deadline completion timeout")
	}
	if len(s.apps) != 2 || s.apps[0].ItemID() != file.ItemID() || s.apps[1].ItemID() != shell.ItemID() || s.retryTimer == nil {
		t.Fatal("file deadline discarded providers or failed to retry")
	}
}
