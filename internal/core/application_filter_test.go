// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestOpenVPNChinesePinyinDiagnosis(t *testing.T) {
	app := Application{Name: "SoftEther VPN 命令行实用工具(vpncmd)", Path: "synthetic-softether"}
	prepared := PrepareSearch([]Application{app})
	// Heteronym heng for 行 previously let o/p come from SoftEther VPN,
	// e/n from heng, and v/p/n from vpncmd, across 35 intervening runes.
	if len(SearchApplications(prepared, "openvpn", 10)) != 0 {
		t.Fatal("unrelated polyphonic name matched long scattered query")
	}
	prepared = append(prepared, Application{Name: "OpenVPN GUI", Path: "synthetic-openvpn"})
	if got := SearchApplications(prepared, "openvpn", 10); len(got) != 1 || got[0].Name != "OpenVPN GUI" {
		t.Fatal("real contiguous OpenVPN match lost")
	}
	for _, query := range []string{"softether", "vpncmd", "mingling", "ml"} {
		if len(SearchApplications(prepared[:1], query, 10)) != 1 {
			t.Fatalf("legitimate Chinese/English/pinyin query lost: %s", query)
		}
	}
	if len(SearchApplications([]Application{{Name: "Other", Path: `C:\openvpn\ignored.exe`, Launch: LaunchIdentity{Target: `C:\openvpn\other.exe`, Arguments: "openvpn"}}}, "openvpn", 10)) != 0 {
		t.Fatal("path/target/arguments entered search surface")
	}
}

func TestBoundedLongSubsequencesKeepShortAcronymsAndAliases(t *testing.T) {
	for _, test := range []struct {
		name, query string
		want        bool
	}{
		{"Visual Studio Code", "vsc", true}, {"Visual Studio Code", "vscode", true},
		{"Open VPN", "openvpn", true}, {"Photoshop", "photoshp", true},
		{"o---p---e---n---v---p---n", "openvpn", false},
	} {
		if _, ok := scoreMatch(foldedRunes(test.name), foldedRunes(test.query)); ok != test.want {
			t.Fatalf("match %q/%q=%v", test.name, test.query, ok)
		}
	}
	apps := PrepareSearch([]Application{{Name: "微信", Path: "synthetic-wechat"}, {Name: "中文别名工具", Path: "synthetic-alias", Aliases: []string{"OpenVPN Fixture"}}})
	apps = WithUserAliases(apps, map[string][]string{apps[0].ItemID(): {"openvpn"}})
	if len(SearchApplications(apps, "wx", 10)) != 1 || len(SearchApplications(apps, "openvpn", 10)) != 2 {
		t.Fatal("initials/explicit alias semantics changed")
	}
}

func TestResolvedTargetFilterKeepsVariantsAndUnknownLinks(t *testing.T) {
	root := t.TempDir()
	metadata := map[string]LaunchIdentity{
		"Gui.lnk":       {Target: filepath.Join(root, "App.exe")},
		"OtherArgs.lnk": {Target: filepath.Join(root, "App.exe"), Arguments: "--alternate"},
		"Admin.lnk":     {Target: filepath.Join(root, "App.exe"), Independent: true, LinkFlags: 0x2000},
		"Virtual.lnk":   {}, "Manual.lnk": {Target: filepath.Join(root, "manual.html")},
	}
	for name, value := range metadata {
		value.AppUserModelIDKnown = true
		metadata[name] = value
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := DefaultScanOptions()
	calls := 0
	options.IncludeResolvedTarget = func(launch LaunchIdentity) bool { calls++; return filepath.Ext(launch.Target) != ".html" }
	resolver := func(path string) (LaunchIdentity, error) { return metadata[filepath.Base(path)], nil }
	before, _ := IndexApplicationsWithResolver([]string{root}, resolver)
	got, errs := IndexApplicationsWithOptions(context.Background(), []string{root}, resolver, options)
	if len(errs) != 0 || len(got) != 4 || calls != 3 {
		t.Fatalf("classification scope: apps=%d calls=%d errors=%v", len(got), calls, errs)
	}
	for _, app := range got {
		found := false
		for _, old := range before {
			if old.ItemID() == app.ItemID() {
				found = reflect.DeepEqual(old.Launch, app.Launch) && old.Path == app.Path
			}
		}
		if !found {
			t.Fatal("filter rewrote or merged launch semantics")
		}
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls = 0
	IndexApplicationsWithOptions(ctx, []string{root}, func(string) (LaunchIdentity, error) {
		cancel()
		return LaunchIdentity{Target: filepath.Join(root, "App.exe")}, nil
	}, options)
	if calls != 0 {
		t.Fatal("classification continued after resolver cancellation")
	}
}
