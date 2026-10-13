//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/core"
	"testing"
)

func TestApplicationSearchMemoDependencies(t *testing.T) {
	s := &launcher{apps: core.PrepareSearch([]core.Application{{Name: "QQ音乐", Path: "synthetic-qq.exe"}, {Name: "Other", Path: "synthetic-other.exe"}})}
	s.query = "qqyy"
	if len(s.applicationResults()) != 1 {
		t.Fatal("pinyin lookup")
	}
	first := s.appSearch.catalog
	for _, query := range []string{"qqyy ", "qqyy  ", "", "QQ音乐"} {
		s.query = query
		s.applicationResults()
		if s.appSearch.query != query || s.appSearch.catalog != first {
			t.Fatal("exact query key or catalog reuse")
		}
	}
	s.query = ""
	s.config.Favorites = []string{s.apps[1].ItemID()}
	if s.applicationResults()[0].Name != "Other" {
		t.Fatal("favorite invalidation")
	}
	s.config.Favorites[0] = s.apps[0].ItemID()
	if s.applicationResults()[0].Name != "QQ音乐" {
		t.Fatal("in-place favorite invalidation")
	}
	s.config.Favorites = nil
	s.config.Recents = []string{s.apps[1].ItemID()}
	if s.applicationResults()[0].Name != "Other" {
		t.Fatal("recent invalidation")
	}
	s.config.Hidden = []string{s.apps[1].ItemID()}
	if len(s.applicationResults()) != 1 {
		t.Fatal("hidden invalidation")
	}
	s.config.Hidden = nil
	s.config.Recents = nil
	s.apps = core.WithUserAliases(s.apps, map[string][]string{s.apps[0].ItemID(): {"monitor"}})
	s.invalidateApplicationSearch()
	s.query = "monitor"
	if len(s.applicationResults()) != 1 {
		t.Fatal("alias revision")
	}
	s.apps = core.WithUserAliases(s.apps, nil)
	s.invalidateApplicationSearch()
	if len(s.applicationResults()) != 0 {
		t.Fatal("removed alias revision")
	}
	s.apps = []core.Application{{Name: "monitor", Path: "synthetic-new.exe"}}
	if len(s.applicationResults()) != 1 {
		t.Fatal("catalog replacement")
	}
}
