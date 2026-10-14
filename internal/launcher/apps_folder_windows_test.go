//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"testing"
	"time"
)

func TestShellApplicationLaunchUsesIdentityAndRetainsPreferences(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			app := core.Application{Name: "Synthetic Shell App", ShellPackaged: true, ShellID: core.AppsFolderParsingPrefix + "Synthetic.Package!App"}
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.config.AppsFolder = false
			s.apps = []core.Application{app}
			s.persistPreferences = func(v settings.Settings) (settings.Settings, error) { return v, nil }
			s.config.Favorites = []string{app.ItemID()}
			s.config.Aliases = map[string][]string{app.ItemID(): {"fixture"}}
			calls := 0
			s.launchApplication = func(string) error { t.Fatal("virtual identity passed to file launcher"); return nil }
			s.launchShellApplication = func(id string) error {
				calls++
				if id != app.ShellID {
					t.Fatal("changed identity")
				}
				return nil
			}
			tester := newPaletteTester(s, scale)
			tester.Key(0, ui.KeyEnter)
			if calls != 1 || len(s.config.Recents) != 1 || s.config.Recents[0] != app.ItemID() {
				t.Fatal("Shell launch/recents")
			}
			app.Name = "合成应用新名称"
			s.refreshApplications(core.WithUserAliases([]core.Application{app}, s.config.Aliases), nil)
			if s.apps[0].ItemID() != s.config.Favorites[0] || s.apps[0].UserAliases[0] != "fixture" {
				t.Fatal("rename lost preferences")
			}
			tester.Frame()
			savePreview(t, tester, fmt.Sprintf("apps-folder-%03d", int(scale*100)))
			s.openSettings()
			tester.Frame()
			if err := tester.Click("AppsFolder 来源"); err != nil {
				t.Fatal(err)
			}
			if !s.settingsDraft.AppsFolder {
				t.Fatal("source control not interactive")
			}
			savePreview(t, tester, fmt.Sprintf("sources-settings-%03d", int(scale*100)))
		})
	}
}

func TestShellSourceCapturedBySerializedScan(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		s := syntheticLauncher()
		s.config = settings.Defaults()
		s.config.AppsFolder = enabled
		completions := make(chan func(), 1)
		s.uiUpdate = func(fn func()) { completions <- fn }
		s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
			return nil, nil, nil
		}
		calls := 0
		s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
			calls++
			return []core.Application{{Name: "Fixture", ShellPackaged: true, ShellID: core.AppsFolderParsingPrefix + "Fixture!App"}}, nil, nil
		}
		s.scan()
		select {
		case fn := <-completions:
			fn()
		case <-time.After(3 * time.Second):
			t.Fatal("worker timeout")
		}
		if enabled && (calls != 1 || len(s.apps) != 1) || !enabled && (calls != 0 || len(s.apps) != 0) {
			t.Fatal("source not honored")
		}
	}
}
