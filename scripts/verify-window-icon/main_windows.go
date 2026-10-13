//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
// Dedicated hidden MyGo window fixture: no tray, app scan, clipboard or input.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/feiniao233/MyCast/internal/branding"
	"github.com/feiniao233/MyCast/internal/iconcheck"
)

func main() {
	exe := flag.String("exe", "dist/MyCast.exe", "this build's verified EXE resource fixture")
	flag.Parse()
	user := syscall.NewLazyDLL("user32.dll")
	gdi := syscall.NewLazyDLL("gdi32.dll")
	kernel := syscall.NewLazyDLL("kernel32.dll")
	send := user.NewProc("SendMessageW")
	info := user.NewProc("GetIconInfo")
	object := gdi.NewProc("GetObjectW")
	del := gdi.NewProc("DeleteObject")
	resources := user.NewProc("GetGuiResources")
	process, _, _ := kernel.NewProc("GetCurrentProcess").Call()
	inspect := func(handle uintptr) ([2]int32, error) {
		var ii struct {
			Icon        int32
			X, Y        uint32
			Mask, Color uintptr
		}
		ok, _, _ := info.Call(handle, uintptr(unsafe.Pointer(&ii)))
		if handle == 0 || ok == 0 {
			return [2]int32{}, fmt.Errorf("invalid HICON")
		}
		defer func() {
			if ii.Mask != 0 {
				del.Call(ii.Mask)
			}
			if ii.Color != 0 {
				del.Call(ii.Color)
			}
		}()
		var bitmap struct {
			Type, Width, Height, WidthBytes int32
			Planes, Bits                    uint16
			Pixels                          uintptr
		}
		read, _, _ := object.Call(ii.Color, unsafe.Sizeof(bitmap), uintptr(unsafe.Pointer(&bitmap)))
		if read == 0 || bitmap.Width <= 0 || bitmap.Height != bitmap.Width {
			return [2]int32{}, fmt.Errorf("invalid icon bitmap")
		}
		return [2]int32{bitmap.Width, bitmap.Height}, nil
	}
	// Read-only image/data mapping: never execute the EXE or run DLL initialization.
	verifyResources := func() error {
		canonical, err := os.ReadFile("assets/brand/mycast.ico")
		if err != nil {
			return err
		}
		path, err := filepath.Abs(*exe)
		if err != nil {
			return err
		}
		if err := iconcheck.Executable(path, canonical); err != nil {
			return err
		}
		wide, err := syscall.UTF16PtrFromString(path)
		if err != nil {
			return err
		}
		module, _, callErr := kernel.NewProc("LoadLibraryExW").Call(uintptr(unsafe.Pointer(wide)), 0, 0x60)
		if module == 0 {
			return fmt.Errorf("data/image resource mapping: %w", callErr)
		}
		defer kernel.NewProc("FreeLibrary").Call(module)
		load := user.NewProc("LoadImageW")
		destroy := user.NewProc("DestroyIcon")
		for _, n := range iconcheck.Sizes {
			handle, _, callErr := load.Call(module, 1, 1, uintptr(n), uintptr(n), 0) // owned IMAGE_ICON; no LR_SHARED
			if handle == 0 {
				return fmt.Errorf("load PE icon %d: %w", n, callErr)
			}
			dim, err := inspect(handle)
			destroy.Call(handle)
			if err != nil || dim != [2]int32{int32(n), int32(n)} {
				return fmt.Errorf("loaded PE icon %d: size=%v error=%v", n, dim, err)
			}
		}
		return nil
	}
	if err := verifyResources(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var failure error
	var dimensions [2][2]int32
	var baseline, after uintptr
	mygo.App.SetName("MyCast dedicated icon ABI fixture")
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "MyCast dedicated icon ABI fixture", Width: 64, Height: 64, Hidden: true})
		defer func() { w.Close(); mygo.App.Quit() }()
		if err := w.SetIcon(branding.WindowPNG()); err != nil {
			failure = err
			return
		}
		for kind := 0; kind < 2; kind++ {
			handle, _, _ := send.Call(w.NativeHandle(), 0x7f, uintptr(kind), 0) // WM_GETICON
			var err error
			dimensions[kind], err = inspect(handle)
			if err != nil {
				failure = err
				return
			}
		}
		if dimensions[0][0] >= dimensions[1][0] {
			failure = fmt.Errorf("small/large icon sizes indistinguishable")
			return
		}
		baseline, _, _ = resources.Call(process, 1) // this dedicated process's USER objects
		for i := 0; i < 32; i++ {
			if err := w.SetIcon(branding.WindowPNG()); err != nil {
				failure = err
				return
			}
		}
		after, _, _ = resources.Call(process, 1)
		if baseline == 0 || after > baseline+2 {
			failure = fmt.Errorf("icon replacement leaked USER objects: %d -> %d", baseline, after)
			return
		}
		if err := w.SetIcon(nil); err != nil {
			failure = err
			return
		}
	})
	if err := mygo.App.Run(); err != nil {
		failure = err
	}
	if failure != nil {
		fmt.Fprintln(os.Stderr, failure)
		os.Exit(1)
	}
	_ = json.NewEncoder(os.Stdout).Encode(struct {
		Status       string      `json:"status"`
		Sizes        [2][2]int32 `json:"windowIconBitmapSizes"`
		Resources    []int       `json:"nativeLoadedResourceSizes"`
		Replacements int         `json:"replacementCycles"`
		Before       uintptr     `json:"userObjectsBefore"`
		After        uintptr     `json:"userObjectsAfter"`
		Native       string      `json:"nativeTaskbarVisual"`
	}{"passed", dimensions, iconcheck.Sizes, 32, baseline, after, "not-tested"})
}
