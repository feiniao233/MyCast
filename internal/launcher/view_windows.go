//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"crypto/sha256"
	"fmt"
	"strings"

	"github.com/egoist/mygo"
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

var menuGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="20" height="20" viewBox="0 0 20 20"><path d="M4 5H16M4 10H16M4 15H16" fill="none" stroke="currentColor" stroke-width="1.5" stroke-linecap="round"/></svg>`))
var returnGlyph = ui.MustParseSVG([]byte(`<svg xmlns="http://www.w3.org/2000/svg" width="14" height="14" viewBox="0 0 14 14"><path d="M11 3V8H3M6 5L3 8L6 11" fill="none" stroke="currentColor" stroke-width="1.25" stroke-linecap="round" stroke-linejoin="round"/></svg>`))

func (s *launcher) resetResults() {
	s.appNeedsSelection = false
	s.selected, s.lastScroll = 0, 0
	s.scroll.X, s.scroll.Y = 0, 0
	s.clipSelection = ""
	s.clipNeedsSelection = false
}
func (s *launcher) switchMode(clipboard bool) {
	if s.clipboardMode != clipboard {
		s.clipboardMode, s.query = clipboard, ""
		s.resetResults()
	}
	s.restoreSearch = true
}
func (s *launcher) closeMenu() {
	s.menuAtRow = false
	s.menu = menuClosed
	s.restoreSearch = true
	s.menuNeedsFocus = false
	s.menuTargetKey = ""
}
func (s *launcher) openMenu(menu paletteMenu) {
	s.menuAtRow = false
	if s.menu == menu {
		s.closeMenu()
	} else {
		s.menu, s.menuSelected = menu, 0
		s.menuNeedsFocus = true
		s.menuTargetKey = ""
		s.menuTargetClipboard = s.clipboardMode
		if menu == menuActions {
			if s.clipboardMode {
				clips := s.clipResults()
				if s.selected >= 0 && s.selected < len(clips) {
					s.menuTargetKey = clipKey(clips[s.selected])
				}
			} else {
				s.menuTargetKey = s.selectedApplicationID()
			}
		}
	}
}

func (s *launcher) openRowMenu(key string, x, y float32) {
	_, _, index, valid := s.resolveMenuTarget(key, s.clipboardMode)
	if !valid || s.settingsOpen || s.menu != menuClosed {
		return
	}
	s.selected = index
	s.appNeedsSelection = false
	s.clipNeedsSelection = false
	s.openMenu(menuActions)
	s.menuTargetKey = key
	s.menuAtRow = true
	s.menuX, s.menuY = x, y
}

func clipKey(text string) string { return fmt.Sprintf("clip-%x", sha256.Sum256([]byte(text))) }

// Resolve the original menu identity again immediately before each effect.
func (s *launcher) resolveMenuTarget(key string, clipboard bool) ([]core.Application, []string, int, bool) {
	if clipboard != s.clipboardMode {
		return nil, nil, 0, false
	}
	if clipboard {
		clips := s.clipResults()
		for index, text := range clips {
			if clipKey(text) == key {
				return nil, clips, index, true
			}
		}
		return nil, nil, 0, false
	}
	apps := s.applicationResults()
	for index, app := range apps {
		if app.ItemID() == key {
			return apps, nil, index, true
		}
	}
	return nil, nil, 0, false
}
func (s *launcher) runMenuTarget(key string, clipboard bool, action string) {
	apps, clips, index, ok := s.resolveMenuTarget(key, clipboard)
	if !ok {
		s.closeMenu()
		s.clipNeedsSelection = clipboard
		s.status = "操作目标已移除，请重新选择。"
		return
	}
	if !clipboard && apps[index].DisplayOnlyID != "" && action != "source" {
		s.status = "快照入口待验证；暂不能执行此操作。"
		return
	}
	s.selected = index
	if !clipboard {
		s.appNeedsSelection = false
	}
	switch action {
	case "primary":
		s.activate(apps, clips, false)
	case "copy":
		s.activate(apps, clips, true)
	case "preview":
		s.openClipPreview(clips[index])
	case "delete":
		s.deleteClip(clips[index])
	case "favorite":
		s.toggleFavorite(apps[index].ItemID())
	case "alias":
		s.openAliases(apps[index])
	case "source":
		s.openApplicationSource(apps[index])
	case "location":
		app := apps[index]
		if app.ShellID != "" || s.showLocation == nil {
			s.status = "此入口没有文件位置。"
			return
		}
		if err := s.showLocation(app.Path); err != nil {
			s.status = "无法打开入口位置：" + err.Error()
		}
	case "hide":
		s.toggleHiddenApplication(apps[index].ItemID())
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
			if app.DisplayOnlyID != "" {
				row.kind = "待验证"
				row.subtitle = "本地显示快照；扫描验证前不能启动"
				row.height = palette.DetailRowHeight
			}
			if app.Preferred(s.config.Favorites) {
				row.kind = "★ 收藏"
			} else if app.Preferred(s.config.Recents) {
				row.kind = "最近"
			}
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
	palette := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
	item := ui.ButtonBase(c).Height(palette.ActionHeight).Shrink(0).Padding(0, palette.Inset).Gap(palette.Inset).Center().Radius(palette.ActionHeight / 2).Label(label).Disabled(disabled)
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
	_, primary := s.primaryLabels()
	if s.menu == menuActions {
		key, clipboard := s.menuTargetKey, s.menuTargetClipboard
		boundApps, boundClips, index, valid := s.resolveMenuTarget(key, clipboard)
		items := []paletteMenuItem{{primary, "↵", !valid, func() { s.runMenuTarget(key, clipboard, "primary") }}}
		if s.clipboardMode {
			items = append(items, paletteMenuItem{"仅复制", "▤", !valid, func() { s.runMenuTarget(key, clipboard, "copy") }})
			if valid && len(boundClips) > 0 {
				items = append(items, paletteMenuItem{"预览全文", "▤", false, func() { s.runMenuTarget(key, clipboard, "preview") }}, paletteMenuItem{"删除此条", "×", false, func() { s.runMenuTarget(key, clipboard, "delete") }})
			}
		}
		if !s.clipboardMode && valid && len(boundApps) > 0 {
			app := boundApps[index]
			if app.DisplayOnlyID != "" {
				return []paletteMenuItem{{"快照待验证", "…", true, nil}, {"查看来源", "↗", false, func() { s.runMenuTarget(key, clipboard, "source") }}}
			}
			favorite := "收藏应用"
			if app.Preferred(s.config.Favorites) {
				favorite = "取消收藏"
			}
			items = append(items, paletteMenuItem{favorite, "★", false, func() { s.runMenuTarget(key, clipboard, "favorite") }}, paletteMenuItem{"编辑搜索别名", "✎", false, func() { s.runMenuTarget(key, clipboard, "alias") }})
		}
		if !s.clipboardMode && valid && len(boundApps) > 0 {
			items = append(items, paletteMenuItem{"查看来源", "?", false, func() { s.runMenuTarget(key, clipboard, "source") }}, paletteMenuItem{"打开所在位置", "?", boundApps[index].ShellID != "", func() { s.runMenuTarget(key, clipboard, "location") }}, paletteMenuItem{"隐藏此入口", "×", false, func() { s.runMenuTarget(key, clipboard, "hide") }})
		}
		return items
	}
	return s.applicationMenuItems()
}

func (s *launcher) applicationMenuItems() []paletteMenuItem {
	record := "开启文本历史"
	if s.recording {
		record = "暂停文本历史"
	}
	return []paletteMenuItem{
		{"应用", "▦", false, func() { s.switchMode(false) }},
		{"剪贴板", "▤", false, func() { s.switchMode(true) }},
		{record, "○", false, func() { s.setRecording(!s.recording) }},
		{s.retentionLabel(), "◷", false, s.cycleRetention},
		{"清空内存历史", "×", s.history.Len() == 0, s.clearHistory},
		{"重新扫描应用", "↻", s.scanning, s.scan},
		{"设置", "⚙", false, s.openSettings},
		{"小工具", "＋", false, s.openUtilities},
		{"清空最近启动", "×", len(s.config.Recents) == 0, func() {
			selectedID := s.selectedApplicationID()
			value := s.config.Clone()
			value.Recents = nil
			if s.saveCatalog(value) {
				s.status = "最近启动已清空。"
				if !s.clipboardMode {
					s.keepApplicationSelected(selectedID)
				}
			}
		}},
		{"隐藏", "−", false, s.hide},
		{"退出 MyCast", "⏻", false, mygo.App.Quit},
	}
}

func (s *launcher) drawMenu(c *ui.Context, apps []core.Application, clips []string, composing bool) {
	palette := paletteFor(c.Theme().Dark, c.Preferences().HighContrast)
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
	if catcher.Clicked() || catcher.RightClicked() {
		s.closeMenu()
		return
	}
	menuHeight := float32(len(items))*(palette.MenuRow+palette.RowGap) - palette.RowGap + 2*palette.MenuInset
	panel := ui.Column(c.Key("menu-panel")).Absolute().Width(palette.MenuWidth).Height(menuHeight).Padding(palette.MenuInset).Gap(palette.RowGap).Radius(palette.MenuRadius).Background(palette.MenuSurface).Border(1, palette.Border).Label("菜单面板").Role(ui.RolePopup).Focusable()
	// A non-clickable focus owner prevents Enter from activating the footer
	// ButtonBase before the menu's registered shortcuts can handle it.
	if s.menuNeedsFocus {
		panel.Focus()
		s.menuNeedsFocus = false
	}
	if s.menuAtRow {
		width, height := c.Size()
		panel.Left(max(palette.Inset, min(s.menuX, width-palette.MenuWidth-palette.Inset))).Top(max(palette.Header+palette.Inset, min(s.menuY, height-palette.Footer-palette.Inset-menuHeight)))
	} else if s.menu == menuApplication {
		panel.Bottom(palette.Footer)
		panel.Left(palette.Inset)
	} else {
		panel.Bottom(palette.Footer)
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
	s.paletteComposing = false
	defer func() {
		if s.nativeKeyGuard != nil {
			s.nativeKeyGuard.SetComposing(s.paletteComposing)
		}
	}()
	dark := true
	switch s.config.Theme {
	case "light":
		dark = false
	case "system":
		dark = c.Theme().Dark
	}
	palette := paletteFor(dark, c.Preferences().HighContrast)
	c.SetTheme(launcherTheme(palette, dark))
	c.Root().Background(ui.Transparent)
	_, height := c.Size()
	s.viewportHeight = height
	background := palette.Fallback
	if c.Vibrancy() && (!s.appearanceChecked || s.nativeAcrylic) {
		background = palette.Scrim
	}
	ui.Box(c.Key("palette-root")).Fill().Background(background).Radius(palette.PanelRadius).Clip().Children(func() {
		queryAtRows := s.query
		apps := s.applicationResults()
		clips := s.clipResults()
		rows := s.rows(apps, clips)
		selectionLost := false
		menuLost := false
		if s.menu == menuActions && s.menuTargetKey != "" {
			if _, _, _, ok := s.resolveMenuTarget(s.menuTargetKey, s.menuTargetClipboard); !ok {
				menuLost = true
				selectionLost = true
				s.closeMenu()
				s.clipNeedsSelection = s.clipboardMode
				s.status = "操作目标已移除，请重新选择。"
				for _, key := range []ui.Key{ui.KeyEnter, ui.KeyEscape, ui.KeyUp, ui.KeyDown} {
					c.Shortcut(0, key)
				}
			}
		}
		if s.clipboardMode && s.clipSelection != "" && s.clipQuery == s.query {
			found := false
			for index, row := range rows {
				if row.key == s.clipSelection {
					if s.selected != index {
						s.selected = index
						s.reveal(rows, height)
					}
					s.selected = index
					found = true
					break
				}
			}
			if !found {
				s.resetResults()
				selectionLost = true
				s.clipNeedsSelection = true
				s.status = "原选文本已移除，请重新选择。"
			}
		}
		count := len(rows)
		if count == 0 {
			s.selected = 0
		} else {
			s.selected = max(0, min(s.selected, count-1))
		}
		ui.Scroll(c.Key("results")).Fill().Margin(palette.Header, 0, palette.Footer).Clip().TrackScroll(&s.scroll).Children(func() {
			ui.Column(c).FillWidth().Shrink(0).Padding(palette.Inset).Gap(palette.RowGap).Children(func() {
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
					sourceDisabled := !s.clipboardMode && !s.config.AppsFolder && !s.scanning
					if sourceDisabled {
						hint = "系统应用来源已关闭；在设置中开启后可搜索商店应用。"
					}
					emptyHeight := 2 * palette.RowHeight
					if sourceDisabled {
						emptyHeight = 3 * palette.RowHeight
					}
					ui.Column(c).FillWidth().Height(emptyHeight).Shrink(0).Padding(palette.Inset, palette.RowInset).Gap(palette.Inset).Children(func() {
						ui.Text(c, label).FontSize(palette.BodySize).TextColor(palette.Secondary)
						ui.Text(c, hint).FontSize(palette.CaptionSize).TextColor(palette.Tertiary)
						if sourceDisabled {
							button := ui.ButtonBase(c.Key("empty-system-source")).Height(palette.ActionHeight).Padding(0, palette.Inset).Radius(palette.RowRadius).Background(palette.Control).Label("设置系统应用来源")
							button.Children(func() { ui.Text(c, "设置系统应用来源").FontSize(palette.CaptionSize) })
							if button.Clicked() {
								s.openSettings()
							}
						}
					})
				}
				for i, row := range rows {
					index, item := i, row
					button := ui.ButtonBase(c.Key(fmt.Sprintf("presentation-%d-%s", s.presentation, item.key))).FillWidth().Height(item.height).Shrink(0).Padding(palette.RowPaddingY, palette.RowInset).Gap(palette.Gap).Justify(ui.Start).AlignItems(ui.Center).Radius(palette.RowRadius).Label("结果：" + item.title).Disabled(s.menu != menuClosed || s.settingsOpen || menuLost)
					if !c.Preferences().HighContrast {
						button.Opacity(rowOpacity(item.top-s.scroll.Y, item.height, height, s.scroll.Y, s.scroll.MaxY-s.scroll.Y))
					}
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
					button.ContextMenu(func(m *ui.Menu) {
						if s.searchWasComposing || menuLost {
							return
						}
						bounds := button.Bounds()
						x, y, over := button.PointerPosition()
						if !over {
							x, y = palette.Inset, bounds.H
						}
						s.openRowMenu(item.key, bounds.X+x, bounds.Y+y)
						c.AnimationFrame()
					})
					if button.Clicked() {
						needsSelection := (s.clipboardMode && s.clipNeedsSelection) || (!s.clipboardMode && s.appNeedsSelection)
						s.clipNeedsSelection = false
						s.appNeedsSelection = false
						s.selected = index
						if !needsSelection {
							s.activate(apps, clips, false)
						}
					}
				}
			})
		})
		var field ui.Element
		header := ui.ButtonBase(c.Key("header")).Absolute().Top(0).Left(0).Right(0).Height(palette.Header).Padding(0, 2*palette.Inset).Gap(palette.Gap).Justify(ui.Start).Label("搜索栏").Disabled(s.settingsOpen)
		header.Children(func() {
			ui.Box(c).Size(palette.IconSlot, palette.IconSlot).Shrink(0).Center().Children(func() {
				if s.clipboardMode {
					back := ui.ButtonBase(c.Key("back")).Size(palette.Icon, palette.Icon).Label("返回应用").Disabled(s.settingsOpen)
					back.Children(func() { ui.Text(c, "‹").FontSize(palette.SearchSize).TextColor(palette.Secondary) })
					if back.Clicked() {
						s.switchMode(false)
					}
				} else {
					ui.Icon(c, searchGlyph).Size(palette.Icon, palette.Icon).TextColor(palette.Secondary)
				}
			})
			field = ui.TextInputBase(c.Key("search"), &s.query).Grow(1).MinWidth(0).Height(palette.Header).Padding(0).FontSize(palette.SearchSize).Background(ui.Transparent).FocusRing(false).Placeholder("搜索应用或剪贴板…").Label("搜索").Bind(&s.input).AutoFocus().Disabled(s.menu != menuClosed || s.settingsOpen || menuLost)
		})
		if header.Clicked() && s.menu == menuClosed && !s.settingsOpen {
			s.restoreSearch = true
		}
		if s.restoreSearch && s.menu == menuClosed && !s.settingsOpen && !menuLost {
			field.Focus()
			s.restoreSearch = false
		}
		if field.Changed() {
			s.resetResults()
		}
		// Candidate confirmation can clear Composing and submit the field in
		// one event. Preserve the preceding state while consuming its keys.
		wasComposing := s.searchWasComposing
		s.searchWasComposing = field.Composing()
		s.paletteComposing = s.paletteComposing || field.Composing()
		if wasComposing || field.Composing() {
			c.Shortcut(0, ui.KeyEnter)
			c.Shortcut(0, ui.KeyEscape)
		}
		if !wasComposing && !field.Composing() && s.menu == menuClosed && !s.settingsOpen && !menuLost {
			// Scope this to the editor. A focused result row keeps its own
			// ContextMenu routing and immutable ID rather than selectedIndex.
			shiftF10 := false
			if s.input.Focused(c) {
				shiftF10 = c.Shortcut(ui.Shift, ui.KeyF10)
			}
			if shiftF10 && count > 0 && !selectionLost && !s.clipNeedsSelection && !s.appNeedsSelection {
				s.openMenu(menuActions)
			}
			if c.Shortcut(0, ui.KeyEscape) {
				s.hide()
			}
			changed := false
			if !s.statusControl.Focused(c) && c.Shortcut(0, ui.KeyDown) && count > 0 {
				s.selected = (s.selected + 1) % count
				changed = true
			}
			if !s.statusControl.Focused(c) && c.Shortcut(0, ui.KeyUp) && count > 0 {
				s.selected = (s.selected + count - 1) % count
				changed = true
			}
			if changed {
				s.clipNeedsSelection = false
				s.appNeedsSelection = false
				s.reveal(rows, height)
				s.lastScroll = s.selected
			}
			if !s.statusControl.Focused(c) && !selectionLost && !s.clipNeedsSelection && !s.appNeedsSelection && (field.Submitted() || c.Shortcut(0, ui.KeyEnter)) {
				s.activate(apps, clips, false)
			}
		}
		footer := ui.ButtonBase(c.Key("footer")).Absolute().Bottom(0).Left(0).Right(0).Height(palette.Footer).Padding(palette.Inset).Gap(palette.Inset).Label("底栏").Disabled(s.settingsOpen)
		footer.Children(func() {
			circle := ui.ButtonBase(c.Key("menu-toggle")).Size(palette.MenuButton, palette.MenuButton).Shrink(0).Radius(palette.MenuButton/2).Background(palette.Control).Border(1, palette.Border).Label("菜单").Tooltip("菜单").Disabled(s.settingsOpen)
			circle.Children(func() {
				ui.Icon(c, menuGlyph).Size(palette.FooterIcon, palette.FooterIcon).TextColor(palette.Secondary).Label("底栏菜单图标")
			})
			if circle.Clicked() {
				s.openMenu(menuApplication)
			}
			mode := "应用"
			if s.clipboardMode {
				mode = "剪贴板"
			}
			if message := s.statusMessage(); message != "" {
				mode = message
			}
			status := ui.ButtonBase(c.Key("status-details-trigger")).Width(palette.StatusWidth).MinWidth(0).Shrink(1).Height(palette.MenuButton).Padding(0).Justify(ui.Start).Clip().Label("状态详情").Tooltip("查看完整状态详情").Bind(&s.statusControl).Disabled(s.statusMessage() == "" || s.settingsOpen || s.menu != menuClosed || menuLost || field.Composing())
			status.Children(func() {
				ui.Text(c, mode).Grow(1).MinWidth(0).FontSize(palette.CaptionSize).FixedLineHeight(palette.ControlLine).TextColor(palette.Secondary).SingleLine()
			})
			if status.Clicked() || status.Shortcut(0, ui.KeyEnter) || status.Shortcut(0, ui.KeySpace) {
				s.openStatusDetails()
			}
			ui.Spacer(c)
			ui.Row(c).Height(palette.MenuButton).Shrink(0).Padding(palette.ControlInset).Gap(palette.RowGap).Radius(palette.MenuButton/2).Background(palette.Control).Border(1, palette.Border).Children(func() {
				label, _ := s.primaryLabels()
				pending := !s.clipboardMode && s.selected >= 0 && s.selected < len(apps) && apps[s.selected].DisplayOnlyID != ""
				if pending {
					label = "等待验证"
				}
				primary := barAction(c.Key(fmt.Sprintf("primary-action-%d", s.presentation)), label, pending || count == 0 || s.menu != menuClosed || s.settingsOpen || selectionLost || s.clipNeedsSelection || (!s.clipboardMode && s.appNeedsSelection))
				primary.Children(func() {
					ui.Text(c, label).FontSize(palette.BodySize).FixedLineHeight(palette.ControlLine).SingleLine().Bold().Label("底栏主操作文字")
					ui.Box(c).Size(palette.KeyCap, palette.KeyCap).Center().Radius(palette.KeyCapRadius).Background(palette.Control).Label("底栏快捷键胶囊").Children(func() {
						ui.Icon(c, returnGlyph).Size(palette.KeyGlyph, palette.KeyGlyph).TextColor(palette.Secondary).Label("底栏回车图标")
					})
				})
				if primary.Clicked() {
					s.activate(apps, clips, false)
				}
				if s.clipboardMode {
					action := barAction(c.Key(fmt.Sprintf("actions-toggle-%d", s.presentation)), "操作", s.settingsOpen || selectionLost || s.clipNeedsSelection || (!s.clipboardMode && s.appNeedsSelection))
					action.Children(func() {
						ui.Text(c, "操作").FontSize(palette.BodySize).FixedLineHeight(palette.ControlLine).SingleLine().TextColor(palette.Secondary).Label("底栏操作文字")
					})
					if action.Clicked() {
						s.openMenu(menuActions)
					}
				}
			})
		})
		if footer.Clicked() && s.menu == menuClosed && !s.settingsOpen {
			s.restoreSearch = true
		}
		if s.menu != menuClosed {
			s.drawMenu(c, apps, clips, field.Composing())
		}
		if s.settingsOpen {
			s.drawSettings(c)
		}
		if menuLost {
			ui.ButtonBase(c.Key("menu-target-lost-catcher")).Absolute().Top(0).Left(0).Right(0).Bottom(0).Label("失效操作已关闭")
		}
		if s.clipboardMode && s.query == queryAtRows && s.selected >= 0 && s.selected < len(rows) && strings.HasPrefix(rows[s.selected].key, "clip-") {
			s.clipSelection = rows[s.selected].key
			s.clipQuery = queryAtRows
		}
	})
	// Without WS_SYSMENU, route Alt+F4 through MyGo's normal close event:
	// tray-backed close hides; no-tray close remains a real close request.
	if c.Shortcut(ui.Alt, ui.KeyF4) && s.requestClose != nil {
		s.requestClose()
	}
}
