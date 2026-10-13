//go:build windows && mycast_duplicate_diagnostic

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"syscall"
	"testing"
	"unsafe"

	"github.com/feiniao233/MyCast/internal/core"
)

// Explicitly supplied identities only: no AppsFolder/source enumeration,
// activation, settings writes, clipboard or executable access.
func TestDirectedDuplicateMetadata(t *testing.T) {
	ids := strings.Split(os.Getenv("MYCAST_DUPLICATE_DIAGNOSTIC_IDS"), ";")
	if len(ids) != 2 || ids[0] == "" || ids[1] == "" {
		t.Skip("two explicit identities required")
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("diagnostic apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	keys := []struct {
		name string
		key  propertyKey
	}{
		{"target", linkTargetKey}, {"arguments", linkArgumentsKey},
		{"app_id", propertyKey{comGUID{0x9f4c2855, 0x9f79, 0x4b39, [8]byte{0xa8, 0xd0, 0xe1, 0xd4, 0x2d, 0xe1, 0xd5, 0xf3}}, 5}},
		{"relaunch", propertyKey{comGUID{0x9f4c2855, 0x9f79, 0x4b39, [8]byte{0xa8, 0xd0, 0xe1, 0xd4, 0x2d, 0xe1, 0xd5, 0xf3}}, 2}},
	}
	// Optional exact shortcut basenames in current configured roots only. No
	// directory enumeration; source paths/names and settings are never logged.
	names := strings.Split(os.Getenv("MYCAST_DUPLICATE_DIAGNOSTIC_NAMES"), ";")
	roots := DefaultAppRoots()
	if base, err := os.UserConfigDir(); err == nil {
		if file, err := os.Open(filepath.Join(base, "MyCast", "settings.json")); err == nil {
			data, readErr := io.ReadAll(io.LimitReader(file, 128*1024+1))
			file.Close()
			var value struct {
				IndexRoots []string `json:"index_roots"`
			}
			if readErr == nil && len(data) <= 128*1024 && json.Unmarshal(data, &value) == nil && len(value.IndexRoots) <= 32 && len(value.IndexRoots) > 0 {
				roots = value.IndexRoots
			}
		}
	}
	for n, id := range ids {
		wide, err := syscall.UTF16PtrFromString(`shell:AppsFolder\` + id)
		if err != nil {
			t.Fatal(err)
		}
		var item *comObject
		hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
		runtime.KeepAlive(wide)
		if err := comError("known duplicate identity", hr); err != nil || item == nil {
			if item != nil {
				item.call(2)
			}
			t.Logf("pair=%d unavailable hr=%08x", n, uint32(hr))
			continue
		}
		for _, key := range keys {
			value, ok := shellStringProperty(item, &key.key)
			t.Logf("pair=%d %s available=%t value=%q", n, key.name, ok, value)
		}
		physical, physicalErr := shellDisplayName(item, 0x80058000)
		arguments, argsOK := shellStringOrEmptyProperty(item, &linkArgumentsKey)
		app, metaErr := readShellApplicationMetadata(item, true)
		t.Logf("pair=%d variant_arguments_available=%t value=%q desktop_profile=%t metadata_error=%v", n, argsOK, arguments, app.ShellDesktop, metaErr)
		if len(names) == 2 && names[n] != "" && filepath.Base(names[n]) == names[n] {
			for _, root := range roots {
				for _, relative := range []string{names[n] + ".lnk", filepath.Join("Programs", names[n]+".lnk"), filepath.Join("Programs", names[n], names[n]+".lnk")} {
					source := filepath.Join(root, relative)
					if LocalAppPath(source) != nil {
						continue
					}
					identity, err := shortcutIdentityWithAppIDReader(source, func(link *comObject) (string, bool) {
						var props *comObject
						qi := link.call(0, uintptr(unsafe.Pointer(&propertyStoreIID)), uintptr(unsafe.Pointer(&props)))
						if props != nil {
							defer props.call(2)
							var value stringPropVariant
							get := props.call(5, uintptr(unsafe.Pointer(&appUserModelIDKey)), uintptr(unsafe.Pointer(&value)))
							t.Logf("pair=%d appid_query_hr=%08x appid_value_hr=%08x variant_type=%d", n, uint32(qi), uint32(get), value.kind)
							clearPropVariant.Call(uintptr(unsafe.Pointer(&value)))
						} else {
							t.Logf("pair=%d appid_query_hr=%08x interface_absent=true", n, uint32(qi))
						}
						return shortcutAppUserModelID(link)
					})
					if err != nil {
						continue
					}
					file := core.Application{Name: "diagnostic-file", Path: source, Launch: identity}
					merged := core.MergeProviderApplications([]core.Application{file, app})
					t.Logf("pair=%d exact_shortcut_found=true target_matches=%t app_id_matches=%t empty_arguments=%t directory_is_exe_parent=%t independent=%t flags=%08x show=%d directed_merge_count=%d", n, strings.EqualFold(filepath.Clean(identity.Target), filepath.Clean(app.Launch.Target)), identity.AppUserModelID == app.Launch.AppUserModelID, identity.Arguments == "", strings.EqualFold(filepath.Clean(identity.WorkingDirectory), filepath.Dir(identity.Target)), identity.Independent, identity.LinkFlags, identity.ShowCommand, len(merged))
				}
			}
		}
		t.Logf("pair=%d filesystem_name=%q err=%v", n, physical, physicalErr)
		bhid := comGUID{0x3981e225, 0xf559, 0x11d3, [8]byte{0x8e, 0x3a, 0, 0xc0, 0x4f, 0x68, 0x37, 0xd5}}
		var link *comObject
		result := item.call(3, 0, uintptr(unsafe.Pointer(&bhid)), uintptr(unsafe.Pointer(&shellLinkInterface)), uintptr(unsafe.Pointer(&link)))
		t.Logf("pair=%d shell_link_binding_hr=%08x present=%t", n, uint32(result), link != nil)
		if link != nil {
			link.call(2)
		}
		item.call(2)
	}
}
