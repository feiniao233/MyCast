// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build windows

// Package launcher owns the native MyGo UI. State mutations occur on its UI thread.
package launcher

import (
	"bytes"
	"context"
	"errors"
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
	"github.com/feiniao233/MyCast/internal/settings"
)

type launcher struct {
	searchWasComposing                             bool
	statusControl                                  ui.Handle
	statusDetailsOpen                              bool
	statusDetail                                   string
	statusDetailScroll                             ui.ScrollState
	requestClose                                   func()
	appearanceChecked, nativeAcrylic               bool
	syncNativeTheme                                func(string)
	clipQuery                                      string
	pasteText                                      func(uintptr, string) error
	menuTargetKey                                  string
	menuTargetClipboard, clipNeedsSelection        bool
	utilityOpen, linkEditor                        bool
	calcText, linksText, utilityNote               string
	utilityScroll                                  ui.ScrollState
	openURL                                        func(string) error
	previewScroll                                  ui.ScrollState
	clipPreview, clipSelection                     string
	previewOpen                                    bool
	retention                                      time.Duration
	copyText                                       func(string) error
	persistPreferences                             func(settings.Settings) (settings.Settings, error)
	scanQueue                                      core.ScanQueue
	scanCancel                                     context.CancelFunc
	scanWork                                       func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error)
	uiUpdate                                       func(func())
	aliasID, aliasText, aliasName                  string
	launchApplication                              func(string) error
	win                                            *mygo.Window
	tray                                           *mygo.Tray
	input                                          ui.Handle
	scroll                                         ui.ScrollState
	icons                                          map[string]*ui.Bitmap
	menu                                           paletteMenu
	menuSelected                                   int
	menuNeedsFocus                                 bool
	restoreSearch                                  bool
	config                                         settings.Settings
	store                                          *settings.Store
	shortcuts                                      *settings.Shortcuts
	settingsOpen, settingsNeedsFocus               bool
	settingsDraft                                  settings.Settings
	hotkeyText, rootsText, themeText, settingsNote string
	applyPreferences                               func(settings.Settings) (settings.Settings, error)
	apps                                           []core.Application
	history                                        *core.History
	query, status                                  string
	selected                                       int
	lastScroll                                     int
	clipboardMode, recording, scanning, quitting   bool
	sequence                                       uint32
	target                                         uintptr
	stop                                           chan struct{}
}

func Run() error {
	s := &launcher{history: core.NewHistory(50, 64*1024), stop: make(chan struct{}), config: settings.Defaults(), launchApplication: platform.Launch, copyText: platform.WriteClipboardText, openURL: platform.OpenHTTPURL}
	mygo.App.SetName("MyCast")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if path, err := settings.DefaultPath(); err == nil {
		s.store = &settings.Store{Path: path}
		if value, err := s.store.Load(); err == nil {
			s.config = value
		} else {
			s.status = "设置无效，暂按默认运行；原文件未覆盖。"
		}
	} else {
		s.status = "无法定位应用设置目录，暂按默认运行。"
	}
	// Override this app's native appearance before window creation, so DWM
	// and the UI start in the same saved theme, independently of Windows.
	mygo.Theme.SetSource(nativeThemeSource(s.config.Theme))
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
	s.pasteText = platform.PasteToWindow
	s.win = mygo.NewWindow(mygo.WindowOptions{
		Title: "MyCast", Width: int(palette.Width), Height: int(palette.Height), Hidden: true, Frameless: true,
		Transparent: true, BackgroundColor: "#00000000", Vibrancy: mygo.VibrancyAcrylic,
		DisableResize: true, DisableMaximize: true, DisableMinimize: true,
		SkipTaskbar: true, AlwaysOnTop: true, Content: ui.View(s.view),
	})
	platform.RemovePaletteSystemMenu(s.win.NativeHandle())
	appearance := platform.ConfigurePaletteWindow(s.win.NativeHandle())
	s.appearanceChecked, s.nativeAcrylic = true, appearance.Acrylic
	s.syncNativeTheme = func(theme string) { mygo.Theme.SetSource(nativeThemeSource(theme)) }
	s.win.OnBlur(s.hide)
	s.requestClose = s.win.Close
	s.uiUpdate = s.win.Update
	s.scanWork = s.readApps
	s.win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting && s.tray != nil {
			e.PreventDefault()
			s.hide()
		}
	})
	var err error
	s.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: icon(), ToolTip: "MyCast · " + s.config.Hotkey})
	if err != nil {
		// Keep a discoverable window if the tray cannot be created.
		s.win.SetSkipTaskbar(false)
		s.status = "托盘创建失败：" + err.Error()
	} else {
		s.tray.OnClick(func() { s.show(false) })
		menu := mygo.NewMenu([]*mygo.MenuItem{
			{Label: "打开 MyCast", Click: func(*mygo.MenuItem, *mygo.Window) { s.show(false) }},
			{Label: "重新扫描应用", Click: func(*mygo.MenuItem, *mygo.Window) { s.scan() }},
			{Label: "清空剪贴板历史", Click: func(*mygo.MenuItem, *mygo.Window) { s.clearHistory(); s.win.Invalidate() }},
			mygo.Separator(), {Label: "退出 MyCast", Click: func(*mygo.MenuItem, *mygo.Window) { mygo.App.Quit() }},
		})
		s.tray.OnRightClick(func() { s.hide(); s.tray.SetMenu(menu); defer s.tray.SetMenu(nil); s.tray.PopUpMenu() })
	}
	s.shortcuts = &settings.Shortcuts{Register: func(key string) error { return mygo.GlobalShortcut.Register(key, s.onHotkey) }, Unregister: mygo.GlobalShortcut.Unregister}
	if err := s.shortcuts.Register(s.config.Hotkey); err != nil {
		s.status = "热键注册失败（可能已被占用）；请使用托盘打开。"
	} else {
		s.shortcuts.Current = s.config.Hotkey
	}
	s.applyPreferences = s.applySettings
	s.persistPreferences = s.saveOrdinarySettings
	mygo.App.OnWillQuit(func(*mygo.QuitEvent) {
		close(s.stop)
		if s.scanCancel != nil {
			s.scanCancel()
		}
		mygo.GlobalShortcut.UnregisterAll()
		s.clearHistory()
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
	s.statusDetailsOpen, s.statusDetail = false, ""
	s.closeMenu()
	s.settingsOpen = false
	s.utilityOpen, s.linkEditor = false, false
	s.calcText, s.linksText = "", ""
	s.clipPreview, s.clipSelection = "", ""
	s.clipNeedsSelection = false
	s.previewOpen = false
	if s.tray != nil && s.win != nil {
		s.win.Hide()
	}
}

// Only a hotkey activation captures a paste target. Tray clicks never guess one.
func (s *launcher) show(captureTarget bool) {
	s.statusDetailsOpen, s.statusDetail = false, ""
	if captureTarget {
		s.target = platform.ForegroundWindow()
	} else {
		s.target = 0
	}
	s.query, s.selected, s.lastScroll = "", 0, 0
	s.menu, s.restoreSearch = menuClosed, true
	s.settingsOpen = false
	s.utilityOpen, s.linkEditor = false, false
	s.calcText, s.linksText = "", ""
	s.clipPreview, s.clipSelection = "", ""
	s.clipNeedsSelection = false
	s.previewOpen = false
	s.scroll.X, s.scroll.Y = 0, 0
	s.win.Center()
	s.input.Focus()
	s.win.Show()
	s.win.Invalidate()
}

func (s *launcher) scan() {
	if s.scanWork == nil || s.uiUpdate == nil {
		return
	}
	generation, start := s.scanQueue.Request()
	if !start {
		if s.scanCancel != nil {
			s.scanCancel()
		}
		return
	}
	s.scanning = true
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	s.scanCancel = cancel
	roots := append([]string(nil), s.config.IndexRoots...)
	if len(roots) == 0 {
		roots = platform.DefaultAppRoots()
	}
	go func() {
		apps, icons, errs := s.scanWork(ctx, roots)
		cancel()
		s.uiUpdate(func() {
			publish, restart := s.scanQueue.Finish(generation)
			s.scanning = false
			if s.quitting {
				return
			}
			if restart {
				s.scan()
				return
			}
			if !publish {
				return
			}
			selectedID := s.selectedApplicationID()
			s.apps, s.scanning = core.WithUserAliases(apps, s.config.Aliases), false
			s.icons = icons
			s.scroll.X, s.scroll.Y, s.lastScroll = 0, 0, 0
			if !s.clipboardMode {
				s.keepApplicationSelected(selectedID)
			}
			if len(errs) > 0 {
				s.status = fmt.Sprintf("已索引 %d 个应用；%d 个位置无法读取。", len(apps), len(errs))
				for _, err := range errs {
					if errors.Is(err, core.ErrScanBudget) || errors.Is(err, context.DeadlineExceeded) {
						s.status = fmt.Sprintf("扫描达到数量或时限预算，显示 %d 个应用；可缩小索引目录。", len(apps))
						break
					}
				}
			}
		})
	}()
}

func (s *launcher) readApps(ctx context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
	options := core.DefaultScanOptions()
	options.AllowPath = platform.LocalAppPath
	apps, errs := core.IndexApplicationsWithOptions(ctx, roots, platform.ShortcutIdentity, options)
	apps = core.PrepareSearchContext(ctx, apps)
	icons := map[string]*ui.Bitmap{}
	for i, app := range apps {
		if ctx.Err() != nil || i >= 256 {
			break
		}
		if im := platform.ApplicationIcon(app.Launch.Target); im != nil {
			icons[app.ItemID()] = ui.NewBitmap(im)
		}
	}
	if err := ctx.Err(); err != nil {
		if len(errs) < 128 {
			errs = append(errs, err)
		} else {
			errs[len(errs)-1] = err
		}
	}
	return apps, icons, errs
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
				s.expireHistory()
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
			if s.copyText == nil {
				s.status = "复制接口不可用。"
				return
			}
			if err := s.copyText(text); err != nil {
				s.status = "复制失败：" + err.Error()
				return
			}
			s.sequence = platform.ClipboardSequence()
			s.status = "已复制。切换到目标应用后按 Ctrl+V。"
			return
		}
		if s.pasteText == nil {
			s.status = "回填接口不可用。"
			return
		}
		target := s.target
		s.target = 0
		s.hide()
		if err := s.pasteText(target, text); err != nil {
			s.status = "回填未确认：" + err.Error() + "。请先检查目标以免重复粘贴；必要时用“仅复制”手动粘贴。"
			s.show(false)
		}
		s.sequence = platform.ClipboardSequence()
	} else {
		if s.selected < 0 || s.selected >= len(apps) {
			return
		}
		if s.launchApplication == nil {
			s.status = "启动接口不可用。"
			return
		}
		app := apps[s.selected]
		if err := s.launchApplication(app.Path); err != nil {
			s.status = "启动失败：" + err.Error()
			return
		}
		s.recordRecent(app.ItemID())
		s.keepApplicationSelected(app.ItemID())
		s.hide()
	}
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
