# 0.4.0-alpha.1 incremental acceptance

Development candidate on `feat/0.4-compact`, based on reviewed 0.3 commit
`f941515aa562d4282435df82f0ca3568c7d99f76`. Published 0.1 and separate 0.2/0.3
candidates remain preserved. This version adds only compact/expanded presentation.
Use the delivered MANIFEST/BUILDINFO and ZIP checksum for the exact candidate.

| Check | Evidence and limit |
| --- | --- |
| Default off and missing old preference | Pure/temporary-store migration and save/restart tests pass; existing favorites/recents remain immediately visible |
| Unified size/result visibility decision | Full state truth table and transitions pass; result count/index/icon updates cannot change the decision |
| Empty opted-in call starts compact | 108-DIP synthetic opening and own hidden native/MyGo pre-Show sizing pass; no desktop flash claim |
| Enter cannot launch hidden or resizing results | Compact/transition action refusal tests pass, including direct activation and stale presentation guards |
| Down/visible expand selects first result | 100%/150%/200% synthetic focus, first selection and viewport reset pass |
| Typed/composing search expands without remounting editor | Synthetic input/composition/focus cases pass; actual Microsoft Pinyin and third-party IME transitions not tested |
| Settings, menus, clipboard and Esc layering | Synthetic access/focus/return-mode/reopen tests pass; manual expansion resets on the next call |
| Save applies on next call | Cancel/opt-in/save/current-full/next-compact tests pass; ordinary preference only |
| Work-area/DPI geometry and top anchor | Pure 96/144/192-DPI negative/constrained bounds and owned native RECT/MONITORINFO fixtures pass; real fixture uses current OS DPI only |
| Repeated expansion | 64 native/MyGo cycles preserve physical top/left and bounded USER/GDI counts; headless cycles retain icon pointers and bound callbacks |
| Constrained menu/settings/results | 320-DIP synthetic 100%/150%/200% first/last rows, keyboard-selected menu item and fixed Save/Cancel actions pass |
| Refresh and failed native sizing | Stable selection/viewport/no extra resize and readable sticky full fallback tests pass |
| Full inherited safeguards | Dependency verification, full tests, vet, CGO-disabled GUI build and existing native icon/key fixtures required on final commit |

Suggested checks on the exact candidate with dedicated synthetic text:

1. Leave the preference off and confirm the existing full empty-query panel.
2. Enable **空查询紧凑呼出**, save, hide and reopen. Search and footer should appear
   directly; Enter should launch nothing. Press Down and confirm first visible
   selection, then reopen to confirm temporary expansion resets.
3. Type/edit/clear a query; confirm the editor retains focus and text. Confirm and
   cancel real IME candidates before any launcher action. Check the same behavior
   at 100%/150%/200% using existing display settings, without publishing private
   application inventories or screenshots.
4. Open menus/settings/clipboard from compact and return to applications. Check
   Esc closes one layer, menu click-away consumes the covered click, and settings
   Save/Cancel remain visible in a smaller work area. Disable compact when done
   if the old presentation is preferred.
5. Check top-edge placement and row/menu access across monitors and work-area
   changes. Background indexing should preserve query, selection and viewport.

Actual desktop keyboard/IME, multi-monitor/work-area transitions and Windows
material/corner behavior are **not tested** by hidden fixtures or synthetic PNGs.
Unusually tiny work areas unable to fit even one expanded row remain
activation-guarded, without a support promise. New mandatory native acceptance
remains a release blocker. The prior owner acceptance is not upgraded to a pass.

No current-system startup enablement, clipboard read, display configuration,
global hook, external service, credentials, whole-disk search, AI, plugins,
automatic update or signing is added. The binary is unsigned; signing is deferred.
No main merge, tag or Release publication is authorized.

Data cleanup remains unchanged: disable explicitly enabled startup from its
registered executable before moving it; disable/clear the optional snapshot in
Settings. After quitting, optional removal of MyCast's own
`%APPDATA%\MyCast\settings.json` and `startup-snapshot.json` removes ordinary
preferences/display metadata. Compact adds only `compact_on_show` to settings;
clipboard history has no disk file.
