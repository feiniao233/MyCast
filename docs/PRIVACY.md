# Privacy and configured-source boundaries

Current approved Store/source follow-up: AppsFolder defaults on; missing legacy fields migrate with notice, while explicit false remains off. Default desktop enumeration/watches are shallow and use actual Known Folder paths; custom roots remain recursive. File/Shell budgets are independent on one worker. See [current scope/evidence](STORE_SOURCE_FIX.md). Earlier off-by-default/shared-budget stage notes below are historical and superseded.


MyCast has no telemetry, online services, accounts, API keys or inventory upload.
Application discovery is local: configured application directories, plus the
current-user AppsFolder source only when explicitly enabled. Network/unknown
volumes and reparse ancestors are refused before filesystem indexing or watch
subscription. No full-drive or document-content search is added.

Names, icons and source details normally remain in memory. The new explicit
`startup_snapshot` preference defaults off. When enabled, a separate bounded
`startup-snapshot.json` in MyCast's own data directory may save display names and
identity hashes, source-configuration fingerprint and timestamp. It contains no
paths, launch arguments, clipboard, queries or equivalence proofs. Loaded rows
are labeled pending and cannot launch or open a location until rediscovered.
Disabling stops reads/writes; use Settings > startup snapshot options > clear to
remove existing data immediately. Unknown formats are preserved until explicit
clear. Only complete configured-source scans can update a snapshot.

Favorites, successful-launch
recents, aliases, hidden identity hashes and ordinary preferences may be saved
in MyCast's own settings file after user actions. Source details may reveal local
paths/arguments on screen; they are not logged, exported or saved as screenshots
by the application. Tests/previews contain dedicated synthetic data only.
For already saved preferences, a proved shortcut/Shell association may persist
as a bounded hash-only identity relation in settings. It includes no application
name, path, launch arguments or inventory and only links referenced preferences.

Clipboard recording starts disabled every session. Opt-in establishes a fresh
sequence baseline and collects only future bounded plain text. It does not import
the current clipboard, OS history or rich/image formats. Text/history remain
memory-only; ordinary settings never contain text or recording state. Paste
retains existing target/focus/modifier/sequence checks and never injects global
keys or elevates. See [clipboard limits](CLIPBOARD.md).

Startup defaults off. Only an explicit saved change writes the current-user
`MyCast.Launcher` Run value for the quoted current executable and `--background`.
No machine-wide registration, policy change, elevation, scheduled task or extra
credential is used. Disabling verifies its exact current command and touches
only its own value. Native tests create unique isolated volatile fixture keys
outside Run/RunOnce and remove them; they do not enable the user's startup.

Before moving/uninstalling, disable startup from the registered executable,
quit, then remove the extracted app. Optional ordinary data removal concerns
only `%APPDATA%\MyCast\settings.json` and the optional
`%APPDATA%\MyCast\startup-snapshot.json`. Clipboard history has no disk file.
