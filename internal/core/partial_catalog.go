// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"path/filepath"
	"strings"
)

// PathInApplicationRoots is lexical only; platform adapters still validate
// volume/reparse ancestors before accessing a path. Removed configured roots
// cannot be resurrected from a prior partial snapshot.
func PathInApplicationRoots(path string, roots []string) bool {
	for _, root := range roots {
		relative, err := filepath.Rel(root, path)
		if err == nil && relative != ".." && !filepath.IsAbs(relative) && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

// RetainUnconfirmed keeps old identities within still-configured failed sources.
// For a fresh same-ID row, keep unknown duplicate source paths as provenance
// without replacing its current launch adapter. Missing IDs prove absence only
// of the old selected Path, never all sibling paths in that identity group.
// A successful complete source snapshot must not call this retention function.
func RetainUnconfirmed(fresh, previous []Application, missing map[string]bool, retain func(Application) bool, limit int) []Application {
	result := append([]Application(nil), fresh...)
	seen := map[string]int{}
	for i, app := range result {
		seen[app.ItemID()] = i
	}
	for _, old := range previous {
		id := old.ItemID() // Keep the group identity before substituting source paths.
		if index, found := seen[id]; found {
			if old.ShellID != "" || result[index].ShellID != "" {
				continue
			}
			// A full application budget cannot prevent this same-row metadata
			// update. Clone before append so neither input snapshot is modified.
			paths := append([]string(nil), result[index].SourcePaths...)
			for _, source := range old.SourcePaths {
				candidate := old
				candidate.Path = source
				if source == "" || missing[id] && pathKey(source) == pathKey(old.Path) || !retain(candidate) {
					continue
				}
				paths = appendUnique(paths, source)
			}
			result[index].SourcePaths = paths
			continue
		}
		if len(result) >= limit || missing[id] || !retain(old) {
			continue
		}
		// Retained rows also discard provenance from no-longer-configured
		// sources. Unknown paths remain metadata, never replacement adapters.
		app := old
		app.SourcePaths = nil
		for _, source := range old.SourcePaths {
			candidate := old
			candidate.Path = source
			if source != "" && retain(candidate) {
				app.SourcePaths = appendUnique(app.SourcePaths, source)
			}
		}
		seen[id] = len(result)
		result = append(result, app)
	}
	return result
}

// ExpandDuplicateSourceRoots keeps a root-only refresh complete for the affected
// duplicate groups. Paths are memory-only scanner output; only configured roots
// can be added, and the file adapter revalidates them before reading anything.
func ExpandDuplicateSourceRoots(requested, roots []string, previous []Application) []string {
	if len(requested) == 0 {
		return requested
	}
	included := make([]bool, len(roots))
	for i, root := range roots {
		for _, request := range requested {
			if strings.EqualFold(root, request) {
				included[i] = true
			}
		}
	}
	for pass := 0; pass < len(roots); pass++ {
		changed := false
		for _, app := range previous {
			if len(app.SourcePaths) < 2 {
				continue
			}
			members := make([]bool, len(roots))
			affected := false
			for i, root := range roots {
				for _, source := range app.SourcePaths {
					if PathInApplicationRoots(source, []string{root}) {
						members[i] = true
						affected = affected || included[i]
						break
					}
				}
			}
			if affected {
				for i := range roots {
					if members[i] && !included[i] {
						included[i], changed = true, true
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	result := make([]string, 0, len(roots))
	for i, root := range roots {
		if included[i] {
			result = append(result, root)
		}
	}
	if len(result) == 0 {
		return append([]string(nil), requested...)
	}
	return result
}
