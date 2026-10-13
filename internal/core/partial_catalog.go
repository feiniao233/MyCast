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

// RetainUnconfirmed keeps only old identities within still-configured failed
// sources, excluding fresh identities and confirmed absence. A successful
// complete source snapshot needs no retention. Callers enforce the scan budget.
func RetainUnconfirmed(fresh, previous []Application, missing map[string]bool, retain func(Application) bool, limit int) []Application {
	result := append([]Application(nil), fresh...)
	seen := map[string]bool{}
	for _, app := range result {
		seen[app.ItemID()] = true
	}
	for _, app := range previous {
		id := app.ItemID()
		if len(result) >= limit {
			break
		}
		if !seen[id] && !missing[id] && retain(app) {
			result = append(result, app)
			seen[id] = true
		}
	}
	return result
}
