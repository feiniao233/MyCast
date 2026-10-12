//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"slices"
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func (s *launcher) saveCatalog(candidate settings.Settings) bool {
	if s.applyPreferences == nil {
		s.status = "应用偏好暂时无法保存。"
		return false
	}
	value, err := s.applyPreferences(candidate)
	if err != nil {
		s.status = "应用偏好未保存，原设置保留。" + err.Error()
		return false
	}
	s.config = value.Clone()
	s.apps = core.WithUserAliases(s.apps, s.config.Aliases)
	return true
}

func (s *launcher) toggleFavorite(id string) {
	candidate := s.config.Clone()
	if index := slices.Index(candidate.Favorites, id); index >= 0 {
		candidate.Favorites = slices.Delete(candidate.Favorites, index, index+1)
	} else {
		candidate.Favorites = append(candidate.Favorites, id)
	}
	if s.saveCatalog(candidate) {
		s.resetResults()
		s.status = "收藏已更新。"
	}
}

func (s *launcher) recordRecent(id string) {
	candidate := s.config.Clone()
	if index := slices.Index(candidate.Recents, id); index >= 0 {
		candidate.Recents = slices.Delete(candidate.Recents, index, index+1)
	}
	candidate.Recents = append([]string{id}, candidate.Recents...)
	if len(candidate.Recents) > 50 {
		candidate.Recents = candidate.Recents[:50]
	}
	s.saveCatalog(candidate)
}

func (s *launcher) openAliases(app core.Application) {
	s.openSettings()
	s.aliasID, s.aliasName = app.ItemID(), app.Name
	s.aliasText = strings.Join(s.config.Aliases[s.aliasID], "\n")
	s.settingsNote = "别名只用于搜索，不修改原快捷方式或启动参数。"
}

func (s *launcher) saveAliases() {
	candidate := s.config.Clone()
	if candidate.Aliases == nil {
		candidate.Aliases = map[string][]string{}
	}
	values := strings.Split(strings.ReplaceAll(s.aliasText, "\r\n", "\n"), "\n")
	var aliases []string
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			aliases = append(aliases, value)
		}
	}
	if len(aliases) == 0 {
		delete(candidate.Aliases, s.aliasID)
	} else {
		candidate.Aliases[s.aliasID] = aliases
	}
	value, err := candidate.Validated()
	if err != nil {
		s.settingsNote = "无法保存别名：" + err.Error()
		return
	}
	if !s.saveCatalog(value) {
		s.settingsNote = s.status
		return
	}
	s.status = "搜索别名已保存。"
	s.resetResults()
	s.closeSettings()
}

func (s *launcher) drawAliases(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("alias-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("别名背景")
	ui.Column(c.Key("alias-sheet")).Absolute().Left(60).Right(60).Top(36).Bottom(36).Padding(20).Gap(12).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Label("别名面板").Role(ui.RoleDialog).Children(func() {
		ui.Text(c, "编辑搜索别名").FontSize(20).Bold()
		ui.Text(c, s.aliasName).FontSize(14).NoWrap().Ellipsis("…")
		ui.Text(c, "每行一个；最多 8 个，每个最多 80 字。留空移除。").FontSize(12).TextColor(p.Secondary)
		field := ui.TextAreaBase(c.Key("alias-input"), &s.aliasText).FillWidth().Height(150).Shrink(0).Padding(8).Radius(8).Background(p.Control).Border(1, p.Border).FontSize(14).Label("应用别名")
		if s.settingsNeedsFocus {
			field.Focus()
			s.settingsNeedsFocus = false
		}
		ui.Text(c, s.settingsNote).FillWidth().Height(40).Shrink(0).FontSize(11).TextColor(p.Secondary)
		ui.Spacer(c)
		ui.Row(c).FillWidth().Height(32).Shrink(0).Gap(8).Justify(ui.End).Children(func() {
			cancel := ui.ButtonBase(c.Key("alias-cancel")).Height(32).Padding(6, 16).Radius(8).Background(p.Control).Border(1, p.Border).Label("取消别名")
			cancel.Children(func() { ui.Text(c, "取消").FontSize(14) })
			if cancel.Clicked() {
				s.closeSettings()
			}
			save := ui.ButtonBase(c.Key("alias-save")).Height(32).Padding(6, 16).Radius(8).Background(p.Selection).Border(1, p.Border).Label("保存别名")
			save.Children(func() { ui.Text(c, "保存").FontSize(14).Bold() })
			if save.Clicked() && !field.Composing() {
				s.saveAliases()
			}
		})
		if !field.Composing() && c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}
