//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"slices"
	"testing"
)

func TestLegacyWindowsPathIdentityGoldens(t *testing.T) {
	// Fixed hashes of the former Windows Unicode simple-fold path keys.
	// These inputs require no installed apps, filesystem access or shortcuts.
	for _, test := range []struct{ source, canonical, id string }{
		{`C:\Synthetic\Fixture.lnk`, `C:\SYNTHETIC\FIXTURE.LNK`, "app-4a5bd4767c284e5b8ce35b28f0bc6b829dba583a08536d51ee1f19be70b17d49"},
		{`C:\Synthetic\.\sub\..\Fixture.lnk`, `C:\SYNTHETIC\FIXTURE.LNK`, "app-4a5bd4767c284e5b8ce35b28f0bc6b829dba583a08536d51ee1f19be70b17d49"},
		{`C:\Σynthetic\σς.lnk`, `C:\ΣYNTHETIC\ΣΣ.LNK`, "app-c226faf43ecbf066a99ff401cd1ea358d442e032c7acb23eb136b4557ce62d02"},
		{`C:\Kelvin\kK.lnk`, `C:\KELVIN\KK.LNK`, "app-38b917927b164f8d633f4c5bf0c589d05a6b44d63a955fe55674fe3a28c2ff89"},
		{`C:\µ\μΜ.lnk`, `C:\µ\µµ.LNK`, "app-0d289e5cc5321a0fef94a3fe5de9971567be825071dd6667060e4f91091a4b12"},
	} {
		if got := pathKey(test.source); got != test.canonical {
			t.Fatalf("Windows path key=%q, want %q", got, test.canonical)
		}
		app := Application{Name: "Fixture", Path: test.source}
		if got := legacyApplicationID(app); got != test.id {
			t.Fatalf("historical identity=%s, want %s", got, test.id)
		}
		merged := MergeProviderApplications([]Application{app})
		if len(merged) != 1 || !slices.Contains(merged[0].ProvedIDs, test.id) {
			t.Fatal("fixed historical path ID was not offered for migration")
		}
	}
}

func TestLegacyWindowsTargetTupleIdentityGolden(t *testing.T) {
	app := Application{Name: "Fixture", Path: `C:\Synthetic\Fixture.lnk`, Launch: LaunchIdentity{
		Target: `C:\Synthetic\Kelvinς.exe`, Arguments: `--KeepCase "A  B"`,
		WorkingDirectory: `C:\Synthetic\Work`, AppUserModelID: "org.Synthetic.Desktop", AppUserModelIDKnown: true,
	}}
	// SHA-256 of the historical quoted canonical target/args/directory tuple
	// plus exact AppID. Path case folds; arguments and AppID retain their case.
	const want = "app-9e83ec737c8f53af67131e7681fa0c1ac6ec64b74e5210bd22e0f50009854c9e"
	if got := legacyApplicationID(app); got != want {
		t.Fatalf("historical target/AppID identity=%s, want %s", got, want)
	}
	merged := MergeProviderApplications([]Application{app})
	if len(merged) != 1 || !slices.Contains(merged[0].LegacyTargetIDs, want) {
		t.Fatal("fixed historical target/AppID hash was not offered for migration")
	}
}

func TestLegacyWindowsEmptyDirectoryGolden(t *testing.T) {
	app := Application{Name: "Fixture", Path: `C:\Synthetic\Fixture.lnk`, Launch: LaunchIdentity{Target: `C:\Synthetic\Host.exe`, Arguments: "--MixedCase", AppUserModelIDKnown: true}}
	// filepath.Clean("") historically contributed ".", not an empty string.
	const want = "app-84f7a8dbf7540ad5d7537e65ac97a7007027a29e9cddb2ca4a894a35c554b24d"
	if got := legacyApplicationID(app); got != want {
		t.Fatalf("empty-directory historical identity=%s, want %s", got, want)
	}
}
