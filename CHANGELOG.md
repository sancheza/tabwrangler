# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/).

## 1.0.18 - 2026-11-24

### Added

- The diagnostics screen (`i`) shows each browser extension's version and warns when it is older than TabWrangler or does not report a version, with the fix. The extension sends its version when it registers, so reload the extension after updating.

### Fixed

- The diagnostics screen shows the real bridge address on every platform (it showed the macOS/Linux socket path on Windows).

## 1.0.17 - 2026-11-24

### Added

- `tabwrangler --update` (`-u`) installs the latest release from GitHub. `--dry-run` checks without installing. Downloads are verified against the release's `SHA256SUMS` file; a release without it is refused. On Windows the update runs `setup`, which also updates the extension folder.
- Releases include a `SHA256SUMS` file (`make checksums`).

## 1.0.16 - 2026-11-24

### Fixed

- The browser's bridge host stays running when TabWrangler quits and registers again when TabWrangler starts. Relaunching within a few seconds of quitting no longer waits for the browser to restart the host (about 2 seconds on Windows). The host looks for TabWrangler every 200 ms.
- `tabwrangler setup` on Windows replaces the installed program even while a browser has it running, and reports a warning when it cannot, instead of skipping the copy silently.

### Documentation

- The Windows upgrade steps use `setup`, which replaces the installed program.
- README explains the harmless `background.scripts` warning in Chrome.

## 1.0.15 - 2026-11-24

### Changed

- The diagnostic log records the startup path: the bridge server, each browser registering, each refresh request and result (tab counts and timing), and each request to the browser extension. It never records tab titles or URLs.

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
