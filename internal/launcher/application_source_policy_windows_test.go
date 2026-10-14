//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestPartialShellRefreshKeepsVerifiedStoreButDropsOldDesktopRows(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic`}
	store := core.Application{Name: "微软商店", ShellID: core.AppsFolderParsingPrefix + "Microsoft.WindowsStore_8wekyb3d8bbwe!App", ShellPackaged: true}
	desktop := core.Application{Name: "Old Desktop", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop", ShellDesktop: true}
	s.apps = []core.Application{desktop, store}
	updates := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { updates <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		return nil, nil, nil
	}
	s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
		return nil, nil, []error{errors.New("synthetic package API unavailable")}
	}
	s.scan()
	select {
	case fn := <-updates:
		fn()
	case <-time.After(time.Second):
		t.Fatal("scan timeout")
	}
	if len(s.apps) != 1 || s.apps[0].ItemID() != store.ItemID() || !s.apps[0].ShellPackaged {
		t.Fatal("partial retention erased verified Store or resurrected desktop source")
	}
}

func TestPartialRootRefreshPreservesGlobalSourcePriority(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.AppsFolder = false
	s.config.IndexRoots = []string{`C:\Synthetic\Z-user`, `C:\Synthetic\A-common`}
	user := core.Application{Name: "Fixture", Path: `C:\Synthetic\Z-user\Fixture.lnk`, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, Arguments: "--user", AppUserModelIDKnown: true}}
	common := user
	common.Path, common.Launch.Arguments = `C:\Synthetic\A-common\Fixture.lnk`, "--common"
	s.apps = core.MergeProviderApplications([]core.Application{user})
	updates := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { updates <- fn }
	s.scanWork = func(_ context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		if len(roots) != 1 || roots[0] != s.config.IndexRoots[1] {
			t.Error("wrong partial source")
		}
		// A one-root scan numbers its source zero; full configured order wins.
		return []core.Application{common}, nil, nil
	}
	s.automaticRefreshRoots(scanFiles, []string{s.config.IndexRoots[1]})
	select {
	case fn := <-updates:
		fn()
	case <-time.After(time.Second):
		t.Fatal("scan timeout")
	}
	if len(s.apps) != 1 || s.apps[0].Path != user.Path || s.apps[0].Launch.Arguments != "--user" {
		t.Fatal("partial source arrival replaced higher priority original link")
	}
}
