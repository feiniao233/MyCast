// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"errors"
	"path/filepath"
	"testing"
)

func TestStartupExplicitTransitionsRollbackAndOwnership(t *testing.T) {
	for _, enabled := range []bool{false, true} {
		for _, failSave := range []bool{false, true} {
			oldPresent := !enabled
			present := oldPresent
			value := "owned"
			writes, deletes, saves := 0, 0, 0
			registration := &StartupRegistration{Command: "owned", Read: func() (string, bool, error) { return value, present, nil }, Write: func(v string) error { writes++; value = v; present = true; return nil }, Delete: func() error { deletes++; present = false; return nil }}
			err := registration.Apply(enabled, func() error {
				saves++
				if failSave {
					return errors.New("synthetic save refusal")
				}
				return nil
			})
			if failSave && (err == nil || present != oldPresent) || !failSave && (err != nil || present != enabled) || saves != 1 {
				t.Fatal("startup transition/rollback")
			}
			if failSave && writes+deletes != 2 || !failSave && writes+deletes != 1 {
				t.Fatal("unexpected registration writes")
			}
		}
	}
	registration := &StartupRegistration{Command: "owned", Read: func() (string, bool, error) { return "another", true, nil }, Write: func(string) error { t.Fatal("overwrote foreign startup command"); return nil }, Delete: func() error { t.Fatal("deleted foreign startup command"); return nil }}
	for _, enabled := range []bool{false, true} {
		if registration.Apply(enabled, func() error { t.Fatal("saved collision"); return nil }) == nil {
			t.Fatal("collision accepted")
		}
	}
}

func TestStartupRollbackExternalChangeAndRegistrationFailure(t *testing.T) {
	present := false
	value := "owned"
	writes := 0
	registration := &StartupRegistration{Command: "owned", Read: func() (string, bool, error) { return value, present, nil }, Write: func(v string) error { writes++; present = true; value = v; return nil }, Delete: func() error { t.Fatal("deleted externally replaced value"); return nil }}
	if registration.Apply(true, func() error { value = "external"; return errors.New("save failed") }) == nil || value != "external" || writes != 1 {
		t.Fatal("external rollback guard")
	}
	registration.Write = func(string) error { return errors.New("synthetic denied") }
	present = false
	if registration.Apply(true, func() error { t.Fatal("save after failed registration"); return nil }) == nil {
		t.Fatal("registration failure ignored")
	}
}

func TestStartupDefaultAndPersistence(t *testing.T) {
	value := Defaults()
	if value.Startup {
		t.Fatal("startup default enabled")
	}
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	value.Startup = true
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil || !loaded.Startup {
		t.Fatal("startup preference round trip")
	}
}
