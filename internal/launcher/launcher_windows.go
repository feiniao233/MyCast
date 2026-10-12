// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build windows

// Package launcher owns the native MyGo UI. State mutations occur on its UI thread.
package launcher

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
)

const shortcut = "Ctrl+Alt+Space"

type launcher struct {
	win                                          *mygo.Window
	tray                                         *mygo.Tray
	input                                        ui.Handle
	apps                                         []core.Application
	history                                      *core.History
	query, status                                string
	selected                                     int
	lastScroll                                   int
	clipboardMode, recording, scanning, quitting bool
	sequence                                     uint32
	target                                       uintptr
	stop                                         chan struct{}
}

func Run() error {
	s := &launcher{history: core.NewHistory(50, 64*1024), stop: make(chan struct{}), lastScroll: -1}
	mygo.App.SetName("MyCast")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	mygo.App.OnSecondInstance(func([]string, string) {
		if s.win != nil {
			s.show(false)
		}
	})
	mygo.App.OnBeforeQuit(func(*mygo.QuitEvent) { s.quitting = true })
	mygo.App.WhenReady(s.ready)
	return mygo.App.Run()
}

func (s *launcher) ready() {
	s.win = mygo.NewWindow(mygo.WindowOptions{
		Title: "MyCast", Width: 720, Height: 610, Hidden: true, Frameless: true,
		DisableResize: true, DisableMaximize: true, DisableMinimize: true,
		SkipTaskbar: true, AlwaysOnTop: true, Content: ui.View(s.view),
	})
	s.win.OnBlur(s.hide)
	s.win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting && s.tray != nil {
			e.PreventDefault()
			s.hide()
		}
	})
	var err error
	s.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: icon(), ToolTip: "MyCast · Ctrl+Alt+Space"})
	if err != nil {
		// Keep a discoverable window if the tray cannot be created.
		s.win.SetSkipTaskbar(false)
		s.status = "托盘创建失败：" + err.Error()
	} else {
		s.tray.OnClick(func() { s.show(false) })
		menu := mygo.NewMenu([]*mygo.MenuItem{
			{Label: "打开 MyCast", Click: func(*mygo.MenuItem, *mygo.Window) { s.show(false) }},
			{Label: "重新扫描应用", Click: func(*mygo.MenuItem, *mygo.Window) { s.scan() }},
			{Label: "清空剪贴板历史", Click: func(*mygo.MenuItem, *mygo.Window) { s.history.Clear(); s.win.Invalidate() }},
			mygo.Separator(), {Label: "退出 MyCast", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
		})
		s.tray.OnRightClick(func() { s.hide(); s.tray.SetMenu(menu); defer s.tray.SetMenu(nil); s.tray.PopUpMenu() })
	}
	if err := mygo.GlobalShortcut.Register(shortcut, func() {
		if s.win.IsVisible() {
			s.hide()
		} else {
			s.show(true)
		}
	}); err != nil {
		s.status = "热键注册失败（可能已被占用）；请使用托盘打开。"
	}
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		close(s.stop)
		mygo.GlobalShortcut.UnregisterAll()
		s.history.Clear()
		if s.tray != nil {
			s.tray.Destroy()
		}
	})
	s.scan()
	go s.pollClipboard()
	s.show(false)
}

// With no tray, keep the fallback window reachable even if the hotkey failed.
func (s *launcher) hide() {
	if s.tray != nil {
		s.win.Hide()
	}
}

// Only a hotkey activation captures a paste target. Tray clicks never guess one.
func (s *launcher) show(captureTarget bool) {
	if captureTarget {
		s.target = platform.ForegroundWindow()
	} else {
		s.target = 0
	}
	s.query, s.selected, s.lastScroll = "", 0, -1
	s.win.Center()
	s.input.Focus()
	s.win.Show()
	s.win.Invalidate()
}

func (s *launcher) scan() {
	if s.scanning {
		return
	}
	s.scanning = true
	go func() {
		apps, errs := core.IndexApplications(platform.DefaultAppRoots())
		s.win.Update(func() {
			s.apps, s.scanning = apps, false
			if len(errs) > 0 {
				s.status = fmt.Sprintf("已索引 %d 个应用；%d 个位置无法读取。", len(apps), len(errs))
			}
		})
	}()
}

func (s *launcher) setRecording(enabled bool) {
	s.recording = enabled
	// Establish a new baseline without reading the existing clipboard.
	s.sequence = platform.ClipboardSequence()
	if !enabled {
		s.status = "已暂停记录；已有历史仅留在内存，可随时清空。"
	} else {
		s.status = "仅记录此刻之后复制的文本，退出即清空。不要复制密码等敏感信息。"
	}
}

func (s *launcher) pollClipboard() {
	ticker := time.NewTicker(600 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-s.stop:
			return
		case <-ticker.C:
			s.win.Update(func() {
				if s.quitting || !s.recording {
					return
				}
				seq := platform.ClipboardSequence()
				if seq == 0 || seq == s.sequence {
					return
				}
				if s.sequence == 0 {
					s.sequence = seq
					return
				}
				text, err := platform.ReadClipboardText()
				if err != nil {
					return
				} // Busy clipboard may become readable on the next tick.
				if seq != platform.ClipboardSequence() {
					return
				}
				s.sequence = seq
				s.history.Add(text)
			})
		}
	}
}

func (s *launcher) clipResults() []string {
	var found []string
	q := strings.ToLower(strings.TrimSpace(s.query))
	for _, entry := range s.history.Entries() {
		if strings.Contains(strings.ToLower(entry), q) {
			found = append(found, entry)
		}
	}
	return found
}

func preview(text string) string {
	text = strings.Join(strings.Fields(text), " ")
	r := []rune(text)
	if len(r) > 65 {
		return string(r[:65]) + "…"
	}
	return text
}

func (s *launcher) activate(apps []core.Application, clips []string, copyOnly bool) {
	if s.clipboardMode {
		if s.selected < 0 || s.selected >= len(clips) {
			return
		}
		text := clips[s.selected]
		if copyOnly || s.target == 0 || s.tray == nil {
			if err := platform.WriteClipboardText(text); err != nil {
				s.status = "复制失败：" + err.Error()
				return
			}
			s.sequence = platform.ClipboardSequence()
			s.status = "已复制。切换到目标应用后按 Ctrl+V。"
			return
		}
		target := s.target
		s.target = 0
		s.hide()
		if err := platform.PasteToWindow(target, text); err != nil {
			s.status = "回填未确认：" + err.Error() + "。请先检查目标以免重复粘贴；必要时用“仅复制”手动粘贴。"
			s.show(false)
		}
		s.sequence = platform.ClipboardSequence()
	} else {
		if s.selected < 0 || s.selected >= len(apps) {
			return
		}
		if err := platform.Launch(apps[s.selected].Path); err != nil {
			s.status = "启动失败：" + err.Error()
			return
		}
		s.hide()
	}
}

func (s *launcher) view(c *ui.Context) {
	theme := c.Theme()
	ui.Column(c).Fill().Padding(22).Gap(12).Children(func() {
		ui.Row(c).FillWidth().Gap(10).Children(func() {
			ui.Text(c, "MyCast").FontSize(23).Bold().Grow(1)
			if ui.Button(c, "应用").Clicked() {
				s.clipboardMode = false
				s.selected = 0
				s.lastScroll = -1
				s.query = ""
				s.input.Focus()
			}
			if ui.Button(c, "剪贴板").Clicked() {
				s.clipboardMode = true
				s.selected = 0
				s.lastScroll = -1
				s.query = ""
				s.input.Focus()
			}
			if ui.Button(c, "隐藏").Clicked() {
				s.hide()
			}
		})
		field := ui.TextInput(c.Key("search"), &s.query).FillWidth().FontSize(20).Placeholder("搜索应用或剪贴板…").Label("搜索").Bind(&s.input).AutoFocus()
		if field.Changed() {
			s.selected = 0
			s.lastScroll = -1
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
		// Do not treat an IME candidate confirmation/cancel as launcher action.
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
		if s.clipboardMode {
			ui.Row(c).Gap(8).Children(func() {
				label := "开启文本历史（仅内存）"
				if s.recording {
					label = "暂停记录"
				}
				if ui.Button(c, label).Clicked() {
					s.setRecording(!s.recording)
				}
				if ui.Button(c, "清空历史").Clicked() {
					s.history.Clear()
					s.selected = 0
				}
			})
		}
		ui.Scroll(c).Grow(1).FillWidth().Children(func() {
			ui.Column(c).FillWidth().Gap(5).Children(func() {
				if count == 0 {
					label := "没有匹配的应用。可点击“重新扫描”。"
					if s.scanning {
						label = "正在扫描开始菜单和桌面…"
					}
					if s.clipboardMode {
						label = "暂无历史。开启记录后，只收集之后复制的文本。"
					}
					ui.Text(c, label).TextColor(theme.TextMuted)
				}
				for i := 0; i < count; i++ {
					title := ""
					if s.clipboardMode {
						title = preview(clips[i])
					} else {
						title = apps[i].Name
					}
					if i == s.selected {
						title = "› " + title
					}
					item := ui.Button(c.Key(fmt.Sprintf("result-%t-%d", s.clipboardMode, i)), title).FillWidth().Height(38)
					if i == s.selected && s.lastScroll != s.selected {
						item.ScrollIntoView()
						s.lastScroll = s.selected
					}
					if item.Clicked() {
						s.selected = i
						s.activate(apps, clips, false)
					}
				}
			})
		})
		ui.Divider(c)
		ui.Row(c).Gap(8).Children(func() {
			if s.clipboardMode {
				if ui.Button(c, "仅复制所选").Disabled(count == 0).Clicked() {
					s.activate(apps, clips, true)
				}
			} else if ui.Button(c, "重新扫描").Disabled(s.scanning).Clicked() {
				s.scan()
			}
			ui.Spacer(c)
			ui.Text(c, "↑↓ 选择 · Enter 打开/回填 · Esc 隐藏").FontSize(11).TextColor(theme.TextMuted)
		})
		ui.Text(c, s.status).FontSize(11).TextColor(theme.TextMuted)
		ui.Text(c, "Ctrl+Alt+Space 呼出 · 本地运行 · 无遥测").FontSize(11).TextColor(theme.TextMuted)
		if s.tray == nil {
			ui.Button(c, "退出 MyCast").OnClick(mygo.App.Quit)
		}
	})
}

func icon() []byte {
	im := image.NewNRGBA(image.Rect(0, 0, 32, 32))
	for y := 3; y < 29; y++ {
		for x := 3; x < 29; x++ {
			ink := color.NRGBA{R: 87, G: 82, B: 232, A: 255}
			if (x >= 8 && x <= 10) || (x >= 21 && x <= 23) || (y >= 8 && y <= 11 && x >= 10 && x <= 21) || (x >= 14 && x <= 17 && y >= 10 && y <= 18) {
				ink = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
			}
			im.SetNRGBA(x, y, ink)
		}
	}
	var b bytes.Buffer
	_ = png.Encode(&b, im)
	return b.Bytes()
}
