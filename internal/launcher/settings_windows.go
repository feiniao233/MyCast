//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/settings"
)

func (s *launcher) onHotkey() {
	if s.win.IsVisible() {
		s.hide()
	} else {
		s.show(true)
	}
}

func (s *launcher) applySettings(candidate settings.Settings) (settings.Settings, error) {
	value, err := candidate.Validated()
	if err != nil {
		return settings.Settings{}, err
	}
	if value.Hotkey == s.config.Hotkey {
		return s.saveOrdinarySettings(value)
	}
	if s.store == nil || s.shortcuts == nil {
		return settings.Settings{}, fmt.Errorf("settings storage is unavailable")
	}
	value, err = s.shortcuts.Apply(value, s.store.Save)
	if err != nil {
		return settings.Settings{}, err
	}
	return value, nil
}

func (s *launcher) saveOrdinarySettings(candidate settings.Settings) (settings.Settings, error) {
	value, err := candidate.Validated()
	if err != nil {
		return settings.Settings{}, err
	}
	if s.store == nil {
		return settings.Settings{}, fmt.Errorf("settings storage is unavailable")
	}
	if value.Hotkey != s.config.Hotkey {
		return settings.Settings{}, fmt.Errorf("hotkey changes require an explicit settings transition")
	}
	if err := s.store.Save(value); err != nil {
		return settings.Settings{}, err
	}
	return value, nil
}

func (s *launcher) openSettings() {
	s.utilityOpen, s.linkEditor = false, false
	s.clipPreview = ""
	s.previewOpen = false
	s.aliasID = ""
	s.closeMenu()
	s.settingsDraft = s.config.Clone()
	if s.settingsDraft.Version == 0 {
		s.settingsDraft = settings.Defaults()
	}
	s.hotkeyText = s.settingsDraft.Hotkey
	s.rootsText = strings.Join(s.settingsDraft.IndexRoots, "\n")
	s.themeText = map[string]string{"dark": "深色", "light": "浅色", "system": "跟随系统"}[s.settingsDraft.Theme]
	s.settingsNote = "剪贴板记录每次启动均关闭，文本历史只保留在内存。"
	s.settingsOpen, s.settingsNeedsFocus = true, true
}

func (s *launcher) closeSettings() {
	s.utilityOpen, s.linkEditor = false, false
	s.calcText, s.linksText = "", ""
	s.clipPreview = ""
	s.previewOpen = false
	s.settingsOpen, s.settingsNeedsFocus = false, false
	s.restoreSearch = true
}

func (s *launcher) saveSettings() {
	candidate := s.settingsDraft.Clone()
	candidate.Hotkey = s.hotkeyText
	candidate.Theme = map[string]string{"深色": "dark", "浅色": "light", "跟随系统": "system"}[s.themeText]
	candidate.IndexRoots = strings.Split(strings.ReplaceAll(s.rootsText, "\r\n", "\n"), "\n")
	value, err := candidate.Validated()
	if err == nil {
		if s.applyPreferences == nil {
			err = fmt.Errorf("settings storage is unavailable")
		} else {
			value, err = s.applyPreferences(value)
		}
	}
	if err != nil {
		s.settingsNote = "保存失败，当前设置与热键保留。" + err.Error()
		return
	}
	changedRoots := !slices.Equal(s.config.IndexRoots, value.IndexRoots)
	s.config = value.Clone()
	if s.tray != nil {
		s.tray.SetToolTip("MyCast · " + s.config.Hotkey)
	}
	s.status = "设置已保存。"
	s.closeSettings()
	if changedRoots {
		// Old scan completions are ignored; the new worker captures its own roots.
		s.scan()
	}
}

func (s *launcher) drawSettings(c *ui.Context) {
	if s.utilityOpen {
		s.drawUtilities(c)
		return
	}
	if s.previewOpen {
		s.drawClipPreview(c)
		return
	}
	if s.aliasID != "" {
		s.drawAliases(c)
		return
	}
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	// Modal catcher consumes covered results, including clicks outside the sheet.
	ui.ButtonBase(c.Key("settings-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("设置背景")
	ui.Column(c.Key("settings-sheet")).Absolute().Left(36).Right(36).Top(12).Bottom(12).Padding(16).Gap(4).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Label("设置面板").Role(ui.RoleDialog).Children(func() {
		ui.Row(c).FillWidth().Height(26).Shrink(0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "设置").FontSize(20).Bold()
			ui.Spacer(c)
			ui.Text(c, "仅此应用 · 不启用开机启动").FontSize(11).TextColor(p.Secondary)
		})
		ui.Text(c, "呼出 / 隐藏快捷键").FontSize(12).TextColor(p.Secondary)
		hotkey := ui.TextInputBase(c.Key("settings-hotkey"), &s.hotkeyText).FillWidth().Height(34).Shrink(0).Padding(6, 10).Radius(8).Background(p.Control).Border(1, p.Border).FontSize(14).Label("快捷键设置")
		if s.settingsNeedsFocus {
			hotkey.Focus()
			s.settingsNeedsFocus = false
		}
		ui.Text(c, "Ctrl / Alt / Shift + 字母、数字、Space 或 F1–F24").FontSize(11).TextColor(p.Secondary)
		ui.Row(c).FillWidth().Height(34).Shrink(0).Gap(16).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "外观").FontSize(12).TextColor(p.Secondary)
			ui.Select(c.Key("settings-theme"), &s.themeText, []string{"深色", "浅色", "跟随系统"}).Width(210).Height(32).Label("主题设置")
		})
		ui.Text(c, "应用目录 · 每行一个绝对本地路径；留空使用系统应用目录").FontSize(12).TextColor(p.Secondary)
		roots := ui.TextAreaBase(c.Key("settings-roots"), &s.rootsText).FillWidth().Height(56).Shrink(0).Padding(6, 10).Radius(8).Background(p.Control).Border(1, p.Border).FontSize(12).Label("应用目录设置")
		ui.Text(c, "仅索引 .lnk / .exe，不跟随符号链接，不读取文档内容。").FontSize(11).TextColor(p.Secondary)
		ui.Text(c, s.settingsNote).FillWidth().Height(40).Shrink(0).FontSize(11).TextColor(p.Secondary).Label("设置提示")
		ui.Spacer(c)
		ui.Row(c).FillWidth().Height(32).Shrink(0).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.ButtonBase(c.Key("settings-cancel")).Height(32).Padding(6, 16).Radius(8).Background(p.Control).Border(1, p.Border).Label("取消设置")
			cancel.Children(func() { ui.Text(c, "取消").FontSize(14) })
			if cancel.Clicked() {
				s.closeSettings()
			}
			save := ui.ButtonBase(c.Key("settings-save")).Height(32).Padding(6, 16).Radius(8).Background(p.Selection).Border(1, p.Border).Label("保存设置")
			save.Children(func() { ui.Text(c, "保存").FontSize(14).Bold() })
			if save.Clicked() && !hotkey.Composing() && !roots.Composing() {
				s.saveSettings()
			}
		})
		// Enter edits/submits fields only; it never launches a covered result.
		if !hotkey.Composing() && !roots.Composing() && c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}
