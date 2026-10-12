//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"slices"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestFavoriteAndAliasEditorDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
			app := s.apps[0]
			tester := newPaletteTester(s, scale)
			tester.Click("操作")
			tester.Click("菜单项：收藏应用")
			if !slices.Contains(s.config.Favorites, app.ItemID()) {
				t.Fatal("favorite action failed")
			}
			tester.Click("操作")
			tester.Click("菜单项：编辑搜索别名")
			if !s.settingsOpen || !tester.Focused("应用别名") {
				t.Fatal("alias editor focus missing")
			}
			s.aliasText = "合成别名\nfixture"
			tester.Frame()
			for _, label := range []string{"应用别名", "保存别名", "取消别名"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < 36 || box.Y+box.H > palette.Height-36 {
					t.Fatalf("alias control clipped: %s %+v", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("aliases-%03d", int(scale*100)))
			tester.Click("保存别名")
			if s.settingsOpen || !tester.Focused("搜索") || len(core.SearchApplications(s.apps, "fixture", 10)) != 1 || len(core.SearchApplications(s.apps, "hcbm", 10)) != 1 {
				t.Fatal("alias save did not update search/focus")
			}
			s.openAliases(app)
			tester.Frame()
			s.aliasText = ""
			tester.Frame()
			tester.Click("保存别名")
			if len(core.SearchApplications(s.apps, "fixture", 10)) != 0 {
				t.Fatal("alias removal retained old search terms")
			}
		})
	}
}

func TestRecentsOnlyAfterSuccessfulLaunchAndMetadataSave(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	saveCalls := 0
	s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { saveCalls++; return value.Validated() }
	s.launchApplication = func(string) error { return errors.New("synthetic missing application") }
	s.activate(s.apps, nil, false)
	if saveCalls != 0 || len(s.config.Recents) != 0 {
		t.Fatal("failed launch recorded a recent")
	}
	s.launchApplication = func(string) error { return nil }
	s.activate(s.apps, nil, false)
	s.activate(s.apps, nil, false)
	if saveCalls != 2 || len(s.config.Recents) != 1 || s.config.Recents[0] != s.apps[0].ItemID() {
		t.Fatal("successful launch recents did not deduplicate")
	}
	old := s.config.Clone()
	s.applyPreferences = func(settings.Settings) (settings.Settings, error) {
		return settings.Settings{}, errors.New("synthetic save failure")
	}
	s.selected = 1
	s.activate(s.apps, nil, false)
	if !slices.Equal(old.Recents, s.config.Recents) {
		t.Fatal("failed persistence changed saved recents")
	}
	for i := 0; i < 60; i++ {
		s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
		s.recordRecent(core.Application{Path: fmt.Sprintf(`C:\Synthetic\Extra%02d.exe`, i)}.ItemID())
	}
	if len(s.config.Recents) != 50 {
		t.Fatal("recents are unbounded")
	}
}
