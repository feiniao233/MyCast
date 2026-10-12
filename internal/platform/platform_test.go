// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestClipboardTextRoundTrip(t *testing.T) {
	for _, text := range []string{"", "plain text", "你好 🌍\r\nsecond line", strings.Repeat("a", MaxClipboardBytes/2-1)} {
		units, err := encodeClipboardText(text)
		if err != nil {
			t.Fatal(err)
		}
		got, err := decodeClipboardText(units)
		if err != nil || got != text {
			t.Fatalf("round trip failed: error=%v", err)
		}
	}
}

func TestClipboardTextRejectsInvalidAndOversize(t *testing.T) {
	for _, tc := range []struct {
		name string
		text string
		want error
	}{
		{"nul", "a\x00b", ErrInvalidClipboardText},
		{"invalid UTF-8", "\xff", ErrInvalidClipboardText},
		{"UTF-16 limit", strings.Repeat("a", MaxClipboardBytes/2), ErrClipboardTooLarge},
		{"UTF-8 limit", strings.Repeat("中", MaxClipboardBytes/3+1), ErrClipboardTooLarge},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := encodeClipboardText(tc.text); !errors.Is(err, tc.want) {
				t.Fatalf("got %v; want %v", err, tc.want)
			}
		})
	}
	for _, units := range [][]uint16{nil, {'a'}, makeNonzeroUnits(MaxClipboardBytes/2 + 1)} {
		if _, err := decodeClipboardText(units); err == nil {
			t.Fatal("accepted unterminated clipboard storage")
		}
	}
	units := makeNonzeroUnits(MaxClipboardBytes / 2)
	units = append(units, 0)
	if _, err := decodeClipboardText(units); err == nil {
		t.Fatal("accepted terminator beyond read limit")
	}
	units = make([]uint16, MaxClipboardBytes/3+2)
	for i := range units[:len(units)-1] {
		units[i] = '中'
	}
	if _, err := decodeClipboardText(units); !errors.Is(err, ErrClipboardTooLarge) {
		t.Fatalf("UTF-8 output limit: got %v", err)
	}
}

func makeNonzeroUnits(length int) []uint16 {
	units := make([]uint16, length)
	for i := range units {
		units[i] = 'x'
	}
	return units
}

func TestValidateLaunchPath(t *testing.T) {
	dir := t.TempDir()
	for _, name := range []string{"App.EXE", "My App.lnk"} {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte("test fixture, not executable"), 0600); err != nil {
			t.Fatal(err)
		}
		got, err := validateLaunchPath(path)
		if err != nil || got != path {
			t.Fatalf("valid path failed: got %q, %v", got, err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "folder.exe"), 0700); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{
		"", "app.exe", "https://example.com/app.exe", "cmd.exe /c something",
		filepath.Join(dir, "missing.exe"), filepath.Join(dir, "script.bat"),
		filepath.Join(dir, "folder.exe"), filepath.Join(dir, "file\x00.exe"),
	} {
		if _, err := validateLaunchPath(path); err == nil {
			t.Fatalf("accepted invalid path %q", path)
		}
	}
}

type fakePaste struct {
	writtenSequence  uint32
	foreground       uintptr
	windows          map[uintptr]windowIdentity
	keys             bool
	activationDenied bool
	onCopy           func()
	copyErr          error
	sequence         uint32
	sendErr          error
	calls            []string
}

func newFakePaste() *fakePaste {
	return &fakePaste{
		foreground:      1,
		writtenSequence: 5,
		sequence:        5,
		windows: map[uintptr]windowIdentity{
			1: {processID: 10, threadID: 11}, // Target.
			2: {processID: 20, threadID: 21}, // Our launcher.
			3: {processID: 30, threadID: 31}, // Unrelated app.
		},
	}
}

func (f *fakePaste) actions() pasteActions {
	return pasteActions{
		currentProcess: 20,
		identity:       func(hwnd uintptr) (windowIdentity, bool) { id, ok := f.windows[hwnd]; return id, ok },
		foreground:     func() uintptr { return f.foreground },
		activate: func(hwnd uintptr) {
			f.calls = append(f.calls, "activate")
			if !f.activationDenied {
				f.foreground = hwnd
			}
		},
		keysHeld: func() bool { return f.keys },
		writeClipboard: func(string) (uint32, error) {
			f.calls = append(f.calls, "copy")
			if f.onCopy != nil {
				f.onCopy()
			}
			return f.writtenSequence, f.copyErr
		},
		clipboardSequence: func() uint32 { return f.sequence },
		sendPaste:         func() error { f.calls = append(f.calls, "paste"); return f.sendErr },
	}
}

func TestPasteAllowsOnlyImmediateVerifiedTarget(t *testing.T) {
	for _, initial := range []uintptr{0, 1, 2} {
		f := newFakePaste()
		f.foreground = initial
		if err := pasteSafely(1, "text", f.actions()); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(f.calls, []string{"activate", "copy", "paste"}) {
			t.Fatalf("unexpected actions: %v", f.calls)
		}
	}
}

func TestPasteFailsClosed(t *testing.T) {
	copyErr := errors.New("clipboard unavailable")
	sendErr := errors.New("delivery uncertain")
	for _, tc := range []struct {
		name   string
		target uintptr
		setup  func(*fakePaste)
		want   error
		calls  []string
	}{
		{"zero target", 0, func(*fakePaste) {}, ErrTargetUnavailable, nil},
		{"missing target", 99, func(*fakePaste) {}, ErrTargetUnavailable, nil},
		{"other app already focused", 1, func(f *fakePaste) { f.foreground = 3 }, ErrFocusChanged, nil},
		{"unknown window focused", 1, func(f *fakePaste) { f.foreground = 99 }, ErrFocusChanged, nil},
		{"held keys", 1, func(f *fakePaste) { f.keys = true }, ErrModifiersHeld, nil},
		{"activation denied", 1, func(f *fakePaste) { f.foreground = 2; f.activationDenied = true }, ErrFocusChanged, []string{"activate"}},
		{"clipboard busy", 1, func(f *fakePaste) { f.copyErr = copyErr }, copyErr, []string{"activate", "copy"}},
		{"focus changed during copy", 1, func(f *fakePaste) { f.onCopy = func() { f.foreground = 3 } }, ErrFocusChanged, []string{"activate", "copy"}},
		{"target destroyed during copy", 1, func(f *fakePaste) { f.onCopy = func() { delete(f.windows, 1) } }, ErrFocusChanged, []string{"activate", "copy"}},
		{"target handle reused", 1, func(f *fakePaste) { f.onCopy = func() { f.windows[1] = windowIdentity{30, 31} } }, ErrFocusChanged, []string{"activate", "copy"}},
		{"keys pressed during copy", 1, func(f *fakePaste) { f.onCopy = func() { f.keys = true } }, ErrModifiersHeld, []string{"activate", "copy"}},
		{"written sequence unavailable", 1, func(f *fakePaste) { f.writtenSequence = 0 }, ErrClipboardChanged, []string{"activate", "copy"}},
		{"clipboard replaced", 1, func(f *fakePaste) { f.onCopy = func() { f.sequence++ } }, ErrClipboardChanged, []string{"activate", "copy"}},
		{"clipboard sequence unavailable", 1, func(f *fakePaste) { f.onCopy = func() { f.sequence = 0 } }, ErrClipboardChanged, []string{"activate", "copy"}},
		{"delivery failure not retried", 1, func(f *fakePaste) { f.sendErr = sendErr }, sendErr, []string{"activate", "copy", "paste"}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFakePaste()
			tc.setup(f)
			if err := pasteSafely(tc.target, "text", f.actions()); !errors.Is(err, tc.want) {
				t.Fatalf("got %v, want %v", err, tc.want)
			}
			if !reflect.DeepEqual(f.calls, tc.calls) {
				t.Fatalf("got actions %v, want %v", f.calls, tc.calls)
			}
		})
	}
}
