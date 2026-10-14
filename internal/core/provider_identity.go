// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"crypto/sha256"
	"fmt"
	"path"
	"path/filepath"
	"slices"
	"sort"
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

// FilterApplicationSources enforces the source policy again after partial-scan
// retention. A stale desktop namespace row or fallback must never reappear.
func FilterApplicationSources(apps []Application) []Application {
	result := make([]Application, 0, len(apps))
	for _, app := range apps {
		if app.ShellID != "" && !app.ShellPackaged {
			continue
		}
		app = cloneSearchApplication(app)
		if app.ShellFallback != nil && !app.ShellFallback.ShellPackaged {
			app.ShellFallback = nil
		}
		if app.Launch.DesktopBridge {
			// The old source policy proved this association while both sources
			// were present. Retain its preference hash, never its launch adapter.
			app.ProvedIDs = appendUnique(app.ProvedIDs, legacyApplicationID(app))
			app.LegacyTargetIDs = appendUnique(app.LegacyTargetIDs, legacyApplicationID(app))
			app.Launch.DesktopBridge = false
			app.Launch.ShellTarget = ""
		}
		result = append(result, app)
	}
	return result
}

// PrioritizeApplicationSources reapplies the full configured root order after a
// partial refresh (whose local root indexes may start at zero).
func PrioritizeApplicationSources(apps []Application, roots []string) []Application {
	result := append([]Application(nil), apps...)
	for i := range result {
		result[i].SourcePriority = len(roots)
		for rank, root := range roots {
			if PathInApplicationRoots(result[i].Path, []string{root}) {
				result[i].SourcePriority = rank
				break
			}
		}
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
		result[i].RelatedIDs = slices.Clone(result[i].RelatedIDs)
		for _, id := range old[result[i].ItemID()] {
			result[i].RelatedIDs = appendUnique(result[i].RelatedIDs, id)
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

func appendUnique(values []string, value string) []string {
	if value != "" && !slices.Contains(values, value) {
		return append(values, value)
	}
	return values
}

// MergeProviderApplications makes .lnk rows authoritative for desktop apps and
// supplements only verified packaged AUMIDs. Selection is deterministic: links,
// configured source order, then source path. It never rewrites a shortcut.
func MergeProviderApplications(apps []Application) []Application {
	apps = FilterApplicationSources(apps)
	packaged := map[string]bool{}
	for _, app := range apps {
		if app.ShellID != "" && app.ShellPackaged {
			packaged[strings.ToLower(app.ShellID)] = true
		}
	}
	for i := range apps {
		app := &apps[i]
		if app.SourceName == "" {
			app.SourceName = app.Name
		}
		// Only the immediate PIDL child read from the original link, together
		// with package registration proof, bridges a packaged shortcut. An
		// ordinary desktop AppID or matching executable is never sufficient.
		if app.ShellID == "" && packaged[strings.ToLower(app.Launch.ShellTarget)] {
			app.ProvedIDs = appendUnique(app.ProvedIDs, app.ItemID())
			app.ShellPackaged = true
		}
	}
	apps = attachLegacyTargetPreferenceIDs(apps)
	sort.Slice(apps, func(i, j int) bool { return preferredApplicationSource(apps[i], apps[j]) })
	result := make([]Application, 0, len(apps))
	seen := map[string]int{}
	for _, app := range apps {
		key := app.Identity()
		if index, ok := seen[key]; ok {
			preferred := &result[index]
			if preferred.Path != "" && app.ShellID != "" && app.ShellPackaged {
				fallback := app
				fallback.ShellFallback = nil
				preferred.ShellFallback = &fallback
			}
			for _, alias := range append([]string{app.Name}, app.Aliases...) {
				if alias != preferred.Name {
					preferred.Aliases = appendUnique(preferred.Aliases, alias)
				}
			}
			for _, id := range app.PreferenceIDs() {
				if id != preferred.ItemID() {
					preferred.RelatedIDs = appendUnique(preferred.RelatedIDs, id)
				}
			}
			for _, id := range app.ProvedIDs {
				preferred.ProvedIDs = appendUnique(preferred.ProvedIDs, id)
			}
			for _, id := range app.LegacyTargetIDs {
				preferred.LegacyTargetIDs = appendUnique(preferred.LegacyTargetIDs, id)
			}
			for _, source := range app.SourcePaths {
				preferred.SourcePaths = appendUnique(preferred.SourcePaths, source)
			}
			for _, source := range app.NamePaths {
				preferred.NamePaths = appendUnique(preferred.NamePaths, source)
			}
			continue
		}
		seen[key] = len(result)
		result = append(result, app)
	}
	sort.Slice(result, func(i, j int) bool { return applicationLess(result[i], result[j]) })
	return result
}

func preferredApplicationSource(a, b Application) bool {
	rank := func(app Application) int {
		if strings.EqualFold(filepath.Ext(app.Path), ".lnk") {
			return 0
		}
		if app.Path != "" {
			return 1
		}
		return 2
	}
	if rank(a) != rank(b) {
		return rank(a) < rank(b)
	}
	if a.SourcePriority != b.SourcePriority {
		return a.SourcePriority < b.SourcePriority
	}
	if strings.ToLower(a.Path) != strings.ToLower(b.Path) {
		return strings.ToLower(a.Path) < strings.ToLower(b.Path)
	}
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	if a.ShellID != b.ShellID {
		return a.ShellID < b.ShellID
	}
	// The same source can appear in partial snapshots. Deterministic tie data
	// avoids goroutine arrival deciding which original metadata wins.
	return fmt.Sprintf("%#v", a.Launch) < fmt.Sprintf("%#v", b.Launch)
}

// legacyApplicationID implements the former identity solely for migration.
// New deduplication must never use argument or working-directory equivalence.
func legacyApplicationID(a Application) string {
	identity := ""
	switch {
	case a.DisplayOnlyID != "":
		return a.DisplayOnlyID
	case a.ShellID != "":
		identity = "appsfolder:" + a.ShellID
	case a.Launch.ShellTarget != "" && !a.Launch.Independent && a.Launch.AppUserModelIDKnown && (a.Launch.AppUserModelID == "" || a.Launch.AppUserModelID == strings.TrimPrefix(a.Launch.ShellTarget, AppsFolderParsingPrefix)) && a.Launch.Arguments == "" && (a.Launch.WorkingDirectory == "" || a.Launch.DesktopBridge && defaultDesktopDirectory(a.Launch.Target, a.Launch.WorkingDirectory)):
		identity = "appsfolder:" + a.Launch.ShellTarget
	case a.Launch.Target == "" || a.Launch.Independent || strings.EqualFold(filepath.Ext(a.Path), ".lnk") && !a.Launch.AppUserModelIDKnown:
		identity = pathKey(a.Path)
	default:
		identity = a.legacyTargetIdentity()
		if a.Launch.AppUserModelID != "" {
			identity += fmt.Sprintf("|app-id:%q", a.Launch.AppUserModelID)
		}
	}
	return fmt.Sprintf("app-%x", sha256.Sum256([]byte(identity)))
}

// Historical target hashes omitted names, then later included AppIDs. Attach
// only hashes owned by one new identity in the complete available file snapshot.
// Ambiguous historical preferences remain stored without guessing an owner.
func attachLegacyTargetPreferenceIDs(apps []Application) []Application {
	owners := map[string]string{}
	ambiguous := map[string]bool{}
	candidates := make([][]string, len(apps))
	targetIDs := make([][]string, len(apps))
	for i, app := range apps {
		candidates[i] = slices.Clone(app.ProvedIDs)
		targetIDs[i] = slices.Clone(app.LegacyTargetIDs)
		if app.DisplayOnlyID == "" {
			candidates[i] = appendUnique(candidates[i], legacyApplicationID(app))
			if app.ShellID != "" || app.Launch.Target != "" && app.Launch.AppUserModelIDKnown && !app.Launch.Independent {
				targetIDs[i] = appendUnique(targetIDs[i], legacyApplicationID(app))
			}
		}
		if app.Path != "" {
			candidates[i] = appendUnique(candidates[i], fmt.Sprintf("app-%x", sha256.Sum256([]byte(pathKey(app.Path)))))
		}
		if app.ShellID == "" && app.Launch.Target != "" && (!app.Launch.Independent || !app.Launch.AppUserModelIDKnown) {
			old := app
			old.Launch.AppUserModelID, old.Launch.ShellTarget = "", ""
			old.Launch.DesktopBridge = false
			old.Launch.AppUserModelIDKnown = true
			old.Launch.Independent = false
			id := legacyApplicationID(old)
			candidates[i] = appendUnique(candidates[i], id)
			if app.Launch.AppUserModelIDKnown {
				targetIDs[i] = appendUnique(targetIDs[i], id)
			}
			if !app.Launch.AppUserModelIDKnown {
				ambiguous[id] = true
			}
		}
		for _, id := range candidates[i] {
			if owner, exists := owners[id]; exists && owner != app.ItemID() {
				ambiguous[id] = true
			} else {
				owners[id] = app.ItemID()
			}
		}
	}
	result := append([]Application(nil), apps...)
	for i := range result {
		app := &result[i]
		app.RelatedIDs = slices.Clone(app.RelatedIDs)
		app.ProvedIDs = nil
		app.LegacyTargetIDs = nil
		for _, id := range candidates[i] {
			if ambiguous[id] {
				app.RelatedIDs = slices.DeleteFunc(app.RelatedIDs, func(value string) bool { return value == id })
				continue
			}
			if id != app.ItemID() {
				app.RelatedIDs = appendUnique(app.RelatedIDs, id)
				app.ProvedIDs = appendUnique(app.ProvedIDs, id)
				if slices.Contains(targetIDs[i], id) {
					app.LegacyTargetIDs = appendUnique(app.LegacyTargetIDs, id)
				}
			}
		}
	}
	return result
}
