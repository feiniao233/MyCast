# Windows 0.1.0 alpha roadmap

The first public alpha should be useful and predictable on Windows 10/11 before
adding other platforms or broad integrations. A phase is complete only when its
code, regression tests, build and stated manual checks have evidence.

| Phase | Scope | Acceptance | Current state |
| --- | --- | --- | --- |
| 0 — Native foundation | Palette, tray, hotkey, keyboard/IME protection, shortcut identity, bounded in-memory clipboard | Build/test/vet; synthetic DPI/scroll/menu tests; actual Windows checks below | Code and local automated checks pass. UI alignment commit `0ec65e58a56e41ce96e79c8bf4c12bfd8491a546`; desktop interactions remain untested |
| 1 — Settings | Validated settings storage with same-directory replacement; configurable hotkey, dark/light/system theme, application roots | Invalid/corrupt settings recover safely; failed hotkey change retains access; no clipboard import or startup registration | Implemented with storage/rollback and headless DPI/input/theme tests; desktop hotkey conflict and native appearance remain untested. See [settings](SETTINGS.md) |
| 2 — Finding applications | Favorites, successful-launch recents, stable ranking, aliases and offline pinyin search | No namesake or launch-semantic merging; repeated scans stay bounded; metadata saved only in app data | Implemented with identity/ranking/pinyin, bounded persistence and headless alias editor tests. See [applications](APPLICATIONS.md); actual app launch remains untested |
| 3 — Clipboard usability | Full bounded text preview, individual/whole-history cleanup, optional memory retention controls | Recording disabled on session start; only future opt-in text; clearing and expiry remove memory entries | Implemented with deterministic expiry and synthetic DPI/scroll/copy/delete/selection tests. See [clipboard](CLIPBOARD.md); actual desktop collection/paste remains untested |
| 4 — Small utilities | Safe local arithmetic and user-created HTTP(S) quicklinks | No code/shell evaluation; explicit copy/open actions; validated bounded input | Implemented with bounded parser/URL/storage and synthetic keyboard/DPI/mock-effect tests. See [utilities](UTILITIES.md); actual browser/copy remains untested |
| 5 — Alpha hardening | Error recovery, settings migration, packaging, notices, checksums, docs and manual validation | Release checklist has no unresolved mandatory failures or untested items | Prepared clean-commit packaging with vendored corresponding source and checksums; history/notice checks underway. Public readiness remains blocked on actual Windows interaction evidence |

File-content search, window management, AI/network services, API keys and Raycast
extension compatibility are outside the alpha requirements. Compact/expanded
palette states are a later UI iteration, not a reason to delay the foundation.

Do not bundle all phases into one change. Complete and push a buildable phase,
report its evidence, accept review, and continue independent work. Repository
publication, main merges and Releases remain coordinated by the release owner.
