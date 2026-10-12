//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"strings"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
)

func (s *launcher) openUtilities() {
	s.openSettings()
	s.utilityOpen = true
	s.calcText, s.utilityNote = "", ""
	s.utilityScroll.X, s.utilityScroll.Y = 0, 0
}
func (s *launcher) editLinks() {
	var rows []string
	for _, link := range s.config.QuickLinks {
		rows = append(rows, link.Name+" | "+link.URL)
	}
	s.linksText = strings.Join(rows, "\n")
	s.linkEditor = true
	s.settingsNeedsFocus = true
	s.utilityNote = "链接保存到普通设置；只有点击打开时才交给默认浏览器。"
}
func (s *launcher) saveLinks() {
	if len(s.linksText) > 64*1024 {
		s.utilityNote = "链接输入过长。"
		return
	}
	candidate := s.config.Clone()
	candidate.QuickLinks = nil
	for _, line := range strings.Split(s.linksText, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "|", 2)
		if len(parts) != 2 {
			s.utilityNote = "每行使用：名称 | https://地址"
			return
		}
		candidate.QuickLinks = append(candidate.QuickLinks, core.QuickLink{Name: parts[0], URL: parts[1]})
		if len(candidate.QuickLinks) > 32 {
			s.utilityNote = "最多 32 个链接。"
			return
		}
	}
	value, err := candidate.Validated()
	if err != nil {
		s.utilityNote = "无法保存链接：" + err.Error()
		return
	}
	if !s.saveCatalog(value) {
		s.utilityNote = s.status
		return
	}
	s.linkEditor = false
	s.settingsNeedsFocus = true
	s.utilityNote = "快捷链接已保存。"
	s.utilityScroll.X, s.utilityScroll.Y = 0, 0
}

func (s *launcher) drawUtilities(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("utility-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("小工具背景")
	ui.Column(c.Key("utility-sheet")).Absolute().Left(36).Right(36).Top(12).Bottom(12).Padding(16).Gap(8).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Label("小工具面板").Role(ui.RoleDialog).Children(func() {
		if s.linkEditor {
			s.drawLinkEditor(c, p)
			return
		}
		ui.Text(c, "小工具").Height(26).Shrink(0).FontSize(20).Bold()
		field := ui.TextInputBase(c.Key("calculator"), &s.calcText).FillWidth().Height(34).Shrink(0).Padding(6, 10).Radius(8).Background(p.Control).Border(1, p.Border).FontSize(16).Placeholder("1 + 2 * (3 - 1)").Label("计算表达式")
		if s.settingsNeedsFocus {
			field.Focus()
			s.settingsNeedsFocus = false
		}
		result, err := core.Calculate(s.calcText)
		valid := err == nil && strings.TrimSpace(s.calcText) != ""
		ui.Row(c).FillWidth().Height(32).Shrink(0).Gap(8).AlignItems(ui.Center).Children(func() {
			label := "输入四则运算表达式"
			if strings.TrimSpace(s.calcText) != "" {
				if valid {
					label = "= " + result
				} else {
					label = "表达式无效"
				}
			}
			ui.Text(c, label).FontSize(16).Grow(1).Label("计算结果")
			copy := ui.ButtonBase(c.Key("copy-calculation")).Height(30).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("复制计算结果").Disabled(!valid || field.Composing())
			copy.Children(func() { ui.Text(c, "复制结果").FontSize(13) })
			if copy.Clicked() {
				if s.copyText == nil {
					s.utilityNote = "复制接口不可用。"
				} else if err := s.copyText(result); err != nil {
					s.utilityNote = "复制失败。"
				} else {
					s.sequence = platform.ClipboardSequence()
					s.utilityNote = "结果已复制。"
				}
			}
		})
		ui.Text(c, "四则运算与括号 · 最多 12 位有效数字 · 表达式不保存").FontSize(11).TextColor(p.Secondary)
		ui.Row(c).FillWidth().Height(30).Shrink(0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "快捷链接").FontSize(14).Bold()
			ui.Spacer(c)
			edit := ui.ButtonBase(c.Key("edit-links")).Height(28).Padding(4, 12).Radius(8).Background(p.Control).Border(1, p.Border).Label("编辑快捷链接")
			edit.Children(func() { ui.Text(c, "编辑链接").FontSize(12) })
			if edit.Clicked() {
				s.editLinks()
			}
		})
		ui.Scroll(c.Key("quick-link-list")).FillWidth().Grow(1).MinHeight(0).TrackScroll(&s.utilityScroll).Children(func() {
			ui.Column(c).FillWidth().Gap(4).Children(func() {
				if len(s.config.QuickLinks) == 0 {
					ui.Text(c, "尚无链接，在编辑中添加名称与 HTTP(S) 地址。").FontSize(12).TextColor(p.Secondary)
				}
				for _, link := range s.config.QuickLinks {
					entry := link
					row := ui.ButtonBase(c.Key(link.ItemID())).FillWidth().Height(44).Shrink(0).Padding(6, 10).Radius(8).Background(p.Control).Justify(ui.Start).Label("打开链接：" + link.Name).Tooltip(link.URL)
					row.Children(func() {
						ui.Column(c).FillWidth().Gap(2).Children(func() {
							ui.Text(c, entry.Name).FontSize(14).NoWrap().Ellipsis("…")
							ui.Text(c, entry.URL).FontSize(10).TextColor(p.Secondary).NoWrap().Ellipsis("…")
						})
					})
					if row.Clicked() {
						if _, err := entry.Validated(); err != nil || s.openURL == nil {
							s.utilityNote = "链接不可用。"
						} else if err := s.openURL(entry.URL); err != nil {
							s.utilityNote = "默认浏览器打开失败。"
						} else {
							s.utilityNote = "已交给默认浏览器。"
						}
					}
				}
			})
		})
		ui.Text(c, s.utilityNote).FillWidth().Height(24).Shrink(0).FontSize(11).TextColor(p.Secondary)
		ui.Row(c).FillWidth().Height(32).Shrink(0).Justify(ui.End).Children(func() {
			close := ui.ButtonBase(c.Key("utility-close")).Height(32).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("关闭小工具")
			close.Children(func() { ui.Text(c, "关闭").FontSize(14) })
			if close.Clicked() {
				s.closeSettings()
			}
		})
		if !field.Composing() && c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}

func (s *launcher) drawLinkEditor(c *ui.Context, p paletteTokens) {
	ui.Text(c, "编辑快捷链接").Height(26).Shrink(0).FontSize(20).Bold()
	ui.Text(c, "每行：名称 | https://地址 · 最多 32 条 · 留空删除全部").FontSize(12).TextColor(p.Secondary)
	field := ui.TextAreaBase(c.Key("link-editor"), &s.linksText).FillWidth().Grow(1).MinHeight(0).Padding(8).Radius(8).Background(p.Control).Border(1, p.Border).FontSize(13).Label("快捷链接内容")
	if s.settingsNeedsFocus {
		field.Focus()
		s.settingsNeedsFocus = false
	}
	ui.Text(c, s.utilityNote).FillWidth().Height(40).Shrink(0).FontSize(11).TextColor(p.Secondary)
	ui.Row(c).FillWidth().Height(32).Shrink(0).Gap(8).Justify(ui.End).Children(func() {
		cancel := ui.ButtonBase(c.Key("cancel-links")).Height(32).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("取消快捷链接编辑")
		cancel.Children(func() { ui.Text(c, "取消").FontSize(14) })
		if cancel.Clicked() {
			s.linkEditor = false
			s.settingsNeedsFocus = true
		}
		save := ui.ButtonBase(c.Key("save-links")).Height(32).Padding(6, 14).Radius(8).Background(p.Selection).Border(1, p.Border).Label("保存快捷链接")
		save.Children(func() { ui.Text(c, "保存").FontSize(14).Bold() })
		if save.Clicked() && !field.Composing() {
			s.saveLinks()
		}
	})
	if !field.Composing() && c.Shortcut(0, ui.KeyEscape) {
		s.linkEditor = false
		s.settingsNeedsFocus = true
	}
}
