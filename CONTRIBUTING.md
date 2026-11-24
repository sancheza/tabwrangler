# Contributing to TabWrangler

## 1. Development Setup

**Requirements:** Go 1.24+, Git.

```bash
git clone https://github.com/sancheza/tabwrangler.git
cd tabwrangler
make build
make test
```

Expected output:

```
ok  	github.com/sancheza/tabwrangler/core/bridge
ok  	github.com/sancheza/tabwrangler/core/setup
ok  	github.com/sancheza/tabwrangler/core/tui
ok  	github.com/sancheza/tabwrangler/tests
```

## 2. Repository Layout

| Path | Contents |
|:--- |:--- |
| `cmd/tabwrangler/` | Entry point |
| `core/bridge/` | Native messaging host and local IPC server |
| `core/browser/` | Per-browser and per-OS adapters |
| `core/collector/` | Tab aggregation across browsers |
| `core/setup/` | Native messaging manifest installer |
| `core/telemetry/` | Process CPU and memory correlation |
| `core/tui/` | Terminal UI |
| `core/types/` | Shared data types and version |
| `extension/` | Manifest V3 companion extension (Chrome, Firefox) |
| `tests/` | Integration tests |
| `docs/` | Design specification and architecture decision records |

## 3. Pull Requests

1. Open an issue for any large change before starting.
2. Branch from `main`.
3. Add or update tests. `go test ./...` must pass; CI also runs on Windows and Linux.
4. Run `gofmt` on changed files.
5. Update `README.md` and `CHANGELOG.md` for user-visible changes.
6. Keep each pull request to one change.

## 4. Conventions

| Area | Rule |
|:--- |:--- |
| Command line | Every command answers `-h/--help` and `-v/--version`. |
| Output | User-facing messages go to stdout. Fatal errors go to stderr with a non-zero exit status. |
| Colors | Green for info, amber for warnings, red for errors. Respect `NO_COLOR`. |
| Errors | Handle specific errors. No silent catch-alls. |
| Logging | Never log tab titles, URLs, or message contents. Log sizes and event names only. |
| Dependencies | Declared in `go.mod`. Never installed at runtime. |
| Versions | `core/types/version.go`, `extension/manifest.json`, `install.sh`, and `Makefile` carry the same version. |

## 5. License of Contributions

By submitting a contribution, you agree that it is licensed under the [PolyForm Noncommercial License 1.0.0](LICENSE) and that the project owner may also offer it under a separate commercial license.

## 6. Security Issues

Report vulnerabilities privately. See [SECURITY.md](SECURITY.md).
