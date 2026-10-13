//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"syscall"
	"unsafe"
)

type PaletteAppearance struct{ Acrylic, RoundingAccepted bool }

var paletteDWM = syscall.NewLazyDLL("dwmapi.dll")
var paletteDWMSet = paletteDWM.NewProc("DwmSetWindowAttribute")
var paletteDWMGet = paletteDWM.NewProc("DwmGetWindowAttribute")

const (
	paletteCornerAttribute   = 33 // DWMWA_WINDOW_CORNER_PREFERENCE
	paletteBackdropAttribute = 38 // DWMWA_SYSTEMBACKDROP_TYPE
	paletteRound             = 2  // DWMWCP_ROUND (the HWND, not a renderer cutout)
	paletteAcrylic           = 3  // DWMSBT_TRANSIENTWINDOW / Desktop Acrylic
)

// ConfigurePaletteWindow affects only the app's own newly created HWND. MyGo
// creates the DirectComposition surface and extends its frame for Acrylic.
// Check HRESULT and readback instead of treating the requested material string
// as acceptance. Success still does not prove visible blur: OS policies,
// high contrast, power state and remote sessions affect presentation.
func ConfigurePaletteWindow(hwnd uintptr) PaletteAppearance {
	if hwnd == 0 || paletteDWMSet.Find() != nil || paletteDWMGet.Find() != nil {
		return PaletteAppearance{}
	}
	return configurePaletteAppearance(func(attribute, value uint32) bool {
		r, _, _ := paletteDWMSet.Call(hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&value)), 4)
		return int32(r) >= 0
	}, func(attribute uint32) (uint32, bool) {
		var value uint32
		r, _, _ := paletteDWMGet.Call(hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&value)), 4)
		return value, int32(r) >= 0
	})
}

func configurePaletteAppearance(set func(uint32, uint32) bool, get func(uint32) (uint32, bool)) PaletteAppearance {
	accepted := func(attribute, value uint32) bool {
		if !set(attribute, value) {
			return false
		}
		actual, ok := get(attribute)
		return ok && actual == value
	}
	return PaletteAppearance{
		RoundingAccepted: accepted(paletteCornerAttribute, paletteRound),
		Acrylic:          accepted(paletteBackdropAttribute, paletteAcrylic),
	}
}
