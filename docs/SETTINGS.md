# Ordinary settings

Open **菜单 → 设置**. Save explicitly; Cancel or Escape discards the draft.
IME candidate Escape/Enter is handled by the editor before launcher actions.

The version-1 JSON file is `%APPDATA%\MyCast\settings.json`, created only when
settings are saved. It contains the hotkey, appearance and application roots.
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
  scan; stale scan completions cannot replace the new result set.

Saving validates bounded UTF-8 input, writes and synchronizes a temporary file in
the same directory, then replaces the destination. Failures preserve the current
settings. Detected external edits are rejected before preparation and replacement;
this is a best-effort conflict check, not a cross-process compare-and-swap.

Corrupt, oversized, unsupported-version or unknown-field files use defaults for
this session and are preserved. Saving remains refused to avoid destroying newer
or unrelated data. To recover, quit MyCast, back up/move the original settings file
yourself, then restart and save a fresh configuration. Do not remove unrelated
files. Uninstall by quitting and deleting the extracted app; remove its own
settings file separately if desired. No startup entry is installed.

Synthetic previews: [100%](previews/settings-100.png),
[150%](previews/settings-150.png), [light](previews/launcher-light-100.png),
[high contrast](previews/launcher-high-contrast-100.png).
