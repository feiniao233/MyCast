# Local arithmetic and explicit quicklinks

Open **菜单 → 小工具**. The calculator accepts decimal/scientific numbers,
unary signs, `+ - * /` and parentheses. Common multiplication/division/minus
glyphs are normalized. Results use finite float64 arithmetic and up to twelve
significant digits. Expressions stay in memory and are discarded when the sheet
closes. Enter edits/submits the field without copying; **复制结果** is explicit.

The parser accepts at most 512 input bytes, 128 tokens, recursion depth 32 and
64 bytes per numeric token. Division by zero, nonfinite values, identifiers,
function calls and trailing unsupported syntax are rejected. It has no code,
shell, external process or network evaluation path.

**编辑链接** uses one `name | https://address` row per entry. Up to 32 links are
saved as ordinary preferences, with names of 1–80 characters and raw/canonical
URLs of at most 2048 bytes. Names exclude the row separator and controls. URLs
must be absolute HTTP(S), with a host, a valid port if supplied, no embedded
credentials, control data or raw unsafe URI characters. Use percent encoding
for supported special characters. File/script/custom protocols are rejected.

Save validates the entire draft before changing settings; Cancel/Escape keeps
the prior links. Saving or displaying a link never requests its URL. Clicking
an entry validates it again and delegates it to the Windows default browser;
the launcher does not fetch pages or claim their reachability. Links are saved
in the same bounded ordinary settings file, independently of hotkey availability.
No default links or online services are added.

Tests use synthetic `example.com` addresses and mock copy/open adapters. Actual
browser launch, clipboard writes and native IME/focus remain untested. Synthetic
previews: [100%](previews/utilities-100.png), [150%](previews/utilities-150.png),
[editor at 100%](previews/quicklinks-editor-100.png),
[editor at 150%](previews/quicklinks-editor-150.png).
