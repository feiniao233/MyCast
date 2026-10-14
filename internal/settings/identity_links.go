// SPDX-License-Identifier: AGPL-3.0-or-later
package settings

import (
	"maps"
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
	proved, legacyTargets, currentIDs := provedIdentityOwners(apps)

	// The former desktop merge bound an old target tuple to a Shell identity.
	// A uniquely proved tuple can now move that entire component to its .lnk
	// identity. A reused source path, or an applied saved preference, cannot.
	migrations := map[string]string{}
	conflicting := map[string]bool{}
	for old, bound := range s.IdentityLinks {
		current, ok := legacyTargets[old]
		if !ok || bound == current {
			continue
		}
		if currentIDs[bound] {
			conflicting[bound] = true
		}
		if other, exists := s.IdentityLinks[current]; exists && other != bound {
			conflicting[bound] = true
		}
		if other, exists := migrations[bound]; exists && other != current {
			conflicting[bound] = true
		}
		migrations[bound] = current
	}
	for bound := range conflicting {
		delete(migrations, bound)
	}

	links := map[string]string{}
	for old, bound := range s.IdentityLinks {
		if current, ok := migrations[bound]; ok {
			bound = current
		}
		if old != bound && (referenced[old] || referenced[bound]) {
			links[old] = bound
		}
	}
	for old, current := range migrations {
		if referenced[old] {
			links[old] = current
		}
	}

	boundIDs := map[string]bool{}
	for _, bound := range links {
		boundIDs[bound] = true
	}
	for old, current := range proved {
		if old == current || (!referenced[old] && !referenced[current]) {
			continue
		}
		// Existing evidence is never silently redirected. Only the explicit
		// target-tuple migration above may flatten an existing component.
		if _, exists := s.IdentityLinks[old]; exists {
			continue
		}
		if _, exists := links[old]; exists {
			continue
		}
		if _, nested := links[current]; nested || boundIDs[old] {
			continue
		}
		links[old] = current
		boundIDs[current] = true
	}

	// A favorite may have many duplicate source links. Keep saved settings
	// bounded, preferring directly referenced identities and existing links.
	keys := slices.Sorted(maps.Keys(links))
	slices.SortStableFunc(keys, func(a, b string) int {
		if referenced[a] != referenced[b] {
			if referenced[a] {
				return -1
			}
			return 1
		}
		_, savedA := s.IdentityLinks[a]
		_, savedB := s.IdentityLinks[b]
		if savedA != savedB {
			if savedA {
				return -1
			}
			return 1
		}
		return 0
	})
	for _, old := range keys[min(len(keys), 650):] {
		delete(links, old)
	}
	value := s.Clone()
	value.IdentityLinks = links
	return value, !maps.Equal(s.IdentityLinks, links)
}

// ProvedIDs contain only scan evidence. RelatedIDs also contain previously
// applied settings and must never authorize a new persisted association.
func provedIdentityOwners(apps []core.Application) (proved, legacyTargets map[string]string, currentIDs map[string]bool) {
	proved, legacyTargets, currentIDs = map[string]string{}, map[string]string{}, map[string]bool{}
	ambiguous := map[string]bool{}
	for _, app := range apps {
		if app.DisplayOnlyID != "" {
			continue
		}
		current := app.ItemID()
		currentIDs[current] = true
		for _, old := range app.ProvedIDs {
			if !validID(old) {
				continue
			}
			if owner, exists := proved[old]; exists && owner != current {
				ambiguous[old] = true
			}
			proved[old] = current
			if slices.Contains(app.LegacyTargetIDs, old) {
				legacyTargets[old] = current
			}
		}
	}
	for old, owner := range proved {
		if ambiguous[old] || currentIDs[old] && old != owner {
			delete(proved, old)
			delete(legacyTargets, old)
		}
	}
	return
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
