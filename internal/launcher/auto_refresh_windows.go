//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"github.com/feiniao233/MyCast/internal/platform"
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
	if sources == 0 {
		return
	}
	if s.quitting {
		return
	}
	if s.scanQueue.Running {
		s.autoDirty = true
		s.autoSources |= sources
		return
	}
	s.scanSources(sources)
}

func (s *launcher) restartSourceWatch() {
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
	if len(roots) == 0 {
		roots = platform.DefaultAppRoots()
	}
	update := s.uiUpdate
	shellEnabled := s.config.AppsFolder
	go platform.WatchApplicationSources(ctx, roots, shellEnabled, func(files, shell bool) {
		sources := uint8(0)
		if files {
			sources |= scanFiles
		}
		if shell {
			sources |= scanShell
		}
		update(func() {
			s.deliverSourceChange(epoch, sources)
		})
	})
}

func (s *launcher) deliverSourceChange(epoch uint64, sources uint8) {
	if !s.quitting && s.watchEpoch == epoch {
		s.automaticRefresh(sources)
	}
}
