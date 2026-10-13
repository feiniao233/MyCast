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

Explicit default-off per-user startup registration with tray-only single-instance startup. Signing, auto-update,
full-drive search, AI and plugins remain outside this candidate.

Startup is now implemented in stage 5 below; remaining work is candidate
packaging, exact-commit CI/review, and the explicitly unverified user acceptance.

## Stage 2: automatic configured-source refresh

Safe directory roots use FindFirst/Next/CloseChangeNotification, with zero-time
waits on one handle-owning worker and 500-ms event merging. Subscription safety
checks local volume and reparse ancestors before opening the watch. Missing/file
roots are not broadened to parents; they are retried only at their configured
locations every 30 seconds. The same periodic refresh covers AppsFolder package
changes; filesystem notifications alone do not promise UWP install coverage.

**Stage 4 supersedes that initial fallback policy:** healthy directories now
refresh only on notifications. The 30-second maintenance tick checks just
configured root identity/file metadata and retries failed subscriptions; it does
not traverse unchanged healthy trees. Replacing a directory at the same path
retires its old handle and rebinds the new identity. AppsFolder, if explicitly
enabled, receives its own 30-second Shell-only refresh. It skips file traversal,
localized file resources and file icons, even while the palette is hidden.

Automatic requests during a scan mark one follow-up without cancelling current
work. Explicit source changes still cancel/coalesce and reject stale generations.
Watch epochs reject retired callbacks; quit cancels subscriptions/timers without
blocking the UI on its own posted callbacks. Scan publication retains current
query, selection and visible anchor using the existing UI-thread rules.

Partial file or Shell source failures preserve prior unconfirmed entries only in
still-configured sources. Freshly probed FILE/PATH_NOT_FOUND under safe fixed
local ancestors can remove a source/ordinary target; inaccessible, offline,
network and reparse states cannot establish uninstall. A complete AppsFolder
snapshot can remove its absent identities even when an unrelated local root
failed. Combined catalogs are capped at 5,000 entries; icon reads remain bounded
at 256 per provider (512 maximum with both sources enabled).

Shell icons use a bounded in-memory cache for ten minutes, including unavailable
icons, then attempt refresh; unreadable/over-budget results may retain a previous
known bitmap until a successful read. File icons are cached by file identity/size/modtime
with a 256-entry bound. Retained and unscanned entries retain existing bitmaps.
The Shell fallback still performs bounded enumeration/name/search preparation,
so it has a background cost. It never activates the window or restores focus.
Native calls may delay a worker beyond its cooperative deadline; one STA is used
without a separate application message loop, so third-party Shell-handler
compatibility beyond the controlled synchronous fixtures remains unverified.

## Stage 3: source, location and hide management

The result menu shows the selected entry's original source and launch metadata
in an in-memory details sheet. File location uses a safely parsed PIDL to select
the original `.lnk`/`.exe` in Explorer. AppsFolder is virtual, so its location
action is disabled rather than passing a fabricated path to Explorer.

Hidden entry IDs (up to 200) persist in ordinary settings, retaining their
favorites/aliases/recents. Settings restores each hidden entry, including an
unavailable identity whose label cannot currently be discovered. Hiding affects
MyCast only; original files are never modified or deleted. Filtering happens
before ranking/result limits. Identity-bound menu closures are revalidated before
effects, and deleting/hiding selection blocks pending old activation until an
explicit new selection. Partial-source errors and deadlines retain unconfirmed
items independently for file and Shell providers.

Clearly labeled synthetic headless previews (dedicated fixture data only):
[menu 100%](previews/entry-menu-100.png), [menu 150%](previews/entry-menu-150.png),
[source 100%](previews/entry-source-100.png), [source 150%](previews/entry-source-150.png),
[hidden 100%](previews/entry-hidden-100.png), [hidden 150%](previews/entry-hidden-150.png),
[settings 100%](previews/sources-settings-100.png), [settings 150%](previews/sources-settings-150.png).
These are not desktop screenshots or proof of actual native material/IME behavior.

## Stage 5: opt-in per-user startup

Saving a changed default-off switch writes only HKCU Run `MyCast.Launcher`, using
the quoted current executable and `--background`. No application launch/ordinary
save re-registers startup. Existing different commands are preserved. Settings or
hotkey failures attempt to restore the original exact registration; detected
external changes and rollback failure are reported. No transactional guarantee
against other writers or Windows-wide crash durability is claimed.

Background startup creates the existing single instance and hidden palette,
scans configured sources and exposes tray access without showing/focusing the
window. A duplicate `--background` request leaves the existing instance alone.
Normal duplicate launch still opens it. If tray creation fails, the visible
fallback keeps the application accessible. Actual login/focus behavior requires
incremental user acceptance; mocks and process survival do not prove it.

Synthetic settings: [100%](previews/startup-settings-100.png),
[150%](previews/startup-settings-150.png). These show a draft toggle only; generating
them never registers startup. See [settings recovery/uninstall](SETTINGS.md) and
[privacy](PRIVACY.md).
