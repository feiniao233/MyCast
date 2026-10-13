//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/feiniao233/MyCast/internal/core"
)

var malformedResourceQuotes = []string{
	`@",-1`, `@"",-1`, `@" ",-1`, `@"a",-1`,
	`@"C:\Synthetic\App.dll,-1`, `@C:\Synthetic\App.dll",-1`,
	`@""C:\Synthetic\App.dll"",-1`, `@"C:\Synthetic\"App.dll",-1`,
	`@",-`, `@,-1`, `@',-1`, `@'',-1`, `@'C:\Synthetic\App.dll',-1`,
	`@"C:\Synthetic\App.dll",-1,-2`, "@C:\\Synthetic\\\xff.dll,-1",
}

func TestMalformedResourceQuotesRefuseWithoutPanicOrRead(t *testing.T) {
	for _, ref := range malformedResourceQuotes {
		t.Run(ref, func(t *testing.T) {
			if _, _, err := parseLocalizedResourceReference(ref, `C:\Windows`); err == nil {
				t.Fatal("malformed quoting accepted")
			}
			opened := 0
			reader := localizedNameReader{windowsDirectory: `C:\Windows`, allow: func(string) error { return nil }, read: func(string) ([]byte, error) { return []byte("[LocalizedFileNames]\nApp.lnk=" + ref + "\n"), nil }, resource: func(string, uint32) (string, error) { opened++; return "错误替换", nil }}
			app := core.Application{Name: "Original", Path: `C:\Synthetic\App.lnk`}
			got := reader.apply(context.Background(), []core.Application{app})[0]
			if opened != 0 || got.Name != app.Name || got.ItemID() != app.ItemID() {
				t.Fatal("malformed resource did not safely retain original")
			}
		})
	}
}

func FuzzLocalizedResourceReference(f *testing.F) {
	for _, ref := range append(append([]string{}, malformedResourceQuotes...), "", `@`, `@C:\Synthetic\App.dll,-1`, `@"C:\Synthetic\App.dll",-1`, `@%SystemRoot%\System32\shell32.dll,-22067`, strings.Repeat("x", 2049)) {
		f.Add(ref)
	}
	f.Fuzz(func(t *testing.T, ref string) {
		// Input stays small; no filesystem/resource/clipboard API is involved.
		if len(ref) > 4096 {
			t.Skip()
		}
		path, id, err := parseLocalizedResourceReference(ref, `C:\Windows`)
		if err != nil {
			return
		}
		if !filepath.IsAbs(path) || len(filepath.VolumeName(path)) != 2 || strings.HasPrefix(path, `\\`) || strings.ContainsAny(path, "%\"\x00\r\n?*<>|") || id == 0 || id > 65535 {
			t.Fatalf("unsafe successful parse: path=%q id=%d", path, id)
		}
		if ext := filepath.Ext(path); !strings.EqualFold(ext, ".dll") && !strings.EqualFold(ext, ".exe") {
			t.Fatal("unsupported resource extension accepted")
		}
	})
}
