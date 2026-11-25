# How TabWrangler Works

This page explains what TabWrangler installs, what runs, what listens, what each part can do, and where the risks are. For the privacy summary, see [Privacy & Security](../README.md#7-privacy--security-architecture). To report a vulnerability, see [SECURITY.md](../SECURITY.md).

## 1. Components

| Component | Installed by | Where | Runs when | Listens on | Can do |
|:--- |:--- |:--- |:--- |:--- |:--- |
| **TabWrangler UI** (`tabwrangler`) | You (installer, download, or build) | A folder you choose, such as `~/.local/bin` | Only while you run it in a terminal | Unix socket (macOS/Linux) or `127.0.0.1:42823` (Windows) | Show tabs, send commands to browsers, read process CPU and RAM, run `osascript` for Safari |
| **Bridge host** (`tabwrangler bridge`, the same binary) | The browser starts it. `tabwrangler setup` registers it. | The `tabwrangler` binary | While Chrome, Edge, Brave, or Firefox keep the extension connected, including when the UI is closed | Nothing. It connects out to the UI's socket. | Relay messages between the browser and the UI. It has no tab logic. |
| **Companion extension** | You (load unpacked or temporary add-on) | The `extension/` folder | While the browser runs. The browser controls the background worker's lifetime. | Nothing | List, focus, hibernate, close, reopen, and reload tabs through the `tabs` API |
| **Native messaging manifest** | `tabwrangler setup` | Per-browser `NativeMessagingHosts` folders, plus registry keys on Windows | Never runs. It is a file the browser reads. | Nothing | Tell the browser which program to start and which extension IDs may start it |
| **Safari automation** | Nothing to install | macOS built-in `osascript` | Each refresh or action, while the UI runs | Nothing | List, focus, close, and open Safari tabs |

TabWrangler has no daemon, service, launch agent, or scheduled task. Nothing runs at login.

## 2. Data Flow

```text
 Terminal                                         Browser
┌───────────────┐   local socket   ┌──────────────┐  stdin/stdout  ┌─────────────────┐
│ TabWrangler   │◄────────────────►│ Bridge host  │◄──────────────►│ Companion       │
│ UI            │  (token on       │ (started by  │  (browser's    │ extension       │
│               │   register)      │  the browser)│   own pipe)    │ tabs API        │
└──────┬────────┘                  └──────────────┘                └─────────────────┘
       │ osascript (macOS only)  ──►  Safari
       │ process table read      ──►  CPU and RAM of browser processes
       └ session file read (Firefox, only if the extension is not connected)
```

| Step | What happens |
|:--- |:--- |
| 1. Start | The UI creates the socket, generates a random token, and writes the token to an owner-only file. |
| 2. Connect | The bridge host (already running because the browser started it) finds the token file, connects to the socket, and registers with the token. The UI rejects a registration without the correct token. |
| 3. Refresh | The UI sends `list_tabs`. The extension runs `tabs.query` and returns each tab's ID, window ID, title, URL, favicon URL, active, pinned, discarded, last-accessed time, incognito flag, and load status. |
| 4. Act | When you press a key, the UI sends one command (`focus_tab`, `discard_tabs`, `close_tabs`, `reopen_tabs`, `wake_tabs`). The extension runs it and reports counts. |
| 5. Quit | The UI closes the socket and deletes the socket and token files. The bridge host keeps waiting and registers again when the UI next starts. |

The UI refreshes every 5 seconds while it is open, after each action, and when an extension registers. A refresh takes up to 3 seconds before it gives up on a browser.

## 3. Long-Running Processes

| Process | Lifetime | Activity while idle |
|:--- |:--- |:--- |
| UI | Until you press `q` | Refresh every 5 seconds: tab query, process-table read, Safari script (macOS) |
| Bridge host | As long as the browser keeps the extension's connection open. It is not tied to the UI. | When no UI is running, checks every 200 ms for the token file and socket, and reads and dials only those. It makes no network connection. |
| Extension worker | Managed by the browser | After a disconnect, retries the connection every 3 seconds. Also retries on tab and window events. |

To stop the bridge host, disable or remove the extension, or quit the browser.

## 4. What Is Listening

| Listener | Platform | Address | Access control |
|:--- |:--- |:--- |:--- |
| UI socket | macOS, Linux | `/tmp/tabwrangler-<uid>.sock` | Mode `0600`, created under a restrictive umask. Registration needs the token. |
| UI loopback port | Windows | `127.0.0.1:42823` | Loopback only, so other computers cannot connect. Any process on the machine can connect, but registration needs the token. The token file is in your `%LOCALAPPDATA%` folder. |
| Bridge host, extension | All | None | Not applicable |

TabWrangler opens no listener on a non-loopback address.

## 5. Permissions, Authentication, and Entitlements

### 5.1 Browser Permissions

| Permission | Granted to | Gives access to |
|:--- |:--- |:--- |
| `tabs` | Extension | Title, URL, and state of every open tab in every window of that browser profile. Also lets the extension focus, discard, close, create, and reload tabs. |
| `nativeMessaging` | Extension | Starting the registered bridge host and exchanging messages with it |

The extension has no `host_permissions`, no content scripts, no `storage`, no `webRequest`, and no `history` or `bookmarks` access. It cannot read page content, cookies, or form data.

### 5.2 Who May Start the Bridge Host

| Browser | Rule in the manifest |
|:--- |:--- |
| Chrome, Edge, Brave | `allowed_origins` lists `chrome-extension://<id>/`. The default ID comes from the `key` in `extension/manifest.json`. `tabwrangler setup --chrome-id <id>` adds another ID. |
| Firefox | `allowed_extensions` lists `companion@tabwrangler.local` |

The browser refuses to start the host for any other extension.

### 5.3 Authentication Between the Bridge Host and the UI

| Property | Value |
|:--- |:--- |
| Token | 256 random bits from the operating system's secure random source, hex-encoded |
| Lifetime | One UI run. A new token is written at each start and deleted at exit. |
| Storage | Owner-only (`0600`) file. See [Files](#6-files-and-system-changes). |
| Check | Constant-time comparison on the first (`register`) message. A connection without a valid token is closed. |
| Message limit | 16 MB per native message |

### 5.4 Operating System Permissions

| Platform | Needs | Notes |
|:--- |:--- |:--- |
| All | Your normal user account | No administrator, root, or service account. No code is installed into system folders by `setup`. |
| macOS | **Automation** permission for your terminal app, to control Safari and System Events | macOS asks on first use. `tabwrangler setup` runs a test and prints a warning if the permission is missing. Deny it and Safari is unavailable; Chrome and Firefox are unaffected. |
| macOS | None other | TabWrangler declares no special entitlements and is not sandboxed. It is a command-line program that runs with your user's rights. |
| Windows | Write access to `HKCU` (your own registry hive) and `%LOCALAPPDATA%` | No administrator rights. Focusing a window uses Win32 calls and a simulated Alt key press to bring it forward. |
| Linux | Write access to your home folder | RAM and CPU figures are not collected on Linux. Window raising is not implemented. |

## 6. Files and System Changes

| Item | Location | Mode | Contents |
|:--- |:--- |:--- |:--- |
| UI socket | `/tmp/tabwrangler-<uid>.sock` (macOS/Linux) | `0600` | None. Deleted at exit. |
| Token | `/tmp/tabwrangler-<uid>.sock.token` (macOS/Linux), `%LOCALAPPDATA%\TabWrangler\bridge.token` (Windows) | `0600` | Random per-run token. Deleted at exit. |
| Diagnostic log | `/tmp/tabwrangler-<uid>.log` (macOS/Linux), `%LOCALAPPDATA%\TabWrangler\tabwrangler.log` (Windows) | `0600` | Timestamps, process IDs, connection events, counts, durations, message sizes. No tab titles or URLs. Grows until you delete it. |
| Chrome manifest | macOS `~/Library/Application Support/Google/Chrome/NativeMessagingHosts/`, Linux `~/.config/google-chrome/NativeMessagingHosts/`, Windows `%LOCALAPPDATA%\TabWrangler\NativeMessagingHosts\` and Chrome's `User Data\NativeMessagingHosts\` | `0644` | Binary path and allowed extension IDs |
| Firefox manifest | macOS `~/Library/Application Support/Mozilla/NativeMessagingHosts/`, Linux `~/.mozilla/native-messaging-hosts/`, Windows `%APPDATA%\Mozilla\NativeMessagingHosts\` | `0644` | Binary path and allowed extension ID |
| Windows registry | `HKCU\Software\{Google\Chrome, Microsoft\Edge, Chromium, BraveSoftware\Brave-Browser, Mozilla}\NativeMessagingHosts\com.tabwrangler.bridge` (32-bit and 64-bit views) | Per user | Path to the manifest |
| Windows install folder | `%LOCALAPPDATA%\TabWrangler\` | Per user | Copy of the program and the `extension/` folder |

Files the program reads:

| File | When | Purpose |
|:--- |:--- |:--- |
| Firefox `sessionstore-backups/recovery.jsonlz4` in your Firefox profiles | Only when the Firefox extension is not connected | Shows Firefox tabs without the extension. Read-only: no focus, hibernate, or close. |
| Chrome, Edge, and Brave `Preferences` files (Windows) | During `tabwrangler setup` | Finds the ID of an already-loaded TabWrangler extension |
| Browser process list and command lines (macOS, Windows) | Every refresh | Computes RAM and CPU per browser process. Only processes whose names match Chrome, Firefox, or Safari/WebKit are kept. |

`setup` also copies the extension folder path to the clipboard (`pbcopy` on macOS, `clip` on Windows).

To remove TabWrangler: delete the binary, delete the manifests and registry keys above, and remove the extension from the browser.

## 7. Network Activity

| Command | Connects to | When |
|:--- |:--- |:--- |
| `tabwrangler --update` | `api.github.com`, `github.com` over HTTPS | Only when you run it. Verifies the download against the release's `SHA256SUMS` before replacing anything. |
| `install.sh` | `github.com` over HTTPS | Only when you run it. Downloads the latest release binary. It does not check `SHA256SUMS`. |
| Everything else | Nothing | The UI, bridge host, and extension make no network requests. |

## 8. Risks

| Risk | Detail | Mitigation |
|:--- |:--- |:--- |
| The extension sees every open tab | The `tabs` permission exposes titles and URLs across all windows | The extension is one file ([`extension/background.js`](../extension/background.js)) with no network calls. Tab data goes only to the local bridge host. |
| The extension can close and navigate tabs | Any code in the extension can call `tabs.remove`, `tabs.create`, and `tabs.update` | Review `background.js`. It runs only the six actions the UI sends. |
| Another program running as you can talk to the UI | It can read the token (same user), connect, register as a browser, and return fake tab lists | Same trust level as any program running as you, which can already read your browser's files. Out of scope in [SECURITY.md](../SECURITY.md). |
| Other users on the machine | Socket and token are owner-only | `0600` modes and a restrictive umask at creation |
| Another extension or website | Cannot reach the socket. Only the registered extension IDs can start the bridge host. | Browser-enforced manifest allowlist |
| Unpacked extension | Loaded in developer mode, so there is no store review or automatic update. Firefox removes temporary add-ons when it restarts. | Reload after upgrades. See [Upgrading](upgrading.md). |
| Installer without checksum | `install.sh` trusts the HTTPS download | Use `tabwrangler --update`, or compare against `SHA256SUMS` yourself |
| Log file growth | The log is never rotated | Delete it at any time. It contains no tab titles or URLs. |
| Safari scripting | Automation permission lets the terminal app control Safari | Revoke in System Settings → Privacy & Security → Automation |
