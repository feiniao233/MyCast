// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestMetadataBoundedValidatedAndCloned(t *testing.T) {
	id := "app-" + strings.Repeat("a", 64)
	value := Defaults()
	value.Favorites = []string{id, id}
	value.Recents = []string{id}
	value.Aliases = map[string][]string{id: {" Fixture ", "fixture", "合成"}}
	got, err := value.Validated()
	if err != nil || len(got.Favorites) != 1 || len(got.Aliases[id]) != 2 {
		t.Fatal("metadata normalization failed")
	}
	clone := got.Clone()
	clone.Aliases[id][0] = "changed"
	clone.Favorites[0] = "changed"
	clone.Recents[0] = "changed"
	if got.Aliases[id][0] != "Fixture" || got.Favorites[0] != id || got.Recents[0] != id {
		t.Fatal("clone shared metadata backing storage")
	}
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(got); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: store.Path}).Load()
	if err != nil || !reflect.DeepEqual(got, loaded) {
		t.Fatal("metadata persistence failed")
	}
	for _, bad := range []Settings{
		func() Settings { v := Defaults(); v.Favorites = []string{"invalid"}; return v }(),
		func() Settings {
			v := Defaults()
			v.Aliases = map[string][]string{id: {strings.Repeat("字", 81)}}
			return v
		}(),
		func() Settings { v := Defaults(); v.Aliases = map[string][]string{id: {"a\nb"}}; return v }(),
		func() Settings {
			v := Defaults()
			for i := 0; i < 51; i++ {
				v.Recents = append(v.Recents, fmt.Sprintf("app-%064x", i))
			}
			return v
		}(),
	} {
		if _, err := bad.Validated(); err == nil {
			t.Fatal("invalid/unbounded metadata accepted")
		}
	}
}
