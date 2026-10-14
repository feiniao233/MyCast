# 0.4 source-policy and one-row feedback correction

Baseline: `420159838b917fa38386fc87b320acd2cf0b8911`, branch
`feat/0.4-compact`. This is a development correction, not a Release or a claim
that Windows desktop acceptance has passed.

## Application sources

Ordinary desktop launchers come from configured file roots. The supplemental
Shell source includes verified packaged applications only, deduplicated by
case-insensitive AUMID. A broad AppsFolder desktop catalog is no longer added as
a second set of ordinary desktop launchers. Existing `.exe` support in explicitly
configured roots is retained.

Shortcut deduplication uses the case-insensitive original source name and target
path, falling back to the shortcut path when the target is unavailable. Arguments
and working directory do not participate. Thus same-name/same-target variants
can collapse even when parameters or other shortcut launch settings differ.
This is an explicit product trade-off, not proof of equivalent process behavior.
The deterministic winning source remains the original `.lnk`; its bytes, arguments,
working directory and launch flags are not rewritten. Different names for one
executable remain distinct. Chinese display localization and aliases must not
change an already assigned identity.

Old identities of merged entries remain preference migration inputs for favorites,
recents, hidden state and aliases. Unknown or ambiguous relationships are not
invented. In particular a former desktop Shell item with no verified relationship
to a surviving shortcut cannot safely be assigned merely by matching a name.
The old preferences remain stored. Missing-target and uninstall filtering still
apply before duplicate selection; only confirmed missing executable targets are
removed, while unknown launch semantics stay conservative. Duplicate-source
paths stay in memory so a root-only refresh can also refresh that group’s other
already configured roots when its selected shortcut disappears; no real user shortcuts or inventories were read for this cloud correction.

The behavioral reference is ZTools at
[`6c9dee8`](https://github.com/ZToolsCenter/ZTools/blob/6c9dee8c998ed7b4e6d0a80ad4311d3d8d24f72c/src/main/core/commandScanner/windowsScanner.ts#L144-L161)
(MIT license checked). Its original-shortcut launch adapter and separate
[packaged AUMID deduplication](https://github.com/ZToolsCenter/ZTools/blob/6c9dee8c998ed7b4e6d0a80ad4311d3d8d24f72c/src/main/api/renderer/commands.ts#L294-L310)
are implemented independently in Go. No TS/C++ source or native module is bundled.
MyCast keeps bounded native reads and existing Shell activation rather than
adding a manifest parser or copying native-api implementation details.

## Compact layout

The compact palette is a 56-DIP single row: small menu, the same mounted search
editor, bounded scan/error indicator, and icon-only expand control. The full
footer and result list are rendered only when expanded. Search key, parent and
binding stay stable. Geometry readiness uses the actual presentation minimum,
so a valid single-row height does not permanently disable menu/expand controls.

Default remains off. Hidden-result Enter cannot launch; Down expands. Settings,
menus, clipboard and IME composition require expansion. Scanning does not drive
height changes. Existing top-anchor, DPI and work-area clamping remain in place.

## Verification boundary

This correction is developed on Linux using a workspace-local official Go 1.27.2
toolchain, with its archive checksum verified against the official download
manifest. No global Go configuration or toolchain was changed. Local pure tests,
Windows cross-compilation and compile-only Windows test checks are separate from
Windows execution. The exact checks and final results are recorded in
[acceptance](ACCEPTANCE.md).

Windows CI must execute the Windows suites and owned hidden geometry/key/icon
fixtures on the final commit. It also emits a separate
`MyCast-compact-synthetic-previews` artifact at 100%/150%/200% for visual review;
these are offscreen synthetic renders, not desktop screenshots. Live Windows UI, native menus, real Microsoft Pinyin,
Store activation, installed-app duplicates and 100%/150% rendered visuals remain
unverified by this Linux workspace. Compile-only success is not a passing Windows
test run. No merge, Release, signing or real account access is part of this change.

## Hover and count follow-up

User feedback after candidate `0699b6e` identified two separate presentation
issues. The result section used the capped returned slice (100), while the scan
status used the complete indexed catalog. Searching already examines the full
catalog; the cap is not a 100-application indexing limit. The follow-up keeps the
cap and bounded result rows, and distinguishes displayed rows from searchable or
matching totals.

MyGo 0.4.6's tooltip is an in-window overlay. Its default vertical placement
cannot fit around a 32-DIP button inside a 56-DIP palette; the overlay can exist
but be clipped out of view. Compact tooltip placement therefore needs an
in-window horizontal anchor, preserving the existing hover delay, dismissal,
pass-through behavior and search editor rather than changing window height.

Scan problems are bounded recorded errors from multiple sources, not a count of
unreadable directories. Both file scanning and Shell metadata collection have
128-entry bounds; package-proof failures are coalesced. The reported number alone
does not identify the user's failing source. Clearly excluded nonpackaged Shell
entries should be skipped before requiring display metadata, while unknown or
packaged failures remain partial-scan warnings.

The final simplified UI is planned separately: counts and scan detail belong in
diagnostics, with only brief scanning/attention indications on the main palette.
The present candidate retains accurate detail to help testing.
