//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"fmt"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
)

func (s *launcher) statusMessage() string {
	if !s.scanning {
		return s.status
	}
	progress := s.scanProgress
	if progress == "" {
		progress = "正在扫描应用…"
	}
	if s.status != "" {
		progress += "\n" + s.status
	}
	return progress
}

func (s *launcher) completeScan(apps []core.Application, icons map[string]*ui.Bitmap, problems []error, workerErr error) {
	// Retain important/newer diagnostics without appending old scan summaries.
	if s.status != s.scanLastStatus {
		s.scanDiagnostic = s.status
	}
	cancelled := errors.Is(workerErr, context.Canceled)
	budget := errors.Is(workerErr, context.DeadlineExceeded)
	for _, err := range problems {
		cancelled = cancelled || errors.Is(err, context.Canceled)
		budget = budget || errors.Is(err, core.ErrScanBudget) || errors.Is(err, context.DeadlineExceeded)
	}
	summary := ""
	switch {
	case cancelled:
		summary = "扫描已取消，保留当前结果。"
	case len(apps) == 0 && len(problems) > 0 && !budget:
		summary = fmt.Sprintf("扫描失败：%d 个位置无法读取，保留当前结果。", len(problems))
	default:
		s.refreshApplications(core.WithUserAliases(apps, s.config.Aliases), icons)
		summary = fmt.Sprintf("扫描完成：索引 %d 个应用。", len(apps))
		if budget {
			summary = fmt.Sprintf("扫描达到数量或时限预算：%d 个应用；可缩小索引目录。", len(apps))
		} else if len(problems) > 0 {
			summary = fmt.Sprintf("扫描部分完成：%d 个应用，%d 个位置无法读取。", len(apps), len(problems))
		}
		if !s.clipboardMode && s.appNeedsSelection {
			summary += " 原应用已移除，请重新选择。"
		}
	}
	if s.scanDiagnostic != "" && s.scanDiagnostic != s.scanFailure {
		summary += "\n" + s.scanDiagnostic
	}
	if s.scanFailure != "" {
		summary += "\n" + s.scanFailure
		s.scanFailure = ""
	}
	s.status = summary
	s.scanLastStatus = summary
}

// Publication snapshots the CURRENT UI state, after any intervening scroll,
// query edit, navigation or mode switch. Selection and viewport are independent.
// Only explicit navigation calls reveal; a catalog refresh never does.
func (s *launcher) refreshApplications(apps []core.Application, icons map[string]*ui.Bitmap) {
	if s.clipboardMode {
		s.apps, s.icons = apps, icons
		return
	}
	oldApps := core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents)
	oldRows := s.rows(oldApps, nil)
	selectedID := ""
	oldSelected := s.selected
	if oldSelected >= 0 && oldSelected < len(oldRows) {
		selectedID = oldRows[oldSelected].key
	}
	anchor := -1
	visibleTop := s.scroll.Y + palette.Header + palette.Inset
	for i, row := range oldRows {
		if row.top+row.height > visibleTop {
			anchor = i
			break
		}
	}
	s.apps, s.icons = apps, icons
	newApps := core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents)
	newRows := s.rows(newApps, nil)
	positions := map[string]int{}
	for i, row := range newRows {
		positions[row.key] = i
	}
	if index, ok := positions[selectedID]; ok {
		s.selected = index
	} else {
		s.selected = max(0, min(oldSelected, len(newRows)-1))
		if selectedID != "" {
			s.appNeedsSelection = true
		}
	}
	s.lastScroll = s.selected
	if len(newRows) == 0 {
		s.scroll.Y = 0
		return
	}
	if anchor >= 0 {
		index, ok := positions[oldRows[anchor].key]
		// If the visible anchor disappeared, prefer the nearest old neighbor.
		for distance := 1; !ok && distance < len(oldRows); distance++ {
			for _, candidate := range []int{anchor + distance, anchor - distance} {
				if candidate >= 0 && candidate < len(oldRows) {
					if found, exists := positions[oldRows[candidate].key]; exists {
						index, ok = found, true
						break
					}
				}
			}
		}
		if !ok {
			index = min(anchor, len(newRows)-1)
		}
		offset := visibleTop - oldRows[anchor].top
		// Keep partial-row offsets meaningful when duplicate subtitles shrink.
		offset = min(offset, newRows[index].height-1)
		s.scroll.Y = newRows[index].top + offset - palette.Header - palette.Inset
	}
	viewport := s.viewportHeight
	if viewport <= 0 {
		viewport = palette.Height
	}
	last := newRows[len(newRows)-1]
	maximum := max(float32(0), last.top+last.height+palette.Footer+palette.Inset-viewport)
	s.scroll.Y = max(float32(0), min(s.scroll.Y, maximum))
}
