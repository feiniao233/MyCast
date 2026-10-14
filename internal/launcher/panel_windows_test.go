//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
	"github.com/feiniao233/MyCast/internal/settings"
	"math"
	"strings"
	"testing"
)

type panelHarness struct {
	s               *launcher
	tester          *ui.Tester
	queued          []func()
	resizes, shows  int
	maxHeight       int
	searchID        uint64
	searchFocused   bool
	searchComposing bool
}

func compactHarness(t *testing.T, scale float32, maxHeight int) *panelHarness {
	t.Helper()
	h := &panelHarness{s: syntheticLauncher(), maxHeight: maxHeight}
	h.s.config = settings.Defaults()
	h.s.config.CompactOnShow = true
	h.s.uiUpdate = func(fn func()) { h.queued = append(h.queued, fn) }
	h.s.resizePanel = func(layout core.PanelLayout, center bool) (platform.PaletteSize, error) {
		h.resizes++
		height := layout.Height
		if h.maxHeight > 0 {
			height = min(height, h.maxHeight)
		}
		if h.tester != nil {
			h.tester.SetSize(layout.Width, height)
		}
		return platform.PaletteSize{Width: float32(layout.Width), Height: float32(height)}, nil
	}
	h.s.showPanel = func() {
		h.shows++
		if h.s.panelApplied.Height != int(palette.CompactHeight) {
			t.Fatal("first Show flashed expanded height")
		}
	}
	h.s.show(false)
	h.tester = ui.NewTester(func(c *ui.Context) {
		h.s.view(c)
		field := c.Resolve(h.s.input)
		if !field.Valid() {
			t.Fatal("shared search input was unmounted")
		}
		// MyGo's ID includes the keyed parent path. Check every build, not
		// just focus at the end, so replacing/reparenting the editor fails.
		if h.searchID != 0 && field.ID() != h.searchID {
			t.Fatal("search input or its parent identity changed", h.searchID, field.ID())
		}
		h.searchID = field.ID()
		h.searchFocused = h.s.input.Focused(c)
		h.searchComposing = field.Composing()
	}, int(h.s.panelActual.Width), int(h.s.panelActual.Height))
	h.tester.SetScale(scale)
	h.drain(t)
	return h
}
func (h *panelHarness) drain(t *testing.T) {
	t.Helper()
	for i := 0; i < 8; i++ {
		if len(h.queued) == 0 {
			h.tester.Frame()
			if len(h.queued) == 0 {
				return
			}
		}
		work := h.queued
		h.queued = nil
		for _, fn := range work {
			fn()
		}
		h.tester.Frame()
	}
	t.Fatal("unbounded resize callbacks")
}

func TestCompactSingleRowStatusAndPointerAccessDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		for _, theme := range []string{"dark", "light"} {
			t.Run(fmt.Sprintf("%s-%v", theme, scale), func(t *testing.T) {
				h := compactHarness(t, scale, 0)
				s, tt := h.s, h.tester
				s.config.Theme = theme
				h.drain(t)
				if palette.CompactHeight < 52 || palette.CompactHeight > 60 || s.panelActual.Height != palette.CompactHeight {
					t.Fatal("compact must be one 52–60 DIP row", s.panelActual)
				}
				searchID := h.searchID
				search, ok := tt.Find("搜索")
				if !ok || !h.searchFocused || s.panelGeometryPending {
					t.Fatal("compact search/controls not ready")
				}
				previousRight := float32(0)
				for _, label := range []string{"菜单", "搜索", "状态详情", "展开结果"} {
					box, ok := tt.Find(label)
					if !ok || box.X < previousRight || box.X+box.W > palette.Width || box.Y < 0 || box.Y+box.H > palette.CompactHeight || math.Abs(float64(box.Y+box.H/2-palette.CompactHeight/2)) > .5 {
						t.Fatal("compact controls must share one unclipped row", label, box)
					}
					previousRight = box.X + box.W
				}
				for _, label := range []string{"底栏", "底栏主操作文字", "底栏快捷键胶囊", "打开应用", "结果：Synthetic App 00"} {
					if _, ok := tt.Find(label); ok {
						t.Fatal("compact mounted expanded content", label)
					}
				}
				if box, _ := tt.Find("菜单"); box.W != palette.CompactButton || box.H != palette.CompactButton {
					t.Fatal("compact menu is not a small button", box)
				}
				before := h.resizes
				s.scanning = true
				s.scanProgress = "正在扫描应用…" + strings.Repeat("专用合成扫描进度", 20)
				s.refreshApplications(s.apps, s.icons)
				h.drain(t)
				if _, ok := tt.Find("正在扫描应用"); !ok || tt.HasText(s.scanProgress) {
					t.Fatal("compact scan should show an indicator, not full progress text")
				}
				savePreview(t, tt, fmt.Sprintf("compact-single-row-scanning-%s-%03d", theme, int(scale*100)))
				s.scanning = false
				s.status = syntheticLongStatus()
				h.drain(t)
				if _, ok := tt.Find("有状态消息"); !ok || tt.HasText(s.status) {
					t.Fatal("compact diagnostics should stay behind the status indicator")
				}
				after, _ := tt.Find("搜索")
				if after != search || h.resizes != before || !s.panelLayout().Compact || h.searchID != searchID || !h.searchFocused {
					t.Fatal("scan/status refresh changed compact geometry or search identity")
				}
				savePreview(t, tt, fmt.Sprintf("compact-single-row-status-%s-%03d", theme, int(scale*100)))
				if err := tt.Click("状态详情"); err != nil {
					t.Fatal(err)
				}
				h.drain(t)
				if s.panelLayout().Compact || !s.statusDetailsOpen || s.statusDetail != s.status || h.searchID != searchID {
					t.Fatal("compact status detail did not force full panel/preserve editor")
				}
				tt.Key(0, ui.KeyEscape)
				h.drain(t)
				if !s.panelLayout().Compact || !h.searchFocused || h.searchID != searchID || s.panelGeometryPending {
					t.Fatal("status dismissal did not restore the compact editor")
				}
				if err := tt.Click("展开结果"); err != nil {
					t.Fatal(err)
				}
				h.drain(t)
				if s.panelLayout().Compact || s.selected != 0 || !h.searchFocused || h.searchID != searchID || s.panelGeometryPending {
					t.Fatal("compact expand pointer did not reveal first result/preserve editor")
				}
				if _, ok := tt.Find("底栏"); !ok {
					t.Fatal("expanded footer was not restored")
				}
			})
		}
	}
}

func TestPanelSizeObservationUsesCurrentLayoutMinimum(t *testing.T) {
	for _, test := range []struct {
		name                                        string
		compact                                     bool
		actualHeight, observedWidth, observedHeight float32
		ready                                       bool
	}{
		{"single row", true, 56, 750, 56, true},
		{"compact clipped", true, 55, 750, 55, false},
		{"stale expanded frame", true, 56, 750, 475, false},
		{"stale width", true, 56, 748, 56, false},
		{"expanded", false, 475, 750, 475, true},
		{"bounded expanded", false, 320, 750, 320, true},
		{"expanded clipped results", false, 96, 750, 96, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := syntheticLauncher()
			s.config.CompactOnShow = test.compact
			s.resizePanel = func(core.PanelLayout, bool) (platform.PaletteSize, error) { return platform.PaletteSize{}, nil }
			s.panelApplied = s.panelLayout()
			s.panelActual = platform.PaletteSize{Width: 750, Height: test.actualHeight}
			s.panelGeometryPending = true
			s.observePanelSize(test.observedWidth, test.observedHeight)
			if s.panelGeometryPending == test.ready {
				t.Fatal("incorrect geometry readiness", s.panelApplied, s.panelActual)
			}
		})
	}
}

func TestCompactDirectShowTypingDownAndHiddenEnterDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			h := compactHarness(t, scale, 0)
			s, tt := h.s, h.tester
			searchID := h.searchID
			launches := 0
			s.launchApplication = func(string) error { launches++; return nil }
			if h.shows != 1 || h.resizes != 1 || !s.panelLayout().Compact || !h.searchFocused || s.panelGeometryPending {
				t.Fatal("direct compact/focus", h.shows, h.resizes)
			}
			if _, ok := tt.Find("结果：Synthetic App 00"); ok {
				t.Fatal("invisible results remained interactive/mounted")
			}
			for _, label := range []string{"搜索", "菜单", "展开结果"} {
				box, ok := tt.Find(label)
				if !ok || box.Y < 0 || box.Y+box.H > palette.CompactHeight {
					t.Fatal("compact control clipped", label, box)
				}
			}
			tt.Key(0, ui.KeyEnter)
			s.activate(s.applicationResults(), nil, false)
			if launches != 0 || s.panelExpanded {
				t.Fatal("compact Enter activated invisible row")
			}
			savePreview(t, tt, fmt.Sprintf("compact-empty-%03d", int(scale*100)))
			tt.Key(0, ui.KeyDown)
			if !s.panelExpanded || s.selected != 0 || !s.panelGeometryPending {
				t.Fatal("Down did not explicitly choose first/pending layout")
			}
			tt.Key(0, ui.KeyEnter)
			if launches != 0 {
				t.Fatal("transition Enter launched before visibility")
			}
			h.drain(t)
			if s.panelGeometryPending || !h.searchFocused || h.searchID != searchID || s.selected != 0 {
				t.Fatal("expanded focus/first selection")
			}
			if _, ok := tt.Find("结果：Synthetic App 00"); !ok {
				t.Fatal("expanded results missing")
			}
			savePreview(t, tt, fmt.Sprintf("compact-expanded-%03d", int(scale*100)))
			before := h.resizes
			for i := 0; i < 12; i++ {
				tt.Frame()
			}
			s.refreshApplications(s.apps, s.icons)
			tt.Frame()
			h.drain(t)
			if h.resizes != before {
				t.Fatal("refresh/render repeatedly resized")
			}
			s.show(false)
			h.drain(t)
			if !s.panelLayout().Compact || s.panelExpanded {
				t.Fatal("reopen did not reset temporary expansion")
			}
			tt.Type("Synthetic App 0")
			h.drain(t)
			if s.panelLayout().Compact || s.query != "Synthetic App 0" || !h.searchFocused || h.searchID != searchID {
				t.Fatal("typing expansion lost input/focus")
			}
		})
	}
}

func TestCompactIMEAndFullPanelAccessDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			h := compactHarness(t, scale, 0)
			s, tt := h.s, h.tester
			searchID := h.searchID
			launches := 0
			s.launchApplication = func(string) error { launches++; return nil }
			tt.Compose("合成", 2)
			h.drain(t)
			if s.panelLayout().Compact || !s.searchWasComposing || !h.searchComposing || !h.searchFocused || h.searchID != searchID {
				t.Fatal("composition remounted/lost editor")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Type("合成")
			tt.Compose("", 0)
			h.drain(t)
			if launches != 0 || !h.searchFocused || h.searchID != searchID {
				t.Fatal("candidate confirmation leaked activation/focus")
			}
			s.show(false)
			h.drain(t)
			if err := tt.Click("菜单"); err != nil {
				t.Fatal(err)
			}
			h.drain(t)
			if s.panelLayout().Compact || s.menu != menuApplication {
				t.Fatal("application menu not accessible from compact")
			}
			if err := tt.Click("菜单项：设置"); err != nil {
				t.Fatal(err)
			}
			h.drain(t)
			if !s.settingsOpen || s.panelLayout().Compact {
				t.Fatal("settings not full")
			}
			savePreview(t, tt, fmt.Sprintf("compact-settings-%03d", int(scale*100)))
			tt.Key(0, ui.KeyEscape)
			h.drain(t)
			if s.settingsOpen || !tt.Focused("搜索") || !s.panelLayout().Compact {
				t.Fatal("settings Escape/focus/return state")
			}
			s.switchMode(true)
			tt.Frame()
			h.drain(t)
			if s.panelLayout().Compact {
				t.Fatal("clipboard inaccessible")
			}
			s.switchMode(false)
			tt.Frame()
			h.drain(t)
			if !s.panelLayout().Compact {
				t.Fatal("return to applications did not reset compact state")
			}
		})
	}
}

func TestCompactConstrainedSettingsMenuAndRefreshDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			h := compactHarness(t, scale, 320)
			s, tt := h.s, h.tester
			tt.Key(0, ui.KeyDown)
			h.drain(t)
			tt.Key(0, ui.KeyDown)
			tt.Scroll(400, 150, 0, 320)
			selected, offset, before := s.selectedApplicationID(), s.scroll.Y, h.resizes
			s.refreshApplications(s.apps, s.icons)
			tt.Frame()
			h.drain(t)
			if s.selectedApplicationID() != selected || s.scroll.Y != offset || h.resizes != before {
				t.Fatal("refresh moved selection/viewport/size")
			}
			if err := tt.Click("菜单"); err != nil {
				t.Fatal(err)
			}
			h.drain(t)
			items := s.menuItems(s.applicationResults(), nil)
			for i := 0; i < len(items)-1; i++ {
				tt.Key(0, ui.KeyDown)
			}
			if s.menuScroll.Y <= 0 {
				t.Fatal("constrained menu did not exercise scrolling")
			}
			box, ok := tt.Find("菜单项：" + items[len(items)-1].text)
			if !ok || box.Y < palette.Header || box.Y+box.H > 320-palette.Footer {
				t.Fatal("last menu item clipped", box)
			}
			savePreview(t, tt, fmt.Sprintf("compact-constrained-menu-%03d", int(scale*100)))
			tt.Key(0, ui.KeyEscape)
			h.drain(t)
			if !tt.Focused("搜索") {
				t.Fatal("menu dismissal focus")
			}
			s.openSettings()
			tt.Frame()
			h.drain(t)
			for _, label := range []string{"保存设置", "取消设置"} {
				box, ok := tt.Find(label)
				if !ok || box.Y+box.H > 308 || box.Y < 12 {
					t.Fatal("settings action clipped", label, box)
				}
			}
			tt.Scroll(400, 170, 0, 800)
			if _, ok := tt.Find("应用目录设置"); !ok {
				t.Fatal("last editable settings control inaccessible")
			}
			savePreview(t, tt, fmt.Sprintf("compact-constrained-settings-%03d", int(scale*100)))
			tt.Key(0, ui.KeyEscape)
			h.drain(t)
			if s.settingsOpen || !tt.Focused("搜索") {
				t.Fatal("constrained settings close")
			}
		})
	}
}

func TestCompactPreferenceSaveKeepsActiveResultsAndCancel(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.applyPreferences = func(v settings.Settings) (settings.Settings, error) { return v, nil }
	tt := newPaletteTester(s, 1)
	s.openSettings()
	tt.Frame()
	tt.Click("紧凑呼出")
	tt.Click("取消设置")
	if s.config.CompactOnShow || s.panelLayout().Compact {
		t.Fatal("cancel changed ordinary preference")
	}
	s.openSettings()
	tt.Frame()
	tt.Click("紧凑呼出")
	tt.Click("保存设置")
	if !s.config.CompactOnShow || s.panelLayout().Compact || !s.panelExpanded {
		t.Fatal("saving opt-in hid active results")
	}
	s.show(false)
	tt.SetSize(int(palette.Width), int(palette.CompactHeight))
	tt.Frame()
	if !s.panelLayout().Compact {
		t.Fatal("next call did not apply explicit opt-in")
	}
}

func TestCompactFirstLastRowsAndKeyboardMenuDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5, 2} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			h := compactHarness(t, scale, 320)
			s, tt := h.s, h.tester
			tt.Key(0, ui.KeyDown)
			h.drain(t)
			for i := 0; i < len(s.apps)-1; i++ {
				tt.Key(0, ui.KeyDown)
			}
			selected := s.applicationResults()[s.selected]
			box, ok := tt.Find("结果：" + selected.Name)
			if !ok || box.Y < palette.Header+palette.Inset || box.Y+box.H > 320-palette.Footer-palette.Inset {
				t.Fatal("last row clipped", box)
			}
			before := h.resizes
			tt.Key(ui.Shift, ui.KeyF10)
			h.drain(t)
			if s.menu != menuActions || s.menuTargetKey != selected.ItemID() || h.resizes != before {
				t.Fatal("keyboard menu used stale geometry/identity")
			}
			for _, item := range s.menuItems(s.applicationResults(), nil) {
				box, ok := tt.Find("菜单项：" + item.text)
				if !ok || box.Y < palette.Header || box.Y+box.H > 320-palette.Footer {
					t.Fatal("result menu clipped", item.text, box)
				}
			}
			tt.Key(0, ui.KeyEscape)
			h.drain(t)
			tt.Key(0, ui.KeyDown)
			box, ok = tt.Find("结果：" + s.applicationResults()[0].Name)
			if s.selected != 0 || !ok || box.Y < palette.Header+palette.Inset || box.Y+box.H > 320-palette.Footer-palette.Inset {
				t.Fatal("first resting row clipped", box)
			}
		})
	}
}

func TestCompactResizeFailureKeepsFullPanelAndNoRetryLoop(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.CompactOnShow = true
	calls := 0
	s.resizePanel = func(core.PanelLayout, bool) (platform.PaletteSize, error) {
		calls++
		return platform.PaletteSize{}, errors.New("controlled geometry failure")
	}
	s.show(false)
	tt := newPaletteTester(s, 1)
	if !s.panelUnavailable || s.panelLayout().Compact || s.panelGeometryPending || !s.config.CompactOnShow {
		t.Fatal("full/access/settings fallback failed")
	}
	for i := 0; i < 20; i++ {
		tt.Frame()
	}
	if calls != 1 {
		t.Fatal("failed adapter retried every render", calls)
	}
}

func TestCompactRepeatedCyclesReuseIconsAndBoundCallbacks(t *testing.T) {
	h := compactHarness(t, 1, 0)
	s, tt := h.s, h.tester
	icons := map[string]*ui.Bitmap{}
	for id, bitmap := range s.icons {
		icons[id] = bitmap
	}
	for cycle := 0; cycle < 32; cycle++ {
		s.expandPanelFirst()
		h.drain(t)
		s.panelExpanded = false
		tt.Frame()
		h.drain(t)
		if len(h.queued) != 0 || !s.panelLayout().Compact || len(s.icons) != len(icons) {
			t.Fatal("persistent queued work/icon growth")
		}
		for id, bitmap := range icons {
			if s.icons[id] != bitmap {
				t.Fatal("resize decoded/replaced cached icon")
			}
		}
	}
	if h.resizes != 65 {
		t.Fatal("unexpected repeated native updates", h.resizes)
	}
}
