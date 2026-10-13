// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestProvedPreferenceBindingSurvivesRestartWithoutShortcut(t *testing.T) {
	legacy := (core.Application{Path: `C:\Synthetic\Original.lnk`}).ItemID()
	shell := core.Application{Name: "Fixture", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App"}
	proved := shell
	proved.RelatedIDs = []string{legacy}
	value := Defaults()
	value.Favorites = []string{legacy}
	value.Recents = []string{legacy}
	value.Hidden = []string{legacy}
	value.Aliases = map[string][]string{legacy: {"bound-alias"}}
	value, changed := value.BindProvedApplicationIdentities([]core.Application{proved})
	if !changed || value.IdentityLinks[legacy] != shell.ItemID() {
		t.Fatal("proof was not bound")
	}
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	restored := loaded.ApplyIdentityLinks([]core.Application{shell})
	if !restored[0].Preferred(loaded.Favorites) || !restored[0].Preferred(loaded.Recents) || !restored[0].Preferred(loaded.Hidden) {
		t.Fatal("restart lost bounded ordinary preferences")
	}
	restored = core.WithUserAliases(restored, loaded.Aliases)
	if len(core.SearchApplications(restored, "bound-alias", 10)) != 1 {
		t.Fatal("restart lost alias")
	}
	clone := loaded.Clone()
	clone.IdentityLinks[legacy] = "bad"
	if loaded.IdentityLinks[legacy] != shell.ItemID() {
		t.Fatal("binding clone shared map")
	}
	unrelated := Defaults()
	unrelated, changed = unrelated.BindProvedApplicationIdentities([]core.Application{proved})
	if changed || len(unrelated.IdentityLinks) != 0 {
		t.Fatal("unused catalog identities persisted")
	}
}

func TestIdentityBindingBoundsChainsAndRetargetRefusal(t *testing.T) {
	old := (core.Application{Path: "Old.lnk"}).ItemID()
	first := (core.Application{Path: "First.exe"}).ItemID()
	second := (core.Application{Path: "Second.exe"}).ItemID()
	value := Defaults()
	value.Favorites = []string{old}
	value.IdentityLinks = map[string]string{old: first}
	repurposed := core.Application{Path: "Second.exe", RelatedIDs: []string{old}}
	candidate, changed := value.BindProvedApplicationIdentities([]core.Application{repurposed})
	if changed || candidate.IdentityLinks[old] != first {
		t.Fatal("existing preference silently retargeted")
	}
	if value.ApplyIdentityLinks([]core.Application{repurposed})[0].HasPreferenceID(old) {
		t.Fatal("old binding still applied to different canonical app")
	}
	value.IdentityLinks[first] = second
	if _, err := value.Validated(); err == nil {
		t.Fatal("chained binding accepted")
	}
	bounded := Defaults()
	bounded.IdentityLinks = map[string]string{}
	for i := 0; i < 651; i++ {
		id := (core.Application{Path: fmt.Sprintf("Synthetic-%d.lnk", i)}).ItemID()
		bounded.IdentityLinks[id] = first
	}
	if _, err := bounded.Validated(); err == nil {
		t.Fatal("oversized identity bindings accepted")
	}
}

func TestDesktopBindingsKeepPreferencesWhenEitherProviderSurvives(t *testing.T) {
	shell := core.Application{Name: "Fixture", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop", ShellDesktop: true, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, AppUserModelID: "org.synthetic.desktop"}}
	file := core.Application{Name: "Fixture Link", Path: `C:\Synthetic\Fixture.lnk`, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, WorkingDirectory: `C:\Synthetic`}}
	old := file.ItemID()
	value := Defaults()
	value.Favorites = []string{old}
	merged := core.MergeProviderApplications([]core.Application{file, shell})
	value, changed := value.BindProvedApplicationIdentities(merged)
	if !changed || value.IdentityLinks[old] != shell.ItemID() {
		t.Fatal("desktop proof did not bind old preference")
	}
	value.Favorites = []string{shell.ItemID()}
	value.Recents = value.Favorites
	value.Hidden = value.Favorites
	value.Aliases = map[string][]string{shell.ItemID(): {"synthetic-alias"}}
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	for _, remaining := range []core.Application{file, shell} {
		apps := loaded.ApplyIdentityLinks([]core.Application{remaining})
		if !apps[0].Preferred(loaded.Favorites) || !apps[0].Preferred(loaded.Recents) || !apps[0].Preferred(loaded.Hidden) || len(core.SearchApplications(core.WithUserAliases(apps, loaded.Aliases), "synthetic-alias", 10)) != 1 {
			t.Fatal("provider disappearance lost persisted preferences")
		}
		if _, changed := loaded.BindProvedApplicationIdentities(apps); changed {
			t.Fatal("applying persisted preferences invented reversed proof")
		}
	}
}
