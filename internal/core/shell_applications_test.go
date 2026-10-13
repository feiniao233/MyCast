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

func TestStoreBareShellIdentityAndActivationForm(t *testing.T) {
	const leaf = "Microsoft.WindowsStore_8wekyb3d8bbwe!App"
	id, err := AppsFolderChildIdentity(leaf)
	if err != nil || id != AppsFolderParsingPrefix+leaf {
		t.Fatal("bare AUMID rejected")
	}
	app := Application{Name: "Microsoft Store", ShellID: id, Aliases: SystemApplicationAliases(id)}
	for _, query := range []string{"store", "微软商店", "weiruanshangdian"} {
		if len(SearchApplications(PrepareSearch([]Application{app}), query, 10)) != 1 {
			t.Fatal("Store search missing", query)
		}
	}
	activation, err := ShellApplicationActivationName(id)
	if err != nil || activation != `shell:AppsFolder\`+leaf {
		t.Fatal("wrong Shell activation form")
	}
	if (Application{ShellID: AppsFolderParsingPrefix + leaf}).ItemID() != app.ItemID() {
		t.Fatal("canonical preference identity changed")
	}
	for _, bad := range []string{`C:\Synthetic.exe`, `shell:Other`, leaf + `\Nested`, "", leaf + "\x00"} {
		if _, err := AppsFolderChildIdentity(bad); err == nil {
			t.Fatal("unsafe child accepted")
		}
	}
	if len(SystemApplicationAliases(AppsFolderParsingPrefix+"Unrelated!App")) != 0 {
		t.Fatal("aliases assigned by name")
	}
}
