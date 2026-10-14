//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import "github.com/egoist/mygo/ui"

// compactButtonTooltip stays beside the control within the single search row.
// MyGo 0.4.6's default Tooltip is an in-window overlay: its above/below
// attachment cannot fit beside a 32-DIP control in a 56-DIP window. TooltipBase
// retains MyGo's delayed hover, press/Escape dismissal and pass-through behavior
// while this horizontal attachment keeps the complete label in the client area.
func compactButtonTooltip(c *ui.Context, anchor ui.Element, text string, p paletteTokens) ui.Element {
	anchor.Description(text)
	return ui.TooltipBase(c, anchor, func(tip ui.Element) {
		width, _ := c.Size()
		bounds := anchor.Bounds()
		left := bounds.X - p.Inset - p.ControlInset
		right := width - bounds.X - bounds.W - p.Inset - p.ControlInset
		available := right
		if left > right {
			available = left
			tip.AttachTo(anchor, ui.AnchorLeft, ui.AnchorRight).Margin(0, p.ControlInset, 0, 0)
		} else {
			tip.AttachTo(anchor, ui.AnchorRight, ui.AnchorLeft).Margin(0, 0, 0, p.ControlInset)
		}
		tip.MaxWidth(max(1, min(p.MenuWidth, available))).Padding(p.ControlInset, p.Inset).
			Radius(p.RowRadius).Background(p.Text).TextColor(p.Fallback).
			FontSize(p.BodySize).Label("提示：" + text)
		tip.Children(func() {
			ui.Text(c, text).FixedLineHeight(p.ControlLine).SingleLine().Ellipsis("…")
		})
	})
}
