# Security Policy

## 1. Reporting a Vulnerability

Use GitHub private vulnerability reporting: **Security** tab → **Report a vulnerability**.

Do not open a public issue for a vulnerability.

Include:

- Affected version (`tabwrangler --version`) and operating system
- Steps to reproduce
- Impact

## 2. Supported Versions

| Version | Supported |
|:--- |:--- |
| Latest release | Yes |
| Older releases | No. Upgrade to the latest release. |

## 3. Scope

| In scope | Out of scope |
|:--- |:--- |
| Local IPC between the native messaging host and the TUI (socket, token, log files) | Vulnerabilities in browsers or operating systems |
| Native messaging manifest installation (`tabwrangler setup`) | Attacks that require an already-compromised user account |
| Companion extension (`extension/`) | |
| Installer (`install.sh`) | |

## 4. Security Model

| Property | Behavior |
|:--- |:--- |
| Network | No background connections. `tabwrangler --update` contacts GitHub over HTTPS when run, and installs a download only after its SHA-256 matches the release's `SHA256SUMS`. The IPC channel is a Unix socket (`0600`) on macOS/Linux and `127.0.0.1:42823` on Windows. |
| IPC authentication | A random per-run token in an owner-only file; registrations without it are rejected. |
| Data at rest | No tab history is stored. The diagnostic log holds timestamps, process IDs, and message sizes, not tab titles or URLs. |
| Extension permissions | `tabs` and `nativeMessaging` only. |

Details: [README Section 6](README.md#6-privacy--security-architecture).
