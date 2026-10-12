//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"reflect"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestSettingsModalFocusCancelAndFailedSave(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.query = "Synthetic"
			before := s.config.Clone()
			s.recording = true
			s.history.Add("Dedicated synthetic settings fixture")
			calls := 0
			s.applyPreferences = func(settings.Settings) (settings.Settings, error) {
				calls++
				return settings.Settings{}, errors.New("synthetic unavailable hotkey")
			}
			tester := newPaletteTester(s, scale)
			if err := tester.Click("菜单"); err != nil {
				t.Fatal(err)
			}
			if err := tester.Click("菜单项：设置"); err != nil {
				t.Fatal(err)
			}
			if !s.settingsOpen || !tester.Focused("快捷键设置") {
				t.Fatal("settings did not open with editable field focus")
			}
			for _, label := range []string{"快捷键设置", "应用目录设置", "保存设置", "取消设置"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < 12 || box.Y+box.H > palette.Height-12 {
					t.Fatalf("settings control clipped: %s %+v", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("settings-%03d", int(scale*100)))
			tester.Key(0, ui.KeyEnter)
			if !s.settingsOpen || calls != 0 || s.status != "" {
				t.Fatal("field Enter launched or saved covered state")
			}
			s.hotkeyText = "invalid"
			tester.Frame()
			tester.Click("保存设置")
			if calls != 0 || !s.settingsOpen || !reflect.DeepEqual(before, s.config) {
				t.Fatal("invalid draft was applied")
			}
			s.hotkeyText = "Ctrl+F12"
			tester.Frame()
			tester.Click("保存设置")
			if calls != 1 || !s.settingsOpen || !reflect.DeepEqual(before, s.config) || !s.recording || s.history.Len() != 1 {
				t.Fatal("failed save changed live settings or session history")
			}
			tester.Click("取消设置")
			if s.settingsOpen || !tester.Focused("搜索") || s.query != "Synthetic" {
				t.Fatal("cancel lost search state/focus")
			}
			s.openSettings()
			tester.Frame()
			tester.Compose("合成", 2)
			tester.Key(0, ui.KeyEscape)
			if !s.settingsOpen {
				t.Fatal("IME Escape cancelled settings")
			}
		})
	}
}

func TestSettingsSaveAndThemeContrast(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.recording = true
	s.history.Add("Dedicated synthetic retained text")
	s.applyPreferences = func(value settings.Settings) (settings.Settings, error) { return value, nil }
	tester := newPaletteTester(s, 1)
	s.openSettings()
	tester.Frame()
	s.hotkeyText = "alt+ctrl+f12"
	s.themeText = "浅色"
	s.rootsText = `C:\Synthetic\Apps`
	tester.Frame()
	tester.Click("保存设置")
	if s.settingsOpen || s.config.Hotkey != "Ctrl+Alt+F12" || s.config.Theme != "light" || !s.recording || s.history.Len() != 1 || !tester.Focused("搜索") {
		t.Fatal("save did not apply ordinary settings separately from clipboard state")
	}
	if pixel := tester.Image().RGBAAt(375, 410); pixel.R < 200 {
		t.Fatal("light palette not applied")
	}
	savePreview(t, tester, "launcher-light-100")
	s.config.Theme = "system"
	tester.SetDark(false)
	tester.Frame()
	if pixel := tester.Image().RGBAAt(375, 410); pixel.R < 200 {
		t.Fatal("system light theme ignored")
	}
	tester.SetDark(true)
	tester.Frame()
	if pixel := tester.Image().RGBAAt(375, 410); pixel.R > 80 {
		t.Fatal("system dark theme ignored")
	}
	tester.SetVibrancy(true)
	tester.SetPreferences(ui.Preferences{HighContrast: true, TextScale: 1})
	if pixel := tester.Image().RGBAAt(375, 410); pixel.A != 255 {
		t.Fatal("high contrast material is translucent")
	}
	p := paletteFor(true, true)
	if p.Secondary != p.Text || p.Tertiary != p.Text {
		t.Fatal("high contrast muted text lost readability")
	}
	savePreview(t, tester, "launcher-high-contrast-100")
}
