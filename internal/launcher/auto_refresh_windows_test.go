//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"testing"
	"time"
)

func TestAutoEventsDoNotCancelRunningScanAndCoalesceOneFollowup(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic`}
	done := make(chan func(), 4)
	started := make(chan context.Context, 4)
	release := make(chan struct{})
	s.uiUpdate = func(fn func()) { done <- fn }
	calls := 0
	s.scanWork = func(ctx context.Context, _ []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		started <- ctx
		<-release
		return []core.Application{{Name: "Fixture", Path: `C:\Synthetic\Fixture.exe`}}, nil, nil
	}
	s.scan()
	var ctx context.Context
	select {
	case ctx = <-started:
	case <-time.After(time.Second):
		t.Fatal("not started")
	}
	for n := 0; n < 50; n++ {
		s.automaticRefresh()
	}
	if ctx.Err() != nil || s.scanQueue.Generation != 1 {
		t.Fatal("auto events cancelled/starved worker")
	}
	close(release)
	for n := 0; n < 2; n++ {
		select {
		case fn := <-done:
			fn()
			calls++
		case <-time.After(3 * time.Second):
			t.Fatal("missing scan")
		}
	}
	if calls != 2 || s.scanning || s.autoDirty || len(s.apps) != 1 {
		t.Fatal("events did not merge to one followup")
	}
	select {
	case fn := <-done:
		_ = fn
		t.Fatal("extra worker")
	default:
	}
}

func TestMixedPartialScanKeepsOfflineEntryButDropsConfirmedAndRemovedRoot(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic\Kept`}
	fresh := core.Application{Name: "A", Path: `C:\Synthetic\Kept\A.exe`}
	offline := core.Application{Name: "B", Path: `C:\Synthetic\Kept\B.exe`}
	missing := core.Application{Name: "C", Path: `C:\Synthetic\Kept\C.exe`}
	removed := core.Application{Name: "D", Path: `C:\Synthetic\Removed\D.exe`}
	shell := core.Application{Name: "Shell", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App"}
	s.apps = []core.Application{fresh, offline, missing, removed, shell}
	done := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { done <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		return []core.Application{fresh}, nil, []error{errors.New("synthetic offline source")}
	}
	s.scanMissing = func(context.Context, []core.Application) map[string]bool {
		return map[string]bool{missing.ItemID(): true}
	}
	s.scan()
	select {
	case fn := <-done:
		fn()
	case <-time.After(3 * time.Second):
		t.Fatal("worker timeout")
	}
	if len(s.apps) != 2 || s.apps[1].ItemID() != offline.ItemID() {
		t.Fatal("mixed partial snapshot discarded unconfirmed or revived missing/unconfigured")
	}
}

func TestFileSuccessAndShellFailureOrDeadlineRetainOnlyShellUntilRecovery(t *testing.T) {
	for _, deadline := range []bool{false, true} {
		t.Run(fmt.Sprint(deadline), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.config.IndexRoots = []string{`C:\Synthetic`}
			s.config.AppsFolder = true
			fresh := core.Application{Name: "Fresh", Path: `C:\Synthetic\Fresh.exe`}
			oldFile := core.Application{Name: "Old file", Path: `C:\Synthetic\Removed.exe`}
			oldShell := core.Application{Name: "Old shell", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App"}
			s.apps = []core.Application{oldFile, oldShell}
			s.selected = 1
			done := make(chan func(), 2)
			s.uiUpdate = func(fn func()) { done <- fn }
			s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
				return []core.Application{fresh}, nil, nil
			}
			s.scanTimeout = 20 * time.Millisecond
			s.scanShellWork = func(ctx context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
				if deadline {
					<-ctx.Done()
					return nil, nil, []error{ctx.Err()}
				}
				return nil, nil, []error{errors.New("synthetic Shell provider unavailable")}
			}
			s.scan()
			select {
			case fn := <-done:
				fn()
			case <-time.After(time.Second):
				t.Fatal("scan timeout")
			}
			if len(s.apps) != 2 || s.apps[1].ItemID() != oldShell.ItemID() || s.appNeedsSelection {
				t.Fatal("Shell failure lost selected identity or retained removed complete file source")
			}
			s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) { return nil, nil, nil }
			s.scan()
			select {
			case fn := <-done:
				fn()
			case <-time.After(time.Second):
				t.Fatal("recovery timeout")
			}
			if len(s.apps) != 1 || s.apps[0].ItemID() != fresh.ItemID() || !s.appNeedsSelection {
				t.Fatal("successful Shell recovery did not remove absent virtual identity")
			}
		})
	}
}

func TestEmptyPartialSnapshotCannotReviveRemovedRootsOrDisabledShell(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic\New`}
	s.apps = []core.Application{{Name: "Removed root", Path: `C:\Synthetic\Old\Fixture.exe`}, {Name: "Disabled Shell", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App"}}
	done := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { done <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		return nil, nil, []error{errors.New("synthetic unavailable configured new root")}
	}
	s.scan()
	select {
	case fn := <-done:
		fn()
	case <-time.After(time.Second):
		t.Fatal("worker timeout")
	}
	if len(s.apps) != 0 {
		t.Fatal("UI fallback revived removed/disabled provider")
	}
}
