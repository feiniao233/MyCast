//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

// These focused fixtures depend only on public MyGo APIs and the production
// tooltip/tokens. They can also run as explicit named Go files on a non-Windows
// host; that proves renderer behavior, not Windows desktop interactions.
func TestCompactTooltipPlacementAndDismissalDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		for _, dark := range []bool{true, false} {
			for _, name := range []string{"menu", "status", "expand"} {
				t.Run(fmt.Sprintf("%s-dark-%t-%03d", name, dark, int(scale*100)), func(t *testing.T) {
					p := paletteFor(dark, false)
					labels := map[string]string{"menu": "菜单", "status": "查看完整状态详情", "expand": "展开结果（↓）"}
					query, clicks := "", 0
					disabled := false
					var searchID uint64
					build := func(c *ui.Context) {
						c.SetTheme(launcherTheme(p, dark))
						ui.Box(c.Key("root")).Fill().Background(p.Fallback).Clip().Children(func() {
							ui.ButtonBase(c.Key("header")).Absolute().Top(0).Left(0).Right(0).Height(p.CompactHeight).Padding(0, p.Inset).Gap(p.Inset).AlignItems(ui.Center).Children(func() {
								button := func(key, glyph string, size float32) {
									b := ui.ButtonBase(c.Key(key)).Size(size, size).Label(key).Disabled(disabled)
									compactButtonTooltip(c, b, labels[key], p)
									b.Children(func() { ui.Text(c, glyph) })
									if b.Clicked() {
										clicks++
									}
								}
								button("menu", "≡", p.CompactButton)
								field := ui.TextInputBase(c.Key("search"), &query).Grow(1).MinWidth(0).Height(p.Header).Label("search").AutoFocus()
								if searchID != 0 && field.ID() != searchID {
									t.Fatal("tooltip remounted search")
								}
								searchID = field.ID()
								button("status", "i", p.CompactStatus)
								button("expand", "↓", p.CompactButton)
							})
						})
					}
					tt := ui.NewTester(build, int(p.Width), int(p.CompactHeight))
					tt.SetScale(scale)
					before := image.NewRGBA(tt.Image().Bounds())
					copy(before.Pix, tt.Image().Pix)
					anchor, _ := tt.Find(name)
					tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
					if _, ok := tt.Find("提示：" + labels[name]); ok {
						t.Fatal("tip appeared before the hover delay")
					}
					// MyGo's public Tester uses its real clock. Do not reach into its
					// private runtime or install a second hover timer in production.
					time.Sleep(650 * time.Millisecond)
					tt.Frame()
					tip, ok := tt.Find("提示：" + labels[name])
					if !ok || tip.H != p.ControlLine+2*p.ControlInset || tip.Y < 0 || tip.Y+tip.H > p.CompactHeight || tip.X < 0 || tip.X+tip.W > p.Width {
						t.Fatal("tooltip is absent or clipped", tip)
					}
					if tip.X < anchor.X+anchor.W && tip.X+tip.W > anchor.X {
						t.Fatal("tooltip obscures its hovered control", tip, anchor)
					}
					if !tt.Focused("search") || query != "" {
						t.Fatal("hover changed search focus/query")
					}
					assertTooltipPainted(t, before, tt.Image(), tip, scale)
					saveTooltipPreview(t, tt, fmt.Sprintf("tooltip-placement-%s-dark-%t-%03d", name, dark, int(scale*100)))
					tt.Move(p.Width/2, p.CompactHeight/2)
					if _, ok := tt.Find("提示：" + labels[name]); ok {
						t.Fatal("tip persisted after moving away")
					}
					tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
					tt.Frame()
					if _, ok := tt.Find("提示：" + labels[name]); !ok {
						t.Fatal("tip did not return during MyGo's warm hover period")
					}
					tt.Key(0, ui.KeyEscape)
					tt.Frame()
					if _, ok := tt.Find("提示：" + labels[name]); ok || !tt.Focused("search") {
						t.Fatal("Escape did not dismiss tip while preserving search focus")
					}
					tt.Move(p.Width/2, p.CompactHeight/2)
					tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
					if err := tt.Click(name); err != nil {
						t.Fatal(err)
					}
					if _, ok := tt.Find("提示：" + labels[name]); ok || clicks != 1 {
						t.Fatal("click did not dismiss tip and reach the button", clicks)
					}
					time.Sleep(650 * time.Millisecond)
					tt.Frame()
					if _, ok := tt.Find("提示：" + labels[name]); ok {
						t.Fatal("tip reopened while pointer remained after press")
					}
					tt.Move(p.Width/2, p.CompactHeight/2)
					disabled = true
					tt.Frame()
					tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
					if _, ok := tt.Find("提示：" + labels[name]); ok {
						t.Fatal("disabled control exposed a stale tooltip")
					}
				})
			}
		}
	}
}

func TestCompactTooltipNarrowWidthDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		for _, left := range []bool{true, false} {
			t.Run(fmt.Sprintf("left-%t-%03d", left, int(scale*100)), func(t *testing.T) {
				const width = 320
				text := strings.Repeat("专用合成长提示", 10)
				p := paletteFor(true, false)
				tt := ui.NewTester(func(c *ui.Context) {
					c.SetTheme(launcherTheme(p, true))
					ui.Box(c).Fill().Background(p.Fallback).Clip().Children(func() {
						b := ui.ButtonBase(c.Key("anchor")).Absolute().Top((p.CompactHeight-p.CompactButton)/2).Size(p.CompactButton, p.CompactButton).Label("anchor")
						if left {
							b.Left(p.Inset)
						} else {
							b.Right(p.Inset)
						}
						compactButtonTooltip(c, b, text, p)
					})
				}, width, int(p.CompactHeight))
				tt.SetScale(scale)
				anchor, _ := tt.Find("anchor")
				tt.Move(anchor.X+anchor.W/2, anchor.Y+anchor.H/2)
				time.Sleep(650 * time.Millisecond)
				tt.Frame()
				tip, ok := tt.Find("提示：" + text)
				if !ok || tip.X < p.Inset || tip.X+tip.W > width-p.Inset || tip.Y < 0 || tip.Y+tip.H > p.CompactHeight || tip.H != p.ControlLine+2*p.ControlInset {
					t.Fatal("long tooltip overflowed narrow client area", tip)
				}
			})
		}
	}
}

func assertTooltipPainted(t *testing.T, before, after *image.RGBA, r ui.Rect, scale float32) {
	t.Helper()
	changed := 0
	for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
		for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
			if before.RGBAAt(x, y) != after.RGBAAt(x, y) {
				changed++
			}
		}
	}
	if changed < int(r.W*r.H*scale*scale/4) {
		t.Fatal("tooltip bounds exist but its surface was not painted", r, changed)
	}
}

func saveTooltipPreview(t *testing.T, tt *ui.Tester, name string) {
	t.Helper()
	if root := os.Getenv("MYCAST_PREVIEW_DIR"); root != "" {
		if err := os.MkdirAll(root, 0700); err != nil {
			t.Fatal(err)
		}
		file, err := os.Create(filepath.Join(root, name+".png"))
		if err != nil {
			t.Fatal(err)
		}
		defer file.Close()
		if err := png.Encode(file, tt.Image()); err != nil {
			t.Fatal(err)
		}
	}
}
