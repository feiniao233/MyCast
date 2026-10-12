//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"testing"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestUtilityExplicitCopyLinksAndDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.persistPreferences = func(value settings.Settings) (settings.Settings, error) { return value.Validated() }
			var copied, opened []string
			s.copyText = func(value string) error { copied = append(copied, value); return nil }
			s.openURL = func(value string) error { opened = append(opened, value); return nil }
			tester := newPaletteTester(s, scale)
			tester.Click("菜单")
			tester.Click("菜单项：小工具")
			if !s.utilityOpen || !tester.Focused("计算表达式") {
				t.Fatal("utility field not focused")
			}
			tester.Type("1 + 2 * (3 - 1)")
			tester.Key(0, ui.KeyEnter)
			if len(copied) != 0 || !s.utilityOpen {
				t.Fatal("calculator Enter copied or launched covered state")
			}
			tester.Click("复制计算结果")
			if len(copied) != 1 || copied[0] != "5" {
				t.Fatal("explicit calculation copy failed")
			}
			tester.Click("编辑快捷链接")
			if !s.linkEditor || !tester.Focused("快捷链接内容") {
				t.Fatal("link editor focus missing")
			}
			s.linksText = "Synthetic docs | https://example.com/docs\nSynthetic tools | https://example.com/tools"
			tester.Frame()
			savePreview(t, tester, fmt.Sprintf("quicklinks-editor-%03d", int(scale*100)))
			tester.Click("保存快捷链接")
			if s.linkEditor || len(s.config.QuickLinks) != 2 || len(opened) != 0 || !tester.Focused("计算表达式") {
				t.Fatal("save opened a URL or failed focus/persistence")
			}
			for _, label := range []string{"复制计算结果", "编辑快捷链接", "关闭小工具", "打开链接：Synthetic docs"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < 12 || box.Y+box.H > palette.Height-12 {
					t.Fatalf("utility control clipped: %s %+v", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("utilities-%03d", int(scale*100)))
			tester.Click("打开链接：Synthetic docs")
			if len(opened) != 1 || opened[0] != "https://example.com/docs" {
				t.Fatal("explicit link selection did not reach adapter")
			}
			tester.Click("编辑快捷链接")
			s.linksText = "Invalid | javascript:alert(1)"
			tester.Frame()
			tester.Click("保存快捷链接")
			if !s.linkEditor || len(s.config.QuickLinks) != 2 || len(opened) != 1 {
				t.Fatal("invalid draft was saved or opened")
			}
			tester.Key(0, ui.KeyEscape)
			if s.linkEditor || !tester.Focused("计算表达式") {
				t.Fatal("editor Escape did not return to calculator")
			}
			tester.Compose("合成", 2)
			tester.Key(0, ui.KeyEscape)
			if !s.utilityOpen {
				t.Fatal("IME cancel closed utility")
			}
			tester.Click("关闭小工具")
			if s.utilityOpen || s.calcText != "" || !tester.Focused("搜索") {
				t.Fatal("close retained expression or lost search focus")
			}
		})
	}
}
