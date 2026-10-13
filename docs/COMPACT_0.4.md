# 0.4.0-alpha.1 compact palette design and evidence

Baseline: reviewed `f941515aa562d4282435df82f0ca3568c7d99f76`, branch
`feat/0.4-compact`. The 0.2/0.3 candidates, published 0.1 main/tag and packages
stay separate. Scope is compact/expanded presentation only, with no link
parameters, plugins, AI, whole-disk search or new services.

## One state decision

A pure state function receives the compact preference, actual query, temporary
explicit expansion, composition and full-panel requirements. It returns both
desired window dimensions and result visibility. Result counts, indexing state,
icon changes and ranking are not inputs, so refresh cannot oscillate the window.
Clipboard, menus and all settings/subpanels require the full panel. The same
explicitly keyed search control stays mounted in both presentations.

Default **off** preserves the existing immediately visible favorites/recents and
keyboard habit. Missing old configuration remains off. Saving an opt-in keeps
the current presentation expanded; the preference takes effect on the next call.
For an opted-in empty application query, compact starts directly while hidden
before the first Show. Any typed text, including whitespace, expands without
rewriting input. Composition also keeps the full panel. Clearing typed text may
return to compact unless Down/the visible expand control explicitly expanded
this call. Temporary expansion resets on the next call.

Compact keeps search and existing footer access, at 108 DIP versus 475 DIP full
height; metrics/colors remain in `palette_tokens_windows.go`. It mounts no result
rows and cannot activate an invisible selection. Search Enter is consumed without
launching. Down explicitly expands, resets the viewport and selects the first
visible result. Up and result-action shortcuts cannot manipulate hidden rows.
Closing menus/settings retains existing Esc/focus rules; returning from clipboard
to applications resets temporary expansion and uses the preference clearly.

## Window and input compatibility

Use real MyGo 0.4.6 lifecycle/Update/NativeHandle APIs and a narrow own-window
Windows geometry adapter. Decide dimensions while hidden; establish the initial anchor using the
maximum expanded extent before compact sizing, so opening is not an expanded
flash. Later transitions keep the top edge where the work area permits and clamp
to the current own-window monitor work area. Do not change display configuration,
activate another window, add timers, decode icons again or install global hooks.

Size changes invalidate stale result pointer/session geometry, menu anchors and
viewport assumptions without changing search identity or IME composition. Menu
placement uses current viewport size. Existing UI-thread mutations continue via
Window.Update. Work-area/DPI/ownership failures retain an accessible full-panel
fallback and are reported; native acceptance is distinct from fixture evidence.
Scrollable results and settings must retain first/last item and action access
when the work area forces a smaller expanded size.

MyGo's public DIP position setters round coordinates at fractional DPI. The
adapter therefore uses validated owning-thread HWND geometry and SetWindowPos
in physical pixels, with NOACTIVATE/NOZORDER and no SHOWWINDOW. It reads bounded
native frame insets (0–128 pixels) rather than assuming a frameless window has
zero non-client border; desired dimensions describe the client area. Unknown
geometry fails closed. [SetWindowPos](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-setwindowpos),
[GetDpiForWindow](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getdpiforwindow),
and [GetMonitorInfoW](https://learn.microsoft.com/en-us/windows/win32/api/winuser/nf-winuser-getmonitorinfow)
contracts were checked. Native fixtures keep the exact physical top/left anchor
through 64 compact/full cycles; the first overly strict zero-border fixture
failed and was corrected after checking the real hidden MyGo window.

Keep Windows Acrylic and its verified-capability/solid fallback. No macOS glass,
refraction, desktop capture or custom material pipeline is added.

## Fixed behavioral references (independent implementation)

Tinycast, AGPL-3.0-or-later, at `144727d2bb75b2e36d000b9f0f5563d359944463`:

- [Central collapsed condition and explicit expansion](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/Tinycast/Palette/PaletteCoordinator.swift#L139-L155).
- [Hidden sizing and top-anchored expansion](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/Tinycast/Palette/PaletteWindowController.swift#L473-L492).
- [Collapsed Return guard](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/Tinycast/Palette/RootPaletteView.swift#L1482-L1489).
- [UI/token specification](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/docs/ui.md), read for 0.3 and reused for this narrow extension.

No Swift source is copied or translated. Windows geometry/behavior is implemented
independently with the locked Go 1.27.2/MyGo v0.4.6 runtime.

## Acceptance plan / current status

Design recorded before implementation. Pure transition/state tests first, then
owned hidden Windows/MyGo window fixtures and 100%/150%/200% synthetic compact,
expanded, settings/menu, composition and constrained-work-area regressions.
Check empty direct opening, typed expansion, Down first selection, hidden Enter
refusal, Esc layering, reopen reset, stable refresh/viewport and repeated expansion
resource/callback bounds. Each runtime stage requires full verify/test/vet/GUI
build and exact-commit CI before final source/license/privacy/hash/offline package
verification. Stage evidence follows; actual multi-monitor/work-area/IME behavior
is not tested. No native screen-control surface is currently exposed; controlled
windows and synthetic images will not be described as desktop screenshots.

Stage 1: `core.DecidePanelLayout` and default-off `compact_on_show` ordinary
preference are implemented. The 64-case pure state truth table and typed/explicit/
required-panel/reopen transitions pass; temporary-store old-config/opt-in/opt-out
round trips pass. Full Go 1.27.2 verify/test/vet/Windows GUI build passes locally.
Runtime sizing/visibility integration and new native/DPI fixtures follow in stage 2.

Stage 2 integration: same search key stays mounted; a single queued UI update
re-evaluates current state before resizing. Hidden sizing precedes Show. Enter,
result actions and stale pointer keys are protected while geometry is pending;
the renderer must observe the applied size before activation. Only result/footer
pointer identities change, never the search key. Expanded settings controls and
long menus scroll inside bounded viewports; Save/Cancel remain fixed and visible.

Tests pass at 100%/150%/200% for direct compact call, hidden/transition Enter
refusal, Down first selection, typed/composing input and focus preservation,
menu/settings/clipboard access, Esc/reopen reset, stable scan refresh/viewport,
320-DIP constrained controls, first/last rows and current-ID keyboard menus.
Native RECT/MONITORINFO layout, own-thread hidden popup geometry and real hidden
MyGo window resizing pass, preserving exact physical top/left and bounded USER/GDI
resource counts through 64 cycles.
Native tests do not show/activate windows or change display configuration.

Full local Go 1.27.2 verify/test/vet/CGO-disabled GUI build passes. Native DPI is
the current fixture's DPI; 100%/150%/200% geometry/renderer cases are synthetic,
not an OS display-setting matrix. Actual multi-monitor/work-area/IME acceptance
remains not tested. Native sizes too small to expose even one complete result
remain activation-guarded; unusual/tiny work areas are not promised supported.

Synthetic [compact 100%](previews/compact-empty-100.png),
[150%](previews/compact-empty-150.png), [200%](previews/compact-empty-200.png),
[expanded 200%](previews/compact-expanded-200.png),
[settings 200%](previews/compact-settings-200.png),
[constrained menu 200%](previews/compact-constrained-menu-200.png),
[constrained settings 200%](previews/compact-constrained-settings-200.png).
