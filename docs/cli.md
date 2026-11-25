# CLI Reference

## 1. Commands

Run `tabwrangler -h` or `tabwrangler --help` to view all command options:

```bash
tabwrangler [flags]
tabwrangler setup [--open] [--chrome-id <id>]
tabwrangler -u [--dry-run]
tabwrangler bridge
```

## 2. Flags

| Flag | Description |
|:--- |:--- |
| `-h`, `--help` | Show usage screen with full keyboard shortcut list and exit |
| `-v`, `--version` | Print application version (`1.0.22`) and exit |
| `-u`, `--update` | Download and install the latest release from GitHub. With `--dry-run`, check only |
| `--browser <name>` | Pre-filter initial view: `all`, `chrome`, `firefox`, `safari` (default: `all`) |
| `--dry-run` | Preview tab focus, discard, and close mutations in terminal without modifying browser state |
| `--mock` | Launch with simulated multi-browser tab data for testing and offline demonstrations |

## 3. Examples

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

