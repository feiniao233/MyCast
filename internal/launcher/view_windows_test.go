//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
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

func TestLauncherLayoutAndScrollReset(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("scale-%.1f", scale), func(t *testing.T) {
			s := syntheticLauncher()
			tester := ui.NewTester(s.view, 680, 490)
			tester.SetScale(scale)
			search, ok := tester.Find("搜索")
			if !ok {
				t.Fatal("search field not found")
			}
			first, ok := tester.Find("Synthetic App 00")
			if !ok {
				t.Fatal("first row not found")
			}
			footer, ok := tester.Find("设置 ···")
			if !ok {
				t.Fatal("footer not found")
			}
			if search.Y < 0 || search.Y+search.H > 64 || first.Y < 96 || first.Y+first.H > 144 || footer.Y < 448 || footer.Y+footer.H > 490 {
				t.Fatalf("clipped layout: search=%+v first=%+v footer=%+v", search, first, footer)
			}
			savePreview(t, tester, fmt.Sprintf("launcher-%03d", int(scale*100)))
			tester.Scroll(400, 250, 0, 450)
			if s.scroll.Y == 0 {
				t.Fatal("list did not scroll")
			}
			s.resetResults()
			tester.Frame()
			first, ok = tester.Find("Synthetic App 00")
			if !ok || first.Y < 96 || first.Y+first.H > 144 || s.scroll.Y != 0 {
				t.Fatalf("first row clipped after reset: rect=%+v scroll=%v", first, s.scroll.Y)
			}
			s.query = "App 29"
			s.resetResults()
			tester.Frame()
			last, ok := tester.Find("Synthetic App 29")
			if !ok || last.Y < 96 || last.Y+last.H > 144 {
				t.Fatalf("filtered first row clipped: %+v", last)
			}
			s.switchMode(true)
			tester.Frame()
			if s.scroll.Y != 0 || !tester.HasText("暂无文本历史") {
				t.Fatal("mode switch did not reset view")
			}
			savePreview(t, tester, fmt.Sprintf("clipboard-empty-%03d", int(scale*100)))
		})
	}
}

func TestNamesakeSubtitleAndSelectedRow(t *testing.T) {
	s := syntheticLauncher()
	s.apps = []core.Application{
		{Name: "Synthetic Editor", Path: `C:\Synthetic\A\Editor.exe`},
		{Name: "Synthetic Editor", Path: `C:\Synthetic\B\Editor.exe`},
	}
	tester := ui.NewTester(s.view, 680, 490)
	if !tester.HasText(s.apps[0].Subtitle()) || !tester.HasText(s.apps[1].Subtitle()) {
		t.Fatal("namesakes do not show distinct subtitles")
	}
	tester.Key(0, ui.KeyDown)
	if s.selected != 1 {
		t.Fatal("keyboard selection did not change")
	}
	savePreview(t, tester, "namesakes")
}
