// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"reflect"
	"strings"
	"testing"
)

func TestOfflinePinyinAndUserAliasesDoNotChangeIdentity(t *testing.T) {
	app := Application{Name: "微信", Path: `C:\Synthetic\WeChat.exe`}
	apps := PrepareSearch([]Application{app})
	for _, query := range []string{"微信", "weixin", "wx"} {
		got := SearchApplications(apps, query, 10)
		if len(got) != 1 || got[0].Identity() != app.Identity() {
			t.Fatalf("query %q missed or changed identity", query)
		}
	}
	apps = WithUserAliases(apps, map[string][]string{app.ItemID(): {"社交工具", "chat"}})
	for _, query := range []string{"chat", "shejiaogongju", "sjgj"} {
		if len(SearchApplications(apps, query, 10)) != 1 {
			t.Fatalf("alias %q missed", query)
		}
	}
	without := WithUserAliases(apps, nil)
	if len(SearchApplications(without, "chat", 10)) != 0 {
		t.Fatal("removed alias remained searchable")
	}
	if !reflect.DeepEqual(apps[0].SearchTerms, PrepareSearch(apps)[0].SearchTerms) {
		t.Fatal("repeated prepare accumulated derived terms")
	}
	if terms := PinyinTerms(strings.Repeat("重", 100)); len(terms) > 16 {
		t.Fatal("polyphonic expansion exceeded bound")
	}
	if terms := PinyinTerms(strings.Repeat("中", 257)); len(terms) != 0 {
		t.Fatal("oversized input generated terms")
	}
	if terms := PinyinTerms("微信 WeChat"); !containsString(terms, "weixin wechat") {
		t.Fatal("Latin suffix lost")
	}
}

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func TestPreferenceRankingKeepsMatchClassesAndNamesakes(t *testing.T) {
	apps := []Application{{Name: "Alpha", Path: `C:\Synthetic\Alpha.exe`}, {Name: "Beta", Path: `C:\Synthetic\Beta.exe`}, {Name: "Alpha", Path: `C:\Synthetic\Other\Alpha.exe`}}
	fav, rec := apps[1].ItemID(), apps[2].ItemID()
	got := SearchApplicationsRanked(apps, "", 10, []string{fav}, []string{rec})
	if len(got) != 3 || got[0].ItemID() != fav || got[1].ItemID() != rec {
		t.Fatal("empty query preference ranking or namesake preservation failed")
	}
	got = SearchApplicationsRanked(apps, "Alpha", 10, []string{fav}, []string{rec})
	if len(got) != 2 || got[0].ItemID() != rec {
		t.Fatal("exact namesake query lost recent priority")
	}
	apps[1].Name = "Alphabet"
	got = SearchApplicationsRanked(apps, "Alpha", 10, []string{fav}, nil)
	if got[0].ItemID() == fav {
		t.Fatal("favorite prefix outranked exact name")
	}
	before := append([]Application(nil), apps...)
	SearchApplicationsRanked(apps, "", 10, nil, nil)
	if !reflect.DeepEqual(before, apps) {
		t.Fatal("ranking mutated caller inventory")
	}
}
