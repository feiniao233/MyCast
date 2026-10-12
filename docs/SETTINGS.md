# Ordinary settings

Open **菜单 → 设置**. Save explicitly; Cancel or Escape discards the draft.
IME candidate Escape/Enter is handled by the editor before launcher actions.

The version-1 JSON file is `%APPDATA%\MyCast\settings.json`, created when
ordinary settings or application preferences are saved. It contains the hotkey,
appearance, application roots and bounded [application metadata](APPLICATIONS.md).
Clipboard recording state and text are absent. Recording starts disabled every
session, regardless of ordinary settings; existing clipboard text is not imported.

- Hotkey: Ctrl/Alt/Shift with a letter, digit, Space or F1–F24. The new shortcut
  must register before the old shortcut is removed. Registration or saving failure
  keeps the old registration; tray access remains available.
- Appearance: dark, light or system. High contrast uses opaque surfaces and full
  text contrast. Native Acrylic availability and system changes require desktop
  validation; synthetic renderer evidence does not prove OS composition.
- Application roots: at most 32 absolute local paths, one per line. Empty means
  the usual Windows application directories. Only `.lnk` and `.exe` are indexed,
  without following symlinks or reading document content. Root changes start a new
  scan; stale scan completions cannot replace the new result set. Before any
  indexing access, Windows volume metadata rejects mapped network/unknown drives.
  Reparse points and ancestors are rejected before deeper lookups, including
  junctions and remote placeholders. Both default and explicit roots use this gate.

Scans use one worker. Changes cancel the current scan and coalesce into the latest
roots, waiting for its current native/filesystem call to return before starting
the next worker. Cancellation is cooperative, so a blocked Windows call cannot
be forcibly interrupted. Each scan has a 15-second deadline, at most 20,000 visited
entries, 5,000 apps, 32 roots/levels, 128 reported problems and 256 icon reads.
Directory entries are read in batches of 128. Budget exhaustion shows a partial
index and a prompt to narrow the roots; generic icons cover unread icons.

Saving validates bounded UTF-8 input, writes and synchronizes a temporary file in
the same directory, then requests a same-volume Windows
[MoveFileExW](https://learn.microsoft.com/en-us/windows/win32/api/winbase/nf-winbase-movefileexw)
move with WRITE_THROUGH. Existing destinations use REPLACE_EXISTING; first creation
omits it, so a newly appeared file is refused. No copy/delete, reboot or in-place
write fallback is used. This is **not a general atomicity or crash-durability
guarantee** across Windows filesystems; Go's non-Unix Rename likewise does not
promise atomicity. Live preferences and the old hotkey survive a failure, and
replacement errors block further saves until the file is inspected/reloaded.
Detected external edits are rejected before preparation and replacement; this is
a best-effort conflict check, not a cross-process compare-and-swap.

Corrupt, oversized, unsupported-version or unknown-field files use defaults for
this session and are preserved. Saving remains refused to avoid destroying newer
or unrelated data. To recover, quit MyCast, back up/move the original settings file
yourself, then restart and save a fresh configuration. Do not remove unrelated
files. Uninstall by quitting and deleting the extracted app; remove its own
settings file separately if desired. No startup entry is installed.

Synthetic previews: [100%](previews/settings-100.png),
[150%](previews/settings-150.png), [light](previews/launcher-light-100.png),
[high contrast](previews/launcher-high-contrast-100.png).
