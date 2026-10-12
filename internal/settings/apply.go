// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import "fmt"

// Shortcuts represents this application's own registrations, never global
// keyboard hooks. Current is the actually active key, not just a desired value.
type Shortcuts struct {
	Current    string
	Register   func(string) error
	Unregister func(string)
}

// Apply registers the candidate before removing the current key. Save failure
// rolls the candidate back; registration failure never persists an unusable key.
func (s *Shortcuts) Apply(candidate Settings, save func(Settings) error) (Settings, error) {
	value, err := candidate.Validated()
	if err != nil {
		return Settings{}, err
	}
	if s.Register == nil || s.Unregister == nil || save == nil {
		return Settings{}, fmt.Errorf("settings transition is unavailable")
	}
	changed := value.Hotkey != s.Current
	if changed {
		if err := s.Register(value.Hotkey); err != nil {
			return Settings{}, fmt.Errorf("hotkey is unavailable; previous registration retained")
		}
	}
	if err := save(value); err != nil {
		if changed {
			s.Unregister(value.Hotkey)
		}
		return Settings{}, err
	}
	if changed {
		if s.Current != "" {
			s.Unregister(s.Current)
		}
		s.Current = value.Hotkey
	}
	return value, nil
}
