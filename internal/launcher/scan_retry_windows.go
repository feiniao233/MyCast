//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"sort"
	"time"
)

func (s *launcher) stopSourceRetry() {
	s.retryEpoch++
	if s.retryTimer != nil {
		s.retryTimer.Stop()
		s.retryTimer = nil
	}
	s.incompleteRoots = nil
	s.retryAttempts = 0
}

func (s *launcher) recordFileScanCompletion(roots []string, partial bool) {
	if s.incompleteRoots == nil {
		s.incompleteRoots = map[string]bool{}
	}
	for _, root := range roots {
		if partial {
			s.incompleteRoots[root] = true
		} else {
			delete(s.incompleteRoots, root)
		}
	}
	if len(s.incompleteRoots) == 0 {
		if s.retryTimer != nil {
			s.retryTimer.Stop()
			s.retryTimer = nil
		}
		s.retryEpoch++
		s.retryAttempts = 0
		return
	}
	if s.quitting || s.retryTimer != nil || s.uiUpdate == nil {
		return
	}
	delay := s.retryDelay
	if delay <= 0 {
		delay = 30 * time.Second
	}
	shift := min(s.retryAttempts, 4)
	delay = min(delay*time.Duration(1<<shift), 5*time.Minute)
	s.retryAttempts++
	epoch := s.retryEpoch
	update := s.uiUpdate
	s.retryTimer = time.AfterFunc(delay, func() {
		update(func() {
			if s.quitting || s.retryEpoch != epoch {
				return
			}
			s.retryTimer = nil
			roots := make([]string, 0, len(s.incompleteRoots))
			for root := range s.incompleteRoots {
				roots = append(roots, root)
			}
			sort.Strings(roots)
			if len(roots) > 0 {
				s.automaticRefreshRoots(scanFiles, roots)
			}
		})
	})
}
