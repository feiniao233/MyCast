// SPDX-License-Identifier: AGPL-3.0-or-later
// Package settings validates ordinary application preferences. Clipboard text
// and session recording state are deliberately absent from this schema.
package settings

import (
	"fmt"
	"github.com/feiniao233/MyCast/internal/core"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf8"
)

const SchemaVersion = 1
const MaxFileBytes = 128 * 1024
const DefaultHotkey = "Ctrl+Alt+Space"

type Settings struct {
	Startup    bool                `json:"startup,omitempty"`
	Hidden     []string            `json:"hidden,omitempty"`
	AppsFolder bool                `json:"apps_folder,omitempty"`
	Version    int                 `json:"version"`
	Hotkey     string              `json:"hotkey"`
	Theme      string              `json:"theme"`
	IndexRoots []string            `json:"index_roots,omitempty"`
	Favorites  []string            `json:"favorites,omitempty"`
	Recents    []string            `json:"recents,omitempty"`
	Aliases    map[string][]string `json:"aliases,omitempty"`
	QuickLinks []core.QuickLink    `json:"quick_links,omitempty"`
}

func Defaults() Settings {
	return Settings{Version: SchemaVersion, Hotkey: DefaultHotkey, Theme: "dark"}
}
func (s Settings) Clone() Settings {
	s.Hidden = append([]string(nil), s.Hidden...)
	s.IndexRoots = append([]string(nil), s.IndexRoots...)
	s.Favorites = append([]string(nil), s.Favorites...)
	s.Recents = append([]string(nil), s.Recents...)
	s.QuickLinks = append([]core.QuickLink(nil), s.QuickLinks...)
	if s.Aliases != nil {
		copied := make(map[string][]string, len(s.Aliases))
		for id, aliases := range s.Aliases {
			copied[id] = append([]string(nil), aliases...)
		}
		s.Aliases = copied
	}
	return s
}

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
	if s.Favorites, err = validateIDs(s.Favorites, 200); err != nil {
		return Settings{}, err
	}
	if s.Recents, err = validateIDs(s.Recents, 50); err != nil {
		return Settings{}, err
	}
	if s.Hidden, err = validateIDs(s.Hidden, 200); err != nil {
		return Settings{}, err
	}
	if len(s.Aliases) > 200 {
		return Settings{}, fmt.Errorf("too many application aliases")
	}
	if s.Aliases != nil {
		aliases := make(map[string][]string, len(s.Aliases))
		for id, values := range s.Aliases {
			if !validID(id) || len(values) > 8 {
				return Settings{}, fmt.Errorf("invalid application alias entry")
			}
			seen := map[string]bool{}
			for _, alias := range values {
				alias = strings.TrimSpace(alias)
				if alias == "" {
					continue
				}
				if len(alias) > 512 || utf8.RuneCountInString(alias) > 80 || !utf8.ValidString(alias) || strings.ContainsAny(alias, "\x00\r\n") {
					return Settings{}, fmt.Errorf("aliases require at most 80 characters each")
				}
				key := strings.ToLower(alias)
				if !seen[key] {
					seen[key] = true
					aliases[id] = append(aliases[id], alias)
				}
			}
		}
		s.Aliases = aliases
	}
	if len(s.QuickLinks) > 32 {
		return Settings{}, fmt.Errorf("too many quick links")
	}
	if s.QuickLinks != nil {
		links := make([]core.QuickLink, 0, len(s.QuickLinks))
		seen := map[string]bool{}
		for _, link := range s.QuickLinks {
			value, err := link.Validated()
			if err != nil {
				return Settings{}, err
			}
			if !seen[value.ItemID()] {
				seen[value.ItemID()] = true
				links = append(links, value)
			}
		}
		s.QuickLinks = links
	}
	return s, nil
}

func validID(id string) bool {
	if len(id) != 68 || !strings.HasPrefix(id, "app-") {
		return false
	}
	for _, r := range id[4:] {
		if !(r >= 'a' && r <= 'f' || r >= '0' && r <= '9') {
			return false
		}
	}
	return true
}
func validateIDs(ids []string, limit int) ([]string, error) {
	if len(ids) > limit {
		return nil, fmt.Errorf("too many application preferences")
	}
	var result []string
	seen := map[string]bool{}
	for _, id := range ids {
		if !validID(id) {
			return nil, fmt.Errorf("invalid application identity")
		}
		if !seen[id] {
			seen[id] = true
			result = append(result, id)
		}
	}
	return result, nil
}
