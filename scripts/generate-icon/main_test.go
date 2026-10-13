// SPDX-License-Identifier: AGPL-3.0-or-later
package main

import (
	"bytes"
	"debug/pe"
	"encoding/binary"
	"os"
	"strings"
	"testing"

	"github.com/feiniao233/MyCast/internal/iconcheck"
)

func TestGeneratedOutputsMatchSVG(t *testing.T) {
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir("../.."); err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(old)
	if err := run(true); err != nil {
		t.Fatal(err)
	}
}

func TestUnsupportedSVGDoesNotSilentlyDisappear(t *testing.T) {
	svg, err := os.ReadFile("../../assets/brand/mycast.svg")
	if err != nil {
		t.Fatal(err)
	}
	for _, bad := range []string{strings.Replace(string(svg), "</svg>", "<circle cx=\"16\" cy=\"16\" r=\"3\"/></svg>", 1), strings.Replace(string(svg), "fill=\"url(#mark)\"", "fill=\"url(#mark)\" transform=\"rotate(10)\"", 1), strings.Replace(string(svg), "M 7 23", "Q 7 23 8 24", 1)} {
		if _, err := raster([]byte(bad), 16); err == nil {
			t.Fatal("unsupported/invalid geometry accepted")
		}
	}
}
func TestCOFFResourceRelocationsAndGroup(t *testing.T) {
	data, err := os.ReadFile("../../assets/brand/mycast.ico")
	if err != nil {
		t.Fatal(err)
	}
	images, err := iconcheck.ICO(data)
	if err != nil {
		t.Fatal(err)
	}
	f, err := pe.NewFile(bytes.NewReader(coff(images)))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if f.Machine != 0x8664 || f.TimeDateStamp != 0 || len(f.Sections) != 1 || f.Sections[0].Name != ".rsrc" || f.Sections[0].Characteristics != 0x40000040 || len(f.Sections[0].Relocs) != 9 || len(f.Symbols) != 1 {
		t.Fatal("unexpected code/timestamp/section/relocation metadata")
	}
	resource, err := f.Sections[0].Data()
	if err != nil {
		t.Fatal(err)
	}
	for i, r := range f.Sections[0].Relocs {
		if r.Type != 3 || r.SymbolTableIndex != 0 || int(r.VirtualAddress)+16 > len(resource) {
			t.Fatal("unsafe resource relocation")
		}
		off := binary.LittleEndian.Uint32(resource[r.VirtualAddress:])
		length := binary.LittleEndian.Uint32(resource[r.VirtualAddress+4:])
		if uint64(off)+uint64(length) > uint64(len(resource)) {
			t.Fatal("bad resource payload")
		}
		payload := resource[off : off+length]
		if i < 8 {
			if !bytes.Equal(payload, images[i]) {
				t.Fatal("COFF image differs")
			}
		} else {
			if err := iconcheck.Group(payload, images); err != nil {
				t.Fatal(err)
			}
			bad := append([]byte(nil), payload...)
			bad[18] = 9
			if iconcheck.Group(bad, images) == nil {
				t.Fatal("corrupt group accepted")
			}
		}
	}
}
