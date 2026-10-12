//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

func ForegroundWindow() uintptr                       { return 0 }
func ClipboardSequence() uint32                       { return 0 }
func DefaultAppRoots() []string                       { return nil }
func PasteToWindow(target uintptr, text string) error { return ErrUnsupported }
func ReadClipboardText() (string, error)              { return "", ErrUnsupported }
func WriteClipboardText(text string) error            { return ErrUnsupported }
func Launch(path string) error                        { return ErrUnsupported }
func OpenHTTPURL(value string) error                  { return ErrUnsupported }
