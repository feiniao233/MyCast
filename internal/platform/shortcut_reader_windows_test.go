//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"crypto/sha256"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"

	"github.com/feiniao233/MyCast/internal/core"
)

func createTestShortcut(t *testing.T, path string, launch core.LaunchIdentity) {
	t.Helper()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("initialize fixture", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	var link *comObject
	hr, _, _ = createCOMInstance.Call(uintptr(unsafe.Pointer(&shellLinkClass)), 0, 1, uintptr(unsafe.Pointer(&shellLinkInterface)), uintptr(unsafe.Pointer(&link)))
	if err := comError("create fixture", hr); err != nil {
		t.Fatal(err)
	}
	defer link.call(2)
	for _, field := range []struct {
		method int
		value  string
	}{{20, launch.Target}, {11, launch.Arguments}, {9, launch.WorkingDirectory}} {
		value, err := syscall.UTF16PtrFromString(field.value)
		if err != nil {
			t.Fatal(err)
		}
		if err := comError("set fixture", link.call(field.method, uintptr(unsafe.Pointer(value)))); err != nil {
			t.Fatal(err)
		}
		runtime.KeepAlive(value)
	}
	show := launch.ShowCommand
	if show == 0 {
		show = 1
	}
	if err := comError("set fixture show mode", link.call(15, uintptr(show))); err != nil {
		t.Fatal(err)
	}
	if launch.LinkFlags != 0 {
		var data *comObject
		if err := comError("query fixture flags", link.call(0, uintptr(unsafe.Pointer(&shellLinkDataInterface)), uintptr(unsafe.Pointer(&data)))); err != nil {
			t.Fatal(err)
		}
		defer data.call(2)
		var flags uint32
		if err := comError("read fixture flags", data.call(6, uintptr(unsafe.Pointer(&flags)))); err != nil {
			t.Fatal(err)
		}
		if err := comError("set fixture flags", data.call(7, uintptr(flags|launch.LinkFlags))); err != nil {
			t.Fatal(err)
		}
	}
	var file *comObject
	if err := comError("persist fixture", link.call(0, uintptr(unsafe.Pointer(&persistFileInterface)), uintptr(unsafe.Pointer(&file)))); err != nil {
		t.Fatal(err)
	}
	defer file.call(2)
	value, _ := syscall.UTF16PtrFromString(path)
	if err := comError("save fixture", file.call(6, uintptr(unsafe.Pointer(value)), 1)); err != nil {
		t.Fatal(err)
	}
	runtime.KeepAlive(value)
}

func TestShortcutReaderRoundTripIsReadOnly(t *testing.T) {
	root := t.TempDir()
	launch := core.LaunchIdentity{Target: filepath.Join(root, "Synthetic Editor.exe"), Arguments: `--profile "test space" --flag`, WorkingDirectory: filepath.Join(root, "workspace")}
	path := filepath.Join(root, "Synthetic.lnk")
	createTestShortcut(t, path, launch)
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	got, err := ShortcutIdentity(path)
	if err != nil {
		t.Fatal(err)
	}
	if got.Target != launch.Target || got.Arguments != launch.Arguments || got.WorkingDirectory != launch.WorkingDirectory || got.ShowCommand != 1 || got.Independent {
		t.Fatalf("metadata=%+v, want %+v", got, launch)
	}
	after, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("metadata read changed shortcut")
	}
	second := filepath.Join(root, "Duplicate.lnk")
	createTestShortcut(t, second, launch)
	apps, problems := core.IndexApplicationsWithResolver([]string{root}, ShortcutIdentity)
	if len(problems) != 0 || len(apps) != 1 {
		t.Fatalf("real shortcut merge: count=%d, problems=%v", len(apps), problems)
	}
}

func TestShortcutSpecialLaunchModesRemainIndependent(t *testing.T) {
	root := t.TempDir()
	ordinary := core.LaunchIdentity{Target: filepath.Join(root, "Synthetic.exe"), ShowCommand: 1}
	admin := ordinary
	admin.LinkFlags = 0x2000 // SLDF_RUNAS_USER: fixture only; never executed.
	minimized := ordinary
	minimized.ShowCommand = 7
	unknown := ordinary
	unknown.LinkFlags = 0x20000 // Shim launch semantics are not inferred.
	for name, launch := range map[string]core.LaunchIdentity{"ordinary.lnk": ordinary, "duplicate.lnk": ordinary, "admin.lnk": admin, "minimized.lnk": minimized, "shim.lnk": unknown} {
		path := filepath.Join(root, name)
		createTestShortcut(t, path, launch)
		got, err := ShortcutIdentity(path)
		if err != nil {
			t.Fatal(err)
		}
		wantIndependent := launch.LinkFlags != 0 || launch.ShowCommand != 1
		if got.Independent != wantIndependent || got.ShowCommand != launch.ShowCommand || got.LinkFlags&launch.LinkFlags != launch.LinkFlags {
			t.Fatalf("launch mode for fixture %s was not retained: %+v", name, got)
		}
	}
	apps, problems := core.IndexApplicationsWithResolver([]string{root}, ShortcutIdentity)
	if len(problems) != 0 || len(apps) != 4 {
		t.Fatalf("ordinary duplicate must merge, permission/show/shim variants must not: count=%d, problems=%v", len(apps), problems)
	}
}

func TestIconReaderFixture(t *testing.T) {
	// This is the test executable, not an installed/user application inventory.
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	icon := ApplicationIcon(path)
	if icon == nil || icon.Bounds().Dx() != 32 || icon.Bounds().Dy() != 32 {
		t.Fatal("local executable icon extraction failed")
	}
	visible := false
	for y := 0; y < 32; y++ {
		for x := 0; x < 32; x++ {
			_, _, _, alpha := icon.At(x, y).RGBA()
			if alpha != 0 {
				visible = true
			}
		}
	}
	if !visible {
		t.Fatal("extracted icon is entirely transparent")
	}
}
