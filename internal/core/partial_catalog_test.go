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
	shell := Application{Name: "Shell", ShellID: AppsFolderParsingPrefix + "Fixture!App"}
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
