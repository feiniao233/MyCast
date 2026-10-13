# Development preview package

Use official Go 1.27.2 / MyGo v0.4.6. Before packaging, verify dependencies, run
tests and vet, commit the reviewed change and build from that clean commit with
the README Windows GUI command. The executable's VCS metadata must match HEAD,
`vcs.modified=false`, Windows amd64 and CGO disabled. The package helper rejects
a dirty checkout, mismatched binary or wrong toolchain.
BUILDINFO records a relative executable name, never the packager's absolute
checkout/account path. The packaging helper rejects a checkout-path leak.

PowerShell 7:

```powershell
./scripts/package-windows.ps1 -GoExe 'C:\path\to\official-go1.27.2\bin\go.exe' -Version 0.3.0-alpha.1
```

The output is a version/commit-named ZIP and adjacent SHA256 text under `dist`.
The outer package contains MyCast.exe, README, project and third-party licenses,
documentation, BUILDINFO, MANIFEST and SHA256SUMS. A separate corresponding-source
ZIP inside it contains the exact tracked commit plus vendored module sources and
their notices. It can be built without fetching dependency source using
`go build -mod=vendor` with the required installed Go toolchain.

The manifest records commit/tree, toolchain/target, binary/file checksums and
`development-preview` / `nativeDesktopAcceptance=not-tested`. It does not declare
a public release. Verify every package file against SHA256SUMS after extraction;
the adjacent checksum identifies the outer ZIP itself.

CI uses `-LayoutOnly` and uploads the flat package directory: the downloaded
Actions artifact directly contains the executable and notices, not another
compressed executable distribution. The corresponding-source ZIP is intentional.
No tag, Release, repository visibility or main branch is changed by the helper.

Windows 10/11 x64 are the intended alpha targets. The current build is unsigned.
Remaining native checks are accepted limitations for this first alpha; preserve
the unverified limits in ACCEPTANCE.md.
Original-icon/experience review, exact-commit CI and owner publication remain.
To prepare the owner's reviewed alpha candidate, use `-Version 0.1.0-alpha`;
the helper still does not publish or declare native acceptance. It also verifies
the EXE's group and eight icon PNGs against the canonical ICO before packaging.
Extract to an ordinary user-writable
directory and run without elevation. Quit before replacing/deleting the app.
Ordinary data lives in `%APPDATA%\MyCast\settings.json`; remove only that own file
separately if desired. Clipboard history has no disk file. Startup is default off;
if explicitly enabled, disable it from the registered executable before moving
or uninstalling, removing only the owned MyCast.Launcher value.

For the current development candidate use `-Version 0.3.0-alpha.1`. Its incremental
acceptance guide is [INCREMENTAL_ACCEPTANCE_0.3.md](INCREMENTAL_ACCEPTANCE_0.3.md).
The optional display snapshot also lives at `%APPDATA%\MyCast\startup-snapshot.json`;
disable/clear it in Settings before optional own-data cleanup or a downgrade.
The published 0.1 artifact and separate 0.2 candidate remain unchanged; a new candidate ZIP is not a tag,
main merge or Release publication. New Windows interactions remain unverified
until the user records their results on the exact candidate.
