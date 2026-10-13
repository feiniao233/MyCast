//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"context"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"image"
)

func ForegroundWindow() uintptr                       { return 0 }
func ClipboardSequence() uint32                       { return 0 }
func DefaultAppRoots() []string                       { return nil }
func PasteToWindow(target uintptr, text string) error { return ErrUnsupported }
func ReadClipboardText() (string, error)              { return "", ErrUnsupported }
func WriteClipboardText(text string) error            { return ErrUnsupported }
func Launch(path string) error                        { return ErrUnsupported }
func OpenHTTPURL(value string) error                  { return ErrUnsupported }
func LaunchShellApplication(string) error             { return ErrUnsupported }
func ShowApplicationLocation(string) error            { return ErrUnsupported }
func NewStartupRegistration(string) (*settings.StartupRegistration, error) {
	return nil, ErrUnsupported
}
func AppsFolderApplications(context.Context, ...map[string]image.Image) ([]core.Application, map[string]image.Image, []error) {
	return nil, nil, []error{ErrUnsupported}
}
func WatchApplicationSources(context.Context, []string, bool, func(bool, bool, []string)) {}
