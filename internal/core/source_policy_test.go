// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestDesktopShallowPolicyAndCustomRecursiveRoots(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "Child")
	if err := os.Mkdir(child, 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(root, "Top.exe"), filepath.Join(child, "Nested.exe")} {
		if err := os.WriteFile(path, []byte("inert"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	opts := DefaultScanOptions()
	opts.ShallowRoots = []string{root}
	apps, problems := IndexApplicationsWithOptions(context.Background(), []string{root}, nil, opts)
	if len(apps) != 1 || apps[0].Name != "Top" || len(problems) != 0 {
		t.Fatal("default desktop entered children or reported false budget")
	}
	opts.ShallowRoots = nil
	apps, problems = IndexApplicationsWithOptions(context.Background(), []string{root}, nil, opts)
	if len(apps) != 2 || len(problems) != 0 {
		t.Fatal("custom source recursion changed")
	}
}

func TestUninstallActionsAndOrdinaryUtilityCounterexamples(t *testing.T) {
	cases := []struct {
		name, target, args string
		uninstall          bool
	}{
		{"卸载示例", `C:\Synthetic\unins000.exe`, "", true},
		{"示例卸载", `C:\Synthetic\uninstall.exe`, "", true},
		{"Uninstall Fixture", `C:\Synthetic\Uninstall.exe`, "", true},
		{"Fixture", `C:\Windows\System32\msiexec.exe`, `/x "{00000000-0000-0000-0000-000000000000}" /qn`, true},
		{"Browser", `C:\Synthetic\browser.exe`, "--uninstall", true},
		{"Revo Uninstaller", `C:\Synthetic\RevoUnin.exe`, "", false},
		{"卸载工具", `C:\Synthetic\UninstallTool.exe`, "", false},
		{"Uninstall Helper", `C:\Synthetic\Ordinary.exe`, "--help", false},
		{"Installer", `C:\Windows\System32\msiexec.exe`, "/i Fixture.msi", false},
		{"Browser", `C:\Synthetic\browser.exe`, "--uninstall-help", false},
		{"Fixture", `C:\Synthetic\ordinary.exe`, `"an argument containing --uninstall"`, false},
	}
	for _, c := range cases {
		t.Run(c.name+"/"+c.args, func(t *testing.T) {
			app := Application{Name: c.name, Path: c.target, Launch: LaunchIdentity{Target: c.target, Arguments: c.args}}
			if got := IsUninstallEntry(app); got != c.uninstall {
				t.Fatalf("got %t want %t", got, c.uninstall)
			}
		})
	}
}

func TestFairLimitDoesNotTruncateAllShellApps(t *testing.T) {
	var apps []Application
	for i := 0; i < 5000; i++ {
		apps = append(apps, Application{Name: fmt.Sprint(i), Path: fmt.Sprintf(`C:\Synthetic\%d.exe`, i)})
	}
	shell := Application{Name: "Microsoft Store", ShellPackaged: true, ShellID: AppsFolderParsingPrefix + "Microsoft.WindowsStore_8wekyb3d8bbwe!App"}
	apps = append(apps, shell)
	limited := FairProviderLimit(apps, nil, 5000)
	if len(limited) != 5000 {
		t.Fatal("limit not enforced")
	}
	found := false
	for _, app := range limited {
		found = found || app.ItemID() == shell.ItemID()
	}
	if !found {
		t.Fatal("file append order erased Shell")
	}
}
