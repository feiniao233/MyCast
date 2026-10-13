//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
)

func syntheticLauncher() *launcher {
	s := &launcher{history: core.NewHistory(50, 64*1024), icons: map[string]*ui.Bitmap{}}
	for i := 0; i < 30; i++ {
		app := core.Application{Name: fmt.Sprintf("Synthetic App %02d", i), Path: fmt.Sprintf(`C:\Synthetic\App%02d.exe`, i)}
		s.apps = append(s.apps, app)
		im := image.NewRGBA(image.Rect(0, 0, 32, 32))
		for y := 0; y < 32; y++ {
			for x := 0; x < 32; x++ {
				im.SetRGBA(x, y, color.RGBA{R: uint8(60 + i*5), G: 125, B: 205, A: 255})
			}
		}
		s.icons[app.ItemID()] = ui.NewBitmap(im)
	}
	return s
}

func savePreview(t *testing.T, tester *ui.Tester, name string) {
	t.Helper()
	root := os.Getenv("MYCAST_PREVIEW_DIR")
	if root == "" {
		return
	}
	if err := os.MkdirAll(root, 0700); err != nil {
		t.Fatal(err)
	}
	file, err := os.Create(filepath.Join(root, name+".png"))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	if err := png.Encode(file, tester.Image()); err != nil {
		t.Fatal(err)
	}
}

func newPaletteTester(s *launcher, scale float32) *ui.Tester {
	tester := ui.NewTester(s.view, int(palette.Width), int(palette.Height))
	tester.SetScale(scale)
	return tester
}

func TestPaletteLayoutScrollAndModes(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			tester := newPaletteTester(s, scale)
			search, ok := tester.Find("搜索")
			if !ok {
				t.Fatal("missing search")
			}
			first, ok := tester.Find("Synthetic App 00")
			if !ok {
				t.Fatal("missing first title")
			}
			circle, ok := tester.Find("菜单")
			if !ok {
				t.Fatal("missing menu circle")
			}
			if search.Y != 0 || search.H != palette.Header || first.X != search.X || first.Y < palette.Header || first.Y+first.H > palette.Height-palette.Footer || circle.W != palette.MenuButton || circle.H != palette.MenuButton || circle.Y != palette.Height-palette.Footer+palette.Inset {
				t.Fatalf("bad geometry/alignment: search=%+v first=%+v circle=%+v", search, first, circle)
			}
			savePreview(t, tester, fmt.Sprintf("launcher-%03d", int(scale*100)))
			tester.Scroll(450, 200, 0, 400)
			if s.scroll.Y == 0 {
				t.Fatal("list did not scroll")
			}
			manual := s.scroll.Y
			tester.Frame()
			if s.scroll.Y != manual {
				t.Fatal("wheel scroll was pulled back")
			}
			// Up wraps to the last result. It must be completely selectable above
			// the footer; Down then restores the origin including the section label.
			tester.Key(0, ui.KeyUp)
			last, ok := tester.Find("Synthetic App 29")
			if !ok || s.selected != 29 || last.Y < palette.Header || last.Y+last.H > palette.Height-palette.Footer-palette.Inset {
				t.Fatalf("last row is obscured: %+v, selected=%d scroll=%v", last, s.selected, s.scroll.Y)
			}
			savePreview(t, tester, fmt.Sprintf("last-selected-%03d", int(scale*100)))
			tester.Key(0, ui.KeyDown)
			if s.selected != 0 || s.scroll.Y != 0 {
				t.Fatal("first row did not return to origin")
			}
			s.query = "App 29"
			s.resetResults()
			tester.Frame()
			last, ok = tester.Find("Synthetic App 29")
			if !ok || last.Y < palette.Header || s.scroll.Y != 0 {
				t.Fatal("filter reset clipped first result")
			}
			s.switchMode(true)
			tester.Frame()
			if !tester.HasText("暂无文本历史") || s.scroll.Y != 0 || !tester.Focused("搜索") {
				t.Fatal("mode switch did not restore empty view and focus")
			}
			savePreview(t, tester, fmt.Sprintf("clipboard-empty-%03d", int(scale*100)))
			s.history.Add("Synthetic clipboard sample A")
			s.history.Add("Synthetic clipboard sample B")
			tester.Frame()
			clip, ok := tester.Find("Synthetic clipboard sample B")
			search, _ = tester.Find("搜索")
			if !ok || clip.X != search.X {
				t.Fatalf("clipboard title column moved: clip=%+v search=%+v", clip, search)
			}
			savePreview(t, tester, fmt.Sprintf("clipboard-%03d", int(scale*100)))
		})
	}
}

func TestPaletteMenusConsumeClickAndRestoreFocus(t *testing.T) {
	s := syntheticLauncher()
	s.query = "Synthetic"
	tester := newPaletteTester(s, 1)
	if err := tester.Click("菜单"); err != nil {
		t.Fatal(err)
	}
	if s.menu != menuApplication || !tester.HasText("开启文本历史") || !tester.HasText("退出 MyCast") {
		t.Fatal("real app menu entries missing")
	}
	tester.Type("unwanted")
	if s.query != "Synthetic" {
		t.Fatal("palette query changed while menu owns input")
	}
	savePreview(t, tester, "menu-100")
	tester.SetScale(1.5)
	savePreview(t, tester, "menu-150")
	tester.ClickAt(450, 180)
	if s.menu != menuClosed || s.query != "Synthetic" || !tester.Focused("搜索") || s.status != "" {
		t.Fatal("click-away leaked to a result or did not restore focus")
	}
	if err := tester.Click("菜单"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyEscape)
	if s.menu != menuClosed || !tester.Focused("搜索") {
		t.Fatal("Escape did not close menu and restore focus")
	}
	if err := tester.Click("菜单"); err != nil {
		t.Fatal(err)
	}
	if err := tester.Click("菜单项：剪贴板"); err != nil {
		t.Fatal(err)
	}
	if !s.clipboardMode || s.menu != menuClosed || !tester.Focused("搜索") {
		t.Fatal("menu mode action/focus failed")
	}
	if err := tester.Click("操作"); err != nil {
		t.Fatal(err)
	}
	if s.menu != menuActions || !tester.HasText("仅复制") {
		t.Fatal("clipboard copy-only action missing")
	}
	tester.Key(0, ui.KeyEscape)
	if !tester.Focused("搜索") {
		t.Fatal("action menu close lost focus")
	}
}

func TestPaletteMaterialFallbackAndCornerPixels(t *testing.T) {
	s := syntheticLauncher()
	s.apps = nil
	tester := newPaletteTester(s, 1)
	_, _, _, corner := tester.Image().At(0, 0).RGBA()
	_, _, _, center := tester.Image().At(700, 250).RGBA()
	// The client background fills the corners; only DWM may clip the HWND.
	if corner != 65535 || center != 65535 {
		t.Fatalf("fallback/corner alpha wrong: corner=%d body=%d", corner, center)
	}
	tester.SetVibrancy(true)
	_, _, _, center = tester.Image().At(700, 250).RGBA()
	if center == 0 || center >= 65535 {
		t.Fatalf("opaque root masks material: alpha=%d", center)
	}
	savePreview(t, tester, "material-layer-simulated-100")
}

func TestPaletteFooterOpticalLayoutAndNativeFallback(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("dark-%v-dpi-%03d", dark, int(scale*100)), func(t *testing.T) {
				s := syntheticLauncher()
				if dark {
					s.config.Theme = "dark"
				} else {
					s.config.Theme = "light"
				}
				tester := newPaletteTester(s, scale)
				center := palette.Height - palette.Footer/2
				for _, label := range []string{"底栏菜单图标", "底栏主操作文字", "底栏操作文字", "底栏快捷键胶囊", "底栏回车图标"} {
					r, ok := tester.Find(label)
					if !ok || r.Y+r.H/2 < center-.75 || r.Y+r.H/2 > center+.75 {
						t.Fatalf("footer center: %s %+v", label, r)
					}
				}
				// Check painted CJK ink as well as its line box. A geometrically
				// centered box alone misses fallback-font baseline drift/cropping.
				painted := tester.Image()
				for _, label := range []string{"底栏主操作文字", "底栏操作文字"} {
					r, _ := tester.Find(label)
					minY, maxY := painted.Bounds().Dy(), -1
					for y := int(r.Y * scale); y < int((r.Y+r.H)*scale); y++ {
						for x := int(r.X * scale); x < int((r.X+r.W)*scale); x++ {
							pixel := painted.RGBAAt(x, y)
							// The secondary label has 60% ink opacity, unlike the
							// primary text. Both separate from the capsule surface.
							ink := dark && pixel.R > 110 && pixel.G > 110 && pixel.B > 110 || !dark && pixel.R < 145 && pixel.G < 145 && pixel.B < 145
							if ink {
								minY = min(minY, y)
								maxY = max(maxY, y)
							}
						}
					}
					inkCenter := float32(minY+maxY+1) / (2 * scale)
					if maxY < 0 || math.Abs(float64(inkCenter-center)) > 2.5 || minY <= int(r.Y*scale) || maxY >= int((r.Y+r.H)*scale)-1 {
						t.Fatalf("CJK ink drift/clipping: %s [%d,%d] box %+v", label, minY, maxY, r)
					}
				}
				savePreview(t, tester, fmt.Sprintf("feedback-footer-dark-%v-%03d", dark, int(scale*100)))
				tester.SetVibrancy(true)
				if tester.Image().RGBAAt(0, 0).A == 0 {
					t.Fatal("renderer cutout exposes native backdrop corner")
				}
				s.appearanceChecked, s.nativeAcrylic = true, false
				tester.Frame()
				for _, point := range [][2]int{{0, 0}, {int(palette.Width*scale) - 1, 0}, {0, int(palette.Height*scale) - 1}, {int(palette.Width*scale) - 1, int(palette.Height*scale) - 1}} {
					if pixel := tester.Image().RGBAAt(point[0], point[1]); pixel.A != 255 {
						t.Fatalf("unsupported Acrylic not opaque: %+v", pixel)
					}
				}
			})
		}
	}
}

func TestPaletteMenuKeyboardExecutesHighlightedItem(t *testing.T) {
	s := syntheticLauncher()
	tester := newPaletteTester(s, 1)
	if err := tester.Click("菜单"); err != nil {
		t.Fatal(err)
	}
	if !tester.Focused("菜单面板") {
		t.Fatal("menu panel does not own keyboard focus")
	}
	tester.Key(0, ui.KeyDown)
	if s.menu != menuApplication || s.menuSelected != 1 {
		t.Fatal("Down did not highlight clipboard menu entry")
	}
	tester.Key(0, ui.KeyEnter)
	if !s.clipboardMode || s.menu != menuClosed || !tester.Focused("搜索") {
		t.Fatalf("Enter did not execute highlighted entry: clipboard=%v menu=%d searchFocus=%v", s.clipboardMode, s.menu, tester.Focused("搜索"))
	}
	if err := tester.Click("操作"); err != nil {
		t.Fatal(err)
	}
	tester.Key(0, ui.KeyDown)
	if s.menu != menuActions || s.menuSelected != 1 {
		t.Fatal("action menu navigation failed")
	}
	tester.Key(0, ui.KeyEscape)
	if s.menu != menuClosed || !tester.Focused("搜索") {
		t.Fatal("keyboard close did not restore search focus")
	}
	if err := tester.Click("菜单"); err != nil {
		t.Fatal(err)
	}
	// Disabled rows must not dismiss through the click-away catcher or move
	// keyboard focus back to a clickable footer control.
	if err := tester.Click("菜单项：清空内存历史"); err != nil {
		t.Fatal(err)
	}
	if s.menu != menuApplication || !tester.Focused("菜单面板") {
		t.Fatal("disabled menu row lost the menu/focus owner")
	}
	tester.Key(0, ui.KeyDown)
	tester.Key(0, ui.KeyUp)
	tester.Key(0, ui.KeyEnter)
	if s.clipboardMode || s.menu != menuClosed || !tester.Focused("搜索") {
		t.Fatal("Up/Enter did not return to application mode")
	}
}

func TestRowDissolveKeepsRestingEdgesOpaque(t *testing.T) {
	if rowOpacity(0, 38, 475, 0, 0) != 1 {
		t.Fatal("non-scrolling list was faded")
	}
	if rowOpacity(76, 38, 475, 0, 100) != 1 {
		t.Fatal("resting first row was faded")
	}
	if rowOpacity(385, 38, 475, 100, 0) != 1 {
		t.Fatal("resting last row was faded")
	}
	if top := rowOpacity(0, 38, 475, 100, 100); top < palette.TopFloor-.001 || top >= 1 {
		t.Fatal("top dissolve missing")
	}
	if bottom := rowOpacity(437, 38, 475, 100, 100); bottom < palette.BottomFloor-.001 || bottom >= 1 {
		t.Fatal("bottom dissolve missing")
	}
}

func TestNamesakeSubtitleAndIMESelection(t *testing.T) {
	s := syntheticLauncher()
	s.apps = []core.Application{{Name: "Synthetic Editor", Path: `C:\Synthetic\A\Editor.exe`}, {Name: "Synthetic Editor", Path: `C:\Synthetic\B\Editor.exe`}}
	tester := newPaletteTester(s, 1)
	if !tester.HasText(s.apps[0].Subtitle()) || !tester.HasText(s.apps[1].Subtitle()) {
		t.Fatal("namesakes need distinct subtitles")
	}
	tester.Compose("合成", 2)
	tester.Key(0, ui.KeyDown)
	if s.selected != 0 {
		t.Fatal("IME candidate navigation reached launcher")
	}
	savePreview(t, tester, "namesakes-ime")
}
