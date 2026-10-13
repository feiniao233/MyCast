//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestApplicationRowContextMenuTargetsStableScrolledIdentity(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.persistPreferences = func(v settings.Settings) (settings.Settings, error) { return v, nil }
			launches := 0
			s.launchApplication = func(string) error { launches++; return nil }
			tester := newPaletteTester(s, scale)
			if _, ok := tester.Find("操作"); ok {
				t.Fatal("application footer action button remains")
			}
			tester.Scroll(palette.Width/2, palette.Header+palette.SectionHeight+palette.RowHeight, 0, 260)
			apps := s.applicationResults()
			rows := s.rows(apps, nil)
			var chosen core.Application
			for i, row := range rows {
				box, ok := tester.Find("结果：" + row.title)
				if ok && box.Y >= palette.Header+palette.Inset && box.Y+box.H < palette.Height-palette.Footer {
					chosen = apps[i]
					break
				}
			}
			if chosen.Path == "" {
				t.Fatal("no scrolled visible row")
			}
			if err := tester.RightClick("结果：" + chosen.Name); err != nil {
				t.Fatal(err)
			}
			if s.menu != menuActions || s.menuTargetKey != chosen.ItemID() || launches != 0 {
				t.Fatal("right click selected/activated wrong identity")
			}
			for _, label := range []string{"菜单项：收藏应用", "菜单项：查看来源", "菜单项：隐藏此入口"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < palette.Header || box.Y+box.H > palette.Height-palette.Footer {
					t.Fatal("context item clipped", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("context-menu-%03d", int(scale*100)))
			// Ranking and mutable selected index cannot change the captured effect.
			s.selected = 0
			s.config.Recents = []string{s.apps[len(s.apps)-1].ItemID()}
			tester.Frame()
			if err := tester.Click("菜单项：收藏应用"); err != nil {
				t.Fatal(err)
			}
			if !chosen.Preferred(s.config.Favorites) || launches != 0 {
				t.Fatal("favorite moved to replacement index or launched")
			}
			if !tester.Focused("搜索") {
				t.Fatal("context close lost search focus")
			}
			// Reopen, then remove its identity: a click at the old menu coordinates
			// must be consumed without operating a new result underneath.
			s.openRowMenu(chosen.ItemID(), palette.Width-palette.Inset, palette.Height-palette.Footer)
			tester.Frame()
			box, _ := tester.Find("菜单项：收藏应用")
			if box.H == 0 {
				box, _ = tester.Find("菜单项：取消收藏")
			}
			s.apps = deleteSyntheticID(s.apps, chosen.ItemID())
			tester.ClickAt(box.X+box.W/2, box.Y+box.H/2)
			if s.menu != menuClosed || launches != 0 {
				t.Fatal("disappearing target transferred old click")
			}
			s.runMenuTarget(chosen.ItemID(), false, "primary")
			if launches != 0 {
				t.Fatal("stale context closure launched replacement")
			}
		})
	}
}

func deleteSyntheticID(apps []core.Application, id string) []core.Application {
	result := make([]core.Application, 0, len(apps))
	for _, app := range apps {
		if app.ItemID() != id {
			result = append(result, app)
		}
	}
	return result
}

func TestContextMenuKeyboardEmptyAndDismissal(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.apps = s.apps[:3]
			launches := 0
			s.launchApplication = func(string) error { launches++; return nil }
			tester := newPaletteTester(s, scale)
			for _, key := range []struct {
				mods ui.Modifiers
				key  ui.Key
			}{{ui.Shift, ui.KeyF10}} {
				tester.Key(key.mods, key.key)
				if s.menu != menuActions || s.menuTargetKey != s.applicationResults()[s.selected].ItemID() {
					t.Fatalf("keyboard context did not bind selected identity: key=%v mods=%v menu=%v target=%q nativeMenu=%v query=%q searchFocus=%t composing=%t selectionGuard=%t settings=%t", key.key, key.mods, s.menu, s.menuTargetKey, tester.Menu(), s.query, tester.Focused("搜索"), s.searchWasComposing, s.appNeedsSelection, s.settingsOpen)
				}
				tester.Key(0, ui.KeyEscape)
				if s.menu != menuClosed || !tester.Focused("搜索") || launches != 0 {
					t.Fatal("keyboard dismissal launched/lost focus")
				}
			}
			tester.RightClickAt(palette.Width/2, palette.Height-palette.Footer-10)
			if s.menu != menuClosed || launches != 0 {
				t.Fatal("empty space opened a row menu")
			}
			tester.RightClick("结果：" + s.apps[0].Name)
			covered, _ := tester.Find("结果：" + s.apps[1].Name)
			tester.ClickAt(covered.X+palette.Inset, covered.Y+covered.H/2)
			if s.menu != menuClosed || launches != 0 {
				t.Fatal("click-away launched covered result")
			}
			s.apps = nil
			tester.Frame()
			tester.Key(ui.Shift, ui.KeyF10)
			if s.menu != menuClosed {
				t.Fatal("empty result keyboard menu")
			}
		})
	}
}

func TestTabFocusedRowOwnsBothKeyboardContextEntrances(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		for _, key := range []struct {
			mods ui.Modifiers
			key  ui.Key
		}{{0, ui.KeyContextMenu}, {ui.Shift, ui.KeyF10}} {
			t.Run(fmt.Sprintf("%v/%v", scale, key.key), func(t *testing.T) {
				s := syntheticLauncher()
				s.apps = s.apps[:3]
				s.config = settings.Defaults()
				s.persistPreferences = func(v settings.Settings) (settings.Settings, error) { return v, nil }
				launches := 0
				s.launchApplication = func(string) error { launches++; return nil }
				tester := newPaletteTester(s, scale)
				target := s.apps[1]
				for i := 0; i < 20 && !tester.Focused("结果："+target.Name); i++ {
					tester.Key(ui.Shift, ui.KeyTab)
				}
				if !tester.Focused("结果："+target.Name) || s.selected != 0 {
					t.Fatal("nonselected row did not receive Tab focus")
				}
				tester.Key(key.mods, key.key)
				if s.menu != menuActions || s.menuTargetKey != target.ItemID() || launches != 0 {
					t.Fatal("keyboard context used mutable selected index")
				}
				tester.Click("菜单项：收藏应用")
				if !target.Preferred(s.config.Favorites) || s.apps[0].Preferred(s.config.Favorites) || launches != 0 || !tester.Focused("搜索") {
					t.Fatal("row context effect/focus transferred")
				}
			})
		}
	}
}

func TestSearchMouseContextRetainsTextEditingMenu(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			tester := newPaletteTester(s, scale)
			tester.RightClick("搜索")
			items := tester.Menu()
			if s.menu != menuClosed || len(items) == 0 {
				t.Fatal("search right-click intercepted into application actions")
			}
			for _, label := range items {
				if label == "收藏应用" || label == "查看来源" || label == "打开所选" {
					t.Fatal("application item in editor menu")
				}
			}
			tester.CloseMenu()
		})
	}
}
