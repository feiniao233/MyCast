# Windows UI feedback repair

Review branch: `fix/windows-ui-feedback`, based on
`9017fade4a975a7addbfae3e05a0975eb60f4dc3`.

The reported dropdown defect concerns its display, not incorrect selection.
MyGo v0.4.6 fills a Select popup with Theme.Background, and highlighted text uses
AccentText over Accent. Making Background transparent and replacing Accent with
white while retaining white AccentText caused the observed class of corruption.
Normal opaque theme surfaces and the upstream semantic accent pair are retained;
only Context.Root's background is transparent. The existing Select component's
placement, modal hit testing and keyboard behavior are preserved.

The saved dark/light/system preference now reaches MyGo.Theme.SetSource before
App.Run, and after successful settings saves. This affects the app's own native
window, not Windows preferences. Draft edits, cancel and failed saves do not
change it. A Windows adapter requests DWMSBT_TRANSIENTWINDOW (Desktop Acrylic),
checks the HRESULT/readback, and requests DWMWCP_ROUND for that own HWND. The
client-area scrim fills the rectangle; it no longer cuts a separate 26-DIP shape
out of the renderer. Windows owns the outer clipping. No SetWindowRgn, desktop
capture, custom shadow windows or system settings changes are used.

The menu/Return symbols are SVG assets with centered view boxes. Footer text has
one tokenized 22-DIP line height at 14-DIP body size, with regression checks on
both line boxes and painted CJK ink to detect baseline drift/cropping. This keeps
font fallback out of the icon geometry and leaves space for Chinese text metrics.

These are **synthetic headless** images, not the user's private screenshots or
proof of a working desktop effect. Application names and data are dedicated tests.

| Theme | 100% footer | 150% footer | 100% expanded select | 150% expanded select |
| --- | --- | --- | --- | --- |
| Dark | [image](previews/feedback-footer-dark-true-100.png) | [image](previews/feedback-footer-dark-true-150.png) | [image](previews/feedback-dropdown-dark-true-100.png) | [image](previews/feedback-dropdown-dark-true-150.png) |
| Light | [image](previews/feedback-footer-dark-false-100.png) | [image](previews/feedback-footer-dark-false-150.png) | [image](previews/feedback-dropdown-dark-false-100.png) | [image](previews/feedback-dropdown-dark-false-150.png) |

The two user images could not be downloaded locally: this environment exposes
Library upload actions but no supported materialization/download entry point.
They have not been viewed locally, added to the repository, or shared. Source
analysis used the fixed MyGo release and the reported defects. The Tinycast UI
reference (`docs/ui.md` at `9aea65b8d86b164dfb6e2375090fba625112540f`) was read;
its macOS 26-DIP clipping is deliberately replaced by Windows-native geometry.

Limits: successful DWM calls/readback prove the requested attributes were accepted,
not the visible effect. OS policies, high contrast, power state and remote sessions
can alter material presentation; the corner hint is not a guarantee. Windows 10
uses the opaque fallback and may keep square corners. Actual Acrylic, HWND corners,
Microsoft Pinyin and pointer appearance require another real desktop/user check.

References: [Microsoft backdrop values](https://learn.microsoft.com/en-us/windows/win32/api/dwmapi/ne-dwmapi-dwm_systembackdrop_type),
[native corner guidance](https://learn.microsoft.com/en-us/windows/apps/desktop/modernize/ui/apply-rounded-corners).
