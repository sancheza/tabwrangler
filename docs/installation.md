# Installation Guide

Setup steps after you have a TabWrangler binary. To get the binary, see [Quick Start](../README.md#3-quick-start--installation).

## 1. Register Native Messaging Host

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

## 2. Load Companion Extension into Browsers

Objective: Install the unpacked Manifest V3 companion extension in your browser.

### Google Chrome:
1. Open Chrome and navigate to `chrome://extensions`.
2. Toggle on **Developer mode** in the top right corner.
3. Click the **Load unpacked** button in the top left header.
4. Select the `extension/` folder in the file dialog (or paste the path already on your clipboard into the folder selection bar).
5. Confirm that **TabWrangler Companion** appears in your extension list with no error badges.

> [!NOTE]
> Chrome may show the warning `'background.scripts' requires manifest version of 2 or lower`. It is harmless. The manifest lists both `service_worker` (used by Chrome and Edge) and `scripts` (used by Firefox) so one extension folder works in every browser. Each browser ignores the key it does not support.

### Mozilla Firefox:
1. Open Firefox and navigate to `about:debugging#/runtime/this-firefox` in your address bar.
2. Click the **Load Temporary Add-on...** button in the top right.
3. Select `extension/manifest.json` (or paste the path from your clipboard).
> **Note:** Do **not** use `about:addons` ➔ *"Install Add-on From File..."*. Standard Firefox requires Mozilla-signed `.xpi` archives for that menu and will report a *"corrupt"* error if given an unsigned local manifest file. Unsigned extensions must be loaded via `about:debugging#/runtime/this-firefox`.

### Apple Safari:
No extension installation is needed. Safari tabs are read and controlled directly via macOS AppleScript automation. If prompted on first launch, click **OK** in macOS System Settings to grant automation permission.

## 3. Launch for the First Time

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

