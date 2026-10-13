// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"strings"
	"testing"
)

func TestShellApplicationIdentity(t *testing.T) {
	id := AppsFolderParsingPrefix + "Synthetic.Package!App"
	a := Application{Name: "Old", ShellID: id}
	b := a
	b.Name = "新名称"
	b.UserAliases = []string{"fixture"}
	if a.ItemID() != b.ItemID() || a.Identity() == (Application{Path: id}).Identity() {
		t.Fatal("display name/path affected shell identity")
	}
	if _, err := ValidateShellApplication(strings.ToUpper(AppsFolderParsingPrefix) + "Synthetic.Package!App"); err != nil {
		t.Fatal(err)
	}
	for _, v := range []string{"", `C:\Fixture.exe`, "shell:AppsFolder", AppsFolderParsingPrefix, id + `\Child`, id + "\x00", id + `"`, strings.Repeat("a", 2049)} {
		if _, err := ValidateShellApplication(v); err == nil {
			t.Fatalf("accepted %q", v)
		}
	}
	independent := Application{Name: a.Name, Path: `C:\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Fixture.exe`, Arguments: "--different", WorkingDirectory: `C:\Fixture`, Independent: true}}
	if len(SearchApplications([]Application{a, independent}, "", 0)) != 2 {
		t.Fatal("unproven Shell/shortcut equivalence merged")
	}
}
