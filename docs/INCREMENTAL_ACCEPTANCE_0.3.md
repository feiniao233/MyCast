# 0.3.0-alpha.1 incremental acceptance

Development candidate only, unsigned by the user's explicit deferred-signing
decision. No main merge, tag or Release. Exact source and checksum provenance
is in MANIFEST/BUILDINFO/SHA256SUMS and the accompanying vendored source archive.

| Check | Evidence / remaining acceptance |
| --- | --- |
| Exact/prefix/substring/subsequence and preference ordering | Synthetic differential corpus; previous scores unchanged |
| Chinese, pinyin, mixed text, OpenVPN/SoftEther, alias removal, trailing spaces and match recovery | Core/full-catalog and UI memo regressions pass |
| Repeated-query performance | 5,000 synthetic entries measured; preparation separately recorded in SEARCH_0.3.md |
| Default-off cache and explicit setting/clear | Temporary-store and 100%/150% synthetic controls; actual user opt-in/restart/clear not tested |
| Cached rows cannot launch, open Explorer, grant equivalence or alter preferences | Effect-refusal regression; cached format has no paths/arguments/proofs |
| Partial/offline/unknown-source discovery | Pending rows remain unverified; complete-source-only writes and generation/config tests |
| Initial partial scan followed by independent source recovery | One serialized full review converges stale display rows; no inferred uninstall or launch permission before review |
| Slow snapshot write followed by clear and normal exit | Autonomous serial clear survives missing UI callbacks; at most 750 ms wait, explicit failure if not finished |
| Missing, corrupt, oversized, unknown/old format, stale/future time and failed cache replacement | Bounded temporary fixtures, original preservation and cleanup tests |
| Selection/query/viewport through verification | Stable-ID replacement and existing current-state refresh regressions |
| Read-only matching field, identity policy, current adapter and source batch diagnostics | Actual matcher evidence and memory-bound tests; 100%/150% synthetic panel |
| Esc returns from snapshot options to settings and then search; menu/source dismissal | Headless focus/layer tests; desktop keyboard/focus acceptance not tested |
| Auto-repeat Enter/Esc/menu key/F10 and IME pass-through | Owned hidden native subclass and real hidden MyGoSurface fixture; actual Microsoft Pinyin/third-party IME and held-key interactions not tested |
| Existing clipboard/startup/application safeguards | Full inherited automated/native fixtures; user startup remains disabled and no personal clipboard is read |

Suggested user checks on this exact candidate, using dedicated synthetic inputs:
edit Chinese/pinyin queries forward/backward, add/remove an alias, change favorite
or hidden state without changing the query, and inspect the winning diagnostic
field. Enable the snapshot explicitly, restart, verify pending rows refuse launch,
then wait for discovery and confirm the same identity/selection. Clear and disable
the snapshot. Hold Esc in a child layer: it should close only that layer rather
than continue into the launcher. Confirm/cancel actual IME candidates before
launching or dismissing anything. Do not publish an installed-app list, private
paths or clipboard screenshots.

Prior 0.2 user evidence covers Store launch and ordinary favorite restart only.
DTV, desktop right-click/removal, startup login and other previously unverified
desktop interactions remain not tested. They are not prerequisites retroactively
reclassified as failures/passes for beginning 0.3, and new mandatory desktop
acceptance remains a release blocker. Controlled compilation/process survival
and synthetic previews never substitute for those interactions.

Data cleanup: disable startup from its registered executable before moving it.
Disable/clear the optional snapshot in Settings. After quitting, optional removal
of MyCast's own `%APPDATA%\MyCast\settings.json` and `startup-snapshot.json`
removes ordinary preferences/display metadata. Clipboard history has no file.
No whole-disk search, AI, plugins, update service, credentials or OS policy changes.
