// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"unicode/utf8"
)

type Store struct {
	Path           string
	readOnly       bool
	loaded, exists bool
	digest         [32]byte
}

func DefaultPath() (string, error) {
	root, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "MyCast", "settings.json"), nil
}

func (s *Store) Load() (Settings, error) {
	info, err := os.Lstat(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		s.loaded = true
		s.exists = false
		return Defaults(), nil
	}
	if err != nil {
		return Defaults(), fmt.Errorf("settings cannot be accessed")
	}
	if !info.Mode().IsRegular() || info.Size() > MaxFileBytes {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings file is not a bounded regular file")
	}
	file, err := os.Open(s.Path)
	if err != nil {
		return Defaults(), fmt.Errorf("settings cannot be read")
	}
	defer file.Close()
	opened, err := file.Stat()
	if err != nil || !opened.Mode().IsRegular() || !os.SameFile(info, opened) {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings file changed while opening")
	}
	data, err := io.ReadAll(io.LimitReader(file, MaxFileBytes+1))
	if err != nil {
		return Defaults(), fmt.Errorf("settings cannot be read")
	}
	if len(data) > MaxFileBytes {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings file exceeds size limit")
	}
	if !utf8.Valid(data) {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings contain invalid text; original file preserved")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	var value Settings
	if err := decoder.Decode(&value); err != nil {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings are invalid; original file is preserved")
	}
	var extra any
	if err := decoder.Decode(&extra); err != io.EOF {
		s.readOnly = true
		return Defaults(), fmt.Errorf("settings contain trailing data")
	}
	value, err = value.Validated()
	if err != nil {
		s.readOnly = true
		return Defaults(), err
	}
	s.loaded, s.exists, s.digest = true, true, sha256.Sum256(data)
	return value, nil
}

func (s *Store) Save(value Settings) error {
	if !s.loaded && !s.readOnly {
		if _, err := s.Load(); err != nil {
			return err
		}
	}
	if s.readOnly {
		return fmt.Errorf("original settings need recovery before saving; file was not overwritten")
	}
	value, err := value.Validated()
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	if len(data) > MaxFileBytes {
		return fmt.Errorf("settings exceed size limit")
	}
	if err := s.checkUnchanged(); err != nil {
		return err
	}
	dir := filepath.Dir(s.Path)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("settings directory cannot be created")
	}
	file, err := os.CreateTemp(dir, ".settings-*.tmp")
	if err != nil {
		return fmt.Errorf("settings cannot be prepared")
	}
	name := file.Name()
	defer os.Remove(name)
	if _, err = file.Write(append(data, '\n')); err != nil {
		file.Close()
		return fmt.Errorf("settings write failed")
	}
	if err = file.Sync(); err != nil {
		file.Close()
		return fmt.Errorf("settings synchronization failed")
	}
	if err = file.Close(); err != nil {
		return fmt.Errorf("settings close failed")
	}
	if err := s.checkUnchanged(); err != nil {
		return err
	}
	if err = os.Rename(name, s.Path); err != nil {
		return fmt.Errorf("settings replacement failed; existing settings preserved")
	}
	s.exists, s.digest = true, sha256.Sum256(append(data, '\n'))
	return nil
}

// Detect external edits before preparation and replacement. This is a
// best-effort conflict check, not a cross-process atomic compare-and-swap.
func (s *Store) checkUnchanged() error {
	currentInfo, statErr := os.Lstat(s.Path)
	if s.exists {
		if statErr != nil || !currentInfo.Mode().IsRegular() || currentInfo.Size() > MaxFileBytes {
			return fmt.Errorf("settings changed externally; reload before saving")
		}
		currentFile, err := os.Open(s.Path)
		if err != nil {
			return fmt.Errorf("settings cannot be checked")
		}
		opened, err := currentFile.Stat()
		if err != nil || !opened.Mode().IsRegular() || !os.SameFile(currentInfo, opened) {
			currentFile.Close()
			return fmt.Errorf("settings changed while checking")
		}
		current, err := io.ReadAll(io.LimitReader(currentFile, MaxFileBytes+1))
		currentFile.Close()
		if err != nil || len(current) > MaxFileBytes || sha256.Sum256(current) != s.digest {
			return fmt.Errorf("settings changed externally; original file preserved")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return fmt.Errorf("settings appeared externally; original file preserved")
	}
	return nil
}
