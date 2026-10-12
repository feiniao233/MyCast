// SPDX-License-Identifier: AGPL-3.0-or-later
// Package settings validates ordinary application preferences. Clipboard text
// and session recording state are deliberately absent from this schema.
package settings

import (
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const SchemaVersion = 1
const MaxFileBytes = 128 * 1024
const DefaultHotkey = "Ctrl+Alt+Space"

type Settings struct {
	Version    int      `json:"version"`
	Hotkey     string   `json:"hotkey"`
	Theme      string   `json:"theme"`
	IndexRoots []string `json:"index_roots,omitempty"`
}

func Defaults() Settings {
	return Settings{Version: SchemaVersion, Hotkey: DefaultHotkey, Theme: "dark"}
}
func (s Settings) Clone() Settings { s.IndexRoots = append([]string(nil), s.IndexRoots...); return s }

func NormalizeHotkey(value string) (string, error) {
	if len(value) > 80 || !utf8.ValidString(value) {
		return "", fmt.Errorf("hotkey is invalid")
	}
	parts := strings.Split(strings.TrimSpace(value), "+")
	if len(parts) < 2 || len(parts) > 4 {
		return "", fmt.Errorf("use Ctrl/Alt/Shift plus one key")
	}
	mods := map[string]bool{}
	for _, part := range parts[:len(parts)-1] {
		var name string
		switch strings.ToLower(strings.TrimSpace(part)) {
		case "ctrl", "control":
			name = "Ctrl"
		case "alt":
			name = "Alt"
		case "shift":
			name = "Shift"
		default:
			return "", fmt.Errorf("unsupported hotkey modifier")
		}
		if mods[name] {
			return "", fmt.Errorf("duplicate hotkey modifier")
		}
		mods[name] = true
	}
	key := strings.ToUpper(strings.TrimSpace(parts[len(parts)-1]))
	if key == "SPACE" {
		key = "Space"
	} else if len(key) == 1 && (key[0] >= 'A' && key[0] <= 'Z' || key[0] >= '0' && key[0] <= '9') {
	} else if strings.HasPrefix(key, "F") {
		n, err := strconv.Atoi(key[1:])
		if err != nil || n < 1 || n > 24 || key != "F"+strconv.Itoa(n) {
			return "", fmt.Errorf("unsupported hotkey key")
		}
	} else {
		return "", fmt.Errorf("use a letter, digit, Space, or F1–F24")
	}
	var ordered []string
	for _, name := range []string{"Ctrl", "Alt", "Shift"} {
		if mods[name] {
			ordered = append(ordered, name)
		}
	}
	return strings.Join(append(ordered, key), "+"), nil
}

func (s Settings) Validated() (Settings, error) {
	if s.Version != SchemaVersion {
		return Settings{}, fmt.Errorf("unsupported settings version")
	}
	key, err := NormalizeHotkey(s.Hotkey)
	if err != nil {
		return Settings{}, err
	}
	s.Hotkey = key
	if s.Theme != "dark" && s.Theme != "light" && s.Theme != "system" {
		return Settings{}, fmt.Errorf("unsupported theme")
	}
	if len(s.IndexRoots) > 32 {
		return Settings{}, fmt.Errorf("too many application roots")
	}
	roots := make([]string, 0, len(s.IndexRoots))
	seen := map[string]bool{}
	for _, root := range s.IndexRoots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if len(root) > 4096 || !utf8.ValidString(root) || strings.ContainsAny(root, "\x00\r\n") || !filepath.IsAbs(root) || strings.HasPrefix(root, `\\`) || strings.HasPrefix(root, "//") {
			return Settings{}, fmt.Errorf("application roots must be bounded absolute local paths")
		}
		root = filepath.Clean(root)
		identity := strings.ToLower(root)
		if !seen[identity] {
			seen[identity] = true
			roots = append(roots, root)
		}
	}
	s.IndexRoots = roots
	return s, nil
}
