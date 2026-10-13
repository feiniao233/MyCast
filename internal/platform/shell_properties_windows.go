//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
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
