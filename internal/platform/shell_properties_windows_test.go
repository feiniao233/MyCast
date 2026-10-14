//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"crypto/sha256"
	"fmt"
	"github.com/feiniao233/MyCast/internal/core"
	"github.com/feiniao233/MyCast/internal/settings"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unicode"
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

func TestDesktopMetadataDoesNotCreateSupplementalSource(t *testing.T) {
	app := core.Application{Name: "Synthetic Desktop", ShellID: core.AppsFolderParsingPrefix + "org.synthetic.desktop", Launch: core.LaunchIdentity{Target: `C:\Synthetic\Fixture.exe`, AppUserModelID: "org.synthetic.desktop"}}
	if len(core.MergeProviderApplications([]core.Application{app})) != 0 {
		t.Fatal("desktop AppID alone created a supplemental row")
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
			if len(merged) != 2 {
				t.Fatal("ordinary desktop namespace survived source policy")
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
			knownLaunch, err := ShortcutIdentity(filepath.Join(links, "Alpha.lnk"))
			if err != nil {
				t.Fatal(err)
			}
			legacy := historicalNativeTupleID(knownLaunch)
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

func TestNativeAppIDScanUsesNamesAndMigratesOnlyUniqueHistoricalTuple(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "Host.exe")
	if err := os.WriteFile(target, []byte("inert fixture"), 0600); err != nil {
		t.Fatal(err)
	}
	links := filepath.Join(root, "Links")
	if err := os.Mkdir(links, 0700); err != nil {
		t.Fatal(err)
	}
	for name, id := range map[string]string{"Synthetic Alpha.lnk": "org.synthetic.alpha", "Synthetic Alpha Copy.lnk": "org.synthetic.alpha", "Synthetic Beta.lnk": "org.synthetic.beta"} {
		createTestShortcut(t, filepath.Join(links, name), core.LaunchIdentity{Target: target, WorkingDirectory: root, AppUserModelID: id})
	}
	apps, errs := core.IndexApplicationsWithResolver([]string{links}, ShortcutIdentity)
	if len(errs) != 0 || len(apps) != 3 {
		t.Fatal("different source names collapsed", errs, len(apps))
	}
	read, err := ShortcutIdentity(filepath.Join(links, "Synthetic Alpha.lnk"))
	if err != nil {
		t.Fatal(err)
	}
	legacy := fmt.Sprintf("app-%x", sha256.Sum256([]byte(fmt.Sprintf("%q|%q|%q", historicalNativePathKey(read.Target), read.Arguments, historicalNativePathKey(read.WorkingDirectory)))))
	for _, id := range []string{"org.synthetic.alpha", "org.synthetic.beta"} {
		apps = append(apps, core.Application{Name: "Namespace " + id, ShellID: core.AppsFolderParsingPrefix + id, ShellDesktop: true, Launch: core.LaunchIdentity{Target: target, AppUserModelID: id}})
	}
	merged := core.PrepareSearch(core.MergeProviderApplications(apps))
	if len(merged) != 3 {
		t.Fatal("desktop namespace competed with shortcut names", len(merged))
	}
	result := core.SearchApplications(merged, "Synthetic Beta", 10)
	if len(result) != 1 || result[0].Path != filepath.Join(links, "Synthetic Beta.lnk") || result[0].Launch.AppUserModelID != "org.synthetic.beta" {
		t.Fatal("search lost original Beta shortcut")
	}
	for _, app := range merged {
		if app.HasPreferenceID(legacy) {
			t.Fatal("ambiguous old target hash guessed onto a name")
		}
	}
	value := settings.Defaults()
	value.Favorites, value.Recents, value.Hidden = []string{legacy}, []string{legacy}, []string{legacy}
	value.Aliases = map[string][]string{legacy: {"legacy-fixture"}}
	unchanged, changed := value.BindProvedApplicationIdentities(merged)
	if changed || len(unchanged.Favorites) != 1 || len(unchanged.Aliases[legacy]) != 1 {
		t.Fatal("ambiguous old preferences redirected or deleted")
	}
	for _, name := range []string{"Synthetic Beta.lnk", "Synthetic Alpha Copy.lnk"} {
		if err := os.Remove(filepath.Join(links, name)); err != nil {
			t.Fatal(err)
		}
	}
	unique, errs := core.IndexApplicationsWithResolver([]string{links}, ShortcutIdentity)
	if len(errs) != 0 || len(unique) != 1 || !unique[0].HasPreferenceID(legacy) {
		t.Fatal("unique historical tuple was not retained", errs)
	}
	bound, changed := value.BindProvedApplicationIdentities(unique)
	if !changed || bound.IdentityLinks[legacy] != unique[0].ItemID() {
		t.Fatal("unique old preferences were not bound")
	}
	store := &settings.Store{Path: filepath.Join(root, "settings.json")}
	if err := store.Save(bound); err != nil {
		t.Fatal(err)
	}
	loaded, err := (&settings.Store{Path: store.Path}).Load()
	if err != nil {
		t.Fatal(err)
	}
	fresh, _ := core.IndexApplicationsWithResolver([]string{links}, ShortcutIdentity)
	restarted := core.WithUserAliases(loaded.ApplyIdentityLinks(fresh), loaded.Aliases)
	if !restarted[0].Preferred(loaded.Favorites) || !restarted[0].Preferred(loaded.Recents) || !restarted[0].Preferred(loaded.Hidden) || len(core.SearchApplications(restarted, "legacy-fixture", 10)) != 1 {
		t.Fatal("migration lost preferences after restart")
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

// Windows pathKey in the pre-source-policy catalog used minimum Unicode
// simple-fold representatives. It did not lowercase the shortcut metadata.
func historicalNativePathKey(path string) string {
	units := []rune(filepath.Clean(path))
	for i, unit := range units {
		for next := unicode.SimpleFold(unit); next != unit; next = unicode.SimpleFold(next) {
			if next < units[i] {
				units[i] = next
			}
		}
	}
	return string(units)
}

func TestNativeHistoricalPathHashGoldens(t *testing.T) {
	for _, test := range []struct{ input, canonical, id string }{
		{`C:\Synthetic\Fixture.lnk`, `C:\SYNTHETIC\FIXTURE.LNK`, "app-4a5bd4767c284e5b8ce35b28f0bc6b829dba583a08536d51ee1f19be70b17d49"},
		{`C:\Σynthetic\σς.lnk`, `C:\ΣYNTHETIC\ΣΣ.LNK`, "app-c226faf43ecbf066a99ff401cd1ea358d442e032c7acb23eb136b4557ce62d02"},
		{`C:\Kelvin\kK.lnk`, `C:\KELVIN\KK.LNK`, "app-38b917927b164f8d633f4c5bf0c589d05a6b44d63a955fe55674fe3a28c2ff89"},
		{`C:\µ\μΜ.lnk`, `C:\µ\µµ.LNK`, "app-0d289e5cc5321a0fef94a3fe5de9971567be825071dd6667060e4f91091a4b12"},
	} {
		canonical := historicalNativePathKey(test.input)
		if canonical != test.canonical || fmt.Sprintf("app-%x", sha256.Sum256([]byte(canonical))) != test.id {
			t.Fatal("native historical-ID fixture differs from fixed baseline golden")
		}
	}
}

func historicalNativeTupleID(launch core.LaunchIdentity) string {
	identity := fmt.Sprintf("%q|%q|%q", historicalNativePathKey(launch.Target), launch.Arguments, historicalNativePathKey(launch.WorkingDirectory))
	return fmt.Sprintf("app-%x", sha256.Sum256([]byte(identity)))
}
