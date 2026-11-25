# Upgrading Guide

## 1. Upgrading TabWrangler

Objective: Update an existing TabWrangler installation to the latest version.

### Option A: Built-In Update (all platforms)

Run the update command. It finds the latest release, verifies the download against the release's `SHA256SUMS` file, and replaces the installed program. On Windows it also runs `setup`, which updates the installed program and extension folder in `%LOCALAPPDATA%\TabWrangler\`.

```bash
tabwrangler --update
```

> [!NOTE]
> The update command works from the first release that includes it (v1.0.17). Earlier versions use Option B or C.
> If the program is in a folder that needs administrator rights, such as `/usr/local/bin`, run `sudo tabwrangler --update`.

### Option B: One-Command Upgrade (macOS & Linux)

Run the installer to download the latest release binary and atomically replace `~/.local/bin/tabwrangler`:

```bash
curl -fsSL https://raw.githubusercontent.com/sancheza/tabwrangler/main/install.sh | bash
```

### Option C: Manual Binary Replacement

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

### Option D: Build from Source

Pull the latest release and compile:

```bash
git pull origin main
go build -o tabwrangler ./cmd/tabwrangler
cp tabwrangler ~/.local/bin/tabwrangler
```

### Reloading Companion Extensions

Confirm the installed version after upgrading:

```cmd
%LOCALAPPDATA%\TabWrangler\tabwrangler.exe --version
```

When an upgrade updates the companion WebExtension in `extension/`:
- **Chrome:** Open `chrome://extensions` and click the **Reload** (circular arrow) icon on the TabWrangler Companion card.
- **Firefox:** Open `about:debugging#/runtime/this-firefox` and click **Reload** under TabWrangler Companion.
- **Safari:** No action required; native automation updates with the CLI binary.

> [!NOTE]
> **Windows:** `tabwrangler --update` also replaces the extension files in `%LOCALAPPDATA%\TabWrangler\extension`. If Chrome loaded the extension from that folder, **Reload** is all it needs. If Chrome loaded it from another folder, the update says so and names both folders.
> **macOS / Linux:** `tabwrangler --update` replaces the program only. Update your extension folder to the same release (for example `git pull`), then click **Reload**.
> Press `i` in TabWrangler to see each extension's version.

