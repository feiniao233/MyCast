//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
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

func TestClipboardMenuDoesNotRetargetRemovedOrExpiredText(t *testing.T) {
	for _, remove := range []string{"delete", "expire"} {
		for _, event := range []string{"enter", "copy", "delete"} {
			for _, renderFirst := range []bool{false, true} {
				for _, pasteMode := range []bool{false, true} {
					t.Run(fmt.Sprintf("%s-%s-render-%v-paste-%v", remove, event, renderFirst, pasteMode), func(t *testing.T) {
						now := time.Unix(1000, 0)
						s := syntheticLauncher()
						s.clipboardMode = true
						s.history = core.NewHistoryWithClock(50, 64*1024, func() time.Time { return now })
						s.history.SetRetention(10 * time.Minute)
						s.history.Add("Dedicated A")
						now = now.Add(9 * time.Minute)
						s.history.Add("Dedicated B")
						copies, pastes := 0, 0
						s.copyText = func(string) error { copies++; return nil }
						s.pasteText = func(uintptr, string) error { pastes++; return nil }
						if pasteMode {
							s.target = 123
							s.tray = &mygo.Tray{}
						}
						tester := newPaletteTester(s, 1)
						tester.Key(0, ui.KeyDown)
						tester.Click("操作")
						if s.menuTargetKey != clipKey("Dedicated A") {
							t.Fatal("menu target was not bound on open")
						}
						label := "菜单项：仅复制"
						if event == "delete" {
							label = "菜单项：删除此条"
						}
						box, ok := tester.Find(label)
						if !ok {
							t.Fatal("missing original menu action")
						}
						if remove == "delete" {
							s.history.Delete("Dedicated A")
						} else {
							now = now.Add(2 * time.Minute)
						}
						if renderFirst {
							tester.Frame()
							if s.menu != menuClosed {
								t.Fatal("stale menu remained open after render")
							}
						}
						if event == "enter" {
							tester.Key(0, ui.KeyEnter)
						} else {
							tester.ClickAt(box.X+box.W/2, box.Y+box.H/2)
						}
						if copies != 0 || pastes != 0 || s.history.Len() != 1 || s.history.Entries()[0] != "Dedicated B" || s.menu != menuClosed {
							t.Fatal("stale menu copied/pasted/deleted a replacement")
						}
						// A second stale Enter must not act until the user deliberately reselects.
						tester.Key(0, ui.KeyEnter)
						if copies != 0 || pastes != 0 {
							t.Fatal("closed stale menu fell through to a replacement")
						}
						tester.Key(0, ui.KeyDown)
						tester.Click("操作")
						tester.Click("菜单项：仅复制")
						if copies != 1 {
							t.Fatal("explicit reselection did not recover actions")
						}
					})
				}
			}
		}
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

func TestClipboardFilteringDoesNotReportSelectionDeletion(t *testing.T) {
	s := syntheticLauncher()
	s.clipboardMode = true
	s.history.Add("Dedicated A")
	s.history.Add("Dedicated B")
	tester := newPaletteTester(s, 1)
	tester.Type("Dedicated A")
	if s.status != "" || s.clipNeedsSelection || s.clipResults()[s.selected] != "Dedicated A" {
		t.Fatal("ordinary query change treated as deletion")
	}
}
