//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"errors"
	"runtime"
	"strings"
	"syscall"
	"unsafe"
)

type propertyKey struct {
	format comGUID
	id     uint32
}

var shellItem2IID = comGUID{0x7e9fb0d3, 0x919f, 0x4307, [8]byte{0xab, 0x2e, 0x9b, 0x18, 0x60, 0x31, 0x0c, 0x93}}
var linkTargetKey = propertyKey{comGUID{0xb9b4b3fc, 0x2b51, 0x4a42, [8]byte{0xb5, 0xd8, 0x32, 0x41, 0x46, 0xaf, 0xcf, 0x25}}, 2}
var linkArgumentsKey = propertyKey{comGUID{0x436f2667, 0x14e2, 0x4feb, [8]byte{0xb3, 0x0a, 0x14, 0x6c, 0x53, 0xb5, 0xb6, 0x74}}, 100}

// PKEY_AppUserModel_ID is {9F4C2855-9F79-4B39-A8D0-E1D42DE1D5F3}, PID 5.
// It identifies desktop applications too; its presence never proves packaging.
// https://learn.microsoft.com/windows/win32/properties/props-system-appusermodel-id
var appUserModelIDKey = propertyKey{comGUID{0x9f4c2855, 0x9f79, 0x4b39, [8]byte{0xa8, 0xd0, 0xe1, 0xd4, 0x2d, 0xe1, 0xd5, 0xf3}}, 5}
var propertyStoreIID = comGUID{0x886d8eeb, 0x8cf2, 0x4446, [8]byte{0x8d, 0x02, 0xcd, 0xba, 0x1d, 0xbd, 0xcf, 0x99}}
var clearPropVariant = ole32.NewProc("PropVariantClear")

// Windows OLE Automation, required for the documented VT_BSTR string form.
var oleAutomation = syscall.NewLazyDLL("oleaut32.dll")
var bstrLength = oleAutomation.NewProc("SysStringLen")

// PROPVARIANT's eight-byte header and sixteen-byte union on Windows amd64.
type stringPropVariant struct {
	kind     uint16
	reserved [3]uint16
	value    [2]uintptr
}

func readStringVariant(object *comObject, method int, key *propertyKey) (string, bool) {
	var v stringPropVariant
	hr := object.call(method, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&v)))
	defer clearPropVariant.Call(uintptr(unsafe.Pointer(&v)))
	// GetValue documents S_OK and INPLACE_S_TRUNCATED canonicalized strings.
	if uint32(hr) != 0 && uint32(hr) != 0x000401a0 {
		return "", false
	}
	if v.kind == 0 {
		return "", uint32(hr) == 0
	} // Successful VT_EMPTY, distinct from a failed read.
	if v.kind != 31 && v.kind != 8 {
		return "", false
	}
	// Read the pointer-typed union overlay directly; do not round-trip a native
	// output pointer through an integer (also preserves the vet/ABI convention).
	pointer := (*struct{ pointer *uint16 })(unsafe.Pointer(&v.value[0])).pointer
	if v.kind == 8 {
		count, _, _ := bstrLength.Call(uintptr(unsafe.Pointer(pointer)))
		if count > 2048 {
			return "", false
		}
		units := make([]uint16, 0, int(count))
		for i := 0; i < int(count); i++ {
			unit := *(*uint16)(unsafe.Add(unsafe.Pointer(pointer), i*2))
			if unit == 0 {
				return "", false
			} // Embedded NUL cannot be an AppID.
			units = append(units, unit)
		}
		return syscall.UTF16ToString(units), true
	}
	if pointer == nil {
		return "", false
	}
	units := make([]uint16, 0, 2048)
	for i := 0; i < 2048; i++ {
		unit := *(*uint16)(unsafe.Add(unsafe.Pointer(pointer), i*2))
		if unit == 0 {
			return syscall.UTF16ToString(units), true
		}
		units = append(units, unit)
	}
	return "", false
}

func shellStringOrEmptyProperty(item *comObject, key *propertyKey) (string, bool) {
	var properties *comObject
	hr := item.call(0, uintptr(unsafe.Pointer(&shellItem2IID)), uintptr(unsafe.Pointer(&properties)))
	if properties != nil {
		defer properties.call(2)
	}
	if comError("query Shell properties", hr) != nil || properties == nil {
		return "", false
	}
	return readStringVariant(properties, 13, key)
}

func shortcutAppUserModelID(link *comObject) (string, bool) {
	var properties *comObject
	hr := link.call(0, uintptr(unsafe.Pointer(&propertyStoreIID)), uintptr(unsafe.Pointer(&properties)))
	if properties != nil {
		defer properties.call(2)
	}
	if uint32(hr) != 0 || properties == nil {
		return "", false
	}
	return readStringVariant(properties, 5, &appUserModelIDKey)
}

func shellStringProperty(item *comObject, key *propertyKey) (string, bool) {
	var properties *comObject
	hr := item.call(0, uintptr(unsafe.Pointer(&shellItem2IID)), uintptr(unsafe.Pointer(&properties)))
	if properties != nil {
		defer properties.call(2)
	}
	if comError("query Shell properties", hr) != nil || properties == nil {
		return "", false
	}
	var value *uint16
	hr = properties.call(17, uintptr(unsafe.Pointer(key)), uintptr(unsafe.Pointer(&value)))
	if value != nil {
		defer freeCOMMemory.Call(uintptr(unsafe.Pointer(value)))
	}
	if comError("read Shell metadata", hr) != nil || value == nil {
		return "", false
	}
	units := make([]uint16, 0, 2048)
	for i := 0; i < 2048; i++ {
		unit := *(*uint16)(unsafe.Add(unsafe.Pointer(value), i*2))
		if unit == 0 {
			return syscall.UTF16ToString(units), true
		}
		units = append(units, unit)
	}
	return "", false
}

// These documented package APIs query only a specific family registered to the
// current user. They never enumerate other users or inspect WindowsApps files.
// https://learn.microsoft.com/windows/win32/api/appmodel/nf-appmodel-getpackagesbypackagefamily
// https://learn.microsoft.com/windows/win32/api/appmodel/nf-appmodel-openpackageinfobyfullname
// https://learn.microsoft.com/windows/win32/api/appmodel/nf-appmodel-getpackageapplicationids
// https://learn.microsoft.com/windows/win32/api/appmodel/nf-appmodel-closepackageinfo
var (
	getPackagesByPackageFamily = kernel32.NewProc("GetPackagesByPackageFamily")
	openPackageInfoByFullName  = kernel32.NewProc("OpenPackageInfoByFullName")
	getPackageApplicationIDs   = kernel32.NewProc("GetPackageApplicationIds")
	closePackageInfo           = kernel32.NewProc("ClosePackageInfo")
)

type shellPackageAPI struct {
	packages     func(*uint16, *uint32, **uint16, *uint32, *uint16) uint32
	open         func(*uint16, *uintptr) uint32
	applications func(uintptr, *uint32, *byte, *uint32) uint32
	close        func(uintptr)
}

func (api shellPackageAPI) available() bool {
	return api.packages != nil && api.open != nil && api.applications != nil && api.close != nil
}

func nativeShellPackageAPI() shellPackageAPI {
	// LazyProc.Call panics if an export is unavailable. Missing package APIs
	// therefore disable this supplement rather than breaking shortcut indexing.
	for _, proc := range []*syscall.LazyProc{getPackagesByPackageFamily, openPackageInfoByFullName, getPackageApplicationIDs, closePackageInfo} {
		if proc.Find() != nil {
			return shellPackageAPI{}
		}
	}
	return shellPackageAPI{
		packages: func(family *uint16, count *uint32, names **uint16, length *uint32, buffer *uint16) uint32 {
			result, _, _ := getPackagesByPackageFamily.Call(uintptr(unsafe.Pointer(family)), uintptr(unsafe.Pointer(count)), uintptr(unsafe.Pointer(names)), uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(buffer)))
			return uint32(result)
		},
		open: func(fullName *uint16, reference *uintptr) uint32 {
			result, _, _ := openPackageInfoByFullName.Call(uintptr(unsafe.Pointer(fullName)), 0, uintptr(unsafe.Pointer(reference)))
			return uint32(result)
		},
		applications: func(reference uintptr, length *uint32, buffer *byte, count *uint32) uint32 {
			result, _, _ := getPackageApplicationIDs.Call(reference, uintptr(unsafe.Pointer(length)), uintptr(unsafe.Pointer(buffer)), uintptr(unsafe.Pointer(count)))
			return uint32(result)
		},
		close: func(reference uintptr) { closePackageInfo.Call(reference) },
	}
}

var errShellPackageProof = errors.New("packaged application verification unavailable")

func packagedAUMIDFamily(appID string) (string, bool) {
	family, relative, found := strings.Cut(appID, "!")
	if !found || family == "" || relative == "" || strings.Contains(relative, "!") {
		return "", false
	}
	wide, err := syscall.UTF16FromString(appID)
	// APPLICATION_USER_MODEL_ID_MAX_LENGTH includes the NUL terminator.
	// https://learn.microsoft.com/windows/win32/appxpkg/identity-constants
	if err != nil || len(wide) > 130 {
		return "", false
	}
	return family, true
}

// The family!application shape is just a bounded lookup key, not evidence.
// Require the exact AUMID in the package API's application list, so neither a
// desktop AppID nor a made-up application in an installed family is accepted.
// A healthy negative is distinct from unavailable/inconsistent native output:
// a partial scan must not erase previously indexed Store applications.
func registeredPackageApplication(appID string, api shellPackageAPI) (bool, error) {
	family, valid := packagedAUMIDFamily(appID)
	if !valid {
		return false, nil
	}
	if !api.available() {
		return false, errShellPackageProof
	}
	familyWide, err := syscall.UTF16PtrFromString(family)
	if err != nil {
		return false, errShellPackageProof
	}
	var count, length uint32
	result := api.packages(familyWide, &count, nil, &length, nil)
	if result == 0 && count == 0 && length == 0 {
		return false, nil // No package registered to this user for the family.
	}
	if result != uint32(syscall.ERROR_INSUFFICIENT_BUFFER) || count == 0 || count > 64 || length == 0 || length > 64*1024 {
		return false, errShellPackageProof
	}
	names := make([]*uint16, count)
	buffer := make([]uint16, length)
	result = api.packages(familyWide, &count, &names[0], &length, &buffer[0])
	runtime.KeepAlive(familyWide)
	if result != 0 || count == 0 || count > uint32(len(names)) || length == 0 || length > uint32(len(buffer)) {
		return false, errShellPackageProof
	}
	// Copy validated names before calling more native APIs. Package lists can
	// change between the size/read calls; a failed or oversized result is partial.
	fullNames := make([]string, 0, count)
	for _, name := range names[:count] {
		fullName, ok := packageBufferString(buffer[:length], name, 0)
		if !ok || fullName == "" {
			return false, errShellPackageProof
		}
		fullNames = append(fullNames, fullName)
	}
	var problem error
	for _, fullName := range fullNames {
		found, err := packageContainsApplication(fullName, appID, api)
		if found {
			return true, nil
		}
		if err != nil {
			problem = err
		}
	}
	return false, problem
}

func packageContainsApplication(fullName, appID string, api shellPackageAPI) (bool, error) {
	wide, err := syscall.UTF16PtrFromString(fullName)
	if err != nil {
		return false, errShellPackageProof
	}
	var reference uintptr
	result := api.open(wide, &reference)
	runtime.KeepAlive(wide)
	if result != 0 || reference == 0 {
		return false, errShellPackageProof
	}
	defer api.close(reference)
	var length, count uint32
	result = api.applications(reference, &length, nil, &count)
	if result == 0 && length == 0 && count == 0 {
		return false, nil // Framework/resource packages can have no applications.
	}
	if result != uint32(syscall.ERROR_INSUFFICIENT_BUFFER) || length == 0 || length > 64*1024 {
		return false, errShellPackageProof
	}
	// The returned buffer begins with count PWSTRs followed by UTF-16 strings.
	// Allocate pointer-aligned storage, validate every pointer against this owned
	// buffer, and never read beyond its returned byte length.
	wordSize := uint32(unsafe.Sizeof(uintptr(0)))
	buffer := make([]uintptr, (length+wordSize-1)/wordSize)
	capacity := length
	if api.applications(reference, &length, (*byte)(unsafe.Pointer(&buffer[0])), &count) != 0 || length > capacity || length%2 != 0 || count == 0 || count > 128 || uint64(count)*uint64(wordSize) >= uint64(length) {
		return false, errShellPackageProof
	}
	units := unsafe.Slice((*uint16)(unsafe.Pointer(&buffer[0])), int(length/2))
	matched := false
	for i := uint32(0); i < count; i++ {
		pointer := *(**uint16)(unsafe.Pointer(&buffer[i]))
		id, ok := packageBufferString(units, pointer, int(count*wordSize/2))
		if !ok || id == "" {
			return false, errShellPackageProof
		}
		matched = matched || strings.EqualFold(id, appID)
	}
	runtime.KeepAlive(buffer)
	return matched, nil
}

func packageBufferString(buffer []uint16, pointer *uint16, minimum int) (string, bool) {
	if len(buffer) == 0 || pointer == nil {
		return "", false
	}
	base, address := uintptr(unsafe.Pointer(&buffer[0])), uintptr(unsafe.Pointer(pointer))
	if address < base || (address-base)%2 != 0 || (address-base)/2 >= uintptr(len(buffer)) || (address-base)/2 < uintptr(minimum) {
		return "", false
	}
	start := int((address - base) / 2)
	for i := start; i < len(buffer) && i-start < 2048; i++ {
		if buffer[i] == 0 {
			return syscall.UTF16ToString(buffer[start:i]), true
		}
	}
	return "", false
}
