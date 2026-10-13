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

func shellStringProperty(item *comObject, key *propertyKey) (string, bool) {
	var properties *comObject
	hr := item.call(0, uintptr(unsafe.Pointer(&shellItem2IID)), uintptr(unsafe.Pointer(&properties)))
	if comError("query Shell properties", hr) != nil || properties == nil {
		return "", false
	}
	defer properties.call(2)
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
