// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import "slices"

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
