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

## Minimal manual Windows pass

Use a standard account and a temporary text window with dedicated test text.
Do not copy or expose an existing clipboard value. Leave startup integration off.

1. Start once, check tray access, then start again and confirm one instance.
2. Invoke/hide with the default hotkey; check arrows, Enter, Escape, close and
   focus-loss behavior. Test hotkey conflict recovery after configurable hotkeys
   are implemented. Both first and last results must be reachable at 100%/150%.
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

Record OS version, DPI, executable checksum, tested commit and passed/failed/not
tested results. Shared evidence must omit personal app inventories, documents
and clipboard contents. Cosmetic material limits may be documented; mandatory
launch/input/privacy failures must be resolved before public alpha readiness.
