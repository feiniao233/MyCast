// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"reflect"
	"testing"
)

func TestDesktopNamespaceNeverCompetesWithShortcut(t *testing.T) {
	shell := Application{Name: "Localized Namespace", ShellID: AppsFolderParsingPrefix + "org.synthetic.desktop", ShellDesktop: true, Launch: LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, AppUserModelID: "org.synthetic.desktop"}}
	for _, arguments := range []string{"", "--profile other"} {
		file := Application{Name: "Different Link Name", Path: `C:\Synthetic\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, Arguments: arguments, WorkingDirectory: `C:\Other`, AppUserModelIDKnown: true}}
		merged := MergeProviderApplications([]Application{shell, file})
		if len(merged) != 1 || merged[0].Path != file.Path || merged[0].ItemID() != file.ItemID() || merged[0].ShellFallback != nil || !reflect.DeepEqual(merged[0].Launch, file.Launch) {
			t.Fatal("ordinary desktop namespace competed with original shortcut")
		}
	}
	if len(MergeProviderApplications([]Application{shell})) != 0 {
		t.Fatal("desktop namespace row survived without a configured shortcut")
	}
}

func TestOldDesktopFallbackCannotReturnAfterPartialRefresh(t *testing.T) {
	shell := Application{Name: "Old Desktop", ShellID: AppsFolderParsingPrefix + "org.synthetic.desktop", ShellDesktop: true}
	file := Application{Name: "Fixture", Path: `C:\Synthetic\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, AppUserModelIDKnown: true, DesktopBridge: true, ShellTarget: shell.ShellID}, ShellFallback: &shell}
	old := legacyApplicationID(file)
	merged := MergeProviderApplications([]Application{shell, file})
	if len(merged) != 1 || merged[0].ShellFallback != nil || merged[0].Launch.DesktopBridge || merged[0].Launch.ShellTarget != "" || !merged[0].HasPreferenceID(old) {
		t.Fatal("old desktop provider authority survived, or migration evidence was lost")
	}
}
