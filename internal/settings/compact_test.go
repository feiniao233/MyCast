// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCompactPreferenceDefaultUpgradeAndExplicitSave(t *testing.T) {
	if Defaults().CompactOnShow {
		t.Fatal("existing full-panel habit changed")
	}
	path := filepath.Join(t.TempDir(), "settings.json")
	if err := os.WriteFile(path, []byte(`{"version":1,"hotkey":"Ctrl+Alt+Space","theme":"dark","apps_folder":false}`), 0600); err != nil {
		t.Fatal(err)
	}
	s := &Store{Path: path}
	value, err := s.Load()
	if err != nil || value.CompactOnShow || value.AppsFolder {
		t.Fatal("old configuration changed", value, err)
	}
	value.CompactOnShow = true
	if err = s.Save(value); err != nil {
		t.Fatal(err)
	}
	value, err = (&Store{Path: path}).Load()
	if err != nil || !value.CompactOnShow {
		t.Fatal("explicit opt-in lost", err)
	}
	value.CompactOnShow = false
	if err = (&Store{Path: path}).Save(value); err != nil {
		t.Fatal(err)
	}
	value, err = (&Store{Path: path}).Load()
	if err != nil || value.CompactOnShow {
		t.Fatal("opt-out lost", err)
	}
}
