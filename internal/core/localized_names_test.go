// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestChineseDisplayNamePreservesLaunchIdentityAndSearch(t *testing.T) {
	for _, launch := range []LaunchIdentity{{Target: filepath.Join(t.TempDir(), "App.exe"), Arguments: "--safe", WorkingDirectory: t.TempDir()}, {Independent: true, LinkFlags: 0x2000, ShowCommand: 3}} {
		original := Application{Name: "English Fixture", Path: filepath.Join(t.TempDir(), "English Fixture.lnk"), Launch: launch, Aliases: []string{"English duplicate"}}
		localized := PreferChineseDisplayName(original, "  中文工具  ")
		if localized.Name != "中文工具" || localized.ItemID() != original.ItemID() || localized.Path != original.Path || !reflect.DeepEqual(localized.Launch, original.Launch) || len(original.Aliases) != 1 {
			t.Fatal("localization changed launch data or input aliases")
		}
		apps := WithUserAliases(PrepareSearch([]Application{localized}), map[string][]string{original.ItemID(): {"我的别名"}})
		for _, query := range []string{"中文工具", "zhongwengongju", "zwgj", "English Fixture", "English duplicate", "我的别名"} {
			if len(SearchApplications(apps, query, 10)) != 1 {
				t.Fatalf("localized search lost %q", query)
			}
		}
		other := Application{Name: "Another", Path: filepath.Join(t.TempDir(), "Other.exe")}
		ranked := SearchApplicationsRanked(append(apps, other), "", 10, []string{original.ItemID()}, []string{original.ItemID()})
		if ranked[0].ItemID() != original.ItemID() {
			t.Fatal("localization detached favorites/recents")
		}
		if got := PreferChineseDisplayName(localized, "English resource"); got.Name != localized.Name {
			t.Fatal("English resource downgraded Chinese name")
		}
	}
	for _, name := range []string{"Brand English", "坏\x00名称", "坏\u202e名称", "坏\ufffd名称", strings.Repeat("中", 257)} {
		if ChineseDisplayName(name) {
			t.Fatalf("invalid localized candidate accepted: %q", name)
		}
	}
}

func TestMergedNameSourcePathsKeepIndependentVariants(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target.exe")
	for _, name := range []string{"Original.lnk", "Duplicate.lnk", "Argument.lnk", "Privilege.lnk"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	resolver := func(path string) (LaunchIdentity, error) {
		identity := LaunchIdentity{Target: target, AppUserModelIDKnown: true}
		switch filepath.Base(path) {
		case "Argument.lnk":
			identity.Arguments = "--alternate"
		case "Privilege.lnk":
			identity.Independent = true
			identity.LinkFlags = 0x2000
		}
		return identity, nil
	}
	apps, errs := IndexApplicationsWithOptions(context.Background(), []string{root}, resolver, DefaultScanOptions())
	if len(errs) != 0 || len(apps) != 3 {
		t.Fatalf("unexpected variants: %d %v", len(apps), errs)
	}
	paths := 0
	ids := map[string]bool{}
	for _, app := range apps {
		paths += len(app.NamePaths)
		ids[app.ItemID()] = true
	}
	if paths != 4 || len(ids) != 3 {
		t.Fatal("lost duplicate name sources or merged launch variants")
	}
}
