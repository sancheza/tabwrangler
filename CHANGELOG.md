# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/).

## 1.0.14 - 2026-11-24

### Fixed

- Windows: pressing Enter now raises the browser window that holds the tab. The window is chosen by the owning program (`chrome.exe`, `msedge.exe`, `firefox.exe`) and by tab title, not by window class, which other Chromium-based apps share. The focus calls run on one OS thread. Window selection is logged.
- A refresh that finishes late no longer replaces newer tab data with an older, empty result. Refreshes are logged with tab counts and timing.
- README: the link after "Build from Source" points to section 3.4.

## 1.0.13 - 2026-11-24

### Security

- The native messaging host authenticates to the TUI with a random per-run token stored in an owner-only file. Registrations without the token are rejected.
- The IPC socket is created with owner-only permissions from the start (umask `0177` around `listen`).
- The diagnostic log no longer records tab titles, URLs, or message contents. Log files are created with mode `0600`.

### License

- Changed from MIT to PolyForm Noncommercial 1.0.0. Commercial use requires a separate license.

### Documentation

- Added `SECURITY.md`, `CONTRIBUTING.md`, and this changelog.
- Documented the files TabWrangler writes to disk.

## 1.0.12

- Version shown in the TUI.
- "Discard" renamed to "Hibernate", with `h` and `d` shortcuts.

## 1.0.11

- Undo for close and discard actions.

## Earlier versions

See [GitHub Releases](https://github.com/sancheza/tabwrangler/releases).
