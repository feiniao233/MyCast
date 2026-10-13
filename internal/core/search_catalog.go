// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"slices"
	"sort"
	"strings"
)

// SearchCatalog owns query-independent data. Replace it after any application
// metadata change. Search always examines the entire catalog, never an earlier
// result set. It contains no platform access or persistence.
type SearchCatalog struct{ entries []preparedApplication }
type preparedApplication struct {
	app                     Application
	id, identity, orderName string
	fields                  []preparedField
}
type preparedField struct {
	kind, text string
	runes      []rune
}

// SearchEvidence is the actual winning field, including preference tie boosts.
// Empty input has no winning field. Text remains memory-only.
type SearchEvidence struct {
	Field, Text                string
	Class, Gaps, Start, Length int
	Favorite                   bool
	Recent                     int
}
type ApplicationMatch struct {
	Application Application
	Evidence    SearchEvidence
}

func cloneSearchApplication(a Application) Application {
	a.RelatedIDs = slices.Clone(a.RelatedIDs)
	a.Aliases = slices.Clone(a.Aliases)
	a.SearchTerms = slices.Clone(a.SearchTerms)
	a.UserAliases = slices.Clone(a.UserAliases)
	a.UserAliasKinds = slices.Clone(a.UserAliasKinds)
	a.NamePaths = slices.Clone(a.NamePaths)
	if a.ShellFallback != nil {
		fallback := *a.ShellFallback
		fallback.ShellFallback = nil
		fallback = cloneSearchApplication(fallback)
		a.ShellFallback = &fallback
	}
	return a
}

func NewSearchCatalog(apps []Application) *SearchCatalog {
	c := &SearchCatalog{entries: make([]preparedApplication, 0, len(apps))}
	for _, a := range apps {
		a = cloneSearchApplication(a)
		e := preparedApplication{app: a, id: a.ItemID(), identity: a.Identity(), orderName: string(foldedRunes(a.Name))}
		add := func(kind, text string) { e.fields = append(e.fields, preparedField{kind, text, foldedRunes(text)}) }
		add("name", a.Name)
		for _, text := range a.Aliases {
			add("alias", text)
		}
		for _, text := range a.SearchTerms {
			add("pinyin", text)
		}
		for i, text := range a.UserAliases {
			kind := "user-alias-or-pinyin"
			if i < len(a.UserAliasKinds) && (a.UserAliasKinds[i] == "user-alias" || a.UserAliasKinds[i] == "user-alias-pinyin") {
				kind = a.UserAliasKinds[i]
			}
			add(kind, text)
		}
		c.entries = append(c.entries, e)
	}
	return c
}

// Search preserves SearchApplicationsRanked's scores, identity deduplication and
// deterministic order. Hidden identities are checked before matching/boosting.
// Returned applications are independent copies; callers cannot mutate catalog.
func (c *SearchCatalog) Search(query string, limit int, favorites, recents, hidden []string) []ApplicationMatch {
	if c == nil {
		return nil
	}
	q := foldedRunes(strings.TrimSpace(query))
	favorite := map[string]bool{}
	for _, id := range favorites {
		favorite[id] = true
	}
	recent := map[string]int{}
	for i, id := range recents {
		recent[id] = len(recents) - i
	}
	type match struct {
		entry    int
		score    matchScore
		evidence SearchEvidence
	}
	matches := make([]match, 0, len(c.entries))
	// Propagate related IDs before sorting, as the original implementation does.
	for _, e := range c.entries {
		if e.app.Preferred(hidden) {
			continue
		}
		for _, id := range e.app.RelatedIDs {
			if favorite[id] {
				favorite[e.id] = true
			}
			if recent[id] > recent[e.id] {
				recent[e.id] = recent[id]
			}
		}
	}
	for i, e := range c.entries {
		if e.app.Preferred(hidden) {
			continue
		}
		var best matchScore
		var evidence SearchEvidence
		ok := false
		for _, f := range e.fields {
			if score, found := scoreMatch(f.runes, q); found && (!ok || score.less(best)) {
				best, ok = score, true
				evidence = SearchEvidence{Field: f.kind, Text: f.text, Class: score.class, Gaps: score.gaps, Start: score.start, Length: score.length}
			}
		}
		if ok {
			if len(q) == 0 {
				evidence = SearchEvidence{}
			}
			evidence.Favorite, evidence.Recent = favorite[e.id], recent[e.id]
			matches = append(matches, match{i, best, evidence})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		x, y := c.entries[a.entry], c.entries[b.entry]
		if len(q) != 0 && a.score.class != b.score.class {
			return a.score.class < b.score.class
		}
		if favorite[x.id] != favorite[y.id] {
			return favorite[x.id]
		}
		if recent[x.id] != recent[y.id] {
			return recent[x.id] > recent[y.id]
		}
		if len(q) != 0 && a.score != b.score {
			return a.score.less(b.score)
		}
		if x.orderName != y.orderName {
			return x.orderName < y.orderName
		}
		if x.app.Name != y.app.Name {
			return x.app.Name < y.app.Name
		}
		if x.app.Path != y.app.Path {
			return x.app.Path < y.app.Path
		}
		return x.identity < y.identity
	})
	seen := map[string]bool{}
	results := make([]ApplicationMatch, 0, len(matches))
	for _, m := range matches {
		e := c.entries[m.entry]
		if e.app.Path != "" || e.app.ShellID != "" {
			if seen[e.identity] {
				continue
			}
			seen[e.identity] = true
		}
		results = append(results, ApplicationMatch{cloneSearchApplication(e.app), m.evidence})
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}
