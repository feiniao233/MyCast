# MyCast contributor instructions

MyCast is a Windows-first native Go launcher. Make small, reviewable changes on
development branches. Keep each phase buildable and update `docs/ROADMAP.md` and
`docs/ACCEPTANCE.md` with evidence and remaining blockers.

## Architecture

- `cmd/mycast`: platform entry points only.
- `internal/core`: pure search/index identity, history and other domain logic.
  Do not launch processes, read the clipboard, or call UI APIs here.
- `internal/platform`: narrow Windows adapters and non-Windows stubs. Validate
  inputs and fail closed; isolate native resource ownership and cleanup.
- `internal/launcher`: MyGo window, input, menus, state and background workers.
  Mutate UI state on its UI thread through `Window.Update`.
- `internal/settings`: bounded ordinary preferences and safe storage/shortcut
  transitions; clipboard text and recording state must never enter this schema.
- Settings storage should have a small validated schema, atomic writes, bounded
  input and injectable paths for tests. Do not mix it into rendering.

## Build and verification

Use official Go **1.27.2**, MyGo **v0.4.6**, and the locked `go.mod`/`go.sum`.
Do not silently change the machine's global toolchain or Go configuration.
Before submitting runtime changes, run:

```powershell
go mod verify
go test ./...
go vet ./...
$env:CGO_ENABLED = '0'
go build -trimpath -tags mygo_noinspector -ldflags '-H windowsgui -s -w' -o dist/MyCast.exe ./cmd/mycast
```

Add regression tests for meaningful behavior and native ABI/resource changes.
Use synthetic fixtures and temporary directories/windows. Pure documentation
changes need accurate links and artifact provenance, not another UI run.
Headless UI tests prove layout/input behavior in that harness; compilation,
cross-compilation and process survival do not prove desktop interactions.

## UI conventions

Read the relevant upstream Tinycast UI/palette specifications when changing its
inspired design. Reuse real MyGo APIs; do not invent supported effects.
Keep metrics and colors in `palette_tokens_windows.go`, search mounted, result
identities stable, selection order identical to visible order, and IME candidate
confirmation/cancel protected from launcher actions.
Prefer Windows-native material and readable lightweight controls. Optional glass
may sample only same-window content; do not build Apple-style refraction or
desktop-capture pipelines. Desktop material comes from Window Transparent/Vibrancy, checked by
`Context.Vibrancy`, with a readable solid fallback. Declare approximations and
unverified native corners/materials. Never sacrifice first/last-row visibility.
Menus must preserve search focus and consume click-away without launching a
covered result. Compare rendered 100%/150% DPI, empty and populated states.
Publish only clearly labeled synthetic preview images to `docs/previews`.

## Privacy and safety boundaries

- No telemetry, network services, account systems or API keys in the alpha scope.
- Clipboard recording starts disabled each session. Opt-in begins from a fresh
  sequence baseline and only captures future bounded text. Never import the
  current clipboard, OS clipboard history, or rich/image data silently.
- Clipboard content stays in memory; persistence remains off unless a separately
  implemented and clearly explained option is explicitly enabled by the user.
- Do not persist clipboard contents in settings, logs, screenshots or repository
  fixtures. Tests and shared previews use only dedicated synthetic text.
- User actions may save ordinary settings, favorites, recents and aliases in the
  application's own data directory; do not log or commit real app inventories.
- Index only explicitly configured application roots and supported entries;
  do not follow symbolic links or add document-content scanning implicitly.
- Keep shortcut target/arguments/directory and special launch semantics safe.
  Unproven equivalence must retain separate original shortcut identities.
- Preserve verified-target, focus, modifier and clipboard-sequence checks for
  paste. Do not inject global keys or elevate to bypass another app's rights.
- No silent startup registration, credentials, security/privacy policy changes,
  unknown DLLs, or assistant-side desktop-control workarounds.

## Release checks

Use `docs/RELEASE_CHECKLIST.md`. Report checks as passed, failed or not tested.
Unverified mandatory Windows keyboard/IME/tray/clipboard interactions remain
release blockers, even when CI is green. Record honest known limitations.
Check licenses/notices, clean commit provenance, reproducible build instructions,
packaging, checksums, settings migration and uninstall/data cleanup guidance.
Development work must not change repository visibility, merge main or publish a
Release; the release owner coordinates those final operations and authorization.
