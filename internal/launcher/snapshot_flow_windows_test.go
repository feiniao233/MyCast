//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
)

func TestSnapshotControlsAndPendingPreviewDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprint(scale), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			calls := 0
			s.applyPreferences = func(v settings.Settings) (settings.Settings, error) { calls++; return v, nil }
			tester := newPaletteTester(s, scale)
			s.openSettings()
			tester.Frame()
			if err := tester.Click("启动快照选项"); err != nil {
				t.Fatal(err)
			}
			if err := tester.Click("启动显示快照"); err != nil {
				t.Fatal(err)
			}
			if !s.settingsDraft.StartupSnapshot || s.config.StartupSnapshot || calls != 0 {
				t.Fatal("draft enabled persistence")
			}
			for _, label := range []string{"启动显示快照", "清除本地启动快照", "返回设置"} {
				box, ok := tester.Find(label)
				if !ok || box.Y < 18 || box.Y+box.H > palette.Height-18 {
					t.Fatal("clipped snapshot control", label, box)
				}
			}
			savePreview(t, tester, fmt.Sprintf("snapshot-settings-%03d", int(scale*100)))
			tester.Key(0, ui.KeyEscape)
			if s.snapshotOpen || !s.settingsOpen || !tester.Focused("快捷键设置") {
				t.Fatal("snapshot escape lost parent focus")
			}
			tester.Click("取消设置")
			if s.config.StartupSnapshot || calls != 0 {
				t.Fatal("cancel enabled snapshot")
			}
			s.openSettings()
			tester.Frame()
			tester.Click("启动快照选项")
			tester.Click("启动显示快照")
			tester.Click("返回设置")
			tester.Click("保存设置")
			if calls != 1 || !s.config.StartupSnapshot {
				t.Fatal("explicit save did not enable setting")
			}
			value := settings.NewSnapshot(s.apps, settings.SourceFingerprint(nil, nil, true), time.Now())
			s.apps = nil
			s.pendingApps = snapshotDisplayApplications(value, nil)
			s.invalidateApplicationSearch()
			tester.Frame()
			if !tester.HasText("本地显示快照；扫描验证前不能启动") {
				t.Fatal("pending state hidden")
			}
			savePreview(t, tester, fmt.Sprintf("snapshot-pending-%03d", int(scale*100)))
		})
	}
}

func TestSnapshotWorkerPublicationAndPartialProtection(t *testing.T) {
	for _, partial := range []bool{false, true} {
		t.Run(fmt.Sprint(partial), func(t *testing.T) {
			root := t.TempDir()
			store := &settings.SnapshotStore{Path: filepath.Join(root, "snapshot.json")}
			app := core.Application{Name: "Synthetic Cached", Path: filepath.Join(root, "Synthetic.exe")}
			fp := settings.SourceFingerprint([]string{root}, nil, false)
			before := settings.NewSnapshot([]core.Application{app}, fp, time.Now())
			if err := store.Save(before, true, store.Epoch(), time.Now()); err != nil {
				t.Fatal(err)
			}
			s := &launcher{config: settings.Defaults(), snapshotStore: store}
			s.config.IndexRoots = []string{root}
			s.config.AppsFolder = false
			s.config.StartupSnapshot = true
			callbacks := make(chan func(), 4)
			release := make(chan struct{})
			s.uiUpdate = func(fn func()) { callbacks <- fn }
			s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
				<-release
				if partial {
					return nil, nil, []error{errors.New("synthetic inaccessible source")}
				}
				app.Name = "Synthetic Verified"
				return []core.Application{app}, nil, nil
			}
			s.scan()
			awaitScanUI(t, callbacks)
			if len(s.apps) != 0 || len(s.pendingApps) != 1 {
				t.Fatal("startup display did not remain untrusted")
			}
			s.query = "Synthetic"
			s.selected = 0
			close(release)
			awaitScanUI(t, callbacks)
			if partial {
				if len(s.pendingApps) != 1 || len(s.apps) != 0 {
					t.Fatal("partial source treated pending row as verified/deleted")
				}
			} else {
				if len(s.pendingApps) != 0 || s.selectedApplicationID() != app.ItemID() || s.query != "Synthetic" {
					t.Fatal("verification disrupted state")
				}
				awaitScanUI(t, callbacks)
			}
			got, err := store.Load(fp, time.Now())
			if err != nil {
				t.Fatal(err)
			}
			want := "Synthetic Verified"
			if partial {
				want = "Synthetic Cached"
			}
			if got.Entries[0].Name != want {
				t.Fatal("snapshot completeness protection", got)
			}
		})
	}
}

func TestSnapshotLateLoadIsDiscarded(t *testing.T) {
	root := t.TempDir()
	store := &settings.SnapshotStore{Path: filepath.Join(root, "snapshot.json")}
	value := settings.NewSnapshot([]core.Application{{Name: "Synthetic Old", Path: "synthetic.exe"}}, settings.SourceFingerprint([]string{root}, nil, false), time.Now())
	if err := store.Save(value, true, store.Epoch(), time.Now()); err != nil {
		t.Fatal(err)
	}
	s := &launcher{config: settings.Defaults(), snapshotStore: store}
	s.config.IndexRoots = []string{root}
	s.config.AppsFolder = false
	s.config.StartupSnapshot = true
	callbacks := make(chan func(), 4)
	release := make(chan struct{})
	s.uiUpdate = func(fn func()) { callbacks <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		<-release
		return nil, nil, nil
	}
	s.scan()
	var loaded func()
	select {
	case loaded = <-callbacks:
	case <-time.After(3 * time.Second):
		t.Fatal("load callback missing")
	}
	store.Invalidate()
	s.config.StartupSnapshot = false
	loaded()
	if len(s.pendingApps) != 0 {
		t.Fatal("late disabled cache published")
	}
	close(release)
	awaitScanUI(t, callbacks)
}

func TestStartupSnapshotFirstResultsTiming(t *testing.T) {
	// Fresh test process; dedicated files only. OS cache state is uncontrolled.
	root := t.TempDir()
	for i := 0; i < 512; i++ {
		if err := os.WriteFile(filepath.Join(root, fmt.Sprintf("Synthetic 显示 %04d.exe", i)), nil, 0600); err != nil {
			t.Fatal(err)
		}
	}
	start := time.Now()
	apps, errs := core.IndexApplications([]string{root})
	apps = core.PrepareSearch(apps)
	live := core.NewSearchCatalog(apps).Search("", 100, nil, nil, nil)
	scanElapsed := time.Since(start)
	if len(errs) != 0 || len(live) != 100 {
		t.Fatal("fixture discovery", errs)
	}
	store := &settings.SnapshotStore{Path: filepath.Join(t.TempDir(), "snapshot.json")}
	fp := settings.SourceFingerprint([]string{root}, nil, false)
	if err := store.Save(settings.NewSnapshot(apps, fp, time.Now()), true, store.Epoch(), time.Now()); err != nil {
		t.Fatal(err)
	}
	start = time.Now()
	value, err := store.Load(fp, time.Now())
	if err != nil {
		t.Fatal(err)
	}
	pending := core.NewSearchCatalog(snapshotDisplayApplications(value, nil)).Search("", 100, nil, nil, nil)
	cacheElapsed := time.Since(start)
	if len(pending) != 100 || pending[0].Application.DisplayOnlyID == "" {
		t.Fatal("cache display")
	}
	t.Logf("512 synthetic files, first-result path: discovery=%s, display-only snapshot=%s; OS cache uncontrolled; excludes window/font startup", scanElapsed, cacheElapsed)
}
