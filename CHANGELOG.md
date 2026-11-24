# Changelog

Format: [Keep a Changelog](https://keepachangelog.com/en/1.1.0/). Versions follow [Semantic Versioning](https://semver.org/).

## Unreleased

### Security

- The native messaging host authenticates to the TUI with a random per-run token stored in an owner-only file. Registrations without the token are rejected.
- The IPC socket is created with owner-only permissions from the start (umask `0177` around `listen`).
- The diagnostic log no longer records tab titles, URLs, or message contents. Log files are created with mode `0600`.

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
