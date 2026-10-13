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

## Keyboard and diagnostics (third stage, not implemented yet)

Esc closes the active layer and restores search focus; repeats and IME remain
protected. Read-only diagnostics report the winning search field and actual
identity/adapter/source status evidence. Keep private inventory and query data
in bounded memory; do not export automatically.

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
