// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import "testing"

func TestDesktopProviderAssociationUsesMetadataAndKeepsVariants(t *testing.T) {
	id := AppsFolderParsingPrefix + "org.synthetic.desktop"
	shell := Application{Name: "Localized Namespace", ShellID: id, ShellDesktop: true, Launch: LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, AppUserModelID: "org.synthetic.desktop"}}
	file := Application{Name: "Different Link Name", Path: `C:\Synthetic\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, WorkingDirectory: `C:\Synthetic\`, ShowCommand: 1, AppUserModelID: "org.synthetic.desktop"}}
	old := file.ItemID()
	merged := MergeProviderApplications([]Application{file, shell})
	if len(merged) != 1 || merged[0].ItemID() != shell.ItemID() || !merged[0].HasPreferenceID(old) || merged[0].Path != file.Path || merged[0].Launch.WorkingDirectory != file.Launch.WorkingDirectory || merged[0].ShellFallback == nil {
		t.Fatal("default-directory desktop association lost identity/adapter/preferences")
	}
	if file.Launch.ShellTarget != "" {
		t.Fatal("merge modified caller snapshot")
	}
	for _, change := range []func(*Application){
		func(a *Application) { a.Launch.Arguments = "--profile other" },
		func(a *Application) { a.Launch.WorkingDirectory = `C:\Other` },
		func(a *Application) { a.Launch.Independent = true },
		func(a *Application) { a.Launch.AppUserModelID = "org.synthetic.other" },
		func(a *Application) { a.Launch.Target = `C:\Other\Fixture.exe` },
	} {
		variant := file
		change(&variant)
		if len(MergeProviderApplications([]Application{variant, shell})) != 2 {
			t.Fatal("different launch semantics merged")
		}
	}
	bare := file
	bare.Launch.WorkingDirectory = ""
	bare.Launch.AppUserModelID = ""
	if len(MergeProviderApplications([]Application{bare, shell})) != 1 {
		t.Fatal("unique empty/default desktop profile did not merge")
	}
	other := shell
	other.ShellID = AppsFolderParsingPrefix + "org.synthetic.other"
	other.Launch.AppUserModelID = "org.synthetic.other"
	if len(MergeProviderApplications([]Application{bare, shell, other})) != 3 {
		t.Fatal("ambiguous target-only namespaces merged")
	}
	unproved := shell
	unproved.ShellDesktop = false
	if len(MergeProviderApplications([]Application{file, unproved})) != 2 {
		t.Fatal("missing or failed namespace metadata proved equivalence")
	}
}
