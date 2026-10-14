//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"fmt"
	"slices"
	"strings"

	"github.com/feiniao233/MyCast/internal/core"
)

const applicationResultLimit = 100

// One UI-thread-owned slot. Catalog replacement is explicit; preference/query
// dependencies are copied so in-place slice edits cannot leave stale results.
type applicationSearch struct {
	catalog                    *core.SearchCatalog
	first                      *core.Application
	pendingFirst               *core.Application
	pendingLength              int
	length                     int
	query                      string
	favorites, recents, hidden []string
	valid                      bool
	results                    []core.Application
	total                      int
	evidence                   map[string]core.SearchEvidence
}

func (s *launcher) invalidateApplicationSearch() { s.appSearch = applicationSearch{} }

func (s *launcher) applicationResults() []core.Application {
	m := &s.appSearch
	var first *core.Application
	if len(s.apps) > 0 {
		first = &s.apps[0]
	}
	var pendingFirst *core.Application
	if len(s.pendingApps) > 0 {
		pendingFirst = &s.pendingApps[0]
	}
	if m.catalog == nil || first != m.first || len(s.apps) != m.length || pendingFirst != m.pendingFirst || len(s.pendingApps) != m.pendingLength {
		apps := append(append([]core.Application(nil), s.apps...), s.pendingApps...)
		*m = applicationSearch{catalog: core.NewSearchCatalog(apps), first: first, length: len(s.apps), pendingFirst: pendingFirst, pendingLength: len(s.pendingApps)}
	}
	if !m.valid || m.query != s.query || !slices.Equal(m.favorites, s.config.Favorites) || !slices.Equal(m.recents, s.config.Recents) || !slices.Equal(m.hidden, s.config.Hidden) {
		matches, total := m.catalog.SearchWithCount(s.query, applicationResultLimit, s.config.Favorites, s.config.Recents, s.config.Hidden)
		m.total = total
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

// The result cap is a presentation limit, not the searchable catalog size.
// Favorites and recents affect ordering; hidden rows are excluded from total.
func (s *launcher) resultCountLabel(displayed int) string {
	searching := strings.TrimSpace(s.query) != ""
	if s.clipboardMode {
		if searching {
			return fmt.Sprintf("文本历史 · 匹配 %d 条", displayed)
		}
		return fmt.Sprintf("文本历史 · %d 条", displayed)
	}
	if searching {
		return fmt.Sprintf("搜索结果 · 显示 %d / 匹配 %d", displayed, s.appSearch.total)
	}
	return fmt.Sprintf("应用 · 显示 %d / 可搜索 %d", displayed, s.appSearch.total)
}
