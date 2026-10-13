//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestOpenVPNApplicationOnlyFixtureAndSearch(t *testing.T) {
	root := t.TempDir()
	links := filepath.Join(root, "links")
	if err := os.Mkdir(links, 0700); err != nil {
		t.Fatal(err)
	}
	gui := filepath.Join(root, "gui.exe")
	manual := filepath.Join(root, "manual.html")
	notes := filepath.Join(root, "notes.txt")
	cli := filepath.Join(root, "vpncmd.exe")
	folder := filepath.Join(root, "config")
	for _, path := range []string{gui, manual, notes, cli} {
		if err := os.WriteFile(path, []byte("dedicated inert fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	identities := map[string]core.LaunchIdentity{
		"OpenVPN GUI.lnk": {Target: gui}, "OpenVPN Manual Page.lnk": {Target: manual}, "OpenVPN Windows Notes.lnk": {Target: notes},
		"OpenVPN Log File Directory.lnk": {Target: folder}, "OpenVPN Sample Configuration Files.lnk": {Target: folder}, "OpenVPN Configuration File Directory.lnk": {Target: folder},
		"SoftEther VPN 命令行实用工具(vpncmd).lnk": {Target: cli},
	}
	for name := range identities {
		if err := os.WriteFile(filepath.Join(links, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	options := core.DefaultScanOptions()
	options.AllowPath = LocalAppPath
	options.IncludeResolvedTarget = IncludeApplicationTarget
	apps, errs := core.IndexApplicationsWithOptions(context.Background(), []string{links}, func(path string) (core.LaunchIdentity, error) { return identities[filepath.Base(path)], nil }, options)
	if len(errs) != 0 || len(apps) != 2 {
		t.Fatalf("known target fixture classification: %d %v", len(apps), errs)
	}
	apps = core.PrepareSearch(apps)
	if got := core.SearchApplications(apps, "openvpn", 10); len(got) != 1 || got[0].Name != "OpenVPN GUI" {
		t.Fatal("fixture query retained unrelated/non-application entries")
	}
	if got := core.SearchApplications(apps, "vpncmd", 10); len(got) != 1 {
		t.Fatal("valid CLI executable was hidden")
	}
}

func TestApplicationTargetTypesWithoutNameBlacklist(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"Manual.exe", "Gui.html", "notes.txt", "CLI.exe", "script.cmd", "profile.ovpn"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("dedicated fixture; never executed"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	folder := filepath.Join(root, "Configuration")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		keep bool
	}{{"Manual.exe", true}, {"Gui.html", false}, {"notes.txt", false}, {"CLI.exe", true}, {"script.cmd", true}, {"profile.ovpn", true}, {"Configuration", false}, {"Missing.exe", true}} {
		if got := IncludeApplicationTarget(core.LaunchIdentity{Target: filepath.Join(root, test.name)}); got != test.keep {
			t.Fatalf("target %s keep=%v", test.name, got)
		}
	}
}

func TestApplicationTargetRefusalBeforeStat(t *testing.T) {
	for _, path := range []string{`\\server\share\Folder`, `Z:\Mapped\Folder`, `C:\Synthetic\Reparse\Folder`} {
		calls := 0
		got := includeApplicationTarget(path, func(string) error { return errors.New("synthetic denied volume/reparse") }, func(string) (fs.FileInfo, error) { calls++; return nil, errors.New("must not run") })
		if !got || calls != 0 {
			t.Fatal("denied target was followed or guessed")
		}
	}
	calls := 0
	if includeApplicationTarget(`C:\Synthetic\Manual.HTML`, func(string) error { calls++; return nil }, os.Lstat) || calls != 0 {
		t.Fatal("document classification accessed filesystem")
	}
}
