//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"path/filepath"
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
			s.persistPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
			app := s.apps[0]
			tester := newPaletteTester(s, scale)
			tester.Key(ui.Shift, ui.KeyF10)
			tester.Click("菜单项：收藏应用")
			if !slices.Contains(s.config.Favorites, app.ItemID()) {
				t.Fatal("favorite action failed")
			}
			tester.Key(ui.Shift, ui.KeyF10)
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

func TestUnavailableStartupHotkeyDoesNotBlockMetadata(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.store = &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	registerCalls := 1 // The startup registration failed, leaving Current empty.
	s.shortcuts = &settings.Shortcuts{Register: func(string) error { registerCalls++; return errors.New("synthetic hotkey occupied") }, Unregister: func(string) { t.Fatal("unavailable old hotkey was unregistered") }}
	s.persistPreferences = s.saveOrdinarySettings
	s.applyPreferences = s.applySettings
	app := s.apps[0]
	s.toggleFavorite(app.ItemID())
	s.openAliases(app)
	s.aliasText = "dedicated alias"
	s.saveAliases()
	s.launchApplication = func(string) error { return nil }
	s.activate(s.apps, nil, false)
	if registerCalls != 1 || len(s.config.Favorites) != 1 || len(s.config.Recents) != 1 || len(s.config.Aliases[app.ItemID()]) != 1 {
		t.Fatal("metadata retried unavailable hotkey or failed to save")
	}
	loaded, err := (&settings.Store{Path: s.store.Path}).Load()
	if err != nil || len(loaded.Favorites) != 1 || len(loaded.Recents) != 1 || len(loaded.Aliases[app.ItemID()]) != 1 {
		t.Fatal("metadata not persisted while hotkey unavailable")
	}
	theme := s.config.Clone()
	theme.Theme = "light"
	if _, err := s.applySettings(theme); err != nil || registerCalls != 1 {
		t.Fatal("unrelated theme change retried unavailable hotkey")
	}
	changed := s.config.Clone()
	changed.Hotkey = "Ctrl+F12"
	if _, err := s.applySettings(changed); err == nil || registerCalls != 2 || s.shortcuts.Current != "" {
		t.Fatal("explicit hotkey change bypassed registration failure")
	}
}

func TestNoTrayLaunchReorderKeepsSelectedIdentityOnRealEnter(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.apps = []core.Application{{Name: "A", Path: `C:\Synthetic\A.exe`}, {Name: "B", Path: `C:\Synthetic\B.exe`}, {Name: "C", Path: `C:\Synthetic\C.exe`}}
			s.persistPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
			var launched []string
			s.launchApplication = func(path string) error { launched = append(launched, path); return nil }
			tester := newPaletteTester(s, scale)
			tester.Key(0, ui.KeyDown)
			if s.selected != 1 {
				t.Fatal("keyboard did not select non-first app")
			}
			tester.Key(0, ui.KeyEnter)
			tester.Frame()
			ranked := core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents)
			if s.tray != nil || s.selected != 0 || ranked[s.selected].Name != "B" {
				t.Fatal("visible no-tray launcher lost selected identity after recent reorder")
			}
			box, ok := tester.Find("B")
			if !ok || box.Y < palette.Header || box.Y+box.H > palette.Height-palette.Footer {
				t.Fatal("retained selection not visible")
			}
			tester.Key(0, ui.KeyEnter)
			if len(launched) != 2 || launched[0] != `C:\Synthetic\B.exe` || launched[1] != `C:\Synthetic\B.exe` {
				t.Fatalf("second real Enter launched another item: %v", launched)
			}
		})
	}
}

func TestRecentsOnlyAfterSuccessfulLaunchAndMetadataSave(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	saveCalls := 0
	s.persistPreferences = func(value settings.Settings) (settings.Settings, error) { saveCalls++; return value.Validated() }
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
	s.persistPreferences = func(settings.Settings) (settings.Settings, error) {
		return settings.Settings{}, errors.New("synthetic save failure")
	}
	s.selected = 1
	s.activate(s.apps, nil, false)
	if !slices.Equal(old.Recents, s.config.Recents) {
		t.Fatal("failed persistence changed saved recents")
	}
	for i := 0; i < 60; i++ {
		s.persistPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
		s.recordRecent(core.Application{Path: fmt.Sprintf(`C:\Synthetic\Extra%02d.exe`, i)}.ItemID())
	}
	if len(s.config.Recents) != 50 {
		t.Fatal("recents are unbounded")
	}
}
