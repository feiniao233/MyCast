// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"fmt"
	"math/rand"
	"reflect"
	"testing"
)

func searchCorpus(n int) []Application {
	names := []string{"显示设置", "腾讯会议", "OpenVPN GUI", "SoftEther VPN Client", "微信WeChat", "Σigma", "Kelvin", "QQ音乐", "重启工具", "ab", "cab", "a-b"}
	apps := make([]Application, n)
	for i := range apps {
		apps[i] = Application{Name: fmt.Sprintf("%s %04d", names[i%len(names)], i), Path: fmt.Sprintf("synthetic-%04d.exe", i), Aliases: []string{"工具"}, RelatedIDs: []string{fmt.Sprintf("legacy-%d", i)}}
	}
	return WithUserAliases(PrepareSearch(apps), map[string][]string{apps[0].ItemID(): {"屏幕设置", "monitor"}})
}

func matchApplications(matches []ApplicationMatch) []Application {
	apps := make([]Application, 0, len(matches))
	for _, match := range matches {
		apps = append(apps, match.Application)
	}
	return apps
}

func TestPreparedSearchDifferential(t *testing.T) {
	apps := searchCorpus(180)
	apps = append(apps, apps[0], Application{Name: "empty path"}, Application{Name: "Empty path"})
	catalog := NewSearchCatalog(apps)
	queries := []string{"", " ", "xian", "xians", "xianshi", "显示", "xs", "显示she", "qqyy", "qqyy ", "qqyy  ", "QQ音乐", "工具", "gj", "monitor", "屏幕", "OpenVPN", "ovpn", "SoftEther", "sev", "sigma", "kelvin", "ab", "not found"}
	rng := rand.New(rand.NewSource(3))
	for i := 0; i < 120; i++ {
		name := []rune(apps[rng.Intn(len(apps))].Name)
		start := rng.Intn(len(name))
		queries = append(queries, string(name[start:start+1+rng.Intn(len(name)-start)]))
	}
	for _, query := range queries {
		for _, limit := range []int{-1, 1, 17, 100} {
			for _, prefs := range []struct{ fav, rec []string }{{}, {[]string{apps[7].ItemID(), "legacy-13"}, []string{"legacy-1", apps[14].ItemID()}}} {
				want := SearchApplicationsRanked(apps, query, limit, prefs.fav, prefs.rec)
				got := matchApplications(catalog.Search(query, limit, prefs.fav, prefs.rec, nil))
				if !reflect.DeepEqual(got, want) {
					t.Fatalf("query %q limit %d ranking/metadata changed: got %v want %v", query, limit, appPaths(got), appPaths(want))
				}
			}
		}
	}
}

func TestPreparedSearchFullCatalogAndOwnership(t *testing.T) {
	// Existing density scoring permits a match -> miss -> match sequence.
	// This is a synthetic sentinel, not a change to the pinyin algorithm.
	apps := []Application{{Name: "xian-----------shi", Path: "sentinel", Aliases: []string{"QQ音乐"}}, {Name: "other", Path: "other"}}
	c := NewSearchCatalog(apps)
	apps[0].Name = "changed"
	apps[0].Aliases[0] = "changed"
	for i, query := range []string{"xian", "xians", "xianshi"} {
		got := c.Search(query, 0, nil, nil, nil)
		if (len(got) > 0) != (i != 1) {
			t.Fatalf("full catalog recovery %q: %v", query, got)
		}
	}
	got := c.Search("QQ音乐 ", 0, nil, nil, nil)
	if len(got) != 1 || got[0].Evidence.Field != "alias" || got[0].Evidence.Text != "QQ音乐" {
		t.Fatalf("evidence: %+v", got)
	}
	got[0].Application.Aliases[0] = "mutated return"
	if next := c.Search("QQ音乐", 0, nil, nil, nil); len(next) != 1 {
		t.Fatal("returned metadata mutated catalog")
	}
	if got := c.Search("", 0, nil, nil, []string{apps[1].ItemID()}); len(got) != 1 {
		t.Fatal("hidden identity not filtered")
	}
}

func TestPreparedSearchAliasRemovalAndExplanation(t *testing.T) {
	apps := PrepareSearch([]Application{{Name: "显示", Path: "synthetic.exe"}})
	id := apps[0].ItemID()
	with := NewSearchCatalog(WithUserAliases(apps, map[string][]string{id: {"monitor"}}))
	if got := with.Search("monitor", 0, nil, nil, nil); len(got) != 1 || got[0].Evidence.Field != "user-alias-or-pinyin" {
		t.Fatal(got)
	}
	without := NewSearchCatalog(WithUserAliases(apps, nil))
	if len(without.Search("monitor", 0, nil, nil, nil)) != 0 {
		t.Fatal("removed alias remained searchable")
	}
	if got := without.Search("xianshi", 0, nil, nil, nil); len(got) != 1 || got[0].Evidence.Field != "pinyin" || got[0].Evidence.Class != 0 {
		t.Fatal(got)
	}
}

var benchmarkMatches []ApplicationMatch
var benchmarkApps []Application
var benchmarkCatalog *SearchCatalog

func BenchmarkSearchCatalog(b *testing.B) {
	apps := searchCorpus(5000)
	queries := []string{"xianshi", "qqyy ", "ovpn", "SoftEther", "工具", "not-found"}
	b.Run("Legacy", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkApps = SearchApplicationsRanked(apps, queries[i%len(queries)], 100, nil, nil)
		}
	})
	c := NewSearchCatalog(apps)
	b.Run("Prepared", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkMatches = c.Search(queries[i%len(queries)], 100, nil, nil, nil)
		}
	})
	b.Run("Prepare", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			benchmarkCatalog = NewSearchCatalog(apps)
		}
	})
}
