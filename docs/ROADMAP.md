# Windows alpha roadmap

`0.4.0-alpha.1` begins from reviewed `f941515aa562d4282435df82f0ca3568c7d99f76`
on `feat/0.4-compact`. Scope is only compact/expanded presentation, preserving
search identity, input/IME, stable results and the Windows material fallback.
Default-off compatibility and state/geometry risks are recorded in
[COMPACT_0.4.md](COMPACT_0.4.md). Runtime integration and automated acceptance pass;
the separate 0.2/0.3 candidates and published main/tag remain preserved.
The pure unified layout decision and backward-compatible default-off preference
pass transition/temporary-store tests and full local verify/test/vet/GUI build.
Native sizing and presentation integration are described below.
Native own-window sizing and compact/expanded UI are now implemented with the
same mounted search editor, pre-Show dimensions, pending-geometry action guards
and scrollable constrained settings/menus. Full local checks, controlled native
64-cycle geometry/resource fixtures and 100%/150%/200% synthetic focus/input/
visibility/work-area regressions pass; new actual desktop acceptance remains open.

`0.3.0-alpha.1` development starts from the reviewed 0.2 candidate
`772885ed4c344baed8ce53cb32da1c4131be9fea` on `feat/0.3-search`.
The planned stages are prepared full-catalog search with ranking/benchmark
evidence, a default-off display-only startup snapshot, and keyboard/read-only
diagnostics. See [scope, references and compatibility risks](SEARCH_0.3.md).
Prepared full-catalog search and one UI memo slot are implemented with unchanged
ranking in differential tests. Full local checks pass; 5,000-entry synthetic
queries measure 1.457–1.482 ms versus 15.298–15.571 ms previously, with a separate
13.857–13.980 ms preparation cost. Default-off bounded display-only snapshots
and explicit clear are implemented with serial/epoch/completeness protections.
Full local checks and synthetic DPI controls pass. Read-only actual matching/
identity/source diagnostics and an owned native repeat-key guard are implemented,
with 100%/150% focus/layer tests and a real hidden MyGoSurface fixture.
New actual desktop keyboard/IME/pointer acceptance is not claimed.
Independent snapshot review follow-up converges pending rows after separately
recovered sources through one serialized complete review, and drains queued clear
without exit-time UI callbacks. Slow-write/clear/quit and scoped-recovery tests
pass; shutdown clear waits are bounded and timeout is reported honestly.
F10 review follow-up also protects its native WM_SYSKEYDOWN repeat path, retaining
Alt/system-key and IME handling, with real message-type/native MyGo fixtures.

The independent review follow-up separates AUMIDs during file scanning, keeps
property failures distinct from successful empty values, and gives each Tab-focus
row its own keyboard context target. Real temporary shortcut scan/merge/search/
save/restart and COM fault injection pass, as do 100%/150% focus/editor regressions.
Ambiguous legacy preferences remain stored without guessed reassignment.
The review-fix CI fixture now verifies its owned native target/directory spelling
before testing AppID association; the initial failing run is not reported passed.

Desktop-EXE/AppsFolder association and application-row right-click menus are
implemented, preserving launch variants and both-provider persisted preferences.
Full local checks and 100%/150% headless menus pass; new actual duplicate/menu
acceptance remains open. See [compatibility/evidence](DESKTOP_ASSOCIATION_CONTEXT_MENU.md).
Desktop association also passes the native volume/reparse gate; denied proof
retains the namespace row. Owned-target and injected unknown-target tests pass.

The approved Store/source follow-up now defaults AppsFolder on (explicit false
preserved), fixes bare-AUMID filtering/activation and Chinese Store search,
uses actual Known Folders with shallow default desktops, filters proved uninstall
actions and adds conservative PIDL-based cross-provider identity association.
See [findings, compatibility and verification](STORE_SOURCE_FIX.md).
The review follow-up persists bounded hash-only legacy preference bindings and
retains a proved Shell counterpart during file-only refresh. Restart/source-loss
and failed-save regressions pass; ordinary unproved desktop duplicates remain
distinct. Full local Go 1.27.2 verify/test/vet/GUI build passed for this change.
Confirmed file removal plus an incomplete Shell refresh retains the proved
namespace counterpart; unknown availability is not treated as an uninstall.

`0.1.0-alpha` is published at `5512e940e4693aabb56e02ac2cd0517e5f47dd26`.
The reviewed **0.2.0-alpha.1** candidate is preserved on `feat/0.2-applications`;
the current development candidate is **0.3.0-alpha.1** on `feat/0.3-search`.
Stage 1 adds an explicit AppsFolder source, stable Shell identities, localized
Shell names/icons and default-action activation; synthetic and native fixture
checks cover adapters. See [scope and compatibility](NEXT_ALPHA.md). Automatic
refresh is implemented with guarded directory notifications, 500-ms coalescing,
30-second scoped retries and conservative partial snapshot reconciliation.
Hide/source/location are implemented with identity-bound menus and bounded
persistent hidden IDs, restored through settings; explicit default-off per-user startup is implemented. Native
packaged/desktop activation remains a new-version acceptance blocker.

The first public alpha should be useful and predictable on Windows 10/11 before
adding other platforms or broad integrations. A phase is complete only when its
code, regression tests, build and stated manual checks have evidence.

| Phase | Scope | Acceptance | Current state |
| --- | --- | --- | --- |
| 0 — Native foundation | Palette, tray, hotkey, keyboard/IME protection, shortcut identity, bounded in-memory clipboard | Build/test/vet; synthetic DPI/scroll/menu tests; scoped actual Windows evidence | Code and local automated checks pass; Windows UI follow-ups and original taskbar/EXE icon are implemented. remaining native checks are accepted as deferred for this alpha |
| 1 — Settings | Validated settings storage with same-directory replacement; configurable hotkey, dark/light/system theme, application roots | Invalid/corrupt settings recover safely; failed hotkey change retains access; no clipboard import or startup registration | Implemented with storage/rollback and headless DPI/input/theme tests; desktop hotkey conflict and native appearance remain untested. See [settings](SETTINGS.md) |
| 2 — Finding applications | Favorites, successful-launch recents, stable ranking, aliases and offline pinyin search | No namesake or launch-semantic merging; repeated scans stay bounded; metadata saved only in app data | Implemented with identity/ranking/pinyin, bounded persistence and headless alias editor tests. See [applications](APPLICATIONS.md); actual app launch remains untested |
| 3 — Clipboard usability | Full bounded text preview, individual/whole-history cleanup, optional memory retention controls | Recording disabled on session start; only future opt-in text; clearing and expiry remove memory entries | Implemented with deterministic expiry and synthetic DPI/scroll/copy/delete/selection tests. See [clipboard](CLIPBOARD.md); actual desktop collection/paste remains untested |
| 4 — Small utilities | Safe local arithmetic and user-created HTTP(S) quicklinks | No code/shell evaluation; explicit copy/open actions; validated bounded input | Implemented with bounded parser/URL/storage and synthetic keyboard/DPI/mock-effect tests. See [utilities](UTILITIES.md); actual browser/copy remains untested |
| 5 — Alpha hardening | Error recovery, settings migration, packaging, notices, checksums, docs and manual validation | Exact-commit checks pass; known untested limits explicitly accepted; owner reviews final artifact | Clean-commit packaging/checksums, offline vendored-source build and full history checks are in place. User accepted deferred Windows checks on 2026-10-13; published as 0.1.0-alpha; exact-commit CI and candidate packaging for 0.2.0-alpha.1 are tracked below |

File-content search, window management, AI/network services, API keys and Raycast
extension compatibility are outside the alpha requirements. Compact/expanded
palette states are a later UI iteration, not a reason to delay the foundation.

Do not bundle all phases into one change. Complete and push a buildable phase,
report its evidence, accept review, and continue independent work. Repository
publication, main merges and Releases remain coordinated by the release owner.

## Windows UI feedback follow-up

`fix/windows-ui-feedback` starts from `9017fade4a975a7addbfae3e05a0975eb60f4dc3`.
It addresses the reported appearance of material, corner cutouts, footer text
alignment and theme-dropdown rendering. Dropdown selection itself was not
reported broken. Semantic theme surfaces and native appearance are synchronized;
DWM owns the outer corners, and footer glyphs use vectors with tested text ink.
See [UI feedback evidence](UI_FEEDBACK.md). Synthetic validation cannot close the
actual native material/corner and Microsoft Pinyin acceptance gates.

The completed status follow-up adds reliable full-status details and removes the unused
native caption-menu style on the palette, with explicit normal Alt+F4 close routing.
Chinese display-name enhancement was implemented in a separate change.

The independent Chinese-name follow-up reads bounded desktop.ini/resource data
only for indexed application sources, prioritizes available Han display names,
preserves English filenames/search and launch IDs, and retains existing scan
locality/cancellation limits. It adds no runtime dependency or system language
change. Installed-app naming coverage still requires a user check.

The application-result follow-up classifies known ordinary shortcut targets and
limits long-query scattered matches, including polyphonic pinyin false positives.
It retains executable/CLI and unknown/special entrances without changing launch
identity. GUI-only filtering and wrapper-argument interpretation are separate
product decisions, not inferred from screenshots.

The original Windows-icon follow-up adds the SVG/ICO/PE resource and MyGo
small/large window icons, with the same artwork on the tray. See [icon evidence](ICONS.md).
Native taskbar appearance is untested, rather than inferred from PE/ABI checks.

The removed-target/rescan follow-up omits only confirmed missing ordinary local
executables, adds immediate/completion feedback and preserves the current query,
stable selection and visible anchor during publication. Launch errors queue a
conservative refresh. Dedicated deletion/unknown-state and 100%/150% UI regressions
cover the behavior; actual installed-app verification remains separate. Publication
is paused until this experience follow-up and approved documentation cleanup are reviewed.

The first alpha accepts the documented unverified native limitations after
experience cleanup. This release decision does not convert untested checks into
passes. Icon/experience review precedes coordinated publication.

## 0.2.0-alpha.1 candidate stages

| Stage | Implementation/evidence | State |
| --- | --- | --- |
| 1 | Explicit AppsFolder source, names/icons, stable Shell identity and activation routing | Implemented; native Shell fixtures and headless DPI tests |
| 2 | Serialized automatic refresh and provider-specific partial preservation | Implemented; independent error/deadline/empty-source regressions |
| 3 | Source/location, persistent hide/restore and stale-action guards | Implemented; native PIDL and headless 100%/150% fixtures |
| 4 | Event-driven healthy file roots, Shell-only timer, root identity rebinding and icon caches | Implemented; running cancellation/coalescing/idle/epoch tests |
| 5 | Explicit default-off startup, ownership/rollback, background single-instance routing | Implemented; isolated registry fixture and simulated transitions |

No new main merge/tag/Release is authorized. Actual AppsFolder activation,
install/uninstall latency, idle power use, Explorer selection, login/background
focus, keyboard/IME and native menus remain incrementally unverified. Signature,
updater, AI/plugins and full-drive search remain outside scope. See
[candidate evidence](NEXT_ALPHA.md) and [acceptance](ACCEPTANCE.md).

Candidate review adds scoped two-minute fallback for readable failed-watch
roots, a single bounded exponential retry timer for unfinished file snapshots,
and the locked MyGo second-instance callback-shape correction. Healthy sources
are not periodically traversed. Fault-injection/deadline/epoch/routing regression
evidence is recorded in the candidate acceptance sections above.
