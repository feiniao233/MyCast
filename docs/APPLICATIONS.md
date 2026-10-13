# Finding applications

The **操作** menu for the selected application offers favorite/unfavorite and
search alias editing. Favorites come first for an empty query, followed by the
most recent successful Shell launch calls, then name/path order. For a query,
exact matches still beat prefixes, then substrings, then subsequences; preferences
order results within a match class. Successful launch calls do not guarantee that
the target app finishes startup. Failed calls never enter recents.

Application names, original duplicate-link aliases and user aliases remain
searchable. Chinese names additionally support untoned full pinyin and initials,
using the bundled [go-pinyin v0.21.0](https://github.com/mozillazg/go-pinyin/tree/v0.21.0)
dictionary. No network lookup occurs at runtime. Latin text in mixed names is
preserved. Polyphonic expansion is limited to eight readings per step, with at
most sixteen terms per name; dictionary readings do not infer phrase context.
Missing/ambiguous readings can be supplemented by a user alias.

Derived search terms do not alter the app name, original shortcut, arguments,
working directory or special launch flags. Same-name apps keep separate stable
identities and subtitles. Repeated preparation replaces derived terms instead
of accumulating them; deleting a user alias removes its derived search terms.

Ordinary settings include at most 200 favorite identities, 50 recent identities,
and aliases for at most 200 applications, each with up to eight 80-character
aliases. Identities are hashes of the launch identity, not document indexes.
Metadata is saved only in MyCast's own settings file after the user's favorite,
alias or launch action. A save failure preserves the prior saved state. Use
**菜单 → 清空最近启动** to clear recents; remove favorites individually. Clipboard
history and recording state remain separate, session-only and absent from disk.

Ordinary metadata saving is independent of shortcut registration. A conflicted
startup hotkey does not prevent tray access, favorites, aliases, recents or theme
saving; only an explicit hotkey change attempts a new registration. Reordering
keeps the selected application by its stable identity and reveals its row. In
the no-tray fallback, the visible window therefore keeps the same selected app
after a launch updates recents, so a second Enter cannot launch a displaced row.

The version-1 pre-alpha schema accepts earlier ordinary settings without these
optional fields. Earlier builds that do not understand the new fields preserve
the file and use defaults instead of overwriting it. See [recovery](SETTINGS.md).

Synthetic alias editor previews: [100%](previews/aliases-100.png),
[150%](previews/aliases-150.png). These show dedicated fixture apps/text only.

## Localized Chinese display names

After the existing bounded index/identity merge, before pinyin preparation, each
already indexed source path can contribute display metadata. Proven duplicate
links retain temporary name-source paths so their localized names are not lost
when the preferred original shortcut is chosen. These paths are discarded after
name preparation, never serialized, and never affect launch identity.

Only those indexed files' directories are examined for desktop.ini
`[LocalizedFileNames]`, accepting UTF-8 (optional BOM) or valid UTF-16LE BOM.
Keys must be basenames of .lnk/.exe files. Valid bounded names containing Han text
take display priority; English resources cannot replace an existing Chinese name.
Without Chinese metadata, original branding remains as named. Both original
filenames and duplicate-link names remain searchable aliases, along with existing
user aliases and offline pinyin. Name changes leave Path, exact arguments,
working directory, special flags, ItemID, favorites and recents unchanged.

Indirect values support only `@absolute-local-dll-or-exe,-id` (IDs 1–65535).
Only SystemRoot/windir prefixes are expanded, using the Windows directory API;
unresolved variables, UNC/protocol/relative/stream/malformed references are refused.
INI and resource paths use the same local-volume/reparse-ancestor checks **before
opening**. Resources are mapped with DATAFILE_EXCLUSIVE | IMAGE_RESOURCE, never
initialized/executed or invoked through GetProcAddress. Module size is capped at
128 MiB and string buffers at 512 UTF-16 units. The current Windows language is
used; this feature does not change system or thread UI languages.

Per scan: at most 256 directory maps, 64 KiB per INI, 4 MiB read budget,
1024 entries per map and 256 resource results; missing/invalid results are cached.
Cancellation is checked between filesystem/native operations and shares the
existing scan deadline/serialized worker. Missing, corrupt, over-budget or
unavailable resources safely retain original names. No additional roots,
registry guesses, full-drive searches, Shortcut Resolve, external translation or
UWP scope is introduced.

The behavior was independently implemented after reading the public ZTools
scanner/reference script; its native implementation is not claimed known and
its binary is not used. See [notices](../THIRD_PARTY_NOTICES.md).
The actual user app list remains untested by this agent; synthetic mapping,
encoding/guard/budget/identity tests and a generated resource-only PE fixture
provide automated evidence. The native string test must return the fixture's
known exact string without error; an unavailable or broken reader cannot silently
pass it. Dedicated localized previews: [100%](previews/localized-applications-100.png),
[150%](previews/localized-applications-150.png).
