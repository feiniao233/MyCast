// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"bytes"
	"encoding/json"
	"errors"
	"github.com/feiniao233/MyCast/internal/core"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func snapshotFixture(t *testing.T) (*SnapshotStore, Snapshot, time.Time) {
	t.Helper()
	now := time.Date(2026, 10, 13, 0, 0, 0, 0, time.UTC)
	store := &SnapshotStore{Path: filepath.Join(t.TempDir(), "startup-snapshot.json")}
	app := core.Application{Name: "Synthetic 显示", Path: "synthetic.exe", Launch: core.LaunchIdentity{Arguments: "must-not-persist", WorkingDirectory: "private-dir"}}
	return store, NewSnapshot([]core.Application{app}, SourceFingerprint([]string{"synthetic-root"}, nil, true), now), now
}

func TestSnapshotSafeRoundTripAndTransitions(t *testing.T) {
	store, value, now := snapshotFixture(t)
	if Defaults().StartupSnapshot {
		t.Fatal("snapshot enabled by default")
	}
	if _, err := store.Load(value.Fingerprint, now); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("missing snapshot", err)
	}
	if err := store.Save(value, true, store.Epoch(), now); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(store.Path)
	for _, forbidden := range []string{"must-not-persist", "private-dir", "synthetic.exe", "synthetic-root", "clipboard", "query"} {
		if bytes.Contains(data, []byte(forbidden)) {
			t.Fatal("extra data persisted", forbidden)
		}
	}
	if got, err := store.Load(value.Fingerprint, now); err != nil || len(got.Entries) != 1 {
		t.Fatal(got, err)
	}
	if _, err := store.Load(SourceFingerprint(nil, nil, false), now); err == nil {
		t.Fatal("disabled/changed sources reused snapshot")
	}
	partial := value
	partial.Entries = nil
	if err := store.Save(partial, false, store.Epoch(), now); err == nil {
		t.Fatal("partial scan overwritten complete cache")
	}
	epoch := store.Epoch()
	store.Invalidate()
	if err := store.Save(partial, true, epoch, now); err == nil {
		t.Fatal("old generation written")
	}
	current, _ := os.ReadFile(store.Path)
	if !bytes.Equal(current, data) {
		t.Fatal("old snapshot changed")
	}
	store.replace = func(string, string, bool) error { return errors.New("controlled failure") }
	if err := store.Save(partial, true, store.Epoch(), now); err == nil {
		t.Fatal("replacement failure ignored")
	}
	current, _ = os.ReadFile(store.Path)
	if !bytes.Equal(current, data) {
		t.Fatal("failed write changed original")
	}
	if matches, _ := filepath.Glob(filepath.Join(filepath.Dir(store.Path), ".snapshot-*.tmp")); len(matches) != 0 {
		t.Fatal("temporary file leaked")
	}
	if err := store.Clear(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(store.Path); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("clear", err)
	}
}

func TestSnapshotInvalidInputsPreserveOriginal(t *testing.T) {
	for _, kind := range []string{"oversized", "old-version", "corrupt", "unknown-path", "old-time", "future-time", "duplicate", "oversized-name", "invalid-id", "trailing", "invalid-utf8"} {
		t.Run(kind, func(t *testing.T) {
			store, value, now := snapshotFixture(t)
			bad := value
			switch kind {
			case "old-version":
				bad.Version = 73
			case "old-time":
				bad.Created = now.Add(-MaxSnapshotAge - time.Second)
			case "future-time":
				bad.Created = now.Add(time.Hour)
			case "duplicate":
				bad.Entries = append(bad.Entries, bad.Entries[0])
			case "oversized-name":
				bad.Entries[0].Name = strings.Repeat("a", 513)
			case "invalid-id":
				bad.Entries[0].ID = `C:\synthetic\evil.exe`
			}
			data, _ := json.Marshal(bad)
			switch kind {
			case "oversized":
				data = bytes.Repeat([]byte("a"), MaxSnapshotBytes+1)
			case "corrupt":
				data = []byte("{broken")
			case "unknown-path":
				data = bytes.Replace(data, []byte(`"name":`), []byte(`"path":"\\\\server\\share\\evil.exe","name":`), 1)
			case "trailing":
				data = append(data, []byte(" {}")...)
			case "invalid-utf8":
				data = []byte{0xff}
			}
			if err := os.WriteFile(store.Path, data, 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.Load(value.Fingerprint, now); err == nil {
				t.Fatal("invalid snapshot accepted")
			}
			// Unknown/corrupt formats cannot be silently overwritten. Supported
			// stale/invalid rows can be replaced only by a fresh complete snapshot.
			if kind == "old-version" || kind == "corrupt" || kind == "unknown-path" || kind == "oversized" || kind == "trailing" || kind == "invalid-utf8" {
				if err := store.Save(value, true, store.Epoch(), now); err == nil {
					t.Fatal("unknown original overwritten")
				}
				got, _ := os.ReadFile(store.Path)
				if !bytes.Equal(got, data) {
					t.Fatal("original changed")
				}
			}
			if err := store.Clear(); err != nil {
				t.Fatal("explicit clear failed", err)
			}
		})
	}
}

func TestSnapshotSettingPersistence(t *testing.T) {
	store := &Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	value, err := store.Load()
	if err != nil || value.StartupSnapshot {
		t.Fatal(value, err)
	}
	value.StartupSnapshot = true
	if err = store.Save(value); err != nil {
		t.Fatal(err)
	}
	restarted := &Store{Path: store.Path}
	value, err = restarted.Load()
	if err != nil || !value.StartupSnapshot {
		t.Fatal("explicit opt in", err)
	}
}

func BenchmarkSnapshotFirstResults(b *testing.B) {
	now := time.Now()
	apps := make([]core.Application, 5000)
	for i := range apps {
		apps[i] = core.Application{Name: "Synthetic Display", Path: filepath.Join("fixture", time.Duration(i).String()+".exe")}
	}
	store := &SnapshotStore{Path: filepath.Join(b.TempDir(), "snapshot.json")}
	value := NewSnapshot(apps, SourceFingerprint(nil, nil, true), now)
	if err := store.Save(value, true, store.Epoch(), now); err != nil {
		b.Fatal(err)
	}
	b.ResetTimer()
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if _, err := store.Load(value.Fingerprint, now); err != nil {
			b.Fatal(err)
		}
	}
}
