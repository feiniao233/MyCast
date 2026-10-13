# 0.1.0 alpha release gate

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
      persistence off, no automatic startup or permission/security changes.
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
history persistence, startup integration and expanded/compact window states may
remain unsupported; do not advertise an option or feature that has not shipped.
