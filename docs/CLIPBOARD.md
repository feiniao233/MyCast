# Session-only text history

Recording starts disabled. **菜单 → 开启文本历史** establishes a fresh sequence
baseline without importing the existing clipboard. Only future bounded Unicode
text is captured; rich text/images and OS clipboard history are not imported.
Pausing stops collection, while existing entries remain subject to their memory
retention. Ordinary settings, favorites, recents and aliases never contain the
recording switch, retention policy or clipboard history.

The in-memory history holds at most 50 exact, deduplicated text entries. Each is
accepted whole or rejected at 64 KiB UTF-8; the native reader also bounds UTF-16
clipboard input. Whitespace, line endings and case are preserved. Collecting the
same text again moves it to the newest position and refreshes its expiry time.

Select a text entry and use **操作 → 预览全文** for a scrollable, selectable, read-only
preview. **复制全文** explicitly writes its original complete text to the system
clipboard; Enter on opening the sheet does not copy or paste. Close/Escape restores
search focus. **删除此条** removes one exact entry; **清空内存历史** clears all entries
and releases an open preview. New entries preserve the highlighted text identity.
If that text disappears, the same Enter does not copy a replacement automatically.
An open actions menu is bound to the identity selected when it opened. If that
entry is deleted or expires, the menu closes and its pending input is consumed;
it cannot copy, paste, preview or delete a replacement. Actions remain blocked
until an arrow/query change or an explicit selection confirms a new target.

**菜单 → 保留** cycles among until exit, ten minutes and one hour. This affects only
the current session and applies to existing entries too. Shortening the policy
can remove older entries immediately. The background tick checks expiry about
every 600 ms even while recording is paused; opening/reading history also expires
entries. Expired previews close and release their reference. Restart returns to
recording off and until-exit retention with an empty history.

Copying/pasting changes the shared system clipboard and does not restore its old
contents. The existing verified-target/focus/modifier/sequence checks still gate
return-to-window paste; tray invocation remains copy-only. A successful native
message does not prove text insertion by every target application. Clear/expiry
releases logical references; Go/OS/UI memory copies are not guaranteed securely
erased. Clipboard persistence is not implemented for this alpha.

Tests and shared images use dedicated synthetic text and mocked writes only.
Actual Windows opt-in, copy/paste, Pinyin and focus interactions remain untested
until the manual pass. Synthetic previews: [100%](previews/text-preview-100.png),
[150%](previews/text-preview-150.png), [end at 100%](previews/text-preview-end-100.png),
[end at 150%](previews/text-preview-end-150.png).
