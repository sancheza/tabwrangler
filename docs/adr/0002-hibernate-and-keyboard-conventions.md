# ADR 0002: Hibernate behavior and keyboard conventions

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-11-24 |
| **Scope** | `tabwrangler` repository; refines repo ADR 0001 section 2.5 (action semantics) |

## 1. Context

- A browser cannot discard the tab showing in its window. Hibernating such a tab was skipped, so the key appeared to do nothing for the tab a user most often wants to unload.
- Two keys (`h` and `d`) did the same thing, and four pairs of lower- and upper-case keys (`a`/`A`, `g`/`G`, `s`/`S`, `u`/`U`) ran different commands. Help text listed some commands with two keys and others with one, and a capital letter typed with Caps Lock on ran a different command.

## 2. Decision

### 2.1 Hibernate

- The key is `h`. It replaces `d`, which ADR 0001 section 2.5 listed as the discard key.
- For a tab showing in its window, the extension switches that window to the most recently used loaded tab, then discards the original tab.
- The extension never switches to a tab that is already discarded or is itself being discarded, so a switch never reloads a tab. When the window has no other loaded tab, the tab is skipped.
- The summary message names why tabs were skipped: already hibernated, not supported (Safari), or the only loaded tab in its window.

### 2.2 Keys

- Every command has one lowercase key. Menus, help, and documentation show lowercase only.
- A capital letter runs the same command as its lowercase key.
- A key that is retired shows a message naming the current key and does nothing else. The retired keys are `d`, `D`, `A`, `G`, `S`, and `U`.

| Command | Key |
|---|---|
| Select all visible tabs, or clear the selection when all are selected | `a` |
| Jump to top, jump to bottom | `g` or `Home`, `End` |
| Step the sort, flip the sort direction | `s`, `o` |
| Undo, show or hide the full URL column | `u`, `v` |

## 3. Consequences

- Hibernating changes which tab a browser window shows. Hibernating several selected tabs can switch several windows.
- A user of an earlier version meets a one-line hint instead of the old command for each retired key.
- A new command takes an unused lowercase key; it does not use a capital letter for a related command.

## 4. Alternatives considered

| Alternative | Why not |
|---|---|
| Keep `h` as is and add a second key that also hibernates the tab on screen | Two keys for one idea; the tab on screen is the case users want most |
| Leave the tab on screen alone and report it as skipped | The key appears to do nothing in the most common case |
| Keep capital letters as separate commands | A capital letter with Caps Lock on runs a different command, and help lists some commands with two keys and others with one |
| Remove retired keys with no hint | A stale key would fail silently |
