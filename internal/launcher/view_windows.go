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

func launcherTheme() *ui.Theme {
	theme := *ui.DarkTheme()
	theme.Background = ui.Hex("#191a1e")
	theme.Surface = ui.Hex("#191a1e")
	theme.SurfaceHover = ui.Hex("#303137")
	theme.SurfacePressed = ui.Hex("#383940")
	theme.Border = ui.Hex("#35363c")
	theme.Text = ui.Hex("#eeeeF2")
	theme.TextMuted = ui.Hex("#989aa5")
	theme.Accent = ui.Hex("#bcb7ef")
	theme.Radius = 9
	return &theme
}

func (s *launcher) resetResults() {
	s.selected, s.lastScroll = 0, 0
	s.scroll.X, s.scroll.Y = 0, 0
}

func (s *launcher) switchMode(clipboard bool) {
	if s.clipboardMode == clipboard {
		return
	}
	s.clipboardMode, s.query = clipboard, ""
	s.resetResults()
	s.input.Focus()
}

func flatAction(c *ui.Context, label string) ui.Element {
	item := ui.ButtonBase(c).Height(26).Shrink(0).Padding(3, 8).Radius(6).Label(label)
	if item.Hovered() {
		item.Background(c.Theme().SurfaceHover)
	}
	item.Children(func() { ui.Text(c, label).FontSize(11).TextColor(c.Theme().TextMuted) })
	return item
}

func (s *launcher) settings(menu *ui.Menu) {
	if menu.Item("重新扫描应用").Disabled(s.scanning).Chosen() {
		s.scan()
	}
	menu.Separator()
	label := "开启文本历史（仅记录之后复制的文本）"
	if s.recording {
		label = "暂停文本历史"
	}
	if menu.Item(label).Chosen() {
		s.setRecording(!s.recording)
	}
	if menu.Item("清空内存历史").Disabled(s.history.Len() == 0).Chosen() {
		s.history.Clear()
		s.resetResults()
	}
	menu.Separator()
	if menu.Item("隐藏").Chosen() {
		s.hide()
	}
	if menu.Item("退出 MyCast").Chosen() {
		mygo.App.Quit()
	}
}

func (s *launcher) view(c *ui.Context) {
	c.SetTheme(launcherTheme())
	theme := c.Theme()
	ui.Column(c).Fill().Gap(0).Children(func() {
		// Header and footer never shrink into the list. Search is the first
		// surface, without a title row competing for space or focus.
		var field ui.Element
		ui.Row(c).FillWidth().Height(64).Shrink(0).Padding(10, 18).Gap(10).Children(func() {
			ui.Text(c, "⌕").FontSize(28).TextColor(theme.TextMuted).Shrink(0)
			field = ui.TextInput(c.Key("search"), &s.query).Grow(1).MinWidth(0).Height(40).FontSize(18).BorderWidth(0).Background(ui.Transparent).FocusRing(false).Placeholder("搜索应用或剪贴板…").Label("搜索").Bind(&s.input).AutoFocus()
			ui.Text(c, "Esc").FontSize(11).TextColor(theme.TextMuted).Shrink(0)
		})
		if field.Changed() {
			s.resetResults()
		}
		apps := core.SearchApplications(s.apps, s.query, 100)
		clips := s.clipResults()
		count := len(apps)
		if s.clipboardMode {
			count = len(clips)
		}
		if count == 0 {
			s.selected = 0
		} else {
			s.selected = max(0, min(s.selected, count-1))
		}
		// IME candidate confirmation/cancel must remain with the text field.
		if !field.Composing() {
			if c.Shortcut(0, ui.KeyEscape) {
				s.hide()
			}
			if c.Shortcut(0, ui.KeyDown) && count > 0 {
				s.selected = (s.selected + 1) % count
			}
			if c.Shortcut(0, ui.KeyUp) && count > 0 {
				s.selected = (s.selected + count - 1) % count
			}
			if field.Submitted() || c.Shortcut(0, ui.KeyEnter) {
				s.activate(apps, clips, false)
			}
		}
		ui.Divider(c)
		ui.Row(c).FillWidth().Height(31).Shrink(0).Padding(5, 20).Gap(7).Children(func() {
			label := "应用"
			if s.clipboardMode {
				label = "文本历史"
			}
			ui.Text(c, label).FontSize(11).TextColor(theme.TextMuted)
			ui.Textf(c, "%d", count).FontSize(11).TextColor(theme.TextMuted)
			ui.Spacer(c)
			if s.status != "" {
				ui.Text(c, s.status).FontSize(10).TextColor(theme.TextMuted).MaxWidth(390).NoWrap().Ellipsis("…").Tooltip(s.status)
			}
			if s.scanning {
				ui.Text(c, "正在扫描…").FontSize(11).TextColor(theme.TextMuted)
			}
			if s.clipboardMode && !s.recording {
				ui.Text(c, "记录已暂停").FontSize(11).TextColor(theme.TextMuted)
			}
		})
		names := map[string]int{}
		for _, app := range apps {
			names[strings.ToLower(app.Name)]++
		}
		ui.Scroll(c.Key("results")).Grow(1).MinHeight(0).FillWidth().TrackScroll(&s.scroll).Children(func() {
			// Fixed row heights, a non-shrinking content column and explicit
			// scroll reset keep the first result entirely inside the viewport.
			ui.Column(c).FillWidth().Shrink(0).Padding(0, 8, 8).Gap(2).Children(func() {
				if count == 0 {
					label := "没有匹配的应用"
					if s.scanning {
						label = "正在扫描应用…"
					}
					if s.clipboardMode {
						label = "暂无文本历史"
					}
					ui.Column(c).FillWidth().Padding(40, 20).Gap(8).Children(func() {
						ui.Text(c, label).FontSize(15).TextColor(theme.TextMuted)
						if s.clipboardMode {
							ui.Text(c, "在设置中开启后，仅记录新复制的文本；退出即清空。").FontSize(11).TextColor(theme.TextMuted)
						}
					})
				}
				for i := 0; i < count; i++ {
					index := i
					title, subtitle, category, key := "", "", "应用", ""
					var bitmap *ui.Bitmap
					if s.clipboardMode {
						title, category = preview(clips[i]), "文本"
						key = fmt.Sprintf("clip-%x", sha256.Sum256([]byte(clips[i])))
					} else {
						app := apps[i]
						title, key = app.Name, app.ItemID()
						bitmap = s.icons[key]
						if names[strings.ToLower(title)] > 1 {
							subtitle = app.Subtitle()
						}
					}
					item := ui.ButtonBase(c.Key(key)).FillWidth().Height(48).Shrink(0).Padding(7, 12).Gap(12).Justify(ui.Start).AlignItems(ui.Center).Radius(8).Label(title)
					if i == s.selected {
						item.Background(ui.Hex("#33343b"))
					} else if item.Hovered() {
						item.Background(theme.SurfaceHover)
					}
					if subtitle != "" {
						item.Tooltip(subtitle)
					}
					item.Children(func() {
						if bitmap != nil {
							ui.Image(c, bitmap).Size(28, 28).Shrink(0)
						} else {
							ui.Avatar(c, title, nil).Size(28, 28).Shrink(0)
						}
						ui.Column(c).Grow(1).MinWidth(0).Gap(2).Children(func() {
							ui.Text(c, title).FontSize(14).NoWrap().Ellipsis("…").TextAlign(ui.Start)
							if subtitle != "" {
								ui.Text(c, subtitle).FontSize(10).TextColor(theme.TextMuted).NoWrap().Ellipsis("…").TextAlign(ui.Start)
							}
						})
						ui.Text(c, category).FontSize(11).TextColor(theme.TextMuted).Shrink(0)
					})
					if i == s.selected && s.lastScroll != s.selected {
						item.ScrollIntoView()
						s.lastScroll = s.selected
					}
					if item.Clicked() {
						s.selected = index
						s.activate(apps, clips, false)
					}
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).FillWidth().Height(42).Shrink(0).Padding(7, 12).Gap(3).Children(func() {
			if flatAction(c.Key("apps-mode"), "应用").Clicked() {
				s.switchMode(false)
			}
			if flatAction(c.Key("clips-mode"), "剪贴板").Clicked() {
				s.switchMode(true)
			}
			flatAction(c.Key("settings"), "设置 ···").Tooltip(s.status).Menu(s.settings)
			ui.Spacer(c)
			if s.clipboardMode && flatAction(c.Key("copy"), "仅复制").Disabled(count == 0).Clicked() {
				s.activate(apps, clips, true)
			}
			label := "Enter 打开  ·  ↑↓ 选择"
			if s.clipboardMode {
				label = "Enter 回填  ·  ↑↓ 选择"
			}
			ui.Text(c, label).FontSize(11).TextColor(theme.TextMuted).Shrink(0)
		})
	})
}
