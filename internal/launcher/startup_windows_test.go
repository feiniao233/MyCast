//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"github.com/feiniao233/MyCast/internal/settings"
	"path/filepath"
	"testing"
)

func TestStartupControlCancelAndExplicitSave(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			calls := 0
			s.applyPreferences = func(v settings.Settings) (settings.Settings, error) { calls++; return v, nil }
			tester := newPaletteTester(s, scale)
			s.openSettings()
			tester.Frame()
			if err := tester.Click("开机启动"); err != nil {
				t.Fatal(err)
			}
			if !s.settingsDraft.Startup || s.config.Startup || calls != 0 {
				t.Fatal("draft performed registration")
			}
			savePreview(t, tester, fmt.Sprintf("startup-settings-%03d", int(scale*100)))
			tester.Click("取消设置")
			if s.config.Startup || calls != 0 {
				t.Fatal("cancel enabled startup")
			}
			s.openSettings()
			tester.Frame()
			tester.Click("开机启动")
			tester.Click("保存设置")
			if calls != 1 || !s.config.Startup || s.settingsOpen {
				t.Fatal("explicit save did not apply preference")
			}
		})
	}
}

func TestStartupAndHotkeyConflictRollbackAndOrdinarySaveNoRegistration(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.store = &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if _, err := s.store.Load(); err != nil {
		t.Fatal(err)
	}
	registered := false
	reads := 0
	s.startupRegistration = &settings.StartupRegistration{Command: "owned", Read: func() (string, bool, error) { reads++; return "owned", registered, nil }, Write: func(string) error { registered = true; return nil }, Delete: func() error { registered = false; return nil }}
	ordinary := s.config.Clone()
	ordinary.Theme = "light"
	if _, err := s.applySettings(ordinary); err != nil || reads != 0 || registered {
		t.Fatal("ordinary save touched startup registration")
	}
	candidate := s.config.Clone()
	candidate.Startup = true
	candidate.Hotkey = "Ctrl+F12"
	s.shortcuts = &settings.Shortcuts{Current: s.config.Hotkey, Register: func(string) error { return errors.New("synthetic occupied") }, Unregister: func(string) { t.Fatal("old hotkey unregistered") }}
	if _, err := s.applySettings(candidate); err == nil || registered {
		t.Fatal("hotkey failure left startup enabled")
	}
}

func TestBackgroundInvocationIsExplicit(t *testing.T) {
	if !backgroundInvocation([]string{"--background"}) {
		t.Fatal("background flag absent")
	}
	for _, args := range [][]string{nil, {"--other"}, {"--background", "unexpected"}} {
		if backgroundInvocation(args) {
			t.Fatal("unrequested background mode")
		}
	}
}
