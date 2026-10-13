//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/feiniao233/MyCast/internal/core"
)

func localizedUTF16Fixture(text string) []byte {
	data := []byte{0xff, 0xfe}
	for _, unit := range utf16.Encode([]rune(text)) {
		data = binary.LittleEndian.AppendUint16(data, unit)
	}
	return data
}

func TestLocalizedINIEncodingsAndBounds(t *testing.T) {
	text := "[Other]\nOther.lnk=无关\n[LocalizedFileNames]\nEditor.lnk=中文编辑器\nCalc.EXE=@%SystemRoot%\\System32\\shell32.dll,-22067\n..\\Escape.lnk=坏路径\nC:Escape.lnk=坏路径\n[Next]\nNext.lnk=无关\n"
	for _, data := range [][]byte{[]byte(text), append([]byte{0xef, 0xbb, 0xbf}, []byte(text)...), localizedUTF16Fixture(text)} {
		entries, err := parseLocalizedINI(data)
		if err != nil || len(entries) != 2 || entries["editor.lnk"] != "中文编辑器" {
			t.Fatalf("valid ini rejected: %v", err)
		}
	}
	for _, data := range [][]byte{{0xff}, {0xff, 0xfe, 1}, {0xff, 0xfe, 0, 0xd8}, {0xff, 0xfe, 0, 0xdc}, []byte("[LocalizedFileNames]\nA.lnk=中\x00文"), []byte(strings.Repeat("x", localizedINIBytes+1)), []byte(strings.Repeat("x", 4097)), []byte("[LocalizedFileNames]\n" + strings.Repeat("A.lnk=中文\n", 1025))} {
		if _, err := parseLocalizedINI(data); err == nil {
			t.Fatal("malformed/oversized ini accepted")
		}
	}
}

func TestLocalizedResourceReferenceStrictForms(t *testing.T) {
	for _, ref := range []string{`@%SystemRoot%\System32\shell32.dll,-22067`, `@%windir%\System32\shell32.dll,-22067`, `@"C:\Synthetic\App.dll",-12`, `@C:\Synthetic\App.exe,-1`} {
		if _, _, err := parseLocalizedResourceReference(ref, `C:\Windows`); err != nil {
			t.Fatalf("valid ref rejected: %s", ref)
		}
	}
	for _, ref := range []string{`@\\server\share\App.dll,-1`, `@file:///C:/App.dll,-1`, `@ms-resource:App,-1`, `@relative.dll,-1`, `@%Unknown%\App.dll,-1`, `@C:\App.dll,1`, `@C:\App.dll,-0`, `@C:\App.dll,-65536`, `@C:\App.dll,-+12`, `@C:\App.dll:stream.dll,-1`, `@C:\App.txt,-1`, `@C:\A*.dll,-1`, `@C:\App.dll,-1;other`} {
		if _, _, err := parseLocalizedResourceReference(ref, `C:\Windows`); err == nil {
			t.Fatalf("unsafe ref accepted: %s", ref)
		}
	}
	if _, _, err := parseLocalizedResourceReference(`@%SystemRoot%\App.dll,-1`, ""); err == nil {
		t.Fatal("unresolved builtin path accepted")
	}
}

func TestLocalizedNamesCacheFallbackAndPathGuards(t *testing.T) {
	apps := []core.Application{{Name: "Editor", Path: `C:\Synthetic\Editor.lnk`}, {Name: "Calc", Path: `C:\Synthetic\Calc.lnk`}, {Name: "Second", Path: `C:\Synthetic\Second.lnk`}, {Name: "原中文", Path: `C:\Synthetic\Existing.lnk`}, {Name: "English", Path: `C:\Synthetic\Remote.lnk`}, {Name: "English reparse", Path: `C:\Synthetic\Reparse.lnk`}}
	reads, resources := 0, 0
	reader := localizedNameReader{windowsDirectory: `C:\Windows`, allow: func(path string) error {
		if strings.HasPrefix(path, `Z:\`) || strings.Contains(path, "Reparse.dll") {
			return errors.New("synthetic nonlocal/reparse refusal")
		}
		return nil
	}, read: func(string) ([]byte, error) {
		reads++
		return []byte("[LocalizedFileNames]\nEditor.lnk=中文编辑器\nCalc.lnk=@C:\\Synthetic\\Resource.dll,-12\nSecond.lnk=@C:\\Synthetic\\Resource.dll,-12\nExisting.lnk=English resource\nRemote.lnk=@Z:\\Resource.dll,-12\nReparse.lnk=@C:\\Synthetic\\Reparse.dll,-12\n"), nil
	}, resource: func(path string, id uint32) (string, error) {
		resources++
		if path != `C:\Synthetic\Resource.dll` || id != 12 {
			t.Fatal("unverified resource opened")
		}
		return "资源工具", nil
	}}
	got := reader.apply(context.Background(), apps)
	if reads != 1 || resources != 1 || got[0].Name != "中文编辑器" || got[1].Name != "资源工具" || got[2].Name != "资源工具" || got[3].Name != "原中文" || got[4].Name != "English" || got[5].Name != "English reparse" {
		t.Fatalf("cache/fallback: reads=%d resources=%d", reads, resources)
	}
	for i, app := range got {
		if app.ItemID() != apps[i].ItemID() || app.Path != apps[i].Path {
			t.Fatal("display name changed identity")
		}
	}
	if len(core.SearchApplications(core.PrepareSearch(got), "Editor", 10)) != 1 {
		t.Fatal("English filename alias lost")
	}
	if len(core.SearchApplications(core.PrepareSearch(got), "Editor.lnk", 10)) != 1 {
		t.Fatal("full original filename alias lost")
	}
	reader.allow = func(path string) error {
		if strings.HasSuffix(path, "desktop.ini") {
			return errors.New("synthetic ini reparse")
		}
		return nil
	}
	reader.apply(context.Background(), apps)
	if reads != 1 {
		t.Fatal("ini path guard ran after opening")
	}
}

func TestLocalizedNamesMergedSourcesAndCancellation(t *testing.T) {
	app := core.Application{Name: "English", Path: `C:\Synthetic\One\English.lnk`, NamePaths: []string{`C:\Synthetic\One\English.lnk`, `C:\Synthetic\Two\Duplicate.lnk`}, Aliases: []string{"Duplicate"}, Launch: core.LaunchIdentity{Target: `C:\Synthetic\Target.exe`, Arguments: "--safe"}}
	reader := localizedNameReader{allow: func(string) error { return nil }, read: func(path string) ([]byte, error) {
		if strings.Contains(path, `\Two\`) {
			return []byte("[LocalizedFileNames]\nDuplicate.lnk=中文重复项"), nil
		}
		return nil, errors.New("synthetic missing")
	}, resource: func(string, uint32) (string, error) { t.Fatal("unexpected resource call"); return "", nil }}
	got := reader.apply(context.Background(), []core.Application{app})[0]
	if got.Name != "中文重复项" || got.ItemID() != app.ItemID() || len(got.NamePaths) != 0 || len(app.NamePaths) != 2 {
		t.Fatal("lost merged source localization or input isolation")
	}
	ctx, cancel := context.WithCancel(context.Background())
	calls := 0
	reader.read = func(string) ([]byte, error) {
		calls++
		cancel()
		return []byte("[LocalizedFileNames]\nEnglish.lnk=中文"), nil
	}
	reader.apply(ctx, []core.Application{app, app})
	if calls != 1 {
		t.Fatal("cancellation continued metadata reads")
	}
}

func TestLocalizedNamesBoundedDirectoriesAndResources(t *testing.T) {
	reads, resources := 0, 0
	reader := localizedNameReader{allow: func(string) error { return nil }, read: func(string) ([]byte, error) { reads++; return nil, nil }, resource: func(string, uint32) (string, error) { resources++; return "中文资源", nil }}
	var apps []core.Application
	for i := 0; i < localizedDirectoryBudget+5; i++ {
		apps = append(apps, core.Application{Name: "English", Path: fmt.Sprintf(`C:\Synthetic\Dir%d\App.lnk`, i)})
	}
	reader.apply(context.Background(), apps)
	if reads != localizedDirectoryBudget {
		t.Fatal("directory metadata budget not enforced")
	}
	apps = nil
	var ini strings.Builder
	ini.WriteString("[LocalizedFileNames]\n")
	for i := 1; i <= localizedResourceBudget+5; i++ {
		apps = append(apps, core.Application{Name: "English", Path: fmt.Sprintf(`C:\Synthetic\App%d.lnk`, i)})
		fmt.Fprintf(&ini, "App%d.lnk=@C:\\Synthetic\\Resource.dll,-%d\n", i, i)
	}
	reader.read = func(string) ([]byte, error) { return []byte(ini.String()), nil }
	reader.apply(context.Background(), apps)
	if resources != localizedResourceBudget {
		t.Fatal("resource budget not enforced")
	}
	reads = 0
	reader.read = func(string) ([]byte, error) { reads++; return make([]byte, localizedINIBytes), nil }
	apps = nil
	for i := 0; i < localizedDirectoryBudget; i++ {
		apps = append(apps, core.Application{Name: "English", Path: fmt.Sprintf(`C:\Synthetic\Bytes%d\App.lnk`, i)})
	}
	reader.apply(context.Background(), apps)
	if reader.bytes > localizedINIBudget || reads > localizedINIBudget/localizedINIBytes {
		t.Fatal("metadata byte budget exceeded")
	}
}

func TestLocalizedININativeBoundedReadAndResourceABI(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "desktop.ini")
	if err := os.WriteFile(path, localizedUTF16Fixture("[LocalizedFileNames]\nApp.lnk=专用测试"), 0600); err != nil {
		t.Fatal(err)
	}
	if LocalAppPath(path) != nil {
		t.Fatal("temporary local metadata path rejected")
	}
	data, err := readLocalizedINI(path)
	if err != nil {
		t.Fatal(err)
	}
	entries, err := parseLocalizedINI(data)
	if err != nil || entries["app.lnk"] != "专用测试" {
		t.Fatal("native local ini read failed")
	}
	if err := os.WriteFile(path, make([]byte, localizedINIBytes+1), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := readLocalizedINI(path); err == nil {
		t.Fatal("oversized metadata file read")
	}
	if localizedResourceFlags != 0x60 {
		t.Fatal("resource mapping can initialize executable code")
	}
	// A generated resource-only PE: no code, entrypoint, imports or third-party
	// program. Its known string makes ABI failures fail rather than silently pass.
	module := filepath.Join(root, "synthetic-resources.dll")
	want := "Dedicated synthetic 资源 ABI"
	if err := os.WriteFile(module, localizedResourcePEFixture(want), 0600); err != nil {
		t.Fatal(err)
	}
	name, err := readLocalizedResource(module, 1)
	if err != nil || name != want {
		t.Fatalf("native resource string ABI: got %q err=%v", name, err)
	}
}
