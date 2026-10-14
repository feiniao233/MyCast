//go:build windows

// SPDX-License-Identifier: AGPL-3.0-or-later
package platform

import (
	"context"
	"fmt"
	"image"
	"runtime"
	"strings"
	"syscall"
	"unsafe"

	"github.com/feiniao233/MyCast/internal/core"
)

var (
	getKnownFolderItem = shell32.NewProc("SHGetKnownFolderItem")
	createShellItem    = shell32.NewProc("SHCreateItemFromParsingName")
	freeCOMMemory      = ole32.NewProc("CoTaskMemFree")
	shellExecuteEx     = shell32.NewProc("ShellExecuteExW")
	getBitmapObject    = gdi32.NewProc("GetObjectW")
	getDIBits          = gdi32.NewProc("GetDIBits")
	appsFolderID       = comGUID{0x1e87508d, 0x89c2, 0x42f0, [8]byte{0x8a, 0x7e, 0x64, 0x5a, 0x0f, 0x50, 0xca, 0x58}}
	shellItemIID       = comGUID{0x43826d1e, 0xe718, 0x42ee, [8]byte{0xbc, 0x55, 0xa1, 0xe2, 0x61, 0xc3, 0x7b, 0xfe}}
	enumShellItemsIID  = comGUID{0x70629033, 0xe363, 0x4a28, [8]byte{0xa5, 0x67, 0x0d, 0xb7, 0x80, 0x06, 0xe6, 0xd7}}
	enumItemsBHID      = comGUID{0x94f60519, 0x2850, 0x4924, [8]byte{0xaa, 0x5a, 0xd1, 0x5e, 0x84, 0x86, 0x80, 0x39}}
	imageFactoryIID    = comGUID{0xbcc18b79, 0xba16, 0x442f, [8]byte{0x80, 0xc4, 0x8a, 0x59, 0xc3, 0x0c, 0x46, 0x3b}}
)

// AppsFolderApplications supplements shortcuts with proven packaged applications
// from the current user's Shell directory in one STA.
// Nothing is activated and no WindowsApps paths or other users are scanned.
// COM calls can block: cancellation is cooperative between calls, with no
// abandoned apartment or unbounded replacement goroutines.
func AppsFolderApplications(ctx context.Context, cachedIcons ...map[string]image.Image) ([]core.Application, map[string]image.Image, []error) {
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("initialize AppsFolder reader", hr); err != nil {
		return nil, nil, []error{err}
	}
	defer coUninitialize.Call()
	if err := ctx.Err(); err != nil {
		return nil, nil, []error{err}
	}
	var folder *comObject
	hr, _, _ = getKnownFolderItem.Call(uintptr(unsafe.Pointer(&appsFolderID)), 0, 0, uintptr(unsafe.Pointer(&shellItemIID)), uintptr(unsafe.Pointer(&folder)))
	if err := comError("open AppsFolder", hr); err != nil {
		return nil, nil, []error{err}
	}
	if folder == nil {
		return nil, nil, []error{fmt.Errorf("AppsFolder returned no interface")}
	}
	defer folder.call(2)
	return enumerateShellApplications(ctx, folder, 5000, 256, cachedIcons...)
}

func enumerateShellApplications(ctx context.Context, folder *comObject, limit, iconLimit int, cachedIcons ...map[string]image.Image) ([]core.Application, map[string]image.Image, []error) {
	return enumerateShellApplicationsWithPackageAPI(ctx, folder, limit, iconLimit, nativeShellPackageAPI(), cachedIcons...)
}

func enumerateShellApplicationsWithPackageAPI(ctx context.Context, folder *comObject, limit, iconLimit int, api shellPackageAPI, cachedIcons ...map[string]image.Image) ([]core.Application, map[string]image.Image, []error) {
	root, rootErr := shellDisplayName(folder, 0x80028000)
	verifiedFolder := rootErr == nil && strings.EqualFold(root, strings.TrimSuffix(core.AppsFolderParsingPrefix, `\`))
	var cache map[string]image.Image
	if len(cachedIcons) > 0 {
		cache = cachedIcons[0]
	}
	var enumeration *comObject
	if err := comError("enumerate Shell applications", folder.call(3, 0, uintptr(unsafe.Pointer(&enumItemsBHID)), uintptr(unsafe.Pointer(&enumShellItemsIID)), uintptr(unsafe.Pointer(&enumeration)))); err != nil {
		return nil, nil, []error{err}
	}
	if enumeration == nil {
		return nil, nil, []error{fmt.Errorf("Shell enumeration returned no interface")}
	}
	defer enumeration.call(2)
	var apps []core.Application
	icons := map[string]image.Image{}
	var problems []error
	packageProblem := !api.available()
	if packageProblem {
		problems = append(problems, errShellPackageProof)
	}
	seen := map[string]bool{}
	for n := 0; n < limit; n++ {
		if err := ctx.Err(); err != nil {
			problems = append(problems, err)
			return apps, icons, problems
		}
		var item *comObject
		var fetched uint32
		hr := enumeration.call(3, 1, uintptr(unsafe.Pointer(&item)), uintptr(unsafe.Pointer(&fetched)))
		if err := comError("read Shell application", hr); err != nil {
			if item != nil {
				item.call(2)
			}
			return apps, icons, append(problems, err)
		}
		if uint32(hr) == 1 && fetched == 0 && item == nil {
			return apps, icons, problems
		}
		if uint32(hr) != 0 || fetched != 1 || item == nil {
			if item != nil {
				item.call(2)
			}
			return apps, icons, append(problems, fmt.Errorf("invalid Shell enumeration output"))
		}
		app, metadataErr := readShellApplicationMetadataWithPackageAPI(item, verifiedFolder, api)
		identity := strings.ToLower(app.ShellID)
		if metadataErr != nil {
			if len(problems) < 128 && (metadataErr != errShellPackageProof || !packageProblem) {
				problems = append(problems, metadataErr)
			}
			packageProblem = packageProblem || metadataErr == errShellPackageProof
		} else if app.ShellPackaged && !seen[identity] {
			if core.IsUninstallEntry(app) {
				item.call(2)
				continue
			}
			seen[identity] = true
			apps = append(apps, app)
			if len(apps) <= iconLimit && ctx.Err() == nil {
				im, found := cache[app.ItemID()]
				if !found {
					im = shellItemIcon(item)
				}
				icons[app.ItemID()] = im
			}
		}
		item.call(2)
	}
	return apps, icons, append(problems, core.ErrScanBudget)
}

func readShellApplicationMetadataWithPackageAPI(item *comObject, verifiedFolder bool, api shellPackageAPI) (core.Application, error) {
	app, err := readShellApplicationMetadata(item, verifiedFolder)
	if err != nil {
		return app, err
	}
	app.ShellPackaged, err = shellApplicationIsPackaged(app, api)
	return app, err
}

// Metadata-only callers (including explicitly requested diagnostics) can still
// inspect ordinary desktop Shell items without making a package query.
func readShellApplicationMetadata(item *comObject, verifiedFolder bool) (core.Application, error) {
	name, nameErr := shellDisplayName(item, 0)
	parse, parseErr := shellDisplayName(item, 0x80028000)
	identity, idErr := core.ValidateShellApplication(parse)
	if verifiedFolder && parseErr == nil {
		identity, idErr = core.AppsFolderChildIdentity(parse)
	}
	if nameErr != nil || parseErr != nil || idErr != nil || name == "" {
		return core.Application{}, fmt.Errorf("Shell application metadata unavailable")
	}
	app := core.Application{Name: name, ShellID: identity, Aliases: core.SystemApplicationAliases(identity)}
	app.Launch.Target, _ = shellStringProperty(item, &linkTargetKey)
	app.Launch.Arguments, _ = shellStringProperty(item, &linkArgumentsKey)
	app.Launch.AppUserModelID, app.Launch.AppUserModelIDKnown = shellStringOrEmptyProperty(item, &appUserModelIDKey)
	return app, nil
}

// An AppsFolder identity or an AppUserModelID alone is not package proof: both
// are also used by ordinary desktop applications. Keep their metadata available
// to diagnostics, but never include them in the packaged-app supplement.
func shellApplicationIsPackaged(app core.Application, api shellPackageAPI) (bool, error) {
	identity := strings.TrimPrefix(app.ShellID, core.AppsFolderParsingPrefix)
	// Packaged AUMIDs have a family!relative-app ID. A desktop ID without
	// that form is definitively excluded even when optional properties fail.
	if _, ok := packagedAUMIDFamily(identity); !ok {
		return false, nil
	}
	if !app.Launch.AppUserModelIDKnown || app.Launch.AppUserModelID == "" || !strings.EqualFold(app.Launch.AppUserModelID, identity) {
		return false, errShellPackageProof
	}
	return registeredPackageApplication(app.Launch.AppUserModelID, api)
}

func shellDisplayName(item *comObject, kind uint32) (string, error) {
	var buffer *uint16
	hr := item.call(5, uintptr(kind), uintptr(unsafe.Pointer(&buffer)))
	if buffer != nil {
		defer freeCOMMemory.Call(uintptr(unsafe.Pointer(buffer)))
	}
	if err := comError("read Shell name", hr); err != nil {
		return "", err
	}
	if buffer == nil {
		return "", fmt.Errorf("Shell returned no name")
	}
	// Shell owns a NUL-terminated allocation; bound traversal without slicing
	// beyond its terminator. Only the original allocation is freed.
	units := make([]uint16, 0, 2048)
	for n := 0; n < 2048; n++ {
		v := *(*uint16)(unsafe.Add(unsafe.Pointer(buffer), n*2))
		if v == 0 {
			return syscall.UTF16ToString(units), nil
		}
		units = append(units, v)
	}
	return "", fmt.Errorf("Shell name exceeds limit")
}

type shellBitmap struct {
	Type, Width, Height, WidthBytes int32
	Planes, BitsPixel               uint16
	Bits                            uintptr
}

func shellItemIcon(item *comObject) image.Image {
	var factory *comObject
	hr := item.call(0, uintptr(unsafe.Pointer(&imageFactoryIID)), uintptr(unsafe.Pointer(&factory)))
	if comError("query Shell icon", hr) != nil || factory == nil {
		return nil
	}
	defer factory.call(2)
	var bitmap uintptr
	// SIZE is an eight-byte by-value struct on Windows amd64.
	hr = factory.call(3, uintptr(uint64(32)|uint64(32)<<32), 4, uintptr(unsafe.Pointer(&bitmap))) // ICONONLY
	if bitmap != 0 {
		defer deleteGDIObject.Call(bitmap)
	}
	if comError("read Shell icon", hr) != nil || bitmap == 0 {
		return nil
	}
	return copyShellBitmap(bitmap)
}
func copyShellBitmap(bitmap uintptr) image.Image {
	var object shellBitmap
	count, _, _ := getBitmapObject.Call(bitmap, unsafe.Sizeof(object), uintptr(unsafe.Pointer(&object)))
	if count != unsafe.Sizeof(object) || object.Width <= 0 || object.Height <= 0 || object.Width > 256 || object.Height > 256 {
		return nil
	}
	dc, _, _ := createCompatibleDC.Call(0)
	if dc == 0 {
		return nil
	}
	defer deleteDC.Call(dc)
	header := dibHeader{Size: 40, Width: object.Width, Height: -object.Height, Planes: 1, Bits: 32}
	pixels := make([]byte, int(object.Width*object.Height)*4)
	rows, _, _ := getDIBits.Call(dc, bitmap, 0, uintptr(object.Height), uintptr(unsafe.Pointer(&pixels[0])), uintptr(unsafe.Pointer(&header)), 0)
	runtime.KeepAlive(pixels)
	if rows != uintptr(object.Height) {
		return nil
	}
	result := image.NewRGBA(image.Rect(0, 0, int(object.Width), int(object.Height)))
	visible := false
	for i := 0; i < len(pixels); i += 4 {
		result.Pix[i], result.Pix[i+1], result.Pix[i+2], result.Pix[i+3] = pixels[i+2], pixels[i+1], pixels[i], pixels[i+3]
		visible = visible || pixels[i+3] != 0
	}
	if !visible {
		return nil
	}
	return result
}

// No process handle is requested; no guessed executable or shell command is
// constructed. Shell chooses the appropriate desktop/packaged default action.
type shellExecuteInfo struct {
	Size, Mask                        uint32
	Window                            uintptr
	Verb, File, Parameters, Directory *uint16
	Show                              int32
	Instance, IDList                  uintptr
	Class                             *uint16
	ClassKey                          uintptr
	Hotkey                            uint32
	Icon, Process                     uintptr
}

func LaunchShellApplication(identity string) error {
	value, err := core.ShellApplicationActivationName(identity)
	if err != nil {
		return err
	}
	runtime.LockOSThread()
	defer runtime.UnlockOSThread()
	hr, _, _ := coInitializeEx.Call(0, 2)
	if err := comError("initialize Shell activation", hr); err != nil {
		return err
	}
	defer coUninitialize.Call()
	file, _ := syscall.UTF16PtrFromString(value)
	info := shellExecuteInfo{Mask: 0x0c | 0x100 | 0x400, File: file, Show: 1}
	info.Size = uint32(unsafe.Sizeof(info))
	ok, _, callErr := shellExecuteEx.Call(uintptr(unsafe.Pointer(&info)))
	runtime.KeepAlive(file)
	if ok == 0 {
		return fmt.Errorf("Shell application activation failed: %w", callErr)
	}
	return nil
}
