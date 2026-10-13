//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"slices"
)

func (s *launcher) toggleHiddenApplication(id string) {
	before := s.config.Clone()
	candidate := before.Clone()
	ids := s.applicationPreferenceIDs(id)
	hidden := false
	candidate.Hidden = slices.DeleteFunc(candidate.Hidden, func(value string) bool {
		if slices.Contains(ids, value) {
			hidden = true
			return true
		}
		return false
	})
	if !hidden {
		candidate.Hidden = append(candidate.Hidden, id)
	}
	if !s.saveCatalog(candidate) {
		return
	}
	saved := s.config.Clone()
	s.config = before
	s.refreshApplications(s.apps, s.icons, saved)
	s.status = "隐藏入口已更新；原文件未修改。"
}

func (s *launcher) openApplicationSource(app core.Application) {
	s.openSettings()
	detail := app.Name + "\n\n"
	if app.DisplayOnlyID != "" {
		detail += "来源：本地启动显示快照\n此身份尚未重新验证；没有可启动目标或文件位置。"
	} else if app.ShellID != "" {
		detail += "来源：AppsFolder（系统虚拟应用）\nShell 身份：" + app.ShellID + "\n此入口没有普通文件位置。"
	} else {
		detail += "来源文件：" + app.Path
		if app.Launch.Target != "" {
			detail += "\n目标：" + app.Launch.Target
		}
		if app.Launch.Arguments != "" {
			detail += "\n参数：" + app.Launch.Arguments
		}
		if app.Launch.WorkingDirectory != "" {
			detail += "\n工作目录：" + app.Launch.WorkingDirectory
		}
		detail += fmt.Sprintf("\n快捷方式独立语义：%t；标志：0x%08x；显示模式：%d", app.Launch.Independent, app.Launch.LinkFlags, app.Launch.ShowCommand)
	}
	s.statusDetailsOpen = true
	s.statusDetail = detail
	s.statusDetailScroll = ui.ScrollState{}
}

func (s *launcher) openHiddenApplications() {
	s.openSettings()
	s.hiddenOpen = true
	s.hiddenScroll = ui.ScrollState{}
}

func (s *launcher) drawHiddenApplications(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("hidden-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("隐藏入口背景")
	panel := ui.Column(c.Key("hidden-panel")).Absolute().Left(36).Right(36).Top(18).Bottom(18).Padding(16).Gap(8).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Role(ui.RoleDialog).Label("隐藏入口面板").Focusable()
	if s.settingsNeedsFocus {
		panel.Focus()
		s.settingsNeedsFocus = false
	}
	panel.Children(func() {
		ui.Text(c, "恢复隐藏入口").FontSize(20).Bold().Shrink(0)
		ui.Text(c, "只改变 MyCast 搜索结果；不修改或删除来源文件。").FontSize(12).TextColor(p.Secondary).Shrink(0)
		ui.Scroll(c.Key("hidden-scroll")).FillWidth().Grow(1).MinHeight(0).TrackScroll(&s.hiddenScroll).Label("隐藏入口列表").Children(func() {
			if len(s.config.Hidden) == 0 {
				ui.Text(c, "没有隐藏入口。").FontSize(14)
			}
			for _, id := range append([]string(nil), s.config.Hidden...) {
				name := "暂不可用的入口 · " + id[4:16]
				for _, app := range s.apps {
					if app.HasPreferenceID(id) {
						name = app.Name
						break
					}
				}
				key, label := id, name
				ui.Row(c.Key("hidden-row-" + key)).FillWidth().Height(38).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
					ui.Text(c, label).Grow(1).MinWidth(0).NoWrap().Ellipsis("…").FontSize(14)
					restore := ui.ButtonBase(c.Key("hidden-restore-" + key)).Width(64).Height(28).Radius(8).Background(p.Control).Label("恢复入口：" + label)
					restore.Children(func() { ui.Text(c, "恢复").FontSize(12) })
					if restore.Clicked() {
						s.toggleHiddenApplication(key)
					}
				})
			}
		})
		close := ui.ButtonBase(c.Key("hidden-close")).Height(32).Shrink(0).Padding(6, 14).Radius(8).Background(p.Control).Label("关闭隐藏入口")
		close.Children(func() { ui.Text(c, "完成").FontSize(14) })
		if close.Clicked() || c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}
