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
	// DisplayOnlyID identifies an untrusted startup-cache row. It grants no
	// launch, source-path or provider association authority.
	DisplayOnlyID string
	// ShellPackaged is read-only evidence of a registered packaged AUMID.
	// Desktop namespace rows never qualify as supplemental applications.
	ShellPackaged bool
	// Retained only to recognize old in-memory desktop snapshots during migration.
	ShellDesktop bool
	// SourceName freezes the indexed name before display localization.
	SourceName string
	// SourcePriority is configured-root order, never asynchronous arrival order.
	SourcePriority int
	// SourcePaths retains bounded configured duplicate paths in memory so a
	// partial refresh can rediscover a surviving source after one is removed.
	SourcePaths []string
	// ProvedIDs are current scan evidence; persisted links are never added here.
	ProvedIDs []string
	// LegacyTargetIDs are the unambiguous metadata-based subset of ProvedIDs.
	LegacyTargetIDs []string
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
	// Derived field provenance, parallel to UserAliases; never persisted.
	UserAliasKinds []string
	// NamePaths retain indexed source paths only until localized display names
	// are prepared. They never participate in launch identity or persistence.
	NamePaths []string
}

// LaunchIdentity is shortcut metadata, never a command to execute. Arguments
// remain byte-for-byte intact: changing whitespace or case can change a launch.
type LaunchIdentity struct {
	AppUserModelID string
	// True only after a successful string/VT_EMPTY property read. Unknown is
	// not an empty AppID and cannot authorize historical identity migration.
	AppUserModelIDKnown bool
	// Legacy desktop-association evidence, used only while migrating old rows.
	DesktopBridge bool
	// An immediate AppsFolder PIDL child read from a shortcut. It joins a
	// packaged AUMID only when the package provider also proves registration.
	ShellTarget      string
	Target           string
	Arguments        string
	WorkingDirectory string
	// Independent marks special/unknown metadata for safe target validation and
	// historical migration. The explicit name-target dedup policy still applies;
	// no flags or show modes on the selected original shortcut are rewritten.
	Independent bool
	LinkFlags   uint32
	ShowCommand int32
}

func (a Application) Identity() string {
	if a.DisplayOnlyID != "" {
		return "display-only:" + a.DisplayOnlyID
	}
	if a.ShellID != "" {
		return "appsfolder:" + strings.ToLower(a.ShellID)
	}
	if a.ShellPackaged && a.Launch.ShellTarget != "" {
		return "appsfolder:" + strings.ToLower(a.Launch.ShellTarget)
	}
	if strings.EqualFold(filepath.Ext(a.Path), ".lnk") || strings.EqualFold(filepath.Ext(a.Path), ".exe") && a.Launch.Target != "" {
		name := a.SourceName
		if name == "" {
			name = a.Name
		}
		target := a.Launch.Target
		if target == "" {
			target = a.Path
		}
		// Product policy deliberately ignores arguments, working directory and
		// special flags. The chosen original .lnk is still launched unchanged.
		return "file-app:" + strings.ToLower(name) + "|" + strings.ToLower(target)
	}
	if a.Launch.Target == "" || a.Launch.Independent || strings.EqualFold(filepath.Ext(a.Path), ".lnk") && !a.Launch.AppUserModelIDKnown {
		return pathKey(a.Path)
	}
	identity := a.legacyTargetIdentity()
	if a.Launch.AppUserModelID != "" {
		identity += fmt.Sprintf("|app-id:%q", a.Launch.AppUserModelID)
	}
	return identity
}

func (a Application) legacyTargetIdentity() string {
	return fmt.Sprintf("%q|%q|%q", pathKey(a.Launch.Target), a.Launch.Arguments, pathKey(a.Launch.WorkingDirectory))
}

func (a Application) ItemID() string {
	if a.DisplayOnlyID != "" {
		return a.DisplayOnlyID
	}
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

// IndexApplicationsWithResolver deduplicates shortcuts by case-insensitive
// source name and target (or source path when unresolved). Arguments and working
// directory do not participate; the original preferred link is never rewritten.
// The resolver must only read metadata, never resolve, repair or run.
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
				applications = append(applications, Application{Name: name, SourceName: name, SourcePriority: rootIndex, Path: path, NamePaths: []string{path}, SourcePaths: []string{path}})
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
		resolved := make([]Application, 0, len(applications))
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
				if !launch.AppUserModelIDKnown {
					launch.Independent = true
				}
				if err != nil {
					addProblem(fmt.Errorf("shortcut %q: %w", app.Path, err))
				}
				// Incomplete or relative identities cannot safely prove equivalence.
				virtual := launch.ShellTarget != "" && !launch.Independent && launch.Arguments == "" && launch.WorkingDirectory == ""
				if _, err := ValidateShellApplication(launch.ShellTarget); err != nil {
					virtual = false
				}
				if err == nil && (virtual || filepath.IsAbs(launch.Target)) {
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
				app.Launch.ShowCommand = 1            // ShellExecute uses SW_SHOWNORMAL.
				app.Launch.AppUserModelIDKnown = true // Bare EXE has no shortcut AppID.
			}
			resolved = append(resolved, app)
		}
		applications = MergeProviderApplications(FilterUninstallEntries(resolved))
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
