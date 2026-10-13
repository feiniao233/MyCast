//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"errors"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

type applicationPresence uint8

const (
	applicationUnknown applicationPresence = iota
	applicationPresent
	applicationMissing
)

// Separate from LocalAppPath: missing needs a mounted local root and safe
// directory ancestors. Stop before every child of a missing/denied/reparse path.
func localApplicationPresence(path string) (applicationPresence, uint32) {
	return probeApplicationPresence(path, func(root string) uint32 {
		wide, err := syscall.UTF16PtrFromString(root)
		if err != nil {
			return 0
		}
		kind, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(wide)))
		return uint32(kind)
	}, func(path string) (uint32, error) {
		wide, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			return 0, err
		}
		attrs, _, err := getFileAttributes.Call(uintptr(unsafe.Pointer(wide)))
		if uint32(attrs) == 0xffffffff {
			return 0, err
		}
		return uint32(attrs), nil
	})
}

func probeApplicationPresence(path string, driveType func(string) uint32, attributes func(string) (uint32, error)) (applicationPresence, uint32) {
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) {
		return applicationUnknown, 0
	}
	path = filepath.Clean(path)
	volume := filepath.VolumeName(path)
	if len(volume) != 2 || volume[1] != ':' {
		return applicationUnknown, 0
	}
	root := volume + `\`
	kind := driveType(root)
	switch kind {
	case 2, 3, 5, 6:
	default:
		return applicationUnknown, 0
	}
	attrs, err := attributes(root)
	if err != nil || attrs&0x400 != 0 || attrs&0x10 == 0 {
		return applicationUnknown, 0
	}
	parts := strings.Split(strings.TrimPrefix(path, root), `\`)
	prefix := root
	for _, part := range parts {
		if part == "" {
			continue
		}
		if strings.Contains(part, ":") || attrs&0x10 == 0 {
			return applicationUnknown, 0
		}
		prefix = filepath.Join(prefix, part)
		attrs, err = attributes(prefix)
		if err != nil {
			// Offline removable/optical/unknown states cannot prove deletion.
			if kind == 3 && (errors.Is(err, syscall.ERROR_FILE_NOT_FOUND) || errors.Is(err, syscall.ERROR_PATH_NOT_FOUND)) {
				return applicationMissing, 0
			}
			return applicationUnknown, 0
		}
		if attrs&0x400 != 0 {
			return applicationUnknown, 0
		}
	}
	return applicationPresent, attrs
}
