//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"crypto/sha256"
	"github.com/feiniao233/MyCast/internal/core"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestNativeShellPropertiesAndPIDLGuardPreserveSyntheticShortcut(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Fixture.exe")
	path := filepath.Join(root, "Fixture.lnk")
	if err := os.WriteFile(target, []byte("inert"), 0600); err != nil {
		t.Fatal(err)
	}
	createTestShortcut(t, path, core.LaunchIdentity{Target: target, Arguments: "--synthetic-fixture"})
	before, _ := os.ReadFile(path)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("fixture apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	wide, _ := syscall.UTF16PtrFromString(path)
	var item *comObject
	hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
	if err := comError("fixture Shell item", hr); err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Fatal("missing fixture item")
	}
	defer item.call(2)
	for i := 0; i < 16; i++ {
		got, ok := shellStringProperty(item, &linkArgumentsKey)
		if !ok || got != "--synthetic-fixture" {
			t.Fatal("IShellItem2 property ABI failed")
		}
		got, ok = shellStringProperty(item, &linkTargetKey)
		if !ok {
			t.Fatal("target property missing")
		}
		a, e1 := os.Stat(got)
		b, e2 := os.Stat(target)
		if e1 != nil || e2 != nil || !os.SameFile(a, b) {
			t.Fatal("target property did not identify owned fixture")
		}
	}
	var link *comObject
	hr, _, _ = createCOMInstance.Call(uintptr(unsafe.Pointer(&shellLinkClass)), 0, 1, uintptr(unsafe.Pointer(&shellLinkInterface)), uintptr(unsafe.Pointer(&link)))
	if err := comError("fixture link reader", hr); err != nil {
		t.Fatal(err)
	}
	if link == nil {
		t.Fatal("missing link reader")
	}
	defer link.call(2)
	var file *comObject
	if err := comError("fixture persistence", link.call(0, uintptr(unsafe.Pointer(&persistFileInterface)), uintptr(unsafe.Pointer(&file)))); err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Fatal("missing persistence")
	}
	defer file.call(2)
	if err := comError("fixture read-only load", file.call(5, uintptr(unsafe.Pointer(wide)), 0)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		if shortcutAppsFolderTarget(link) != "" {
			t.Fatal("filesystem PIDL accepted as AppsFolder immediate child")
		}
	}
	after, _ := os.ReadFile(path)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("read-only metadata changed source shortcut")
	}
}
