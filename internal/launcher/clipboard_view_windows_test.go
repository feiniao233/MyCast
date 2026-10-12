//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
)

func TestFullClipboardPreviewCopyDeleteAndDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.clipboardMode = true
			text := "Dedicated synthetic preview\r\n合成文本 😀\n" + strings.Repeat("Synthetic long line\n", 30) + "DEDICATED END MARKER"
			s.history.Add(text)
			var copied []string
			s.copyText = func(value string) error { copied = append(copied, value); return nil }
			tester := newPaletteTester(s, scale)
			tester.Click("操作")
			tester.Click("菜单项：预览全文")
			savePreview(t, tester, fmt.Sprintf("text-preview-%03d", int(scale*100)))
			if !s.previewOpen || !tester.HasText(text) || !tester.Focused("文本预览面板") {
				t.Fatalf("full preview/focus missing: open=%v fullText=%v focus=%v", s.previewOpen, tester.HasText(text), tester.Focused("文本预览面板"))
			}
			for _, label := range []string{"复制预览文本", "关闭文本预览", "删除预览条目"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < 18 || box.Y+box.H > palette.Height-18 {
					t.Fatalf("preview action clipped: %s %+v", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("text-preview-%03d", int(scale*100)))
			tester.Key(0, ui.KeyEnter)
			if len(copied) != 0 || !s.previewOpen {
				t.Fatal("preview Enter copied without explicit action")
			}
			tester.Scroll(350, 200, 0, 9000)
			if s.previewScroll.MaxY <= 0 || s.previewScroll.Y != s.previewScroll.MaxY {
				t.Fatal("full preview end was not reachable")
			}
			savePreview(t, tester, fmt.Sprintf("text-preview-end-%03d", int(scale*100)))
			tester.Click("复制预览文本")
			if len(copied) != 1 || copied[0] != text || !s.previewOpen {
				t.Fatal("explicit copy lost exact text")
			}
			tester.Click("删除预览条目")
			tester.Frame()
			if s.history.Len() != 0 || s.previewOpen || s.clipPreview != "" || !tester.Focused("搜索") {
				t.Fatal("delete retained history/preview references or focus")
			}
			s.history.Add("Dedicated second preview")
			s.openClipPreview("Dedicated second preview")
			tester.Frame()
			tester.Key(0, ui.KeyEscape)
			if s.previewOpen || s.clipPreview != "" || !tester.Focused("搜索") {
				t.Fatal("preview Escape did not close/release")
			}
		})
	}
}

func TestClipboardNewEntryKeepsHighlightedIdentityAndClearReleasesPreview(t *testing.T) {
	s := syntheticLauncher()
	s.clipboardMode = true
	s.history.Add("Dedicated A")
	s.history.Add("Dedicated B")
	tester := newPaletteTester(s, 1)
	tester.Key(0, ui.KeyDown)
	if s.clipResults()[s.selected] != "Dedicated A" {
		t.Fatal("non-first selection failed")
	}
	s.history.Add("Dedicated C")
	tester.Frame()
	if s.clipResults()[s.selected] != "Dedicated A" {
		t.Fatal("future entry displaced highlighted text")
	}
	s.openClipPreview("Dedicated A")
	tester.Frame()
	s.clearHistory()
	tester.Frame()
	if s.previewOpen || s.clipPreview != "" || s.history.Len() != 0 {
		t.Fatal("whole clear retained preview text")
	}
	tester.Click("菜单")
	tester.Click("菜单项：保留：退出前")
	if s.retention != 10*time.Minute || s.recording {
		t.Fatal("retention change altered recording state")
	}
}

func TestRemovedHighlightedTextDoesNotCopyReplacementOnSameEnter(t *testing.T) {
	s := syntheticLauncher()
	s.clipboardMode = true
	s.history.Add("Dedicated old selection")
	s.history.Add("Dedicated remaining")
	calls := 0
	s.copyText = func(string) error { calls++; return nil }
	tester := newPaletteTester(s, 1)
	tester.Key(0, ui.KeyDown)
	s.history.Delete("Dedicated old selection")
	tester.Key(0, ui.KeyEnter)
	if calls != 0 {
		t.Fatal("removing selected text copied a replacement on the same Enter")
	}
}
