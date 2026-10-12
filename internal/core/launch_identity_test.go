// SPDX-License-Identifier: AGPL-3.0-or-later
package core

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestIndexMergesLaunchIdentityAndKeepsVariants(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "program", "Editor.exe")
	other := filepath.Join(root, "other", "Editor.exe")
	work := filepath.Join(root, "workspace")
	metadata := map[string]LaunchIdentity{
		"Editor.lnk":           {Target: target},
		"Editor duplicate.lnk": {Target: target},
		"Other Editor.lnk":     {Target: other},
		"Editor safe.lnk":      {Target: target, Arguments: "--safe"},
		"Editor caps.lnk":      {Target: target, Arguments: "--SAFE"},
		"Editor workspace.lnk": {Target: target, WorkingDirectory: work},
	}
	for name := range metadata {
		if err := os.WriteFile(filepath.Join(root, name), []byte("fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	resolver := func(path string) (LaunchIdentity, error) { return metadata[filepath.Base(path)], nil }
	first, problems := IndexApplicationsWithResolver([]string{root, root}, resolver)
	if len(problems) != 0 || len(first) != 5 {
		t.Fatalf("got %d applications, problems=%v", len(first), problems)
	}
	second, problems := IndexApplicationsWithResolver([]string{root}, resolver)
	if len(problems) != 0 || !reflect.DeepEqual(first, second) {
		t.Fatal("repeated scan changed index or retained duplicates")
	}
	ids := map[string]bool{}
	for _, app := range first {
		if ids[app.ItemID()] {
			t.Fatal("variants share UI identity")
		}
		ids[app.ItemID()] = true
		if filepath.Ext(app.Path) != ".lnk" {
			t.Fatal("original shortcut was not retained")
		}
	}
	if len(SearchApplications(first, "duplicate", 0)) != 1 {
		t.Fatal("merged shortcut alias is not searchable")
	}
}

func TestIndexUnresolvedLinksStayDistinct(t *testing.T) {
	root := t.TempDir()
	for _, name := range []string{"First.lnk", "Second.lnk"} {
		if err := os.WriteFile(filepath.Join(root, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	for _, resolver := range []func(string) (LaunchIdentity, error){
		func(string) (LaunchIdentity, error) { return LaunchIdentity{}, errors.New("unreadable") },
		func(string) (LaunchIdentity, error) { return LaunchIdentity{Target: "relative.exe"}, nil },
		func(string) (LaunchIdentity, error) {
			return LaunchIdentity{Target: filepath.Join(root, "target.exe"), WorkingDirectory: "relative"}, nil
		},
	} {
		apps, _ := IndexApplicationsWithResolver([]string{root}, resolver)
		if len(apps) != 2 {
			t.Fatal("unproven launch identities were merged")
		}
	}
}

func TestIdentityDistinguishesNamesakesAndExactArguments(t *testing.T) {
	root := t.TempDir()
	a := Application{Name: "Editor", Path: filepath.Join(root, "a.lnk"), Launch: LaunchIdentity{Target: filepath.Join(root, "Editor.exe"), Arguments: "--open  a"}}
	b := a
	b.Path = filepath.Join(root, "b.lnk")
	if a.ItemID() != b.ItemID() {
		t.Fatal("shortcut location changed launch identity")
	}
	b.Launch.Arguments = "--open a"
	if a.ItemID() == b.ItemID() || a.Subtitle() == b.Subtitle() {
		t.Fatal("different exact arguments are indistinguishable")
	}
	b = a
	b.Launch.Target = filepath.Join(root, "other", "Editor.exe")
	if len(SearchApplications([]Application{a, b}, "Editor", 0)) != 2 || a.Subtitle() == b.Subtitle() {
		t.Fatal("same-name applications were merged or not distinguishable")
	}
}
