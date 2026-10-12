// SPDX-License-Identifier: AGPL-3.0-or-later

// Package core provides platform-independent application search and ephemeral
// clipboard history. It does not launch programs or read the system clipboard.
package core

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"unicode"
)

// Application is an indexed, local application or Windows shortcut.
type Application struct {
	Name string
	Path string
}

// IndexApplications recursively indexes .lnk and .exe files in the supplied
// roots. No implicit roots are scanned, and symbolic links are never followed
// or indexed. Roots can be files or directories. Overlapping roots and repeated
// file paths are deduplicated; different paths with the same name are retained.
// Returned paths are absolute. Results are sorted by name and then path.
//
// An unreadable or missing path is reported in problems without discarding
// results from accessible paths. An empty root is ignored. No file is executed,
// and shortcut targets are not resolved or inspected.
func IndexApplications(roots []string) (applications []Application, problems []error) {
	seen := make(map[string]bool)
	for _, root := range roots {
		if root == "" {
			continue
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			problems = append(problems, fmt.Errorf("index %q: %w", root, err))
			continue
		}
		err = filepath.WalkDir(absolute, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				problems = append(problems, fmt.Errorf("index %q: %w", path, walkErr))
				return nil
			}
			if !entry.Type().IsRegular() {
				return nil
			}
			ext := filepath.Ext(entry.Name())
			if !strings.EqualFold(ext, ".lnk") && !strings.EqualFold(ext, ".exe") {
				return nil
			}
			name := strings.TrimSuffix(entry.Name(), ext)
			if name == "" {
				return nil
			}
			key := pathKey(path)
			if !seen[key] {
				seen[key] = true
				applications = append(applications, Application{Name: name, Path: path})
			}
			return nil
		})
		if err != nil {
			problems = append(problems, fmt.Errorf("index %q: %w", absolute, err))
		}
	}
	sort.Slice(applications, func(i, j int) bool { return applicationLess(applications[i], applications[j]) })
	return applications, problems
}

// SearchApplications returns name matches in descending relevance: exact,
// prefix, contiguous substring, then subsequence. Within a group, fewer gaps,
// an earlier match, and a shorter name rank first. Ties use deterministic name
// and path ordering rather than the input order. Search uses Unicode simple
// case folding and rune positions, including non-Latin names. It does not
// transliterate names or perform accent stripping or Unicode normalization.
//
// Leading and trailing query whitespace is ignored. An empty query lists all
// applications alphabetically. limit <= 0 means no limit. Input is not modified.
// Repeated applications with the same nonempty path are returned once; the
// highest-ranked matching entry is used if their names differ.
func SearchApplications(applications []Application, query string, limit int) []Application {
	queryRunes := foldedRunes(strings.TrimSpace(query))
	type result struct {
		app   Application
		score matchScore
	}
	matches := make([]result, 0, len(applications))
	for _, application := range applications {
		name := foldedRunes(application.Name)
		score, ok := scoreMatch(name, queryRunes)
		if ok {
			matches = append(matches, result{app: application, score: score})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if len(queryRunes) != 0 && a.score != b.score {
			return a.score.less(b.score)
		}
		return applicationLess(a.app, b.app)
	})
	seen := make(map[string]bool)
	results := make([]Application, 0, len(matches))
	for _, match := range matches {
		// Empty paths are useful in callers' transient lists, and do not
		// establish identity between otherwise distinct applications.
		if match.app.Path != "" {
			key := pathKey(match.app.Path)
			if seen[key] {
				continue
			}
			seen[key] = true
		}
		results = append(results, match.app)
		if limit > 0 && len(results) >= limit {
			break
		}
	}
	return results
}

type matchScore struct {
	class  int
	gaps   int
	start  int
	length int
}

func (a matchScore) less(b matchScore) bool {
	if a.class != b.class {
		return a.class < b.class
	}
	if a.gaps != b.gaps {
		return a.gaps < b.gaps
	}
	if a.start != b.start {
		return a.start < b.start
	}
	return a.length < b.length
}

func scoreMatch(name, query []rune) (matchScore, bool) {
	if len(query) == 0 {
		return matchScore{}, true
	}
	if len(query) > len(name) {
		return matchScore{}, false
	}
	var best matchScore
	found := false
	// For each possible first rune, the earliest subsequent match has the
	// minimum span. Comparing those spans avoids preferring an early but
	// unnecessarily scattered match over a later contiguous one.
	for start := 0; start <= len(name)-len(query); start++ {
		if name[start] != query[0] {
			continue
		}
		q, end := 1, start
		for pos := start + 1; pos < len(name) && q < len(query); pos++ {
			if name[pos] == query[q] {
				q++
				end = pos
			}
		}
		if q != len(query) {
			continue
		}
		score := matchScore{class: 3, gaps: end - start + 1 - len(query), start: start, length: len(name)}
		if score.gaps == 0 {
			score.class = 2
			if start == 0 {
				score.class = 1
				if len(name) == len(query) {
					score.class = 0
				}
			}
		}
		if !found || score.less(best) {
			best, found = score, true
		}
	}
	return best, found
}

func foldedRunes(value string) []rune {
	runes := []rune(value)
	for i, r := range runes {
		// Use the smallest rune in the simple-fold cycle as a canonical
		// representation (including Greek sigma and the Kelvin sign).
		folded := r
		for next := unicode.SimpleFold(r); next != r; next = unicode.SimpleFold(next) {
			if next < folded {
				folded = next
			}
		}
		runes[i] = folded
	}
	return runes
}

func applicationLess(a, b Application) bool {
	af, bf := string(foldedRunes(a.Name)), string(foldedRunes(b.Name))
	if af != bf {
		return af < bf
	}
	if a.Name != b.Name {
		return a.Name < b.Name
	}
	return a.Path < b.Path
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return string(foldedRunes(path))
	}
	return path
}
