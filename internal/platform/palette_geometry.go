// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import "fmt"

type paletteRect struct{ left, top, right, bottom int }
type PaletteSize struct{ Width, Height float32 }
type paletteFrame struct{ width, height int }

// Physical pixels preserve an exact top anchor even at fractional scale. The
// initial anchor reserves the largest expanded height before compact display.
func fitPaletteBounds(current, work paletteRect, width, height, expanded, dpi int, center bool, frames ...paletteFrame) (paletteRect, error) {
	if width < 1 || width > 1600 || height < 1 || height > 1000 || expanded < height || expanded > 1000 || dpi < 48 || dpi > 768 {
		return paletteRect{}, fmt.Errorf("invalid palette dimensions")
	}
	workWidth, workHeight := work.right-work.left, work.bottom-work.top
	if workWidth <= 0 || workHeight <= 0 || workWidth > 200000 || workHeight > 200000 {
		return paletteRect{}, fmt.Errorf("invalid monitor work area")
	}
	px := func(value int) int { return (value*dpi + 48) / 96 }
	frame := paletteFrame{}
	if len(frames) > 0 {
		frame = frames[0]
	}
	if len(frames) > 1 || frame.width < 0 || frame.height < 0 || frame.width > 128 || frame.height > 128 {
		return paletteRect{}, fmt.Errorf("invalid palette frame insets")
	}
	w, h, reserved := min(px(width)+frame.width, workWidth), min(px(height)+frame.height, workHeight), min(px(expanded)+frame.height, workHeight)
	x, y := current.left, current.top
	if center {
		x = work.left + (workWidth-w)/2
		y = work.top + (workHeight-reserved)/2
	}
	x = max(work.left, min(x, work.right-w))
	y = max(work.top, min(y, work.bottom-h))
	return paletteRect{x, y, x + w, y + h}, nil
}
