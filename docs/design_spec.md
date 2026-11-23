# TabWrangler — Cross-Browser Tab Inspector & Control TUI
**Technical & Product Specification (v1.0)**

---

## 1. Product Summary
**TabWrangler** (`tabwrangler`) is a fast, keyboard-driven Terminal User Interface (TUI) for **macOS** and **Windows** that gives users centralized visibility and control over open tabs across **Google Chrome**, **Mozilla Firefox**, and **Apple Safari** (macOS).

Users can inspect every open tab's metadata and resource footprint on demand, filter/sort across browsers, bring any tab to the foreground, discard/unload idle tabs to reclaim memory, or close tabs individually or in bulk.

### Core Design Principles
* **On-Demand & Fast:** Zero background polling loops; fetches fresh snapshots on startup, on explicit refresh (`r`), and immediately after tab mutations (`close` / `discard`).
* **Unified Control, Instant Scoping:** Shows all active browsers in one unified table by default, with single-key switching to isolate any specific browser.
* **Honest Telemetry:** Displays per-tab CPU and RAM alongside explicit measurement scope indicators when a browser engine or OS boundary restricts exact 1:1 tab attribution.

---

## 2. Technology Stack
* **Language:** Go (`1.24+`)
* **TUI Framework:** Charmbracelet v2 stack:
  * `charm.land/bubbletea/v2` (Elm-architecture state loop & keyboard event handling)
  * `charm.land/lipgloss/v2` (declarative layout, badges, borders, adaptive terminal colors)
  * `charm.land/bubbles/v2` (table/viewport, text input filter, help overlay)
* **Process & OS Telemetry:** `github.com/shirou/gopsutil/v4` + OS-specific build-tagged adapters (`*_darwin.go`, `*_windows.go`)
* **Browser Control Bridges:**
  * Lightweight Manifest V3 WebExtension (Chrome & Firefox) communicating over standard Native Messaging (`stdin`/`stdout` length-prefixed JSON) + local IPC socket/pipe.
  * AppleScript / JXA (`osascript`) adapter for Safari on macOS.

---

## 3. Supported Platforms & Browser Matrix

| Browser | macOS | Windows | Metadata (`Title`, `URL`, `Domain`, `Favicon`) | `Last Accessed` | Per-Tab CPU & RAM | Actions (`Focus`, `Discard`, `Close`) |
| :--- | :---: | :---: | :--- | :--- | :--- | :--- |
| **Chrome** | Yes | Yes | Full (via Companion Extension) | Exact (`tab.lastAccessed`) | Renderer PID + Extension correlation | `Focus`, `Discard`, `Close` |
| **Firefox** | Yes | Yes | Full (via Companion Extension) | Exact (`tab.lastAccessed`) | Content-process PID correlation | `Focus`, `Discard`, `Close` |
| **Safari** | Yes | No | `Title`, `URL`, `Domain` (via AppleScript/JXA) | Unavailable (`—`) | Process-level / unavailable per tab | `Focus`, `Close` (`Discard` unsupported) |

> **Architectural Note:** Modern browsers use multi-process/site-isolation models where multiple tabs may share a renderer/content process or spawn subframes. Collectors must attach a `metric_scope` (`tab`, `shared_process`, or `unavailable`) to CPU/RAM values so the UI communicates exactness transparently.

---

## 4. User Experience & Interface Specification

### 4.1 Screen Layout
1. **Top Bar (Browser Scope & Summary):**
   * Interactive filter tabs with tab counts and aggregate RAM:
     `[0] All (42 · 5.4 GB)   [1] Chrome (20 · 3.1 GB)   [2] Firefox (16 · 1.8 GB)   [3] Safari (6 · 0.5 GB)`
   * Browsers unavailable on the host OS are omitted; browsers without an active bridge show a status indicator (e.g., `[2] Firefox (bridge disconnected)`).
2. **Filter & Status Strip:**
   * Active search query (`/filter`), active sort column + direction (`RAM ▼`), and toggle indicators (`[Special Tabs: Hidden]`).
3. **Main Tab Table (Virtualized Viewport):**
   * **Columns:**
     * `Sel`: `[x]` / `[ ]` multi-select checkbox
     * `Brw`: Compact badge (`CH`, `FF`, `SF`)
     * `Domain / Favicon`: Parsed hostname (e.g., `github.com`), prefixed by domain glyph/icon where terminal supports it
     * `Title`: Page `<title>` (flex-width, truncated with `…`)
     * `URL`: Full URL (shown in detail/preview pane or toggled inline with `u`)
     * `Last Used`: Relative elapsed time since last user focus (`now`, `4m`, `2h`, `5d`, or `—`)
     * `CPU`: Percentage (`12.4%`, or `~12.4%` if shared process, or `—`)
     * `RAM`: Formatted memory (`342 MB`, `1.2 GB`, or `~342 MB` if shared process, or `—`)
     * `State`: `Active`, `Discarded`, or `Pinned`
4. **Detail Preview Footer (1–2 lines):**
   * Displays full untruncated `Title`, full `URL`, window/profile label, and exact metric attribution note for the currently highlighted row.
5. **Bottom Keybind Bar:**
   * Context-sensitive shortcut hints + toast feedback for last executed action.

### 4.2 Browser Selection UX
* **Default Launch:** Opens in `[0] All` mode with an immediate on-demand fetch across all detected browsers.
* **Single-Key Switching:**
  * `0`: Show all browsers
  * `1` / `2` / `3`: Isolate specific browser (`Chrome`, `Firefox`, `Safari`)
  * `[` / `]`: Cycle previous/next browser scope
* **CLI Override:** `tabwrangler --browser chrome,firefox` launches pre-filtered to the specified browser(s).

### 4.3 Special & Internal Tabs Filtering
* **Default Behavior:** Internal/special tabs (`chrome://*`, `about:*`, `edge://*`, `safari-resource://*`, `file://*`, extension popups, and blank new-tab pages) are **filtered out by default**.
* **Toggle:** Pressing `t` toggles visibility of special/internal tabs on or off, updating the header count and status badge (`[Special Tabs: Shown]`).

### 4.4 Keyboard Controls

| Key | Action |
| :--- | :--- |
| `j` / `k` or `↑` / `↓` | Move cursor down / up |
| `g` / `G` | Jump to top / bottom of list |
| `Space` | Toggle selection on highlighted tab |
| `a` / `A` | Select all visible tabs / Clear all selections |
| `0`–`3` or `[` / `]` | Switch browser filter (`All`, `Chrome`, `Firefox`, `Safari`) |
| `/` | Enter live fuzzy/substring filter (matches `Title`, `URL`, `Domain`) |
| `Esc` | Clear active search filter or cancel modal/confirmation |
| `s` / `S` | Cycle sort column (`RAM`, `CPU`, `Last Used`, `Browser`, `Title`) / Reverse sort order |
| `t` | Toggle special/internal tabs (`about:*`, `chrome://*`, `file://*`, blank tabs) |
| `u` | Toggle full URL column visibility |
| `Enter` or `f` | **Focus / Foreground:** Bring browser application, window, and specific tab to the foreground |
| `d` | **Discard / Unload:** Unload selected tab(s) (or highlighted tab) from memory while keeping tab open |
| `c` | **Close:** Close selected tab(s) (or highlighted tab) |
| `r` | **Refresh:** Trigger on-demand re-scan of all open tabs and resource stats |
| `i` | Show bridge/permission diagnostics modal |
| `?` | Toggle full help overlay |
| `q` / `Ctrl+C` | Quit `tabwrangler` |

---

## 5. Core Workflows & Action Semantics

### 5.1 On-Demand Refresh (`r`)
* Runs all enabled browser adapters concurrently with a strict per-adapter timeout (default `3s`).
* Displays a non-blocking spinner in the status strip while collecting; never freezes keyboard navigation.
* Preserves active search filter, sort order, and browser scope across refreshes.

### 5.2 Foreground / Focus (`Enter` or `f`)
* Operates on the **currently highlighted tab**.
* **Chrome / Firefox:** Sends `focus_tab` command via Native Messaging (`tabs.update(tabId, { active: true })` + `windows.update(windowId, { focused: true })`) and raises the browser app via OS API (`NSRunningApplication` on macOS / `SetForegroundWindow` on Windows).
* **Safari (macOS):** Executes AppleScript/JXA to set `current tab` of target window, set `index` of window to `1`, and `activate` Safari.

### 5.3 Discard / Unload (`d`)
* Operates on **all selected tabs** (or highlighted tab if selection is empty).
* Calls `tabs.discard(tabId)` via the Chrome/Firefox companion bridge.
* If selection includes Safari tabs or active/un-discardable tabs, skips unsupported items and displays a clear summary toast (e.g., `Discarded 4 tabs (2 Safari tabs skipped — discard unsupported)`).
* Triggers an automatic refresh after completion.

### 5.4 Close (`c`)
* Operates on **all selected tabs** (or highlighted tab if selection is empty).
* **Safety Guard:** If closing **more than 3 tabs** at once, displays an inline confirmation prompt: `Close 7 selected tabs? [y/N]`. Single-tab or `<= 3` tab closes execute immediately.
* Reports partial failures explicitly (e.g., `Closed 6/7 tabs — 1 tab already closed`).
* Triggers an automatic refresh after completion.

---

## 6. Data Schema (Internal & Bridge IPC)

```json
{
  "id": "chrome:104:882",
  "browser": "chrome",
  "profile": "Default",
  "window_id": "104",
  "tab_id": "882",
  "title": "Bubble Tea v2 Release Notes",
  "url": "[https://github.com/charmbracelet/bubbletea/releases](https://github.com/charmbracelet/bubbletea/releases)",
  "domain": "github.com",
  "favicon_url": "[https://github.githubassets.com/favicons/favicon.svg](https://github.githubassets.com/favicons/favicon.svg)",
  "last_accessed_unix_ms": 1795449120000,
  "is_active": false,
  "is_pinned": false,
  "is_discarded": false,
  "is_special": false,
  "metrics": {
    "cpu_percent": 1.4,
    "ram_bytes": 184549376,
    "scope": "tab"
  },
  "capabilities": {
    "can_focus": true,
    "can_discard": true,
    "can_close": true
  }
}
```

---

## 7. Security, Permissions & Packaging

1. **Local-Only IPC:**
   * Companion extensions communicate exclusively via browser-managed Native Messaging manifests (`com.tabwrangler.bridge.json`) and a user-scoped Unix domain socket (`/tmp/tabwrangler-$UID.sock` with `0600` permissions on macOS) or user-ACL Named Pipe (`\\.\pipe\tabwrangler-<user>` on Windows). No open TCP ports.
2. **macOS Permissions:**
   * Provide a `tabwrangler setup` subcommand that installs Native Messaging host manifests for Chrome/Firefox and verifies/prompts for macOS Automation (`System Events` / `Safari`) permissions.
3. **Windows Packaging:**
   * `tabwrangler setup` registers Native Messaging host registry keys under `HKCU\Software\Google\Chrome\NativeMessagingHosts\` and `HKCU\Software\Mozilla\NativeMessagingHosts\` without requiring Administrator privileges.
4. **Privacy:**
   * No telemetry, external network requests, or persistent tab history logging to disk. Incognito/Private windows are excluded by default unless enabled in the extension settings.

---

## 8. Implementation Milestones
1. **Milestone 1 — Core TUI & Mock Provider:** Build `tabwrangler` Go binary with Bubble Tea v2 layout, top browser selector bar, sorting, `/` filtering, `t` special-tab toggle, multi-select, and mock collector data.
2. **Milestone 2 — Companion Extension & Native Bridge:** Implement shared MV3 WebExtension (Chrome + Firefox) + Go Native Messaging host supporting `list_tabs`, `focus_tab`, `discard_tabs`, and `close_tabs`.
3. **Milestone 3 — OS Telemetry & Process Correlation:** Implement `*_darwin.go` and `*_windows.go` process collectors (`gopsutil` + OS memory footprint / private working set APIs) and map renderer/content PIDs to tab entries.
4. **Milestone 4 — Safari Adapter & Setup CLI:** Add macOS AppleScript/JXA adapter for Safari (`list`, `focus`, `close`) and `tabwrangler setup` installer command.
