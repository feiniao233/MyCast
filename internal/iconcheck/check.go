// SPDX-License-Identifier: AGPL-3.0-or-later
// Package iconcheck verifies a build's PE icon resources against the source ICO.
// It performs read-only, bounded binary parsing; it never loads executable code.
package iconcheck

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"fmt"
	"image/png"
	"os"
)

var Sizes = []int{16, 20, 24, 32, 48, 64, 128, 256}
var le = binary.LittleEndian

func ICO(data []byte) ([][]byte, error) {
	if len(data) < 6 || len(data) > 1<<20 || le.Uint16(data) != 0 || le.Uint16(data[2:]) != 1 || int(le.Uint16(data[4:])) != len(Sizes) || len(data) < 6+16*len(Sizes) {
		return nil, fmt.Errorf("invalid ICO header")
	}
	images := make([][]byte, len(Sizes))
	next := 6 + 16*len(Sizes)
	for i, n := range Sizes {
		e := data[6+16*i : 6+16*(i+1)]
		dim := func(b byte) int {
			if b == 0 {
				return 256
			}
			return int(b)
		}
		length, off := int(le.Uint32(e[8:])), int(le.Uint32(e[12:]))
		if dim(e[0]) != n || dim(e[1]) != n || e[2] != 0 || e[3] != 0 || le.Uint16(e[4:]) != 1 || le.Uint16(e[6:]) != 32 || off != next || length < 8 || off > len(data) || length > len(data)-off {
			return nil, fmt.Errorf("invalid ICO entry %d", n)
		}
		images[i] = data[off : off+length]
		next = off + length
		config, err := png.DecodeConfig(bytes.NewReader(images[i]))
		if err != nil || config.Width != n || config.Height != n {
			return nil, fmt.Errorf("invalid PNG dimensions %d", n)
		}
		im, err := png.Decode(bytes.NewReader(images[i]))
		if err != nil {
			return nil, fmt.Errorf("invalid PNG entry %d", n)
		}
		for _, p := range [][2]int{{0, 0}, {n - 1, 0}, {0, n - 1}, {n - 1, n - 1}} {
			_, _, _, a := im.At(p[0], p[1]).RGBA()
			if a != 0 {
				return nil, fmt.Errorf("opaque exterior %d", n)
			}
		}
	}
	if next != len(data) {
		return nil, fmt.Errorf("unaccounted ICO bytes")
	}
	return images, nil
}

func Group(data []byte, images [][]byte) error {
	if len(data) != 6+14*len(Sizes) || le.Uint16(data) != 0 || le.Uint16(data[2:]) != 1 || int(le.Uint16(data[4:])) != len(Sizes) || len(images) != len(Sizes) {
		return fmt.Errorf("invalid RT_GROUP_ICON")
	}
	for i, n := range Sizes {
		e := data[6+14*i:]
		if e[0] != byte(n) || e[1] != byte(n) || e[2] != 0 || e[3] != 0 || le.Uint16(e[4:]) != 1 || le.Uint16(e[6:]) != 32 || int(le.Uint32(e[8:])) != len(images[i]) || int(le.Uint16(e[12:])) != i+1 {
			return fmt.Errorf("group mismatch %d", n)
		}
	}
	return nil
}

func Executable(path string, ico []byte) error {
	images, err := ICO(ico)
	if err != nil {
		return err
	}
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if stat.Size() > 512<<20 {
		return fmt.Errorf("executable exceeds verification limit")
	}
	f, err := pe.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	header, ok := f.OptionalHeader.(*pe.OptionalHeader64)
	if !ok || f.Machine != pe.IMAGE_FILE_MACHINE_AMD64 || header.Subsystem != pe.IMAGE_SUBSYSTEM_WINDOWS_GUI || header.DataDirectory[2].Size == 0 {
		return fmt.Errorf("not a Windows amd64 GUI image with resources")
	}
	rva := header.DataDirectory[2].VirtualAddress
	var resource []byte
	for _, s := range f.Sections {
		if rva >= s.VirtualAddress && uint64(rva) < uint64(s.VirtualAddress)+uint64(s.Size) {
			b, e := s.Data()
			if e != nil {
				return e
			}
			start := int(rva - s.VirtualAddress)
			size := int(header.DataDirectory[2].Size)
			if start > len(b) || size > len(b)-start {
				return fmt.Errorf("invalid resource bounds")
			}
			resource = b[start : start+size]
		}
	}
	if len(resource) == 0 {
		return fmt.Errorf("missing resource section")
	}
	find := func(off int, id uint32) (uint32, error) {
		if off < 0 || off > len(resource)-16 {
			return 0, fmt.Errorf("invalid directory")
		}
		count := int(le.Uint16(resource[off+12:])) + int(le.Uint16(resource[off+14:]))
		if count > 64 || count > (len(resource)-off-16)/8 {
			return 0, fmt.Errorf("invalid directory count")
		}
		for i := 0; i < count; i++ {
			e := resource[off+16+i*8:]
			if le.Uint32(e) == id {
				return le.Uint32(e[4:]), nil
			}
		}
		return 0, fmt.Errorf("missing resource ID %d", id)
	}
	read := func(typ, id uint32) ([]byte, error) {
		off := uint32(0)
		for _, key := range []uint32{typ, id} {
			next, e := find(int(off), key)
			if e != nil {
				return nil, e
			}
			if next&0x80000000 == 0 {
				return nil, fmt.Errorf("expected subdirectory")
			}
			off = next & 0x7fffffff
		}
		entry, e := find(int(off), 0)
		if e != nil {
			return nil, e
		}
		if entry&0x80000000 != 0 || uint64(entry)+16 > uint64(len(resource)) {
			return nil, fmt.Errorf("invalid leaf")
		}
		leaf := resource[int(entry):]
		payloadRVA, length := le.Uint32(leaf), le.Uint32(leaf[4:])
		if payloadRVA < rva || length > 1<<20 {
			return nil, fmt.Errorf("invalid payload")
		}
		start := uint64(payloadRVA - rva)
		if start+uint64(length) > uint64(len(resource)) {
			return nil, fmt.Errorf("out-of-bounds payload")
		}
		return resource[start : start+uint64(length)], nil
	}
	group, err := read(14, 1)
	if err != nil {
		return err
	}
	if err := Group(group, images); err != nil {
		return err
	}
	for i, image := range images {
		got, e := read(3, uint32(i+1))
		if e != nil {
			return e
		}
		if !bytes.Equal(got, image) {
			return fmt.Errorf("PE icon differs from source ICO at %d", Sizes[i])
		}
	}
	return nil
}
