# TabWrangler (`tabwrangler`)

TabWrangler is a fast, keyboard-driven Terminal User Interface (TUI) for macOS, Windows, and Linux that provides centralized visibility, telemetry correlation, and lifecycle control over open tabs across Google Chrome, Mozilla Firefox, and Apple Safari.

> **Quick Start:** To install a pre-built binary and launch TabWrangler in under two minutes without compiling from source, jump down to [3. Quick Start & Installation](#3-quick-start--installation).

---

## 1. Features

- **Unified Cross-Browser Viewport:** Inspect tabs across Chrome, Firefox, and Safari in a single virtualized table, with single-key switching to isolate individual browsers (`0`–`3`, `[` / `]`).
- **Resource Footprint & Honest Telemetry:** Measure tab CPU and RAM on demand, with visual indicators (`~`) distinguishing exact single-tab metrics from shared renderer processes.
- **Fast Lifecycle Actions:**
  - **Focus (`Enter` / `f`):** Bring any window and tab directly to the foreground.
  - **Discard (`d`):** Unload idle tabs from memory to reclaim RAM while keeping them open in the browser.
  - **Close (`c`):** Close single or bulk-selected tabs, guarded by confirmation for batches greater than 3 tabs.
- **Zero Background Polling:** Refreshes on startup, on-demand (`r`), and immediately following user actions.
- **Internal Tab Filtering:** Filters `chrome://*`, `about:*`, and blank tabs by default; toggled instantly with `t`.

---

## 2. Requirements

### Supported Operating Systems
- **Windows:** Windows 10 (build 1809+) or Windows 11 (64-bit only).
- **macOS:** macOS 12 Monterey or later (Apple Silicon or Intel 64-bit).
- **Linux:** Modern Linux distributions with glibc (x86_64, arm64).

> **Compatibility Notice:**
> - **Unsupported:** Windows 7, Windows 8, Windows 8.1, and all 32-bit (x86) operating systems are not supported.
> - Attempting to launch TabWrangler on Windows 7 or 32-bit Windows triggers `not a valid win32 application` due to modern 64-bit Go runtime dependencies and Windows 10+ Virtual Terminal (ConPTY) requirements.

### Supported Browsers
- **Google Chrome / Microsoft Edge / Brave:** Supported via companion WebExtension.
- **Mozilla Firefox:** Supported via companion WebExtension (or session fallback).
- **Apple Safari:** Supported out of the box on macOS via native automation (zero extensions required).

### Development Toolchain (Optional)
- **Go:** Go 1.24+ (only required if building from source).

---

## 3. Quick Start & Installation

### 3.1 Option A: Download Pre-Compiled Binary

Download the archive for your operating system from the [Latest Release](https://github.com/sancheza/tabwrangler/releases/latest):

| Platform | Download | Package Contents |
|:--- |:--- |:--- |
| **Windows (x64)** | [tabwrangler-windows-amd64.zip](https://github.com/sancheza/tabwrangler/releases/latest/download/tabwrangler-windows-amd64.zip) | `tabwrangler.exe`, `extension/` directory |
| **macOS (Apple Silicon)** | [tabwrangler-darwin-arm64](https://github.com/sancheza/tabwrangler/releases/latest/download/tabwrangler-darwin-arm64) | Standalone binary |
| **macOS (Intel)** | [tabwrangler-darwin-amd64](https://github.com/sancheza/tabwrangler/releases/latest/download/tabwrangler-darwin-amd64) | Standalone binary |
| **Linux (x86_64)** | [tabwrangler-linux-amd64](https://github.com/sancheza/tabwrangler/releases/latest/download/tabwrangler-linux-amd64) | Standalone binary |
| **Linux (ARM64)** | [tabwrangler-linux-arm64](https://github.com/sancheza/tabwrangler/releases/latest/download/tabwrangler-linux-arm64) | Standalone binary |

Extract the ZIP archive (Windows) or mark the downloaded binary executable (macOS/Linux):

**Windows (Command Prompt or PowerShell):**
```cmd
tar -xf tabwrangler-windows-amd64.zip
```
*(Alternatively, run `Expand-Archive -Path tabwrangler-windows-amd64.zip -DestinationPath .` in PowerShell, or right-click the file in File Explorer and select **Extract All...**)*

**macOS:**
```bash
chmod +x tabwrangler-darwin-*
mv tabwrangler-darwin-* tabwrangler
```

**Linux:**
```bash
chmod +x tabwrangler-linux-*
mv tabwrangler-linux-* tabwrangler
```

After preparing the binary, proceed to [3.3 Register Native Messaging Host](#33-register-native-messaging-host).

### 3.2 Option B: Build from Source

Objective: Compile the application binary directly from the Git repository.

```bash
git clone https://github.com/sancheza/tabwrangler.git
cd tabwrangler
go build -o tabwrangler ./cmd/tabwrangler
```

Expected output: An executable binary `tabwrangler` (or `tabwrangler.exe` on Windows) generated in the current directory.

### 3.3 Register Native Messaging Host

Objective: Register TabWrangler with the host operating system and browsers so companion extensions can establish local IPC communication.

Run the `setup` subcommand:

**macOS / Linux:**
```bash
./tabwrangler setup
```

**Windows (Command Prompt or PowerShell):**
```cmd
.\tabwrangler.exe setup
```

Expected output:
```text
[INFO] Configuring Native Messaging host for TabWrangler...
[OK] Installed persistent binary: C:\Users\<Username>\AppData\Local\TabWrangler\tabwrangler.exe
[OK] Chrome native messaging manifest installed: C:\Users\<Username>\AppData\Local\TabWrangler\NativeMessagingHosts\com.tabwrangler.bridge.json
[OK] Firefox native messaging manifest installed: C:\Users\<Username>\AppData\Roaming\Mozilla\NativeMessagingHosts\com.tabwrangler.bridge.json
[INFO] Registering Native Messaging hosts in Windows Registry...
[OK] Registered Google\Chrome across 32-bit & 64-bit Windows Registry views
[OK] Registered Microsoft\Edge across 32-bit & 64-bit Windows Registry views
[OK] Registered Chromium across 32-bit & 64-bit Windows Registry views
[OK] Registered BraveSoftware\Brave-Browser across 32-bit & 64-bit Windows Registry views
[OK] Registered Mozilla across 32-bit & 64-bit Windows Registry views
[OK] Copied extension directory to Windows clipboard: C:\Users\<Username>\AppData\Local\TabWrangler\extension
[OK] Setup complete. Load the extension from the 'extension/' directory into your browsers.
```

The setup command places the binary into a persistent application directory (`%LOCALAPPDATA%\TabWrangler\` on Windows) and copies the path to the `extension/` folder directly to your clipboard.

### 3.4 Load Companion Extension into Browsers

Objective: Install the unpacked Manifest V3 companion extension in your browser.

#### Google Chrome / Microsoft Edge / Brave:
1. Open your browser and navigate to `chrome://extensions` (or `edge://extensions` / `brave://extensions`).
2. Toggle on **Developer mode** in the top right corner.
3. Click the **Load unpacked** button in the top left header.
4. Select the `extension/` folder in the file dialog (or paste the path already on your clipboard into the folder selection bar).
5. Confirm that **TabWrangler Companion** appears in your extension list with no error badges.

#### Mozilla Firefox:
1. Open Firefox and navigate to `about:debugging#/runtime/this-firefox`.
2. Click **Load Temporary Add-on...**.
3. Select `extension/manifest.json` from the extracted TabWrangler folder.

#### Apple Safari:
No extension installation is needed. Safari tabs are read and controlled directly via macOS AppleScript automation. If prompted on first launch, click **OK** in macOS System Settings to grant automation permission.

### 3.5 Launch for the First Time

Objective: Launch the interactive terminal UI and inspect your tabs.

Execute the binary from your terminal:

**macOS / Linux:**
```bash
./tabwrangler
```

**Windows:**
```cmd
.\tabwrangler.exe
```

Expected interface:
```text
  TabWrangler v1.0.6                     Tabs: 24 (Filtered: 24)  Selected: 0  Sort: Domain ↑
  [0] ALL (24)   [1] CHROME (14)   [2] FIREFOX (10)   [3] SAFARI (0)

  BROWSER   WINDOW   TAB   TITLE                                  DOMAIN             RAM       CPU      LAST USED
  ● Chrome  win:1    #1    GitHub - sancheza/tabwrangler          github.com         ~142 MB   0.2%     2m ago
  ● Chrome  win:1    #2    Pull Requests · sancheza/tabwrangler   github.com         ~128 MB   0.0%     14m ago
  ● Firefox win:1    #1    Python Documentation                   docs.python.org    ~94 MB    0.0%     1h ago
  ...
  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────
  [Enter/f] Focus   [d] Discard   [c] Close   [/] Filter   [s] Sort   [t] Internal   [r] Refresh   [q] Quit
```

---

## 4. CLI Usage & Flags

Run `tabwrangler -h` or `tabwrangler --help` to view all command options:

```bash
tabwrangler [flags]
tabwrangler setup [--open] [--chrome-id <id>]
tabwrangler bridge
```

### Flags

| Flag | Description |
|:--- |:--- |
| `-h`, `--help` | Show usage screen with full keyboard shortcut list and exit |
| `-v`, `--version` | Print application version (`1.0.6`) and exit |
| `--browser <name>` | Pre-filter initial view: `all`, `chrome`, `firefox`, `safari` (default: `all`) |
| `--dry-run` | Preview tab focus, discard, and close mutations in terminal without modifying browser state |
| `--mock` | Launch with simulated multi-browser tab data for testing and offline demonstrations |

### Examples

Launch pre-filtered to Firefox tabs only:
```bash
tabwrangler --browser firefox
```

Preview mutations safely without closing or discarding tabs:
```bash
tabwrangler --dry-run
```

Launch offline test UI with mock data:
```bash
tabwrangler --mock
```

---

## 5. Keyboard Navigation & Shortcuts

| Key | Context | Action |
|:--- |:--- |:--- |
| `j` / `k` or `↑` / `↓` | Navigation | Move cursor down / up |
| `g` / `G` | Navigation | Jump to top / bottom of tab list |
| `Enter` or `f` | Actions | Focus tab and bring browser window to the front |
| `d` | Actions | Discard memory for highlighted tab (or all selected tabs) |
| `c` | Actions | Close highlighted tab (or all selected tabs; asks confirmation if > 3 tabs) |
| `Space` | Selection | Toggle checkbox selection on highlighted tab |
| `a` / `A` | Selection | Select all visible tabs / Clear all selections |
| `0`–`3` or `[` / `]` | Filters | Switch browser tab filter: `[0] All`, `[1] Chrome`, `[2] Firefox`, `[3] Safari` |
| `/` | Filters | Enter live search filter (matches `Title`, `URL`, `Domain`) |
| `Esc` | Filters | Clear search filter or dismiss modal |
| `s` | Sorting | Step sort direction (most ⇄ least, A ⇄ Z), then advance to next column |
| `S` | Sorting | Invert current sort direction |
| `t` | Display | Toggle visibility of internal/special tabs (`chrome://*`, `about:*`, `file://*`) |
| `u` | Display | Toggle full URL column visibility |
| `r` | Lifecycle | Refresh tab snapshot and host process telemetry on demand |
| `?` | Helpers | Toggle in-app diagnostics modal and bridge connection status |
| `q` / `Ctrl+C` | Lifecycle | Quit TabWrangler |

---

## 6. Architecture & Engineering Standards

TabWrangler is implemented according to repository and engineering architecture standards:
- Dev ADR 0001: Architecture Decisions
- Dev ADR 0002: Standard Repository Layout
- Dev ADR 0003: Code Placement by Scope
- Dev ADR 0005: Command-Line Conventions
- Dev ADR 0020: Terminal Logging & Colors
- Repo ADR 0001: Cross-Browser TUI Architecture
