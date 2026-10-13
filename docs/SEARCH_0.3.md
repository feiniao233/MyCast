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

## Optional startup snapshot (second stage, not implemented yet)

Default off, separate bounded/versioned schema, source configuration fingerprint
and timestamp, explicit clear action. Store only display metadata and identities,
never clipboard/query history or another copy of launch arguments. Cached rows
are pending revalidation and cannot authorize launch. Source changes, malformed
paths, unknown formats and partial scans cannot silently grant authority or
overwrite a complete snapshot. Failures must not block ordinary discovery.

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
- [Tinycast single-slot index memo, AGPL-3.0](https://github.com/abue-ammar/tinycast/tree/144727d2bb75b2e36d000b9f0f5563d359944463).

Main risks are stale memo dependencies, accidental rank changes, loss of aliases
on catalog replacement, and cached metadata being mistaken for launch proof.
Automated differential/native fixtures and synthetic 100%/150% UI evidence will
be labeled separately from actual user acceptance. Prior DTV and desktop
right-click acceptance remains not tested; Store launch and ordinary favorite
restart are the only relevant user-confirmed 0.2 checks.

## Evidence

Design recorded before runtime changes. Implementation, measured results,
exact-commit CI and package provenance will be recorded per stage.
