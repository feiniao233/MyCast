//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/settings"
)

// Select's selected label also appears on its trigger. Infer the three option
// centers from the two unambiguous rendered labels, never from hardcoded Y/DPI.
func themeOptionCenters(t *testing.T, tester *ui.Tester, selected string) [3]ui.Point {
	t.Helper()
	labels := []string{"深色", "浅色", "跟随系统"}
	indices := []int{}
	boxes := []ui.Rect{}
	for i, label := range labels {
		if label == selected {
			continue
		}
		r, ok := tester.Find(label)
		if !ok {
			t.Fatalf("missing expanded option %s", label)
		}
		indices, boxes = append(indices, i), append(boxes, r)
	}
	step := (boxes[1].Y - boxes[0].Y) / float32(indices[1]-indices[0])
	if step <= 0 {
		t.Fatal("options not independently laid out")
	}
	var points [3]ui.Point
	for i := range points {
		points[i] = ui.Point{X: boxes[0].X + boxes[0].W/2, Y: boxes[0].Y + boxes[0].H/2 + float32(i-indices[0])*step}
	}
	return points
}

func TestSettingsThemePopupVisualAndInteraction(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("dark-%v-dpi-%03d", dark, int(scale*100)), func(t *testing.T) {
				s := syntheticLauncher()
				s.config = settings.Defaults()
				if !dark {
					s.config.Theme = "light"
				}
				s.query = "Synthetic"
				launched, saved := 0, 0
				s.launchApplication = func(string) error { launched++; return nil }
				s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { saved++; return value, nil }
				tester := newPaletteTester(s, scale)
				before := s.config.Clone()
				for option, label := range []string{"深色", "浅色", "跟随系统"} {
					s.openSettings()
					tester.Frame()
					original := s.themeText
					if err := tester.Click("主题设置"); err != nil {
						t.Fatal(err)
					}
					points := themeOptionCenters(t, tester, original)
					trigger, _ := tester.Find("主题设置")
					for _, point := range points {
						if point.Y <= trigger.Y+trigger.H || point.Y >= palette.Height-12 {
							t.Fatalf("popup placement: %+v below %+v", point, trigger)
						}
					}
					// Sample unselected row padding, which must be the opaque menu
					// surface, not the roots field or settings sheet underneath.
					unselected := 1
					if original == "浅色" {
						unselected = 2
					}
					p := paletteFor(dark, false)
					pixel := tester.Image().RGBAAt(int((trigger.X+trigger.W-3)*scale), int(points[unselected].Y*scale))
					if pixel.A != 255 || pixel.R != p.MenuSurface.R || pixel.G != p.MenuSurface.G || pixel.B != p.MenuSurface.B {
						t.Fatalf("popup background: %+v want %+v", pixel, p.MenuSurface)
					}
					theme := launcherTheme(p, dark)
					if theme.Background.A != 255 || theme.Surface.A != 255 || theme.Accent == theme.AccentText {
						t.Fatal("semantic theme surfaces/selected contrast corrupted")
					}
					if option == 0 {
						savePreview(t, tester, fmt.Sprintf("feedback-dropdown-dark-%v-%03d", dark, int(scale*100)))
					}
					tester.ClickAt(points[option].X, points[option].Y)
					if s.themeText != label || !s.settingsOpen || launched != 0 || saved != 0 || !reflect.DeepEqual(before, s.config) {
						t.Fatalf("option escaped draft: %s %s", label, s.themeText)
					}
					tester.Key(0, ui.KeyEnter)
					if saved != 0 || launched != 0 {
						t.Fatal("select Enter leaked to save or launcher")
					}
					// Enter on the focused trigger legitimately reopens its popup.
					// Dismiss that popup before clicking the settings Cancel button.
					tester.Key(0, ui.KeyEscape)
					if !s.settingsOpen {
						t.Fatal("popup Escape closed settings")
					}
					tester.Click("取消设置")
					if s.settingsOpen || !reflect.DeepEqual(before, s.config) || !tester.Focused("搜索") {
						t.Fatal("cancel changed live state/focus")
					}
				}
				s.openSettings()
				tester.Frame()
				tester.Click("主题设置")
				tester.Key(0, ui.KeyEscape)
				if !s.settingsOpen {
					t.Fatal("popup Escape closed settings")
				}
				// Click-away is consumed by the Select modal backdrop. It must
				// neither focus/type in roots nor save/cancel/launch underneath.
				tester.Click("主题设置")
				roots, _ := tester.Find("应用目录设置")
				tester.ClickAt(roots.X+roots.W-4, roots.Y+roots.H-4)
				if !s.settingsOpen || saved != 0 || launched != 0 || tester.Focused("应用目录设置") {
					t.Fatal("click-away penetrated popup")
				}
				tester.Click("主题设置")
				points := themeOptionCenters(t, tester, s.themeText)
				tester.ClickAt(points[1].X, points[1].Y)
				tester.Click("保存设置")
				if s.settingsOpen || s.config.Theme != "light" || saved != 1 || launched != 0 || !tester.Focused("搜索") {
					t.Fatal("selected theme save/focus failed")
				}
				s.openSettings()
				tester.Frame()
				if s.themeText != "浅色" {
					t.Fatal("saved option not retained on reopen")
				}
				// Keyboard selection operates on the popup only, with the shared
				// search/launcher and save actions still covered.
				tester.Click("主题设置")
				tester.Key(0, ui.KeyDown)
				tester.Key(0, ui.KeyUp)
				tester.Key(0, ui.KeyEnter)
				if s.themeText != "浅色" || saved != 1 || launched != 0 {
					t.Fatal("popup keyboard action leaked")
				}
				tester.Click("取消设置")
				s.openSettings()
				tester.Frame()
				tester.Compose("合成", 2)
				tester.Key(0, ui.KeyEscape)
				if !s.settingsOpen {
					t.Fatal("IME candidate Escape dismissed settings")
				}
			})
		}
	}
}

func TestNativeThemeSyncOnlyAfterSuccessfulSave(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	var synced []string
	s.syncNativeTheme = func(theme string) { synced = append(synced, theme) }
	s.openSettings()
	s.themeText = "浅色"
	s.closeSettings()
	if len(synced) != 0 {
		t.Fatal("cancel changed native theme")
	}
	s.openSettings()
	s.themeText = "浅色"
	s.applyPreferences = func(settings.Settings) (settings.Settings, error) {
		return settings.Settings{}, errors.New("synthetic storage failure")
	}
	s.saveSettings()
	if len(synced) != 0 {
		t.Fatal("failed save changed native theme")
	}
	s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { return value, nil }
	s.saveSettings()
	if !reflect.DeepEqual(synced, []string{"light"}) {
		t.Fatalf("native theme sync: %v", synced)
	}
	for name, want := range map[string]mygo.ThemeSource{"dark": mygo.ThemeDark, "light": mygo.ThemeLight, "system": mygo.ThemeSystem} {
		if nativeThemeSource(name) != want {
			t.Fatal("invalid native theme mapping")
		}
	}
}
