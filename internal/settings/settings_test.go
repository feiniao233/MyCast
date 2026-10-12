// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestSettingsDefaultsValidationAndPrivacy(t *testing.T) {
	value := Defaults()
	value.Hotkey = "alt + control + f12"
	root := t.TempDir()
	value.IndexRoots = []string{root, root}
	got, err := value.Validated()
	if err != nil || got.Hotkey != "Ctrl+Alt+F12" || len(got.IndexRoots) != 1 {
		t.Fatalf("validation failed: %+v %v", got, err)
	}
	for _, bad := range []string{"A", "Ctrl+Ctrl+A", "Win+Space", "Ctrl+Alt+not-a-key", "Ctrl+F25", "Ctrl++"} {
		if _, err := NormalizeHotkey(bad); err == nil {
			t.Errorf("accepted invalid hotkey")
		}
	}
	for _, bad := range []Settings{{Version: 2, Hotkey: DefaultHotkey, Theme: "dark"}, {Version: 1, Hotkey: DefaultHotkey, Theme: "unknown"}, {Version: 1, Hotkey: DefaultHotkey, Theme: "dark", IndexRoots: []string{"relative"}}} {
		if _, err := bad.Validated(); err == nil {
			t.Fatal("invalid settings accepted")
		}
	}
	encoded, _ := json.Marshal(value)
	if strings.Contains(string(encoded), "clipboard") || strings.Contains(string(encoded), "recording") {
		t.Fatal("session/clipboard data leaked into settings schema")
	}
}

func TestStoreAtomicRoundTripAndExternalEditPreservation(t *testing.T) {
	path := filepath.Join(t.TempDir(), "app-data", "settings.json")
	store := &Store{Path: path}
	value, err := store.Load()
	if err != nil || !reflect.DeepEqual(value, Defaults()) {
		t.Fatal("missing config did not return defaults")
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("load created a settings file")
	}
	value.Theme = "light"
	if err := store.Save(value); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&Store{Path: path}).Load()
	if err != nil || loaded.Theme != "light" {
		t.Fatal("round trip failed")
	}
	value.Theme = "system"
	if err := store.Save(value); err != nil {
		t.Fatal("existing settings could not be replaced:", err)
	}
	loaded, err = (&Store{Path: path}).Load()
	if err != nil || loaded.Theme != "system" {
		t.Fatal("replacement did not round trip")
	}
	leftovers, _ := filepath.Glob(filepath.Join(filepath.Dir(path), ".settings-*.tmp"))
	if len(leftovers) != 0 {
		t.Fatal("temporary settings files remained")
	}
	external := []byte(`{"user_unrelated_note":"retain me"}`)
	if err := os.WriteFile(path, external, 0600); err != nil {
		t.Fatal(err)
	}
	if err := store.Save(Defaults()); err == nil {
		t.Fatal("external edit was overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(external) {
		t.Fatal("unrelated content changed")
	}
}

func TestCorruptOversizeAndUnknownConfigsFallBackWithoutOverwriting(t *testing.T) {
	for _, data := range [][]byte{[]byte("not json"), []byte(`{"version":99,"hotkey":"Ctrl+Alt+Space","theme":"dark"}`), []byte(`{"version":1,"hotkey":"Ctrl+Alt+Space","theme":"dark","clipboard":["synthetic"]}`), []byte(strings.Repeat("x", MaxFileBytes+1))} {
		path := filepath.Join(t.TempDir(), "settings.json")
		os.WriteFile(path, data, 0600)
		store := &Store{Path: path}
		value, err := store.Load()
		if err == nil || !reflect.DeepEqual(value, Defaults()) {
			t.Fatal("bad config did not fall back")
		}
		if err := store.Save(Defaults()); err == nil {
			t.Fatal("bad/newer config was silently replaced")
		}
		after, _ := os.ReadFile(path)
		if string(after) != string(data) {
			t.Fatal("original config changed")
		}
	}
}

func TestHotkeyChangeFailureAndSaveRollback(t *testing.T) {
	for _, fail := range []string{"registration", "save", "none"} {
		var calls []string
		controller := Shortcuts{Current: DefaultHotkey, Register: func(key string) error {
			calls = append(calls, "register:"+key)
			if fail == "registration" {
				return errors.New("busy")
			}
			return nil
		}, Unregister: func(key string) { calls = append(calls, "unregister:"+key) }}
		candidate := Defaults()
		candidate.Hotkey = "Ctrl+F12"
		_, err := controller.Apply(candidate, func(Settings) error {
			calls = append(calls, "save")
			if fail == "save" {
				return errors.New("disk unavailable")
			}
			return nil
		})
		if fail != "none" {
			if err == nil || controller.Current != DefaultHotkey {
				t.Fatal("failed change lost previous active shortcut")
			}
		} else {
			if err != nil || controller.Current != "Ctrl+F12" {
				t.Fatal("successful change did not activate")
			}
		}
		want := []string{"register:Ctrl+F12"}
		if fail == "save" {
			want = append(want, "save", "unregister:Ctrl+F12")
		}
		if fail == "none" {
			want = append(want, "save", "unregister:"+DefaultHotkey)
		}
		if !reflect.DeepEqual(calls, want) {
			t.Fatalf("unsafe transition order: %v", calls)
		}
	}
}

func TestInvalidUTF8SettingsPreserved(t *testing.T) {
	path := filepath.Join(t.TempDir(), "settings.json")
	data := append([]byte(`{"version":1,"hotkey":"Ctrl+Alt+Space","theme":"dark","index_roots":["C:\\`), 0xff)
	data = append(data, []byte(`"]}`)...)
	if err := os.WriteFile(path, data, 0600); err != nil {
		t.Fatal(err)
	}
	store := &Store{Path: path}
	value, err := store.Load()
	if err == nil || !reflect.DeepEqual(value, Defaults()) {
		t.Fatal("invalid UTF-8 did not use safe defaults")
	}
	if err := store.Save(Defaults()); err == nil {
		t.Fatal("invalid UTF-8 file overwritten")
	}
	after, _ := os.ReadFile(path)
	if string(after) != string(data) {
		t.Fatal("original invalid bytes changed")
	}
}
