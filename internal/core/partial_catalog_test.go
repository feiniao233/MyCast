// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"path/filepath"
	"testing"
)

func TestPartialCatalogRetainsOnlyConfiguredUnconfirmedIdentities(t *testing.T) {
	root := filepath.Join(t.TempDir(), "Configured")
	a := Application{Name: "Fresh", Path: filepath.Join(root, "A.exe")}
	b := Application{Name: "Offline", Path: filepath.Join(root, "B.exe")}
	c := Application{Name: "Missing", Path: filepath.Join(root, "C.exe")}
	removed := Application{Name: "Removed root", Path: filepath.Join(root+"Other", "D.exe")}
	shell := Application{Name: "Shell", ShellPackaged: true, ShellID: AppsFolderParsingPrefix + "Fixture!App"}
	previous := []Application{a, b, c, removed, shell}
	result := RetainUnconfirmed([]Application{a}, previous, map[string]bool{c.ItemID(): true}, func(app Application) bool {
		return app.ShellID == "" && PathInApplicationRoots(app.Path, []string{root})
	}, 5000)
	if len(result) != 2 || result[1].ItemID() != b.ItemID() {
		t.Fatalf("partial retained deleted/unconfigured source: %+v", result)
	}
	if PathInApplicationRoots(filepath.Join(root, "..", "Sibling.exe"), []string{root}) {
		t.Fatal("escaped root")
	}
	result = RetainUnconfirmed(nil, previous, nil, func(Application) bool { return true }, 2)
	if len(result) != 2 {
		t.Fatal("unbounded retention")
	}
}

func TestPartialSameIdentityRetainsOnlyUnknownConfiguredSourcePathsAtBudget(t *testing.T) {
	root := t.TempDir()
	aRoot, bRoot, removedRoot := filepath.Join(root, "A"), filepath.Join(root, "B"), filepath.Join(root, "Removed")
	aPath, bPath, removedPath := filepath.Join(aRoot, "Fixture.lnk"), filepath.Join(bRoot, "Fixture.lnk"), filepath.Join(removedRoot, "Fixture.lnk")
	old := Application{Name: "Fixture", Path: aPath, Launch: LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), AppUserModelIDKnown: true}, SourcePaths: []string{aPath, bPath, removedPath}}
	fresh := old
	fresh.SourcePaths = make([]string, 1, 4)
	fresh.SourcePaths[0] = aPath
	retain := func(app Application) bool {
		return app.ShellID == "" && PathInApplicationRoots(app.Path, []string{aRoot, bRoot})
	}
	got := RetainUnconfirmed([]Application{fresh}, []Application{old}, nil, retain, 1)
	if len(got) != 1 || len(got[0].SourcePaths) != 2 || got[0].SourcePaths[0] != aPath || got[0].SourcePaths[1] != bPath {
		t.Fatal("budget or identity dedup lost unknown configured source", got)
	}
	if len(fresh.SourcePaths) != 1 || len(old.SourcePaths) != 3 || got[0].Path != fresh.Path || got[0].Launch != fresh.Launch {
		t.Fatal("retention mutated input snapshot or selected adapter")
	}
	got[0].SourcePaths[0] = "changed"
	if fresh.SourcePaths[0] != aPath || old.SourcePaths[0] != aPath {
		t.Fatal("provenance slice aliases input")
	}

	// A known missing old selected A must not condemn sibling B. The fresh
	// same-ID adapter can come from another root while B remains unavailable.
	cRoot := filepath.Join(root, "C")
	cPath := filepath.Join(cRoot, "Fixture.lnk")
	fresh.Path, fresh.SourcePaths = cPath, []string{cPath}
	retain = func(app Application) bool { return PathInApplicationRoots(app.Path, []string{aRoot, bRoot, cRoot}) }
	got = RetainUnconfirmed([]Application{fresh}, []Application{old}, map[string]bool{old.ItemID(): true}, retain, 1)
	if len(got) != 1 || len(got[0].SourcePaths) != 2 || got[0].SourcePaths[0] != cPath || got[0].SourcePaths[1] != bPath {
		t.Fatal("selected-source absence was applied to entire group", got)
	}
	if got[0].Path != cPath || got[0].ItemID() != old.ItemID() {
		t.Fatal("retained metadata changed fresh canonical adapter")
	}
}
