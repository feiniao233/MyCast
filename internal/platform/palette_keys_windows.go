//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"sync/atomic"
	"syscall"
	"unsafe"
)

var keyGuardThread = kernel32.NewProc("GetCurrentThreadId")
var keyGuardSystemDirectory = kernel32.NewProc("GetSystemDirectoryW")
var keyGuardDLLOnce sync.Once
var keyGuardDLL *syscall.LazyDLL
var keyGuardDLLError error
var keyGuardMu sync.Mutex
var keyGuards = map[uintptr]*PaletteKeyGuard{}
var keyGuardNext atomic.Uint64
var keyGuardCallback uintptr

func init() { keyGuardCallback = syscall.NewCallback(paletteKeySubclass) }

// PaletteKeyGuard owns one same-thread, same-process subclass. Native reference
// data is a numeric token, never a Go pointer. WM_NCDESTROY releases ownership.
type PaletteKeyGuard struct {
	hwnd, token       uintptr
	nativeComposition bool // owning UI thread only
	composition       atomic.Bool
	suppressed        atomic.Uint64
}

func paletteKeysDLL() (*syscall.LazyDLL, error) {
	keyGuardDLLOnce.Do(func() {
		var directory [260]uint16
		n, _, _ := keyGuardSystemDirectory.Call(uintptr(unsafe.Pointer(&directory[0])), uintptr(len(directory)))
		if n == 0 || n >= uintptr(len(directory)) {
			keyGuardDLLError = fmt.Errorf("Windows system directory unavailable")
			return
		}
		keyGuardDLL = syscall.NewLazyDLL(filepath.Join(syscall.UTF16ToString(directory[:n]), "comctl32.dll"))
		for _, name := range []string{"SetWindowSubclass", "RemoveWindowSubclass", "DefSubclassProc"} {
			if err := keyGuardDLL.NewProc(name).Find(); err != nil {
				keyGuardDLLError = fmt.Errorf("Windows subclass API unavailable")
				return
			}
		}
	})
	return keyGuardDLL, keyGuardDLLError
}
func keyGuardOwnThread(hwnd uintptr) bool {
	if hwnd == 0 {
		return false
	}
	var pid uint32
	thread, _, _ := getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&pid)))
	current, _, _ := keyGuardThread.Call()
	return thread != 0 && thread == current && pid == uint32(os.Getpid())
}

// InstallPaletteKeyGuard locates MyGo v0.4.6's own immediate drawing surface.
// A missing/unrecognized surface fails closed; never attach to another app.
func InstallPaletteKeyGuard(palette uintptr) (*PaletteKeyGuard, error) {
	if !keyGuardOwnThread(palette) {
		return nil, fmt.Errorf("palette ownership/thread unavailable")
	}
	class, _ := syscall.UTF16PtrFromString("MyGoSurface")
	surface, _, _ := paletteFindChild.Call(palette, 0, uintptr(unsafe.Pointer(class)), 0)
	if surface == 0 {
		return nil, fmt.Errorf("MyGo drawing surface unavailable")
	}
	return installPaletteKeyGuardSurface(surface)
}
func installPaletteKeyGuardSurface(hwnd uintptr) (*PaletteKeyGuard, error) {
	if !keyGuardOwnThread(hwnd) {
		return nil, fmt.Errorf("surface ownership/thread unavailable")
	}
	dll, err := paletteKeysDLL()
	if err != nil {
		return nil, err
	}
	keyGuardMu.Lock()
	if len(keyGuards) >= 8 {
		keyGuardMu.Unlock()
		return nil, fmt.Errorf("palette guard budget reached")
	}
	for _, g := range keyGuards {
		if g.hwnd == hwnd {
			keyGuardMu.Unlock()
			return nil, fmt.Errorf("surface already guarded")
		}
	}
	token := uintptr(keyGuardNext.Add(1))
	g := &PaletteKeyGuard{hwnd: hwnd, token: token}
	keyGuards[token] = g
	keyGuardMu.Unlock()
	ok, _, _ := dll.NewProc("SetWindowSubclass").Call(hwnd, keyGuardCallback, token, token)
	if ok == 0 {
		keyGuardMu.Lock()
		delete(keyGuards, token)
		keyGuardMu.Unlock()
		return nil, fmt.Errorf("palette subclass registration failed")
	}
	return g, nil
}
func (g *PaletteKeyGuard) SetComposing(value bool) {
	if g != nil {
		g.composition.Store(value)
	}
}
func (g *PaletteKeyGuard) SuppressedCount() uint64 {
	if g == nil {
		return 0
	}
	return g.suppressed.Load()
}
func (g *PaletteKeyGuard) Close() bool {
	if g == nil {
		return true
	}
	keyGuardMu.Lock()
	active := keyGuards[g.token] == g
	keyGuardMu.Unlock()
	if !active {
		return true
	}
	if !keyGuardOwnThread(g.hwnd) {
		return false
	}
	dll, err := paletteKeysDLL()
	if err != nil {
		return false
	}
	ok, _, _ := dll.NewProc("RemoveWindowSubclass").Call(g.hwnd, keyGuardCallback, g.token)
	if ok == 0 {
		return false
	} // Keep referenced state until native destruction.
	keyGuardMu.Lock()
	delete(keyGuards, g.token)
	keyGuardMu.Unlock()
	return true
}
func suppressPaletteRepeat(message, key, flags uintptr, composing bool) bool {
	if flags&(1<<30) == 0 || composing {
		return false
	}
	if message == 0x104 {
		// F10 (including Shift+F10) is also delivered as WM_SYSKEYDOWN.
		// Bit 29 denotes Alt context: preserve Alt+F10 and every other
		// system key, including Alt+F4. No global modifier/key injection.
		return key == 0x79 && flags&(1<<29) == 0
	}
	return message == 0x100 && (key == 0x0d || key == 0x1b || key == 0x5d || key == 0x79)
}
func paletteKeySubclass(hwnd, message, wp, lp, id, reference uintptr) uintptr {
	keyGuardMu.Lock()
	g := keyGuards[reference]
	keyGuardMu.Unlock()
	if g != nil && g.hwnd == hwnd && g.token == id {
		switch message {
		case 0x10d:
			g.nativeComposition = true // WM_IME_STARTCOMPOSITION
		case 0x10e, 0x8:
			g.nativeComposition = false // END / KILLFOCUS
		case 0x82:
			keyGuardDLL.NewProc("RemoveWindowSubclass").Call(hwnd, keyGuardCallback, id)
			keyGuardMu.Lock()
			delete(keyGuards, reference)
			keyGuardMu.Unlock()
		}
		if suppressPaletteRepeat(message, wp, lp, g.nativeComposition || g.composition.Load()) {
			g.suppressed.Add(1)
			return 0
		}
	}
	r, _, _ := keyGuardDLL.NewProc("DefSubclassProc").Call(hwnd, message, wp, lp)
	return r
}
