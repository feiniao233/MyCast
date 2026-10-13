// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"path"
	"path/filepath"
	"slices"
	"strings"
)

func IsLocalDesktopExecutable(target string) bool {
	return len(target) > 3 && target[1] == ':' && (target[2] == '\\' || target[2] == '/') && ((target[0] >= 'A' && target[0] <= 'Z') || (target[0] >= 'a' && target[0] <= 'z')) && !strings.ContainsAny(target[2:], ":\x00") && strings.EqualFold(filepath.Ext(target), ".exe")
}

func defaultDesktopDirectory(target, directory string) bool {
	return IsLocalDesktopExecutable(target) && (directory == "" || desktopPathKey(directory) == path.Dir(desktopPathKey(target)))
}

// Shell paths use Windows grammar even in a non-Windows pure test harness.
func desktopPathKey(value string) string {
	return strings.ToLower(path.Clean(strings.ReplaceAll(value, `\`, "/")))
}

// Names never participate. Prefer an exact explicit AppID; without one require
// one unique namespace default for the same target. The empty/default EXE
// directory profile is normalized only here, keeping the original file adapter.
func associateDesktopApplications(apps []Application) []Application {
	result := append([]Application(nil), apps...)
	type candidate struct {
		app       *Application
		ambiguous bool
	}
	byTarget := map[string]candidate{}
	byExplicit := map[string]candidate{}
	add := func(table map[string]candidate, key string, app *Application) {
		old, exists := table[key]
		if exists && old.app.ShellID != app.ShellID {
			old.ambiguous = true
			table[key] = old
		} else if !exists {
			table[key] = candidate{app: app}
		}
	}
	for i := range apps {
		shell := &apps[i]
		if !shell.ShellDesktop || shell.ShellID == "" {
			continue
		}
		key := desktopPathKey(shell.Launch.Target)
		add(byTarget, key, shell)
		add(byExplicit, key+"\x00"+shell.Launch.AppUserModelID, shell)
	}
	for i := range result {
		app := &result[i]
		if app.ShellID != "" || app.Launch.ShellTarget != "" || app.Launch.Independent || app.Launch.Arguments != "" || !defaultDesktopDirectory(app.Launch.Target, app.Launch.WorkingDirectory) {
			continue
		}
		key := desktopPathKey(app.Launch.Target)
		match := byTarget[key]
		if app.Launch.AppUserModelID != "" {
			match = byExplicit[key+"\x00"+app.Launch.AppUserModelID]
		}
		if match.app == nil || match.ambiguous {
			continue
		}
		old := app.ItemID()
		app.RelatedIDs = append(append([]string(nil), app.RelatedIDs...), old)
		app.Launch.ShellTarget = match.app.ShellID
		app.Launch.DesktopBridge = true
	}
	return result
}

func CarryRelatedPreferenceIDs(apps, previous []Application) []Application {
	old := map[string][]string{}
	for _, app := range previous {
		old[app.ItemID()] = app.RelatedIDs
	}
	result := append([]Application(nil), apps...)
	for i := range result {
		result[i].RelatedIDs = append([]string(nil), result[i].RelatedIDs...)
		for _, id := range old[result[i].ItemID()] {
			if !slices.Contains(result[i].RelatedIDs, id) {
				result[i].RelatedIDs = append(result[i].RelatedIDs, id)
			}
		}
	}
	return result
}

func (a Application) PreferenceIDs() []string { return append([]string{a.ItemID()}, a.RelatedIDs...) }
func (a Application) HasPreferenceID(id string) bool {
	return a.ItemID() == id || slices.Contains(a.RelatedIDs, id)
}
func (a Application) Preferred(ids []string) bool {
	for _, id := range ids {
		if a.HasPreferenceID(id) {
			return true
		}
	}
	return false
}

// Equal stable launch identities only. A file entrance remains the preferred
// launch adapter; namespace names and old shortcut hashes remain searchable/bound.
func MergeProviderApplications(apps []Application) []Application {
	apps = associateDesktopApplications(apps)
	result := make([]Application, 0, len(apps))
	seen := map[string]int{}
	for _, app := range apps {
		key := app.Identity()
		if index, ok := seen[key]; ok {
			old := result[index]
			if old.ShellID != "" && app.Path != "" {
				result[index] = app
				app = old
			}
			preferred := &result[index]
			if preferred.Path != "" && app.ShellID != "" {
				fallback := app
				fallback.ShellFallback = nil
				preferred.ShellFallback = &fallback
			}
			for _, alias := range append([]string{app.Name}, app.Aliases...) {
				if alias != preferred.Name && !slices.Contains(preferred.Aliases, alias) {
					preferred.Aliases = append(preferred.Aliases, alias)
				}
			}
			for _, id := range app.PreferenceIDs() {
				if id != preferred.ItemID() && !slices.Contains(preferred.RelatedIDs, id) {
					preferred.RelatedIDs = append(preferred.RelatedIDs, id)
				}
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, app)
	}
	return result
}
