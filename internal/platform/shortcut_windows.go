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

type comGUID struct {
	A    uint32
	B, C uint16
	D    [8]byte
}
type comObject struct{ methods *[24]uintptr }

var (
	createCOMInstance        = ole32.NewProc("CoCreateInstance")
	expandEnvironmentStrings = kernel32.NewProc("ExpandEnvironmentStringsW")
	shellLinkClass           = comGUID{0x00021401, 0, 0, [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	shellLinkInterface       = comGUID{0x000214f9, 0, 0, [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	persistFileInterface     = comGUID{0x0000010b, 0, 0, [8]byte{0xc0, 0, 0, 0, 0, 0, 0, 0x46}}
	shellLinkDataInterface   = comGUID{0x45e2b4ae, 0xb1c3, 0x11d0, [8]byte{0xb9, 0x2f, 0, 0xa0, 0xc9, 3, 0x12, 0xe1}}
)

// uintptr arguments can carry Go output/string pointers. Match the syscall
// wrapper's contract so the compiler keeps those pointees on the heap/alive
// across stack growth in this wrapper and the native call.
//
//go:uintptrescapes
func (o *comObject) call(index int, args ...uintptr) uintptr {
	if o == nil || o.methods == nil || index < 0 || index >= len(o.methods) || o.methods[index] == 0 {
		return uintptr(0x80004003)
	} // E_POINTER
	params := append([]uintptr{uintptr(unsafe.Pointer(o))}, args...)
	result, _, _ := syscall.SyscallN(o.methods[index], params...)
	runtime.KeepAlive(o)
	return result
}

func comError(operation string, result uintptr) error {
	if uint32(result)&0x80000000 != 0 {
		return fmt.Errorf("%s: HRESULT 0x%08x", operation, uint32(result))
	}
	return nil
}

// ShortcutIdentity loads .lnk metadata in read-only mode. It never calls
// IShellLink.Resolve, Save, ShellExecute, or any method that repairs a link.
// A virtual/advertised target without a filesystem path is left unmerged.
func ShortcutIdentity(path string) (core.LaunchIdentity, error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("initialize shortcut reader", hr); err != nil {
		return core.LaunchIdentity{}, err
	}
	defer coUninitialize.Call()
	var link *comObject
	hr, _, _ = createCOMInstance.Call(uintptr(unsafe.Pointer(&shellLinkClass)), 0, 1, uintptr(unsafe.Pointer(&shellLinkInterface)), uintptr(unsafe.Pointer(&link)))
	if err := comError("create shortcut reader", hr); err != nil {
		return core.LaunchIdentity{}, err
	}
	if link == nil {
		return core.LaunchIdentity{}, fmt.Errorf("shortcut reader returned no interface")
	}
	defer link.call(2)
	var file *comObject
	if err := comError("query shortcut persistence", link.call(0, uintptr(unsafe.Pointer(&persistFileInterface)), uintptr(unsafe.Pointer(&file)))); err != nil {
		return core.LaunchIdentity{}, err
	}
	if file == nil {
		return core.LaunchIdentity{}, fmt.Errorf("shortcut persistence returned no interface")
	}
	defer file.call(2)
	encoded, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return core.LaunchIdentity{}, err
	}
	if err := comError("load shortcut", file.call(5, uintptr(unsafe.Pointer(encoded)), 0)); err != nil {
		return core.LaunchIdentity{}, err
	}
	runtime.KeepAlive(encoded)
	read := func(index int, pathFlags bool) (string, error) {
		buffer := make([]uint16, 32768)
		var result uintptr
		if pathFlags {
			result = link.call(index, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)), 0, 4)
		} else {
			result = link.call(index, uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
		}
		// SLGP_RAWPATH; direct conversions honor the uintptr escape contract.
		runtime.KeepAlive(buffer)
		err := comError("read shortcut metadata", result)
		if buffer[len(buffer)-1] != 0 || buffer[len(buffer)-2] != 0 {
			return "", fmt.Errorf("shortcut metadata exceeds reader limit")
		}
		return syscall.UTF16ToString(buffer), err
	}
	target, err := read(3, true)
	if err != nil {
		return core.LaunchIdentity{}, err
	}
	// GetPath documents a MAX_PATH limit even with a larger buffer. A value
	// reaching that limit cannot prove two long targets are identical.
	if len(syscall.StringToUTF16(target)) >= 260 {
		return core.LaunchIdentity{}, fmt.Errorf("shortcut target reaches the path reader limit")
	}
	arguments, err := read(10, false)
	if err != nil {
		return core.LaunchIdentity{}, err
	}
	directory, err := read(8, false)
	if err != nil {
		return core.LaunchIdentity{}, err
	}
	var data *comObject
	if err := comError("query shortcut launch flags", link.call(0, uintptr(unsafe.Pointer(&shellLinkDataInterface)), uintptr(unsafe.Pointer(&data)))); err != nil {
		return core.LaunchIdentity{}, err
	}
	if data == nil {
		return core.LaunchIdentity{}, fmt.Errorf("shortcut flags returned no interface")
	}
	defer data.call(2)
	var flags uint32
	if err := comError("read shortcut launch flags", data.call(6, uintptr(unsafe.Pointer(&flags)))); err != nil {
		return core.LaunchIdentity{}, err
	}
	var show int32
	if err := comError("read shortcut show mode", link.call(14, uintptr(unsafe.Pointer(&show)))); err != nil {
		return core.LaunchIdentity{}, err
	}
	// Only ordinary link structure, environment target strings and icon
	// strings are understood. RunAsUser (0x2000), advertised/shim links and
	// every unknown option keep their own shortcut identity. No link flags
	// or show modes are modified by this reader.
	const ordinaryFlags uint32 = 0xff | 0x200 | 0x4000
	identity := core.LaunchIdentity{Target: expandShortcutPath(target), Arguments: arguments, WorkingDirectory: expandShortcutPath(directory), LinkFlags: flags, ShowCommand: show, Independent: flags & ^ordinaryFlags != 0 || show != 1}
	identity.AppUserModelID = shortcutAppUserModelID(link)
	if identity.Target == "" && identity.Arguments == "" && identity.WorkingDirectory == "" && !identity.Independent {
		identity.ShellTarget = shortcutAppsFolderTarget(link)
	}
	return identity, nil
}

func expandShortcutPath(path string) string {
	if path == "" {
		return ""
	}
	encoded, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return path
	}
	buffer := make([]uint16, 32768)
	count, _, _ := expandEnvironmentStrings.Call(uintptr(unsafe.Pointer(encoded)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(len(buffer)))
	runtime.KeepAlive(encoded)
	if count == 0 || count > uintptr(len(buffer)) {
		return path
	}
	return syscall.UTF16ToString(buffer)
}
