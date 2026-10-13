//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

var getKnownFolderPath = shell32.NewProc("SHGetKnownFolderPath")
var defaultSourceIDs = []struct {
	id      comGUID
	shallow bool
}{
	{comGUID{0x625b53c3, 0xab48, 0x4ec1, [8]byte{0xba, 0x1f, 0xa1, 0xef, 0x41, 0x46, 0xfc, 0x19}}, false},
	{comGUID{0xa4115719, 0xd62e, 0x491d, [8]byte{0xaa, 0x7c, 0xe7, 0x4b, 0x8b, 0xe3, 0xb0, 0x67}}, false},
	{comGUID{0xb4bfcc3a, 0xdb2c, 0x424c, [8]byte{0xb0, 0x29, 0x7f, 0xe9, 0x9a, 0x87, 0xc6, 0x41}}, true},
	{comGUID{0xc4aa340d, 0xf20f, 0x4863, [8]byte{0xaf, 0xef, 0xf8, 0x7e, 0xf2, 0xe6, 0xba, 0x25}}, true},
}

// DONT_VERIFY reads configured Known Folder locations without filesystem
// verification/creation. Index/watch adapters still reject network/reparse paths.
func defaultKnownFolderPath(id *comGUID) string {
	var value *uint16
	hr, _, _ := getKnownFolderPath.Call(uintptr(unsafe.Pointer(id)), 0x4000, 0, uintptr(unsafe.Pointer(&value)))
	if value != nil {
		defer freeCOMMemory.Call(uintptr(unsafe.Pointer(value)))
	}
	if comError("read default application folder", hr) != nil || value == nil {
		return ""
	}
	units := make([]uint16, 0, 4096)
	for i := 0; i < 4096; i++ {
		v := *(*uint16)(unsafe.Add(unsafe.Pointer(value), i*2))
		if v == 0 {
			return syscall.UTF16ToString(units)
		}
		units = append(units, v)
	}
	return ""
}
func defaultApplicationSources(resolve func(*comGUID) string) (roots, shallow []string) {
	seen := map[string]int{}
	recursive := map[string]bool{}
	for _, source := range defaultSourceIDs {
		path := resolve(&source.id)
		if path == "" || !filepath.IsAbs(path) {
			continue
		}
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if _, exists := seen[key]; !exists {
			seen[key] = len(roots)
			roots = append(roots, path)
		}
		if !source.shallow {
			recursive[key] = true
		}
	}
	for _, source := range defaultSourceIDs {
		if !source.shallow {
			continue
		}
		path := resolve(&source.id)
		if path == "" {
			continue
		}
		path = filepath.Clean(path)
		key := strings.ToLower(path)
		if _, exists := seen[key]; exists && !recursive[key] {
			shallow = append(shallow, path)
			delete(seen, key)
		}
	}
	return
}
func DefaultAppRoots() []string {
	roots, _ := defaultApplicationSources(defaultKnownFolderPath)
	return roots
}
func DefaultDesktopRoots() []string {
	_, shallow := defaultApplicationSources(defaultKnownFolderPath)
	return shallow
}
