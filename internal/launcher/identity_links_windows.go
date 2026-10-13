//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func (s *launcher) persistProvedIdentityLinks(apps []core.Application) ([]core.Application, string) {
	warning := ""
	candidate, changed := s.config.BindProvedApplicationIdentities(apps)
	if changed {
		var value settings.Settings
		var err error
		if s.persistPreferences != nil {
			value, err = s.persistPreferences(candidate)
		}
		if s.persistPreferences == nil || err != nil {
			warning = "旧入口偏好绑定尚未保存；重启前请保留原快捷方式，或先修复设置存储。"
		} else {
			s.config = value.Clone()
		}
	}
	return s.config.ApplyIdentityLinks(apps), warning
}
