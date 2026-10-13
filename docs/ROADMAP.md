# Windows alpha roadmap

`0.1.0-alpha` is published at `5512e940e4693aabb56e02ac2cd0517e5f47dd26`.
The next candidate is **0.2.0-alpha.1**, developed on `feat/0.2-applications`.
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
