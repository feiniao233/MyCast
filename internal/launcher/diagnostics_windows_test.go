//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"strings"
	"testing"
)

func TestActualMatchingFieldDiagnosticsDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.apps = core.PrepareSearch([]core.Application{{Name: "Synthetic 显示", Path: `C:\Synthetic\Display.lnk`, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Display.exe`, AppUserModelIDKnown: true}, Aliases: []string{"screen"}}})
			s.config = settings.Defaults()
			id := s.apps[0].ItemID()
			s.apps = core.WithUserAliases(s.apps, map[string][]string{id: {"屏幕", "monitor"}})
			s.recordSourceDiagnostics([]string{`C:\Synthetic`}, scanFiles, true, false, 7)
			for _, row := range []struct{ query, field string }{{"Synthetic 显示", "显示名"}, {"screen", "索引别名"}, {"syntheticxianshi", "名称/索引别名的拼音"}, {"monitor", "用户别名"}, {"pingmu", "用户别名的拼音"}} {
				s.query = row.query
				detail := s.applicationDiagnostics(s.apps[0])
				if !strings.Contains(detail, "匹配字段："+row.field) || !strings.Contains(detail, "批次部分完成；未观察项的可用性未知") || !strings.Contains(detail, "原始文件 ShellExecute") {
					t.Fatal("not actual matching evidence", row, detail)
				}
			}
			s.query = "pingmu"
			tester := newPaletteTester(s, scale)
			before := s.config.Clone()
			calls := 0
			s.launchApplication = func(string) error { calls++; return nil }
			if err := tester.RightClick("结果：" + s.apps[0].Name); err != nil {
				t.Fatal(err)
			}
			if err := tester.Click("菜单项：查看来源"); err != nil {
				t.Fatal(err)
			}
			if !s.statusDetailsOpen || !tester.HasText("完整状态") || !strings.Contains(s.statusDetail, "匹配字段：用户别名的拼音") {
				t.Fatalf("diagnostic layer: open=%t texts=%q detail=%q", s.statusDetailsOpen, tester.Texts(), s.statusDetail)
			}
			savePreview(t, tester, fmt.Sprintf("search-diagnostics-%03d", int(scale*100)))
			tester.Key(0, ui.KeyEscape)
			if s.settingsOpen || !tester.Focused("搜索") || s.query != "pingmu" || calls != 0 || len(s.config.Favorites) != len(before.Favorites) {
				t.Fatal("read-only diagnostic dismissal altered search/effects")
			}
			s.openSettings()
			tester.Frame()
			tester.Click("启动快照选项")
			tester.Key(0, ui.KeyEscape)
			if s.snapshotOpen || !s.settingsOpen {
				t.Fatal("child escape closed parent")
			}
			tester.Key(0, ui.KeyEscape)
			if s.settingsOpen || !tester.Focused("搜索") {
				t.Fatal("parent escape did not restore search")
			}
			tester.Compose("合成", 2)
			tester.Key(0, ui.KeyEscape)
			if s.query == "" || !tester.Focused("搜索") || calls != 0 {
				t.Fatal("IME cancel leaked through launcher")
			}
		})
	}
}

func TestIdentityAndBoundedSourceDiagnostics(t *testing.T) {
	cases := []struct {
		app  core.Application
		want string
	}{
		{core.Application{DisplayOnlyID: "display"}, "未验证显示身份"},
		{core.Application{Path: "synthetic.lnk"}, "AppID 属性读取未知"},
		{core.Application{Path: "synthetic.lnk", Launch: core.LaunchIdentity{AppUserModelIDKnown: true, Independent: true}}, "独立启动语义"},
		{core.Application{Path: "synthetic.lnk", Launch: core.LaunchIdentity{AppUserModelIDKnown: true, ShellTarget: "fixture", DesktopBridge: true}}, "已验证本地 EXE"},
		{core.Application{Path: "synthetic.exe", Launch: core.LaunchIdentity{Target: "fixture", AppUserModelID: "synthetic-app-id"}}, "显式 AppID"},
	}
	for _, c := range cases {
		if !strings.Contains(applicationIdentityEvidence(c.app), c.want) {
			t.Fatal(c.want)
		}
	}
	s := &launcher{config: settings.Defaults()}
	for i := 0; i < 80; i++ {
		s.recordSourceDiagnostics([]string{fmt.Sprintf("synthetic-%d", i)}, scanFiles|scanShell, false, false, uint64(i))
	}
	if len(s.sourceDiagnostics) != 33 {
		t.Fatal("unbounded source diagnostics")
	}
	s.recordSourceDiagnostics([]string{"synthetic-0"}, scanFiles, true, false, 99)
	if s.sourceDiagnostics[0].generation != 99 || !s.sourceDiagnostics[0].partial {
		t.Fatal("per-source replacement")
	}
}
