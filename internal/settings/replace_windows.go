//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"
)

var moveFileEx = syscall.NewLazyDLL("kernel32.dll").NewProc("MoveFileExW")

// Same-volume move only: no copy/delete or reboot fallback. First creation
// omits REPLACE_EXISTING, refusing a destination that appeared after the check.
// This does not promise cross-filesystem atomicity or crash-proof durability.
func replaceFile(from, to string, exists bool) error {
	encode := func(path string) (*uint16, error) {
		path, err := filepath.Abs(path)
		if err != nil {
			return nil, err
		}
		if len(path) > 240 {
			path = `\\?\` + path
		}
		return syscall.UTF16PtrFromString(path)
	}
	source, err := encode(from)
	if err != nil {
		return err
	}
	target, err := encode(to)
	if err != nil {
		return err
	}
	flags := uintptr(8)
	if exists {
		flags |= 1
	}
	ok, _, callErr := moveFileEx.Call(uintptr(unsafe.Pointer(source)), uintptr(unsafe.Pointer(target)), flags)
	if ok == 0 {
		return fmt.Errorf("Windows settings move failed: %v", callErr)
	}
	return nil
}
