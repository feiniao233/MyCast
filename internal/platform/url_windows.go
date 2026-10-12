//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"fmt"
	"runtime"
	"syscall"
	"unsafe"

	"github.com/feiniao233/MyCast/internal/core"
)

// OpenHTTPURL delegates an explicitly selected, validated link to Windows.
// It never fetches the URL, builds a command line or runs a shell interpreter.
func OpenHTTPURL(value string) error {
	value, err := core.ValidateHTTPURL(value)
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(value)
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 0x2|0x4)
	if int32(hr) < 0 {
		return fmt.Errorf("initialize link opener failed")
	}
	defer coUninitialize.Call()
	result, _, _ := shellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
	runtime.KeepAlive(file)
	runtime.KeepAlive(verb)
	if result <= 32 {
		return fmt.Errorf("Windows link open failed (code %d)", result)
	}
	return nil
}
