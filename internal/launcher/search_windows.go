//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/core"
	"slices"
)

// One UI-thread-owned slot. Catalog replacement is explicit; preference/query
// dependencies are copied so in-place slice edits cannot leave stale results.
type applicationSearch struct {
	catalog                    *core.SearchCatalog
	first                      *core.Application
	length                     int
	query                      string
	favorites, recents, hidden []string
	valid                      bool
	results                    []core.Application
	evidence                   map[string]core.SearchEvidence
}

func (s *launcher) invalidateApplicationSearch() { s.appSearch = applicationSearch{} }

func (s *launcher) applicationResults() []core.Application {
	m := &s.appSearch
	var first *core.Application
	if len(s.apps) > 0 {
		first = &s.apps[0]
	}
	if m.catalog == nil || first != m.first || len(s.apps) != m.length {
		*m = applicationSearch{catalog: core.NewSearchCatalog(s.apps), first: first, length: len(s.apps)}
	}
	if !m.valid || m.query != s.query || !slices.Equal(m.favorites, s.config.Favorites) || !slices.Equal(m.recents, s.config.Recents) || !slices.Equal(m.hidden, s.config.Hidden) {
		matches := m.catalog.Search(s.query, 100, s.config.Favorites, s.config.Recents, s.config.Hidden)
		m.results = make([]core.Application, 0, len(matches))
		m.evidence = make(map[string]core.SearchEvidence, len(matches))
		for _, match := range matches {
			m.results = append(m.results, match.Application)
			m.evidence[match.Application.ItemID()] = match.Evidence
		}
		m.query = s.query
		m.favorites = slices.Clone(s.config.Favorites)
		m.recents = slices.Clone(s.config.Recents)
		m.hidden = slices.Clone(s.config.Hidden)
		m.valid = true
	}
	return m.results
}
