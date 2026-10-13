//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"github.com/feiniao233/MyCast/internal/platform"
)

func (s *launcher) automaticRefresh() {
	if s.quitting {
		return
	}
	if s.scanQueue.Running {
		s.autoDirty = true
		return
	}
	s.scan()
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
	go platform.WatchApplicationSources(ctx, roots, func() {
		update(func() {
			if !s.quitting && s.watchEpoch == epoch {
				s.automaticRefresh()
			}
		})
	})
}
