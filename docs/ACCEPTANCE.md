# Acceptance evidence and current status

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
| Hotkey, tray, focus loss and application launch | Not tested on the actual desktop | Desktop-control component unavailable |
| Actual clipboard opt-in, copy and verified-target paste | Not tested on the actual desktop | Automated tests use fake interfaces; no existing clipboard was read |
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
