//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"github.com/feiniao233/MyCast/internal/core"
	"syscall"
	"unsafe"
)

var getKnownFolderIDList = shell32.NewProc("SHGetKnownFolderIDList")
var isIDListParent = shell32.NewProc("ILIsParent")
var getIDListName = shell32.NewProc("SHGetNameFromIDList")

// Inspect cached link PIDL metadata without Resolve or filesystem target lookup.
// Prove the AppsFolder root and an immediate child before asking for its name.
func shortcutAppsFolderTarget(link *comObject) string {
	var target, root uintptr
	if comError("read shortcut PIDL", link.call(4, uintptr(unsafe.Pointer(&target)))) != nil || target == 0 {
		return ""
	}
	defer freeCOMMemory.Call(target)
	hr, _, _ := getKnownFolderIDList.Call(uintptr(unsafe.Pointer(&appsFolderID)), 0, 0, uintptr(unsafe.Pointer(&root)))
	if root != 0 {
		defer freeCOMMemory.Call(root)
	}
	if comError("read AppsFolder PIDL", hr) != nil || root == 0 {
		return ""
	}
	matches, _, _ := isIDListParent.Call(root, target, 1)
	if matches == 0 {
		return ""
	}
	var name *uint16
	hr, _, _ = getIDListName.Call(target, 0x80028000, uintptr(unsafe.Pointer(&name)))
	if name != nil {
		defer freeCOMMemory.Call(uintptr(unsafe.Pointer(name)))
	}
	if comError("read shortcut Shell identity", hr) != nil || name == nil {
		return ""
	}
	units := make([]uint16, 0, 2048)
	for i := 0; i < 2048; i++ {
		v := *(*uint16)(unsafe.Add(unsafe.Pointer(name), i*2))
		if v == 0 {
			id, err := core.AppsFolderChildIdentity(syscall.UTF16ToString(units))
			if err == nil {
				return id
			}
			return ""
		}
		units = append(units, v)
	}
	return ""
}
