// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAppsFolderOptInAndLegacySettings(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	old := `{"version":1,"hotkey":"Ctrl+Alt+Space","theme":"dark"}`
	if err := os.WriteFile(store.Path, []byte(old), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := store.Load()
	if err != nil || !value.AppsFolder || !store.SourcesDefaulted {
		t.Fatal("legacy settings did not receive the authorized default source migration")
	}
	value.AppsFolder = true
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil || !loaded.AppsFolder {
		t.Fatal("source round trip")
	}
}

func TestExplicitAppsFolderDisabledSurvivesMigrationAndSave(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	data := `{"version":1,"hotkey":"Ctrl+Alt+Space","theme":"dark","apps_folder":false}`
	if err := os.WriteFile(store.Path, []byte(data), 0600); err != nil {
		t.Fatal(err)
	}
	value, err := store.Load()
	if err != nil || value.AppsFolder || store.SourcesDefaulted {
		t.Fatal("explicit source disable overridden")
	}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	reloaded, err := (&Store{Path: store.Path}).Load()
	if err != nil || reloaded.AppsFolder {
		t.Fatal("saved disable omitted")
	}
}
