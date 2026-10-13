# Alpha hardening evidence

The intended targets are Windows 10/11 x64. Automated/harness evidence and actual
desktop evidence are reported separately in ACCEPTANCE.md.

## Source and notices

An official checksum-verified Gitleaks v8.30.1 binary was used locally with its
official tagged default rules, all refs/full history/reflog, no ignore file or
inline allow suppression, full redaction and no upload. At the utilities checkpoint
`b11280cb7081cbb8497203a931c9e41d61fc31a7`, the repository was not shallow;
14 commits were scanned and the exit code was 0 with zero findings. A pattern
scanner is evidence of this scan, not proof that every possible secret is absent.
The final packaging commit is rescanned before delivery.
The first packaging checkpoint `af2a6d2056523cf8c32cc2228847dce9f9a7812a`
adds the fifteenth commit; its complete-history scan also returned zero findings.

The preserved license text for MyGo v0.4.6, purego v0.11.1, typesetting v0.3.5,
go-pinyin v0.21.0 and x/image v0.46.0 is compared against the exact verified module
LICENSE files, allowing only line-ending and final-whitespace differences.
Vendored corresponding source and all upstream notices are included in the source
archive; project license/notices are in the binary package. Publication still
requires the release owner's source/notice review and the release checklist.

## Recovery and privacy

Settings are bounded and validated; unknown/corrupt/newer files are retained and
defaulted with saving refused until recovery. External edits are checked before
preparation/replacement. Windows moves have no copy/delete/reboot fallback and
do not claim filesystem-independent atomicity or crash durability. Explicit hotkey
changes register first and roll back on save failure. Ordinary metadata saving
does not depend on startup hotkey success. Indexing rejects network/unknown volumes
and reparse ancestors before access, with one cancelable, bounded worker.

Clipboard recording starts off, imports no existing value and retains only
future opt-in bounded text in memory. Session retention, deletion, target identity,
focus/modifier/sequence protections and settings separation have regression tests.
Tests, previews and smoke runs use dedicated fixtures; shared files contain no
user app inventory or clipboard contents. No automatic startup, accounts, API keys,
telemetry or online evaluation service is added.

## Package and remaining gate

PACKAGING.md describes clean-commit metadata checks, checksums and vendored source.
The initial package was extracted and all 46 checksum-listed files verified.
Its separate source archive includes vendor/modules.txt and rebuilt successfully
with GOPROXY/GOSUMDB disabled and `-mod=vendor`, using official Go 1.27.2.
The manifest explicitly labels the output as a development preview and native
desktop acceptance as not tested. The build is unsigned. Quit and delete the
extracted app to uninstall; remove only its own ordinary settings file separately
if desired. Clipboard history has no data file to remove.

Native desktop interaction checks are not proven by automated or resource/ABI
checks. Remaining native limits are accepted for this first alpha. Exact-commit
CI, privacy, license/package checks and owner review/publication remain required.
