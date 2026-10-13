# 0.2.0-alpha.1 development candidate

Baseline: published [v0.1.0-alpha](https://github.com/feiniao233/MyCast/releases/tag/v0.1.0-alpha),
commit `5512e940e4693aabb56e02ac2cd0517e5f47dd26`.
Work stays on `feat/0.2-applications`; no main merge, new tag or Release is authorized.
The original workspace and released packages are preserved.

## Stage 1: optional AppsFolder source

Settings offers an explicit AppsFolder source checkbox, off for new and existing
settings until saved by the user. This reads the current user's Shell application
directory, including packaged and desktop entries. It never scans WindowsApps,
other users, registry inventories or document contents.

Names use `SIGDN_NORMALDISPLAY`; identity uses the bounded, fully qualified
AppsFolder parse name. Names, pinyin and aliases cannot change favorite/recent
identity. Icons use `IShellItemImageFactory` with ICONONLY on the serialized
background worker (32 pixels, up to 256 reads); unavailable icons have the
existing generic fallback. COM objects stay on one locked STA thread, Release
and CoTaskMemFree end ownership; HBITMAP is copied and deleted.

Activation passes the validated namespace identity to ShellExecuteExW with
INVOKEIDLIST and NOASYNC, using the default action. An AUMID is not treated as
proof of UWP: the Shell selects the correct handler. No command interpreter,
guessed executable, elevation or global keys are used. File shortcuts keep their
original launch adapter. Cross-provider equivalence is not proven in this stage,
so original shortcut variants are conservatively retained separately.

Enumeration is limited to 5,000 entries, icon reads to 256, and cancellation is
checked between native calls under the existing scan deadline. Synchronous Shell
calls cannot be forcibly interrupted. No replacement worker is spawned while an
old native call is still running. Actual installed-app activation and locale/
icon coverage require user acceptance; compilation and synthetic renders do not
prove them.

Official API references: [BindToHandler](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-ishellitem-bindtohandler),
[ShellExecuteInfo](https://learn.microsoft.com/en-us/windows/win32/api/shellapi/ns-shellapi-shellexecuteinfow),
[Shell image factory](https://learn.microsoft.com/en-us/windows/win32/api/shobjidl_core/nf-shobjidl_core-ishellitemimagefactory-getimage),
[AppUserModel IDs](https://learn.microsoft.com/en-us/windows/win32/shell/appids).

## Planned remaining scope

Configured-source automatic refresh with notification merging, serialized bounded
work, partial-source preservation and stale-generation protection; result source/
location/hide controls and persistent restoration; explicit default-off per-user
startup registration with tray-only single-instance startup. Signing, auto-update,
full-drive search, AI and plugins remain outside this candidate.
