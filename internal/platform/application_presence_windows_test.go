//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"crypto/sha256"
	"errors"
	"os"
	"path/filepath"
	"syscall"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestNativeRescanDropsDeletedExeAndShortcutWithoutRepair(t *testing.T) {
	root := t.TempDir()
	links := filepath.Join(root, "links")
	program := filepath.Join(root, "program")
	for _, dir := range []string{links, program} {
		if err := os.Mkdir(dir, 0700); err != nil {
			t.Fatal(err)
		}
	}
	target := filepath.Join(program, "PicGo.exe")
	if err := os.WriteFile(target, []byte("dedicated inert fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(links, "PicGo.lnk")
	launch := core.LaunchIdentity{Target: target, WorkingDirectory: program, Arguments: "--dedicated", ShowCommand: 1}
	createTestShortcut(t, link, launch)
	before, err := os.ReadFile(link)
	if err != nil {
		t.Fatal(err)
	}
	options := core.DefaultScanOptions()
	options.AllowPath = LocalAppPath
	options.IncludeResolvedTarget = IncludeApplicationTarget
	scan := func() []core.Application {
		t.Helper()
		apps, errs := core.IndexApplicationsWithOptions(context.Background(), []string{links}, ShortcutIdentity, options)
		if len(errs) != 0 {
			t.Fatal(errs)
		}
		return apps
	}
	first := scan()
	if len(first) != 1 {
		t.Fatal("live shortcut missing")
	}
	id := first[0].ItemID()
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if apps := scan(); len(apps) != 0 || len(core.SearchApplicationsRanked(apps, "picgo", 100, []string{id}, []string{id})) != 0 {
		t.Fatal("missing target survived rescan/favorite/recent ranking")
	}
	if err := os.Remove(program); err != nil {
		t.Fatal(err)
	}
	if len(scan()) != 0 {
		t.Fatal("missing ancestor survived rescan")
	}
	after, err := os.ReadFile(link)
	if err != nil || sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("rescan repaired or edited shortcut")
	}
	launch.LinkFlags = 0x2000
	createTestShortcut(t, filepath.Join(links, "Special.lnk"), launch)
	if apps := scan(); len(apps) != 1 || !apps[0].Launch.Independent {
		t.Fatal("special target semantics discarded")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if apps := scan(); len(apps) != 1 || apps[0].Name != "Special" {
		t.Fatal("deleted source shortcut survived fresh index")
	}
}

func TestMissingFallbackUsesFreshLinkAndRetainsSpecialSource(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Live.exe")
	link := filepath.Join(root, "Fixture.lnk")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	launch := core.LaunchIdentity{Target: target, ShowCommand: 1}
	createTestShortcut(t, link, launch)
	app := core.Application{Path: link, Launch: core.LaunchIdentity{Target: filepath.Join(root, "OldGone.exe")}}
	if ConfirmedMissingApplication(app) {
		t.Fatal("cached missing target overrode freshly readable live link")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if !ConfirmedMissingApplication(app) {
		t.Fatal("fresh ordinary missing target not proven")
	}
	launch.LinkFlags = 0x2000
	createTestShortcut(t, link, launch)
	if ConfirmedMissingApplication(app) {
		t.Fatal("special source was mistaken for missing")
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if !ConfirmedMissingApplication(app) {
		t.Fatal("confirmed deleted source not proven")
	}
}

func TestPresenceMissingLeafAndAncestorStopBeforeChild(t *testing.T) {
	for _, missingErr := range []error{syscall.ERROR_FILE_NOT_FOUND, syscall.ERROR_PATH_NOT_FOUND} {
		for _, missing := range []string{`C:\Fixture\PicGo.exe`, `C:\Fixture`} {
			var paths []string
			state, _ := probeApplicationPresence(`C:\Fixture\PicGo.exe`, func(string) uint32 { return 3 }, func(path string) (uint32, error) {
				paths = append(paths, path)
				if path == missing {
					return 0, missingErr
				}
				return 0x10, nil
			})
			if state != applicationMissing || paths[len(paths)-1] != missing {
				t.Fatal("missing component was not decisive or accessed a child")
			}
		}
	}
}

func TestPresenceUnknownStatesNeverImplyDeletedOrTraverse(t *testing.T) {
	for _, kind := range []uint32{0, 1, 4, 99} {
		calls := 0
		state, _ := probeApplicationPresence(`Z:\Fixture\PicGo.exe`, func(string) uint32 { return kind }, func(string) (uint32, error) { calls++; return 0x10, nil })
		if state != applicationUnknown || calls != 0 {
			t.Fatal("network/unknown volume inspected")
		}
	}
	for _, path := range []string{`\\server\share\PicGo.exe`, `relative.exe`, `C:PicGo.exe`, `\\?\C:\PicGo.exe`} {
		calls := 0
		state, _ := probeApplicationPresence(path, func(string) uint32 { calls++; return 3 }, func(string) (uint32, error) { calls++; return 0, nil })
		if state != applicationUnknown || calls != 0 {
			t.Fatal("virtual/relative path inspected")
		}
	}
	for _, err := range []error{syscall.ERROR_ACCESS_DENIED, syscall.Errno(21), syscall.Errno(15), errors.New("unavailable")} {
		calls := 0
		state, _ := probeApplicationPresence(`C:\Fixture\Child\PicGo.exe`, func(string) uint32 { return 3 }, func(path string) (uint32, error) {
			calls++
			if path == `C:\Fixture` {
				return 0, err
			}
			return 0x10, nil
		})
		if state != applicationUnknown || calls != 2 {
			t.Fatal("unavailable ancestor was treated as missing or traversed")
		}
	}
	calls := 0
	state, _ := probeApplicationPresence(`C:\Fixture\Junction\PicGo.exe`, func(string) uint32 { return 3 }, func(path string) (uint32, error) {
		calls++
		if path == `C:\Fixture\Junction` {
			return 0x410, nil
		}
		return 0x10, nil
	})
	if state != applicationUnknown || calls != 3 {
		t.Fatal("reparse ancestor followed")
	}
	for _, kind := range []uint32{2, 5, 6} {
		state, _ := probeApplicationPresence(`Z:\PicGo.exe`, func(string) uint32 { return kind }, func(path string) (uint32, error) {
			if path == `Z:\` {
				return 0x10, nil
			}
			return 0, syscall.ERROR_PATH_NOT_FOUND
		})
		if state != applicationUnknown {
			t.Fatal("temporarily unavailable non-fixed media implied deletion")
		}
	}
	state, _ = probeApplicationPresence(`C:\PicGo.exe`, func(string) uint32 { return 3 }, func(string) (uint32, error) { return 0, syscall.ERROR_PATH_NOT_FOUND })
	if state != applicationUnknown {
		t.Fatal("unavailable root implied deleted target")
	}
}

func TestNativeMissingExeFilterAndSpecialLinks(t *testing.T) {
	root := t.TempDir()
	folder := filepath.Join(root, "uninstalled")
	if err := os.Mkdir(folder, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(folder, "PicGo.exe")
	if err := os.WriteFile(target, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if !IncludeApplicationTarget(core.LaunchIdentity{Target: target}) {
		t.Fatal("existing inert executable hidden")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	if IncludeApplicationTarget(core.LaunchIdentity{Target: target}) {
		t.Fatal("confirmed missing leaf retained")
	}
	if err := os.Remove(folder); err != nil {
		t.Fatal(err)
	}
	if IncludeApplicationTarget(core.LaunchIdentity{Target: target}) {
		t.Fatal("confirmed missing ancestor retained")
	}
	for _, launch := range []core.LaunchIdentity{{Target: target, Independent: true}, {Target: "shell:AppsFolder\\Fixture"}, {Target: "relative.exe"}, {Target: filepath.Join(root, "profile.ovpn")}} {
		if !IncludeApplicationTarget(launch) {
			t.Fatal("special/virtual/unknown association discarded")
		}
	}
}
