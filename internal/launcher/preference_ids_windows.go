//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

func (s *launcher) applicationPreferenceIDs(id string) []string {
	for _, app := range s.apps {
		if app.HasPreferenceID(id) {
			return app.PreferenceIDs()
		}
	}
	return []string{id}
}
