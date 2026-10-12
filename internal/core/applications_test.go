// SPDX-License-Identifier: AGPL-3.0-or-later

package core

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestSearchRanking(t *testing.T) {
	applications := []Application{
		{Name: "a long b", Path: "scattered"},
		{Name: "xxab", Path: "substring-late"},
		{Name: "abacus", Path: "prefix-long"},
		{Name: "a-b", Path: "subsequence"},
		{Name: "cab", Path: "substring"},
		{Name: "AB", Path: "exact"},
		{Name: "abc", Path: "prefix"},
		{Name: "unrelated", Path: "unrelated"},
	}
	before := append([]Application(nil), applications...)
	results := SearchApplications(applications, " ab ", 0)
	want := []string{"exact", "prefix", "prefix-long", "substring", "substring-late", "subsequence", "scattered"}
	if got := appPaths(results); !reflect.DeepEqual(got, want) {
		t.Fatalf("ranking = %v, want %v", got, want)
	}
	if !reflect.DeepEqual(applications, before) {
		t.Fatal("search mutated its input")
	}
	if got := appPaths(SearchApplications(applications, "ab", 2)); !reflect.DeepEqual(got, want[:2]) {
		t.Errorf("limited results = %v, want %v", got, want[:2])
	}
	if got := SearchApplications(applications, "does not match", 0); len(got) != 0 {
		t.Errorf("unexpected matches: %v", got)
	}
}

func TestSearchUnicode(t *testing.T) {
	tests := []struct {
		name, query string
		match       bool
	}{
		{"微信", "微信", true},
		{"微型办公信箱", "微信", true},
		{"微信", "信微", false},
		{"微信", "微信箱", false},
		{"图像编辑器", "图编", true},
		{"ÉDITEUR", "éditeur", true},
		{"Σigma", "ςIGMA", true},
		{"Kelvin", "kelvin", true},
		{"工具🚀箱", "🚀箱", true},
		{"ÉDITEUR", "editeur", false},
	}
	for _, test := range tests {
		t.Run(test.name+"/"+test.query, func(t *testing.T) {
			got := SearchApplications([]Application{{Name: test.name, Path: "app"}}, test.query, 0)
			if (len(got) != 0) != test.match {
				t.Errorf("matched = %v, want %v", len(got) != 0, test.match)
			}
		})
	}
}

func TestSearchEmptyAndDeterministic(t *testing.T) {
	applications := []Application{
		{Name: "beta", Path: "b"},
		{Name: "Alpha", Path: "z"},
		{Name: "Alpha", Path: "a"},
		{Name: "alpha", Path: "c"},
		{Name: "微信", Path: "w"},
	}
	want := []string{"a", "z", "c", "b", "w"}
	for _, query := range []string{"", " \t\n "} {
		if got := appPaths(SearchApplications(applications, query, -1)); !reflect.DeepEqual(got, want) {
			t.Errorf("empty query %q order = %v, want %v", query, got, want)
		}
	}
	first := SearchApplications(applications, "a", 0)
	for left, right := 0, len(applications)-1; left < right; left, right = left+1, right-1 {
		applications[left], applications[right] = applications[right], applications[left]
	}
	second := SearchApplications(applications, "a", 0)
	if !reflect.DeepEqual(first, second) {
		t.Errorf("input order changed output: %v vs %v", first, second)
	}
	if got := SearchApplications(nil, "", 10); len(got) != 0 {
		t.Errorf("nil input returned %v", got)
	}
}

func TestSearchDeduplicatesPathsBeforeLimiting(t *testing.T) {
	applications := []Application{
		{Name: "Alpha", Path: "a"},
		{Name: "Alpha", Path: "a"},
		{Name: "Another", Path: filepath.Join("folder", "..", "a")},
		{Name: "Beta", Path: "b"},
	}
	if got := appPaths(SearchApplications(applications, "", 2)); !reflect.DeepEqual(got, []string{"a", "b"}) {
		t.Errorf("deduplicated paths = %v", got)
	}
	if got := SearchApplications([]Application{{Name: "Alpha"}, {Name: "Beta"}}, "", 0); len(got) != 2 {
		t.Errorf("empty paths incorrectly deduplicated: %v", got)
	}
}

func TestScoreSelectsBestOccurrence(t *testing.T) {
	score, ok := scoreMatch(foldedRunes("a------ab"), foldedRunes("ab"))
	if !ok || score.class != 2 || score.gaps != 0 || score.start != 7 {
		t.Fatalf("best score = %+v, matched %v", score, ok)
	}
}

func TestIndexApplications(t *testing.T) {
	root := t.TempDir()
	wantedFiles := []string{"微信.LNK", "Beta.exe", filepath.Join("nested", "Alpha.lnk"), filepath.Join("other", "Alpha.lnk")}
	for _, name := range append(append([]string{}, wantedFiles...), "Ignored.url", "readme.txt", "script.bat", ".exe") {
		path := filepath.Join(root, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("not executed"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	// A directory with an executable-looking name is not an application.
	if err := os.Mkdir(filepath.Join(root, "Directory.exe"), 0o700); err != nil {
		t.Fatal(err)
	}
	applications, problems := IndexApplications([]string{"", root, root, filepath.Join(root, "nested")})
	if len(problems) != 0 {
		t.Fatalf("unexpected index errors: %v", problems)
	}
	if len(applications) != len(wantedFiles) {
		t.Fatalf("indexed %d applications, want %d: %v", len(applications), len(wantedFiles), applications)
	}
	if applications[0].Name != "Alpha" || applications[1].Name != "Alpha" || applications[2].Name != "Beta" || applications[3].Name != "微信" {
		t.Errorf("unexpected index order/names: %v", applications)
	}
	for _, application := range applications {
		if !filepath.IsAbs(application.Path) {
			t.Errorf("path is not absolute: %q", application.Path)
		}
	}
	fileResults, fileProblems := IndexApplications([]string{filepath.Join(root, "Beta.exe")})
	if len(fileProblems) != 0 || len(fileResults) != 1 || fileResults[0].Name != "Beta" {
		t.Errorf("file root: results %v, problems %v", fileResults, fileProblems)
	}
	if empty, errs := IndexApplications(nil); len(empty) != 0 || len(errs) != 0 {
		t.Errorf("empty roots scanned files: %v, %v", empty, errs)
	}
}

func TestIndexSkipsSymbolicLinks(t *testing.T) {
	root, outside := t.TempDir(), t.TempDir()
	if err := os.WriteFile(filepath.Join(outside, "External.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "linked-directory")); err != nil {
		t.Skipf("cannot create test symlinks on this platform: %v", err)
	}
	if err := os.Symlink(filepath.Join(outside, "External.exe"), filepath.Join(root, "linked-file.exe")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(root, filepath.Join(root, "loop")); err != nil {
		t.Fatal(err)
	}
	applications, problems := IndexApplications([]string{root, filepath.Join(root, "linked-directory"), filepath.Join(root, "linked-file.exe")})
	if len(applications) != 0 || len(problems) != 0 {
		t.Errorf("symbolic links were followed: %v; %v", applications, problems)
	}
}

func TestIndexKeepsResultsAfterMissingRoot(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "Found.exe")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	applications, problems := IndexApplications([]string{filepath.Join(root, "missing"), root})
	if len(applications) != 1 || applications[0].Path != path {
		t.Errorf("accessible result lost: %v", applications)
	}
	if len(problems) != 1 || !errors.Is(problems[0], fs.ErrNotExist) {
		t.Errorf("missing root error = %v", problems)
	}
}

func TestIndexUnreadableDirectory(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("Unix permission-bit fixture is not applicable on Windows")
	}
	root := t.TempDir()
	locked := filepath.Join(root, "locked")
	if err := os.Mkdir(locked, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(locked, "Hidden.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "Visible.exe"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o700) })
	if _, err := os.ReadDir(locked); err == nil {
		t.Skip("process can read directories despite mode 000")
	}
	applications, problems := IndexApplications([]string{root})
	if len(applications) != 1 || applications[0].Name != "Visible" {
		t.Errorf("partial index = %v", applications)
	}
	if len(problems) != 1 || !errors.Is(problems[0], fs.ErrPermission) {
		t.Errorf("unreadable directory problems = %v", problems)
	}
}

func appPaths(applications []Application) []string {
	paths := make([]string, len(applications))
	for index, application := range applications {
		paths[index] = application.Path
	}
	return paths
}
