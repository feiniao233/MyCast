//go:build windows && mycast_store_diagnostic

package platform

import (
	"encoding/json"
	"github.com/feiniao233/MyCast/internal/core"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"syscall"
	"testing"
	"unsafe"
)

// Explicit, temporary, opt-in diagnostic: only one known public Store identity,
// no enumeration, activation, clipboard, complete app list or settings writes.
func TestDirectedMicrosoftStoreDiagnostic(t *testing.T) {
	const storeID = "Microsoft.WindowsStore_8wekyb3d8bbwe!App"
	if root, err := os.UserConfigDir(); err == nil {
		file, err := os.Open(filepath.Join(root, "MyCast", "settings.json"))
		if err != nil {
			t.Log("settings_exists=false")
		} else {
			defer file.Close()
			data, err := io.ReadAll(io.LimitReader(file, 128*1024+1))
			var flags struct {
				AppsFolder bool     `json:"apps_folder"`
				Version    int      `json:"version"`
				Hidden     []string `json:"hidden"`
			}
			var fields map[string]json.RawMessage
			if err != nil || len(data) > 128*1024 || json.Unmarshal(data, &flags) != nil || json.Unmarshal(data, &fields) != nil {
				t.Log("settings_readable_valid=false")
			} else {
				_, present := fields["apps_folder"]
				hidden := false
				id := (core.Application{ShellID: core.AppsFolderParsingPrefix + storeID}).ItemID()
				for _, value := range flags.Hidden {
					hidden = hidden || value == id
				}
				t.Logf("settings_exists=true version=%d apps_folder_field_present=%t apps_folder_enabled=%t store_hidden=%t", flags.Version, present, flags.AppsFolder, hidden)
			}
		}
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("diagnostic apartment", hr); err != nil {
		t.Fatal(err)
	}
	defer coUninitialize.Call()
	var folder *comObject
	hr, _, _ = getKnownFolderItem.Call(uintptr(unsafe.Pointer(&appsFolderID)), 0, 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&folder)))
	t.Logf("known_appsfolder_hresult=%08x folder_non_nil=%t", uint32(hr), folder != nil)
	if folder != nil {
		rootName, rootErr := shellDisplayName(folder, 0x80028000)
		t.Logf("known_appsfolder_parse=%q root_error=%v", rootName, rootErr)
		folder.call(2)
	}
	requested := core.AppsFolderParsingPrefix + storeID
	var item *comObject
	for _, candidate := range []string{requested, `shell:AppsFolder\` + storeID, `shell:::{4234d49b-0245-4df3-b780-3893943456e1}\` + storeID} {
		wide, _ := syscall.UTF16PtrFromString(candidate)
		hr, _, _ = createShellItem.Call(uintptr(unsafe.Pointer(wide)), 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&item)))
		t.Logf("known_store_parse_form=%q hresult=%08x", candidate, uint32(hr))
		if comError("open known Store item", hr) == nil && item != nil {
			break
		}
	}
	if err := comError("open known Store item", hr); err != nil {
		t.Logf("store_direct_item_unavailable=%v", err)
		return
	}
	if item == nil {
		t.Fatal("Store item missing interface")
	}
	defer item.call(2)
	name, nameErr := shellDisplayName(item, 0)
	parse, parseErr := shellDisplayName(item, 0x80028000)
	_, validateErr := core.AppsFolderChildIdentity(parse)
	t.Logf("known_store_name=%q name_error=%v known_store_parse=%q parse_error=%v parsing_filter_error=%v", name, nameErr, parse, parseErr, validateErr)
	app, metadataErr := readShellApplicationMetadata(item, true)
	if metadataErr != nil {
		t.Fatal(metadataErr)
	}
	apps := core.PrepareSearch([]core.Application{app})
	t.Logf("known_store_english_match=%t known_store_chinese_match=%t", len(core.SearchApplications(apps, "store", 10)) == 1, len(core.SearchApplications(apps, "微软商店", 10)) == 1)
}
