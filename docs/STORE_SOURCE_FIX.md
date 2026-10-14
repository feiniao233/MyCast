# Store discovery, source scope and safe duplicate follow-up

> Historical implementation evidence below describes the previous broad
> AppsFolder/argument-equivalence policy. The current 0.4 feedback correction
> supersedes that policy; see [current source and compact changes](FEEDBACK_0.4.md).

This follow-up implements the approved defaults/scope changes after the first
0.2 candidate. It supersedes the earlier optional-off AppsFolder default in
[NEXT_ALPHA.md](NEXT_ALPHA.md), without changing startup/clipboard defaults.

## Targeted findings

Read-only checks on the selected machine queried only Microsoft.WindowsStore,
its public AUMID and the MyCast source switch; no complete installed-app list
was read or uploaded. Store was registered with status Ok. Its normal Shell
display name was `Microsoft Store`; the absolute parsing-name API returned the
bare `Microsoft.WindowsStore_8wekyb3d8bbwe!App`. The previous namespace-prefix-only
filter rejected that item. The old GUID-concatenated activation path returned
E_INVALIDARG, while the documented `shell:AppsFolder\<AUMID>` form resolved.
The old settings file also omitted the source flag, previously defaulting off.

The three private feedback images could not be materialized in this execution
environment: Library upload tools were exposed, but read/prepare_materialize was
not. Their actual pixels were not viewed or committed. User-reported queries
and the minimal native check are distinct from screenshot evidence.

## Resulting behavior

- AppsFolder is included by default. An absent legacy field uses the new
  authorized default with an in-memory migration notice; no file is rewritten
  during load. Explicit `apps_folder:false` remains off, and future saves retain
  the false field. Empty results explain a deliberately disabled source.
- Only entries coming from a proved AppsFolder root can normalize a bare AUMID
  to the prior canonical identity. Favorites keep the same canonical hash.
  Activation translates that validated identity to `shell:AppsFolder\…`.
  Search aliases for the exact public Store identity add 微软商店/微软应用商店
  while preserving the system-provided display name. Unrelated same-name apps
  do not receive those identity-specific aliases.
- Default sources use actual Known Folder Start Menu/Desktop locations with
  DONT_VERIFY. Local-volume/reparse gates still precede indexing and watches.
  Default desktops are shallow for both enumeration and notification scope;
  custom roots retain their original recursive meaning.
- File and Shell providers have independent cooperative 15-second budgets on
  the same worker, so file exhaustion does not suppress the Shell provider.
  The combined 5,000-entry limit reserves space for both providers rather than
  truncating all subsequently appended Shell rows. Native calls can still block
  beyond a cooperative deadline; no replacement worker is abandoned or added.
- Ordinary search excludes metadata proving an uninstall action: a recognized
  uninstall helper with an action label, or explicit uninstall/MSI-remove
  arguments. Names alone do not remove regular utilities such as Revo Uninstaller
  or 卸载工具. Source files are not changed/deleted and no uninstall is invoked.
- Cross-provider association requires the cached shortcut PIDL to be an
  immediate AppsFolder child, with no arguments/working directory and ordinary
  flags/show mode. No Resolve is called. Unknown/independent semantics remain
  distinct. Prior shortcut hashes remain bound to favorites, aliases, recents
  and hidden state, including fallback to a surviving Shell identity. A merged
  file row retains its proved Shell counterpart in memory for a file-only
  refresh, so removing the link does not remove the surviving namespace row.
  An incomplete Shell refresh also retains that counterpart when file removal
  was confirmed; Shell permission/deadline failures do not prove an uninstall.
- Referenced legacy preference hashes are bound to their proved canonical hash
  in bounded `identity_links` settings (at most 650 pairs). No names, paths or
  catalog are stored by this binding. Existing evidence is never silently
  redirected to a repurposed shortcut. The settings adapter saves a binding
  before treating it as persistent; failure preserves the previous file/config
  and shows a warning. Restart with only the Shell source restores all four
  preference kinds. Unreferenced bindings are pruned on a later scan.

## Verification boundary

Pure/headless tests cover normalization, both Store queries and pinyin, explicit
disable migration, shallow/custom roots, English/Chinese uninstall actions and
ordinary-tool counterexamples, capacity fairness, provider deadline isolation,
legacy preference editing/hiding/restoration and protected argument/directory/
permission variants. Native fixtures check Known Folder output ownership,
IShellItem2 property-key ABI and cached PIDL parent rejection/read-only shortcut
bytes. A minimal Store diagnostic using the production metadata routine returns
both English and Chinese matches; it does not activate Store or enumerate peers.
Persistence regressions cover shortcut removal followed by a file-only refresh,
restart with only Shell, all four preference kinds, rejected chained/retargeted
bindings and a simulated settings replacement failure with unchanged bytes.

Actual Store launch, the private screenshot examples and broad native desktop
appearance/input remain unverified; synthetic tests do not pass those gates.
Ordinary desktop EXE shortcuts without a proved AppsFolder PIDL relation may
remain separate from namespace entries; broad name/target merging is not claimed.

ZTools comparison: its public TS layer defaults UWP discovery on, uses a shallow
desktop scope and name-based uninstall patterns. Its lnk key uses name+target,
omitting argument/directory distinctions. Its native scanner/UWP `.node` modules
are binary-only in the inspected repository, so their internals are not claimed
verified. MyCast keeps its conservative launch semantics instead of copying
name-only merging.

Primary API references: [AUMID identification](https://learn.microsoft.com/en-us/windows/configuration/store/find-aumid),
[Known Folder IDs](https://learn.microsoft.com/en-us/windows/win32/shell/knownfolderid),
[Shell target property](https://learn.microsoft.com/en-us/windows/win32/properties/props-system-link-targetparsingpath),
[Shell arguments property](https://learn.microsoft.com/en-us/windows/win32/properties/props-system-link-arguments).
