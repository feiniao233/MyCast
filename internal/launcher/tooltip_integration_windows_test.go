//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"image"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestCompactHoverTooltipsDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		for _, test := range []struct{ name, control, text string }{
			{"menu", "菜单", "菜单"},
			{"status", "状态详情", "查看完整状态详情"},
			{"expand", "展开结果", "展开结果（↓）"},
		} {
			t.Run(fmt.Sprintf("%s-%03d", test.name, int(scale*100)), func(t *testing.T) {
				h := compactHarness(t, scale, 0)
				s, tt := h.s, h.tester
				s.status = "专用合成状态消息"
				h.drain(t)
				searchID, resizes := h.searchID, h.resizes
				search, _ := tt.Find("搜索")
				anchor, _ := tt.Find(test.control)
				before := image.NewRGBA(tt.Image().Bounds())
				copy(before.Pix, tt.Image().Pix)
				tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
				if _, ok := tt.Find("提示：" + test.text); ok {
					t.Fatal("compact tooltip skipped deliberate hover delay")
				}
				time.Sleep(650 * time.Millisecond)
				tt.Frame()
				tip, ok := tt.Find("提示：" + test.text)
				if !ok || tip.H != palette.ControlLine+2*palette.ControlInset || tip.X < 0 || tip.X+tip.W > s.panelActual.Width || tip.Y < 0 || tip.Y+tip.H > s.panelActual.Height {
					t.Fatal("compact tooltip is absent or clipped", tip)
				}
				assertTooltipPainted(t, before, tt.Image(), tip, scale)
				savePreview(t, tt, fmt.Sprintf("compact-hover-%s-%03d", test.name, int(scale*100)))
				// hide always clears calcText, even in a headless fixture with no
				// HWND. The sentinel detects Escape falling through the tooltip.
				s.calcText = "synthetic hide sentinel"
				tt.Key(0, ui.KeyEscape)
				h.drain(t)
				if _, ok := tt.Find("提示：" + test.text); ok || s.calcText == "" || !h.searchFocused {
					t.Fatal("Escape did not dismiss only the tooltip")
				}
				tt.Move(search.X+search.W/2, search.Y+search.H/2)
				tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
				s.scanning, s.scanProgress = true, "专用合成扫描进度"
				h.drain(t)
				after, _ := tt.Find("搜索")
				if search != after || h.searchID != searchID || !h.searchFocused || h.resizes != resizes || s.panelActual.Height != palette.CompactHeight {
					t.Fatal("hover or scan refresh disturbed compact input/geometry")
				}
				tt.Move(search.X+search.W/2, search.Y+search.H/2)
				if _, ok := tt.Find("提示：" + test.text); ok {
					t.Fatal("compact tooltip persisted after moving away")
				}
				tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
				if err := tt.Click(test.control); err != nil {
					t.Fatal(err)
				}
				h.drain(t)
				if _, ok := tt.Find("提示：" + test.text); ok || s.panelLayout().Compact || h.searchID != searchID {
					t.Fatal("tooltip survived expansion or remounted input")
				}
				switch test.name {
				case "menu":
					if s.menu != menuApplication {
						t.Fatal("hover blocked menu action")
					}
				case "status":
					if !s.statusDetailsOpen {
						t.Fatal("hover blocked status details")
					}
				case "expand":
					if !h.searchFocused || s.selected != 0 {
						t.Fatal("hover blocked expansion or focus restore")
					}
				}
			})
		}
	}
}

func TestCompactTooltipCompositionPreservesEditorDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			h := compactHarness(t, scale, 0)
			s, tt := h.s, h.tester
			searchID, launches := h.searchID, 0
			s.launchApplication = func(string) error { launches++; return nil }
			anchor, _ := tt.Find("展开结果")
			tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
			time.Sleep(650 * time.Millisecond)
			tt.Frame()
			if _, ok := tt.Find("提示：展开结果（↓）"); !ok {
				t.Fatal("missing tooltip before composition")
			}
			tt.Compose("合成", 2)
			h.drain(t)
			if _, ok := tt.Find("提示：展开结果（↓）"); ok || !h.searchComposing || !h.searchFocused || h.searchID != searchID || s.panelLayout().Compact {
				t.Fatal("tooltip/composition transition lost input or stayed compact")
			}
			tt.Key(0, ui.KeyEnter)
			tt.Type("合成")
			tt.Compose("", 0)
			h.drain(t)
			if launches != 0 || h.searchID != searchID || !h.searchFocused {
				t.Fatal("tooltip leaked candidate confirmation into activation")
			}
		})
	}
}

func TestExpandedHoverTooltipsDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		for _, test := range []struct{ name, control, text string }{
			{"menu", "菜单", "菜单"},
			{"status", "状态详情", "查看完整状态详情"},
		} {
			t.Run(fmt.Sprintf("%s-%03d", test.name, int(scale*100)), func(t *testing.T) {
				s := syntheticLauncher()
				s.status = "专用合成状态消息"
				tt := newPaletteTester(s, scale)
				anchor, _ := tt.Find(test.control)
				countText := func() int {
					count := 0
					for _, text := range tt.Texts() {
						if text == test.text {
							count++
						}
					}
					return count
				}
				before := countText()
				tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
				time.Sleep(650 * time.Millisecond)
				tt.Frame()
				if countText() != before+1 || !tt.Focused("搜索") {
					t.Fatal("expanded tooltip is absent or stole focus")
				}
				savePreview(t, tt, fmt.Sprintf("expanded-hover-%s-%03d", test.name, int(scale*100)))
				tt.Move(palette.Width/2, palette.Header/2)
				if countText() != before {
					t.Fatal("expanded tooltip persisted after moving away")
				}
			})
		}
	}
}
