//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package launcher

import (
	"context"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/egoist/mygo/ui"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/platform"
	"github.com/feiniao233/MyCast/internal/settings"
)

func awaitScanUI(t *testing.T, callbacks <-chan func()) {
	t.Helper()
	select {
	case callback := <-callbacks:
		callback()
	case <-time.After(3 * time.Second):
		t.Fatal("scan callback did not arrive")
	}
}

func visibleAnchor(s *launcher) paletteRow {
	rows := s.rows(core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents), nil)
	for _, row := range rows {
		if row.top+row.height > s.scroll.Y+palette.Header+palette.Inset {
			return row
		}
	}
	return paletteRow{}
}

func TestScanPublishesCurrentQuerySelectionAndViewportDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.config.IndexRoots = []string{t.TempDir()}
			s.status = "Dedicated previous status"
			callbacks := make(chan func(), 2)
			release := make(chan struct{})
			started := make(chan struct{}, 1)
			newApps := append([]core.Application{{Name: "Synthetic Ahead", Path: `C:\Synthetic\Ahead.exe`}}, s.apps...)
			s.uiUpdate = func(fn func()) { callbacks <- fn }
			s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
				started <- struct{}{}
				<-release
				return newApps, nil, nil
			}
			tester := newPaletteTester(s, scale)
			s.scan()
			<-started
			tester.Frame()
			if !s.scanning || !strings.HasPrefix(s.statusMessage(), "正在扫描应用…") || !tester.HasText(s.statusMessage()) {
				t.Fatal("scan feedback was hidden by old status")
			}
			tester.Type("Synthetic")
			tester.Scroll(450, 200, 0, 400)
			if s.query != "Synthetic" || s.scroll.Y <= 0 || !tester.Focused("搜索") {
				t.Fatal("intervening query/scroll did not apply")
			}
			anchor := visibleAnchor(s)
			screen := anchor.top - s.scroll.Y
			selected := s.selectedApplicationID()
			savePreview(t, tester, fmt.Sprintf("rescan-progress-%03d", int(scale*100)))
			close(release)
			awaitScanUI(t, callbacks)
			tester.Frame()
			if s.scanning || s.query != "Synthetic" || s.selectedApplicationID() != selected || !tester.Focused("搜索") || !strings.HasPrefix(s.status, "扫描完成：") {
				t.Fatal("publication changed live selection/query/focus or omitted completion")
			}
			rows := s.rows(core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents), nil)
			found := false
			for _, row := range rows {
				if row.key == anchor.key {
					found = true
					if math.Abs(float64(row.top-s.scroll.Y-screen)) > .01 {
						t.Fatal("visible anchor moved")
					}
				}
			}
			if !found || s.scroll.Y <= 0 {
				t.Fatal("anchor lost or reset to first screen")
			}
			position := s.scroll.Y
			tester.Frame()
			if s.scroll.Y != position {
				t.Fatal("next render revealed offscreen selection")
			}
			tester.Scroll(450, 200, 0, 40)
			if s.scroll.Y <= position {
				t.Fatal("subsequent wheel moved back to first screen")
			}
			savePreview(t, tester, fmt.Sprintf("rescan-complete-%03d", int(scale*100)))
		})
	}
}

func TestRefreshPreservesAnchorsAcrossHeightChangesAndDeletion(t *testing.T) {
	s := syntheticLauncher()
	s.query = "Synthetic"
	s.scroll.Y = 420
	s.selected = 0
	anchor := visibleAnchor(s)
	screen := anchor.top - s.scroll.Y
	duplicate := core.Application{Name: anchor.title, Path: `C:\Synthetic\0-Duplicate.exe`}
	next := append(append([]core.Application(nil), s.apps...), duplicate)
	s.refreshApplications(next, nil)
	rows := s.rows(core.SearchApplicationsRanked(s.apps, s.query, 100, s.config.Favorites, s.config.Recents), nil)
	for _, row := range rows {
		if row.key == anchor.key && (row.height != palette.DetailRowHeight || math.Abs(float64(row.top-s.scroll.Y-screen)) > .01) {
			t.Fatal("height-changing anchor moved")
		}
	}
	anchor = visibleAnchor(s)
	screen = anchor.top - s.scroll.Y
	oldSelected := s.selectedApplicationID()
	remaining := []core.Application{}
	for _, app := range s.apps {
		if app.ItemID() != anchor.key && app.ItemID() != oldSelected {
			remaining = append(remaining, app)
		}
	}
	s.refreshApplications(remaining, nil)
	if s.scroll.Y <= 0 || s.selected < 0 || s.selected >= len(remaining) || !s.appNeedsSelection {
		t.Fatal("deleted anchor/selection jumped top or left invalid selection")
	}
	if math.Abs(float64(visibleAnchor(s).top-s.scroll.Y-screen)) > float64(palette.DetailRowHeight) {
		t.Fatal("deleted anchor did not choose a nearby visible row")
	}
}

func TestAppRefreshLeavesClipboardViewportAndFocusAlone(t *testing.T) {
	s := syntheticLauncher()
	s.switchMode(true)
	for i := 0; i < 30; i++ {
		s.history.Add(fmt.Sprintf("Dedicated clip %02d", i))
	}
	tester := newPaletteTester(s, 1.5)
	tester.Scroll(450, 200, 0, 360)
	s.query = "Dedicated"
	tester.Frame()
	selected, selection, query, offset := s.selected, s.clipSelection, s.query, s.scroll.Y
	s.refreshApplications(nil, nil)
	tester.Frame()
	if !s.clipboardMode || s.selected != selected || s.clipSelection != selection || s.query != query || s.scroll.Y != offset || !tester.Focused("搜索") {
		t.Fatal("application catalog refreshed clipboard viewport/query/selection/focus")
	}
}

func TestScanCompletionUsesModeSwitchedDuringWorker(t *testing.T) {
	for _, clipboard := range []bool{false, true} {
		t.Run(fmt.Sprintf("finish-clipboard-%v", clipboard), func(t *testing.T) {
			s := syntheticLauncher()
			s.config = settings.Defaults()
			s.config.IndexRoots = []string{t.TempDir()}
			s.switchMode(!clipboard)
			for i := 0; i < 30; i++ {
				s.history.Add(fmt.Sprintf("Dedicated clip %02d", i))
			}
			old := append([]core.Application(nil), s.apps...)
			callbacks := make(chan func(), 1)
			release := make(chan struct{})
			started := make(chan struct{}, 1)
			s.uiUpdate = func(fn func()) { callbacks <- fn }
			s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
				started <- struct{}{}
				<-release
				return old, nil, nil
			}
			tester := newPaletteTester(s, 1.5)
			s.scan()
			<-started
			if err := tester.Click("菜单"); err != nil {
				t.Fatal(err)
			}
			label := "应用"
			query := "Synthetic"
			if clipboard {
				label = "剪贴板"
				query = "Dedicated"
			}
			if err := tester.Click("菜单项：" + label); err != nil {
				t.Fatal(err)
			}
			tester.Type(query)
			tester.Scroll(450, 200, 0, 360)
			offset, selected, clipSelection := s.scroll.Y, s.selected, s.clipSelection
			close(release)
			awaitScanUI(t, callbacks)
			tester.Frame()
			if s.clipboardMode != clipboard || s.query != query || s.scroll.Y != offset || s.selected != selected || s.clipSelection != clipSelection || !tester.Focused("搜索") {
				t.Fatal("completion restored worker-start mode/query/viewport instead of current UI")
			}
		})
	}
}

func TestDeletedUnselectedRowCannotTransferOldPointerRelease(t *testing.T) {
	s := syntheticLauncher()
	var launched []string
	s.launchApplication = func(path string) error { launched = append(launched, path); return nil }
	tester := newPaletteTester(s, 1)
	r, ok := tester.Find("结果：Synthetic App 02")
	if !ok {
		t.Fatal("old row absent")
	}
	tester.Press(r.X+r.W/2, r.Y+r.H/2)
	s.refreshApplications(append(append([]core.Application{}, s.apps[:2]...), s.apps[3:]...), nil)
	tester.Release(r.X+r.W/2, r.Y+r.H/2)
	if len(launched) != 0 || s.appNeedsSelection {
		t.Fatal("removed unselected row click transferred to neighbor or changed selection guard")
	}
}

func TestDeletedSelectionBlocksStaleEnterPrimaryAndRowRelease(t *testing.T) {
	for _, kind := range []string{"enter", "primary", "row"} {
		t.Run(kind, func(t *testing.T) {
			s := syntheticLauncher()
			launched := 0
			s.launchApplication = func(string) error { launched++; return nil }
			tester := newPaletteTester(s, 1)
			s.selected = 2
			tester.Frame()
			label := "打开应用"
			if kind == "row" {
				label = "结果：Synthetic App 02"
			}
			if kind != "enter" {
				r, ok := tester.Find(label)
				if !ok {
					t.Fatal("press target missing")
				}
				tester.Press(r.X+r.W/2, r.Y+r.H/2)
				s.refreshApplications(append(append([]core.Application{}, s.apps[:2]...), s.apps[3:]...), nil)
				tester.Release(r.X+r.W/2, r.Y+r.H/2)
			} else {
				s.refreshApplications(append(append([]core.Application{}, s.apps[:2]...), s.apps[3:]...), nil)
				tester.Key(0, ui.KeyEnter)
			}
			if launched != 0 || !s.appNeedsSelection {
				t.Fatal("old action launched replacement or removed guard")
			}
			tester.Frame()
			tester.Key(0, ui.KeyEnter)
			if launched != 0 {
				t.Fatal("quiet frame bypassed lost-selection guard")
			}
			tester.Key(0, ui.KeyDown)
			tester.Key(0, ui.KeyEnter)
			if launched != 1 {
				t.Fatal("explicit navigation did not restore activation")
			}
		})
	}
}

func TestScanFailureCancelPartialBudgetAndEmptyResults(t *testing.T) {
	for _, kind := range []string{"error", "cancel", "partial", "budget", "empty"} {
		t.Run(kind, func(t *testing.T) {
			s := syntheticLauncher()
			s.query = "Synthetic"
			s.scroll.Y = 420
			s.selected = 12
			s.viewportHeight = palette.Height
			oldID := s.selectedApplicationID()
			oldY := s.scroll.Y
			var apps []core.Application
			var problems []error
			var workerErr error
			switch kind {
			case "error":
				problems = []error{errors.New("dedicated permission denied")}
			case "cancel":
				workerErr = context.Canceled
			case "partial":
				apps = s.apps
				problems = []error{errors.New("dedicated location unavailable")}
			case "budget":
				apps = s.apps
				workerErr = context.DeadlineExceeded
			}
			s.completeScan(apps, nil, problems, workerErr)
			if s.query != "Synthetic" {
				t.Fatal("scan changed query")
			}
			if kind == "empty" {
				if len(s.apps) != 0 || s.selected != 0 || s.scroll.Y != 0 || !strings.Contains(s.status, "0 个应用") {
					t.Fatal("empty publication invalid")
				}
				return
			}
			if s.selectedApplicationID() != oldID || s.scroll.Y != oldY {
				t.Fatal("error/cancel/partial changed current viewport or selected identity")
			}
			prefix := map[string]string{"error": "扫描失败", "cancel": "扫描已取消", "partial": "扫描部分完成", "budget": "扫描达到"}[kind]
			if !strings.HasPrefix(s.status, prefix) {
				t.Fatal("inaccurate scan outcome")
			}
		})
	}
}

func TestFailedLaunchQueuesRefreshWithoutBlindDeletion(t *testing.T) {
	s := syntheticLauncher()
	s.config = settings.Defaults()
	s.config.IndexRoots = []string{t.TempDir()}
	s.selected = 2
	old := append([]core.Application(nil), s.apps...)
	callbacks := make(chan func(), 1)
	release := make(chan struct{})
	started := make(chan struct{}, 1)
	s.launchApplication = func(string) error { return errors.New("dedicated access denied") }
	s.uiUpdate = func(fn func()) { callbacks <- fn }
	s.scanWork = func(context.Context, []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		started <- struct{}{}
		<-release
		return old, nil, nil
	}
	s.activate(s.apps, nil, false)
	<-started
	if !s.scanning || len(s.apps) != len(old) {
		t.Fatal("launch failure removed an unproven entry")
	}
	close(release)
	awaitScanUI(t, callbacks)
	if s.appNeedsSelection || len(s.apps) != len(old) || !strings.Contains(s.status, "dedicated access denied") {
		t.Fatal("refresh discarded valid entry or lost launch diagnostic")
	}
}

func TestSuccessfulRescanKeepsImportantDiagnosticWithoutGrowingHistory(t *testing.T) {
	s := syntheticLauncher()
	s.status = "Dedicated startup hotkey unavailable"
	s.completeScan(s.apps, nil, nil, nil)
	first := s.status
	s.completeScan(s.apps, nil, nil, nil)
	if s.status != first || strings.Count(s.status, "Dedicated startup hotkey unavailable") != 1 {
		t.Fatal("scan erased diagnostic or recursively appended prior scans")
	}
	s.status = "Dedicated newer user feedback"
	s.completeScan(s.apps, nil, nil, nil)
	if !strings.Contains(s.status, "Dedicated newer user feedback") || strings.Contains(s.status, "Dedicated startup hotkey unavailable") {
		t.Fatal("scan restored old feedback over a newer action")
	}
}

func TestZeroResultsWithAnotherFailedRootDoesNotReviveConfirmedMissingApp(t *testing.T) {
	root := t.TempDir()
	links := filepath.Join(root, "links")
	if err := os.Mkdir(links, 0700); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(links, "PicGo.lnk")
	target := filepath.Join(root, "PicGo.exe")
	for _, path := range []string{link, target} {
		if err := os.WriteFile(path, []byte("dedicated inert fixture"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	launch := core.LaunchIdentity{Target: target}
	options := core.DefaultScanOptions()
	options.AllowPath = platform.LocalAppPath
	options.IncludeResolvedTarget = platform.IncludeApplicationTarget
	roots := []string{links, filepath.Join(root, "UnavailableRoot")}
	launch.AppUserModelIDKnown = true
	resolve := func(string) (core.LaunchIdentity, error) { return launch, nil }
	first, _ := core.IndexApplicationsWithOptions(context.Background(), roots, resolve, options)
	if len(first) != 1 {
		t.Fatal("initial dedicated application missing")
	}
	if err := os.Remove(target); err != nil {
		t.Fatal(err)
	}
	s := syntheticLauncher()
	s.apps = first
	s.config = settings.Defaults()
	s.config.IndexRoots = roots
	callbacks := make(chan func(), 1)
	s.uiUpdate = func(fn func()) { callbacks <- fn }
	s.scanWork = func(ctx context.Context, roots []string) ([]core.Application, map[string]*ui.Bitmap, []error) {
		apps, errs := core.IndexApplicationsWithOptions(ctx, roots, resolve, options)
		if len(apps) != 0 || len(errs) == 0 {
			t.Error("mixed-root zero result not reproduced")
		}
		return apps, nil, errs
	}
	s.scanMissing = func(ctx context.Context, previous []core.Application) map[string]bool {
		missing := map[string]bool{}
		for _, app := range previous {
			if !platform.IncludeApplicationTarget(launch) {
				missing[app.ItemID()] = true
			}
		}
		return missing
	}
	s.scan()
	awaitScanUI(t, callbacks)
	if len(s.apps) != 0 {
		t.Fatal("failed secondary root revived confirmed missing application")
	}
	if !strings.Contains(s.status, "移除 1 个") || !strings.Contains(s.status, "已记录 1 个扫描问题") {
		t.Fatalf("mixed-root scan omitted removal/problem summary: %q", s.status)
	}
	// No absence proof means an unavailable source stays cached.
	s.apps = first
	s.completeScan(nil, nil, []error{errors.New("dedicated access denied")}, nil, scanCompletion{missing: map[string]bool{}})
	if len(s.apps) != 1 {
		t.Fatal("unconfirmed permission failure erased cached entry")
	}
}

func TestExplicitShowResetsSelectionGuardAndDiscardsOldPointerSession(t *testing.T) {
	s := syntheticLauncher()
	var launched []string
	s.launchApplication = func(path string) error { launched = append(launched, path); return nil }
	tester := newPaletteTester(s, 1)
	s.selected = 2
	s.refreshApplications(append(append([]core.Application{}, s.apps[:2]...), s.apps[3:]...), nil)
	if !s.appNeedsSelection {
		t.Fatal("deleted selection guard missing")
	}
	s.hide()
	s.show(false)
	tester.Frame()
	tester.Key(0, ui.KeyEnter)
	if s.appNeedsSelection || len(launched) != 1 || launched[0] != s.apps[0].Path {
		t.Fatal("fresh explicit presentation kept prior deleted-selection guard")
	}
	r, ok := tester.Find("打开应用")
	if !ok {
		t.Fatal("primary absent")
	}
	tester.Press(r.X+r.W/2, r.Y+r.H/2)
	s.hide()
	s.show(false)
	tester.Frame()
	tester.Release(r.X+r.W/2, r.Y+r.H/2)
	if len(launched) != 1 {
		t.Fatal("held pointer activation crossed presentation session")
	}
}

func TestResultInkCannotPaintUnderHeaderOrFooterDPI(t *testing.T) {
	for _, scale := range []float32{1, 1.5} {
		t.Run(fmt.Sprintf("dpi-%03d", int(scale*100)), func(t *testing.T) {
			s := syntheticLauncher()
			s.query = "Synthetic"
			s.scanning = true
			s.scroll.Y = 400
			tester := newPaletteTester(s, scale)
			before := tester.Image()
			pixels := append([]byte(nil), before.Pix...)
			for i := range s.apps {
				s.apps[i].Name = fmt.Sprintf("Alternate label %02d", i)
				s.apps[i].Aliases = []string{"Synthetic"}
			}
			tester.Frame()
			after := tester.Image()
			headerEnd := int(palette.Header*scale) - 2
			footerStart := after.Bounds().Dy() - int(palette.Footer*scale) + 2
			for y := 0; y < after.Bounds().Dy(); y++ {
				if y >= headerEnd && y < footerStart {
					continue
				}
				for x := 0; x < after.Bounds().Dx(); x++ {
					off := y*after.Stride + x*4
					for channel := 0; channel < 4; channel++ {
						if pixels[off+channel] != after.Pix[off+channel] {
							t.Fatalf("result ink painted under fixed bar at %d,%d", x, y)
						}
					}
				}
			}
		})
	}
}
