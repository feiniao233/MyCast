//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"runtime"
	"syscall"
	"unsafe"
)

var parseShellDisplayName = shell32.NewProc("SHParseDisplayName")
var openFolderAndSelect = shell32.NewProc("SHOpenFolderAndSelectItems")

// ShowApplicationLocation selects the original safe local source, never a
// guessed UWP path or shortcut target. It does not modify/delete that source.
func ShowApplicationLocation(path string) error {
	if err := LocalAppPath(path); err != nil {
		return err
	}
	value, err := validateLaunchPath(path)
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("initialize source location", hr); err != nil {
		return err
	}
	defer coUninitialize.Call()
	wide, _ := syscall.UTF16PtrFromString(value)
	var pidl uintptr
	hr, _, _ = parseShellDisplayName.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&pidl)), 0, 0)
	if pidl != 0 {
		defer freeCOMMemory.Call(pidl)
	}
	if err := comError("parse source location", hr); err != nil {
		return err
	}
	if pidl == 0 {
		return syscall.EINVAL
	}
	hr, _, _ = openFolderAndSelect.Call(pidl, 0, 0, 0)
	runtime.KeepAlive(wide)
	return comError("show source location", hr)
}
