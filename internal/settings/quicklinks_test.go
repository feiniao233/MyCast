// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

func TestQuickLinksBoundedPersistenceAndClone(t *testing.T) {
	value := Defaults()
	value.QuickLinks = []core.QuickLink{{Name: "Synthetic docs", URL: "https://example.com/docs"}}
	clone := value.Clone()
	clone.QuickLinks[0].Name = "changed"
	if value.QuickLinks[0].Name != "Synthetic docs" {
		t.Fatal("link clone shared storage")
	}
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil || len(loaded.QuickLinks) != 1 || loaded.QuickLinks[0].URL != value.QuickLinks[0].URL {
		t.Fatal("quick links did not round trip")
	}
	value.QuickLinks[0].URL = "file:///C:/Synthetic.exe"
	if _, err := value.Validated(); err == nil {
		t.Fatal("unsupported protocol persisted")
	}
	value.QuickLinks = nil
	for i := 0; i < 33; i++ {
		value.QuickLinks = append(value.QuickLinks, core.QuickLink{Name: fmt.Sprintf("Fixture %d", i), URL: "https://example.com"})
	}
	if _, err := value.Validated(); err == nil {
		t.Fatal("unbounded links accepted")
	}
}
