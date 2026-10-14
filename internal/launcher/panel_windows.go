//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
	"math"
)

func (s *launcher) panelLayout() core.PanelLayout {
	return core.DecidePanelLayout(core.PanelState{
		CompactEnabled: s.config.CompactOnShow, Query: s.query, SessionExpanded: s.panelExpanded,
		FullPanelRequired: s.clipboardMode || s.settingsOpen || s.menu != menuClosed || s.panelUnavailable,
		Composing:         s.searchWasComposing,
	}, core.PanelMetrics{Width: int(palette.Width), ExpandedHeight: int(palette.Height), CompactHeight: int(palette.CompactHeight)})
}

// Only one queued update; decide again on the UI thread rather than publishing
// a captured stale query/mode. No timers or icon/index work are introduced.
func (s *launcher) requestPanelLayout() {
	if s.resizePanel == nil {
		s.panelApplied = s.panelLayout()
		return
	}
	if s.panelApplied == s.panelLayout() || s.panelQueued {
		return
	}
	s.panelGeometryPending = true
	if s.uiUpdate == nil {
		return
	}
	s.panelQueued = true
	s.uiUpdate(func() {
		s.panelQueued = false
		if s.quitting {
			return
		}
		s.applyPanelLayout(false)
	})
}
func (s *launcher) applyPanelLayout(center bool) {
	layout := s.panelLayout()
	if !center && layout == s.panelApplied {
		return
	}
	if s.resizePanel == nil {
		s.panelApplied = layout
		s.panelGeometryPending = false
		return
	}
	s.panelApplied = layout // A synchronous WM_SIZE must not enqueue the same resize.
	size, err := s.resizePanel(layout, center)
	if err != nil {
		s.panelUnavailable = true
		layout = s.panelLayout()
		s.status = "工作区尺寸约束不可用；保持完整面板。"
		if s.win != nil {
			s.win.SetContentSize(layout.Width, layout.Height)
			if center {
				s.win.Center()
			}
			w, h := s.win.ContentSize()
			size = platform.PaletteSize{Width: float32(w), Height: float32(h)}
		} else {
			size = platform.PaletteSize{Width: float32(layout.Width), Height: float32(layout.Height)}
		}
	}
	s.panelApplied = layout
	s.panelActual = size
	s.panelGeometryPending = true // Clear only after the renderer observes size.
	s.presentation++              // Retire stale pointer/button geometry, never search key.
	s.menuAtRow = false
}
func (s *launcher) observePanelSize(width, height float32) {
	if s.resizePanel == nil {
		s.panelApplied = s.panelLayout()
		s.panelGeometryPending = false
		return
	}
	layout := s.panelLayout()
	minimumHeight := palette.CompactHeight
	if layout.ResultsVisible {
		minimumHeight = palette.Header + palette.Footer + palette.SectionHeight + 2*palette.Inset + palette.DetailRowHeight
	}
	if s.panelApplied == layout && width > 0 && height >= minimumHeight && math.Abs(float64(width-s.panelActual.Width)) < 1 && math.Abs(float64(height-s.panelActual.Height)) < 1 {
		s.panelGeometryPending = false
	}
}
func (s *launcher) expandPanelFirst() {
	if s.status == "请先输入或按下箭头展开可见结果。" || s.status == "请先展开可见结果，再打开入口菜单。" {
		s.status = ""
	}
	s.panelExpanded = true
	s.selected, s.lastScroll = 0, 0
	s.appNeedsSelection, s.clipNeedsSelection = false, false
	s.scroll.X, s.scroll.Y = 0, 0
	s.restoreSearch = true
	s.requestPanelLayout()
}
