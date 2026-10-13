//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"testing"
	"time"
)

func TestSnapshotDisplayCannotLaunchOrProveIdentity(t *testing.T) {
	verified := core.Application{Name: "Synthetic Verified", Path: "synthetic-fixture.exe"}
	value := settings.NewSnapshot([]core.Application{verified}, settings.SourceFingerprint(nil, nil, true), time.Now())
	pending := snapshotDisplayApplications(value, nil)
	s := &launcher{pendingApps: pending, selected: 0}
	called := 0
	s.launchApplication = func(string) error { called++; return nil }
	s.launchShellApplication = s.launchApplication
	s.showLocation = s.launchApplication
	s.activate(s.applicationResults(), nil, false)
	for _, action := range []string{"primary", "location", "favorite", "alias", "hide"} {
		s.runMenuTarget(verified.ItemID(), false, action)
	}
	if called != 0 || len(s.config.Favorites) != 0 || len(s.config.Hidden) != 0 || s.aliasID != "" {
		t.Fatal("pending row granted an effect")
	}
	if len(s.apps) != 0 || pending[0].Launch.Target != "" || pending[0].ShellID != "" || pending[0].Path != "" {
		t.Fatal("cached display became launch metadata")
	}
	if pending[0].ItemID() != verified.ItemID() || pending[0].Identity() == verified.Identity() {
		t.Fatal("stable UI key confused provider identity")
	}
	s.config = settings.Defaults()
	if _, changed := s.config.BindProvedApplicationIdentities(pending); changed {
		t.Fatal("cache granted identity proof")
	}
	s.selected = 0
	s.refreshApplications([]core.Application{verified}, nil)
	if len(s.pendingApps) != 0 || s.applicationResults()[s.selected].ItemID() != verified.ItemID() {
		t.Fatal("verified replacement lost restart selection")
	}
}

func TestSnapshotPartialAndCompleteRevalidation(t *testing.T) {
	apps := []core.Application{{Name: "A", Path: "synthetic-a.exe"}, {Name: "B", Path: "synthetic-b.exe"}}
	value := settings.NewSnapshot(apps, settings.SourceFingerprint(nil, nil, true), time.Now())
	s := &launcher{pendingApps: snapshotDisplayApplications(value, nil)}
	s.refreshApplications(apps[:1], nil)
	if len(s.pendingApps) != 1 || s.pendingApps[0].ItemID() != apps[1].ItemID() {
		t.Fatal("partial scan misclassified absence")
	}
	s.pendingComplete = true
	s.refreshApplications(apps[:1], nil)
	if len(s.pendingApps) != 0 {
		t.Fatal("complete discovery retained stale display")
	}
}
