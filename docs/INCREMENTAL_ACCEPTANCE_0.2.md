# 0.2.0-alpha.1 incremental acceptance

This is a development candidate, unsigned by the user's deferred-signing decision.
It is not a main merge, version tag or published Release. The source commit and
checksums are recorded in the candidate MANIFEST, BUILDINFO and SHA256SUMS.

Automated and native-fixture checks are distinct from real desktop acceptance.
Existing 0.1 accepted limits are not reclassified as passes. New-version checks
below remain **not tested** until performed on the exact candidate by a user.
Use a standard Windows 10/11 x64 account and dedicated temporary fixture data.
Do not share a real installed-app list, private paths or clipboard screenshots.

| Incremental user check | Expected outcome | Current evidence |
| --- | --- | --- |
| Tab to a nonselected row, then menu key or Shift+F10; right-click search | Actions belong to focused row; search retains text editing context | 100%/150% headless regression; actual keyboard/pointer acceptance open |
| Search a desktop application exposed through both sources | One proved default profile; independent arguments/directory/mode entries stay separate | Directed single-item native properties and pure regressions; actual removal not tested |
| Right-click a scrolled row; use menu key/Shift+F10 | Captured ID, restored search focus and safe target loss | Headless 100%/150% hit/keyboard/dismissal tests; actual pointer behavior not tested |
| Enable AppsFolder and search a packaged and desktop entry | System-localized names/icons; each default action starts the chosen identity | Adapter/native Shell fixtures and mocked activation; actual app start not tested |
| Change a display name/rescan | Favorites, alias and recents remain on the same identity | Synthetic identity/routing regression |
| Keep same-name/argument/directory/permission shortcut variants | Unproven equivalence stays separate; original shortcut launches | Existing native/pure fixtures; cross-provider dedupe intentionally conservative |
| Install/uninstall or change a configured fixture shortcut | Healthy root notification refreshes; AppsFolder changes appear on its 30-second timer | Running native notification/rebind and serialized worker tests |
| Type, change selection and scroll during refresh at 100%/150% | Publication preserves current input, selected ID and visible anchor | Headless tests; actual desktop input not tested |
| Temporarily deny/disconnect a configured source | Unknown entries and prior icons remain; restore recovers; no false uninstall | Mixed source, deadline, empty snapshot and provider recovery regressions |
| View source/open file location | Original file selected; virtual location disabled | Synthetic menu and native PIDL fixtures; Explorer navigation not tested |
| Hide selection then press a pending Enter | Replacement cannot launch until explicit selection; restore from settings | Headless stale-menu/Enter/restore checks |
| Save ordinary preferences then restart | Hidden IDs/favorites/aliases/recents persist; clipboard recording/history reset | Validated temporary-store round trips and existing privacy tests |
| Enable startup in a dedicated test account, log out/in, then disable | Standard-user tray-only startup, one instance, no focus theft; only MyCast value removed | Isolated registry fixtures/mocks only; actual login not tested |
| Run twice with `--background`; then normal launch | One background instance remains hidden; normal invocation opens it | Flag/routing tests and existing MyGo single-instance mechanism; actual second process not tested |

Turn startup off from the registered executable before moving/deleting it. If a
foreign command or rollback error is reported, inspect only the MyCast own value;
do not change unrelated registration/security settings. Startup generation/tests
have not enabled the current user's real startup setting.

Known limits: cooperative native calls can outlast the scan deadline; third-party
Shell handlers/STA message-loop compatibility is not broadly validated; Shell-only
polling and cached icons have bounded background cost/staleness. Counts beyond the
catalog/icon budgets show partial results. No full-drive search, AI, plugins,
automatic upgrading or signing was added. Native Acrylic/corners/IME/tray/clipboard
limits remain separate in [ACCEPTANCE.md](ACCEPTANCE.md).
