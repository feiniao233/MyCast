//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"crypto/sha256"
	"errors"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"
)

func TestNativeShellPropertiesAndPIDLGuardPreserveSyntheticShortcut(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Fixture.exe")
	path := filepath.Join(root, "Fixture.lnk")
	if err := os.WriteFile(target, []byte("inert"), 0600); err != nil {
		t.Fatal(err)
	}
	createTestShortcut(t, path, core.LaunchIdentity{Target: target, Arguments: "--synthetic-fixture"})
	before, _ := os.ReadFile(path)
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("fixture apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	wide, _ := syscall.UTF16PtrFromString(path)
	var item *comObject
	hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
	if err := comError("fixture Shell item", hr); err != nil {
		t.Fatal(err)
	}
	if item == nil {
		t.Fatal("missing fixture item")
	}
	defer item.call(2)
	for i := 0; i < 16; i++ {
		got, ok := shellStringProperty(item, &linkArgumentsKey)
		if !ok || got != "--synthetic-fixture" {
			t.Fatal("IShellItem2 property ABI failed")
		}
		got, ok = shellStringProperty(item, &linkTargetKey)
		if !ok {
			t.Fatal("target property missing")
		}
		a, e1 := os.Stat(got)
		b, e2 := os.Stat(target)
		if e1 != nil || e2 != nil || !os.SameFile(a, b) {
			t.Fatal("target property did not identify owned fixture")
		}
	}
	var link *comObject
	hr, _, _ = createCOMInstance.Call(uintptr(unsafe.Pointer(&shellLinkClass)), 0, 1, uintptr(unsafe.Pointer(&shellLinkInterface)), uintptr(unsafe.Pointer(&link)))
	if err := comError("fixture link reader", hr); err != nil {
		t.Fatal(err)
	}
	if link == nil {
		t.Fatal("missing link reader")
	}
	defer link.call(2)
	var file *comObject
	if err := comError("fixture persistence", link.call(0, uintptr(unsafe.Pointer(&persistFileInterface)), uintptr(unsafe.Pointer(&file)))); err != nil {
		t.Fatal(err)
	}
	if file == nil {
		t.Fatal("missing persistence")
	}
	defer file.call(2)
	if err := comError("fixture read-only load", file.call(5, uintptr(unsafe.Pointer(wide)), 0)); err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 16; i++ {
		if shortcutAppsFolderTarget(link) != "" {
			t.Fatal("filesystem PIDL accepted as AppsFolder immediate child")
		}
	}
	after, _ := os.ReadFile(path)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("read-only metadata changed source shortcut")
	}
}

func TestDesktopProofRequiresLocalAncestorGateWithoutRemovingNamespace(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Fixture.exe")
	if err := os.WriteFile(target, []byte("inert"), 0600); err != nil {
		t.Fatal(err)
	}
	base := core.Application{Name: "Synthetic Desktop", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop", Launch: core.LaunchIdentity{Target: target}}
	good := base
	markDesktopAssociation(&good, "org.synthetic.desktop", true, "", true, LocalAppPath)
	if !good.ShellDesktop {
		t.Fatal("owned local target not classified")
	}
	denied := base
	calls := 0
	markDesktopAssociation(&denied, "org.synthetic.desktop", true, "", true, func(string) error { calls++; return errors.New("synthetic unknown/network/reparse target") })
	if calls != 1 || denied.ShellDesktop || denied.ItemID() != base.ItemID() || denied.Name != base.Name {
		t.Fatal("unknown target merged/removed namespace")
	}
	failed := base
	markDesktopAssociation(&failed, "org.synthetic.desktop", true, "", false, func(string) error { t.Fatal("failed arguments still probed target"); return nil })
	if failed.ShellDesktop {
		t.Fatal("failed property accepted")
	}
}

func TestNativeExplicitAppIDAndEmptyPropertyOwnership(t *testing.T) {
	if unsafe.Sizeof(stringPropVariant{}) != 24 || unsafe.Sizeof(propertyKey{}) != 20 {
		t.Fatal("property ABI size")
	}
	root := t.TempDir()
	path := filepath.Join(root, "Synthetic Desktop.lnk")
	launch := core.LaunchIdentity{Target: filepath.Join(root, "Fixture.exe"), WorkingDirectory: root, AppUserModelID: "org.synthetic.fixture"}
	createTestShortcut(t, path, launch)
	before, _ := os.ReadFile(path)
	for i := 0; i < 32; i++ {
		got, err := ShortcutIdentity(path)
		if err != nil || got.AppUserModelID != launch.AppUserModelID || got.WorkingDirectory != root || got.Independent {
			t.Fatal("explicit AppID read or original semantics lost", err)
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("fixture apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	wide, _ := syscall.UTF16PtrFromString(path)
	var item *comObject
	hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
	if err := comError("fixture item", hr); err != nil || item == nil {
		t.Fatal("missing fixture item", err)
	}
	defer item.call(2)
	for i := 0; i < 32; i++ {
		id, ok := shellStringOrEmptyProperty(item, &appUserModelIDKey)
		if !ok || id != launch.AppUserModelID {
			t.Fatal("Shell AppID variant read")
		}
		args, ok := shellStringOrEmptyProperty(item, &linkArgumentsKey)
		if !ok || args != "" {
			t.Fatal("successful empty arguments confused with failed metadata")
		}
		runtime.GC()
	}
	after, _ := os.ReadFile(path)
	if sha256.Sum256(before) != sha256.Sum256(after) {
		t.Fatal("metadata changed fixture")
	}
}

func TestAppIDPropertyFailuresStayUnknownThroughNativeIndex(t *testing.T) {
	root := t.TempDir()
	links := filepath.Join(root, "Links")
	if err := os.Mkdir(links, 0700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "Host.exe")
	if err := os.WriteFile(target, []byte("inert fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{"Alpha.lnk", "Beta.lnk"} {
		createTestShortcut(t, filepath.Join(links, name), core.LaunchIdentity{Target: target, WorkingDirectory: root, AppUserModelID: "org.synthetic.beta"})
	}
	for _, failure := range []string{"query-interface", "get-value", "unsupported-property"} {
		t.Run(failure, func(t *testing.T) {
			var methods, propertyMethods [24]uintptr
			properties := &comObject{methods: &propertyMethods}
			link := &comObject{methods: &methods}
			releases := 0
			propertyMethods[2] = syscall.NewCallback(func(*comObject) uintptr { releases++; return 1 })
			propertyMethods[5] = syscall.NewCallback(func(*comObject, *propertyKey, *stringPropVariant) uintptr {
				if failure == "unsupported-property" {
					return 0x80070490
				}
				return 0x80004005
			})
			methods[0] = syscall.NewCallback(func(_ *comObject, _ *comGUID, out **comObject) uintptr {
				if failure == "query-interface" {
					*out = nil
					return 0x80004002
				}
				*out = properties
				return 0
			})
			value, known := shortcutAppUserModelID(link)
			if value != "" || known {
				t.Fatal("COM failure became successful empty AppID")
			}
			if failure != "query-interface" && releases != 1 {
				t.Fatal("failed property read leaked interface")
			}
			read := func(*comObject) (string, bool) { return shortcutAppUserModelID(link) }
			apps, errs := core.IndexApplicationsWithResolver([]string{links}, func(path string) (core.LaunchIdentity, error) { return shortcutIdentityWithAppIDReader(path, read) })
			if len(errs) != 0 || len(apps) != 2 {
				t.Fatal("unreadable AppID shortcuts merged", errs, len(apps))
			}
			shell := core.Application{Name: "Namespace", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.alpha", ShellDesktop: true, Launch: core.LaunchIdentity{Target: target, AppUserModelID: "org.synthetic.alpha"}}
			merged := core.MergeProviderApplications(append(apps, shell))
			if len(merged) != 3 {
				t.Fatal("failed AppID read authorized target-only bridge")
			}
			for _, app := range apps {
				if app.Launch.AppUserModelIDKnown || !app.Launch.Independent || app.Launch.ShellTarget != "" {
					t.Fatal("failure status lost before merge")
				}
			}
			mixed, errs := core.IndexApplicationsWithResolver([]string{links}, func(path string) (core.LaunchIdentity, error) {
				if filepath.Base(path) == "Beta.lnk" {
					return shortcutIdentityWithAppIDReader(path, read)
				}
				return ShortcutIdentity(path)
			})
			if len(errs) != 0 || len(mixed) != 2 {
				t.Fatal("mixed known/failed AppID scan")
			}
			legacy := (core.Application{Path: filepath.Join(links, "Alpha.lnk"), Launch: core.LaunchIdentity{Target: target, WorkingDirectory: root, AppUserModelIDKnown: true}}).ItemID()
			for _, app := range mixed {
				if app.HasPreferenceID(legacy) {
					t.Fatal("unknown sibling authorized ambiguous legacy migration")
				}
			}
			runtime.KeepAlive(link)
			runtime.KeepAlive(properties)
		})
	}
	// A successful VT_EMPTY is the only absent-value control, not an API error.
	var methods, propertyMethods [24]uintptr
	properties := &comObject{methods: &propertyMethods}
	link := &comObject{methods: &methods}
	propertyMethods[2] = syscall.NewCallback(func(*comObject) uintptr { return 1 })
	propertyMethods[5] = syscall.NewCallback(func(_ *comObject, _ *propertyKey, out *stringPropVariant) uintptr {
		*out = stringPropVariant{}
		return 0
	})
	methods[0] = syscall.NewCallback(func(_ *comObject, _ *comGUID, out **comObject) uintptr { *out = properties; return 0 })
	value, known := shortcutAppUserModelID(link)
	if value != "" || !known {
		t.Fatal("successful empty property was not distinguished")
	}
	runtime.KeepAlive(link)
	runtime.KeepAlive(properties)
}

func TestNativeAppIDScanThenProviderMergePreservesBothShortcuts(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Host.exe")
	if err := os.WriteFile(target, []byte("inert fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	// Shell may return runner account/temp ancestors in 8.3 form. Establish
	// the owned fixture spelling through a real link before creating the inputs;
	// this tests AppID semantics, not unproved filesystem spelling equivalence.
	probe := filepath.Join(root, "Probe.lnk")
	createTestShortcut(t, probe, core.LaunchIdentity{Target: target})
	read, err := ShortcutIdentity(probe)
	if err != nil {
		t.Fatal(err)
	}
	a, e1 := os.Stat(target)
	b, e2 := os.Stat(read.Target)
	if e1 != nil || e2 != nil || !os.SameFile(a, b) {
		t.Fatal("native fixture target changed")
	}
	directory := filepath.Dir(read.Target)
	a, e1 = os.Stat(root)
	b, e2 = os.Stat(directory)
	if e1 != nil || e2 != nil || !os.SameFile(a, b) {
		t.Fatal("native fixture directory changed")
	}
	root, target = directory, read.Target
	links := filepath.Join(root, "Links")
	if err := os.Mkdir(links, 0700); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{"Synthetic Alpha.lnk": "org.synthetic.alpha", "Synthetic Alpha Copy.lnk": "org.synthetic.alpha", "Synthetic Beta.lnk": "org.synthetic.beta"} {
		createTestShortcut(t, filepath.Join(links, name), core.LaunchIdentity{Target: target, WorkingDirectory: root, AppUserModelID: id})
	}
	apps, errs := core.IndexApplicationsWithResolver([]string{links}, ShortcutIdentity)
	if len(errs) != 0 || len(apps) != 2 || apps[0].ItemID() == apps[1].ItemID() {
		t.Fatal("file scanner merged conflicting AppIDs or lost normal duplicate dedupe", errs, len(apps))
	}
	for _, id := range []string{"org.synthetic.alpha", "org.synthetic.beta"} {
		apps = append(apps, core.Application{Name: "Namespace " + id, ShellID: core.AppsFolderParsingPrefix + id, ShellDesktop: true, Launch: core.LaunchIdentity{Target: target, AppUserModelID: id}})
	}
	merged := core.PrepareSearch(core.MergeProviderApplications(apps))
	if len(merged) != 2 {
		t.Fatal("provider merge changed independent AppID count", len(merged))
	}
	result := core.SearchApplications(merged, "Synthetic Beta", 10)
	if len(result) != 1 || result[0].Path != filepath.Join(links, "Synthetic Beta.lnk") || result[0].Launch.AppUserModelID != "org.synthetic.beta" {
		t.Fatal("search for Beta routes Alpha shortcut")
	}
	if result[0].ItemID() != (core.Application{ShellID: core.AppsFolderParsingPrefix + "org.synthetic.beta"}).ItemID() {
		t.Fatal("wrong canonical namespace identity")
	}
	legacy := (core.Application{Path: filepath.Join(links, "Synthetic Alpha.lnk"), Launch: core.LaunchIdentity{Target: target, WorkingDirectory: root, AppUserModelIDKnown: true}}).ItemID()
	for _, app := range merged {
		if app.HasPreferenceID(legacy) {
			t.Fatal("ambiguous old target hash guessed onto an AppID")
		}
	}
	value := settings.Defaults()
	value.Favorites = []string{legacy}
	value.Recents = []string{legacy}
	value.Hidden = []string{legacy}
	value.Aliases = map[string][]string{legacy: {"legacy-fixture"}}
	unchanged, changed := value.BindProvedApplicationIdentities(merged)
	if changed || len(unchanged.Favorites) != 1 || len(unchanged.Aliases[legacy]) != 1 {
		t.Fatal("ambiguous old preferences redirected or deleted")
	}
	// When one identity remains, the same real scanner/proof/save/restart chain
	// can conservatively migrate the old tuple without losing ordinary metadata.
	if err := os.Remove(filepath.Join(links, "Synthetic Beta.lnk")); err != nil {
		t.Fatal(err)
	}
	files, errs := core.IndexApplicationsWithResolver([]string{links}, ShortcutIdentity)
	if len(errs) != 0 || len(files) != 1 {
		t.Fatal("normal same-AppID duplicates stopped merging")
	}
	shell := core.Application{Name: "Namespace Alpha", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.alpha", ShellDesktop: true, Launch: core.LaunchIdentity{Target: target, AppUserModelID: "org.synthetic.alpha"}}
	unique := core.MergeProviderApplications(append(files, shell))
	if len(unique) != 1 || !unique[0].HasPreferenceID(legacy) {
		t.Fatal("unambiguous old tuple was not retained")
	}
	bound, changed := value.BindProvedApplicationIdentities(unique)
	if !changed || bound.IdentityLinks[legacy] != shell.ItemID() {
		t.Fatal("unambiguous old preferences were not bound")
	}
	store := &settings.Store{Path: filepath.Join(root, "settings.json")}
	if err := store.Save(bound); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&settings.Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	restarted := core.WithUserAliases(loaded.ApplyIdentityLinks([]core.Application{shell}), loaded.Aliases)
	if !restarted[0].Preferred(loaded.Favorites) || !restarted[0].Preferred(loaded.Recents) || !restarted[0].Preferred(loaded.Hidden) || len(core.SearchApplications(restarted, "legacy-fixture", 10)) != 1 {
		t.Fatal("full-chain migration lost preferences after restart")
	}
}

func TestBSTRStringBoundsNullAndCanonicalSuccess(t *testing.T) {
	for _, test := range []struct {
		name, text string
		hr         uint32
		known      bool
	}{
		{"ordinary", "org.synthetic.desktop", 0, true},
		{"empty", "", 0, true},
		{"canonical-success", "org.synthetic.desktop", 0x000401a0, true},
		{"embedded-nul", "alpha\x00beta", 0, false},
		{"oversized", strings.Repeat("x", 2049), 0, false},
		{"unexpected-success", "", 1, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			var methods [24]uintptr
			props := &comObject{methods: &methods}
			units := make([]uint16, 0, len(test.text))
			for _, r := range test.text {
				units = append(units, uint16(r))
			}
			methods[5] = syscall.NewCallback(func(_ *comObject, _ *propertyKey, out *stringPropVariant) uintptr {
				out.kind = 8
				if len(units) > 0 {
					ptr, _, _ := oleAutomation.NewProc("SysAllocStringLen").Call(uintptr(unsafe.Pointer(&units[0])), uintptr(len(units)))
					if ptr == 0 {
						return 0x8007000e
					}
					out.value[0] = ptr
				}
				return uintptr(test.hr)
			})
			value, known := readStringVariant(props, 5, &appUserModelIDKey)
			if known != test.known || known && value != test.text {
				t.Fatal("BSTR contract/bounds mismatch")
			}
			runtime.KeepAlive(units)
			runtime.KeepAlive(props)
		})
	}
}
