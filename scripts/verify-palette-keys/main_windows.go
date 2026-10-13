//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
// Dedicated hidden MyGo drawing-surface fixture. No tray, app discovery,
// clipboard or input to an existing window; messages target only this fixture.
package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/platform"
)

func main() {
	var failure error
	mygo.App.SetName("MyCast dedicated palette key fixture")
	mygo.App.WhenReady(func() {
		w := mygo.NewWindow(mygo.WindowOptions{Title: "MyCast synthetic palette key fixture", Width: 160, Height: 100, Hidden: true, Content: ui.View(func(c *ui.Context) { ui.Text(c, "Dedicated synthetic fixture") })})
		defer func() { w.Close(); mygo.App.Quit() }()
		guard, err := platform.InstallPaletteKeyGuard(w.NativeHandle())
		if err != nil {
			failure = err
			return
		}
		defer guard.Close()
		user := syscall.NewLazyDLL("user32.dll")
		class, _ := syscall.UTF16PtrFromString("MyGoSurface")
		surface, _, _ := user.NewProc("FindWindowExW").Call(w.NativeHandle(), 0, uintptr(unsafe.Pointer(class)), 0)
		if surface == 0 {
			failure = fmt.Errorf("owned MyGo drawing surface missing")
			return
		}
		send := user.NewProc("SendMessageW")
		for _, key := range []uintptr{0x0d, 0x1b, 0x5d, 0x79} {
			send.Call(surface, 0x100, key, 1<<30)
		}
		if guard.SuppressedCount() != 4 {
			failure = fmt.Errorf("repeated actions reached owned MyGo surface")
			return
		}
		send.Call(surface, 0x104, 0x79, 1<<30) // Real WM_SYSKEYDOWN F10, no Alt context.
		if guard.SuppressedCount() != 5 {
			failure = fmt.Errorf("repeated system F10 reached MyGo surface")
			return
		}
		guard.SetComposing(true)
		send.Call(surface, 0x100, 0x0d, 1<<30)
		send.Call(surface, 0x104, 0x79, 1<<30)
		if guard.SuppressedCount() != 5 {
			failure = fmt.Errorf("composing key was consumed")
			return
		}
		guard.SetComposing(false)
		if !guard.Close() || !guard.Close() {
			failure = fmt.Errorf("surface guard cleanup failed")
		}
	})
	if err := mygo.App.Run(); err != nil {
		failure = err
	}
	if failure != nil {
		fmt.Fprintln(os.Stderr, failure)
		os.Exit(1)
	}
	fmt.Println("Owned hidden MyGoSurface repeat/IME pass-through and cleanup passed; no desktop acceptance claimed.")
}
