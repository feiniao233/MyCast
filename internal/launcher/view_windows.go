//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/plugins/glass"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
)

type paletteMenu uint8

const (
	menuClosed paletteMenu = iota
	menuApplication
	menuActions
)

type paletteRow struct {
	title, subtitle, kind, key string
	bitmap                     *ui.Bitmap
	height, top                float32
}
type paletteMenuItem struct {
	text, glyph string
	disabled    bool
	run         func()
}

var searchGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="24" height="24" viewBox="0 0 24 24"><circle cx="10" cy="10" r="6.5" fill="none" stroke="currentColor" stroke-width="1.8"/><path d="M15 15L21 21" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"/></svg>`))

func (s *launcher) resetResults() { s.selected, s.lastScroll = 0, 0; s.scroll.X, s.scroll.Y = 0, 0 }
func (s *launcher) switchMode(clipboard bool) {
	if s.clipboardMode != clipboard {
		s.clipboardMode, s.query = clipboard, ""
		s.resetResults()
	}
	s.restoreSearch = true
}
func (s *launcher) closeMenu() { s.menu = menuClosed; s.restoreSearch = true }
func (s *launcher) openMenu(menu paletteMenu) {
	if s.menu == menu {
		s.closeMenu()
	} else {
		s.menu, s.menuSelected = menu, 0
	}
}

func (s *launcher) rows(apps []core.Application, clips []string) []paletteRow {
	rows := make([]paletteRow, 0, len(apps)+len(clips))
	names := map[string]int{}
	for _, app := range apps {
		names[strings.ToLower(app.Name)]++
	}
	if s.clipboardMode {
		for _, text := range clips {
			rows = append(rows, paletteRow{title: preview(text), kind: "文本", key: fmt.Sprintf("clip-%x", sha256.Sum256([]byte(text))), height: palette.RowHeight})
		}
	} else {
		for _, app := range apps {
			row := paletteRow{title: app.Name, kind: "应用", key: app.ItemID(), bitmap: s.icons[app.ItemID()], height: palette.RowHeight}
			if names[strings.ToLower(app.Name)] > 1 {
				row.subtitle = app.Subtitle()
				row.height = palette.DetailRowHeight
			}
			rows = append(rows, row)
		}
	}
	y := palette.Header + palette.Inset + palette.SectionHeight
	for i := range rows {
		rows[i].top = y
		y += rows[i].height + palette.RowGap
	}
	return rows
}

// Reveal only on keyboard changes/reset, excluding both floating bars. Wheel
// scrolling is never pulled back and the scroll extent includes resting insets.
func (s *launcher) reveal(rows []paletteRow, viewport float32) {
	if s.selected == 0 {
		s.scroll.Y = 0
		return
	}
	if s.selected < 0 || s.selected >= len(rows) {
		return
	}
	row := rows[s.selected]
	top, bottom := palette.Header+palette.Inset, viewport-palette.Footer-palette.Inset
	if row.top-s.scroll.Y < top {
		s.scroll.Y = max(0, row.top-top)
	}
	if row.top+row.height-s.scroll.Y > bottom {
		s.scroll.Y = max(0, row.top+row.height-bottom)
	}
}

func barAction(c *ui.Context, label string, disabled bool) ui.Element {
	item := ui.ButtonBase(c).Height(palette.ActionHeight).Shrink(0).Padding(palette.ControlInset, palette.Inset).Gap(palette.Inset).Radius(palette.ActionHeight / 2).Label(label).Disabled(disabled)
	if item.Hovered() {
		item.Background(palette.Hover)
	}
	return item
}

func (s *launcher) primaryLabels() (string, string) {
	if !s.clipboardMode {
		return "打开应用", "打开所选"
	}
	if s.target == 0 || s.tray == nil {
		return "复制文本", "复制所选"
	}
	return "回填文本", "回填所选"
}

func (s *launcher) menuItems(apps []core.Application, clips []string) []paletteMenuItem {
	count := len(apps)
	if s.clipboardMode {
		count = len(clips)
	}
	_, primary := s.primaryLabels()
	if s.menu == menuActions {
		items := []paletteMenuItem{{primary, "↵", count == 0, func() { s.activate(apps, clips, false) }}}
		if s.clipboardMode {
			items = append(items, paletteMenuItem{"仅复制", "▤", count == 0, func() { s.activate(apps, clips, true) }})
		}
		return items
	}
	record := "开启文本历史"
	if s.recording {
		record = "暂停文本历史"
	}
	return []paletteMenuItem{
		{"应用", "▦", false, func() { s.switchMode(false) }},
		{"剪贴板", "▤", false, func() { s.switchMode(true) }},
		{record, "○", false, func() { s.setRecording(!s.recording) }},
		{"清空内存历史", "×", s.history.Len() == 0, func() { s.history.Clear(); s.resetResults() }},
		{"重新扫描应用", "↻", s.scanning, s.scan},
		{"隐藏", "−", false, s.hide},
		{"退出 MyCast", "⏻", false, mygo.App.Quit},
	}
}

func (s *launcher) drawMenu(c *ui.Context, apps []core.Application, clips []string, composing bool) {
	items := s.menuItems(apps, clips)
	if len(items) == 0 {
		return
	}
	s.menuSelected = max(0, min(s.menuSelected, len(items)-1))
	if !composing {
		if c.Shortcut(0, ui.KeyEscape) {
			s.closeMenu()
			return
		}
		if c.Shortcut(0, ui.KeyDown) {
			s.menuSelected = (s.menuSelected + 1) % len(items)
		}
		if c.Shortcut(0, ui.KeyUp) {
			s.menuSelected = (s.menuSelected + len(items) - 1) % len(items)
		}
		if c.Shortcut(0, ui.KeyEnter) && !items[s.menuSelected].disabled {
			chosen := items[s.menuSelected]
			s.closeMenu()
			chosen.run()
			return
		}
	}
	// PopoverBase forwards an outside press. A same-window catcher instead
	// consumes the entire click, so a covered result cannot inherit it.
	catcher := ui.ButtonBase(c.Key("menu-dismiss")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Label("关闭菜单")
	if catcher.Clicked() {
		s.closeMenu()
		return
	}
	panel := ui.Column(c.Key("menu-panel")).Absolute().Bottom(palette.Footer).Width(palette.MenuWidth).Height(float32(len(items))*(palette.MenuRow+palette.RowGap) - palette.RowGap + 2*palette.MenuInset).Padding(palette.MenuInset).Gap(palette.RowGap).Radius(palette.MenuRadius).Material(glass.Glass{}).Label("菜单面板")
	if s.menu == menuApplication {
		panel.Left(palette.Inset)
	} else {
		panel.Right(palette.Inset)
	}
	panel.Children(func() {
		for i, item := range items {
			index, entry := i, item
			row := ui.ButtonBase(c.Key(fmt.Sprintf("menu-%d-%d", s.menu, i))).FillWidth().Height(palette.MenuRow).Shrink(0).Padding(palette.ControlInset, palette.Inset).Gap(palette.Inset).Radius(palette.RowRadius).Justify(ui.Start).Label("菜单项：" + item.text).Disabled(item.disabled)
			if index == s.menuSelected {
				row.Background(palette.Selection)
			} else if row.Hovered() {
				row.Background(palette.Hover)
			}
			row.Children(func() {
				ui.Text(c, entry.glyph).Width(palette.Icon).FontSize(palette.BodySize).TextColor(palette.Secondary)
				ui.Text(c, entry.text).FontSize(palette.BodySize).TextColor(palette.Text)
			})
			if row.Clicked() {
				s.closeMenu()
				entry.run()
			}
		}
	})
}

func (s *launcher) view(c *ui.Context) {
	c.SetTheme(launcherTheme())
	_, height := c.Size()
	background := palette.Fallback
	if c.Vibrancy() {
		background = palette.Scrim
	}
	ui.Box(c.Key("palette-root")).Fill().Background(background).Radius(palette.PanelRadius).Clip().Children(func() {
		apps := core.SearchApplications(s.apps, s.query, 100)
		clips := s.clipResults()
		rows := s.rows(apps, clips)
		count := len(rows)
		if count == 0 {
			s.selected = 0
		} else {
			s.selected = max(0, min(s.selected, count-1))
		}
		ui.Scroll(c.Key("results")).Fill().TrackScroll(&s.scroll).Children(func() {
			ui.Column(c).FillWidth().Shrink(0).Padding(palette.Header+palette.Inset, palette.Inset, palette.Footer+palette.Inset).Gap(palette.RowGap).Children(func() {
				ui.Row(c).Height(palette.SectionHeight-palette.RowGap).Shrink(0).Padding(0, palette.RowInset).Children(func() {
					label := "应用"
					if s.query != "" {
						label = "搜索结果"
					}
					if s.clipboardMode {
						label = "文本历史"
					}
					ui.Textf(c, "%s  %d", label, count).FontSize(palette.CaptionSize).TextColor(palette.Secondary)
				})
				if count == 0 {
					label := "没有匹配的应用"
					hint := "在菜单中重新扫描，或更换搜索词。"
					if s.scanning {
						label = "正在扫描应用…"
						hint = ""
					}
					if s.clipboardMode {
						label = "暂无文本历史"
						hint = "在菜单中开启后，仅记录新复制的文本；退出即清空。"
					}
					ui.Column(c).FillWidth().Height(2*palette.RowHeight).Shrink(0).Padding(palette.Inset, palette.RowInset).Gap(palette.Inset).Children(func() {
						ui.Text(c, label).FontSize(palette.BodySize).TextColor(palette.Secondary)
						ui.Text(c, hint).FontSize(palette.CaptionSize).TextColor(palette.Tertiary)
					})
				}
				for i, row := range rows {
					index, item := i, row
					button := ui.ButtonBase(c.Key(item.key)).FillWidth().Height(item.height).Shrink(0).Padding(palette.RowPaddingY, palette.RowInset).Gap(palette.Gap).Justify(ui.Start).AlignItems(ui.Center).Radius(palette.RowRadius).Label("结果：" + item.title).Disabled(s.menu != menuClosed)
					button.Opacity(rowOpacity(item.top-s.scroll.Y, item.height, height, s.scroll.Y, s.scroll.MaxY-s.scroll.Y))
					if index == s.selected {
						button.Background(palette.Selection)
					} else if button.Hovered() {
						button.Background(palette.Hover)
					}
					if item.subtitle != "" {
						button.Tooltip(item.subtitle)
					}
					button.Children(func() {
						ui.Box(c).Size(palette.IconSlot, palette.IconSlot).Shrink(0).Center().Children(func() {
							if item.bitmap != nil {
								ui.Image(c, item.bitmap).Size(palette.Icon, palette.Icon)
							} else {
								ui.Avatar(c, item.title, nil).Size(palette.Icon, palette.Icon).Label("图标：" + item.title)
							}
						})
						ui.Column(c).Grow(1).MinWidth(0).Gap(palette.DetailGap).Children(func() {
							ui.Text(c, item.title).FontSize(palette.BodySize).NoWrap().Ellipsis("…")
							if item.subtitle != "" {
								ui.Text(c, item.subtitle).FontSize(palette.DetailSize).TextColor(palette.Tertiary).NoWrap().Ellipsis("…")
							}
						})
						ui.Text(c, item.kind).FontSize(palette.CaptionSize).TextColor(palette.Tertiary).Shrink(0)
					})
					if button.Clicked() {
						s.selected = index
						s.activate(apps, clips, false)
					}
				}
			})
		})
		var field ui.Element
		header := ui.ButtonBase(c.Key("header")).Absolute().Top(0).Left(0).Right(0).Height(palette.Header).Padding(0, 2*palette.Inset).Gap(palette.Gap).Justify(ui.Start).Label("搜索栏")
		header.Children(func() {
			ui.Box(c).Size(palette.IconSlot, palette.IconSlot).Shrink(0).Center().Children(func() {
				if s.clipboardMode {
					back := ui.ButtonBase(c.Key("back")).Size(palette.Icon, palette.Icon).Label("返回应用")
					back.Children(func() { ui.Text(c, "‹").FontSize(palette.SearchSize).TextColor(palette.Secondary) })
					if back.Clicked() {
						s.switchMode(false)
					}
				} else {
					ui.Icon(c, searchGlyph).Size(palette.Icon, palette.Icon).TextColor(palette.Secondary)
				}
			})
			field = ui.TextInputBase(c.Key("search"), &s.query).Grow(1).MinWidth(0).Height(palette.Header).Padding(0).FontSize(palette.SearchSize).Background(ui.Transparent).FocusRing(false).Placeholder("搜索应用或剪贴板…").Label("搜索").Bind(&s.input).AutoFocus().Disabled(s.menu != menuClosed)
		})
		if header.Clicked() && s.menu == menuClosed {
			s.restoreSearch = true
		}
		if s.restoreSearch && s.menu == menuClosed {
			field.Focus()
			s.restoreSearch = false
		}
		if field.Changed() {
			s.resetResults()
		}
		if !field.Composing() && s.menu == menuClosed {
			if c.Shortcut(0, ui.KeyEscape) {
				s.hide()
			}
			changed := false
			if c.Shortcut(0, ui.KeyDown) && count > 0 {
				s.selected = (s.selected + 1) % count
				changed = true
			}
			if c.Shortcut(0, ui.KeyUp) && count > 0 {
				s.selected = (s.selected + count - 1) % count
				changed = true
			}
			if changed {
				s.reveal(rows, height)
				s.lastScroll = s.selected
			}
			if field.Submitted() || c.Shortcut(0, ui.KeyEnter) {
				s.activate(apps, clips, false)
			}
		}
		footer := ui.ButtonBase(c.Key("footer")).Absolute().Bottom(0).Left(0).Right(0).Height(palette.Footer).Padding(palette.Inset).Gap(palette.Inset).Label("底栏")
		footer.Children(func() {
			circle := ui.ButtonBase(c.Key("menu-toggle")).Size(palette.MenuButton, palette.MenuButton).Shrink(0).Radius(palette.MenuButton / 2).Material(glass.Glass{Interactive: true}).Label("菜单").Tooltip(s.status)
			circle.Children(func() { ui.Text(c, "≡").FontSize(palette.SearchSize).TextColor(palette.Secondary) })
			if circle.Clicked() {
				s.openMenu(menuApplication)
			}
			mode := "应用"
			if s.clipboardMode {
				mode = "剪贴板"
			}
			if s.scanning {
				mode += " · 扫描中"
			}
			if s.status != "" {
				mode = s.status
			}
			ui.Text(c, mode).FontSize(palette.CaptionSize).TextColor(palette.Secondary).MaxWidth(palette.StatusWidth).NoWrap().Ellipsis("…").Tooltip(s.status)
			ui.Spacer(c)
			ui.Row(c).Height(palette.MenuButton).Shrink(0).Padding(palette.ControlInset).Gap(palette.RowGap).Radius(palette.MenuButton / 2).Material(glass.Glass{Interactive: true}).Children(func() {
				label, _ := s.primaryLabels()
				primary := barAction(c.Key("primary-action"), label, count == 0 || s.menu != menuClosed)
				primary.Children(func() {
					ui.Text(c, label).FontSize(palette.BodySize).Bold()
					ui.Box(c).Size(palette.KeyCap, palette.KeyCap).Center().Radius(palette.KeyCapRadius).Background(palette.Control).Children(func() { ui.Text(c, "↵").FontSize(palette.BodySize).TextColor(palette.Secondary) })
				})
				if primary.Clicked() {
					s.activate(apps, clips, false)
				}
				action := barAction(c.Key("actions-toggle"), "操作", false)
				action.Children(func() { ui.Text(c, "操作").FontSize(palette.BodySize).TextColor(palette.Secondary) })
				if action.Clicked() {
					s.openMenu(menuActions)
				}
			})
		})
		if footer.Clicked() && s.menu == menuClosed {
			s.restoreSearch = true
		}
		if s.menu != menuClosed {
			s.drawMenu(c, apps, clips, field.Composing())
		}
	})
}
