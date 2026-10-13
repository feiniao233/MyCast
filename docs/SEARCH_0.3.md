# 0.3.0-alpha.1 search design and acceptance

Development starts at `772885ed4c344baed8ce53cb32da1c4131be9fea` on
`feat/0.3-search`. The published 0.1 tag/main and the separate 0.2 candidate are
unchanged. No 0.4 scope, plugins, AI, whole-disk index, signing or publication.

## Prepared search (first reviewable stage)

Prepare immutable folded name/alias/pinyin fields and identity/order keys when
the catalog changes. Each query searches the complete catalog, retaining the
existing exact/prefix/substring/subsequence scores and preference tie ordering.
Use the existing implementation as a differential oracle over synthetic Chinese,
pinyin, mixed-script, alias-removal, OpenVPN/SoftEther and Unicode cases. Measure
preparation separately from repeated queries; benchmark numbers describe only
that synthetic corpus and this task machine.

The UI may memoize one complete result using the catalog revision, exact input,
limit, favorites, recents and hidden identities. It must replace the slot on any
dependency change, including edits ending in whitespace. It must never search
only the previous query's results. Selection and viewport remain independent.

## Optional startup snapshot (second stage)

Implemented default off, separate bounded/versioned schema, source configuration fingerprint
and timestamp, explicit clear action. Store only display metadata and identities,
never clipboard/query history or another copy of launch arguments. Cached rows
are pending revalidation and cannot authorize launch. Source changes, malformed
paths, unknown formats and partial scans cannot silently grant authority or
overwrite a complete snapshot. Failures must not block ordinary discovery.

The schema stores only name/identity hashes, fingerprint and time: maximum
1 MiB, 5,000 unique entries, 512 UTF-8 bytes/name and seven-day age (at most
five minutes of future clock tolerance). JSON containing paths/unknown fields,
unknown versions, invalid IDs/text or trailing data is rejected. Unknown/corrupt
formats are preserved. An explicit clear can remove an oversized ordinary cache
file too. Same-directory synchronized replacement has no copy/delete fallback.
One active disk operation and one replaceable pending job keep writes bounded.
Epoch and source fingerprint checks prevent stale setting/source publication.
Partial scans keep unobserved rows pending, never as provider launch metadata.

The settings subpanel explains persisted data and pending authority. Its switch
is a draft until the ordinary Settings Save action; Cancel retains default off.
Clear is explicitly immediate and leaves original applications/settings intact.

## Keyboard and diagnostics (third stage)

Esc closes the active layer and restores search focus; repeats and IME remain
protected. Read-only diagnostics report the winning search field and actual
identity/adapter/source status evidence. Keep private inventory and query data
in bounded memory; do not export automatically.

The fixed Tinycast [UI specification](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/docs/ui.md)
was read. New modal metrics/colors remain in `palette_tokens_windows.go`;
existing readable Windows surfaces, clipping and native material fallback stay
as documented approximations. No macOS glass/refraction or desktop capture is added.

Implemented through the existing row-bound source/menu/status panel. The
matcher supplies its actual winning field and score; custom alias pinyin retains
separate derived provenance. Source records are bounded to 33, replace prior
records per configured source and reset when source settings change. A partial
batch is labeled unknown for unobserved entries, without inventing per-root
availability or uninstall conclusions. Diagnostics do not perform new discovery,
persist metadata or export an inventory/query.

MyGo 0.4.6 forwards native repeat flags to input handlers but its window shortcut
API does not expose them. A bounded same-thread subclass on this palette's
immediate `MyGoSurface` refuses repeated Enter/Esc/menu/F10 actions outside
composition, preserving first presses, navigation/text repeats, key releases and
system-key handling. IMM notifications and active UI-field composition pass
through. No global hook, keyboard injection or foreign-window subclass exists.

Review follow-up covers F10's real `WM_SYSKEYDOWN` delivery too: repeated F10
with no Alt context (including Shift+F10) is consumed outside composition.
The Alt-context flag, first press, all other system keys (including Alt+F4) and
IME pass through. Both the owned native receiver and actual hidden MyGoSurface
fixture now send `WM_SYSKEYDOWN`, rather than assuming F10 is only `WM_KEYDOWN`.
See the [system-key message contract](https://learn.microsoft.com/en-us/windows/win32/inputdev/wm-syskeydown).
Only the System32 `comctl32.dll` is loaded; numeric ownership tokens keep Go
pointers out of native reference data, and explicit removal/WM_NCDESTROY clean up.
Failure retains the original input path and displays an unavailable warning.
Held Enter therefore does not add repeated line breaks in MyCast's own multiline
editors outside composition; normal first Enter and text/navigation editing remain.

API contracts: [SetWindowSubclass](https://learn.microsoft.com/en-us/windows/win32/api/commctrl/nf-commctrl-setwindowsubclass),
[WM_KEYDOWN repeat bit](https://learn.microsoft.com/en-us/windows/win32/inputdev/wm-keydown),
[IME start](https://learn.microsoft.com/en-us/windows/win32/intl/wm-ime-startcomposition).
Controlled hidden windows test repeat/IME pass-through, ownership, duplicate
registration and cleanup. `scripts/verify-palette-keys` also validates the real
hidden MyGo drawing surface, not just a simulated callback. These fixtures do
not prove actual IME candidate behavior or desktop interaction.

## References and compatibility risks

Behavioral references, independently implemented in MyCast (no copied code):

- [Wox prepared matching, GPL-3.0](https://github.com/Wox-launcher/Wox/blob/8a3c3f67dbdf8ebc3bc9f2b30cd4015de125dfe7/wox.core/util/fuzzymatch/fuzzy_match.go).
- [Wox complete-catalog query behavior](https://github.com/Wox-launcher/Wox/blob/8a3c3f67dbdf8ebc3bc9f2b30cd4015de125dfe7/wox.core/plugin/system/app/app.go#L2254-L2304).
- [ZTools cache completeness, MIT](https://github.com/ZToolsCenter/ZTools/blob/6c9dee8c998ed7b4e6d0a80ad4311d3d8d24f72c/src/main/api/renderer/commands.ts).
- [Tinycast single-slot index memo, AGPL-3.0](https://github.com/abue-ammar/tinycast/blob/144727d2bb75b2e36d000b9f0f5563d359944463/Tinycast/Features/Launcher/Service/AppIndex.swift#L355-L378).

Main risks are stale memo dependencies, accidental rank changes, loss of aliases
on catalog replacement, and cached metadata being mistaken for launch proof.
Automated differential/native fixtures and synthetic 100%/150% UI evidence will
be labeled separately from actual user acceptance. Prior DTV and desktop
right-click acceptance remains not tested; Store launch and ordinary favorite
restart are the only relevant user-confirmed 0.2 checks.

## Evidence

Design recorded before runtime changes. Implementation, measured results,
exact-commit CI and package provenance will be recorded per stage.

Prepared-search stage: Go 1.27.2 `mod verify`, `test ./...`, `vet ./...` and
the required CGO-disabled GUI build pass locally. Differential synthetic corpus
covers 144 inputs, four limits, empty/preference modes, duplicate/empty identities,
Unicode, Chinese/pinyin/mixed text, aliases and stable ordering. The growing-query
sentinel recovers after an intermediate miss; actual Chinese pinyin still uses
MyCast's original semantics, rather than adopting Wox's syllable matcher.
UI single-slot regressions cover exact trailing-space input keys, in-place
favorite changes, recents, hidden rows, alias removal and catalog replacement.

Three one-second benchmark runs on this task machine (i5-10400, windows/amd64),
5,000 synthetic entries and a rotating six-query batch:

| Operation | Range | Bytes/query or preparation | Allocations |
| --- | --- | --- | --- |
| Existing search | 15.298–15.571 ms/query | about 5.95 MB | about 83,646 |
| Prepared search | 1.457–1.482 ms/query | about 1.06 MB | 263 |
| Catalog preparation | 13.857–13.980 ms/catalog | about 9.29 MB | about 107,942 |

This supports using prepared fields without a ranking change. It is not an
actual inventory or cold-start latency measurement; no universal speed promise.
The prepared catalog and one memo slot replace prior data rather than accumulating
queries. Preparation is currently lazy on catalog replacement on the UI thread;
the measured one-time cost is a known bound for this corpus, not a native-frame
guarantee. Desktop keyboard/IME acceptance remains not tested.

Snapshot stage: full local `mod verify / test ./... / vet ./... / GUI build`
passes after adding the separate options subpanel. Tests cover missing,
oversized, stale/future, corrupt/unknown format, invalid display IDs/paths,
source changes/disable, incomplete scan, superseded read/write, replacement
failure/temporary cleanup, explicit opt-in/restart/clear, pending effect refusal
and selection-preserving verified replacement. 100%/150% controls/list previews
are synthetic: [options 100%](previews/snapshot-settings-100.png),
[options 150%](previews/snapshot-settings-150.png),
[pending 100%](previews/snapshot-pending-100.png),
[pending 150%](previews/snapshot-pending-150.png).

A fresh test process with 512 dedicated synthetic files measured the first
100-result path at 38.7652 ms for discovery/preparation versus 7.0944 ms for
loading/preparing a display-only snapshot. OS file-cache state is uncontrolled;
window/font startup, COM icons and an actual user inventory are excluded. This
is fixture evidence, not a verified desktop cold-start latency promise.

Upgrade: a missing `startup_snapshot` preference remains false. Its separate
cache version cannot authorize launch or silently replace unknown-format data.
An older 0.2 executable preserves unfamiliar settings and falls back to defaults;
disable the snapshot and save in 0.3 before downgrading if the older settings
reader must continue using that file.

The pending display can apply already saved hash-only preference bindings to
retain favorite/alias search while waiting. The cache creates no new binding or
provider proof and never adds a launch path; effect refusal remains in place.

Independent cache review found two recovery gaps, repaired before delivery:

- If the first full scan was partial, independently successful root/provider
  retries previously could leave deleted display rows pending forever. Accepted
  per-source batch evidence now triggers one budgeted complete review when all
  currently configured sources have recovered. Until that review succeeds,
  unobserved rows stay pending and cannot launch. A partial review waits for a
  new recovery transition before another review; the existing serialized scan
  queue/cancellation/generation protections remain in use.
- An immediate clear queued behind a slow write previously depended on a UI
  completion callback which normal exit could abandon. A bounded autonomous
  disk worker now drains one pending job, prioritizing clear and dropping
  intermediate notifications. Shutdown invalidates saves, rejects later jobs,
  preserves queued clear, and waits up to 750 ms for requested clearing. Failed
  or timed-out clearing returns an explicit error that data may remain; arbitrary
  filesystem latency and abrupt process termination are not guarantees of deletion.

Regressions cover partial first scan -> separate root and Shell recovery ->
automatic full review -> removed pending row, with zero launch effects; repeated
recovery gating; slow save -> clear -> quit without UI dispatch; rejection of
late saves and bounded timeout failure. The earlier `9951218` package is an
intermediate build and will not be delivered as the reviewed candidate.

Diagnostics/repeat stage: full local verify/test/vet/GUI build passes, with actual
field provenance, bounded source records, 100%/150% menu/source/Esc focus tests,
native ownership/repeat/IME/cleanup fixture and real hidden MyGoSurface verifier.
Synthetic [100% diagnostics](previews/search-diagnostics-100.png) and
[150% diagnostics](previews/search-diagnostics-150.png) have been visually checked.
Prepared-search CI: [38446720021](https://github.com/feiniao233/MyCast/actions/runs/38446720021)
at `c8d1dc6d5b0fe84806f12988dafcafac3ff7ba72` passes. Snapshot CI:
[38448917388](https://github.com/feiniao233/MyCast/actions/runs/38448917388) at
`f9c0a82690c1a00b216d8ded95819ca9f8881cc0` passes. Final candidate CI/package
provenance is tracked separately; these are not Release publication events.
