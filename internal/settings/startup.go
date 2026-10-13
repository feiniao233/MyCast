// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"errors"
	"fmt"
)

// StartupRegistration owns one exact per-user registration. Call Apply only
// after an explicit startup preference change, never during application startup.
type StartupRegistration struct {
	Command string
	Read    func() (string, bool, error)
	Write   func(string) error
	Delete  func() error
}

func (s *StartupRegistration) Apply(enabled bool, save func() error) error {
	if s == nil || s.Command == "" || s.Read == nil || s.Write == nil || s.Delete == nil || save == nil {
		return fmt.Errorf("startup registration unavailable")
	}
	old, present, err := s.Read()
	if err != nil {
		return err
	}
	if present && old != s.Command {
		return fmt.Errorf("startup value belongs to another command; it was preserved")
	}
	changed := enabled != present
	if changed {
		if enabled {
			err = s.Write(s.Command)
		} else {
			err = s.Delete()
		}
		if err != nil {
			return fmt.Errorf("startup registration failed: %w", err)
		}
	}
	if err := save(); err != nil {
		if !changed {
			return err
		}
		current, exists, readErr := s.Read()
		if readErr != nil || exists != enabled || exists && current != s.Command {
			return errors.Join(err, fmt.Errorf("startup rollback refused after external change"))
		}
		var rollback error
		if present {
			rollback = s.Write(old)
		} else {
			rollback = s.Delete()
		}
		if rollback != nil {
			return errors.Join(err, fmt.Errorf("startup rollback failed; inspect MyCast startup entry: %w", rollback))
		}
		return err
	}
	return nil
}
