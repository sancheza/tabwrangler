# TabWrangler (`tabwrangler`)

TabWrangler is a fast, keyboard-driven Terminal User Interface (TUI) for macOS and Windows that provides centralized visibility and control over open tabs across Google Chrome, Mozilla Firefox, and Apple Safari.

## 1. Features

- **Unified Cross-Browser Viewport:** Inspect tabs across Chrome, Firefox, and Safari in a single virtualized table, with single-key switching to isolate individual browsers (`0`–`3`, `[` / `]`).
- **Resource Footprint & Honest Telemetry:** Measure tab CPU and RAM on demand, with visual indicators (`~`) distinguishing exact single-tab metrics from shared renderer processes.
- **Fast Lifecycle Actions:**
  - **Focus (`Enter` / `f`):** Bring any window and tab directly to the foreground.
  - **Discard (`d`):** Unload idle tabs from memory to reclaim RAM while keeping them open in the browser.
  - **Close (`c`):** Close single or bulk-selected tabs, guarded by confirmation for batches greater than 3 tabs.
- **Zero Background Polling:** Refreshes on startup, on-demand (`r`), and immediately following user actions.
- **Internal Tab Filtering:** Filters `chrome://*`, `about:*`, and blank tabs by default; toggled instantly with `t`.

## 2. Requirements

- **Go:** 1.24+ (to build from source)
- **Supported Operating Systems:** macOS (12+), Windows 10/11
- **Supported Browsers:**
  - Google Chrome (via companion extension)
  - Mozilla Firefox (via companion extension)
  - Apple Safari (macOS only, via native AppleScript automation)

## 3. Installation & Setup

### 3.1 Build from Source

```bash
git clone https://github.com/sancheza/tabwrangler.git
cd tabwrangler
go build -o tabwrangler ./cmd/tabwrangler
```

### 3.2 Register Browser Bridges

Run the setup command to register the Native Messaging host manifests for Chrome and Firefox, and verify macOS automation permissions:

```bash
./tabwrangler setup
```

Expected output:
```text
[OK] Chrome native messaging host manifest installed: ~/Library/Application Support/Google/Chrome/NativeMessagingHosts/com.tabwrangler.bridge.json
[OK] Firefox native messaging host manifest installed: ~/Library/Application Support/Mozilla/NativeMessagingHosts/com.tabwrangler.bridge.json
[OK] Safari AppleScript automation permissions verified
```

### 3.3 Install Companion Extension

1. In Chrome, navigate to `chrome://extensions`, enable **Developer mode**, click **Load unpacked**, and select the `extension/` directory.
2. In Firefox, navigate to `about:debugging#/runtime/this-firefox`, click **Load Temporary Add-on**, and select `extension/manifest.json`.

## 4. Usage

Launch TabWrangler in interactive TUI mode:

```bash
tabwrangler
```

Launch pre-filtered to specific browsers:

```bash
tabwrangler --browser chrome,firefox
```

Launch with mock data for testing or demonstrations:

```bash
tabwrangler --mock
```

Execute non-destructive dry runs:

```bash
tabwrangler --dry-run
```

Display version information:

```bash
tabwrangler --version
```

## 5. Keyboard Navigation

| Key | Action |
|:--- |:--- |
| `j` / `k` or `↑` / `↓` | Move cursor down / up |
| `g` / `G` | Jump to top / bottom of list |
| `Space` | Toggle selection on highlighted tab |
| `a` / `A` | Select all visible tabs / Clear all selections |
| `0`–`3` or `[` / `]` | Switch browser filter (`All`, `Chrome`, `Firefox`, `Safari`) |
| `/` | Enter live filter query (matches `Title`, `URL`, `Domain`) |
| `Esc` | Clear filter or cancel confirmation |
| `s` / `S` | Cycle sort column (`RAM`, `CPU`, `Last Used`, `Browser`, `Title`) / Reverse direction |
| `t` | Toggle special/internal tabs (`chrome://*`, `about:*`, `file://*`) |
| `u` | Toggle full URL column visibility |
| `Enter` or `f` | Focus highlighted tab and bring browser to foreground |
| `d` | Discard selected tabs (or highlighted tab) to free RAM |
| `c` | Close selected tabs (or highlighted tab) |
| `r` | Refresh tab snapshot on demand |
| `i` | Show bridge diagnostics modal |
| `?` | Toggle help overlay |
| `q` / `Ctrl+C` | Quit TabWrangler |

## 6. Architecture & Standards

TabWrangler conforms to repository and coding standards:
- Dev ADR 0001: Architecture Decisions
- Dev ADR 0002: Standard Repository Layout
- Dev ADR 0003: Code Placement by Scope
- Dev ADR 0005: Command-Line Conventions
- Repo ADR 0001: Cross-Browser TUI Architecture
