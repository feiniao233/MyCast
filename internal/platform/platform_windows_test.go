//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"path/filepath"
	"testing"
	"unsafe"
)

func TestGUIThreadInfoABI(t *testing.T) {
	want := uintptr(48)
	if unsafe.Sizeof(uintptr(0)) == 8 {
		want = 72
	}
	if size := unsafe.Sizeof(guiThreadInfo{}); size != want {
		t.Fatalf("GUITHREADINFO size = %d, want %d", size, want)
	}
}

func TestDefaultAppRoots(t *testing.T) {
	t.Setenv("APPDATA", `C:\Users\Tester\AppData\Roaming`)
	t.Setenv("PROGRAMDATA", `C:\ProgramData`)
	t.Setenv("USERPROFILE", `C:\Users\Tester`)
	t.Setenv("PUBLIC", `C:\Users\Public`)
	roots := DefaultAppRoots()
	if len(roots) != 4 {
		t.Fatalf("got %d roots", len(roots))
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			t.Fatalf("relative app root: %s", root)
		}
	}
	t.Setenv("APPDATA", "")
	t.Setenv("PROGRAMDATA", "relative")
	t.Setenv("PUBLIC", `C:\Users\Tester`)
	if roots := DefaultAppRoots(); len(roots) != 1 {
		t.Fatalf("empty, relative, or duplicate roots retained: %v", roots)
	}
}
