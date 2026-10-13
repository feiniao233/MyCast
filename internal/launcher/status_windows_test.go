//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/egoist/mygo/ui"
)

func syntheticLongStatus() string {
	return `Synthetic launch fixture missing: C:\Synthetic\` + strings.Repeat("专用测试目录-long-folder-", 18) + `\MissingFixture.lnk; 请检查合成快捷方式路径。`
}

func TestStatusDetailsPointerKeyboardScrollAndPrivacy(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("dark-%v-dpi-%03d", dark, int(scale*100)), func(t *testing.T) {
				s := syntheticLauncher()
				if dark {
					s.config.Theme = "dark"
				} else {
					s.config.Theme = "light"
				}
				s.query = "Synthetic"
				launched, copied, pasted := 0, 0, 0
				s.launchApplication = func(string) error { launched++; return nil }
				s.copyText = func(string) error { copied++; return nil }
				s.pasteText = func(uintptr, string) error { pasted++; return nil }
				s.status = syntheticLongStatus() + "\n" + strings.Repeat("Dedicated synthetic detail 中文 / English line\n", 40) + "DEDICATED STATUS END"
				original := s.status
				exactStatusFocus := false
				exactSearchFocus := false
				tester := ui.NewTester(func(c *ui.Context) {
					s.view(c)
					exactStatusFocus = s.statusControl.Focused(c)
					exactSearchFocus = s.input.Focused(c)
				}, int(palette.Width), int(palette.Height))
				tester.SetScale(scale)
				button, ok := tester.Find("状态详情")
				primary, _ := tester.Find("打开应用")
				if !ok || button.W > palette.StatusWidth || button.X+button.W > primary.X-palette.ControlInset {
					t.Fatal("status trigger exceeds its footer slot")
				}
				savePreview(t, tester, fmt.Sprintf("status-footer-dark-%v-%03d", dark, int(scale*100)))
				if err := tester.Click("状态详情"); err != nil {
					t.Fatal(err)
				}
				if !s.statusDetailsOpen || !tester.HasText(original) || !tester.Focused("状态详情面板") {
					t.Fatal("full accessible status/focus missing")
				}
				if tester.HasText("复制全文") {
					t.Fatal("status detail introduced a clipboard action")
				}
				savePreview(t, tester, fmt.Sprintf("status-detail-dark-%v-%03d", dark, int(scale*100)))
				s.status = "Dedicated newer background status"
				tester.Frame()
				if !tester.HasText(original) || s.statusDetail != original {
					t.Fatal("open detail unexpectedly retargeted")
				}
				tester.Key(0, ui.KeyEnter)
				tester.ClickAt(12, 120)
				if !s.statusDetailsOpen || launched != 0 || copied != 0 || pasted != 0 {
					t.Fatal("detail actions leaked to covered app/clipboard")
				}
				tester.Scroll(350, 200, 0, 9000)
				if s.statusDetailScroll.MaxY <= 0 || s.statusDetailScroll.Y != s.statusDetailScroll.MaxY {
					t.Fatal("full detail end not reachable")
				}
				tester.Click("关闭状态详情")
				if s.statusDetailsOpen || s.statusDetail != "" || !tester.Focused("搜索") || s.query != "Synthetic" {
					t.Fatal("close retained details or lost search state")
				}
				// Reach the native focusable status button by keyboard Tab. Its
				// Enter must open details, never run the palette's primary action.
				for n := 0; n < 50 && !exactStatusFocus; n++ {
					tester.Key(0, ui.KeyTab)
				}
				if !exactStatusFocus {
					t.Fatal("status details not keyboard reachable")
				}
				tester.Key(0, ui.KeyEnter)
				if !s.statusDetailsOpen || launched != 0 {
					t.Fatalf("status keyboard Enter: details=%v launched=%d", s.statusDetailsOpen, launched)
				}
				tester.Key(0, ui.KeyEscape)
				if s.statusDetailsOpen || !tester.Focused("搜索") {
					t.Fatal("detail Escape/focus failed")
				}
				if !exactSearchFocus {
					t.Fatal("detail close restored a search ancestor, not the input")
				}
				tester.Compose("合成候选", 2)
				tester.Key(0, ui.KeyEnter)
				tester.Key(0, ui.KeyEscape)
				if s.statusDetailsOpen || launched != 0 {
					t.Fatalf("IME candidate action: detail=%v launched=%d search=%v query=%q", s.statusDetailsOpen, launched, exactSearchFocus, s.query)
				}
				tester.Compose("", 0)
				s.openStatusDetails()
				tester.Frame()
				s.hide()
				if s.statusDetailsOpen || s.statusDetail != "" {
					t.Fatal("hide retained detail snapshot")
				}
			})
		}
	}
}

func TestPaletteAltF4UsesNormalCloseRequest(t *testing.T) {
	s := syntheticLauncher()
	calls := 0
	s.requestClose = func() { calls++; s.hide() }
	tester := newPaletteTester(s, 1)
	tester.Key(ui.Alt, ui.KeyF4)
	if calls != 1 {
		t.Fatal("Alt+F4 did not reach normal close policy")
	}
	s.status = "Dedicated close fixture"
	s.openStatusDetails()
	tester.Frame()
	tester.Key(ui.Alt, ui.KeyF4)
	if calls != 2 || s.statusDetailsOpen || s.statusDetail != "" {
		t.Fatal("Alt+F4 close did not release modal state")
	}
}

func TestCandidateEnterCannotBecomeLauncherSubmit(t *testing.T) {
	s := syntheticLauncher()
	launched := 0
	s.launchApplication = func(string) error { launched++; return nil }
	tester := newPaletteTester(s, 1)
	tester.Compose("Synthetic", 2)
	tester.Key(0, ui.KeyEnter)
	if launched != 0 || s.query != "Synthetic" {
		t.Fatal("candidate confirmation launched or lost composed query")
	}
	tester.Key(0, ui.KeyEnter)
	if launched != 1 {
		t.Fatal("ordinary Enter after candidate confirmation remained blocked")
	}
}

func TestCurrentLongStatusDoesNotCoverActionPill(t *testing.T) {
	for _, dark := range []bool{true, false} {
		for _, scale := range []float32{1, 1.5} {
			t.Run(fmt.Sprintf("dark-%v-dpi-%03d", dark, int(scale*100)), func(t *testing.T) {
				s := syntheticLauncher()
				if dark {
					s.config.Theme = "dark"
				} else {
					s.config.Theme = "light"
				}
				s.launchApplication = func(string) error { return errors.New(syntheticLongStatus()) }
				s.activate(s.apps, nil, false)
				tester := newPaletteTester(s, scale)
				status, ok := tester.Find(s.status)
				primary, primaryOK := tester.Find("打开应用")
				if !ok || !primaryOK || status.W > palette.StatusWidth+.01 || status.X+status.W > primary.X-palette.ControlInset {
					t.Fatalf("status overlaps action: %+v primary %+v", status, primary)
				}
				t.Logf("latest SingleLine status box: %+v; primary %+v", status, primary)
				savePreview(t, tester, fmt.Sprintf("status-baseline-dark-%v-%03d", dark, int(scale*100)))
			})
		}
	}
}
