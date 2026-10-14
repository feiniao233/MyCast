// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"path/filepath"
	"strings"
	"unicode"
)

// Labels alone do not exclude ordinary tools such as Revo Uninstaller.
func IsUninstallEntry(app Application) bool {
	target := app.Launch.Target
	if target == "" && strings.EqualFold(filepath.Ext(app.Path), ".exe") {
		target = app.Path
	}
	base := strings.ToLower(target[strings.LastIndexAny(target, `/\`)+1:])
	stem := strings.TrimSuffix(base, filepath.Ext(base))
	actionName := uninstallActionName(app.Name)
	for _, alias := range app.Aliases {
		actionName = actionName || uninstallActionName(alias)
	}
	knownHelper := stem == "uninstall" || stem == "uninstaller" || stem == "uninst"
	if !knownHelper && strings.HasPrefix(stem, "unins") && len(stem) > 5 {
		knownHelper = true
		for _, r := range stem[5:] {
			knownHelper = knownHelper && r >= '0' && r <= '9'
		}
	}
	if actionName && knownHelper {
		return true
	}
	if !strings.EqualFold(filepath.Ext(target), ".exe") {
		return false
	}
	args, ok := windowsMetadataArgs(app.Launch.Arguments)
	if !ok {
		return false
	}
	for i, arg := range args {
		flag := strings.ToLower(arg)
		if flag == "--uninstall" || flag == "-uninstall" || flag == "/uninstall" {
			return true
		}
		if base == "msiexec.exe" {
			if (flag == "/x" || flag == "-x") && i+1 < len(args) && args[i+1] != "" {
				return true
			}
			if len(flag) > 2 && (strings.HasPrefix(flag, "/x{") || strings.HasPrefix(flag, "-x{")) && strings.HasSuffix(flag, "}") {
				return true
			}
		}
	}
	return false
}
func uninstallActionName(name string) bool {
	name = strings.TrimSpace(strings.ToLower(name))
	if strings.HasPrefix(name, "卸载") || strings.HasSuffix(name, "卸载") {
		return true
	}
	for _, word := range []string{"uninstall", "uninstaller"} {
		if name == word || strings.HasPrefix(name, word+" ") || strings.HasPrefix(name, word+":") || strings.HasSuffix(name, " "+word) {
			return true
		}
	}
	return false
}

// Bounded Windows quoting parser. Tokens are metadata and never executed.
func windowsMetadataArgs(value string) ([]string, bool) {
	if len(value) > 16384 || strings.ContainsRune(value, 0) {
		return nil, false
	}
	var args []string
	for i := 0; i < len(value); {
		for i < len(value) && unicode.IsSpace(rune(value[i])) {
			i++
		}
		if i == len(value) {
			break
		}
		var token strings.Builder
		quoted := false
		for i < len(value) {
			if !quoted && unicode.IsSpace(rune(value[i])) {
				break
			}
			slash := 0
			for i < len(value) && value[i] == '\\' {
				slash++
				i++
			}
			if i < len(value) && value[i] == '"' {
				token.WriteString(strings.Repeat(`\`, slash/2))
				if slash%2 == 1 {
					token.WriteByte('"')
				} else {
					quoted = !quoted
				}
				i++
				continue
			}
			token.WriteString(strings.Repeat(`\`, slash))
			if i == len(value) {
				break
			}
			token.WriteByte(value[i])
			i++
		}
		if quoted || len(args) >= 128 {
			return nil, false
		}
		args = append(args, token.String())
	}
	return args, true
}
func FilterUninstallEntries(apps []Application) []Application {
	filtered := make([]Application, 0, len(apps))
	for _, app := range apps {
		if !IsUninstallEntry(app) {
			filtered = append(filtered, app)
		}
	}
	return filtered
}
