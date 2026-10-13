//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later

package platform

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"syscall"
	"time"
	"unsafe"
)

var (
	user32                     = syscall.NewLazyDLL("user32.dll")
	kernel32                   = syscall.NewLazyDLL("kernel32.dll")
	shell32                    = syscall.NewLazyDLL("shell32.dll")
	ole32                      = syscall.NewLazyDLL("ole32.dll")
	getForegroundWindow        = user32.NewProc("GetForegroundWindow")
	setForegroundWindow        = user32.NewProc("SetForegroundWindow")
	isWindow                   = user32.NewProc("IsWindow")
	getWindowThreadProcessID   = user32.NewProc("GetWindowThreadProcessId")
	getAsyncKeyState           = user32.NewProc("GetAsyncKeyState")
	getGUIThreadInfo           = user32.NewProc("GetGUIThreadInfo")
	isChild                    = user32.NewProc("IsChild")
	sendMessageTimeout         = user32.NewProc("SendMessageTimeoutW")
	getClipboardSequenceNumber = user32.NewProc("GetClipboardSequenceNumber")
	openClipboard              = user32.NewProc("OpenClipboard")
	closeClipboard             = user32.NewProc("CloseClipboard")
	emptyClipboard             = user32.NewProc("EmptyClipboard")
	isClipboardFormatAvailable = user32.NewProc("IsClipboardFormatAvailable")
	getClipboardData           = user32.NewProc("GetClipboardData")
	setClipboardData           = user32.NewProc("SetClipboardData")
	createWindowEx             = user32.NewProc("CreateWindowExW")
	destroyWindow              = user32.NewProc("DestroyWindow")
	globalAlloc                = kernel32.NewProc("GlobalAlloc")
	globalFree                 = kernel32.NewProc("GlobalFree")
	globalLock                 = kernel32.NewProc("GlobalLock")
	globalUnlock               = kernel32.NewProc("GlobalUnlock")
	globalSize                 = kernel32.NewProc("GlobalSize")
	moveMemory                 = kernel32.NewProc("RtlMoveMemory")
	shellExecute               = shell32.NewProc("ShellExecuteW")
	coInitializeEx             = ole32.NewProc("CoInitializeEx")
	coUninitialize             = ole32.NewProc("CoUninitialize")
	clipboardMu                sync.Mutex
	pasteMu                    sync.Mutex
	targetMu                   sync.Mutex
	capturedTarget             *pasteTarget
)

const (
	cfUnicodeText = 13
	gmemMoveable  = 0x0002
	wmPaste       = 0x0302
)

type guiThreadInfo struct {
	cbSize    uint32
	flags     uint32
	active    uintptr
	focus     uintptr
	capture   uintptr
	menuOwner uintptr
	moveSize  uintptr
	caret     uintptr
	caretRect [4]int32
}

type pasteTarget struct {
	top           uintptr
	focus         uintptr
	topIdentity   windowIdentity
	focusIdentity windowIdentity
}

func foregroundWindow() uintptr {
	hwnd, _, _ := getForegroundWindow.Call()
	return hwnd
}

func focusedChild(top uintptr, thread uint32) uintptr {
	info := guiThreadInfo{}
	info.cbSize = uint32(unsafe.Sizeof(info))
	ok, _, _ := getGUIThreadInfo.Call(uintptr(thread), uintptr(unsafe.Pointer(&info)))
	if ok == 0 || info.focus == 0 {
		return 0
	}
	if info.focus != top {
		child, _, _ := isChild.Call(top, info.focus)
		if child == 0 {
			return 0
		}
	}
	return info.focus
}

// ForegroundWindow captures both the top-level target and its currently focused
// child before the launcher is shown. The capture is consumed by one paste call.
// Call this for a new user activation, not as a polling or visibility helper.
func ForegroundWindow() uintptr {
	targetMu.Lock()
	defer targetMu.Unlock()
	capturedTarget = nil
	hwnd := foregroundWindow()
	id, ok := identifyWindow(hwnd)
	if !ok || id.processID == uint32(os.Getpid()) {
		return 0
	}
	focus := focusedChild(hwnd, id.threadID)
	focusID, ok := identifyWindow(focus)
	if !ok || focusID.processID != id.processID || foregroundWindow() != hwnd {
		return 0
	}
	capturedTarget = &pasteTarget{top: hwnd, focus: focus, topIdentity: id, focusIdentity: focusID}
	return hwnd
}

// ClipboardSequence is a change token, not a count. Zero can mean unavailable.
func ClipboardSequence() uint32 {
	sequence, _, _ := getClipboardSequenceNumber.Call()
	return uint32(sequence)
}

func win32Error(operation string, err error) error {
	if err == nil || err == syscall.Errno(0) {
		return fmt.Errorf("%s failed", operation)
	}
	return fmt.Errorf("%s: %w", operation, err)
}

// acquireClipboard makes at most six attempts, waiting at most 100 ms total.
func acquireClipboard(owner uintptr) error {
	var last error
	for attempt := 0; attempt < 6; attempt++ {
		ok, _, err := openClipboard.Call(owner)
		if ok != 0 {
			return nil
		}
		last = err
		if attempt < 5 {
			time.Sleep(20 * time.Millisecond)
		}
	}
	return win32Error("open clipboard", last)
}

// ReadClipboardText never reads beyond GlobalSize or 64 KiB, does not enumerate
// other formats, and neither logs clipboard contents nor changes the clipboard.
func ReadClipboardText() (string, error) {
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	if err := acquireClipboard(0); err != nil {
		return "", err
	}
	defer closeClipboard.Call()
	if available, _, _ := isClipboardFormatAvailable.Call(cfUnicodeText); available == 0 {
		return "", ErrClipboardFormatUnavailable
	}
	handle, _, err := getClipboardData.Call(cfUnicodeText)
	if handle == 0 {
		return "", win32Error("read clipboard", err)
	}
	size, _, err := globalSize.Call(handle)
	if size < 2 {
		return "", win32Error("read clipboard allocation size", err)
	}
	if size > MaxClipboardBytes {
		size = MaxClipboardBytes
	}
	ptr, _, err := globalLock.Call(handle)
	if ptr == 0 {
		return "", win32Error("lock clipboard text", err)
	}
	defer globalUnlock.Call(handle)
	// Copy a bounded view into Go-owned memory while the HGLOBAL is locked.
	// Keeping the native address as uintptr avoids unsafe Go pointer conversions.
	units := make([]uint16, int(size/2))
	moveMemory.Call(uintptr(unsafe.Pointer(&units[0])), ptr, uintptr(len(units)*2))
	runtime.KeepAlive(units)
	return decodeClipboardText(units)
}

// WriteClipboardText replaces the clipboard at the user's request. It does not
// save or later restore previous contents, which might overwrite a newer copy.
func WriteClipboardText(text string) error {
	_, err := writeClipboardText(text)
	return err
}

func writeClipboardText(text string) (uint32, error) {
	units, err := encodeClipboardText(text)
	if err != nil {
		return 0, err
	}
	clipboardMu.Lock()
	defer clipboardMu.Unlock()
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()

	// A non-null window is required for clipboard ownership after EmptyClipboard.
	// STATIC is a built-in class; this message-only window is never displayed.
	class, _ := syscall.UTF16PtrFromString("STATIC")
	owner, _, callErr := createWindowEx.Call(0, uintptr(unsafe.Pointer(class)), 0, 0,
		0, 0, 0, 0, ^uintptr(2), 0, 0, 0) // HWND_MESSAGE = (HWND)-3.
	if owner == 0 {
		return 0, win32Error("create clipboard owner", callErr)
	}
	defer destroyWindow.Call(owner)

	handle, _, callErr := globalAlloc.Call(gmemMoveable, uintptr(len(units)*2))
	if handle == 0 {
		return 0, win32Error("allocate clipboard text", callErr)
	}
	owned := true
	defer func() {
		if owned {
			globalFree.Call(handle)
		}
	}()
	ptr, _, callErr := globalLock.Call(handle)
	if ptr == 0 {
		return 0, win32Error("lock clipboard allocation", callErr)
	}
	moveMemory.Call(ptr, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)*2))
	runtime.KeepAlive(units)
	globalUnlock.Call(handle)
	if err := acquireClipboard(owner); err != nil {
		return 0, err
	}
	defer closeClipboard.Call()
	if ok, _, err := emptyClipboard.Call(); ok == 0 {
		return 0, win32Error("empty clipboard", err)
	}
	if ok, _, err := setClipboardData.Call(cfUnicodeText, handle); ok == 0 {
		return 0, win32Error("write clipboard text", err)
	}
	owned = false // Windows now owns and eventually frees the HGLOBAL.
	// Capture while still holding the clipboard, before another writer can
	// replace it. The caller rechecks after CloseClipboard and before delivery.
	return ClipboardSequence(), nil
}

func identifyWindow(hwnd uintptr) (windowIdentity, bool) {
	if hwnd == 0 {
		return windowIdentity{}, false
	}
	if ok, _, _ := isWindow.Call(hwnd); ok == 0 {
		return windowIdentity{}, false
	}
	var processID uint32
	threadID, _, _ := getWindowThreadProcessID.Call(hwnd, uintptr(unsafe.Pointer(&processID)))
	return windowIdentity{processID: processID, threadID: uint32(threadID)}, threadID != 0 && processID != 0
}

func pasteKeysHeld() bool {
	// Refuse while activation modifiers or V are still held. Never synthesize
	// key-up events or wait and deliver a paste after the user has moved on.
	for _, key := range [...]uintptr{0x10, 0x11, 0x12, 0x5B, 0x5C, 0x56} {
		state, _, _ := getAsyncKeyState.Call(key)
		if state&0x8000 != 0 {
			return true
		}
	}
	return false
}

func sendTargetedPaste(target pasteTarget, sequence uint32) error {
	topID, topOK := identifyWindow(target.top)
	focusID, focusOK := identifyWindow(target.focus)
	if !topOK || !focusOK || topID != target.topIdentity || focusID != target.focusIdentity {
		return ErrTargetUnavailable
	}
	if focusedChild(target.top, topID.threadID) != target.focus || foregroundWindow() != target.top {
		return ErrFocusChanged
	}
	// WM_PASTE is addressed only to the captured focused HWND. Unlike global
	// keyboard injection, a focus change cannot redirect it into another app.
	// It is not broadcast, posted for later, retried, or followed by SendInput.
	var result uintptr
	if sequence == 0 || ClipboardSequence() != sequence {
		return ErrClipboardChanged
	}
	ok, _, err := sendMessageTimeout.Call(target.focus, wmPaste, 0, 0,
		0x0001|0x0002|0x0020, 250, uintptr(unsafe.Pointer(&result)))
	if ok == 0 {
		return win32Error("send paste to original control (delivery may be uncertain; do not retry automatically)", err)
	}
	return nil
}

// PasteToWindow must only be called immediately after an explicit user paste
// action and after hiding the launcher. target must come from ForegroundWindow
// before showing the launcher. The capture can be used only once.
//
// The message is sent to the original focused child, with repeated window,
// process, thread, and foreground checks. No global keyboard input is injected.
// A nil error means WM_PASTE was delivered, not that text was inserted: custom
// editors and browsers may ignore it. An error after copying can leave selected
// text on the clipboard. Offer manual Ctrl+V; never retry paste automatically.
// Clipboard sequence checks detect observed replacements, but Windows offers no
// atomic check-and-paste operation; another process can change it at dispatch.
func PasteToWindow(target uintptr, text string) error {
	pasteMu.Lock()
	defer pasteMu.Unlock()
	targetMu.Lock()
	captured := capturedTarget
	if captured != nil && captured.top == target {
		capturedTarget = nil
	} else {
		captured = nil
	}
	targetMu.Unlock()
	if captured == nil {
		return ErrTargetUnavailable
	}
	var copiedSequence uint32
	return pasteSafely(target, text, pasteActions{
		currentProcess: uint32(os.Getpid()),
		identity: func(hwnd uintptr) (windowIdentity, bool) {
			id, ok := identifyWindow(hwnd)
			if hwnd == captured.top && id != captured.topIdentity {
				ok = false
			}
			return id, ok
		},
		foreground: foregroundWindow,
		activate:   func(hwnd uintptr) { setForegroundWindow.Call(hwnd) },
		keysHeld:   pasteKeysHeld,
		writeClipboard: func(text string) (uint32, error) {
			sequence, err := writeClipboardText(text)
			copiedSequence = sequence
			return sequence, err
		},
		clipboardSequence: ClipboardSequence,
		sendPaste:         func() error { return sendTargetedPaste(*captured, copiedSequence) },
	})
}

// Launch invokes an existing absolute .exe or .lnk through ShellExecuteW. No
// shell command string, extra arguments, elevation verb, or URL is constructed.
func Launch(path string) error {
	path, err := validateLaunchPath(path)
	if err != nil {
		return err
	}
	file, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return err
	}
	verb, _ := syscall.UTF16PtrFromString("open")
	dir, _ := syscall.UTF16PtrFromString(filepath.Dir(path))
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	// Shell extensions used to resolve .lnk files can require an STA apartment.
	hr, _, _ := coInitializeEx.Call(0, 0x2|0x4)
	if int32(hr) < 0 {
		return fmt.Errorf("initialize application launch: HRESULT 0x%08x", uint32(hr))
	}
	defer coUninitialize.Call()
	result, _, _ := shellExecute.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0,
		uintptr(unsafe.Pointer(dir)), 1)
	if result <= 32 {
		return fmt.Errorf("Windows could not launch the application (ShellExecute code %d)", result)
	}
	return nil
}
