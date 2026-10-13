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

## Menus and verification

Right-click captures the hit row's stable ID and opens the existing menu beside
it, clamped between header/footer and inside the window. Ranking/selection changes
cannot transfer actions; a disappearing target closes safely. Right-click does
not activate. Click-away is consumed and Escape restores search focus. The
application footer Actions button is removed; clipboard keeps its existing control.
The menu key on the focused search field uses MyGo's ContextMenu hook to open
the same palette menu; a context request on that field also exposes the selected
application's actions. No native popup is built. With no result the editor can use
ordinary editing commands. Shift+F10 and IME action protection remain.

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
