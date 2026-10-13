# Desktop source association and row context menus

The parent task viewed the private duplicate screenshot; its bytes, paths and
application list are not included here. This Windows task queried only the two
supplied Shell identities. One returned the expected EXE target, matching
AppUserModelID and successful VT_EMPTY arguments. The other was unavailable;
that is not evidence of an uninstall or a passed native reproduction.
Exact shortcut basenames were also checked in the current configured roots,
without directory enumeration or source-path output. The available pair's real
shortcuts have matching target, empty arguments, EXE-parent directory, ordinary
flags and normal show mode; production association reduces each directed pair
to one row. This is a metadata reproduction, not a captured live launcher run.

## Compatibility rule

Namespace desktop candidates need a validated identity, matching
System.AppUserModel.ID, local EXE target and a successful empty argument read.
Packaged identities, failed properties and non-EXE/remote targets cannot supply
this association. Shortcut AppUserModelID is read through IPropertyStore;
Resolve, repair, execution and source writes are not used.
The file scanner includes each explicit AppID in its identity before deduplication,
so same-EXE shortcuts with different AppIDs cannot lose a source/name before
provider association. Read status is separate from the string: only S_OK string
or documented canonical-string success means known (VT_EMPTY requires S_OK);
QueryInterface/GetValue failures and unsupported types
keep the original shortcut independent and cannot authorize target-only inference.
The directed real shortcut returned S_OK VT_BSTR with the matching explicit
AppID. Both bounded LPWSTR and BSTR strings are supported and freed through
PropVariantClear; BSTR length comes from Windows OLE Automation. Null/empty BSTR,
length bounds, embedded NUL rejection and canonical-success codes are covered by
controlled native tests. The available real pair again reduces to one entry
through explicit identity matching, not failed-read inference.
Before desktop proof, the existing native volume/ancestor gate also rejects
mapped network/unknown/offline/reparse targets. A failed gate retains the Shell
row independently; it does not declare an uninstall or open target content.

Names never participate. An explicit shortcut AppID must agree. Without one,
exactly one qualifying namespace identity may identify the target; ambiguous
shared-host candidates remain separate. Arguments must be empty and original
shortcut flags/show mode ordinary. Only empty directory or the exact EXE parent
(case/trailing separator normalized) is the default desktop compatibility profile.
Other directories, arguments, permission and show-mode variants stay independent.
Normalization is limited to association, not a claim that arbitrary process
working directories are interchangeable. The original shortcut remains the launch
adapter with its directory unchanged. Association uses target/AppID maps rather
than a quadratic catalog comparison.

The canonical namespace ID retains old target/argument/directory preference
hashes. Referenced hash-only bindings persist and apply when either provider
survives. Applying saved preferences cannot invent a reverse proof; no application
names, paths or catalog are added to settings by the binding.
Favorite/alias/hidden/recent saves also bind their referenced proof immediately,
so an exit before the next automatic scan cannot omit the relation.
Older target/argument/directory hashes are retained only for one unambiguous
known file identity. Conflicting or unreadable AppIDs do not authorize a guessed
migration; ordinary preferences remain stored for the user to reassign. Existing
proved persisted associations are preserved. Real temporary shortcut regressions
exercise scan, provider merge, search routing and preference save/restart together.

## Menus and verification

Right-click captures the hit row's stable ID and opens the existing menu beside
it, clamped between header/footer and inside the window. Ranking/selection changes
cannot transfer actions; a disappearing target closes safely. Right-click does
not activate. Click-away is consumed and Escape restores search focus. The
application footer Actions button is removed; clipboard keeps its existing control.
Each result row owns a ContextMenu hook: mouse context, menu key and Shift+F10
use that actual row's ID even when Tab focus differs from selection. No native
popup is built for application rows. Search-field Shift+F10 operates on selection;
mouse right-click and the menu key in that editor retain its original text menu.
MyGo v0.4.6's public Menu callback does not expose pointer/keyboard trigger origin;
this minimal focus-based split preserves editor behavior without input hooks.
IME protection, click-away consumption and search-focus restoration remain.

Local Go 1.27.2 verify/test/vet/GUI build passed. Pure regressions cover default
directories, ID conflicts, ambiguity and launch variants. Native temporary links
exercise AppID/PROPVARIANT ABI, repeated cleanup/GC and unchanged bytes. Settings
round trips cover both surviving providers and all preference kinds. 100%/150%
headless tests cover scrolling/right-click hit testing, stable action IDs,
keyboard entry, dismissal and stale input. [100%](previews/context-menu-100.png)
and [150%](previews/context-menu-150.png) are visually inspected synthetic previews,
not desktop captures. Actual new duplicate removal/menu behavior is not tested.

User feedback on fd2d141 confirms Microsoft Store starts and ordinary favorites
survive restart only; it does not pass hidden/alias/source-removal paths or all
AppsFolder activation, nor these new changes.

References: [AppUserModel.ID](https://learn.microsoft.com/en-us/windows/win32/properties/props-system-appusermodel-id),
[Shell target parsing path](https://learn.microsoft.com/en-us/windows/win32/properties/props-system-link-targetparsingpath),
MyGo v0.4.6 ContextMenu/RightClicked/Bounds/PointerPosition APIs, and
[Tinycast UI specification at 9aea65b](https://github.com/abue-ammar/tinycast/blob/9aea65b8d86b164dfb6e2375090fba625112540f/docs/ui.md).
Tokenized geometry, mounted search and the readable Windows menu surface remain.
No upstream implementation was copied or translated.
