//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"errors"
	"os"
	"syscall"
	"unsafe"
)

var localizedGetWindowsDirectory = kernel32.NewProc("GetWindowsDirectoryW")
var localizedLoadResourceLibrary = kernel32.NewProc("LoadLibraryExW")
var localizedFreeResourceLibrary = kernel32.NewProc("FreeLibrary")
var localizedLoadString = user32.NewProc("LoadStringW")

const localizedResourceFlags = 0x40 | 0x20 // DATAFILE_EXCLUSIVE | IMAGE_RESOURCE; no DLL initialization/imports.

func localizedWindowsDirectory() string {
	buffer := make([]uint16, 512)
	n, _, _ := localizedGetWindowsDirectory.Call(uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if n == 0 || n >= uintptr(len(buffer)) {
		return ""
	}
	return syscall.UTF16ToString(buffer[:n])
}

func readLocalizedResource(path string, id uint32) (string, error) {
	if id == 0 || id > 65535 || LocalAppPath(path) != nil {
		return "", errors.New("localized resource is not an allowed local file")
	}
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() || info.Size() <= 0 || info.Size() > 128*1024*1024 {
		return "", errors.New("localized resource exceeds module bounds")
	}
	encoded, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return "", err
	}
	module, _, callErr := localizedLoadResourceLibrary.Call(uintptr(unsafe.Pointer(encoded)), 0, localizedResourceFlags)
	if module == 0 {
		return "", callErr
	}
	defer localizedFreeResourceLibrary.Call(module)
	buffer := make([]uint16, 512)
	n, _, _ := localizedLoadString.Call(module, uintptr(id), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	if n == 0 || n >= uintptr(len(buffer)-1) {
		return "", errors.New("localized string missing or exceeds reader bound")
	}
	return syscall.UTF16ToString(buffer[:n]), nil
}
