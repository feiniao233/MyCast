//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"bytes"
	"context"
	"encoding/binary"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"unicode/utf16"
	"unicode/utf8"

	"github.com/feiniao233/MyCast/internal/core"
)

const localizedINIBytes = 64 * 1024
const localizedINIBudget = 4 * 1024 * 1024
const localizedDirectoryBudget = 256
const localizedResourceBudget = 256

type localizedNameReader struct {
	allow            func(string) error
	read             func(string) ([]byte, error)
	resource         func(string, uint32) (string, error)
	windowsDirectory string
	directories      map[string]map[string]string
	resources        map[string]string
	bytes            int
}

// LocalizedApplicationNames examines only directories of files already indexed.
// It never widens roots, resolves shortcuts, guesses brands or changes identities.
func LocalizedApplicationNames(ctx context.Context, apps []core.Application) []core.Application {
	r := localizedNameReader{allow: LocalAppPath, read: readLocalizedINI, resource: readLocalizedResource, windowsDirectory: localizedWindowsDirectory()}
	return r.apply(ctx, apps)
}

func (r *localizedNameReader) apply(ctx context.Context, apps []core.Application) []core.Application {
	r.directories = map[string]map[string]string{}
	r.resources = map[string]string{}
	r.bytes = 0
	result := append([]core.Application(nil), apps...)
	for i, app := range result {
		result[i].NamePaths = nil
		if ctx.Err() != nil {
			continue
		}
		paths := append([]string{app.Path}, app.NamePaths...)
		result[i] = core.WithNameSourceAliases(result[i], paths)
		for _, path := range paths {
			if ctx.Err() != nil {
				break
			}
			candidate := r.name(ctx, path)
			if core.ChineseDisplayName(candidate) {
				result[i] = core.PreferChineseDisplayName(result[i], candidate)
				break
			}
		}
		// A Chinese name from a proven duplicate shortcut is also preferable
		// to an English filename when no localized resource is available.
		if !core.ChineseDisplayName(result[i].Name) {
			for _, alias := range app.Aliases {
				if core.ChineseDisplayName(alias) {
					result[i] = core.PreferChineseDisplayName(result[i], alias)
					break
				}
			}
		}
	}
	return result
}

func (r *localizedNameReader) name(ctx context.Context, path string) string {
	if ctx.Err() != nil || r.allow(path) != nil {
		return ""
	}
	directory := filepath.Dir(path)
	key := strings.ToLower(directory)
	entries, found := r.directories[key]
	if !found {
		if len(r.directories) >= localizedDirectoryBudget || r.bytes > localizedINIBudget-(localizedINIBytes+1) {
			return ""
		}
		r.directories[key] = nil // Cache missing/invalid files as well.
		ini := filepath.Join(directory, "desktop.ini")
		if r.allow(ini) != nil || ctx.Err() != nil {
			return ""
		}
		data, err := r.read(ini)
		r.bytes += len(data)
		if err != nil || ctx.Err() != nil {
			return ""
		}
		if len(data) > localizedINIBytes || r.bytes > localizedINIBudget {
			return ""
		}
		entries, _ = parseLocalizedINI(data)
		r.directories[key] = entries
	}
	value := entries[strings.ToLower(filepath.Base(path))]
	if !strings.HasPrefix(value, "@") {
		return value
	}
	resourcePath, id, err := parseLocalizedResourceReference(value, r.windowsDirectory)
	if err != nil {
		return ""
	}
	resourceKey := strings.ToLower(resourcePath) + "," + strconv.FormatUint(uint64(id), 10)
	if cached, ok := r.resources[resourceKey]; ok {
		return cached
	}
	if len(r.resources) >= localizedResourceBudget {
		return ""
	}
	r.resources[resourceKey] = ""
	if ctx.Err() != nil || r.allow(resourcePath) != nil {
		return ""
	}
	name, err := r.resource(resourcePath, id)
	if err == nil && ctx.Err() == nil {
		r.resources[resourceKey] = name
		return name
	}
	return ""
}

func readLocalizedINI(path string) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > localizedINIBytes {
		return nil, errors.New("localized name metadata unavailable or oversized")
	}
	return io.ReadAll(io.LimitReader(file, localizedINIBytes+1))
}

func parseLocalizedINI(data []byte) (map[string]string, error) {
	bad := errors.New("invalid localized name metadata")
	if len(data) > localizedINIBytes {
		return nil, bad
	}
	var text string
	if bytes.HasPrefix(data, []byte{0xff, 0xfe}) {
		data = data[2:]
		if len(data)%2 != 0 {
			return nil, bad
		}
		units := make([]uint16, len(data)/2)
		for i := range units {
			units[i] = binary.LittleEndian.Uint16(data[i*2:])
		}
		for i := 0; i < len(units); i++ {
			if units[i] >= 0xd800 && units[i] <= 0xdbff {
				if i+1 >= len(units) || units[i+1] < 0xdc00 || units[i+1] > 0xdfff {
					return nil, bad
				}
				i++
			} else if units[i] >= 0xdc00 && units[i] <= 0xdfff {
				return nil, bad
			}
		}
		text = string(utf16.Decode(units))
	} else {
		data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})
		if !utf8.Valid(data) {
			return nil, bad
		}
		text = string(data)
	}
	if strings.ContainsRune(text, 0) {
		return nil, bad
	}
	entries := map[string]string{}
	inSection := false
	count := 0
	for _, line := range strings.Split(text, "\n") {
		if len(line) > 4096 {
			return nil, bad
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			inSection = strings.EqualFold(line, "[LocalizedFileNames]")
			continue
		}
		if !inSection || line == "" || strings.HasPrefix(line, ";") || strings.HasPrefix(line, "#") {
			continue
		}
		name, value, ok := strings.Cut(line, "=")
		name, value = strings.TrimSpace(name), strings.TrimSpace(value)
		if !ok || name == "" || strings.ContainsAny(name, `/\:`) || name == "." || name == ".." || len(name) > 1024 || len(value) > 2048 {
			continue
		}
		ext := filepath.Ext(name)
		if !strings.EqualFold(ext, ".lnk") && !strings.EqualFold(ext, ".exe") {
			continue
		}
		count++
		if count > 1024 {
			return nil, bad
		}
		entries[strings.ToLower(name)] = value
	}
	return entries, nil
}

func parseLocalizedResourceReference(value, windowsDirectory string) (string, uint32, error) {
	bad := errors.New("unsupported localized resource reference")
	if len(value) > 2048 || !strings.HasPrefix(value, "@") || strings.ContainsAny(value, "\x00\r\n") {
		return "", 0, bad
	}
	comma := strings.LastIndex(value, ",")
	if comma < 2 || !strings.HasPrefix(value[comma+1:], "-") {
		return "", 0, bad
	}
	id, err := strconv.ParseUint(value[comma+2:], 10, 16)
	if err != nil || id == 0 {
		return "", 0, bad
	}
	for _, digit := range value[comma+2:] {
		if digit < '0' || digit > '9' {
			return "", 0, bad
		}
	}
	path := strings.TrimSpace(value[1:comma])
	if strings.HasPrefix(path, "\"") && strings.HasSuffix(path, "\"") {
		path = path[1 : len(path)-1]
	}
	for _, prefix := range []string{"%SystemRoot%", "%windir%"} {
		if strings.HasPrefix(strings.ToLower(path), strings.ToLower(prefix)+`\`) {
			path = windowsDirectory + path[len(prefix):]
			break
		}
	}
	if strings.ContainsAny(path, "%\"\x00?*<>|") || !filepath.IsAbs(path) || strings.HasPrefix(path, `\\`) || len(filepath.VolumeName(path)) != 2 || strings.Contains(path[2:], ":") {
		return "", 0, bad
	}
	path = filepath.Clean(path)
	ext := filepath.Ext(path)
	if !strings.EqualFold(ext, ".dll") && !strings.EqualFold(ext, ".exe") {
		return "", 0, bad
	}
	return path, uint32(id), nil
}
