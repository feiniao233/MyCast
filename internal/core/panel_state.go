// SPDX-License-Identifier: AGPL-3.0-or-later
package core

// PanelState intentionally has no result count, scan progress or icon state.
// Only user presentation/input and full-panel requirements decide visibility.
type PanelState struct {
	CompactEnabled    bool
	Query             string
	SessionExpanded   bool
	FullPanelRequired bool
	Composing         bool
}
type PanelMetrics struct{ Width, ExpandedHeight, CompactHeight int }
type PanelLayout struct {
	Width, Height           int
	Compact, ResultsVisible bool
}

// DecidePanelLayout is the single pure decision for both desired dimensions
// and result visibility. It does not inspect settings storage or call UI APIs.
// Any actual typed text, including whitespace, preserves the full result view.
func DecidePanelLayout(state PanelState, metrics PanelMetrics) PanelLayout {
	compact := state.CompactEnabled && state.Query == "" && !state.SessionExpanded && !state.FullPanelRequired && !state.Composing
	height := metrics.ExpandedHeight
	if compact {
		height = metrics.CompactHeight
	}
	return PanelLayout{Width: metrics.Width, Height: height, Compact: compact, ResultsVisible: !compact}
}
