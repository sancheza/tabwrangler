# TabWrangler

TabWrangler is a fast, keyboard-driven Terminal User Interface (TUI) for macOS, Windows, and Linux that provides centralized visibility, telemetry correlation, and lifecycle control over open tabs across Google Chrome, Mozilla Firefox, and Apple Safari.

[![TabWrangler demo: unified tab table, filtering, sorting, hibernate, close, and undo (simulated tab data)](assets/tabwrangler-demo.gif)](assets/tabwrangler-demo.mp4)

*Demo with simulated tab data*

https://github.com/user-attachments/assets/cfb2d2dd-d0b0-43fb-9d08-118fc550cc27


> **Quick Start:** To install a pre-built binary and launch TabWrangler in under two minutes without compiling from source, jump down to [3. Quick Start & Installation](#3-quick-start--installation).

---

## 1. Features

- **Unified Cross-Browser Viewport:** Inspect tabs across Chrome, Firefox, and Safari in a single virtualized table, with single-key switching to isolate individual browsers (`0`–`3`, `[` / `]`).
- **Resource Footprint & Honest Telemetry:** Measure tab CPU and RAM on demand, with visual indicators (`~`) distinguishing exact single-tab metrics from shared renderer processes.
- **Fast Lifecycle Actions:**
  - **Focus (`Enter` / `f`):** Bring any window and tab directly to the foreground.
  - **Hibernate (`h`):** Unload tabs from memory to reclaim RAM while keeping them open in the browser. A tab showing in its browser window cannot be unloaded, so TabWrangler first switches that window to another loaded tab.
  - **Close (`c`):** Close single or bulk-selected tabs, guarded by confirmation for batches greater than 3 tabs.
  - **Undo (`u`):** Reopen closed tabs or wake hibernated tabs back into active memory.
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

### 3.1 One-Command Automated Installation (macOS & Linux)

Install the pre-compiled binary and automatically configure browser native messaging manifests in a single command:

```bash
curl -fsSL https://raw.githubusercontent.com/sancheza/tabwrangler/main/install.sh | bash
```

The script automatically detects your OS and CPU architecture (Apple Silicon, Intel x86_64, ARM64), downloads the latest binary to `~/.local/bin/tabwrangler`, and registers the native messaging manifests.

---

### 3.2 Manual Installation: Pre-Compiled Binaries

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

After preparing the binary, proceed to [3.4 Register Native Messaging Host](#34-register-native-messaging-host).

### 3.3 Build from Source

Objective: Compile the application binary directly from the Git repository.

```bash
git clone https://github.com/sancheza/tabwrangler.git
cd tabwrangler
go build -o tabwrangler ./cmd/tabwrangler
```

Expected output: An executable binary `tabwrangler` (or `tabwrangler.exe` on Windows) generated in the current directory.

### 3.4 Register Native Messaging Host

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

### 3.5 Load Companion Extension into Browsers

Objective: Install the unpacked Manifest V3 companion extension in your browser.

#### Google Chrome / Microsoft Edge / Brave:
1. Open your browser and navigate to `chrome://extensions` (or `edge://extensions` / `brave://extensions`).
2. Toggle on **Developer mode** in the top right corner.
3. Click the **Load unpacked** button in the top left header.
4. Select the `extension/` folder in the file dialog (or paste the path already on your clipboard into the folder selection bar).
5. Confirm that **TabWrangler Companion** appears in your extension list with no error badges.

> [!NOTE]
> Chrome may show the warning `'background.scripts' requires manifest version of 2 or lower`. It is harmless. The manifest lists both `service_worker` (used by Chrome and Edge) and `scripts` (used by Firefox) so one extension folder works in every browser. Each browser ignores the key it does not support.

#### Mozilla Firefox:
1. Open Firefox and navigate to `about:debugging#/runtime/this-firefox` in your address bar.
2. Click the **Load Temporary Add-on...** button in the top right.
3. Select `extension/manifest.json` (or paste the path from your clipboard).
> **Note:** Do **not** use `about:addons` ➔ *"Install Add-on From File..."*. Standard Firefox requires Mozilla-signed `.xpi` archives for that menu and will report a *"corrupt"* error if given an unsigned local manifest file. Unsigned extensions must be loaded via `about:debugging#/runtime/this-firefox`.

#### Apple Safari:
No extension installation is needed. Safari tabs are read and controlled directly via macOS AppleScript automation. If prompted on first launch, click **OK** in macOS System Settings to grant automation permission.

### 3.6 Launch for the First Time

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
  TabWrangler v1.0.22                    Tabs: 24 (Filtered: 24)  Selected: 0  Sort: Domain ↑
  [0] ALL (24)   [1] CHROME (14)   [2] FIREFOX (10)   [3] SAFARI (0)

  BROWSER   WINDOW   TAB   TITLE                                  DOMAIN             RAM       CPU      LAST USED
  ● Chrome  win:1    #1    GitHub - sancheza/tabwrangler          github.com         ~142 MB   0.2%     2m ago
  ● Chrome  win:1    #2    Pull Requests · sancheza/tabwrangler   github.com         ~128 MB   0.0%     14m ago
  ● Firefox win:1    #1    Python Documentation                   docs.python.org    ~94 MB    0.0%     1h ago
  ...
  ─────────────────────────────────────────────────────────────────────────────────────────────────────────────
  [Enter/f] Focus   [h] Hibernate   [c] Close   [u] Undo   [/] Filter   [s] Sort   [t] Internal   [r] Refresh   [q] Quit
```

---

### 3.7 Upgrading TabWrangler

Objective: Update an existing TabWrangler installation to the latest version.

#### Option A: Built-In Update (all platforms)

Run the update command. It finds the latest release, verifies the download against the release's `SHA256SUMS` file, and replaces the installed program. On Windows it also runs `setup`, which updates the installed program and extension folder in `%LOCALAPPDATA%\TabWrangler\`.

```bash
tabwrangler --update
```

> [!NOTE]
> The update command works from the first release that includes it (v1.0.17). Earlier versions use Option B or C.
> If the program is in a folder that needs administrator rights, such as `/usr/local/bin`, run `sudo tabwrangler --update`.

#### Option B: One-Command Upgrade (macOS & Linux)

Run the installer to download the latest release binary and atomically replace `~/.local/bin/tabwrangler`:

```bash
curl -fsSL https://raw.githubusercontent.com/sancheza/tabwrangler/main/install.sh | bash
```

#### Option C: Manual Binary Replacement

1. Download the latest binary for your operating system from the [Releases page](https://github.com/sancheza/tabwrangler/releases/latest).
2. Replace your existing binary:

**macOS / Linux:**
```bash
cp tabwrangler-darwin-arm64 ~/.local/bin/tabwrangler.new && mv -f ~/.local/bin/tabwrangler.new ~/.local/bin/tabwrangler
```

**Windows (Command Prompt):** extract the new `tabwrangler-windows-amd64.zip`, then run setup from the extracted folder. Setup replaces the installed program and the extension folder in `%LOCALAPPDATA%\TabWrangler\`.
```cmd
.\tabwrangler.exe setup
```

#### Option D: Build from Source

Pull the latest release and compile:

```bash
git pull origin main
go build -o tabwrangler ./cmd/tabwrangler
cp tabwrangler ~/.local/bin/tabwrangler
```

#### Reloading Companion Extensions

Confirm the installed version after upgrading:

```cmd
%LOCALAPPDATA%\TabWrangler\tabwrangler.exe --version
```

When an upgrade updates the companion WebExtension in `extension/`:
- **Chrome / Edge / Brave:** Open `chrome://extensions` and click the **Reload** (circular arrow) icon on the TabWrangler Companion card.
- **Firefox:** Open `about:debugging#/runtime/this-firefox` and click **Reload** under TabWrangler Companion.
- **Safari:** No action required; native automation updates with the CLI binary.

> [!NOTE]
> **Windows:** `tabwrangler --update` also replaces the extension files in `%LOCALAPPDATA%\TabWrangler\extension`. If Chrome loaded the extension from that folder, **Reload** is all it needs. If Chrome loaded it from another folder, the update says so and names both folders.
> **macOS / Linux:** `tabwrangler --update` replaces the program only. Update your extension folder to the same release (for example `git pull`), then click **Reload**.
> Press `i` in TabWrangler to see each extension's version.

---

## 4. CLI Usage & Flags

Run `tabwrangler -h` or `tabwrangler --help` to view all command options:

```bash
tabwrangler [flags]
tabwrangler setup [--open] [--chrome-id <id>]
tabwrangler -u [--dry-run]
tabwrangler bridge
```

### Flags

| Flag | Description |
|:--- |:--- |
| `-h`, `--help` | Show usage screen with full keyboard shortcut list and exit |
| `-v`, `--version` | Print application version (`1.0.22`) and exit |
| `-u`, `--update` | Download and install the latest release from GitHub. With `--dry-run`, check only |
| `--browser <name>` | Pre-filter initial view: `all`, `chrome`, `firefox`, `safari` (default: `all`) |
| `--dry-run` | Preview tab focus, discard, and close mutations in terminal without modifying browser state |
| `--mock` | Launch with simulated multi-browser tab data for testing and offline demonstrations |

### Examples

Launch pre-filtered to Firefox tabs only:
```bash
tabwrangler --browser firefox
```

Update to the latest release:
```bash
tabwrangler --update
```

Expected output:
```
New version available: 1.0.18 (installed: 1.0.17).
Downloading tabwrangler-darwin-arm64...
Checksum verified.
[OK] Updated /Users/you/.local/bin/tabwrangler to 1.0.18.
```

Check for a newer release without installing it:
```bash
tabwrangler --update --dry-run
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

### Core Concepts

| Concept | Key | Behavior | Browser Tab Strip | Memory / CPU Impact |
|:--- |:--- |:--- |:--- |:--- |
| **Hibernate** | `h` | Puts tab to sleep using browser memory saver engine (`tabs.discard()`). | **Remains open in browser tab strip.** Retains title, URL, and favicon. Clicking wakes instantly from cache. | Frees ~90%+ renderer RAM and completely halts background CPU cycles. |
| **Close** | `c` | Permanently destroys and removes the tab (`tabs.remove()`). | **Deleted completely.** Tab is removed from the browser window. | Process and memory fully purged. |
| **Undo** | `u` | Reverts your last mutation. | Re-creates closed tabs or wakes hibernated tabs back into memory. | Restores previous tab state. |

### Shortcuts

Every key is shown in lowercase. A capital letter does the same thing (Caps Lock is harmless). The keys `A`, `D`, `G`, `S`, and `U` from earlier versions are retired: they show a hint naming the new key and do nothing else.

| Key | Context | Action |
|:--- |:--- |:--- |
| `j` / `k` or `↑` / `↓` | Navigation | Move cursor down / up |
| `g` or `Home` | Navigation | Jump to top of tab list |
| `End` | Navigation | Jump to bottom of tab list |
| `Enter` or `f` | Actions | Focus tab and bring browser window to the front |
| `h` | Actions | Hibernate memory for highlighted tab (or all selected tabs; keeps tab in browser). A tab showing in its window is hibernated after that window switches to another loaded tab |
| `c` | Actions | Close highlighted tab (or all selected tabs; asks confirmation if > 3 tabs) |
| `u` | Actions | Undo last close or hibernate action (reopens closed tabs or wakes hibernated tabs) |
| `Space` | Selection | Toggle checkbox selection on highlighted tab |
| `a` | Selection | Select all visible tabs; press again to clear the selection |
| `0`–`3` or `[` / `]` | Filters | Switch browser tab filter: `[0] All`, `[1] Chrome`, `[2] Firefox`, `[3] Safari` |
| `/` | Filters | Enter live search filter (matches `Title`, `URL`, `Domain`) |
| `Esc` | Filters | Clear search filter or dismiss modal |
| `s` | Sorting | Step sort direction (most ⇄ least, A ⇄ Z), then advance to next column |
| `o` | Sorting | Invert current sort direction |
| `t` | Display | Toggle visibility of internal/special tabs (`chrome://*`, `about:*`, `file://*`) |
| `v` | Display | Toggle full URL column visibility |
| `r` | Lifecycle | Refresh tab snapshot and host process telemetry on demand |
| `?` | Helpers | Toggle in-app help modal explaining concepts and shortcuts |
| `i` | Helpers | Toggle in-app diagnostics modal: bridge connection status and the version of each companion extension |
| `q` / `Ctrl+C` | Lifecycle | Quit TabWrangler |

---

## 6. Privacy & Security Architecture

We take security and privacy very seriously. TabWrangler operates on your local machine and collects no data. Its only network request is the optional `tabwrangler --update` command, which contacts github.com when you run it. While we encourage you to review the source code directly, we want to be completely transparent about the specific permissions required and why they are necessary.

### Browser Permissions Breakdown

When loading the companion extension, browsers prompt for two permissions:

| Permission Warning | Technical Permission | Why TabWrangler Needs It | What It Actually Does / Does Not Do |
|:--- |:--- |:--- |:--- |
| **"Read your browsing history"** | `"tabs"` | Tab listing, domain aggregation, and switching | **Only inspects currently open tabs** (titles, URLs, active window IDs) so the terminal UI can display them. It **cannot** and **does not** read your past browsing history, search logs, bookmarks, cookies, or cached session data. |
| **"Communicate with cooperating native applications"** | `"nativeMessaging"` | Local loopback bridge communication | Sends open tab metadata directly to the local TabWrangler process on your machine via standard OS standard input/output pipes. It **does not** communicate with any remote servers or third-party applications. |

### Security & Data Handling Guarantees

1. **No Background Network Requests**: The companion extension contains no `fetch()`, `XMLHttpRequest`, analytics libraries, or telemetry calls. The Go TUI makes no network requests while you use it. `tabwrangler --update` contacts `api.github.com` and `github.com` over HTTPS, only when you run it, and refuses to install a download that does not match the release's `SHA256SUMS` file.
2. **Localhost Only**: Communication between the browser extension and TabWrangler uses standard WebExtension Native Messaging (`stdin`/`stdout` managed directly by the browser executable) and a local IPC channel: a Unix domain socket (mode `0600`) on macOS/Linux, or a loopback-only TCP port (`127.0.0.1:42823`) on Windows.
3. **Authenticated IPC**: Each run generates a random 256-bit token, stored in an owner-only (`0600`) file. The native messaging host reads it and presents it when registering; connections without the correct token are rejected. The token file is deleted on exit.
4. **No Persistent History Stored**: Tab metadata exists in volatile memory only while TabWrangler is running. When you exit (`q`), all tab representations are wiped from memory. No database, SQLite file, or record of visited sites is written to disk.
5. **Minimal Permission Footprint**: The extension requests strictly `"tabs"` and `"nativeMessaging"`. It declares no `host_permissions` (`<all_urls>`), no `storage` permission, no `webRequest` interception, and no content scripts injected into websites.
6. **Auditable Source Code**: Both the Go terminal client and the JavaScript extension are open-source. The entire background worker script is contained in a single inspectable file ([extension/background.js](extension/background.js)).

### Files Written to Disk

| File | Location | Mode | Contents |
|:--- |:--- |:--- |:--- |
| IPC socket | `/tmp/tabwrangler-<uid>.sock` (macOS/Linux) | `0600` | None (local socket) |
| IPC token | `/tmp/tabwrangler-<uid>.sock.token` (macOS/Linux), `%LOCALAPPDATA%\TabWrangler\bridge.token` (Windows) | `0600` | Random per-run token; deleted on exit |
| Diagnostic log | `/tmp/tabwrangler-<uid>.log` (macOS/Linux), `%LOCALAPPDATA%\TabWrangler\tabwrangler.log` (Windows) | `0600` | Timestamps, process IDs, connection events, message sizes. No tab titles or URLs. |
| Native messaging manifests | Browser-specific `NativeMessagingHosts` directories | `0644` | Binary path and allowed extension IDs |

The diagnostic log grows over time. Delete it at any time:

```bash
rm /tmp/tabwrangler-$(id -u).log
```

---

## 7. Reporting Security Issues

See [SECURITY.md](SECURITY.md).

---

## 8. Contributing

See [CONTRIBUTING.md](CONTRIBUTING.md). Release history is in [CHANGELOG.md](CHANGELOG.md).

---

## 9. License

[PolyForm Noncommercial License 1.0.0](LICENSE).

| Use | Allowed |
|:--- |:--- |
| Personal use, hobby projects, research, education | Yes |
| Use by charities, schools, and public safety or government organizations | Yes |
| Modify and share for any noncommercial purpose | Yes, with the license text and the `Required Notice` line |
| Use in or for a commercial product, service, or business | Requires a separate commercial license |

For a commercial license, open an issue titled "Commercial license" at <https://github.com/sancheza/tabwrangler/issues>.
