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
	if err != nil || value.AppsFolder {
		t.Fatal("legacy settings silently enabled source")
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
