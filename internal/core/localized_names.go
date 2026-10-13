// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"slices"
	"strings"
	"unicode"
	"unicode/utf8"
)

// ChineseDisplayName accepts bounded display metadata containing Han text.
// It is not a translator and does not change launch data.
func ChineseDisplayName(name string) bool {
	if !utf8.ValidString(name) || utf8.RuneCountInString(name) > 256 {
		return false
	}
	han := false
	for _, r := range name {
		if r == unicode.ReplacementChar || unicode.IsControl(r) || unicode.Is(unicode.Cf, r) {
			return false
		}
		han = han || unicode.Is(unicode.Han, r)
	}
	return han
}

func PreferChineseDisplayName(app Application, candidate string) Application {
	candidate = strings.TrimSpace(candidate)
	if !ChineseDisplayName(candidate) || candidate == app.Name {
		return app
	}
	aliases := slices.Clone(app.Aliases)
	if app.Name != "" && !slices.Contains(aliases, app.Name) {
		aliases = append(aliases, app.Name)
	}
	app.Name, app.Aliases = candidate, aliases
	return app
}

// WithNameSourceAliases keeps original basenames (with and without extension)
// searchable after display localization, including proven duplicate shortcuts.
func WithNameSourceAliases(app Application, paths []string) Application {
	aliases := slices.Clone(app.Aliases)
	seen := map[string]bool{app.Name: true}
	for _, alias := range aliases {
		seen[alias] = true
	}
	for _, path := range paths {
		name := path[strings.LastIndexAny(path, `/\`)+1:]
		stem := name
		if dot := strings.LastIndexByte(name, '.'); dot > 0 {
			stem = name[:dot]
		}
		for _, alias := range []string{name, stem} {
			if alias != "" && !seen[alias] {
				aliases = append(aliases, alias)
				seen[alias] = true
			}
		}
	}
	app.Aliases = aliases
	return app
}
