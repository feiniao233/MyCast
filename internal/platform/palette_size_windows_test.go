//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestPaletteResizeOwnedHiddenWindowABI(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if unsafe.Sizeof(paletteNativeRect{}) != 16 || unsafe.Sizeof(paletteMonitorData{}) != 40 {
		t.Fatal("native geometry ABI")
	}
	class, _ := syscall.UTF16PtrFromString("STATIC")
	title, _ := syscall.UTF16PtrFromString("MyCast synthetic compact geometry fixture")
	hwnd, _, err := createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0x80000000, 0, 0, 750, 475, 0, 0, 0, 0) // owned borderless WS_POPUP
	if hwnd == 0 {
		t.Fatal(err)
	}
	defer destroyWindow.Call(hwnd)
	if _, err := ResizePalette(0, 750, 108, 475, true); err == nil {
		t.Fatal("zero ownership accepted")
	}
	if _, err := ResizePalette(hwnd, 750, 108, 475, true); err != nil {
		t.Fatal(err)
	}
	var anchor paletteNativeRect
	paletteWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&anchor)))
	visible := user32.NewProc("IsWindowVisible")
	for i := 0; i < 64; i++ {
		for _, height := range []int{475, 108} {
			if _, err := ResizePalette(hwnd, 750, height, 475, false); err != nil {
				t.Fatal(err)
			}
			var rect paletteNativeRect
			paletteWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&rect)))
			if rect.Top != anchor.Top || rect.Left != anchor.Left {
				t.Fatal("native top anchor moved")
			}
			shown, _, _ := visible.Call(hwnd)
			if shown != 0 {
				t.Fatal("hidden resize showed fixture")
			}
		}
	}
}
