//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import "github.com/egoist/mygo/ui"

func (s *launcher) openStatusDetails() {
	if s.status == "" {
		return
	}
	s.openSettings()
	s.statusDetail = s.status // A stable in-memory snapshot, never persisted/logged.
	s.statusDetailsOpen = true
	s.statusDetailScroll.X, s.statusDetailScroll.Y = 0, 0
}

func (s *launcher) drawStatusDetails(c *ui.Context) {
	p := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	ui.ButtonBase(c.Key("status-detail-backdrop")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Background(ui.RGBA(0, 0, 0, .30)).Label("状态详情背景")
	panel := ui.Column(c.Key("status-detail-panel")).Absolute().Left(36).Right(36).Top(18).Bottom(18).Padding(16).Gap(8).Radius(18).Background(p.MenuSurface).Border(1, p.Border).Role(ui.RoleDialog).Label("状态详情面板").Focusable()
	if s.settingsNeedsFocus {
		panel.Focus()
		s.settingsNeedsFocus = false
	}
	panel.Children(func() {
		ui.Text(c, "完整状态").FontSize(20).Bold().Shrink(0)
		ui.Scroll(c.Key("status-detail-scroll")).FillWidth().Grow(1).MinHeight(0).Background(p.Control).Radius(8).Padding(10).TrackScroll(&s.statusDetailScroll).Label("状态详情滚动区域").Children(func() {
			ui.Text(c, s.statusDetail).FillWidth().Shrink(0).FontSize(p.BodySize).Selectable()
		})
		ui.Row(c).FillWidth().Height(32).Shrink(0).Justify(ui.End).Children(func() {
			close := ui.ButtonBase(c.Key("status-detail-close")).Height(32).Padding(6, 14).Radius(8).Background(p.Control).Border(1, p.Border).Label("关闭状态详情")
			close.Children(func() { ui.Text(c, "关闭").FontSize(p.BodySize) })
			if close.Clicked() {
				s.closeSettings()
			}
		})
		if c.Shortcut(0, ui.KeyEscape) {
			s.closeSettings()
		}
	})
}
