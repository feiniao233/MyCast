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
	paths := []string{`C:\Synthetic\UserStart`, `C:\Synthetic\CommonStart`, `C:\Synthetic\RedirectedDesktop`, `C:\Synthetic\PublicDesktop`}
	resolve := func(id *comGUID) string {
		for i, source := range defaultSourceIDs {
			if *id == source.id {
				return paths[i]
			}
		}
		return ""
	}
	roots, shallow := defaultApplicationSources(resolve)
	if len(roots) != 4 {
		t.Fatalf("got %d roots", len(roots))
	}
	for _, root := range roots {
		if !filepath.IsAbs(root) {
			t.Fatalf("relative app root: %s", root)
		}
	}
	if len(shallow) != 2 || shallow[0] != paths[2] {
		t.Fatal("redirected desktop was not shallow")
	}
	paths = []string{"", "relative", `C:\Synthetic\Shared`, `C:\Synthetic\Shared`}
	if roots, _ := defaultApplicationSources(resolve); len(roots) != 1 {
		t.Fatal("empty/relative/duplicate default roots retained")
	}
}

func TestNativeKnownFolderPathOutputOwnership(t *testing.T) {
	for i := 0; i < 16; i++ {
		path := defaultKnownFolderPath(&defaultSourceIDs[2].id)
		if path == "" || !filepath.IsAbs(path) {
			t.Fatal("native Known Folder path lookup failed")
		}
	}
}
