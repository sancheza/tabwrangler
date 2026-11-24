# ADR 0001: Cross-browser TUI architecture & multi-bridge telemetry

| | |
|---|---|
| **Status** | Accepted |
| **Date** | 2026-11-23 |
| **Scope** | `tabwrangler` repository |

## 1. Context

Modern users operate multiple web browsers simultaneously (Google Chrome, Mozilla Firefox, Apple Safari). Each browser isolates tabs in multi-process architectures, consumes significant memory and CPU, and lacks a centralized interface for monitoring and controlling tab lifecycles. Existing tab managers are either browser-specific extensions or OS task managers that lack tab-level semantics (title, URL, favicon, domain, discard state).

Building a centralized cross-browser Terminal User Interface (TUI) requires:
1. Low latency, non-blocking asynchronous state management.
2. A unified data model representing tab metadata and honest metric attribution across heterogeneous browser architectures.
3. Communication bridges accommodating both extension-capable browsers (Chrome, Firefox) and native platform scriptable browsers (Safari).
4. Alignment with common engineering standards: a standard repository layout (`cmd/`, `core/`, `tests/`, `docs/`), code placement by scope (shared code in `core/`, entry points in `cmd/`), standard command-line conventions (`-h/--help`, `-v/--version`), and standard terminal status colors (green info, amber warning, red error).

## 2. Decision

### 2.1 Technology Stack & TUI Engine

`tabwrangler` is implemented in Go (`1.24+`) utilizing the Charmbracelet v2 ecosystem:
- `charm.land/bubbletea/v2`: Unidirectional Elm-architecture state loop, non-blocking commands (`tea.Cmd`), and keyboard navigation.
- `charm.land/lipgloss/v2`: Declarative styling, adaptive terminal palettes, status badges, and table layout.
- `charm.land/bubbles/v2`: Virtualized viewport and textinput components for live filtering (`/`).

### 2.2 Repository Layout

The repository separates entry points (`cmd/`), shared code (`core/`), tests (`tests/`), and documentation (`docs/`):
- `cmd/tabwrangler/`: Main entry point compiling to the binary executable.
- `core/types`: Unified tab, metrics, and filter data models.
- `core/collector`: Aggregator engine and browser adapter interfaces.
- `core/browser/safari`: macOS AppleScript/JXA adapter.
- `core/bridge`: Native Messaging host and Unix domain socket / Named Pipe IPC server.
- `core/telemetry`: OS process inspection and PID correlation (`gopsutil`).
- `core/tui`: Bubble Tea v2 state model, views, keymaps, and styles.
- `core/setup`: Native Messaging manifest installer and OS permission verifier.
- `extension/`: Manifest V3 WebExtension companion for Chrome and Firefox.
- `tests/`: Automated test suite mirroring packages.
- `docs/adr/`: Repository ADRs.

### 2.3 Browser Bridge & IPC Architecture

TabWrangler uses a hybrid bridge architecture:
1. **Chrome & Firefox**:
   - Manifest V3 companion WebExtension connects via standard Native Messaging (`stdin`/`stdout` 4-byte length-prefixed JSON frames) to `tabwrangler bridge`.
   - The bridge host forwards requests to and from the TUI via a local user-scoped Unix domain socket (`/tmp/tabwrangler-$UID.sock`, mode `0600`) on macOS/Linux or a loopback TCP port (`127.0.0.1:42823`) on Windows. Each TUI run generates a random token, stored in an owner-only file, that the bridge host must present when it registers.
2. **Safari (macOS)**:
   - Direct execution of AppleScript / JavaScript for Automation (`osascript`) querying Safari's scripting dictionary for windows, tabs, focus activation, and tab closure.

### 2.4 Telemetry & Honest Metric Attribution

Multi-process browsers may group tabs under shared renderer processes or isolate tabs into separate processes with multiple subframe processes. Tab metrics explicitly record an attribution scope:
- `tab`: 1:1 process-to-tab mapping where memory/CPU represents only this tab.
- `shared_process`: Process shared across multiple tabs; figures are displayed prefixed with `~` in the UI to prevent false precision.
- `unavailable`: Operating system or browser boundary precludes per-tab attribution (e.g., Safari tabs without web inspector hooks); displayed as `—`.

### 2.5 Action Semantics & Safety Guards

- **Focus (`Enter`/`f`)**: Focuses active window and tab via extension or AppleScript, raising the host OS window.
- **Discard (`d`)**: Memory reclamation via `tabs.discard()`. Unsupported targets (Safari) are skipped with an explicit toast notice.
- **Close (`c`)**: Immediate close for `<= 3` tabs; inline interactive confirmation prompt (`Close N selected tabs? [y/N]`) when closing `> 3` tabs.
- **Refresh (`r`)**: Concurrent, non-blocking on-demand scan with a 3-second per-adapter timeout.

## 3. Consequences

- Native Messaging requires a one-time registration command (`tabwrangler setup`) to write host manifests to browser configuration paths.
- Safari does not expose a native discard API via AppleScript; discard operations safely skip Safari tabs with clear user feedback.
- Memory and CPU attribution remains transparent and never misleading.

## 4. Alternatives Considered

| Alternative | Why Not |
|---|---|
| Chrome DevTools Protocol (CDP) debugging port | Requires launching browsers with remote-debugging flags, exposing security risks and preventing connection to already-running standard user sessions. |
| Background daemon polling browser state every second | Consumes CPU/battery and goes against the on-demand fast design principle. |
| SQLite history database scraping | Provides stale history rather than live open tab states, lacks process correlation, and cannot perform mutations (focus, discard, close). |
