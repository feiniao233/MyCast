//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"fmt"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var getFileAttributes = kernel32.NewProc("GetFileAttributesW")

// LocalAppPath checks volume metadata before any directory/file access. Every
// ancestor is then checked in order, rejecting all reparse points before a
// deeper lookup could traverse a junction or hydrate a remote placeholder.
func LocalAppPath(path string) error {
	return checkLocalAppPath(path, func(root string) uint32 {
		value, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			return 0
		}
		kind, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(value)))
		return uint32(kind)
	}, func(path string) (uint32, error) {
		value, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			return 0, err
		}
		attrs, _, callErr := getFileAttributes.Call(uintptr(unsafe.Pointer(value)))
		if uint32(attrs) == 0xffffffff {
			return 0, callErr
		}
		return uint32(attrs), nil
	})
}

func checkLocalAppPath(path string, driveType func(string) uint32, attributes func(string) (uint32, error)) error {
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) {
		return fmt.Errorf("application path must be an absolute local drive path")
	}
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' {
		return fmt.Errorf("application volume is unsupported")
	}
	switch driveType(volume + `\`) {
	case 2, 3, 5, 6:
	default:
		return fmt.Errorf("network or unconfirmed application volume was refused")
	}
	prefix := volume + `\`
	parts := strings.Split(strings.TrimPrefix(path, prefix), `\`)
	for _, part := range parts {
		if part == "" {
			continue
		}
		prefix = filepath.Join(prefix, part)
		attrs, err := attributes(prefix)
		if err != nil {
			return fmt.Errorf("application path metadata is unavailable")
		}
		if attrs&0x400 != 0 {
			return fmt.Errorf("application reparse path was refused")
		}
	}
	return nil
}
