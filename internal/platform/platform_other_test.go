//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import "testing"

func TestNonWindowsStubs(t *testing.T) {
	if ForegroundWindow() != 0 || ClipboardSequence() != 0 || len(DefaultAppRoots()) != 0 {
		t.Fatal("non-Windows stubs returned native state")
	}
	if _, err := ReadClipboardText(); err != ErrUnsupported {
		t.Fatalf("clipboard read: %v", err)
	}
	for _, err := range []error{PasteToWindow(1, "text"), WriteClipboardText("text"), Launch("app.exe")} {
		if err != ErrUnsupported {
			t.Fatalf("unsupported operation: %v", err)
		}
	}
}
