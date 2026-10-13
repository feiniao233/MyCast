// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"

	"github.com/feiniao233/MyCast/internal/core"
)

const SnapshotVersion = 1
const MaxSnapshotBytes = 1024 * 1024
const MaxSnapshotAge = 7 * 24 * time.Hour

// Snapshot deliberately has no paths, targets, launch arguments, clipboard,
// query history or identity-equivalence proofs. Rows have display authority only.
type SnapshotEntry struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}
type Snapshot struct {
	Version     int             `json:"version"`
	Fingerprint string          `json:"fingerprint"`
	Created     time.Time       `json:"created"`
	Entries     []SnapshotEntry `json:"entries"`
}

func SourceFingerprint(roots, shallow []string, shell bool) string {
	data, _ := json.Marshal(struct {
		Roots, Shallow []string
		Shell          bool
	}{roots, shallow, shell})
	return fmt.Sprintf("%x", sha256.Sum256(data))
}
func NewSnapshot(apps []core.Application, fingerprint string, now time.Time) Snapshot {
	s := Snapshot{Version: SnapshotVersion, Fingerprint: fingerprint, Created: now.UTC(), Entries: make([]SnapshotEntry, 0, len(apps))}
	for _, app := range apps {
		if app.DisplayOnlyID == "" {
			s.Entries = append(s.Entries, SnapshotEntry{app.ItemID(), app.Name})
		}
	}
	return s
}
func (s Snapshot) validated(now time.Time) error {
	if s.Version != SnapshotVersion {
		return fmt.Errorf("unsupported snapshot version")
	}
	if data, err := hex.DecodeString(s.Fingerprint); err != nil || len(data) != 32 {
		return fmt.Errorf("invalid snapshot fingerprint")
	}
	if s.Created.IsZero() || s.Created.After(now.Add(5*time.Minute)) || now.Sub(s.Created) > MaxSnapshotAge {
		return fmt.Errorf("snapshot is stale")
	}
	if len(s.Entries) > 5000 {
		return fmt.Errorf("too many snapshot entries")
	}
	seen := map[string]bool{}
	for _, e := range s.Entries {
		if !validID(e.ID) || seen[e.ID] || len(e.Name) == 0 || len(e.Name) > 512 || !utf8.ValidString(e.Name) || strings.ContainsAny(e.Name, "\x00\r\n") {
			return fmt.Errorf("invalid snapshot display metadata")
		}
		seen[e.ID] = true
	}
	return nil
}

// SnapshotStore serializes bounded I/O. Invalidate before source/setting/clear
// transitions. Epoch checks refuse queued old writes; fingerprint checks make
// a concurrently completed old replacement unusable for a different source set.
// Same-directory synchronized replacement has no copy/delete fallback and does
// not promise crash durability on arbitrary filesystems.
type SnapshotStore struct {
	Path    string
	mu      sync.Mutex
	epoch   atomic.Uint64
	replace func(string, string, bool) error // controlled failure fixture
}

func (s *SnapshotStore) Epoch() uint64 { return s.epoch.Load() }
func (s *SnapshotStore) Invalidate()   { s.epoch.Add(1) }

func (s *SnapshotStore) read() ([]byte, bool, error) {
	info, err := os.Lstat(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil, false, nil
	}
	if err != nil || !info.Mode().IsRegular() || info.Size() > MaxSnapshotBytes {
		return nil, true, fmt.Errorf("snapshot is not a bounded regular file")
	}
	f, err := os.Open(s.Path)
	if err != nil {
		return nil, true, fmt.Errorf("snapshot cannot be read")
	}
	defer f.Close()
	opened, err := f.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		return nil, true, fmt.Errorf("snapshot changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(f, MaxSnapshotBytes+1))
	if err != nil || len(data) > MaxSnapshotBytes {
		return nil, true, fmt.Errorf("snapshot exceeds read budget")
	}
	return data, true, nil
}
func decodeSnapshot(data []byte) (Snapshot, error) {
	if !utf8.Valid(data) {
		return Snapshot{}, fmt.Errorf("snapshot text is invalid")
	}
	d := json.NewDecoder(bytes.NewReader(data))
	d.DisallowUnknownFields()
	var value Snapshot
	if err := d.Decode(&value); err != nil {
		return Snapshot{}, fmt.Errorf("snapshot format is invalid; original retained")
	}
	var extra any
	if d.Decode(&extra) != io.EOF {
		return Snapshot{}, fmt.Errorf("snapshot has trailing data")
	}
	if value.Version != SnapshotVersion {
		return Snapshot{}, fmt.Errorf("unknown snapshot version; original retained")
	}
	return value, nil
}
func (s *SnapshotStore) Load(fingerprint string, now time.Time) (Snapshot, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	data, exists, err := s.read()
	if err != nil {
		return Snapshot{}, err
	}
	if !exists {
		return Snapshot{}, os.ErrNotExist
	}
	value, err := decodeSnapshot(data)
	if err != nil {
		return Snapshot{}, err
	}
	if err = value.validated(now); err != nil {
		return Snapshot{}, err
	}
	if value.Fingerprint != fingerprint {
		return Snapshot{}, fmt.Errorf("snapshot sources changed")
	}
	return value, nil
}
func (s *SnapshotStore) Save(value Snapshot, complete bool, epoch uint64, now time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !complete || epoch != s.Epoch() {
		return fmt.Errorf("incomplete or superseded snapshot was not saved")
	}
	if err := value.validated(now); err != nil {
		return err
	}
	data, err := json.Marshal(value)
	if err != nil || len(data) > MaxSnapshotBytes {
		return fmt.Errorf("snapshot exceeds write budget")
	}
	original, exists, err := s.read()
	if err != nil {
		return err
	}
	if exists {
		if _, err = decodeSnapshot(original); err != nil {
			return err
		}
	}
	if err = os.MkdirAll(filepath.Dir(s.Path), 0700); err != nil {
		return fmt.Errorf("snapshot directory unavailable")
	}
	f, err := os.CreateTemp(filepath.Dir(s.Path), ".snapshot-*.tmp")
	if err != nil {
		return fmt.Errorf("snapshot preparation failed")
	}
	name := f.Name()
	defer os.Remove(name)
	if _, err = f.Write(data); err != nil {
		f.Close()
		return fmt.Errorf("snapshot write failed")
	}
	if err = f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("snapshot synchronization failed")
	}
	if err = f.Close(); err != nil {
		return fmt.Errorf("snapshot close failed")
	}
	current, currentExists, err := s.read()
	if err != nil || currentExists != exists || !bytes.Equal(current, original) || epoch != s.Epoch() {
		return fmt.Errorf("snapshot changed; original retained")
	}
	replace := s.replace
	if replace == nil {
		replace = replaceFile
	}
	if err = replace(name, s.Path, exists); err != nil {
		return fmt.Errorf("snapshot replacement failed")
	}
	return nil
}
func (s *SnapshotStore) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	info, err := os.Lstat(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	if err != nil || !info.Mode().IsRegular() {
		return fmt.Errorf("snapshot is not an ordinary file")
	}
	if err = os.Remove(s.Path); err != nil {
		return fmt.Errorf("snapshot cannot be cleared")
	}
	return nil
}
