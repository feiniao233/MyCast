//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
// Owned hidden MyGo window only; never show, activate or change display settings.
package main

import (
	"fmt"
	"github.com/egoist/mygo"
	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/platform"
	"os"
	"syscall"
	"unsafe"
)

func main() {
	var failure error
	mygo.App.SetName("MyCast dedicated compact window fixture")
	mygo.App.WhenReady(func() {
		query := "Dedicated synthetic fixture"
		w := mygo.NewWindow(mygo.WindowOptions{Title: "MyCast synthetic compact fixture", Width: 750, Height: 56, Hidden: true, Frameless: true, Content: ui.View(func(c *ui.Context) { ui.TextInputBase(c.Key("synthetic-search"), &query) })})
		defer func() { w.Close(); mygo.App.Quit() }()
		user := syscall.NewLazyDLL("user32.dll")
		kernel := syscall.NewLazyDLL("kernel32.dll")
		getRect := user.NewProc("GetWindowRect")
		visible := user.NewProc("IsWindowVisible")
		resources := user.NewProc("GetGuiResources")
		process, _, _ := kernel.NewProc("GetCurrentProcess").Call()
		var anchor [4]int32
		if _, err := platform.ResizePalette(w.NativeHandle(), 750, 56, 475, true); err != nil {
			failure = err
			return
		}
		getRect.Call(w.NativeHandle(), uintptr(unsafe.Pointer(&anchor[0])))
		before, _, _ := resources.Call(process, 0)
		beforeUser, _, _ := resources.Call(process, 1)
		for i := 0; i < 64; i++ {
			for _, height := range []int{475, 56} {
				if _, err := platform.ResizePalette(w.NativeHandle(), 750, height, 475, false); err != nil {
					failure = err
					return
				}
				var current [4]int32
				getRect.Call(w.NativeHandle(), uintptr(unsafe.Pointer(&current[0])))
				shown, _, _ := visible.Call(w.NativeHandle())
				if shown != 0 || current[0] != anchor[0] || current[1] != anchor[1] {
					failure = fmt.Errorf("hidden visibility/top anchor changed")
					return
				}
			}
		}
		after, _, _ := resources.Call(process, 0)
		afterUser, _, _ := resources.Call(process, 1)
		if beforeUser == 0 || afterUser > beforeUser+2 || after > before+2 || query != "Dedicated synthetic fixture" {
			failure = fmt.Errorf("resize resource/input regression")
		}
	})
	if err := mygo.App.Run(); err != nil {
		failure = err
	}
	if failure != nil {
		fmt.Fprintln(os.Stderr, failure)
		os.Exit(1)
	}
	fmt.Println("Owned hidden MyGo compact/full sizing, top anchor and 64-cycle USER/GDI/input checks passed; actual desktop/DPI acceptance not claimed.")
}
