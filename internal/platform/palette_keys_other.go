//go:build !windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

type PaletteKeyGuard struct{}

func InstallPaletteKeyGuard(uintptr) (*PaletteKeyGuard, error) { return nil, ErrUnsupported }
func (*PaletteKeyGuard) SetComposing(bool)                     {}
func (*PaletteKeyGuard) SuppressedCount() uint64               { return 0 }
func (*PaletteKeyGuard) Close() bool                           { return true }
