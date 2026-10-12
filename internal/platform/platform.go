// SPDX-License-Identifier: AGPL-3.0-or-later

// Package platform provides the small Windows integration surface used by MyCast.
package platform

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf16"
	"unicode/utf8"
)

// MaxClipboardBytes bounds both UTF-16 storage inspected and returned UTF-8 text.
const MaxClipboardBytes = 64 * 1024

var (
	ErrUnsupported                = errors.New("this operation is only available on Windows")
	ErrClipboardFormatUnavailable = errors.New("the clipboard does not contain Unicode text")
	ErrClipboardChanged           = errors.New("clipboard changed; paste was cancelled")
	ErrClipboardTooLarge          = errors.New("clipboard text exceeds the 64 KiB limit")
	ErrInvalidClipboardText       = errors.New("clipboard text must be valid UTF-8 without NUL characters")
	ErrTargetUnavailable          = errors.New("the original window is no longer available")
	ErrFocusChanged               = errors.New("window focus changed; paste was cancelled")
	ErrModifiersHeld              = errors.New("release Ctrl, Alt, Shift, Windows, and V before pasting")
)

func encodeClipboardText(text string) ([]uint16, error) {
	if len(text) > MaxClipboardBytes {
		return nil, ErrClipboardTooLarge
	}
	if !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
		return nil, ErrInvalidClipboardText
	}
	units := utf16.Encode([]rune(text))
	if (len(units)+1)*2 > MaxClipboardBytes {
		return nil, ErrClipboardTooLarge
	}
	return append(units, 0), nil
}

// decodeClipboardText only accepts terminated text within the bounded view.
func decodeClipboardText(units []uint16) (string, error) {
	if len(units) > MaxClipboardBytes/2 {
		units = units[:MaxClipboardBytes/2]
	}
	for i, unit := range units {
		if unit == 0 {
			text := string(utf16.Decode(units[:i]))
			if len(text) > MaxClipboardBytes {
				return "", ErrClipboardTooLarge
			}
			return text, nil
		}
	}
	return "", errors.New("clipboard text is unterminated or exceeds the 64 KiB limit")
}

func validateLaunchPath(path string) (string, error) {
	if strings.ContainsRune(path, 0) || !filepath.IsAbs(path) {
		return "", errors.New("application path must be an absolute file path")
	}
	path = filepath.Clean(path)
	switch strings.ToLower(filepath.Ext(path)) {
	case ".exe", ".lnk":
	default:
		return "", errors.New("only .exe applications and .lnk shortcuts can be launched")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", fmt.Errorf("application is unavailable: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", errors.New("application path must name a regular file")
	}
	return path, nil
}

type windowIdentity struct {
	processID uint32
	threadID  uint32
}

// pasteActions separates the fail-closed decision flow from Win32 calls so focus
// changes and failures can be covered by tests on any development platform.
type pasteActions struct {
	currentProcess    uint32
	identity          func(uintptr) (windowIdentity, bool)
	foreground        func() uintptr
	activate          func(uintptr)
	keysHeld          func() bool
	writeClipboard    func(string) (uint32, error)
	clipboardSequence func() uint32
	sendPaste         func() error
}

func pasteSafely(target uintptr, text string, actions pasteActions) error {
	if target == 0 {
		return ErrTargetUnavailable
	}
	id, ok := actions.identity(target)
	if !ok {
		return ErrTargetUnavailable
	}
	foreground := actions.foreground()
	if foreground != 0 && foreground != target {
		current, ok := actions.identity(foreground)
		if !ok || current.processID != actions.currentProcess {
			return ErrFocusChanged
		}
	}
	if actions.keysHeld() {
		return ErrModifiersHeld
	}
	// Never retry activation or wait for focus: the user may be doing something
	// else by then. A refusal by Windows is a cancelled paste, not a reason to
	// attach another process's input thread or synthesize an activation shortcut.
	actions.activate(target)
	valid := func() bool {
		current, ok := actions.identity(target)
		return ok && current == id && actions.foreground() == target
	}
	if !valid() {
		return ErrFocusChanged
	}
	sequence, err := actions.writeClipboard(text)
	if err != nil {
		return err
	}
	if actions.keysHeld() {
		return ErrModifiersHeld
	}
	if !valid() {
		return ErrFocusChanged
	}
	if sequence == 0 || actions.clipboardSequence() != sequence {
		return ErrClipboardChanged
	}
	return actions.sendPaste()
}
