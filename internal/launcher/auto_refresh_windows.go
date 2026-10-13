//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"github.com/feiniao233/MyCast/internal/platform"
	"slices"
)

const (
	scanFiles uint8 = 1
	scanShell uint8 = 2
)

func (s *launcher) automaticRefresh(requests ...uint8) {
	sources := uint8(scanFiles | scanShell)
	if len(requests) > 0 {
		sources = requests[0]
	}
	s.automaticRefreshRoots(sources, nil)
}

func (s *launcher) automaticRefreshRoots(sources uint8, roots []string) {
	if sources == 0 {
		return
	}
	if s.quitting {
		return
	}
	if s.scanQueue.Running {
		s.autoDirty = true
		s.autoSources |= sources
		if sources&scanFiles != 0 {
			if len(roots) == 0 {
				s.autoAllFiles = true
				s.autoRoots = nil
			} else if !s.autoAllFiles {
				for _, root := range roots {
					if !slices.Contains(s.autoRoots, root) {
						s.autoRoots = append(s.autoRoots, root)
					}
				}
			}
		}
		return
	}
	s.scanSourcesWithRoots(sources, roots)
}

func (s *launcher) restartSourceWatch() {
	s.stopSourceRetry()
	if s.watchCancel != nil {
		s.watchCancel()
	}
	s.watchEpoch++
	epoch := s.watchEpoch
	if s.uiUpdate == nil || s.quitting {
		return
	}
	ctx, cancel := context.WithCancel(context.Background())
	s.watchCancel = cancel
	roots := append([]string(nil), s.config.IndexRoots...)
	var shallow []string
	if len(roots) == 0 {
		roots = platform.DefaultAppRoots()
		shallow = platform.DefaultDesktopRoots()
	}
	update := s.uiUpdate
	shellEnabled := s.config.AppsFolder
	go platform.WatchApplicationSources(ctx, roots, shellEnabled, func(files, shell bool, roots []string) {
		sources := uint8(0)
		if files {
			sources |= scanFiles
		}
		if shell {
			sources |= scanShell
		}
		update(func() {
			if !s.quitting && s.watchEpoch == epoch {
				s.automaticRefreshRoots(sources, roots)
			}
		})
	}, shallow)
}

func (s *launcher) deliverSourceChange(epoch uint64, sources uint8) {
	if !s.quitting && s.watchEpoch == epoch {
		s.automaticRefresh(sources)
	}
}
