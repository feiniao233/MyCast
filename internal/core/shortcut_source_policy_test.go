// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"
)

func TestSameNameTargetDedupUsesSourcePriorityAndOriginalLink(t *testing.T) {
	root := t.TempDir()
	higher, lower := filepath.Join(root, "Z-user"), filepath.Join(root, "A-common")
	for _, dir := range []string{higher, lower} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Fixture.lnk"), []byte("inert original"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(root, "Fixture.exe")
	resolver := func(source string) (LaunchIdentity, error) {
		return LaunchIdentity{Target: target, Arguments: source, WorkingDirectory: filepath.Dir(source), AppUserModelIDKnown: true}, nil
	}
	apps, errs := IndexApplicationsWithOptions(context.Background(), []string{higher, lower}, resolver, DefaultScanOptions())
	if len(errs) != 0 || len(apps) != 1 || apps[0].Path != filepath.Join(higher, "Fixture.lnk") || apps[0].Launch.Arguments != apps[0].Path || apps[0].Launch.WorkingDirectory != higher || len(apps[0].NamePaths) != 2 {
		t.Fatalf("wrong source or mutated launch metadata: %#v, %v", apps, errs)
	}
	reordered, reorderErrs := IndexApplicationsWithResolver([]string{lower, higher}, resolver)
	if len(reorderErrs) != 0 || len(reordered) != 1 || reordered[0].Path != filepath.Join(lower, "Fixture.lnk") || reordered[0].ItemID() != apps[0].ItemID() {
		t.Fatal("configured root priority change lost canonical identity or selected wrong link")
	}
	first := apps[0]
	preferred := Application{Name: "Fixture", SourcePriority: 0, Path: filepath.Join(higher, "Fixture.lnk"), Launch: first.Launch}
	variant := preferred
	variant.Path, variant.SourcePriority = filepath.Join(lower, "Fixture.lnk"), 1
	variant.Launch.Arguments, variant.Launch.WorkingDirectory = "--other", lower
	want := MergeProviderApplications([]Application{preferred, variant})
	got := MergeProviderApplications([]Application{variant, preferred})
	if !reflect.DeepEqual(got, want) || len(got) != 1 || got[0].Path != preferred.Path || !reflect.DeepEqual(got[0].Launch, preferred.Launch) {
		t.Fatal("arrival order selected the launch adapter")
	}
	for _, old := range []string{legacyApplicationID(preferred), legacyApplicationID(variant)} {
		if !got[0].HasPreferenceID(old) || !slices.Contains(got[0].ProvedIDs, old) {
			t.Fatal("merged parameter variant lost migration proof")
		}
	}
}

func TestShortcutKeyCaseNameTargetFallbackAndBareExecutable(t *testing.T) {
	app := Application{Name: "Fixture", Path: `C:\User\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Apps\Fixture.exe`, Arguments: "--one", AppUserModelIDKnown: true}}
	other := app
	other.Name, other.Path, other.Launch.Target = "FIXTURE", `C:\Common\Fixture.LNK`, `c:\apps\fixture.EXE`
	other.Launch.Arguments, other.Launch.WorkingDirectory, other.Launch.AppUserModelID = "--two", `C:\Elsewhere`, "another.desktop.id"
	if app.ItemID() != other.ItemID() {
		t.Fatal("name/target case or args/workdir/AppID changed shortcut key")
	}
	other.Name = "Fixture Safe Mode"
	if app.ItemID() == other.ItemID() {
		t.Fatal("different names collapsed")
	}
	other = app
	other.Launch.Target = `C:\Other\Fixture.exe`
	if app.ItemID() == other.ItemID() {
		t.Fatal("different targets collapsed")
	}
	unresolved := Application{Name: "Fixture", Path: `C:\User\Fixture.lnk`}
	other = unresolved
	other.Path = `C:\Common\Fixture.lnk`
	if unresolved.ItemID() == other.ItemID() {
		t.Fatal("unresolved links lost path fallback")
	}
	executable := app
	executable.Path = app.Launch.Target
	executable.Launch.Arguments = ""
	merged := MergeProviderApplications([]Application{executable, app})
	if len(merged) != 1 || merged[0].Path != app.Path {
		t.Fatal("bare executable displaced original shortcut")
	}
}

func TestPackagedAUMIDCaseDedupAndDesktopAppIDExclusion(t *testing.T) {
	store := Application{Name: "Microsoft Store", ShellID: AppsFolderParsingPrefix + "Microsoft.WindowsStore_8wekyb3d8bbwe!App", ShellPackaged: true}
	duplicate := store
	duplicate.Name = "微软商店"
	duplicate.ShellID = AppsFolderParsingPrefix + "microsoft.windowsstore_8wekyb3d8bbwe!app"
	desktop := Application{Name: "Desktop", ShellID: AppsFolderParsingPrefix + "org.desktop", Launch: LaunchIdentity{AppUserModelID: "org.desktop"}}
	apps := PrepareSearch(MergeProviderApplications([]Application{duplicate, desktop, store}))
	if len(apps) != 1 || len(SearchApplications(apps, "微软商店", 10)) != 1 || !apps[0].HasPreferenceID(legacyApplicationID(duplicate)) || !apps[0].HasPreferenceID(legacyApplicationID(store)) {
		t.Fatal("package dedup/search/migration or desktop source filter failed")
	}
}

func TestAmbiguousHistoricalTargetDoesNotChooseDifferentNames(t *testing.T) {
	a := Application{Name: "Alpha", Path: `C:\Alpha.lnk`, Launch: LaunchIdentity{Target: `C:\Host.exe`, AppUserModelIDKnown: true}}
	b := a
	b.Name, b.Path = "Beta", `C:\Beta.lnk`
	old := legacyApplicationID(a)
	apps := MergeProviderApplications([]Application{a, b})
	if len(apps) != 2 {
		t.Fatal("different names collapsed")
	}
	for _, app := range apps {
		if app.HasPreferenceID(old) || slices.Contains(app.ProvedIDs, old) {
			t.Fatal("ambiguous old target preference attached")
		}
	}
}

func TestUninstallVariantFilteredBeforeSourceDedup(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"A", "B"} {
		if err := os.Mkdir(filepath.Join(root, name), 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, name, "Fixture.lnk"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	apps, errs := IndexApplicationsWithResolver([]string{root}, func(source string) (LaunchIdentity, error) {
		args := ""
		if filepath.Base(filepath.Dir(source)) == "A" {
			args = "--uninstall"
		}
		return LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), Arguments: args, AppUserModelIDKnown: true}, nil
	})
	if len(errs) != 0 || len(apps) != 1 || filepath.Base(filepath.Dir(apps[0].Path)) != "B" {
		t.Fatal("uninstall source hid ordinary shortcut")
	}
}

func TestDuplicateSourceRefreshIncludesSurvivingRootOnly(t *testing.T) {
	root := t.TempDir()
	user, common, unrelated := filepath.Join(root, "user"), filepath.Join(root, "common"), filepath.Join(root, "unrelated")
	for _, dir := range []string{user, common, unrelated} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	for _, dir := range []string{user, common} {
		if err := os.WriteFile(filepath.Join(dir, "Fixture.lnk"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots := []string{user, common, unrelated}
	resolver := func(string) (LaunchIdentity, error) {
		return LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), AppUserModelIDKnown: true}, nil
	}
	apps, errs := IndexApplicationsWithResolver(roots, resolver)
	if len(errs) != 0 || len(apps) != 1 || len(apps[0].SourcePaths) != 2 {
		t.Fatal("missing duplicate source provenance", errs)
	}
	// Display preparation can discard localized-name work without losing scan provenance.
	apps[0].NamePaths = nil
	if got := ExpandDuplicateSourceRoots([]string{"unconfigured"}, roots, apps); !reflect.DeepEqual(got, []string{"unconfigured"}) {
		t.Fatal("unconfigured event expanded to full scan")
	}
	expanded := ExpandDuplicateSourceRoots([]string{user}, roots, apps)
	if !reflect.DeepEqual(expanded, []string{user, common}) {
		t.Fatalf("expanded roots=%v", expanded)
	}
	if err := os.Remove(filepath.Join(user, "Fixture.lnk")); err != nil {
		t.Fatal(err)
	}
	fresh, errs := IndexApplicationsWithResolver(expanded, resolver)
	if len(errs) != 0 || len(fresh) != 1 || fresh[0].Path != filepath.Join(common, "Fixture.lnk") || fresh[0].ItemID() != apps[0].ItemID() {
		t.Fatal("source removal lost surviving duplicate or preferences", errs)
	}
}

func TestRejectedTargetIsRemovedBeforeDuplicateSelection(t *testing.T) {
	root := t.TempDir()
	higher, lower := filepath.Join(root, "higher"), filepath.Join(root, "lower")
	for _, dir := range []string{higher, lower} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Fixture.lnk"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := DefaultScanOptions()
	options.IncludeResolvedTarget = func(launch LaunchIdentity) bool { return launch.Arguments != "--rejected-fixture" }
	apps, errs := IndexApplicationsWithOptions(context.Background(), []string{higher, lower}, func(source string) (LaunchIdentity, error) {
		args := "--valid-fixture"
		if filepath.Dir(source) == higher {
			args = "--rejected-fixture"
		}
		return LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), Arguments: args, AppUserModelIDKnown: true}, nil
	}, options)
	if len(errs) != 0 || len(apps) != 1 || apps[0].Path != filepath.Join(lower, "Fixture.lnk") {
		t.Fatal("rejected target swallowed surviving duplicate", errs)
	}
}

func TestTwoPartialScannerRoundsKeepRecoveredDuplicateDiscoverable(t *testing.T) {
	root := t.TempDir()
	aRoot, bRoot := filepath.Join(root, "A"), filepath.Join(root, "B")
	aPath, bPath := filepath.Join(aRoot, "Fixture.lnk"), filepath.Join(bRoot, "Fixture.lnk")
	for _, dir := range []string{aRoot, bRoot} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(dir, "Fixture.lnk"), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	roots := []string{aRoot, bRoot}
	resolver := func(string) (LaunchIdentity, error) {
		return LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), AppUserModelIDKnown: true}, nil
	}
	previous, errs := IndexApplicationsWithResolver(roots, resolver)
	if len(errs) != 0 || len(previous) != 1 || len(previous[0].SourcePaths) != 2 {
		t.Fatal("initial duplicate fixture", errs)
	}
	id := previous[0].ItemID()
	requested := ExpandDuplicateSourceRoots([]string{aRoot}, roots, previous)
	options := DefaultScanOptions()
	options.AllowPath = func(source string) error {
		if PathInApplicationRoots(source, []string{bRoot}) {
			return errors.New("synthetic temporary B access failure")
		}
		return nil
	}
	fresh, errs := IndexApplicationsWithOptions(context.Background(), requested, resolver, options)
	if len(errs) == 0 || len(fresh) != 1 || len(fresh[0].SourcePaths) != 1 {
		t.Fatal("failed root did not produce fresh-A partial scan", errs)
	}
	partial := RetainUnconfirmed(fresh, previous, nil, func(app Application) bool { return PathInApplicationRoots(app.Path, roots) }, len(fresh))
	if len(partial) != 1 || len(partial[0].SourcePaths) != 2 || partial[0].Path != aPath {
		t.Fatal("fresh A lost unknown B provenance")
	}
	if err := os.Remove(aPath); err != nil {
		t.Fatal(err)
	}
	requested = ExpandDuplicateSourceRoots([]string{aRoot}, roots, partial)
	if !reflect.DeepEqual(requested, roots) {
		t.Fatal("next A event failed to rescan recovered B", requested)
	}
	recovered, errs := IndexApplicationsWithResolver(requested, resolver)
	if len(errs) != 0 || len(recovered) != 1 || recovered[0].Path != bPath || recovered[0].ItemID() != id {
		t.Fatal("second scanner round lost recovered duplicate", errs)
	}
	// Healthy completion does not retain obsolete A, avoiding permanent expansion.
	if !reflect.DeepEqual(recovered[0].SourcePaths, []string{bPath}) {
		t.Fatal("healthy scan resurrected deleted source")
	}
}
