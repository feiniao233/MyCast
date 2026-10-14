// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import "testing"

func TestPanelStateTruthTable(t *testing.T) {
	metrics := PanelMetrics{Width: 750, ExpandedHeight: 475, CompactHeight: 56}
	for mask := 0; mask < 16; mask++ {
		for _, query := range []string{"", " ", "Synthetic", "合成"} {
			state := PanelState{CompactEnabled: mask&1 != 0, SessionExpanded: mask&2 != 0, FullPanelRequired: mask&4 != 0, Composing: mask&8 != 0, Query: query}
			got := DecidePanelLayout(state, metrics)
			compact := mask == 1 && query == ""
			wantHeight := 475
			if compact {
				wantHeight = 56
			}
			if got.Compact != compact || got.ResultsVisible == compact || got.Width != 750 || got.Height != wantHeight {
				t.Fatalf("state=%+v layout=%+v", state, got)
			}
		}
	}
}
func TestPanelExplicitExpansionAndReopen(t *testing.T) {
	metrics := PanelMetrics{750, 475, 56}
	state := PanelState{CompactEnabled: true}
	if !DecidePanelLayout(state, metrics).Compact {
		t.Fatal("empty configured call not compact")
	}
	state.Query = "合成"
	if DecidePanelLayout(state, metrics).Compact {
		t.Fatal("typing did not expand")
	}
	state.Query = ""
	state.SessionExpanded = true
	if DecidePanelLayout(state, metrics).Compact {
		t.Fatal("explicit expanded results hidden")
	}
	state.FullPanelRequired = true
	state.SessionExpanded = false
	if DecidePanelLayout(state, metrics).Compact {
		t.Fatal("menu/settings/clipboard inaccessible")
	}
	state.FullPanelRequired = false
	if !DecidePanelLayout(state, metrics).Compact {
		t.Fatal("reopen/return state did not reset")
	}
}
