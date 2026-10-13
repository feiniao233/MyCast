//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"testing"
	"unicode/utf16"
)

func TestLocalizedResourceFixtureHasNoExecutableCode(t *testing.T) {
	file, err := pe.NewFile(bytes.NewReader(localizedResourcePEFixture("Dedicated synthetic 资源 ABI")))
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	header, ok := file.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || header.AddressOfEntryPoint != 0 || header.DataDirectory[1].Size != 0 || header.DataDirectory[2].VirtualAddress == 0 {
		t.Fatal("resource fixture has entrypoint/imports or no resources")
	}
	for _, section := range file.Sections {
		if section.Characteristics&(0x20|0x20000000) != 0 {
			t.Fatal("fixture contains executable/code section")
		}
	}
}

// localizedResourcePEFixture creates one read-only .rsrc section containing
// RT_STRING/1/neutral and string ID 1. All code/entrypoint/import directories are
// zero. The production data/image mapping reads it without DLL initialization.
func localizedResourcePEFixture(value string) []byte {
	units := utf16.Encode([]rune(value))
	payload := []byte{0, 0} // string ID 0 is empty
	payload = binary.LittleEndian.AppendUint16(payload, uint16(len(units)))
	for _, unit := range units {
		payload = binary.LittleEndian.AppendUint16(payload, unit)
	}
	for i := 2; i < 16; i++ {
		payload = binary.LittleEndian.AppendUint16(payload, 0)
	}
	resource := make([]byte, 88+len(payload))
	put16 := func(offset int, value uint16) { binary.LittleEndian.PutUint16(resource[offset:], value) }
	put32 := func(offset int, value uint32) { binary.LittleEndian.PutUint32(resource[offset:], value) }
	put16(14, 1)
	put32(16, 6)
	put32(20, 0x80000000|24) // RT_STRING
	put16(24+14, 1)
	put32(40, 1)
	put32(44, 0x80000000|48) // block 1
	put16(48+14, 1)
	put32(64, 0)
	put32(68, 72) // neutral language
	put32(72, 0x1000+88)
	put32(76, uint32(len(payload)))
	copy(resource[88:], payload)
	image := make([]byte, 1024)
	copy(image, "MZ")
	binary.LittleEndian.PutUint32(image[60:], 128)
	copy(image[128:], []byte{'P', 'E', 0, 0})
	coff := image[132:152]
	binary.LittleEndian.PutUint16(coff, 0x8664) // amd64
	binary.LittleEndian.PutUint16(coff[2:], 1)
	binary.LittleEndian.PutUint16(coff[16:], 240)
	binary.LittleEndian.PutUint16(coff[18:], 0x2002) // DLL, executable image; no executable section
	opt := image[152:392]
	binary.LittleEndian.PutUint16(opt, 0x20b)
	binary.LittleEndian.PutUint32(opt[8:], 512)
	binary.LittleEndian.PutUint64(opt[24:], 0x180000000)
	binary.LittleEndian.PutUint32(opt[32:], 4096)
	binary.LittleEndian.PutUint32(opt[36:], 512)
	binary.LittleEndian.PutUint16(opt[40:], 6)
	binary.LittleEndian.PutUint16(opt[48:], 6)
	binary.LittleEndian.PutUint32(opt[56:], 8192)
	binary.LittleEndian.PutUint32(opt[60:], 512)
	binary.LittleEndian.PutUint16(opt[68:], 2)
	binary.LittleEndian.PutUint16(opt[70:], 0x100)
	binary.LittleEndian.PutUint64(opt[72:], 1<<20)
	binary.LittleEndian.PutUint64(opt[80:], 4096)
	binary.LittleEndian.PutUint64(opt[88:], 1<<20)
	binary.LittleEndian.PutUint64(opt[96:], 4096)
	binary.LittleEndian.PutUint32(opt[108:], 16)
	binary.LittleEndian.PutUint32(opt[128:], 0x1000)
	binary.LittleEndian.PutUint32(opt[132:], uint32(len(resource)))
	section := image[392:432]
	copy(section, ".rsrc")
	binary.LittleEndian.PutUint32(section[8:], uint32(len(resource)))
	binary.LittleEndian.PutUint32(section[12:], 0x1000)
	binary.LittleEndian.PutUint32(section[16:], 512)
	binary.LittleEndian.PutUint32(section[20:], 512)
	binary.LittleEndian.PutUint32(section[36:], 0x40000040)
	copy(image[512:], resource)
	return image
}
