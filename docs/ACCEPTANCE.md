# Acceptance evidence and current status

## Independent review corrections

Native synthetic Alpha/Beta shortcuts sharing an EXE/arguments/directory but
different AppIDs stay separate through the real file scanner and provider merger;
search Beta routes Beta's original shortcut. Same-AppID duplicate shortcuts still
merge. QueryInterface/GetValue/unsupported-property fault injection preserves
unknown state and blocks target-only association. Unique old hashes survive
save/restart with all ordinary preferences; ambiguous/unknown siblings do not
authorize migration or erase stored preferences. Tab to a nonselected row then
menu key/Shift+F10 targets that row at 100%/150%; search mouse context retains text
editing. Full local verify/test/vet/Windows GUI build passed. Actual new desktop
interaction and DTV reproduction remain open; no further native pass is inferred.
The first review-fix CI run failed its synthetic namespace/shortcut comparison
because Shell shortened runner temp ancestors. The fixture now establishes a
native spelling and requires os.SameFile for both owned EXE and directory before
creating test links. Production still refuses unproved path-spelling equivalence.

## Desktop association and row-menu follow-up

User feedback on fd2d141 passes Store activation and ordinary favorite restart
persistence only. The parent viewed the private duplicate screenshot; this
Windows task did not view its pixels. Directed properties confirmed one supplied
desktop target/AppID and empty argument variant; the second identity was
unavailable and is not a passed reproduction. New metadata association,
both-provider preference persistence, native temporary AppID/resource fixtures
and 100%/150% right-click/keyboard/stale-target regressions pass locally with
verify/test/vet/GUI build. Actual new duplicate/menu behavior remains not tested.
See [details](DESKTOP_ASSOCIATION_CONTEXT_MENU.md).
Desktop proof additionally requires the existing local volume/ancestor gate.
An owned native EXE fixture passes; injected denied/unknown proof keeps the
namespace identity and name while preventing association. Full local checks pass.

## Store/source follow-up

Read-only single-Store metadata diagnosis confirmed a registered Store, a bare
AUMID incorrectly rejected by the old filter, and the usable shell:AppsFolder
activation form. After correction, the production metadata routine matches both
`store` and `微软商店`. No Store activation or complete app enumeration occurred.
The approved default-source migration, explicit disable, shallow default desktop,
uninstall-action counterexamples, provider deadline/capacity isolation and legacy
identity preference regressions pass locally. Native Known Folder/property/PIDL
fixtures and full verify/test/vet/Windows GUI build pass. Private attachment
pixels were unavailable and are not claimed reviewed. See [details](STORE_SOURCE_FIX.md).
The preference persistence follow-up passes removal/file-only refresh and restart
with only Shell: favorites, recents, aliases and hidden hashes stay bound. A
simulated failed replacement leaves disk/config unchanged and reports the missing
persistence. Bound maps reject chains and conflicting retargets. Full dependency
verification, tests, vet and Windows GUI build pass locally. Actual application
activation and unproved desktop-EXE/namespace duplicate coverage remain open.
The merged-counterpart regression also covers confirmed shortcut deletion with
a simultaneous failed/partial Shell refresh, retaining unknown namespace state.

## 0.2.0-alpha.1 stage 1

AppsFolder identities are separate from file paths. Pure tests cover input
validation, display-name changes, distinct shortcuts and source opt-in migration.
A real Shell COM fixture over a dedicated temporary directory repeatedly checks
normal/parse names, enumeration and 32-pixel icon copy/resource cleanup; no user's
installed app inventory is enumerated by tests. Headless 100%/150% tests route
virtual activation through a mock identity adapter and retain favorites, aliases
and recents after rename. Source configuration is captured by the worker.
Module verification, full tests, vet and release-style build are required on
the stage commit. Actual AppsFolder activation, installed-app naming/icons and
native input remain **not tested**. See [candidate notes](NEXT_ALPHA.md).

## 0.2.0-alpha.1 stage 2

Local Go 1.27.2 dependency verification, full tests, vet and GUI build passed.
Native directory-notification fixtures use only a temporary local directory,
verify signal/rearm/close and reject network/file watches. Automatic event tests
send 50 requests during one blocked scan: it is not cancelled, publishes, and
runs exactly one follow-up. Mixed partial-source tests retain an unknown/offline
entry while removing confirmed missing entries and roots explicitly removed by
the user. Existing publication-time query/viewport/selection, deletion guards,
rapid root changes, IME and 100%/150% harness regressions still pass.
Actual install/uninstall notification latency, disk disconnect/reconnect and
native typing/scrolling during refresh are **not tested**. Filesystem watches
cannot guarantee package-change delivery; the bounded 30-second configured-source
refresh covers this separately. A blocked native call remains cooperatively
cancellable only after it returns. First-stage exact-commit CI is tracked
separately from local checks; a failed run is not a local pass. The first-stage
CI failed its fixture string-equality assertion because Shell canonicalized the
runner's temporary directory name. The assertion now requires `os.SameFile`
against the owned directory, preserving the exact-file requirement across
Windows spelling/short-name differences rather than skipping the test.
Stage 2 exact-commit [CI passed](https://github.com/feiniao233/MyCast/actions/runs/38399106551)
for `1568ea8707afa333bfba94c24f3e9164ab79a68e`.

## 0.2.0-alpha.1 stage 3

Identity-bound result menus show the original source and metadata, select the
original local file in Explorer, and persist up to 200 hidden identities. Virtual
items disable file-location actions. Headless 100%/150% tests cover menu geometry,
source details, hide/restore, search focus, stale menu closures, old Enter after
hiding selection, virtual path refusal and failed persistence. Settings tests
cover hidden-list bounds, clone isolation and disk round trips. Native PIDL
fixtures repeatedly parse/free only an inert temporary source; they do not open
Explorer or delete/change source links. Actual Explorer navigation and desktop
menu input remain **not tested**.
Local Go 1.27.2 module verification, full tests, vet and Windows GUI build passed.
The independent-provider regressions cover Shell error/deadline with a complete
file snapshot, subsequent successful empty Shell recovery, and an empty partial
snapshot after removing a root/disabling Shell. The reconciled worker snapshot
cannot be overwritten by the UI's older fallback; old icons will be preserved
in the following scheduling/cache correction stage.

## 0.2.0-alpha.1 stage 4: scheduling and cache corrections

Healthy file sources no longer receive periodic full scans. Accelerated native
watch tests leave a real fixture unchanged across multiple retry periods and
require zero refresh callbacks, then exercise a merged 16-write burst, same-path
directory replacement/rebind, running cancellation and closed native handles.
Separate Shell-timer tests require Shell-only requests. Launcher tests require
zero file work on a Shell-only refresh, preserved old file/Shell icons on source
failure, and rejection of retired watch epochs/quit callbacks. All use synthetic
fixtures. Full dependency/test/vet/GUI-build checks pass locally; actual idle
power consumption and third-party Shell-handler behavior remain **not tested**.

## 0.2.0-alpha.1 stage 5: explicit startup

Pure tests cover default-off settings, quoted bounded command construction,
explicit enable/disable, collisions, registration errors, save/hotkey failure
rollback and external-replacement preservation. Headless 100%/150% tests toggle
the draft, cancel without effects, then explicitly save. Ordinary theme saves
must not touch registration. Background routing recognizes only `--background`;
the existing MyGo single-instance lock remains in use and duplicate background
invocation does not show the palette.

The sandbox account could not create the isolated HKCU native fixture (access
denied), so the full Go 1.27.2 verify/test/vet/build pass was repeated in the
machine's standard-user context. Its registry test creates one random volatile
Software fixture outside Run/RunOnce, repeatedly checks UTF-16 REG_SZ round trips,
disable/save-failure restoration, unrelated-value preservation and foreign-command
refusal, then removes that own fixture key. No real startup value was installed
or changed. No user clipboard or app inventory is accessed by these tests.
Actual Windows login startup, tray-focus behavior and second-process desktop
presentation remain **not tested**, alongside incremental launch/input gates.

## Candidate review recovery corrections

Fault injection now keeps a readable configured root's subscription failing and
requires a two-minute degraded fallback scoped to only that root; accelerated
tests prove the healthy root is excluded. Unfinished file snapshots schedule one
30/60/120-second retry, capped at five minutes, scoped to unfinished roots. A
successful retry clears the failure/timer without a directory event. Fifty rapid
failure publications reuse one timer; retired configuration epochs reject posted
retry callbacks. A file-stage deadline with Shell never executed retains both
providers and schedules recovery. Retry/watch work remain serialized with the
existing scan worker and cooperative budget/cancellation limits.

MyGo v0.4.6's second-instance message already forwards `os.Args[1:]`. The callback
now checks that actual shape directly, avoiding a second slice that opened an
existing window for `--background`. `TestSecondInstanceCallbackReceivesArgsWithoutExecutable`
checks background/normal/other invocations through the same routing helper.
Full local verify/test/vet/GUI-build checks passed after these corrections; real
IPC/login/focus remains separately **not tested**.

## Alpha verification boundary

Native IME, successful automatic paste, tray/hotkey, high-DPI and multi-monitor
checks remain unverified or incomplete. These are accepted first-alpha limits,
not passes. Exact-commit automated/CI/privacy/license/package checks and final
owner review remain required before publication.

## Original Windows icon

The [original SVG, synthetic previews and reproducible pipeline](ICONS.md) cover
eight transparent PNG/ICO sizes. The EXE verifier requires RT_GROUP_ICON and all
eight RT_ICON payloads to match the source ICO. A dedicated hidden MyGo window
checks valid small/large native handles and bitmap sizes, repeated replacement
and restoration; it does not scan apps or touch clipboard/input. Local taskbar,
Explorer/tray appearance, shell cache and DPI transitions are **not visually
tested**, and are recorded separately from resource/ABI evidence.

## UI foundation

Runtime/UI alignment commit: `0ec65e58a56e41ce96e79c8bf4c12bfd8491a546`.
Verified using official Go 1.27.2, MyGo v0.4.6 and the Windows amd64 build.

| Check | Result | Evidence/limit |
| --- | --- | --- |
| Dependency integrity | Passed | `go mod verify` |
| Unit/native fixture/UI tests | Passed | `go test ./...`; synthetic shortcuts and text only |
| Static checks | Passed | `go vet ./...` |
| Windows GUI build | Passed | README command, no inspector, CGO disabled |
| Synthetic 100%/150% DPI | Passed and visually inspected | `docs/previews`; not desktop screenshots |
| First/last selection and scroll reset | Passed in headless harness | Both rows kept outside floating bars; wheel position not pulled back |
| Menu close/click-away/mode focus | Passed in headless harness | Search query frozen while menu is open; close restores focus |
| Menu Down/Enter highlighted action | Passed after a reproduced regression fix | Actual tester click and key events; menu has a non-clickable keyboard focus owner |
| IME candidate navigation guard | Passed in headless harness | Actual Microsoft Pinyin confirmation/cancel remains untested |
| Vibrancy fallback and corner alpha | Passed in headless harness | Simulated material availability; no desktop backdrop sampled |
| Hotkey, tray, focus loss and application launch | Not verified by automated checks | Actual native interactions remain an accepted alpha limitation |
| Actual clipboard opt-in, copy and verified-target paste | Not verified by automated checks | Successful automatic paste remains unconfirmed |
| Native Acrylic/corner clipping/transparent edges | Not tested | Renderer tests do not prove OS composition |

The current edge fade is a supported per-row alpha approximation rather than a
pixel gradient mask. No desktop interaction should be described as passed from
the headless tests, a process-only smoke check or a successful build.

## Windows UI feedback (2026-10-13)

The user reported material appearance, visible corner blocks, footer alignment
and visual corruption in the theme dropdown. Selection behavior was not reported
broken. The two private user images were not accessible through this execution
environment's Library download tools and were **not viewed locally** or committed.
Source diagnosis and synthetic render evidence are distinct from those images.

The theme no longer makes all widget backgrounds transparent or pairs a white
accent with white accent text. Only the root rendering surface is cleared. The
same saved appearance is applied to MyGo before native creation and after a
successful settings save; cancel/failed save cannot change native appearance.
The full client scrim has no larger renderer corner cutout; DWM receives the
standard rounded-corner hint and an explicit Desktop Acrylic request, with
HRESULT/readback checks. Unsupported material renders an opaque fallback.

Automated checks cover deep/light 100%/150%, opaque expanded dropdown geometry,
all three pointer choices, keyboard choice/Enter/Escape, click-away consumption,
cancel/save/reopen, settings IME Escape, footer line boxes and actual CJK ink
centering/cropping. A task-owned invisible HWND checks native DWM argument ABI
and attribute acceptance; it does **not** prove visible blur or native rounding.
Synthetic images are visually inspected in [UI_FEEDBACK.md](UI_FEEDBACK.md).

Actual material/corner clipping, Microsoft Pinyin and desktop pointer appearance
remain **not tested by this agent**: the current controller exposes only browser
control, without native app/window selection or screenshot/input APIs.

## Settings phase

Go 1.27.2 / MyGo v0.4.6 local tests, vet and Windows build passed. Storage tests
exercise missing-file defaults, first save and replacement, corrupt/unknown/newer/
oversized fallback without overwrite, external-edit preservation and temporary
file cleanup. Shortcut mocks verify candidate registration, rollback on save
failure and removal of the old registration only after success.

Headless 100%/150% tests exercise opening via the menu, field focus, covered-result
protection, invalid/failed saves, Cancel focus restoration, IME Escape protection,
theme changes and separation from session clipboard state. The synthetic previews
in [settings](SETTINGS.md) were visually inspected. Actual native shortcut conflict,
Microsoft Pinyin and theme/material changes remain **not tested**.

## Settings review corrections

Local tests exercise remote/unknown volume refusal before path attributes,
reparse-ancestor refusal before child lookup, native metadata on a temporary local
fixture, bounded traversal/resolution cancellation, and 50 rapid root changes
with a blocked old worker. They confirm at most one active worker, latest-root
coalescing and no stale result publication. No network share or mapped-drive
resource was accessed or configured for these tests.

First settings creation now refuses an appeared destination in the native move
operation; existing settings round-trip replacement and cleanup are tested.
Windows filesystem-wide atomicity/crash durability is explicitly not claimed.

## Finding applications phase

Local unit/native/UI tests, module integrity, vet and Windows build passed with
pinned go-pinyin v0.21.0. Tests cover full pinyin/initials and mixed Latin names,
bounded polyphonic expansion, alias removal without accumulation, unchanged launch
identity, separate namesakes, match-class ranking, bounded metadata persistence,
clone isolation, successful/failed launch adapters and save failure. Alias editor
100%/150% previews were inspected and controls remain within the sheet.

Native app startup, actual disk permissions in the user's application-data
directory and real keyboard/IME interaction remain **not tested**. No user
application inventory, document or clipboard data was used in shared fixtures.

Review regressions additionally cover failed startup hotkey registration with
successful favorites/aliases/recents and unrelated theme persistence; explicit
hotkey changes still register and reject conflicts. At 100%/150%, actual headless
Down/Enter events select the non-first B row, launch through a fake adapter,
rerender recents as B/A/C and press Enter again. Both calls target B, whose selected
row stays visible in the no-tray path. Native desktop access remains untested.

## Clipboard usability phase

Local tests, integrity verification, vet and Windows build passed. Deterministic
clock tests verify expiry, duplicate refresh, exact deletion, shortened retention
and session defaults. Headless 100%/150% tests render full synthetic multiline/
Chinese/emoji text, scroll to a dedicated end marker, copy its exact complete
value through a mock writer, delete/clear preview references and restore focus.
New entries retain highlighted identity; an Enter after removal does not copy a
replacement. No existing user clipboard content was read, restored or recorded.
The synthetic preview images were visually inspected. Native collection, expiry
tick scheduling, copy/verified-target paste and real IME remain **not tested**.

## Small utilities phase

Integrity, tests, vet and Windows build passed with the pinned toolchain. Parser
tests cover precedence, unary signs, decimals/exponents, glyph normalization,
division by zero, nonfinite results and length/token/depth limits. URL tests reject
unsupported protocols, embedded credentials, invalid ports/control data and raw
unsafe characters/oversized canonical encodings. Settings tests cover bounded link
validation, clone isolation and round-trip persistence. Headless 100%/150% tests
verify explicit result copying, no effect on Enter/save, link editing/cancellation,
invalid-draft preservation, explicit mocked opening and IME Escape protection.
The four synthetic previews were visually inspected. No URL was opened or fetched
and no existing clipboard was read during these tests.

Native desktop input/material interactions remain unverified by the automated checks.

## Minimal manual Windows pass

The clipboard menu review regression covers 24 combinations of deletion/expiry,
Enter/copy/delete, an optional render before input, and copy/paste modes. The menu
captures its target identity on open and resolves it again before effects. Tests
use dedicated A/B text, a deterministic clock and mocked copy/paste adapters:
no adapter is called and B remains unchanged after A disappears. A stale follow-up
Enter remains blocked; explicit reselection restores actions. Mode and query
changes retain ordinary focus/filter behavior.

Use a standard account and a temporary text window with dedicated test text.
Do not copy or expose an existing clipboard value. Leave startup integration off.

1. Start once, check tray access, then start again and confirm one instance.
2. Invoke/hide with the default hotkey; check arrows, Enter, Escape, close and
   focus-loss behavior. Test configured hotkey conflict recovery and metadata
   access. Both first and last results must be reachable at 100%/150%.
3. With Microsoft Pinyin, compose a synthetic application query. Candidate Enter
   confirms composition without launching; candidate Escape cancels without
   hiding. Then test ordinary Enter/Escape after composition ends.
4. Use a dedicated temporary application shortcut/executable fixture. Check
   normal launch, missing-target feedback, duplicate merging, and distinct
   argument/working-directory/permission/show-mode variants.
5. Confirm no history appears by default. Set dedicated text before enabling;
   enable recording and confirm that value is not imported. Copy new dedicated
   text, verify dedupe, preview, pause, clear and restart-empty behavior.
6. Invoke from the temporary text window, select dedicated text and test return
   to that exact target. Close or change the target and confirm safe refusal or
   copy-only fallback. Tray invocation must not guess a paste target.
7. Over a bright background, inspect Acrylic availability/fallback, true window
   corners and transparent edges. Test native menus, focus return and full-row
   visibility. Record visual limitations separately from functional failures.
8. Save ordinary theme/roots/favorites/aliases and a dedicated test link, restart
   and confirm those preferences round-trip while clipboard recording remains
   off and history empty. With a deliberately occupied hotkey, tray/window access
   and metadata saving must still work. In the no-tray path, launch a non-first
   fixture and press Enter again; selection must stay on the same identity.
   Check calculator explicit copy and open only the deliberately chosen test URL.

Record OS version, DPI, executable checksum, tested commit and passed/failed/not
tested results. Shared evidence must omit personal app inventories, documents
and clipboard contents. Cosmetic material limits may be documented; mandatory
launch/input/privacy failures must be resolved before public alpha readiness.

## Chinese-name follow-up (separate from status/chrome)

Synthetic UTF-8/BOM/UTF-16LE fixtures cover localized Chinese display selection,
original English/full-filename/duplicate search, retained Chinese names when a
resource is English, invalid/oversized metadata fallback, strict local resource
syntax, mapped-remote/reparse refusal before reads, cached results, cancellation,
directory/byte/resource budgets and independent launch variants. Pure identity
tests retain arguments/working directory/flags and favorite/recent bindings.
A generated PE containing only resources is read in data/image-resource mode to
check native ABI against its known exact synthetic string. It has no code section,
entrypoint or imports, checked separately with the standard PE reader. The earlier
system-resource check could accept any error and did not prove ABI success; it
has been replaced by this mandatory successful-read assertion. Headless localized
name previews use synthetic applications only. Actual installed-app naming and
locale-specific resource coverage await a user check. See [applications](APPLICATIONS.md).

Review correction: a one-character double-quote resource path previously reached
an invalid slice. Quoting is now validated for pairing, nonempty minimum length
and embedded/extra quotes before slicing. Malformed-reference table tests verify
original-name/identity fallback with zero resource opens; bounded pure-parser
fuzzing checks for panic and unsafe successful parses. No real user metadata is
used by these checks.

## OpenVPN result triage

Source inspection confirms that the prior index accepted .lnk by source extension
without target classification, while search excluded paths/targets/arguments but
allowed unlimited-gap subsequences over display names, aliases and pinyin.
Using the supplied SoftEther Chinese name and synthetic paths reproduced the
35-gap polyphonic `heng` match. Known-target fixtures now exclude ordinary
documents/directories; executable variants, independent/unknown links, Chinese
and English aliases, short initials, cancellation and locality guards are retained.
An integrated inert-file fixture returns only OpenVPN GUI for `openvpn` and keeps
the CLI searchable by `vpncmd`. No actual user shortcut target was read or changed;
the screenshot alone cannot establish the types of the user's six extra entries.

## Removed-target and rescan regressions

Native tests use only dedicated COM shortcuts/inert executable files. Both a
removed final executable and a removed target directory are omitted after a
fresh scan; deleted source links do not survive, special links remain, and
shortcut bytes are unchanged. Pure probes retain permission/not-ready/network/
reparse/unknown states and prove no deeper access after a refused/missing ancestor.

Headless 100%/150% tests preserve the publication-time query, selected ID, visible
anchor/offset and search focus while a worker runs. They cover subtitle height
changes, missing anchors/selection, Enter/primary actions after refresh and held pointer releases,
clipboard/app mode switches, empty results and cancel/error/partial/budget status.
One-worker rapid-request regression remains in place. Launch failures request a
fresh scan and do not blindly delete unconfirmed targets. See [details/previews](APPLICATIONS.md#confirmed-removed-executables-and-rescanning).
Actual installed-app/UI screenshot versions and target metadata are not inferred
from reported images; these tests do not prove a particular user's desktop rescan.
Additional regressions cover zero results plus a failed secondary root without
reviving a confirmed missing target, fresh shortcut metadata in fallback probes,
explicit hide/show guard reset, old held-pointer session isolation and list ink
clipping away from header/footer. Keyboard input after refresh is tested; a
native same-frame queued Enter boundary is not directly tested.
# 0.3 development evidence

Scope and compatibility design: [SEARCH_0.3.md](SEARCH_0.3.md).
Prepared-search differential/memo regressions and full local verify/test/vet/GUI
build pass. Synthetic benchmarks and preparation cost are recorded in the design
document. Snapshot safety/partial/generation/persistence regressions and
100%/150% options/pending-row controls pass in the synthetic harness; full local
checks pass. Actual winning-field/identity/adapter/source-batch diagnostics,
bounded memory, Esc layers and 100%/150% focus tests pass. Owned hidden subclass
and real hidden MyGoSurface repeat/IME pass-through/cleanup fixtures pass.
Full verify/test/vet/GUI build passes; new actual desktop keyboard/IME/pointer
acceptance is not claimed. The 0.2 candidate remains independently available.
Snapshot review regressions pass for initial partial -> independently recovered
sources -> full review -> missing pending removal without launch authority, and
slow write -> clear -> normal shutdown without UI dispatch. Clear timeout returns
an explicit bounded failure; abrupt termination/deletion on arbitrary filesystems
is not guaranteed.
F10 repeat checks include native WM_SYSKEYDOWN and the real hidden MyGo surface;
first press, Alt+F10/other system keys, Alt+F4 and composing messages pass through.
Actual desktop Shift+F10/IME acceptance remains not tested.

# 0.4 development evidence

Compact/expanded scope and compatibility design: [COMPACT_0.4.md](COMPACT_0.4.md).
Pure state truth table/transitions and old-config/opt-in/opt-out temporary-store
tests pass, as do full local verify/test/vet/Windows GUI build. Stage 2 native/DPI
fixture evidence follows. Actual multi-monitor/work-area/
IME behavior remains not tested; no display configuration is changed.
Integrated compact/expanded, hidden/transition Enter, first Down selection,
mounted search focus/composition, layer access/Esc/reopen, stable refresh, and
320-DIP settings/menu/first-last result tests pass at synthetic 100%/150%/200%.
Owned hidden window/real MyGo geometry and 64-cycle USER/GDI/input checks pass;
full local verify/test/vet/GUI build passes. Native DPI matrix and actual
multi-monitor/work-area/IME interaction remains not tested.

Integration commit `07a7d3a8a0ac15b73713a4faa3ba6026e358411c` passed
[exact Windows CI](https://github.com/feiniao233/MyCast/actions/runs/38464300823),
including owned native size/icon/key fixtures and packaging. Final candidate
verification remains separate; use [0.4 incremental acceptance](INCREMENTAL_ACCEPTANCE_0.4.md).

