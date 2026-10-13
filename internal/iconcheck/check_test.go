// SPDX-License-Identifier: AGPL-3.0-or-later
package iconcheck

import (
	"bytes"
	"image/png"
	"os"
	"testing"
)

func TestICOTransparentSizesAndMarkContrast(t *testing.T) {
	data, err := os.ReadFile("../../assets/brand/mycast.ico")
	if err != nil {
		t.Fatal(err)
	}
	images, err := ICO(data)
	if err != nil {
		t.Fatal(err)
	}
	for i, b := range images {
		im, err := png.Decode(bytes.NewReader(b))
		if err != nil {
			t.Fatal(err)
		}
		n := Sizes[i]
		r, g, blue, a := im.At(n/2, n/2).RGBA()
		tr, tg, tb, _ := im.At(n/2, n/6).RGBA()
		if a < 60000 || g < 40000 || blue < 40000 || int(r)+int(g)+int(blue) < 2*(int(tr)+int(tg)+int(tb)) {
			t.Fatalf("M does not stand out against tile at %dpx", n)
		}
	}
	for _, mutate := range []func([]byte){func(b []byte) { b[4] = 9 }, func(b []byte) { b[6] = 17 }, func(b []byte) { b[18] = 0 }, func(b []byte) { b[14] = 0 }, func(b []byte) { b[len(b)-8] ^= 255 }} {
		bad := append([]byte(nil), data...)
		mutate(bad)
		if _, err := ICO(bad); err == nil {
			t.Fatal("corrupted ICO accepted")
		}
	}
	for _, n := range []int{0, 5, 20, len(data) - 1} {
		if _, err := ICO(data[:n]); err == nil {
			t.Fatal("truncated ICO accepted")
		}
	}
}
