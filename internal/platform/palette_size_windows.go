//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"fmt"
	"unsafe"
)

var paletteWindowRect = user32.NewProc("GetWindowRect")
var paletteClientRect = user32.NewProc("GetClientRect")
var paletteMonitor = user32.NewProc("MonitorFromWindow")
var paletteMonitorInfo = user32.NewProc("GetMonitorInfoW")
var paletteWindowDPI = user32.NewProc("GetDpiForWindow")

type paletteNativeRect struct{ Left, Top, Right, Bottom int32 }
type paletteMonitorData struct {
	Size          uint32
	Monitor, Work paletteNativeRect
	Flags         uint32
}

func (r paletteNativeRect) geometry() paletteRect {
	return paletteRect{int(r.Left), int(r.Top), int(r.Right), int(r.Bottom)}
}

// ResizePalette operates only on an owning UI thread's borderless top-level
// window. It never shows, activates or changes z-order. MyGo's real NativeHandle
// is used; normal WM_SIZE updates its existing drawing surface/search control.
func ResizePalette(hwnd uintptr, width, height, expanded int, center bool) (PaletteSize, error) {
	if !keyGuardOwnThread(hwnd) {
		return PaletteSize{}, fmt.Errorf("palette ownership/thread unavailable")
	}
	if paletteWindowDPI.Find() != nil {
		return PaletteSize{}, fmt.Errorf("window DPI API unavailable")
	}
	style, _, _ := paletteGetStyle.Call(hwnd, ^uintptr(15))
	if style&0x40000000 != 0 {
		return PaletteSize{}, fmt.Errorf("palette must be a top-level window")
	}
	var outer, client paletteNativeRect
	ok, _, _ := paletteWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&outer)))
	if ok == 0 {
		return PaletteSize{}, fmt.Errorf("palette bounds unavailable")
	}
	ok, _, _ = paletteClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	if ok == 0 {
		return PaletteSize{}, fmt.Errorf("palette client bounds unavailable")
	}
	frame := paletteFrame{int(outer.Right) - int(outer.Left) - int(client.Right), int(outer.Bottom) - int(outer.Top) - int(client.Bottom)}
	monitor, _, _ := paletteMonitor.Call(hwnd, 2)
	if monitor == 0 {
		return PaletteSize{}, fmt.Errorf("palette monitor unavailable")
	}
	info := paletteMonitorData{Size: uint32(unsafe.Sizeof(paletteMonitorData{}))}
	ok, _, _ = paletteMonitorInfo.Call(monitor, uintptr(unsafe.Pointer(&info)))
	if ok == 0 {
		return PaletteSize{}, fmt.Errorf("monitor work area unavailable")
	}
	dpi, _, _ := paletteWindowDPI.Call(hwnd)
	target, err := fitPaletteBounds(outer.geometry(), info.Work.geometry(), width, height, expanded, int(dpi), center, frame)
	if err != nil {
		return PaletteSize{}, err
	}
	if target != outer.geometry() {
		ok, _, _ = paletteFrameChanged.Call(hwnd, 0, uintptr(target.left), uintptr(target.top), uintptr(target.right-target.left), uintptr(target.bottom-target.top), 0x14) // NOZORDER | NOACTIVATE, never SHOWWINDOW.
		if ok == 0 {
			return PaletteSize{}, fmt.Errorf("palette resize failed")
		}
	}
	ok, _, _ = paletteWindowRect.Call(hwnd, uintptr(unsafe.Pointer(&outer)))
	if ok == 0 || outer.geometry() != target {
		return PaletteSize{}, fmt.Errorf("palette resize readback differs")
	}
	ok, _, _ = paletteClientRect.Call(hwnd, uintptr(unsafe.Pointer(&client)))
	if ok == 0 {
		return PaletteSize{}, fmt.Errorf("palette resized client unavailable")
	}
	return PaletteSize{float32(client.Right) * 96 / float32(dpi), float32(client.Bottom) * 96 / float32(dpi)}, nil
}
