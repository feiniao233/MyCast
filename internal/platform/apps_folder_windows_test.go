//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestNativeShellItemNamesEnumerationAndIcon(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("fixture apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	directory := filepath.Join(t.TempDir(), "Synthetic Shell Fixture")
	if err := os.Mkdir(directory, 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, "Fixture.exe"), []byte("inert"), 0600); err != nil {
		t.Fatal(err)
	}
	wide, _ := syscall.UTF16PtrFromString(directory)
	var item *comObject
	hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
	if err := comError("fixture Shell item", hr); err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Fatal("missing item")
	}
	defer item.call(2)
	for n := 0; n < 32; n++ {
		name, err := shellDisplayName(item, 0)
		if err != nil || name != "Synthetic Shell Fixture" {
			t.Fatalf("display name %q: %v", name, err)
		}
		parse, err := shellDisplayName(item, 0x80028000)
		if err != nil {
			t.Fatal(err)
		}
		parsedInfo, parsedErr := os.Stat(parse)
		fixtureInfo, fixtureErr := os.Stat(directory)
		if parsedErr != nil || fixtureErr != nil || !os.SameFile(parsedInfo, fixtureInfo) {
			t.Fatal("Shell parse name did not identify the owned fixture directory")
		}
		if im := shellItemIcon(item); im == nil || im.Bounds().Dx() != 32 || im.Bounds().Dy() != 32 {
			t.Fatal("Shell icon bitmap ABI/copy failed")
		}
	}
	// The real native enumerator sees just the inert fixture. Filesystem Shell
	// identities are intentionally rejected by the AppsFolder-only provider.
	apps, _, errs := enumerateShellApplications(context.Background(), item, 10, 0)
	if len(apps) != 0 || len(errs) != 1 {
		t.Fatalf("fixture enumeration: %d apps %d errors", len(apps), len(errs))
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, errs = enumerateShellApplications(ctx, item, 10, 0)
	if len(errs) != 1 || errs[0] != context.Canceled {
		t.Fatal("cancel not checked")
	}
}

func TestShellExecuteInfoABIAndInvalidIdentity(t *testing.T) {
	if unsafe.Sizeof(uintptr(0)) == 8 {
		var info shellExecuteInfo
		if unsafe.Sizeof(info) != 112 || unsafe.Offsetof(info.IDList) != 64 || unsafe.Offsetof(info.Process) != 104 {
			t.Fatal("SHELLEXECUTEINFOW ABI")
		}
	}
	if LaunchShellApplication(`C:\Synthetic.exe`) == nil {
		t.Fatal("filesystem path reached virtual activation")
	}
}
