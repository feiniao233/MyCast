//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

func TestPaletteAppearanceRequiresAcceptedReadback(t *testing.T) {
	for _, scenario := range []string{"accepted", "set-failed", "get-failed", "mica-readback", "corner-unavailable"} {
		t.Run(scenario, func(t *testing.T) {
			values := map[uint32]uint32{}
			appearance := configurePaletteAppearance(func(attribute, value uint32) bool {
				if scenario == "set-failed" || scenario == "corner-unavailable" && attribute == 33 {
					return false
				}
				values[attribute] = value
				return true
			}, func(attribute uint32) (uint32, bool) {
				if scenario == "get-failed" {
					return 0, false
				}
				if scenario == "mica-readback" && attribute == 38 {
					return 2, true
				}
				return values[attribute], true
			})
			wantAcrylic := scenario == "accepted" || scenario == "corner-unavailable"
			wantRounded := scenario == "accepted" || scenario == "mica-readback"
			if appearance.Acrylic != wantAcrylic || appearance.RoundingAccepted != wantRounded {
				t.Fatalf("unexpected appearance: %+v", appearance)
			}
			if scenario != "set-failed" && values[38] != 3 {
				t.Fatal("must request Desktop Acrylic, never Auto or Mica")
			}
		})
	}
	if got := ConfigurePaletteWindow(0); got != (PaletteAppearance{}) {
		t.Fatal("zero HWND accepted")
	}
}

func TestPaletteAppearanceNativeAttributeABI(t *testing.T) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	class, _ := syscall.UTF16PtrFromString("STATIC")
	title, _ := syscall.UTF16PtrFromString("MyCast synthetic DWM fixture")
	// An invisible, task-owned window. Never activate, read or send input to
	// any existing window. No clipboard or user application data is involved.
	hwnd, _, err := createWindowEx.Call(0x00200000, uintptr(unsafe.Pointer(class)), uintptr(unsafe.Pointer(title)), 0x00cf0000, 0, 0, 160, 100, 0, 0, 0, 0)
	if hwnd == 0 {
		t.Fatalf("synthetic window creation: %v", err)
	}
	defer destroyWindow.Call(hwnd)
	if paletteDWMSet.Find() != nil || paletteDWMGet.Find() != nil {
		t.Skip("DWM attributes unavailable; opaque fallback")
	}
	reference := func(attribute, value uint32) bool {
		r, _, _ := paletteDWMSet.Call(hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&value)), 4)
		if int32(r) < 0 {
			return false
		}
		var actual uint32
		r, _, _ = paletteDWMGet.Call(hwnd, uintptr(attribute), uintptr(unsafe.Pointer(&actual)), 4)
		return int32(r) >= 0 && actual == value
	}
	want := PaletteAppearance{Acrylic: reference(38, 3), RoundingAccepted: reference(33, 2)}
	got := ConfigurePaletteWindow(hwnd)
	if got != want {
		t.Fatalf("native attribute ABI: got %+v want %+v", got, want)
	}
	t.Logf("synthetic HWND attribute acceptance: %+v; visible blur/rounding not tested", got)
}
