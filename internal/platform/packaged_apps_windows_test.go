//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"image"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/feiniao233/MyCast/internal/core"
)

const syntheticPackageFamily = "Synthetic.Package_123456789abcd"
const syntheticPackageAUMID = syntheticPackageFamily + "!App"

// This fake writes the native APIs' actual pointer-buffer formats into caller
// owned storage. No package, installed app, shortcut or real inventory is read.
func syntheticPackageAPI(t *testing.T, appIDs ...string) (shellPackageAPI, *int) {
	t.Helper()
	familyName := syntheticPackageFamily
	if len(appIDs) > 0 {
		familyName, _, _ = strings.Cut(appIDs[0], "!")
	}
	packageName, publisher, _ := strings.Cut(familyName, "_")
	packageFullName := packageName + "_1.0.0.0_x64__" + publisher
	closed := new(int)
	return shellPackageAPI{
		packages: func(family *uint16, count *uint32, names **uint16, length *uint32, buffer *uint16) uint32 {
			want, _ := syscall.UTF16FromString(familyName)
			if !strings.EqualFold(stringFromSyntheticPointer(family, len(want)), familyName) {
				*count, *length = 0, 0
				return 0
			}
			fullName, _ := syscall.UTF16FromString(packageFullName)
			*count, *length = 1, uint32(len(fullName))
			if buffer == nil || names == nil {
				return uint32(syscall.ERROR_INSUFFICIENT_BUFFER)
			}
			copy(unsafe.Slice(buffer, len(fullName)), fullName)
			*names = buffer
			return 0
		},
		open: func(fullName *uint16, reference *uintptr) uint32 {
			want, _ := syscall.UTF16FromString(packageFullName)
			if stringFromSyntheticPointer(fullName, len(want)) != packageFullName {
				t.Error("queried a package outside the requested family")
				return 1168
			}
			*reference = 123
			return 0
		},
		applications: func(reference uintptr, length *uint32, buffer *byte, count *uint32) uint32 {
			if reference != 123 {
				t.Error("wrong package reference")
				return 87
			}
			return writeSyntheticPackageApplications(appIDs, length, buffer, count)
		},
		close: func(reference uintptr) {
			if reference != 123 {
				t.Error("closed wrong package reference")
			}
			*closed++
		},
	}, closed
}

func stringFromSyntheticPointer(pointer *uint16, length int) string {
	return syscall.UTF16ToString(unsafe.Slice(pointer, length))
}

func writeSyntheticPackageApplications(ids []string, length *uint32, buffer *byte, count *uint32) uint32 {
	pointerBytes := int(unsafe.Sizeof(uintptr(0))) * len(ids)
	units := make([]uint16, pointerBytes/2)
	offsets := make([]int, 0, len(ids))
	for _, id := range ids {
		offsets = append(offsets, len(units))
		wide, _ := syscall.UTF16FromString(id)
		units = append(units, wide...)
	}
	required := uint32(len(units) * 2)
	capacity := *length
	*length, *count = required, uint32(len(ids))
	if buffer == nil || capacity < required {
		return uint32(syscall.ERROR_INSUFFICIENT_BUFFER)
	}
	out := unsafe.Slice((*uint16)(unsafe.Pointer(buffer)), len(units))
	copy(out, units)
	for i, offset := range offsets {
		*(**uint16)(unsafe.Add(unsafe.Pointer(buffer), uintptr(i)*unsafe.Sizeof(uintptr(0)))) = &out[offset]
	}
	return 0
}

func TestPackagedShellProofRequiresRegisteredExactApplication(t *testing.T) {
	api, closed := syntheticPackageAPI(t, syntheticPackageAUMID, syntheticPackageFamily+"!Other")
	for _, test := range []struct {
		name, shellID, appID string
		known, want          bool
	}{
		{"packaged", syntheticPackageAUMID, syntheticPackageAUMID, true, true},
		{"another-packaged-app", syntheticPackageFamily + "!Other", syntheticPackageFamily + "!Other", true, true},
		{"ordinary-desktop-appid", "org.synthetic.desktop", "org.synthetic.desktop", true, false},
		{"desktop-with-package-shaped-id", "Unregistered.Package_123456789abcd!App", "Unregistered.Package_123456789abcd!App", true, false},
		{"installed-family-wrong-app", syntheticPackageFamily + "!Missing", syntheticPackageFamily + "!Missing", true, false},
		{"appid-mismatch", syntheticPackageAUMID, syntheticPackageFamily + "!Other", true, false},
		{"property-failed", syntheticPackageAUMID, syntheticPackageAUMID, false, false},
		{"property-absent", syntheticPackageAUMID, "", true, false},
		{"no-family", "!App", "!App", true, false},
		{"no-application", syntheticPackageFamily + "!", syntheticPackageFamily + "!", true, false},
		{"extra-delimiter", syntheticPackageAUMID + "!Other", syntheticPackageAUMID + "!Other", true, false},
		{"oversized", syntheticPackageAUMID + strings.Repeat("x", 130), syntheticPackageAUMID + strings.Repeat("x", 130), true, false},
		{"embedded-nul", syntheticPackageAUMID + "\x00", syntheticPackageAUMID + "\x00", true, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := core.Application{Name: "任意本地化名称", ShellID: core.AppsFolderParsingPrefix + test.shellID, Launch: core.LaunchIdentity{AppUserModelID: test.appID, AppUserModelIDKnown: test.known}}
			got, err := shellApplicationIsPackaged(app, api)
			if got != test.want {
				t.Fatalf("package proof = %t, want %t", got, test.want)
			}
			wantError := test.name == "property-failed" || test.name == "property-absent" || test.name == "appid-mismatch"
			if (err != nil) != wantError {
				t.Fatalf("proof error = %v, want error %t", err, wantError)
			}
		})
	}
	if *closed != 3 {
		t.Fatalf("opened package references not closed exactly once: %d", *closed)
	}
	if got, err := registeredPackageApplication(syntheticPackageAUMID, shellPackageAPI{}); got || err == nil {
		t.Fatal("unavailable package APIs became a healthy result")
	}
}

func TestPackageProofFailsClosedOnNativeFailuresAndBounds(t *testing.T) {
	for _, failure := range []string{
		"family-query-fails", "family-empty", "family-count-budget", "family-buffer-budget",
		"family-fill-fails", "family-count-grows", "family-buffer-grows", "family-null-name", "family-unterminated-name",
		"open-fails", "open-null", "applications-query-fails", "applications-buffer-budget",
		"applications-fill-fails", "applications-buffer-grows", "applications-odd-bytes", "applications-empty", "applications-count-budget",
		"applications-null-id", "applications-header-pointer", "applications-unterminated-id", "applications-unrelated",
	} {
		t.Run(failure, func(t *testing.T) {
			api, closed := syntheticPackageAPI(t, syntheticPackageAUMID)
			packages, open, applications := api.packages, api.open, api.applications
			api.packages = func(family *uint16, count *uint32, names **uint16, length *uint32, buffer *uint16) uint32 {
				result := packages(family, count, names, length, buffer)
				if buffer == nil {
					switch failure {
					case "family-query-fails":
						return 5
					case "family-empty":
						*count = 0
					case "family-count-budget":
						*count = 65
					case "family-buffer-budget":
						*length = 64*1024 + 1
					}
				} else {
					switch failure {
					case "family-fill-fails":
						return 5
					case "family-count-grows":
						*count++
					case "family-buffer-grows":
						*length++
					case "family-null-name":
						*names = nil
					case "family-unterminated-name":
						unsafe.Slice(buffer, int(*length))[*length-1] = 'x'
					}
				}
				return result
			}
			api.open = func(fullName *uint16, reference *uintptr) uint32 {
				if failure == "open-fails" {
					return 1168 // Uninstalled between family lookup and open.
				}
				if failure == "open-null" {
					return 0
				}
				return open(fullName, reference)
			}
			api.applications = func(reference uintptr, length *uint32, buffer *byte, count *uint32) uint32 {
				result := applications(reference, length, buffer, count)
				if buffer == nil {
					switch failure {
					case "applications-query-fails":
						return 5
					case "applications-buffer-budget":
						*length = 64*1024 + 1
					}
				} else {
					switch failure {
					case "applications-fill-fails":
						return 5
					case "applications-buffer-grows":
						*length += 2
					case "applications-odd-bytes":
						*length--
					case "applications-empty":
						*count = 0
					case "applications-count-budget":
						*count = 129
					case "applications-null-id":
						*(**uint16)(unsafe.Pointer(buffer)) = nil
					case "applications-header-pointer":
						*(**uint16)(unsafe.Pointer(buffer)) = (*uint16)(unsafe.Pointer(buffer))
					case "applications-unterminated-id":
						units := unsafe.Slice((*uint16)(unsafe.Pointer(buffer)), int(*length/2))
						units[len(units)-1] = 'x'
					case "applications-unrelated":
						pointer := *(**uint16)(unsafe.Pointer(buffer))
						*pointer = 'X'
					}
				}
				return result
			}
			got, err := registeredPackageApplication(syntheticPackageAUMID, api)
			if got {
				t.Fatal("failed or malformed package query became proof")
			}
			if (err != nil) != (failure != "applications-unrelated") {
				t.Fatalf("unknown/negative proof distinction: %v", err)
			}
			wantClosed := 0
			if strings.HasPrefix(failure, "applications-") {
				wantClosed = 1
			}
			if *closed != wantClosed {
				t.Fatalf("package reference cleanup: %d, want %d", *closed, wantClosed)
			}
		})
	}
}

func TestPackageBufferStringRejectsOutsideMisalignedAndTruncatedPointers(t *testing.T) {
	buffer, _ := syscall.UTF16FromString("Synthetic")
	other, _ := syscall.UTF16FromString("Outside")
	for _, test := range []struct {
		name    string
		pointer *uint16
		minimum int
	}{
		{"nil", nil, 0},
		{"outside", &other[0], 0},
		{"misaligned", (*uint16)(unsafe.Add(unsafe.Pointer(&buffer[0]), 1)), 0},
		{"inside-pointer-table", &buffer[0], 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if _, ok := packageBufferString(buffer, test.pointer, test.minimum); ok {
				t.Fatal("invalid native pointer accepted")
			}
		})
	}
	if got, ok := packageBufferString(buffer, &buffer[0], 0); !ok || got != "Synthetic" {
		t.Fatal("valid owned buffer rejected")
	}
	if _, ok := packageBufferString(buffer[:len(buffer)-1], &buffer[0], 0); ok {
		t.Fatal("read string terminator outside returned length")
	}
}

// Exercise the real COM reader/enumerator using synthetic vtables and real COM
// string allocations, without enumerating the user's AppsFolder.
func syntheticPackagedShellItem(t *testing.T, name, parsing, appID string, propertyOK bool) (*comObject, *int) {
	t.Helper()
	var methods [24]uintptr
	item := &comObject{methods: &methods}
	releases := new(int)
	methods[2] = syscall.NewCallback(func(*comObject) uintptr { *releases++; return 1 })
	methods[0] = syscall.NewCallback(func(_ *comObject, iid *comGUID, out **comObject) uintptr {
		if *iid != shellItem2IID {
			*out = nil
			return 0x80004002
		}
		*out = item
		return 0
	})
	methods[5] = syscall.NewCallback(func(_ *comObject, kind uint32, out **uint16) uintptr {
		value := name
		if kind != 0 {
			value = parsing
		}
		return duplicateSyntheticShellString(value, out)
	})
	methods[13] = syscall.NewCallback(func(_ *comObject, key *propertyKey, out *stringPropVariant) uintptr {
		if *key != appUserModelIDKey || !propertyOK {
			return 0x80070490
		}
		if appID == "" {
			return 0 // Successful VT_EMPTY.
		}
		out.kind = 31
		return duplicateSyntheticShellString(appID, (**uint16)(unsafe.Pointer(&out.value[0])))
	})
	methods[17] = syscall.NewCallback(func(_ *comObject, _ *propertyKey, _ **uint16) uintptr { return 0x80070490 })
	return item, releases
}

func duplicateSyntheticShellString(value string, out **uint16) uintptr {
	wide, _ := syscall.UTF16PtrFromString(value)
	result, _, _ := syscall.NewLazyDLL("shlwapi.dll").NewProc("SHStrDupW").Call(uintptr(unsafe.Pointer(wide)), uintptr(unsafe.Pointer(out)))
	runtime.KeepAlive(wide)
	return result
}

func TestAppsFolderSupplementSkipsDesktopRowsWithoutMetadataErrors(t *testing.T) {
	api, closed := syntheticPackageAPI(t, syntheticPackageAUMID)
	var items []*comObject
	var releases []*int
	for _, row := range []struct {
		name, id, property string
		known              bool
	}{
		{"Desktop without AppID", "org.synthetic.no-appid", "", true},
		{"Desktop with AppID", "org.synthetic.desktop", "org.synthetic.desktop", true},
		{"Unknown property", "org.synthetic.failed", "", false},
		{"Package-shaped desktop", syntheticPackageFamily + "!Missing", syntheticPackageFamily + "!Missing", true},
		{"本地化应用", syntheticPackageAUMID, syntheticPackageAUMID, true},
		{"Duplicate package identity", strings.ToUpper(syntheticPackageAUMID), strings.ToUpper(syntheticPackageAUMID), true},
	} {
		item, released := syntheticPackagedShellItem(t, row.name, row.id, row.property, row.known)
		items, releases = append(items, item), append(releases, released)
	}
	folder, enumerationReleases := syntheticPackageFolder(t, items)
	apps, icons, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 100, 0, api)
	if len(problems) != 0 || len(apps) != 1 || len(icons) != 0 {
		t.Fatalf("supplement: %d apps, %d icons, %d errors", len(apps), len(icons), len(problems))
	}
	if apps[0].Name != "本地化应用" || !apps[0].ShellPackaged || apps[0].ShellID != core.AppsFolderParsingPrefix+syntheticPackageAUMID || apps[0].Path != "" {
		t.Fatal("packaged row lost localized name, identity, or Shell launch")
	}
	if *enumerationReleases != 1 || *closed != 3 {
		t.Fatal("enumerator or package references leaked")
	}
	for i, released := range releases {
		want := 4 // Three property queries plus enumerator's item reference.
		if i < 3 {
			want = 1
		} // Known desktop rows skip all optional properties.
		if *released != want {
			t.Fatalf("Shell reference releases: %d, want %d", *released, want)
		}
	}
	// The same nonpackage row remains available to narrowly scoped diagnostics.
	app, err := readShellApplicationMetadata(items[1], true)
	if err != nil || app.ShellPackaged || app.ShellID != core.AppsFolderParsingPrefix+"org.synthetic.desktop" || app.Launch.AppUserModelID != "org.synthetic.desktop" {
		t.Fatal("diagnostic desktop metadata was removed or classified as packaged")
	}
	runtime.KeepAlive(items)
	runtime.KeepAlive(folder)
}

func syntheticPackageFolder(t *testing.T, items []*comObject) (*comObject, *int) {
	t.Helper()
	folder, _ := syntheticPackagedShellItem(t, "Applications", strings.TrimSuffix(core.AppsFolderParsingPrefix, `\`), "", false)
	var enumerationMethods [24]uintptr
	enumerator := &comObject{methods: &enumerationMethods}
	position, enumerationReleases := 0, new(int)
	enumerationMethods[2] = syscall.NewCallback(func(*comObject) uintptr { *enumerationReleases++; return 1 })
	enumerationMethods[3] = syscall.NewCallback(func(_ *comObject, count uint32, out **comObject, fetched *uint32) uintptr {
		if count != 1 {
			return 0x80070057
		}
		if position == len(items) {
			*out, *fetched = nil, 0
			return 1
		}
		*out, *fetched = items[position], 1
		position++
		return 0
	})
	folder.methods[3] = syscall.NewCallback(func(_ *comObject, _ uintptr, _ *comGUID, _ *comGUID, out **comObject) uintptr {
		*out = enumerator
		return 0
	})
	return folder, enumerationReleases
}

func TestAppsFolderPackageVerificationFailureReportsPartialScan(t *testing.T) {
	for _, failure := range []string{"missing-appid", "unavailable-api", "family-read-fails", "uninstalled-during-read"} {
		t.Run(failure, func(t *testing.T) {
			item, released := syntheticPackagedShellItem(t, "Packaged fixture", syntheticPackageAUMID, syntheticPackageAUMID, failure != "missing-appid")
			folder, enumeratorReleases := syntheticPackageFolder(t, []*comObject{item})
			api, _ := syntheticPackageAPI(t, syntheticPackageAUMID)
			switch failure {
			case "unavailable-api":
				api = shellPackageAPI{}
			case "family-read-fails":
				api.packages = func(*uint16, *uint32, **uint16, *uint32, *uint16) uint32 { return 5 }
			case "uninstalled-during-read":
				api.open = func(*uint16, *uintptr) uint32 { return 1168 }
			}
			apps, _, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 10, 0, api)
			if len(apps) != 0 || len(problems) != 1 || problems[0] != errShellPackageProof {
				t.Fatalf("unknown package proof became a healthy empty scan: %d apps, %v", len(apps), problems)
			}
			if *released != 4 || *enumeratorReleases != 1 {
				t.Fatal("partial enumeration leaked COM references")
			}
			app, err := readShellApplicationMetadata(item, true)
			if err != nil || app.Name != "Packaged fixture" || app.ShellID != core.AppsFolderParsingPrefix+syntheticPackageAUMID {
				t.Fatal("package-query failure removed diagnostic metadata")
			}
			runtime.KeepAlive(item)
			runtime.KeepAlive(folder)
		})
	}
}

func TestPackagedStoreSupplementPreservesNamesAliasesAndShellActivation(t *testing.T) {
	const storeAUMID = "Microsoft.WindowsStore_8wekyb3d8bbwe!App"
	api, _ := syntheticPackageAPI(t, storeAUMID)
	item, _ := syntheticPackagedShellItem(t, "Microsoft Store", storeAUMID, storeAUMID, true)
	folder, _ := syntheticPackageFolder(t, []*comObject{item})
	apps, _, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 10, 0, api)
	if len(problems) != 0 || len(apps) != 1 || !apps[0].ShellPackaged || apps[0].Path != "" {
		t.Fatalf("packaged Store fixture rejected: %v", problems)
	}
	prepared := core.PrepareSearch(apps)
	for _, query := range []string{"store", "微软商店", "微软应用商店"} {
		if len(core.SearchApplications(prepared, query, 10)) != 1 {
			t.Fatalf("Store query %q lost its match", query)
		}
	}
	activation, err := core.ShellApplicationActivationName(apps[0].ShellID)
	if err != nil || activation != `shell:AppsFolder\`+storeAUMID {
		t.Fatal("Store lost its Shell activation target")
	}
	runtime.KeepAlive(item)
	runtime.KeepAlive(folder)
}

func TestUnavailablePackageAPIKeepsEmptyEnumerationPartial(t *testing.T) {
	folder, _ := syntheticPackageFolder(t, nil)
	apps, _, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 10, 0, shellPackageAPI{})
	if len(apps) != 0 || len(problems) != 1 || problems[0] != errShellPackageProof {
		t.Fatal("unavailable package provider reported a healthy empty snapshot")
	}
	runtime.KeepAlive(folder)
}

func TestFilteredDesktopRowsDoNotConsumePackagedIconBudget(t *testing.T) {
	api, _ := syntheticPackageAPI(t, syntheticPackageAUMID)
	desktop, _ := syntheticPackagedShellItem(t, "Desktop", "org.synthetic.desktop", "org.synthetic.desktop", true)
	packaged, _ := syntheticPackagedShellItem(t, "Package", syntheticPackageAUMID, syntheticPackageAUMID, true)
	folder, _ := syntheticPackageFolder(t, []*comObject{desktop, packaged})
	app := core.Application{ShellID: core.AppsFolderParsingPrefix + syntheticPackageAUMID}
	icon := image.NewRGBA(image.Rect(0, 0, 1, 1))
	apps, icons, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 100, 1, api, map[string]image.Image{app.ItemID(): icon})
	if len(problems) != 0 || len(apps) != 1 || icons[app.ItemID()] != icon {
		t.Fatal("excluded desktop rows exhausted accepted-package icon budget")
	}
	runtime.KeepAlive(folder)
	runtime.KeepAlive(desktop)
	runtime.KeepAlive(packaged)
}

func TestPackagedSupplementClassifiesIdentityBeforeOptionalName(t *testing.T) {
	for _, test := range []struct {
		name, parsing, display                     string
		parseFails, nameFails, verified, wantError bool
	}{
		{"desktop-empty-name", "org.synthetic.desktop", "", false, false, true, false},
		{"desktop-name-query-fails", "org.synthetic.desktop", "Desktop", false, true, true, false},
		{"package-empty-name", syntheticPackageAUMID, "", false, false, true, true},
		{"package-name-query-fails", syntheticPackageAUMID, "Package", false, true, true, true},
		{"desktop-parsing-query-fails", "org.synthetic.desktop", "Desktop", true, false, true, true},
		{"package-parsing-query-fails", syntheticPackageAUMID, "Package", true, false, true, true},
		{"invalid-parsing-identity", `C:\Synthetic\Desktop.exe`, "Desktop", false, false, true, true},
		{"unverified-bare-identity", "org.synthetic.desktop", "Desktop", false, false, false, true},
		{"malformed-package-identity", syntheticPackageFamily + "!", "Package", false, false, true, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			api, _ := syntheticPackageAPI(t, syntheticPackageAUMID)
			item, _ := syntheticPackagedShellItem(t, test.display, test.parsing, test.parsing, true)
			nameReads := 0
			item.methods[5] = syscall.NewCallback(func(_ *comObject, kind uint32, out **uint16) uintptr {
				if kind == 0 {
					nameReads++
					if test.nameFails {
						return 0x80004005
					}
					return duplicateSyntheticShellString(test.display, out)
				}
				if test.parseFails {
					return 0x80004005
				}
				return duplicateSyntheticShellString(test.parsing, out)
			})
			app, err := readShellApplicationMetadataWithPackageAPI(item, test.verified, api)
			if (err != nil) != test.wantError || app.ShellPackaged {
				t.Fatalf("classification app=%+v error=%v, wantError=%t", app, err, test.wantError)
			}
			if !test.wantError && nameReads != 0 {
				t.Fatal("excluded desktop row still queried optional name")
			}
			if !test.wantError {
				if _, err := readShellApplicationMetadata(item, true); err == nil {
					t.Fatal("diagnostic reader silently hid requested missing display metadata")
				}
			}
			runtime.KeepAlive(item)
		})
	}
}

func TestExcludedDesktopNameFailuresDoNotFillMetadataWarningCap(t *testing.T) {
	const storeID = "Microsoft.WindowsStore_8wekyb3d8bbwe!App"
	api, _ := syntheticPackageAPI(t, storeID)
	var items []*comObject
	// Reuse the inert fixture to exercise 140 enumeration results without
	// consuming a native callback slot for every logically identical row.
	desktop, _ := syntheticPackagedShellItem(t, "", "org.synthetic.desktop", "org.synthetic.desktop", true)
	for i := 0; i < 140; i++ {
		items = append(items, desktop)
	}
	store, _ := syntheticPackagedShellItem(t, "Microsoft Store", storeID, storeID, true)
	items = append(items, store)
	folder, _ := syntheticPackageFolder(t, items)
	apps, _, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 200, 0, api)
	if len(problems) != 0 || len(apps) != 1 || apps[0].ShellID != core.AppsFolderParsingPrefix+storeID {
		t.Fatalf("excluded display errors polluted scan: apps=%d, problems=%d", len(apps), len(problems))
	}
	runtime.KeepAlive(folder)
	runtime.KeepAlive(items)
}

func TestPackagedSupplementRetainsAggregatedProofFailureWarning(t *testing.T) {
	api, _ := syntheticPackageAPI(t, syntheticPackageAUMID)
	var items []*comObject
	for i := 0; i < 3; i++ {
		item, _ := syntheticPackagedShellItem(t, "Package", syntheticPackageAUMID, "", false)
		items = append(items, item)
	}
	folder, _ := syntheticPackageFolder(t, items)
	apps, _, problems := enumerateShellApplicationsWithPackageAPI(context.Background(), folder, 10, 0, api)
	if len(apps) != 0 || len(problems) != 1 || problems[0] != errShellPackageProof {
		t.Fatal("true package proof failure was hidden or multiplied", problems)
	}
	runtime.KeepAlive(folder)
	runtime.KeepAlive(items)
}
