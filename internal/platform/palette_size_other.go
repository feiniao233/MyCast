//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

func ResizePalette(uintptr, int, int, int, bool) (PaletteSize, error) {
	return PaletteSize{}, ErrUnsupported
}
