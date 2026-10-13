# Alpha release and candidate gate

`0.1.0-alpha` is published at `5512e940e4693aabb56e02ac2cd0517e5f47dd26`.
The current **0.3.0-alpha.1** is a development candidate on a feature branch;
the reviewed 0.2 candidate remains separately available.
The prior owner acceptance does not pass the new native interactions. Use
[incremental acceptance](INCREMENTAL_ACCEPTANCE_0.3.md) alongside the existing
gates; no new main merge, tag or Release has been authorized.

This checklist is a gate, not a claim that the alpha is ready.

User decision on 2026-10-13: existing actual-Windows testing is sufficient for
this first alpha after experience cleanup. Remaining IME/paste/tray/hotkey/DPI/
multi-monitor coverage is explicitly accepted as deferred; preserve its partial
or untested result. Final original-icon review and exact-commit automated/CI,
privacy, license and package checks still apply. The owner coordinates publication.

- [ ] All mandatory roadmap phases are implemented and reviewed.
- [ ] Windows dependency verification, tests, vet and release-style build pass
      on the exact clean release commit; CI agrees.
- [ ] Actual Windows hotkey, keyboard/IME, tray, focus, launch and clipboard
      checks in `ACCEPTANCE.md` are passed or have an explicitly accepted limit.
- [ ] Invalid settings, upgrade/migration and unavailable hotkeys have safe
      recovery paths; access through tray/window remains available.
- [ ] Privacy defaults are verified: no telemetry, no clipboard import, history
      persistence off, startup and display snapshot off unless explicitly saved,
      no permission/security changes. Snapshot rows cannot grant launch authority.
- [ ] Tests/previews/logs and packaged files contain no user data or credentials.
- [ ] All dependency licenses, AGPL obligations and notices are current.
- [ ] The package contains the executable, README, LICENSE and third-party
      notices/licenses; source and build instructions are available.
- [ ] SHA256 checksums identify each delivered binary/package and source commit.
- [ ] Known limitations, unsigned-binary status, supported Windows versions and
      clean uninstall/application-data handling are documented accurately.
- [ ] Public repository visibility, main merge, version tag and prerelease
      publication are coordinated and authorized by the release owner.

No network services/API credentials are necessary for this alpha. Clipboard
history persistence and expanded/compact window states may
remain unsupported; do not advertise an option or feature that has not shipped.
