//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"image"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

var (
	iconFileInfo       = shell32.NewProc("SHGetFileInfoW")
	destroyIcon        = user32.NewProc("DestroyIcon")
	drawIcon           = user32.NewProc("DrawIconEx")
	gdi32              = syscall.NewLazyDLL("gdi32.dll")
	createCompatibleDC = gdi32.NewProc("CreateCompatibleDC")
	deleteDC           = gdi32.NewProc("DeleteDC")
	createDIBSection   = gdi32.NewProc("CreateDIBSection")
	selectGDIObject    = gdi32.NewProc("SelectObject")
	deleteGDIObject    = gdi32.NewProc("DeleteObject")
	gdiFlush           = gdi32.NewProc("GdiFlush")
	getDriveType       = kernel32.NewProc("GetDriveTypeW")
)

type shellIconInfo struct {
	Icon        uintptr
	Index       int32
	Attributes  uint32
	DisplayName [260]uint16
	TypeName    [80]uint16
}
type dibHeader struct {
	Size                   uint32
	Width, Height          int32
	Planes, Bits           uint16
	Compression, ImageSize uint32
	XPels, YPels           int32
	Used, Important        uint32
}

// ApplicationIcon reads the icon of a local executable, without opening a
// shortcut target or starting an application. GDI draws only the HICON into a
// private memory bitmap; this does not capture or access the desktop.
func ApplicationIcon(path string) image.Image {
	if !filepath.IsAbs(path) || !strings.EqualFold(filepath.Ext(path), ".exe") || strings.HasPrefix(path, `\\`) {
		return nil
	}
	if err := LocalAppPath(path); err != nil {
		return nil
	}
	volume, err := syscall.UTF16PtrFromString(filepath.VolumeName(path) + `\`)
	if err != nil {
		return nil
	}
	drive, _, _ := getDriveType.Call(uintptr(unsafe.Pointer(volume)))
	if drive != 2 && drive != 3 {
		return nil
	} // Never visit network drives.
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if comError("initialize icon reader", hr) != nil {
		return nil
	}
	defer coUninitialize.Call()
	encoded, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return nil
	}
	var info shellIconInfo
	ok, _, _ := iconFileInfo.Call(uintptr(unsafe.Pointer(encoded)), 0, uintptr(unsafe.Pointer(&info)), unsafe.Sizeof(info), 0x100)
	runtime.KeepAlive(encoded)
	if ok == 0 || info.Icon == 0 {
		return nil
	}
	defer destroyIcon.Call(info.Icon)
	dc, _, _ := createCompatibleDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer deleteDC.Call(dc)
	header := dibHeader{Size: 40, Width: 32, Height: -32, Planes: 1, Bits: 32}
	var pixels unsafe.Pointer
	bitmap, _, _ := createDIBSection.Call(dc, uintptr(unsafe.Pointer(&header)), 0, uintptr(unsafe.Pointer(&pixels)), 0, 0)
	if bitmap == 0 || pixels == nil {
		return nil
	}
	defer deleteGDIObject.Call(bitmap)
	previous, _, _ := selectGDIObject.Call(dc, bitmap)
	defer selectGDIObject.Call(dc, previous)
	data := unsafe.Slice((*byte)(pixels), 32*32*4)
	clear(data)
	ok, _, _ = drawIcon.Call(dc, 0, 0, info.Icon, 32, 32, 0, 0, 3)
	if ok == 0 {
		return nil
	}
	gdiFlush.Call()
	result := image.NewRGBA(image.Rect(0, 0, 32, 32))
	visible := false
	for i := 0; i < len(data); i += 4 {
		result.Pix[i], result.Pix[i+1], result.Pix[i+2], result.Pix[i+3] = data[i+2], data[i+1], data[i], data[i+3]
		visible = visible || data[i+3] != 0
	}
	if !visible {
		return nil
	} // Older mask-only icons use the UI fallback.
	return result
}
