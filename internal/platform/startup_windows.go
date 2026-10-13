//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"fmt"
	"github.com/feiniao233/MyCast/internal/settings"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"unicode/utf16"
	"unsafe"
)

const startupKey = `Software\Microsoft\Windows\CurrentVersion\Run`
const startupValue = "MyCast.Launcher"

var (
	registryDLL    = syscall.NewLazyDLL("advapi32.dll")
	regOpen        = registryDLL.NewProc("RegOpenKeyExW")
	regCreate      = registryDLL.NewProc("RegCreateKeyExW")
	regClose       = registryDLL.NewProc("RegCloseKey")
	regQuery       = registryDLL.NewProc("RegQueryValueExW")
	regSet         = registryDLL.NewProc("RegSetValueExW")
	regDeleteValue = registryDLL.NewProc("RegDeleteValueW")
)

func startupCommand(path string) (string, error) {
	if !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) || !strings.EqualFold(filepath.Ext(path), ".exe") || strings.ContainsAny(path, "\x00\r\n\"") {
		return "", fmt.Errorf("invalid startup executable")
	}
	command := `"` + filepath.Clean(path) + `" --background`
	if len(utf16.Encode([]rune(command))) >= 260 {
		return "", fmt.Errorf("startup command exceeds Windows Run limit")
	}
	return command, nil
}

func NewStartupRegistration(executable string) (*settings.StartupRegistration, error) {
	if err := LocalAppPath(executable); err != nil {
		return nil, err
	}
	if _, err := validateLaunchPath(executable); err != nil {
		return nil, err
	}
	command, err := startupCommand(executable)
	if err != nil {
		return nil, err
	}
	return startupRegistrationAt(startupKey, startupValue, command), nil
}

type startupRegistry struct{ key, name string }

func startupRegistrationAt(key, name, command string) *settings.StartupRegistration {
	registry := startupRegistry{key: key, name: name}
	return &settings.StartupRegistration{Command: command, Read: registry.read, Write: registry.write, Delete: registry.delete}
}
func registryError(result uintptr) error {
	if result == 0 {
		return nil
	}
	return syscall.Errno(result)
}

func (r startupRegistry) open(write, create bool) (uintptr, error) {
	wide, err := syscall.UTF16PtrFromString(r.key)
	if err != nil {
		return 0, err
	}
	rights := uintptr(1)
	if write {
		rights |= 2
	}
	var key uintptr
	var result uintptr
	if create {
		result, _, _ = regCreate.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(wide)), 0, 0, 0, rights, 0, uintptr(unsafe.Pointer(&key)), 0)
	} else {
		result, _, _ = regOpen.Call(uintptr(syscall.HKEY_CURRENT_USER), uintptr(unsafe.Pointer(wide)), 0, rights, uintptr(unsafe.Pointer(&key)))
	}
	runtime.KeepAlive(wide)
	if err := registryError(result); err != nil {
		return 0, err
	}
	if key == 0 {
		return 0, fmt.Errorf("registry returned no handle")
	}
	return key, nil
}
func (r startupRegistry) read() (string, bool, error) {
	key, err := r.open(false, false)
	if err == syscall.ERROR_FILE_NOT_FOUND {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	defer regClose.Call(key)
	name, err := syscall.UTF16PtrFromString(r.name)
	if err != nil {
		return "", false, err
	}
	buffer := make([]uint16, 512)
	bytes := uint32(len(buffer) * 2)
	var kind uint32
	result, _, _ := regQuery.Call(key, uintptr(unsafe.Pointer(name)), 0, uintptr(unsafe.Pointer(&kind)), uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(&bytes)))
	runtime.KeepAlive(name)
	runtime.KeepAlive(buffer)
	if result == uintptr(syscall.ERROR_FILE_NOT_FOUND) {
		return "", false, nil
	}
	if err := registryError(result); err != nil {
		return "", false, err
	}
	if kind != 1 || bytes < 2 || bytes%2 != 0 || bytes > uint32(len(buffer)*2) || buffer[bytes/2-1] != 0 {
		return "", false, fmt.Errorf("startup registry value is not bounded REG_SZ")
	}
	units := buffer[:bytes/2-1]
	for _, v := range units {
		if v == 0 {
			return "", false, fmt.Errorf("startup registry value has embedded terminator")
		}
	}
	return syscall.UTF16ToString(units), true, nil
}
func (r startupRegistry) write(command string) error {
	units, err := syscall.UTF16FromString(command)
	if err != nil || len(units) > 260 {
		return fmt.Errorf("invalid startup command")
	}
	key, err := r.open(true, true)
	if err != nil {
		return err
	}
	defer regClose.Call(key)
	name, err := syscall.UTF16PtrFromString(r.name)
	if err != nil {
		return err
	}
	result, _, _ := regSet.Call(key, uintptr(unsafe.Pointer(name)), 0, 1, uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)*2))
	runtime.KeepAlive(name)
	runtime.KeepAlive(units)
	return registryError(result)
}
func (r startupRegistry) delete() error {
	key, err := r.open(true, false)
	if err == syscall.ERROR_FILE_NOT_FOUND {
		return nil
	}
	if err != nil {
		return err
	}
	defer regClose.Call(key)
	name, err := syscall.UTF16PtrFromString(r.name)
	if err != nil {
		return err
	}
	result, _, _ := regDeleteValue.Call(key, uintptr(unsafe.Pointer(name)))
	runtime.KeepAlive(name)
	if result == uintptr(syscall.ERROR_FILE_NOT_FOUND) {
		return nil
	}
	return registryError(result)
}
