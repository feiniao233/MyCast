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

The version-1 pre-alpha schema accepts earlier ordinary settings without these
optional fields. Earlier builds that do not understand the new fields preserve
the file and use defaults instead of overwriting it. See [recovery](SETTINGS.md).

Synthetic alias editor previews: [100%](previews/aliases-100.png),
[150%](previews/aliases-150.png). These show dedicated fixture apps/text only.
