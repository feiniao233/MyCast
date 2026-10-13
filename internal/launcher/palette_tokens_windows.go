//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import "github.com/egoist/mygo/ui"

// Standard DIPs; the renderer handles DPI. Compact preserves header/footer access.
type paletteTokens struct {
	CompactHeight                                            float32
	ModalSide, ModalTop, ModalPadding, ModalGap, ModalRadius float32
	ModalBody, ModalClear, ModalBack                         float32
	ModalScrim                                               ui.Color

	FooterIcon, KeyGlyph, ControlLine                                                   float32
	Width, Height, Header, Footer, PanelRadius, RowRadius                               float32
	Inset, RowInset, Gap, RowGap, RowHeight, DetailRowHeight                            float32
	Icon, IconSlot, MenuButton, ActionHeight, MenuWidth, MenuRadius                     float32
	MenuInset, MenuRow, SectionHeight, SearchSize, BodySize, CaptionSize                float32
	RowPaddingY, ControlInset, DetailGap, DetailSize, KeyCap, KeyCapRadius, StatusWidth float32
	TopFade, BottomFade, TopFloor, BottomFloor                                          float32
	Fallback, Scrim, Text, Secondary, Tertiary, Selection, Hover, Control, Border       ui.Color
	MenuSurface                                                                         ui.Color
}

var palette = paletteTokens{
	CompactHeight: 108,
	ModalSide:     36, ModalTop: 18, ModalPadding: 16, ModalGap: 12, ModalRadius: 18,
	ModalBody: 13, ModalClear: 30, ModalBack: 32, ModalScrim: ui.RGBA(0, 0, 0, .30),

	FooterIcon: 20, KeyGlyph: 14, ControlLine: 22,
	// The HWND owns outer rounding; a second transparent cutout would expose
	// the system backdrop between the UI and the native window outline.
	Width: 750, Height: 475, Header: 44, Footer: 52, PanelRadius: 0, RowRadius: 10,
	Inset: 8, RowInset: 8, Gap: 10, RowGap: 2, RowHeight: 38, DetailRowHeight: 46,
	Icon: 24, IconSlot: 26, MenuButton: 36, ActionHeight: 28, MenuWidth: 276, MenuRadius: 16,
	MenuInset: 6, MenuRow: 30, SectionHeight: 24, SearchSize: 20, BodySize: 14, CaptionSize: 11,
	RowPaddingY: 6, ControlInset: 4, DetailGap: 1, DetailSize: 10, KeyCap: 18, KeyCapRadius: 6, StatusWidth: 280,
	TopFade: 84, BottomFade: 80, TopFloor: .15, BottomFloor: .25,
	Fallback: ui.RGB(24, 24, 24), Scrim: ui.RGBA(0, 0, 0, .40), Text: ui.RGBA(255, 255, 255, 1),
	Secondary: ui.RGBA(255, 255, 255, .60), Tertiary: ui.RGBA(255, 255, 255, .40),
	Selection: ui.RGBA(255, 255, 255, .10), Hover: ui.RGBA(255, 255, 255, .05),
	Control: ui.RGBA(255, 255, 255, .10), Border: ui.RGBA(255, 255, 255, .20),
	MenuSurface: ui.RGB(38, 38, 38),
}

func paletteFor(dark, highContrast bool) paletteTokens {
	p := palette
	if !dark {
		p.Fallback, p.MenuSurface = ui.RGB(245, 245, 245), ui.RGB(250, 250, 250)
		p.Scrim = ui.RGBA(255, 255, 255, .55)
		p.Text = ui.RGB(0, 0, 0)
		p.Secondary, p.Tertiary = ui.RGBA(0, 0, 0, .60), ui.RGBA(0, 0, 0, .42)
		p.Selection, p.Hover = ui.RGBA(0, 0, 0, .09), ui.RGBA(0, 0, 0, .045)
		p.Control, p.Border = ui.RGBA(0, 0, 0, .08), ui.RGBA(0, 0, 0, .18)
	}
	if highContrast {
		p.Secondary, p.Tertiary, p.Border = p.Text, p.Text, p.Text
		p.Scrim = p.Fallback
		if dark {
			p.Selection, p.Hover, p.Control = ui.RGB(64, 64, 64), ui.RGB(48, 48, 48), ui.RGB(40, 40, 40)
		} else {
			p.Selection, p.Hover, p.Control = ui.RGB(220, 220, 220), ui.RGB(235, 235, 235), ui.RGB(240, 240, 240)
		}
	}
	return p
}

func launcherTheme(palette paletteTokens, dark bool) *ui.Theme {
	theme := *ui.DarkTheme()
	if !dark {
		theme = *ui.LightTheme()
	}
	// Widgets, especially Select's overlay, need opaque semantic surfaces.
	// Only c.Root is cleared in view; transparency is not a theme-wide setting.
	theme.Background, theme.Surface = palette.MenuSurface, palette.Fallback
	theme.Border, theme.Text, theme.TextMuted = palette.Border, palette.Text, palette.Tertiary
	// Retain the upstream accent/AccentText pair instead of white on white.
	theme.Focus, theme.Radius = palette.Border, palette.RowRadius
	return &theme
}

// Supported per-row alpha approximation, not a pixel gradient mask. It keeps
// scroll geometry intact and does not fade a resting first/last row at its edge.
func rowOpacity(y, height, viewport, offset, remaining float32) float32 {
	alpha := float32(1)
	if offset > 0 {
		strength := min(offset/palette.TopFade, 1)
		position := max(0, min((y+height/2-palette.Header)/(palette.TopFade-palette.Header), 1))
		alpha = min(alpha, 1-strength*(1-palette.TopFloor)*(1-position))
	}
	if remaining > 0 {
		strength := min(remaining/palette.BottomFade, 1)
		position := max(0, min((viewport-palette.Footer-y-height/2)/(palette.BottomFade-palette.Footer), 1))
		alpha = min(alpha, 1-strength*(1-palette.BottomFloor)*(1-position))
	}
	return alpha
}
