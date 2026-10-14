// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"crypto/sha256"
	"fmt"
	"maps"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestProvedPreferenceBindingSurvivesRestartWithoutShortcut(t *testing.T) {
	legacy := historicalPathID(`C:\Synthetic\Original.lnk`)
	shell := core.Application{Name: "Fixture", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App", ShellPackaged: true}
	proved := shell
	proved.RelatedIDs = []string{legacy}
	proved.ProvedIDs = []string{legacy}
	value := preferencesForIdentity(legacy)
	value, changed := value.BindProvedApplicationIdentities([]core.Application{proved})
	if !changed || value.IdentityLinks[legacy] != shell.ItemID() {
		t.Fatal("proof was not bound")
	}
	loaded := restartIdentitySettings(t, value)
	assertIdentityPreferences(t, loaded, shell)
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
	old := historicalPathID("Old.lnk")
	first := (core.Application{Path: "First.exe"}).ItemID()
	second := (core.Application{Path: "Second.exe"}).ItemID()
	value := Defaults()
	value.Favorites = []string{old}
	value.IdentityLinks = map[string]string{old: first}
	repurposed := core.Application{Path: "Second.exe", RelatedIDs: []string{old}, ProvedIDs: []string{old}}
	candidate, changed := value.BindProvedApplicationIdentities([]core.Application{repurposed})
	if changed || candidate.IdentityLinks[old] != first {
		t.Fatal("existing preference silently retargeted")
	}
	if value.ApplyIdentityLinks([]core.Application{repurposed})[0].HasPreferenceID(old) {
		t.Fatal("old binding still applied to different canonical app")
	}
	for name, links := range map[string]map[string]string{
		"chain": {old: first, first: second},
		"cycle": {old: first, first: old},
		"self":  {old: old},
	} {
		t.Run(name, func(t *testing.T) {
			value.IdentityLinks = links
			if _, err := value.Validated(); err == nil {
				t.Fatal("non-one-hop binding accepted")
			}
		})
	}
	bounded := Defaults()
	bounded.IdentityLinks = map[string]string{}
	for i := 0; i < 651; i++ {
		id := historicalPathID(fmt.Sprintf("Synthetic-%d.lnk", i))
		bounded.IdentityLinks[id] = first
	}
	if _, err := bounded.Validated(); err == nil {
		t.Fatal("oversized identity bindings accepted")
	}
}

func TestShortcutProofsPreserveAllPreferencesAfterDuplicateRemoval(t *testing.T) {
	original := shortcutIdentityFixture("Fixture", "first")
	original.Launch.AppUserModelID = "org.synthetic.desktop"
	duplicate := original
	duplicate.Path = `C:\Synthetic\second\Fixture.lnk`
	duplicate.SourcePriority = 1
	duplicate.Launch.Arguments = "--different-arguments"
	duplicate.Launch.WorkingDirectory = `C:\Synthetic\Elsewhere`
	for name, old := range map[string]string{
		"path":           historicalPathID(original.Path),
		"target":         historicalTupleID(original, false),
		"target-and-app": historicalTupleID(original, true),
	} {
		t.Run(name, func(t *testing.T) {
			merged := core.MergeProviderApplications([]core.Application{duplicate, original})
			if len(merged) != 1 || merged[0].Path != original.Path || !slices.Contains(merged[0].ProvedIDs, old) {
				t.Fatal("shortcut scan did not retain original selection and historical proof")
			}
			value := preferencesForIdentity(old)
			bound, changed := value.BindProvedApplicationIdentities(merged)
			if !changed || bound.IdentityLinks[old] != merged[0].ItemID() {
				t.Fatal("ordinary .lnk preference proof was not saved")
			}
			loaded := restartIdentitySettings(t, bound)
			remaining := core.MergeProviderApplications([]core.Application{duplicate})
			if len(remaining) != 1 || remaining[0].ItemID() != merged[0].ItemID() {
				t.Fatal("removing selected duplicate changed canonical identity")
			}
			assertIdentityPreferences(t, loaded, remaining[0])
			if _, changed := loaded.BindProvedApplicationIdentities(loaded.ApplyIdentityLinks(remaining)); changed {
				t.Fatal("saved preferences invented fresh proof after duplicate removal")
			}
		})
	}
}

func TestStoredDesktopBindingsMigrateToShortcut(t *testing.T) {
	original := shortcutIdentityFixture("Fixture", "first")
	duplicate := original
	duplicate.Path = `C:\Synthetic\second\Fixture.lnk`
	duplicate.SourcePriority = 1
	duplicate.Launch.Arguments = "--different"
	shell := core.Application{Name: "Fixture", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop", ShellDesktop: true, Launch: core.LaunchIdentity{Target: original.Launch.Target}}
	oldTuple, oldPath := historicalTupleID(original, false), historicalPathID(original.Path)
	for name, selected := range map[string]string{"shell": shell.ItemID(), "tuple": oldTuple, "path": oldPath} {
		t.Run(name, func(t *testing.T) {
			value := preferencesForIdentity(selected)
			value.IdentityLinks = map[string]string{oldTuple: shell.ItemID(), oldPath: shell.ItemID()}
			merged := core.MergeProviderApplications([]core.Application{shell, duplicate, original})
			if len(merged) != 1 || merged[0].ShellID != "" || !slices.Contains(merged[0].LegacyTargetIDs, oldTuple) {
				t.Fatal("desktop Shell row survived or old tuple was not proved")
			}
			bound, changed := value.BindProvedApplicationIdentities(merged)
			if !changed || bound.IdentityLinks[selected] != merged[0].ItemID() {
				t.Fatal("saved old desktop component did not migrate to the shortcut")
			}
			if _, err := bound.Validated(); err != nil {
				t.Fatal("migration introduced a chain or invalid binding", err)
			}
			loaded := restartIdentitySettings(t, bound)
			remaining := core.MergeProviderApplications([]core.Application{duplicate})
			assertIdentityPreferences(t, loaded, remaining[0])
		})
	}
}

func TestAmbiguousDesktopBindingsDoNotChooseShortcut(t *testing.T) {
	first := shortcutIdentityFixture("First", "first")
	second := shortcutIdentityFixture("Second", "second")
	shell := (core.Application{ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop"}).ItemID()
	for _, conflictingTuples := range []bool{false, true} {
		name := "one-tuple-with-two-owners"
		if conflictingTuples {
			name = "two-tuples-with-conflicting-owners"
		}
		t.Run(name, func(t *testing.T) {
			other := second
			if conflictingTuples {
				other.Launch.Arguments = "--different"
			}
			value := preferencesForIdentity(shell)
			value.IdentityLinks = map[string]string{historicalTupleID(first, false): shell, historicalTupleID(other, false): shell}
			merged := core.MergeProviderApplications([]core.Application{first, other})
			if len(merged) != 2 {
				t.Fatal("different original shortcut names were collapsed")
			}
			bound, changed := value.BindProvedApplicationIdentities(merged)
			if changed || !maps.Equal(bound.IdentityLinks, value.IdentityLinks) {
				t.Fatal("ambiguous historical desktop association was migrated")
			}
			for _, app := range bound.ApplyIdentityLinks(merged) {
				if app.Preferred(bound.Favorites) || app.HasPreferenceID(shell) {
					t.Fatal("ambiguous old Shell preference guessed a new owner")
				}
			}
		})
	}
}

func TestRepurposedShortcutPathDoesNotMigrateOldDesktopBinding(t *testing.T) {
	original := shortcutIdentityFixture("Fixture", "first")
	oldPath, oldTuple := historicalPathID(original.Path), historicalTupleID(original, false)
	shell := (core.Application{ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop"}).ItemID()
	value := preferencesForIdentity(oldPath)
	value.IdentityLinks = map[string]string{oldPath: shell, oldTuple: shell}
	repurposed := original
	repurposed.Launch.Target = `C:\Unrelated\Other.exe`
	apps := core.MergeProviderApplications([]core.Application{repurposed})
	bound, _ := value.BindProvedApplicationIdentities(apps)
	if bound.IdentityLinks[oldPath] != shell {
		t.Fatal("reused shortcut source path redirected historical binding")
	}
	for _, app := range bound.ApplyIdentityLinks(apps) {
		if app.Preferred(bound.Favorites) || app.HasPreferenceID(shell) {
			t.Fatal("reused shortcut inherited old desktop preferences")
		}
	}
}

func TestRelatedPreferencesCannotAuthorizeNewProof(t *testing.T) {
	app := shortcutIdentityFixture("Fixture", "first")
	old := historicalTupleID(app, false)
	shell := (core.Application{ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop"}).ItemID()
	app.RelatedIDs = []string{old, shell}
	app.LegacyTargetIDs = []string{old} // Must also occur in current ProvedIDs.
	value := preferencesForIdentity(old)
	bound, changed := value.BindProvedApplicationIdentities([]core.Application{app})
	if changed || len(bound.IdentityLinks) != 0 {
		t.Fatal("applied preferences were persisted as fresh proof")
	}
	value = preferencesForIdentity(shell)
	value.IdentityLinks = map[string]string{old: shell}
	bound, changed = value.BindProvedApplicationIdentities([]core.Application{app})
	if changed || !maps.Equal(value.IdentityLinks, bound.IdentityLinks) {
		t.Fatal("unproved target annotation authorized a historical retarget")
	}
}

func TestLiveCanonicalIdentityPreventsLegacyComponentMigration(t *testing.T) {
	app := shortcutIdentityFixture("Fixture", "first")
	old := historicalTupleID(app, false)
	app.ProvedIDs, app.LegacyTargetIDs, app.RelatedIDs = []string{old}, []string{old}, []string{old}
	shell := core.Application{Name: "Packaged", ShellID: core.AppsFolderParsingPrefix + "Synthetic!App", ShellPackaged: true}
	value := preferencesForIdentity(shell.ItemID())
	value.IdentityLinks = map[string]string{old: shell.ItemID()}
	bound, changed := value.BindProvedApplicationIdentities([]core.Application{app, shell})
	if changed || !maps.Equal(bound.IdentityLinks, value.IdentityLinks) {
		t.Fatal("a live canonical identity was redirected to a different shortcut")
	}
	applied := bound.ApplyIdentityLinks([]core.Application{app, shell})
	if applied[0].Preferred(bound.Favorites) || !applied[1].Preferred(bound.Favorites) {
		t.Fatal("live packaged preference changed owners")
	}
}

func TestConflictingProvedIDsAreNotPersisted(t *testing.T) {
	first := shortcutIdentityFixture("First", "first")
	second := shortcutIdentityFixture("Second", "second")
	old := historicalTupleID(first, false)
	for _, app := range []*core.Application{&first, &second} {
		app.ProvedIDs = []string{old}
		app.LegacyTargetIDs = []string{old}
	}
	value := preferencesForIdentity(old)
	bound, changed := value.BindProvedApplicationIdentities([]core.Application{first, second})
	if changed || len(bound.IdentityLinks) != 0 {
		t.Fatal("conflicting proof owners were guessed at the persistence boundary")
	}
	// The application's own current ID is also an owner, even when it has no
	// legacy proof entries of its own.
	first.ProvedIDs = []string{second.ItemID()}
	first.LegacyTargetIDs = first.ProvedIDs
	second.ProvedIDs, second.LegacyTargetIDs = nil, nil
	value = preferencesForIdentity(second.ItemID())
	bound, changed = value.BindProvedApplicationIdentities([]core.Application{first, second})
	if changed || len(bound.IdentityLinks) != 0 {
		t.Fatal("proof replaced another application's current identity")
	}
}

func TestNewIdentityBindingsRemainBoundedAndDeterministic(t *testing.T) {
	app := shortcutIdentityFixture("Fixture", "first")
	for i := 0; i < 1000; i++ {
		app.ProvedIDs = append(app.ProvedIDs, historicalPathID(fmt.Sprintf("Synthetic-%d.lnk", i)))
	}
	value := Defaults()
	value.Favorites = []string{app.ItemID(), app.ProvedIDs[999]}
	bound, changed := value.BindProvedApplicationIdentities([]core.Application{app})
	if !changed || len(bound.IdentityLinks) != 650 || bound.IdentityLinks[app.ProvedIDs[999]] != app.ItemID() {
		t.Fatal("binding bound was exceeded or a referenced identity was discarded")
	}
	if _, err := bound.Validated(); err != nil {
		t.Fatal("binding generated invalid settings", err)
	}
	slices.Reverse(app.ProvedIDs)
	other, _ := value.BindProvedApplicationIdentities([]core.Application{app})
	if !maps.Equal(bound.IdentityLinks, other.IdentityLinks) {
		t.Fatal("proof arrival order changed bounded saved bindings")
	}
	if _, changed := bound.BindProvedApplicationIdentities([]core.Application{app}); changed {
		t.Fatal("same evidence repeatedly churned bounded settings")
	}
}

func preferencesForIdentity(id string) Settings {
	value := Defaults()
	value.Favorites, value.Recents, value.Hidden = []string{id}, []string{id}, []string{id}
	value.Aliases = map[string][]string{id: {"bound-alias"}}
	return value
}

func restartIdentitySettings(t *testing.T, value Settings) Settings {
	t.Helper()
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	return loaded
}

func assertIdentityPreferences(t *testing.T, value Settings, app core.Application) {
	t.Helper()
	apps := value.ApplyIdentityLinks([]core.Application{app})
	if !apps[0].Preferred(value.Favorites) || !apps[0].Preferred(value.Recents) || !apps[0].Preferred(value.Hidden) {
		t.Fatal("restart lost ordinary preferences")
	}
	if len(core.SearchApplications(core.WithUserAliases(apps, value.Aliases), "bound-alias", 10)) != 1 {
		t.Fatal("restart lost alias")
	}
}

func shortcutIdentityFixture(name, directory string) core.Application {
	return core.Application{Name: name, Path: `C:\Synthetic\` + directory + `\` + name + `.lnk`, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, WorkingDirectory: `C:\Synthetic`, AppUserModelIDKnown: true}}
}

func historicalPathID(path string) string {
	return fmt.Sprintf("app-%x", sha256.Sum256([]byte(historicalPathKey(path))))
}

func historicalPathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		path = strings.ToLower(path)
	}
	return path
}

func historicalTupleID(app core.Application, includeAppID bool) string {
	identity := fmt.Sprintf("%q|%q|%q", historicalPathKey(app.Launch.Target), app.Launch.Arguments, historicalPathKey(app.Launch.WorkingDirectory))
	if includeAppID && app.Launch.AppUserModelID != "" {
		identity += fmt.Sprintf("|app-id:%q", app.Launch.AppUserModelID)
	}
	return fmt.Sprintf("app-%x", sha256.Sum256([]byte(identity)))
}
