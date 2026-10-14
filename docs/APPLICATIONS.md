# Finding applications

> Current 0.4 shortcut/source policy supersedes earlier argument-preserving
> deduplication descriptions below. See [feedback correction](FEEDBACK_0.4.md)
> for the deterministic name/target key, packaged supplement and migration limits.

Current desktop association and row context menus: see
[compatibility and verification](DESKTOP_ASSOCIATION_CONTEXT_MENU.md).

The approved 0.2 follow-up includes system AppsFolder applications by default,
fixes Store search/activation parsing, excludes proved uninstall actions and
adds a conservative virtual-shortcut identity bridge. It retains different
arguments, working directories and special modes. See [scope/evidence](STORE_SOURCE_FIX.md).

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

## Application targets and search precision

Ordinary readable .lnk metadata is now classified before identity merging.
Confirmed document targets (text/Markdown/RTF/HTML/PDF/help/Office formats) and
allowed local directory targets are omitted from application results, regardless
of their shortcut names. Local-volume/reparse checks precede target stat; no
network/reparse target is followed. Executable/CLI entries remain applications.
Unknown associations such as scripts/profiles, unavailable targets, unreadable or
virtual links and special/independent semantics are conservatively retained.
Executable hosts with arguments are not reinterpreted as documents. No name
blacklist, shortcut editing, resolution/repair, global GUI-only policy or weaker
deduplication is introduced. A retained link still launches its original path.

Search uses Name/Aliases/SearchTerms/UserAliases, never full paths, targets or
arguments. The old unrestricted subsequence rule let the polyphonic spelling
`softether vpn minglinghengshiyonggongju(vpncmd)` match `openvpn` across 35 gaps.
Longer queries (four or more runes) now permit at most twice the query length in
gaps; exact, prefix and contiguous matches are unaffected. One-to-three-rune
acronyms/pinyin initials retain prior behavior. Chinese display names, original
English/full-filename aliases, explicit user aliases and stable launch IDs remain.

The reported screenshots did not reveal real shortcut targets. A dedicated fixture
with a GUI executable, known HTML/text/directory links and the supplied SoftEther
name now returns only OpenVPN GUI for `openvpn`, while `vpncmd` still finds the
CLI executable. This is synthetic evidence; executable wrapper links and actual
user target coverage require a rescan/user check, not inference from label text.

## Confirmed removed executables and rescanning

Ordinary absolute `.exe` shortcut targets have a separate three-state metadata
probe: present, confirmed missing, or unknown. The probe checks local volume
type before path access, requires an available non-reparse directory root, then
checks ancestors in order. Only FILE_NOT_FOUND/PATH_NOT_FOUND under confirmed
fixed local storage establishes absence; it stops at a missing ancestor. Access
denial, not-ready/invalid-drive states, network/unconfirmed volumes and reparse
paths remain unknown. Unavailable removable/optical media is retained. The strict
LocalAppPath adapter used by other indexing/resource callers is unchanged.

A fresh index omits confirmed missing ordinary executable targets. Special,
virtual, relative and unreadable launch semantics are preserved. Rescanning
replaces the current catalog; favorites/recents rank present items only and cannot
recreate a removed source/target. Original shortcuts/files are never repaired or
deleted. A launch error alone does not establish absence: it queues the same
serialized metadata refresh rather than deleting the failed row immediately.

The footer shows scanning immediately ahead of existing diagnostic text and
reports completion, partial/budget results, cancellation or failure accurately.
Repeated requests coalesce through one worker; stale generations never publish.
Publication uses the current UI query/mode/selection and visible stable-ID anchor,
including intra-row offsets and changed duplicate/subtitle heights. It does not
call keyboard navigation/reveal, reset scroll to zero or restore input focus.
An application refresh leaves the clipboard viewport/selection/query alone.
If the selected application disappears, a nearby valid item is highlighted;
activation requires explicit selection/navigation so a pending old Enter/primary
action cannot launch its replacement. Result rows keep stable identity keys.

When zero new applications coincide with a failed root, the background worker
rechecks prior known source paths and freshly readable ordinary shortcut metadata.
The UI retains unknown entries but removes IDs whose absence was confirmed; an
unrelated failed root cannot restore those deleted targets. No cached target alone
is treated as proof, and cancelled generations still do not publish.

An explicit new show session resets the deleted-selection guard with the default
query/selection. Row, primary and action widgets use presentation-specific keys
so a held pointer release cannot transfer from the prior session. Fresh Enter
after the new presentation is tested; native cross-session OS key queues are not
claimed verified. Within the same presentation, the lost-selection guard remains.

The results scroll viewport is clipped between the fixed header and footer.
Content padding preserves row positions and scroll extent; neither fixed bar can
show result ink behind its text. The overall Acrylic/scrim and widget semantic
theme surfaces remain unchanged.

Dedicated COM shortcuts and inert executables test final-file/ancestor deletion,
source deletion, unchanged shortcut bytes, special semantics and stale preferences.
Headless 100%/150% tests cover scrolling/typing during a worker, mode changes,
changed row heights, deleted anchors/selections, old pointer releases, empty
results, cancel/error/budget feedback and serialized rapid requests. These are
synthetic fixtures, not a scan of any user's installed applications.
Keyboard coverage sends Enter after refresh; it does not directly exercise a
native same-frame queued key boundary. Held pointer releases are exercised across
both refresh and explicit hide/show, and raster comparisons check that changing
result text cannot alter header/footer pixels at 100%/150%.
The COM metadata wrapper declares its uintptr-pointer escape contract and passes
output buffers directly, preserving heap/liveness through wrapper stack growth.
Native repeated flags/persistence/read-only/deletion fixtures exercise that ABI;
missing interface outputs become errors rather than nil-pointer dereferences.
Previews: [scanning 100%](previews/rescan-progress-100.png),
[150%](previews/rescan-progress-150.png),
[completed 100%](previews/rescan-complete-100.png),
[150%](previews/rescan-complete-150.png).
