//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestPreferenceMigrationSaveFailureDoesNotClaimPersistence(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	old := (core.Application{Path: `C:\Synthetic\Legacy.lnk`}).ItemID()
	shell := core.Application{Name: "Fixture", ShellPackaged: true, ShellID: core.AppsFolderParsingPrefix + "Synthetic!App", RelatedIDs: []string{old}, ProvedIDs: []string{old}}
	s.config.Favorites = []string{old}
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(s.config); err != nil {
		t.Fatal(err)
	}
	before, _ := os.ReadFile(store.Path)
	s.persistPreferences = func(settings.Settings) (settings.Settings, error) {
		return settings.Settings{}, errors.New("synthetic replacement failure")
	}
	live, warning := s.persistProvedIdentityLinks([]core.Application{shell})
	after, _ := os.ReadFile(store.Path)
	if warning == "" || len(s.config.IdentityLinks) != 0 || string(before) != string(after) || !live[0].Preferred(s.config.Favorites) {
		t.Fatal("failed migration changed disk/config or hid warning/live preference")
	}
}

func TestFirstShortcutFavoriteSaveImmediatelyPersistsProof(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	file := core.Application{Name: "Fixture Link", Path: `C:\Synthetic\Fixture.lnk`, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, WorkingDirectory: `C:\Synthetic`, AppUserModelIDKnown: true}}
	s.apps = core.MergeProviderApplications([]core.Application{file})
	if len(s.apps[0].ProvedIDs) == 0 {
		t.Fatal("historical shortcut proof missing")
	}
	old := s.apps[0].ProvedIDs[0]
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	s.persistPreferences = func(v settings.Settings) (settings.Settings, error) {
		if err := store.Save(v); err != nil {
			return settings.Settings{}, err
		}
		return v, nil
	}
	s.toggleFavorite(file.ItemID())
	loaded, err := (&settings.Store{Path: store.Path}).Load()
	if err != nil || loaded.IdentityLinks[old] != file.ItemID() {
		t.Fatal("first save omitted shortcut proof", err)
	}
	remaining := loaded.ApplyIdentityLinks([]core.Application{file})
	if !remaining[0].Preferred(loaded.Favorites) {
		t.Fatal("immediate restart lost favorite")
	}
}

func TestMergedFileDisappearancePreservesShellAndRestartBindings(t *testing.T) {
	for _, partialShell := range []bool{false, true} {
		t.Run(map[bool]string{false: "file-only", true: "partial-shell"}[partialShell], func(t *testing.T) {
			testMergedFileDisappearance(t, partialShell)
		})
	}
}

func testMergedFileDisappearance(t *testing.T, partialShell bool) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{`C:\Synthetic`}
	file := core.Application{Name: "Fixture Link", Path: `C:\Synthetic\Legacy.lnk`}
	legacy := file.ItemID()
	id := core.AppsFolderParsingPrefix + "Synthetic!App"
	file.Launch.ShellTarget = id
	file.Launch.AppUserModelIDKnown = true
	file.RelatedIDs = []string{legacy}
	shell := core.Application{Name: "Fixture Shell", ShellPackaged: true, ShellID: id}
	s.apps = core.MergeProviderApplications([]core.Application{file, shell})
	s.config.Favorites = []string{legacy}
	s.config.Hidden = []string{legacy}
	s.config.Aliases = map[string][]string{legacy: {"persisted-alias"}}
	store := &settings.Store{Path: filepath.Join(t.TempDir(), "settings.json")}
	if err := store.Save(s.config); err != nil {
		t.Fatal(err)
	}
	s.persistPreferences = func(v settings.Settings) (settings.Settings, error) {
		if err := store.Save(v); err != nil {
			return settings.Settings{}, err
		}
		return v, nil
	}
	bound, warning := s.persistProvedIdentityLinks(s.apps)
	if warning != "" {
		t.Fatal(warning)
	}
	s.apps = bound
	updates := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { updates <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		return nil, nil, nil
	}
	s.scanShellWork = func(context.Context) ([]core.Application, map[string]*ui.Bitmap, []error) {
		if partialShell {
			return nil, nil, []error{errors.New("synthetic unavailable Shell source")}
		}
		t.Error("file refresh unexpectedly rescanned Shell")
		return nil, nil, nil
	}
	if partialShell {
		s.automaticRefresh(scanFiles | scanShell)
	} else {
		s.automaticRefresh(scanFiles)
	}
	select {
	case fn := <-updates:
		fn()
	case <-time.After(time.Second):
		t.Fatal("refresh timeout")
	}
	if len(s.apps) != 1 || s.apps[0].ShellID != id || len(s.applicationResults()) != 0 {
		t.Fatal("file removal lost known Shell counterpart or revealed hidden row")
	}
	loaded, err := (&settings.Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	restarted := syntheticLauncher()
	restarted.config = loaded
	restarted.apps = core.WithUserAliases(loaded.ApplyIdentityLinks([]core.Application{shell}), loaded.Aliases)
	if len(restarted.applicationResults()) != 0 || !restarted.apps[0].Preferred(loaded.Favorites) || len(core.SearchApplications(restarted.apps, "persisted-alias", 10)) != 1 {
		t.Fatal("restart after source disappearance lost preferences")
	}
}
