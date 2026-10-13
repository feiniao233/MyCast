//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"os"
	"syscall"
	"unsafe"
)

var paletteGetStyle = user32.NewProc("GetWindowLongPtrW")
var paletteSetStyle = user32.NewProc("SetWindowLongPtrW")
var paletteFrameChanged = user32.NewProc("SetWindowPos")
var paletteFindChild = user32.NewProc("FindWindowExW")

const paletteSystemMenu uint32 = 0x00080000

// RemovePaletteSystemMenu applies only to the app's own frameless palette.
// MyGo v0.4.6 removes non-client geometry but keeps WS_SYSMENU, which lets DWM
// paint caption chrome. Keep the caption/border/composition styles intact and
// refuse a window with MyGo's deliberately retained title-bar controls. Alt+F4
// is handled by the launcher through Window.Close, preserving its OnClose policy.
func RemovePaletteSystemMenu(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	var owner uint32
	getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&owner)))
	if owner != uint32(os.Getpid()) {
		return false
	}
	class, _ := syscall.UTF16PtrFromString("MyGoCaption")
	if child, _, _ := paletteFindChild.Call(hwnd, 0, uintptr(unsafe.Pointer(class)), 0); child != 0 {
		return false
	}
	before, _, _ := paletteGetStyle.Call(hwnd, ^uintptr(15)) // GWL_STYLE = -16
	if before == 0 {
		return false
	}
	if uint32(before)&paletteSystemMenu == 0 {
		return true
	}
	after := uint32(before) &^ paletteSystemMenu
	paletteSetStyle.Call(hwnd, ^uintptr(15), uintptr(after))
	actual, _, _ := paletteGetStyle.Call(hwnd, ^uintptr(15))
	if uint32(actual) != after {
		return false
	}
	// FRAMECHANGED | NOACTIVATE | NOZORDER | NOMOVE | NOSIZE; no activation,
	// move, resize or z-order change to any existing window.
	changed, _, _ := paletteFrameChanged.Call(hwnd, 0, 0, 0, 0, 0, 0x37)
	return changed != 0
}
