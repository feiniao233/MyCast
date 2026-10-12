//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestNativeLocalPathDedicatedFixture(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Synthetic.exe")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	if err := LocalAppPath(path); err != nil {
		t.Fatal("local native metadata check rejected fixture:", err)
	}
	if err := LocalAppPath(filepath.Join(root, "Missing.exe")); err == nil {
		t.Fatal("missing path was confirmed")
	}
}

func TestRemoteAndUnconfirmedVolumesRefusedBeforeAccess(t *testing.T) {
	for _, kind := range []uint32{0, 1, 4, 99} {
		calls := 0
		err := checkLocalAppPath(`Z:\Apps`, func(root string) uint32 {
			if root != `Z:\` {
				t.Fatal("wrong drive metadata root")
			}
			return kind
		}, func(string) (uint32, error) { calls++; return 0, nil })
		if err == nil || calls != 0 {
			t.Fatal("remote/unknown volume touched path metadata")
		}
	}
	for _, path := range []string{`\\server\share\Apps`, `\\?\C:\Apps`, `C:Apps`, `relative`} {
		calls := 0
		err := checkLocalAppPath(path, func(string) uint32 { calls++; return 3 }, func(string) (uint32, error) { calls++; return 0, nil })
		if err == nil || calls != 0 {
			t.Fatal("unsupported path accessed metadata")
		}
	}
}

func TestLocalPathRejectsReparseAncestorBeforeDeeperLookup(t *testing.T) {
	var paths []string
	err := checkLocalAppPath(`C:\Fixture\Junction\Apps`, func(string) uint32 { return 3 }, func(path string) (uint32, error) {
		paths = append(paths, path)
		if path == `C:\Fixture\Junction` {
			return 0x400, nil
		}
		return 0x10, nil
	})
	if err == nil || len(paths) != 2 {
		t.Fatal("reparse ancestor was followed")
	}
	if err := checkLocalAppPath(`C:\Fixture\Apps`, func(string) uint32 { return 3 }, func(string) (uint32, error) { return 0, errors.New("unavailable") }); err == nil {
		t.Fatal("unconfirmed path accepted")
	}
	for _, kind := range []uint32{2, 3, 5, 6} {
		if err := checkLocalAppPath(`C:\Fixture\Apps`, func(string) uint32 { return kind }, func(string) (uint32, error) { return 0x10, nil }); err != nil {
			t.Fatal("confirmed local path rejected")
		}
	}
}
