//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"slices"
	"time"

	"github.com/egoist/mygo/ui"
)

func (s *launcher) clearHistory() {
	s.history.Clear()
	s.clipSelection = ""
	if s.previewOpen {
		s.closeSettings()
	}
	if s.clipboardMode {
		s.resetResults()
	}
}
func (s *launcher) expireHistory() {
	s.history.Expire()
	if s.previewOpen && !slices.Contains(s.history.Entries(), s.clipPreview) {
		s.closeSettings()
		s.status = "预览条目已到期。"
	}
}
func (s *launcher) deleteClip(text string) {
	s.history.Delete(text)
	s.clipSelection = ""
	s.resetResults()
	if s.previewOpen {
		s.closeSettings()
	}
	s.status = "文本条目已从内存移除。"
}
func (s *launcher) retentionLabel() string {
	switch s.retention {
	case 10 * time.Minute:
		return "保留：10 分钟"
	case time.Hour:
		return "保留：1 小时"
	default:
		return "保留：退出前"
	}
}
func (s *launcher) cycleRetention() {
	switch s.retention {
	case 0:
		s.retention = 10 * time.Minute
	case 10 * time.Minute:
		s.retention = time.Hour
	default:
		s.retention = 0
	}
	s.history.SetRetention(s.retention)
	s.expireHistory()
	s.status = s.retentionLabel() + "，仅本次会话；不保存到磁盘。"
}
func (s *launcher) openClipPreview(text string) {
	s.previewScroll.X, s.previewScroll.Y = 0, 0
	s.closeMenu()
	s.clipPreview = text
	s.previewOpen, s.settingsOpen, s.settingsNeedsFocus = true, true, true
	s.settingsNote = "全文预览 · 仅在内存；复制会写入系统剪贴板。"
}

func (s *launcher) drawClipPreview(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("clip-preview-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("文本预览背景")
	panel := ui.Column(c.Key("clip-preview-sheet")).Absolute().Left(36).Right(36).Top(18).Bottom(18).Padding(16).Gap(8).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Label("文本预览面板").Role(ui.RoleDialog).Focusable()
	if s.settingsNeedsFocus {
		panel.Focus()
		s.settingsNeedsFocus = false
	}
	panel.Children(func() {
		ui.Row(c).FillWidth().Height(26).Shrink(0).AlignItems(ui.Center).Children(func() {
			ui.Text(c, "文本预览").FontSize(20).Bold()
			ui.Spacer(c)
			ui.Text(c, fmt.Sprintf("%d 字节", len(s.clipPreview))).FontSize(11).TextColor(p.Secondary)
		})
		ui.Scroll(c.Key("clip-preview-scroll")).FillWidth().Grow(1).MinHeight(0).Background(p.Control).Radius(8).Padding(10).TrackScroll(&s.previewScroll).Label("预览全文").Children(func() {
			ui.Text(c, s.clipPreview).FillWidth().Shrink(0).FontSize(14).Selectable()
		})
		ui.Text(c, s.settingsNote).FillWidth().Height(28).Shrink(0).FontSize(11).TextColor(p.Secondary)
		ui.Row(c).FillWidth().Height(32).Shrink(0).Gap(8).Children(func() {
			del := ui.ButtonBase(c.Key("clip-preview-delete")).Height(32).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("删除预览条目")
			del.Children(func() { ui.Text(c, "删除此条").FontSize(14) })
			if del.Clicked() {
				s.deleteClip(s.clipPreview)
			}
			ui.Spacer(c)
			close := ui.ButtonBase(c.Key("clip-preview-close")).Height(32).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("关闭文本预览")
			close.Children(func() { ui.Text(c, "关闭").FontSize(14) })
			if close.Clicked() {
				s.closeSettings()
			}
			copy := ui.ButtonBase(c.Key("clip-preview-copy")).Height(32).Padding(6, 14).Radius(8).Background(p.Selection).Border(1, p.Border).Label("复制预览文本")
			copy.Children(func() { ui.Text(c, "复制全文").FontSize(14).Bold() })
			if copy.Clicked() && s.previewOpen {
				selected := s.selected
				s.selected = 0
				s.activate(nil, []string{s.clipPreview}, true)
				s.selected = selected
				s.settingsNote = s.status
			}
		})
		if c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}
