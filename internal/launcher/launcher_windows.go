// SPDX-License-Identifier: AGPL-3.0-or-later
//go:build windows

// Package launcher owns the native MyGo UI. State mutations occur on its UI thread.
package launcher

import (
	"context"
	"image"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/branding"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
	"github.com/feiniao233/MyCast/internal/settings"
)

type launcher struct {
	snapshotOpen                                                       bool
	snapshotStore                                                      *settings.SnapshotStore
	snapshotAttempted, snapshotBusy, snapshotClearing, pendingComplete bool
	snapshotNext                                                       *snapshotJob
	snapshotStatus                                                     string
	pendingApps                                                        []core.Application

	appSearch                                      applicationSearch
	autoRoots                                      []string
	autoAllFiles                                   bool
	retryTimer                                     *time.Timer
	retryEpoch                                     uint64
	retryAttempts                                  int
	incompleteRoots                                map[string]bool
	retryDelay                                     time.Duration
	backgroundStart                                bool
	startupRegistration                            *settings.StartupRegistration
	autoSources                                    uint8
	shellImages                                    map[string]image.Image
	shellImagesAt                                  time.Time
	fileImages                                     map[string]fileIconEntry
	scanTimeout                                    time.Duration
	hiddenOpen                                     bool
	hiddenScroll                                   ui.ScrollState
	showLocation                                   func(string) error
	autoDirty                                      bool
	watchEpoch                                     uint64
	watchCancel                                    context.CancelFunc
	scanShellWork                                  func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error)
	launchShellApplication                         func(string) error
	presentation                                   uint64
	scanMissing                                    func(context.Context, []core.Application) map[string]bool
	scanLastStatus, scanDiagnostic                 string
	appNeedsSelection                              bool
	scanProgress, scanFailure                      string
	viewportHeight                                 float32
	nativeCaptionRemoved                           bool
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
	menuAtRow                                      bool
	menuX, menuY                                   float32
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
	s.launchShellApplication = platform.LaunchShellApplication
	s.showLocation = platform.ShowApplicationLocation
	s.backgroundStart = backgroundInvocation(os.Args[1:])
	if executable, err := os.Executable(); err == nil {
		s.startupRegistration, _ = platform.NewStartupRegistration(executable)
	}
	mygo.App.SetName("MyCast")
	if !mygo.App.RequestSingleInstanceLock() {
		return nil
	}
	if path, err := settings.DefaultPath(); err == nil {
		s.store = &settings.Store{Path: path}
		s.snapshotStore = &settings.SnapshotStore{Path: filepath.Join(filepath.Dir(path), "startup-snapshot.json")}
		if value, err := s.store.Load(); err == nil {
			s.config = value
			if s.store.SourcesDefaulted {
				s.status = "已兼容旧配置：默认加入系统应用来源；可在设置中关闭。原文件未自动改写。"
			}
		} else {
			s.status = "设置无效，暂按默认运行；原文件未覆盖。"
		}
	} else {
		s.status = "无法定位应用设置目录，暂按默认运行。"
	}
	// Override this app's native appearance before window creation, so DWM
	// and the UI start in the same saved theme, independently of Windows.
	mygo.Theme.SetSource(nativeThemeSource(s.config.Theme))
	mygo.App.OnSecondInstance(func(args []string, _ string) {
		if s.win != nil {
			handleSecondInvocation(args, func() { s.show(false) })
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
	if err := s.win.SetIcon(branding.WindowPNG()); err != nil {
		s.status = "窗口图标加载失败：" + err.Error()
	}
	s.nativeCaptionRemoved = platform.RemovePaletteSystemMenu(s.win.NativeHandle())
	appearance := platform.ConfigurePaletteWindow(s.win.NativeHandle())
	s.appearanceChecked, s.nativeAcrylic = true, appearance.Acrylic
	s.syncNativeTheme = func(theme string) { mygo.Theme.SetSource(nativeThemeSource(theme)) }
	s.win.OnBlur(s.hide)
	s.requestClose = s.win.Close
	s.uiUpdate = s.win.Update
	s.scanWork = s.readApps
	s.scanShellWork = s.readShellApps
	s.scanMissing = func(ctx context.Context, previous []core.Application) map[string]bool {
		missing := map[string]bool{}
		for _, app := range previous {
			if ctx.Err() != nil {
				break
			}
			if platform.ConfirmedMissingApplication(app) {
				missing[app.ItemID()] = true
			}
		}
		return missing
	}
	s.win.OnClose(func(e *mygo.CloseEvent) {
		if !s.quitting && s.tray != nil {
			e.PreventDefault()
			s.hide()
		}
	})
	var err error
	s.tray, err = mygo.NewTray(mygo.TrayOptions{Icon: branding.TrayPNG(), ToolTip: "MyCast · " + s.config.Hotkey})
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
		s.stopSourceRetry()
		if s.watchCancel != nil {
			s.watchCancel()
		}
		if s.scanCancel != nil {
			s.scanCancel()
		}
		mygo.GlobalShortcut.UnregisterAll()
		s.clearHistory()
		if s.tray != nil {
			s.tray.Destroy()
		}
	})
	s.restartSourceWatch()
	s.scan()
	go s.pollClipboard()
	if !s.backgroundStart || s.tray == nil {
		s.show(false)
	}
}

func backgroundInvocation(args []string) bool { return len(args) == 1 && args[0] == "--background" }

// With no tray, keep the fallback window reachable even if the hotkey failed.
func (s *launcher) hide() {
	s.hiddenOpen = false
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
	s.presentation++
	s.appNeedsSelection = false
	s.searchWasComposing = false
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
	if s.win == nil {
		s.input.Focus()
		return
	}
	s.win.Center()
	s.input.Focus()
	s.win.Show()
	s.win.Invalidate()
}

func (s *launcher) scan() {
	s.scanSources(scanFiles | scanShell)
}

func (s *launcher) scanSources(sources uint8) {
	s.scanSourcesWithRoots(sources, nil)
}

func (s *launcher) scanSourcesWithRoots(sources uint8, requestedRoots []string) {
	if s.scanWork == nil || s.uiUpdate == nil {
		return
	}
	generation, start := s.scanQueue.Request()
	if !start {
		s.scanProgress = "正在扫描应用…（已合并重扫请求）"
		if s.scanCancel != nil {
			s.scanCancel()
		}
		if s.win != nil {
			s.win.Invalidate()
		}
		return
	}
	s.scanning = true
	s.autoDirty = false
	s.autoSources = 0
	s.autoRoots = nil
	s.autoAllFiles = false
	s.scanProgress = "正在扫描应用…"
	if s.win != nil {
		s.win.Invalidate()
	}
	timeout := s.scanTimeout
	if timeout <= 0 {
		timeout = 15 * time.Second
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.scanCancel = cancel
	roots := append([]string(nil), s.config.IndexRoots...)
	if len(roots) == 0 {
		roots = platform.DefaultAppRoots()
		ctx = context.WithValue(ctx, defaultShallowRootsKey{}, platform.DefaultDesktopRoots())
	}
	scanRoots := append([]string(nil), roots...)
	if len(requestedRoots) > 0 {
		scanRoots = nil
		for _, root := range roots {
			for _, requested := range requestedRoots {
				if strings.EqualFold(root, requested) {
					scanRoots = append(scanRoots, root)
					break
				}
			}
		}
	}
	previousApps := append([]core.Application(nil), s.apps...)
	previousIcons := make(map[string]*ui.Bitmap, len(s.icons))
	for id, icon := range s.icons {
		previousIcons[id] = icon
	}
	probeMissing := s.scanMissing
	shellEnabled, shellWork := s.config.AppsFolder, s.scanShellWork
	cacheEnabled, cacheStore := s.config.StartupSnapshot, s.snapshotStore
	cacheRead := cacheEnabled && cacheStore != nil && !s.snapshotAttempted
	if cacheRead {
		s.snapshotAttempted = true
	}
	cacheEpoch := uint64(0)
	if cacheStore != nil {
		cacheEpoch = cacheStore.Epoch()
	}
	var shallow []string
	if len(s.config.IndexRoots) == 0 {
		shallow = platform.DefaultDesktopRoots()
	}
	cacheFingerprint := settings.SourceFingerprint(roots, shallow, shellEnabled)
	go func() {
		if cacheRead && ctx.Err() == nil {
			value, loadErr := cacheStore.Load(cacheFingerprint, time.Now())
			s.uiUpdate(func() {
				if s.quitting || !s.scanQueue.Running || s.scanQueue.Generation != generation || cacheStore.Epoch() != cacheEpoch || !s.config.StartupSnapshot || len(s.apps) != 0 {
					return
				}
				if loadErr != nil {
					s.snapshotStatus = "未使用启动快照；正常发现应用。"
					return
				}
				s.pendingApps = snapshotDisplayApplications(value, s.config.Aliases)
				s.invalidateApplicationSearch()
				s.appNeedsSelection = true
				s.snapshotStatus = "快照显示待验证；验证前不能启动。"
			})
		}
		var apps []core.Application
		icons := map[string]*ui.Bitmap{}
		var errs []error
		filePartial := false
		fileCtx, fileCancel := context.WithTimeout(ctx, timeout)
		defer fileCancel()
		if sources&scanFiles != 0 {
			apps, icons, errs = s.scanWork(fileCtx, scanRoots)
			filePartial = len(errs) > 0 || fileCtx.Err() != nil
			if fileCtx.Err() != nil {
				errs = append(errs, fileCtx.Err())
			}
			for _, app := range previousApps {
				if app.ShellID == "" && core.PathInApplicationRoots(app.Path, roots) && !core.PathInApplicationRoots(app.Path, scanRoots) {
					apps = append(apps, app)
				}
			}
		} else {
			for _, app := range previousApps {
				if app.ShellID == "" && core.PathInApplicationRoots(app.Path, roots) {
					apps = append(apps, app)
				}
			}
		}
		var missing map[string]bool
		if filePartial && probeMissing != nil && fileCtx.Err() == nil && ctx.Err() == nil {
			var prior []core.Application
			for _, app := range previousApps {
				if app.ShellID == "" && core.PathInApplicationRoots(app.Path, scanRoots) {
					prior = append(prior, app)
				}
			}
			missing = probeMissing(fileCtx, prior)
		}
		fileCancel()
		shellPartial := shellEnabled && sources&scanShell != 0 && (shellWork == nil || ctx.Err() != nil)
		if shellEnabled && sources&scanShell == 0 {
			for _, app := range previousApps {
				if app.ShellID != "" {
					apps = append(apps, app)
				} else if app.ShellFallback != nil {
					apps = append(apps, *app.ShellFallback)
				}
			}
		}
		if shellEnabled && sources&scanShell != 0 && shellWork != nil && ctx.Err() == nil {
			shellCtx, shellCancel := context.WithTimeout(ctx, timeout)
			virtual, virtualIcons, virtualErrs := shellWork(shellCtx)
			shellPartial = len(virtualErrs) > 0 || shellCtx.Err() != nil
			if shellCtx.Err() != nil {
				virtualErrs = append(virtualErrs, shellCtx.Err())
			}
			shellCancel()
			apps = append(apps, virtual...)
			if icons == nil {
				icons = map[string]*ui.Bitmap{}
			}
			for id, icon := range virtualIcons {
				icons[id] = icon
			}
			errs = append(errs, virtualErrs...)
		}
		if shellEnabled && shellPartial && ctx.Err() != context.Canceled {
			// A failed namespace refresh cannot prove the cached counterpart gone,
			// even if the file provider confirms that its shortcut was removed.
			for _, app := range previousApps {
				if app.ShellFallback != nil {
					apps = append(apps, *app.ShellFallback)
				}
			}
		}
		if len(apps) > 5000 {
			apps = core.FairProviderLimit(apps, previousApps, 5000)
			errs = append(errs, core.ErrScanBudget)
			filePartial, shellPartial = true, shellEnabled
		}
		if (filePartial || shellPartial) && ctx.Err() != context.Canceled {
			apps = core.RetainUnconfirmed(apps, previousApps, missing, func(app core.Application) bool {
				if app.ShellID != "" {
					return shellEnabled && shellPartial
				}
				return filePartial && core.PathInApplicationRoots(app.Path, scanRoots)
			}, 5000)
		}
		workerErr := ctx.Err()
		apps = core.MergeProviderApplications(core.FilterUninstallEntries(apps))
		apps = core.CarryRelatedPreferenceIDs(apps, previousApps)
		apps = core.PrepareSearchContext(ctx, apps)
		if icons == nil {
			icons = map[string]*ui.Bitmap{}
		}
		for _, app := range apps {
			id := app.ItemID()
			if icons[id] == nil && previousIcons[id] != nil {
				icons[id] = previousIcons[id]
			}
		}
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
			s.scanProgress = ""
			bindingWarning := ""
			if workerErr != context.Canceled {
				apps, bindingWarning = s.persistProvedIdentityLinks(apps)
			}
			completeSources := workerErr == nil && !filePartial && !shellPartial && len(errs) == 0 && sources&scanFiles != 0 && len(scanRoots) == len(roots) && (!shellEnabled || sources&scanShell != 0)
			s.pendingComplete = completeSources
			s.completeScan(apps, icons, errs, workerErr, scanCompletion{missing: missing, reconciled: true})
			if completeSources && cacheEnabled && s.config.StartupSnapshot && cacheStore != nil && cacheEpoch == cacheStore.Epoch() && !s.snapshotClearing {
				s.queueSnapshot(snapshotJob{epoch: cacheEpoch, value: settings.NewSnapshot(apps, cacheFingerprint, time.Now())})
			}
			if bindingWarning != "" {
				s.status += "\n" + bindingWarning
				s.scanLastStatus = s.status
			}
			if sources&scanFiles != 0 && workerErr != context.Canceled {
				s.recordFileScanCompletion(scanRoots, filePartial)
			}
			if s.autoDirty {
				s.automaticRefreshRoots(s.autoSources, s.autoRoots)
			}
		})
	}()
}

func (s *launcher) readApps(ctx context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
	options := core.DefaultScanOptions()
	if shallow, ok := ctx.Value(defaultShallowRootsKey{}).([]string); ok {
		options.ShallowRoots = shallow
	}
	options.AllowPath = platform.LocalAppPath
	options.IncludeResolvedTarget = platform.IncludeApplicationTarget
	apps, errs := core.IndexApplicationsWithOptions(ctx, roots, platform.ShortcutIdentity, options)
	apps = core.FilterUninstallEntries(apps)
	apps = platform.LocalizedApplicationNames(ctx, apps)
	apps = core.PrepareSearchContext(ctx, apps)
	icons := map[string]*ui.Bitmap{}
	for i, app := range apps {
		if ctx.Err() != nil || i >= 256 {
			break
		}
		if im := s.readFileIcon(app.Launch.Target); im != nil {
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
		app := apps[s.selected]
		if app.DisplayOnlyID != "" {
			s.status = "快照入口待验证；请等待来源扫描完成后重新选择。"
			return
		}
		launch, identity := s.launchApplication, app.Path
		if app.ShellID != "" {
			launch, identity = s.launchShellApplication, app.ShellID
		}
		if launch == nil {
			s.status = "启动接口不可用。"
			return
		}
		if s.appNeedsSelection {
			s.status = "原应用已移除，请重新选择。"
			return
		}
		if err := launch(identity); err != nil {
			s.status = "启动失败：" + err.Error()
			s.scanFailure = s.status
			// Launch errors do not prove deletion. Refresh metadata through the
			// existing serialized worker and its conservative target probe.
			s.scan()
			return
		}
		s.recordRecent(app.ItemID())
		s.keepApplicationSelected(app.ItemID())
		s.hide()
	}
}
