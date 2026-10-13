//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"testing"
)

func TestMergedSourceLegacyPreferenceActionsAndExplicitDisabledHint(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			old := core.Application{Name: "Fixture Link", Path: `C:\Synthetic\Fixture.lnk`}
			legacy := old.ItemID()
			old.RelatedIDs = []string{legacy}
			old.Launch.ShellTarget = core.AppsFolderParsingPrefix + "Fixture!App"
			old.Launch.AppUserModelIDKnown = true
			s.apps = core.MergeProviderApplications([]core.Application{old, {Name: "系统名称", ShellID: old.Launch.ShellTarget}})
			s.config.Favorites = []string{legacy}
			s.config.Aliases = map[string][]string{legacy: {"old-alias"}}
			s.persistPreferences = func(v settings.Settings) (settings.Settings, error) { return v, nil }
			s.toggleFavorite(s.apps[0].ItemID())
			if len(s.config.Favorites) != 0 {
				t.Fatal("legacy favorite could not be removed")
			}
			s.recordRecent(s.apps[0].ItemID())
			if len(s.config.Recents) != 1 {
				t.Fatal("merged recent failed")
			}
			s.toggleHiddenApplication(s.apps[0].ItemID())
			if len(s.applicationResults()) != 0 {
				t.Fatal("merged row not hidden")
			}
			s.toggleHiddenApplication(legacy)
			if len(s.applicationResults()) != 1 {
				t.Fatal("legacy restore failed")
			}
			s.openAliases(s.apps[0])
			if s.aliasText != "old-alias" {
				t.Fatal("alias editor lost old binding")
			}
			s.closeSettings()
			s.config.AppsFolder = false
			s.query = "store"
			tester := newPaletteTester(s, scale)
			if err := tester.Click("设置系统应用来源"); err != nil {
				t.Fatal(err)
			}
			if !s.settingsOpen || s.settingsDraft.AppsFolder {
				t.Fatal("hint silently enabled explicit disabled source")
			}
		})
	}
}
