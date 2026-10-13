// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"slices"

	"github.com/feiniao233/MyCast/internal/core"
)

// Persist only relations for identities actually referenced by ordinary user
// metadata. No names, paths, arguments, inventories or clipboard state enter it.
func (s Settings) BindProvedApplicationIdentities(apps []core.Application) (Settings, bool) {
	referenced := map[string]bool{}
	for _, ids := range [][]string{s.Favorites, s.Recents, s.Hidden} {
		for _, id := range ids {
			referenced[id] = true
		}
	}
	for id := range s.Aliases {
		referenced[id] = true
	}
	value := s.Clone()
	changed := false
	for old := range value.IdentityLinks {
		if !referenced[old] && !referenced[value.IdentityLinks[old]] {
			delete(value.IdentityLinks, old)
			changed = true
		}
	}
	for _, app := range apps {
		// Persist only provider-proved canonical associations, never relations
		// added solely to apply an already persisted preference in either direction.
		if app.ShellID == "" && app.Launch.ShellTarget == "" {
			continue
		}
		current := app.ItemID()
		for _, old := range app.RelatedIDs {
			if (!referenced[old] && !referenced[current]) || old == current {
				continue
			}
			// Existing evidence is not silently redirected if a path is repurposed.
			if _, exists := value.IdentityLinks[old]; exists {
				continue
			}
			if value.IdentityLinks == nil {
				value.IdentityLinks = map[string]string{}
			}
			value.IdentityLinks[old] = current
			changed = true
		}
	}
	return value, changed
}

func (s Settings) ApplyIdentityLinks(apps []core.Application) []core.Application {
	result := append([]core.Application(nil), apps...)
	for i := range result {
		result[i].RelatedIDs = append([]string(nil), result[i].RelatedIDs...)
		current := result[i].ItemID()
		result[i].RelatedIDs = slices.DeleteFunc(result[i].RelatedIDs, func(old string) bool { bound, exists := s.IdentityLinks[old]; return exists && bound != current })
		for old, bound := range s.IdentityLinks {
			if old == current && !result[i].HasPreferenceID(bound) {
				result[i].RelatedIDs = append(result[i].RelatedIDs, bound)
			}
			if bound == current && !result[i].HasPreferenceID(old) {
				result[i].RelatedIDs = append(result[i].RelatedIDs, old)
			}
		}
	}
	return result
}
