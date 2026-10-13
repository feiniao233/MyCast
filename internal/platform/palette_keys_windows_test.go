//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestPaletteRepeatClassification(t *testing.T) {
	for _, key := range []uintptr{0x0d, 0x1b, 0x5d, 0x79} {
		if !suppressPaletteRepeat(0x100, key, 1<<30, false) || suppressPaletteRepeat(0x100, key, 0, false) || suppressPaletteRepeat(0x101, key, 1<<30, false) || suppressPaletteRepeat(0x100, key, 1<<30, true) {
			t.Fatal("repeat/IME classification", key)
		}
	}
	if !suppressPaletteRepeat(0x104, 0x79, 1<<30, false) || suppressPaletteRepeat(0x104, 0x79, (1<<30)|(1<<29), false) || suppressPaletteRepeat(0x104, 0x79, 0, false) || suppressPaletteRepeat(0x104, 0x79, 1<<30, true) {
		t.Fatal("system F10 repeat/Alt/first/IME classification")
	}
	for _, key := range []uintptr{0x0d, 0x1b, 0x5d, 0x73, 0x41} {
		if suppressPaletteRepeat(0x104, key, 1<<30, false) || suppressPaletteRepeat(0x104, key, (1<<30)|(1<<29), false) {
			t.Fatal("other system key changed", key)
		}
	}
	for _, key := range []uintptr{0x26, 0x28, 0x08, 0x20, 0x41} {
		if suppressPaletteRepeat(0x100, key, 1<<30, false) {
			t.Fatal("navigation/text repeat blocked", key)
		}
	}
}

func TestPaletteRepeatNativeSubclassOwnershipAndCleanup(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := syscall.UTF16PtrFromString("STATIC")
	title, _ := syscall.UTF16PtrFromString("MyCast synthetic key guard fixture")
	hwnd, _, err := createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0, 0, 0, 100, 100, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatal("owned hidden fixture", err)
	}
	defer destroyWindow.Call(hwnd)
	if _, err := InstallPaletteKeyGuard(hwnd); err == nil {
		t.Fatal("unrecognized surface accepted")
	}
	dll, err := paletteKeysDLL()
	if err != nil {
		t.Fatal(err)
	}
	count := 0
	receiver := syscall.NewCallback(func(h, m, w, l, id, ref uintptr) uintptr {
		if m == 0x100 || m == 0x101 || m == 0x104 {
			count++
			return 0
		}
		r, _, _ := dll.NewProc("DefSubclassProc").Call(h, m, w, l)
		return r
	})
	ok, _, _ := dll.NewProc("SetWindowSubclass").Call(hwnd, receiver, 9001, 0)
	if ok == 0 {
		t.Fatal("fixture receiver subclass")
	}
	defer dll.NewProc("RemoveWindowSubclass").Call(hwnd, receiver, 9001)
	guard, err := installPaletteKeyGuardSurface(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := installPaletteKeyGuardSurface(hwnd); err == nil {
		t.Fatal("duplicate guard accepted")
	}
	send := user32.NewProc("SendMessageW")
	for _, key := range []uintptr{0x0d, 0x1b, 0x5d, 0x79} {
		before := count
		send.Call(hwnd, 0x100, key, 0)
		send.Call(hwnd, 0x100, key, 1<<30)
		if count != before+1 {
			t.Fatal("native repeated action reached receiver", key, count, before)
		}
	}
	before := count
	send.Call(hwnd, 0x104, 0x79, 0)
	send.Call(hwnd, 0x104, 0x79, 1<<30)
	send.Call(hwnd, 0x104, 0x79, (1<<30)|(1<<29))
	if count != before+2 {
		t.Fatal("real system F10 message bypassed repeat or blocked Alt/first")
	}
	before = count
	send.Call(hwnd, 0x100, 0x28, 1<<30)
	send.Call(hwnd, 0x101, 0x0d, 1<<30)
	send.Call(hwnd, 0x104, 0x73, (1<<30)|(1<<29)) // Alt+F4 system context.
	if count != before+3 {
		t.Fatal("navigation/release/system keys changed")
	}
	send.Call(hwnd, 0x10d, 0, 0)
	before = count
	send.Call(hwnd, 0x100, 0x0d, 1<<30)
	if count != before+1 {
		t.Fatal("native composing key consumed")
	}
	send.Call(hwnd, 0x10e, 0, 0)
	guard.SetComposing(true)
	before = count
	send.Call(hwnd, 0x100, 0x1b, 1<<30)
	if count != before+1 {
		t.Fatal("UI composing key consumed")
	}
	guard.SetComposing(false)
	guard.SetComposing(true)
	before = count
	send.Call(hwnd, 0x104, 0x79, 1<<30)
	if count != before+1 {
		t.Fatal("composing system F10 was consumed")
	}
	guard.SetComposing(false)
	if !guard.Close() || !guard.Close() {
		t.Fatal("guard cleanup not idempotent")
	}
	before = count
	send.Call(hwnd, 0x100, 0x0d, 1<<30)
	if count != before+1 {
		t.Fatal("removed subclass still filtering")
	}
	guard, err = installPaletteKeyGuardSurface(hwnd)
	if err != nil {
		t.Fatal(err)
	}
	destroyWindow.Call(hwnd)
	if !guard.Close() {
		t.Fatal("WM_NCDESTROY did not release token")
	}
	if _, err = installPaletteKeyGuardSurface(0); err == nil {
		t.Fatal("zero HWND accepted")
	}
}
