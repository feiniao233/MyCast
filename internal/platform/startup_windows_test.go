//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestStartupCommandBoundsAndQuotes(t *testing.T) {
	value, err := startupCommand(`C:\Synthetic Folder\MyCast.exe`)
	if err != nil || value != `"C:\Synthetic Folder\MyCast.exe" --background` {
		t.Fatal("startup command quoting")
	}
	for _, path := range []string{"", `relative.exe`, `C:\Synthetic.lnk`, `C:\bad"name.exe`, `C:\` + strings.Repeat("x", 260) + ".exe", "C:\\bad\x00.exe"} {
		if _, err := startupCommand(path); err == nil {
			t.Fatal("invalid startup path accepted")
		}
	}
}

func TestNativeStartupRegistryIsolatedFixture(t *testing.T) {
	// A unique volatile fixture key is not under Run/RunOnce/Startup. It never
	// registers an OS startup command. Cleanup deletes only this owned key.
	var random [16]byte
	if _, err := rand.Read(random[:]); err != nil {
		t.Fatal(err)
	}
	keyName := fmt.Sprintf(`Software\MyCastNativeFixture-%x`, random)
	wide, _ := syscall.UTF16PtrFromString(keyName)
	var key uintptr
	var disposition uint32
	result, _, _ := regCreate.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(wide)), 0, 0, 1, 3, 0, uintptr(unsafe.Pointer(&key)), uintptr(unsafe.Pointer(&disposition)))
	if err := registryError(result); err != nil {
		t.Fatal(err)
	}
	if key == 0 || disposition != 1 {
		if key != 0 {
			regClose.Call(key)
		}
		t.Fatal("fixture key was not newly owned")
	}
	regClose.Call(key)
	defer func() {
		result, _, _ := registryDLL.NewProc("RegDeleteKeyW").Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(wide)))
		if err := registryError(result); err != nil {
			t.Error("fixture key cleanup failed:", err)
		}
	}()
	command, err := startupCommand(`C:\Synthetic\MyCast.exe`)
	if err != nil {
		t.Fatal(err)
	}
	registration := startupRegistrationAt(keyName, startupValue, command)
	other := startupRegistry{key: keyName, name: "OtherFixture"}
	if err := other.write("unrelated-fixture"); err != nil {
		t.Fatal(err)
	}
	for n := 0; n < 16; n++ {
		if err := registration.Apply(true, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
		got, present, err := registration.Read()
		if err != nil || !present || got != command {
			t.Fatal("native REG_SZ round trip")
		}
		if err := registration.Apply(false, func() error { return errors.New("synthetic save refusal") }); err == nil {
			t.Fatal("missing rollback error")
		}
		got, present, err = registration.Read()
		if err != nil || !present || got != command {
			t.Fatal("native rollback did not restore registration")
		}
		if err := registration.Apply(false, func() error { return nil }); err != nil {
			t.Fatal(err)
		}
	}
	if got, present, err := other.read(); err != nil || !present || got != "unrelated-fixture" {
		t.Fatal("other registry value modified")
	}
	if err := registration.Write("foreign-fixture"); err != nil {
		t.Fatal(err)
	}
	if registration.Apply(false, func() error { return nil }) == nil {
		t.Fatal("foreign value deleted")
	}
	if got, present, err := registration.Read(); err != nil || !present || got != "foreign-fixture" {
		t.Fatal("collision not preserved")
	}
}
