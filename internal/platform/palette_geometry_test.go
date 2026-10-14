// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import "testing"

func TestPaletteTopAnchorDPIAndWorkArea(t *testing.T) {
	for _, dpi := range []int{96, 144, 192} {
		for _, work := range []paletteRect{{0, 0, 1920, 1040}, {-1920, -200, 0, 840}, {20, 30, 900, 670}} {
			compact, err := fitPaletteBounds(paletteRect{0, 0, 750, 475}, work, 750, 56, 475, dpi, true)
			if err != nil {
				t.Fatal(err)
			}
			if compact.bottom-compact.top != 56*dpi/96 {
				t.Fatal("compact physical height is not one 56-DIP row", dpi, compact)
			}
			for cycle := 0; cycle < 64; cycle++ {
				full, err := fitPaletteBounds(compact, work, 750, 475, 475, dpi, false)
				if err != nil {
					t.Fatal(err)
				}
				if full.top != compact.top || full.left != compact.left || full.right > work.right || full.bottom > work.bottom || full.top < work.top || full.left < work.left {
					t.Fatal("anchor/work-area drift", dpi, work, compact, full)
				}
				next, err := fitPaletteBounds(full, work, 750, 56, 475, dpi, false)
				if err != nil || next != compact {
					t.Fatal("compact cycle drift", dpi, next, err)
				}
			}
		}
	}
}
func TestPaletteGeometryRejectsInvalidAndClampsMovedWindow(t *testing.T) {
	work := paletteRect{10, 20, 1010, 820}
	got, err := fitPaletteBounds(paletteRect{990, 800, 1740, 1275}, work, 750, 475, 475, 96, false)
	if err != nil || got.left != 260 || got.top != 345 || got.right != 1010 || got.bottom != 820 {
		t.Fatal(got, err)
	}
	for _, m := range [][4]int{{0, 56, 475, 96}, {750, 1001, 1001, 96}, {750, 475, 56, 96}, {750, 56, 475, 0}, {750, 56, 475, 900}} {
		if _, err := fitPaletteBounds(got, work, m[0], m[1], m[2], m[3], false); err == nil {
			t.Fatal("invalid geometry accepted", m)
		}
	}
}

func TestPaletteValidatedFrameInsetsAndClamping(t *testing.T) {
	work := paletteRect{0, 0, 1600, 900}
	compact, err := fitPaletteBounds(paletteRect{}, work, 750, 56, 475, 144, true, paletteFrame{16, 16})
	if err != nil {
		t.Fatal(err)
	}
	if compact.bottom-compact.top != 100 {
		t.Fatal("compact client/frame height", compact)
	}
	full, err := fitPaletteBounds(compact, work, 750, 475, 475, 144, false, paletteFrame{16, 16})
	if err != nil || full.top != compact.top || full.bottom-full.top != 729 {
		t.Fatal("client/frame geometry", full, err)
	}
	if _, err := fitPaletteBounds(compact, work, 750, 56, 475, 144, false, paletteFrame{-1, 16}); err == nil {
		t.Fatal("negative native inset accepted")
	}
	if _, err := fitPaletteBounds(compact, work, 750, 56, 475, 144, false, paletteFrame{16, 129}); err == nil {
		t.Fatal("unknown large frame accepted")
	}
}
