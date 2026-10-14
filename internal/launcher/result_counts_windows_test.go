//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func syntheticCountLauncher() *launcher {
	s := syntheticLauncher()
	s.apps = nil
	for i := 0; i < 235; i++ {
		alias := "groupalpha"
		if i >= 150 {
			alias = "groupbeta"
		}
		s.apps = append(s.apps, core.Application{Name: fmt.Sprintf("Synthetic App %03d", i), Path: fmt.Sprintf(`C:\Synthetic\App%03d.exe`, i), Aliases: []string{alias}})
	}
	return s
}

func TestResultCountsRespectPresentationAndSearch(t *testing.T) {
	s := syntheticCountLauncher()
	assert := func(want string, shown, total int) {
		t.Helper()
		apps := s.applicationResults()
		if len(apps) != shown || s.appSearch.total != total || s.resultCountLabel(len(apps)) != want {
			t.Fatalf("label=%q shown/total=%d/%d; want %q %d/%d", s.resultCountLabel(len(apps)), len(apps), s.appSearch.total, want, shown, total)
		}
	}
	assert("应用 · 显示 100 / 可搜索 235", 100, 235)
	s.config.Favorites = []string{s.apps[234].ItemID()}
	assert("应用 · 显示 100 / 可搜索 235", 100, 235)
	if s.applicationResults()[0].ItemID() != s.apps[234].ItemID() {
		t.Fatal("favorites no longer rank ahead of other applications")
	}
	s.config.Hidden = []string{s.apps[234].ItemID()}
	assert("应用 · 显示 100 / 可搜索 234", 100, 234)
	s.config.Hidden[0] = s.apps[0].ItemID()
	s.query = "groupalpha"
	assert("搜索结果 · 显示 100 / 匹配 149", 100, 149)
	s.query = "Synthetic App 234"
	assert("搜索结果 · 显示 1 / 匹配 1", 1, 1)
	s.query = "absentneedle"
	assert("搜索结果 · 显示 0 / 匹配 0", 0, 0)
	s.query = "  "
	assert("应用 · 显示 100 / 可搜索 234", 100, 234)
	// Display-only snapshots remain searchable but never become launch evidence.
	s.pendingApps = []core.Application{{Name: "Pending synthetic", DisplayOnlyID: "pending-synthetic"}}
	assert("应用 · 显示 100 / 可搜索 235", 100, 235)
	s.switchMode(true)
	s.history.Add("Synthetic clipboard alpha")
	s.history.Add("Synthetic clipboard beta")
	if got := s.resultCountLabel(len(s.clipResults())); got != "文本历史 · 2 条" {
		t.Fatal(got)
	}
	s.query = "alpha"
	if got := s.resultCountLabel(len(s.clipResults())); got != "文本历史 · 匹配 1 条" {
		t.Fatal(got)
	}
	s.query = "absentneedle"
	if got := s.resultCountLabel(len(s.clipResults())); got != "文本历史 · 匹配 0 条" {
		t.Fatal(got)
	}
}

func TestResultCountHeaderAndScanSummary(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticCountLauncher()
			problems := make([]error, 128)
			for i := range problems {
				problems[i] = errors.New("synthetic metadata problem")
			}
			s.completeScan(s.apps, s.icons, problems, nil)
			if !strings.Contains(s.status, "已索引 235 个应用") || !strings.Contains(s.status, "已记录 128 个扫描问题") || strings.Contains(s.status, "位置无法读取") {
				t.Fatal(s.status)
			}
			tester := newPaletteTester(s, scale)
			if !tester.HasText("应用 · 显示 100 / 可搜索 235") {
				t.Fatal("header confuses the result cap with the index size")
			}
			savePreview(t, tester, fmt.Sprintf("result-counts-%03d", int(scale*100)))
			s.query = "groupalpha"
			s.resetResults()
			tester.Frame()
			if !tester.HasText("搜索结果 · 显示 100 / 匹配 150") {
				t.Fatal("query header does not show all matches")
			}
			s.query = "Synthetic App 234"
			s.resetResults()
			tester.Frame()
			if !tester.HasText("搜索结果 · 显示 1 / 匹配 1") || !tester.HasText("Synthetic App 234") {
				t.Fatal("entry beyond the initial result cap is not searchable")
			}
			s.switchMode(true)
			s.history.Add("Synthetic clipboard count fixture")
			tester.Frame()
			if !tester.HasText("文本历史 · 1 条") {
				t.Fatal("clipboard header borrowed the application index count")
			}
		})
	}
}
