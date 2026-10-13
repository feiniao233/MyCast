// SPDX-License-Identifier: AGPL-3.0-or-later

// Package core provides platform-independent application search and ephemeral
// clipboard history. It does not launch programs or read the system clipboard.
package core

import (
	"context"
	"crypto/sha256"
	"errors"
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
	// Successful namespace counterpart, memory-only and without nested fallback.
	ShellFallback *Application
	// ShellID is populated only by the configured AppsFolder provider. Path is
	// empty for virtual items; file adapters must never consume ShellID.
	ShellID     string
	RelatedIDs  []string
	Name        string
	Path        string
	Launch      LaunchIdentity
	Aliases     []string
	SearchTerms []string
	UserAliases []string
	// NamePaths retain indexed source paths only until localized display names
	// are prepared. They never participate in launch identity or persistence.
	NamePaths []string
}

// LaunchIdentity is shortcut metadata, never a command to execute. Arguments
// remain byte-for-byte intact: changing whitespace or case can change a launch.
type LaunchIdentity struct {
	// Only a proved immediate AppsFolder PIDL child with unchanged default
	// shortcut semantics can populate this cross-provider identity bridge.
	ShellTarget      string
	Target           string
	Arguments        string
	WorkingDirectory string
	// Independent preserves the original link's identity when launch flags
	// or a non-default show mode have additional semantics we do not merge.
	Independent bool
	LinkFlags   uint32
	ShowCommand int32
}

func (a Application) Identity() string {
	if a.ShellID != "" {
		return "appsfolder:" + a.ShellID
	}
	if a.Launch.ShellTarget != "" && !a.Launch.Independent && a.Launch.Arguments == "" && a.Launch.WorkingDirectory == "" {
		return "appsfolder:" + a.Launch.ShellTarget
	}
	if a.Launch.Target == "" || a.Launch.Independent {
		return pathKey(a.Path)
	}
	return fmt.Sprintf("%q|%q|%q", pathKey(a.Launch.Target), a.Launch.Arguments, pathKey(a.Launch.WorkingDirectory))
}

func (a Application) ItemID() string {
	return fmt.Sprintf("app-%x", sha256.Sum256([]byte(a.Identity())))
}

// Subtitle distinguishes namesakes without changing how the original link runs.
func (a Application) Subtitle() string {
	if a.ShellID != "" {
		return "AppsFolder · " + strings.TrimPrefix(a.ShellID, AppsFolderParsingPrefix)
	}
	target := a.Launch.Target
	if target == "" {
		target = a.Path
	}
	if a.Launch.Arguments != "" {
		target += " · " + a.Launch.Arguments
	}
	if a.Launch.WorkingDirectory != "" {
		target += " · " + a.Launch.WorkingDirectory
	}
	if a.Launch.Independent {
		target += " · " + a.Path
	}
	return target
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
	return IndexApplicationsWithResolver(roots, nil)
}

// IndexApplicationsWithResolver additionally merges readable shortcuts that
// have the same target, arguments and working directory. Unresolved links stay
// distinct. The resolver must only read metadata, never resolve, repair or run.
func IndexApplicationsWithResolver(roots []string, resolve func(string) (LaunchIdentity, error)) (applications []Application, problems []error) {
	return IndexApplicationsWithOptions(context.Background(), roots, resolve, DefaultScanOptions())
}

func IndexApplicationsWithOptions(ctx context.Context, roots []string, resolve func(string) (LaunchIdentity, error), options ScanOptions) (applications []Application, problems []error) {
	if options.MaxEntries <= 0 || options.MaxApplications <= 0 || options.MaxDepth <= 0 {
		return nil, []error{ErrScanBudget}
	}
	visited := 0
	addProblem := func(err error) {
		if len(problems) < 128 {
			problems = append(problems, err)
		} else if errors.Is(err, ErrScanBudget) || errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
			problems[len(problems)-1] = err
		}
	}
	seen := make(map[string]bool)
	for rootIndex, root := range roots {
		if rootIndex >= 32 {
			addProblem(ErrScanBudget)
			break
		}
		if err := ctx.Err(); err != nil {
			addProblem(err)
			break
		}
		if root == "" {
			continue
		}
		absolute, err := filepath.Abs(root)
		if err != nil {
			addProblem(fmt.Errorf("index %q: %w", root, err))
			continue
		}
		rootOptions := options
		for _, shallow := range options.ShallowRoots {
			if pathKey(absolute) == pathKey(shallow) {
				rootOptions.shallow = true
			}
		}
		err = boundedWalk(ctx, absolute, rootOptions, &visited, 0, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				addProblem(fmt.Errorf("index %q: %w", path, walkErr))
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
				if len(applications) >= options.MaxApplications {
					return ErrScanBudget
				}
				seen[key] = true
				applications = append(applications, Application{Name: name, Path: path, NamePaths: []string{path}})
			}
			return nil
		})
		if err != nil {
			addProblem(fmt.Errorf("index %q: %w", absolute, err))
			if errors.Is(err, ErrScanBudget) || ctx.Err() != nil {
				break
			}
		}
	}
	sort.Slice(applications, func(i, j int) bool { return applicationLess(applications[i], applications[j]) })
	if resolve != nil {
		merged := make([]Application, 0, len(applications))
		identities := make(map[string]int)
		for _, app := range applications {
			if err := ctx.Err(); err != nil {
				addProblem(err)
				break
			}
			if options.AllowPath != nil {
				if err := options.AllowPath(app.Path); err != nil {
					addProblem(err)
					continue
				}
			}
			if strings.EqualFold(filepath.Ext(app.Path), ".lnk") {
				launch, err := resolve(app.Path)
				if err != nil {
					addProblem(fmt.Errorf("shortcut %q: %w", app.Path, err))
				}
				// Incomplete or relative identities cannot safely prove equivalence.
				virtual := launch.ShellTarget != "" && !launch.Independent && launch.Arguments == "" && launch.WorkingDirectory == ""
				if _, err := ValidateShellApplication(launch.ShellTarget); err != nil {
					virtual = false
				}
				if err == nil && (virtual || filepath.IsAbs(launch.Target) && (launch.WorkingDirectory == "" || filepath.IsAbs(launch.WorkingDirectory))) {
					if virtual {
						app.RelatedIDs = []string{app.ItemID()}
					}
					app.Launch = launch
					if virtual {
						app.Aliases = append(app.Aliases, SystemApplicationAliases(launch.ShellTarget)...)
					}
					if ctx.Err() != nil {
						addProblem(ctx.Err())
						break
					}
					if !launch.Independent && options.IncludeResolvedTarget != nil && !options.IncludeResolvedTarget(launch) {
						continue
					}
				}
			} else {
				app.Launch.Target = app.Path
				app.Launch.ShowCommand = 1 // ShellExecute uses SW_SHOWNORMAL.
			}
			key := app.Identity()
			if index, found := identities[key]; found {
				merged[index].RelatedIDs = append(merged[index].RelatedIDs, app.RelatedIDs...)
				sources := append(append([]string(nil), merged[index].NamePaths...), app.NamePaths...)
				merged[index].Aliases = append(merged[index].Aliases, app.Name)
				// Prefer a shortcut to a bare executable, retaining shell launch semantics.
				if strings.EqualFold(filepath.Ext(app.Path), ".lnk") && !strings.EqualFold(filepath.Ext(merged[index].Path), ".lnk") {
					app.Aliases = append(merged[index].Aliases, merged[index].Name)
					app.NamePaths = sources
					merged[index] = app
				} else {
					merged[index].NamePaths = sources
				}
				continue
			}
			identities[key] = len(merged)
			merged = append(merged, app)
		}
		applications = merged
		sort.Slice(applications, func(i, j int) bool { return applicationLess(applications[i], applications[j]) })
	}
	return applications, problems
}

// SearchApplications returns name matches in descending relevance: exact,
// prefix, contiguous substring, then subsequence. Within a group, fewer gaps,
// an earlier match, and a shorter name rank first. Ties use deterministic name
// and path ordering rather than the input order. Search uses Unicode simple
// case folding and rune positions, including non-Latin names. It does not
// transliterate names or perform accent stripping or Unicode normalization.
// Subsequence queries of at least four runes require matched-span density of
// at least one third; short initials/acronym queries retain their prior behavior.
//
// Leading and trailing query whitespace is ignored. An empty query lists all
// applications alphabetically. limit <= 0 means no limit. Input is not modified.
// Repeated applications with the same nonempty path are returned once; the
// highest-ranked matching entry is used if their names differ.
func SearchApplications(applications []Application, query string, limit int) []Application {
	return SearchApplicationsRanked(applications, query, limit, nil, nil)
}

// Ranking keeps better match classes ahead of preference boosts. Empty-query
// results put favorites first, then successful-launch recents, then name/path.
func SearchApplicationsRanked(applications []Application, query string, limit int, favorites, recents []string) []Application {
	favorite := map[string]bool{}
	for _, id := range favorites {
		favorite[id] = true
	}
	recent := map[string]int{}
	for i, id := range recents {
		recent[id] = len(recents) - i
	}
	queryRunes := foldedRunes(strings.TrimSpace(query))
	type result struct {
		id    string
		app   Application
		score matchScore
	}
	matches := make([]result, 0, len(applications))
	for _, application := range applications {
		for _, id := range application.RelatedIDs {
			if favorite[id] {
				favorite[application.ItemID()] = true
			}
			if recent[id] > recent[application.ItemID()] {
				recent[application.ItemID()] = recent[id]
			}
		}
		name := foldedRunes(application.Name)
		score, ok := scoreMatch(name, queryRunes)
		for _, alias := range application.Aliases {
			if candidate, matches := scoreMatch(foldedRunes(alias), queryRunes); matches && (!ok || candidate.less(score)) {
				score, ok = candidate, true
			}
		}
		for _, terms := range [][]string{application.SearchTerms, application.UserAliases} {
			for _, term := range terms {
				if candidate, matched := scoreMatch(foldedRunes(term), queryRunes); matched && (!ok || candidate.less(score)) {
					score, ok = candidate, true
				}
			}
		}
		if ok {
			matches = append(matches, result{app: application, score: score, id: application.ItemID()})
		}
	}
	sort.Slice(matches, func(i, j int) bool {
		a, b := matches[i], matches[j]
		if len(queryRunes) != 0 && a.score.class != b.score.class {
			return a.score.class < b.score.class
		}
		ai, bi := a.id, b.id
		if favorite[ai] != favorite[bi] {
			return favorite[ai]
		}
		if recent[ai] != recent[bi] {
			return recent[ai] > recent[bi]
		}
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
		if match.app.Path != "" || match.app.ShellID != "" {
			key := match.app.Identity()
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
		// Long queries should not assemble an unrelated name out of distant
		// characters (including polyphonic pinyin). Preserve short acronym/
		// initials queries; long subsequences must have at least 1/3 density.
		if len(query) >= 4 && score.class == 3 && score.gaps > 2*len(query) {
			continue
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
	if a.Path != b.Path {
		return a.Path < b.Path
	}
	return a.Identity() < b.Identity()
}

func pathKey(path string) string {
	path = filepath.Clean(path)
	if runtime.GOOS == "windows" {
		return string(foldedRunes(path))
	}
	return path
}
