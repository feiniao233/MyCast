// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import "testing"

func TestProvenVirtualShortcutMergeKeepsLegacyPreferencesAndVariants(t *testing.T) {
	id := AppsFolderParsingPrefix + "Synthetic.Package!App"
	old := Application{Name: "File branding", Path: `C:\Synthetic\Fixture.lnk`}
	oldID := old.ItemID()
	link := old
	link.Launch = LaunchIdentity{ShellTarget: id, ShowCommand: 1, AppUserModelIDKnown: true}
	link.RelatedIDs = []string{oldID}
	shell := Application{Name: "系统名称", ShellID: id}
	merged := PrepareSearch(MergeProviderApplications([]Application{shell, link}))
	if len(merged) != 1 || merged[0].Path != link.Path || !merged[0].HasPreferenceID(oldID) || merged[0].ItemID() != shell.ItemID() {
		t.Fatal("proved source association lost adapter/identity")
	}
	merged = WithUserAliases(merged, map[string][]string{oldID: {"legacy-alias"}})
	if len(SearchApplications(merged, "legacy-alias", 10)) != 1 || !merged[0].Preferred([]string{oldID}) {
		t.Fatal("old alias/favorite lost")
	}
	restored := CarryRelatedPreferenceIDs([]Application{shell}, merged)
	if !restored[0].HasPreferenceID(oldID) {
		t.Fatal("file removal lost preference bridge to surviving Shell identity")
	}
	for _, variant := range []LaunchIdentity{{ShellTarget: id, Arguments: "--different"}, {ShellTarget: id, WorkingDirectory: `C:\Synthetic`}, {ShellTarget: id, Independent: true, LinkFlags: 0x2000}, {Target: `C:\Synthetic\Unrelated.exe`}} {
		separate := old
		separate.Launch = variant
		if len(MergeProviderApplications([]Application{shell, separate})) != 2 {
			t.Fatal("unproven launch semantic merged")
		}
	}
}
