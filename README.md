<div align="center">

<img src="assets/branding/pocketclaw-mark.png" alt="PocketClaw" width="120" />

# PocketClaw

**A private AI agent that runs on your Android phone — not on someone else's server.**

The model is remote. Everything else — the gateway, the tools, the workspace,
the credentials — lives on the device you are holding.

<br />

[![Download v0.2.0](https://img.shields.io/badge/⬇_Download-v0.2.0_APK-3DDC84?style=for-the-badge&logo=android&logoColor=white)](https://github.com/Lord1Egypt/PocketClaw/releases/latest)

<br />

[![Release gate](https://github.com/Lord1Egypt/PocketClaw/actions/workflows/release-gate.yml/badge.svg)](https://github.com/Lord1Egypt/PocketClaw/actions/workflows/release-gate.yml)
[![Latest release](https://img.shields.io/github/v/release/Lord1Egypt/PocketClaw?color=success&label=release)](https://github.com/Lord1Egypt/PocketClaw/releases/latest)
[![Downloads](https://img.shields.io/github/downloads/Lord1Egypt/PocketClaw/total?color=success)](https://github.com/Lord1Egypt/PocketClaw/releases/latest)
![Platform](https://img.shields.io/badge/platform-Android-3DDC84?logo=android&logoColor=white)
![ABI](https://img.shields.io/badge/ABI-arm64--v8a-0A7EA4)
![minSdk](https://img.shields.io/badge/minSdk-24-555555)
![targetSdk](https://img.shields.io/badge/targetSdk-36-555555)
<br />
![Flutter](https://img.shields.io/badge/Flutter-3.47-02569B?logo=flutter&logoColor=white)
![Dart](https://img.shields.io/badge/Dart-3.13-0175C2?logo=dart&logoColor=white)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![Locales](https://img.shields.io/badge/locales-12_app_·_14_dashboard-8A2BE2)
![Signed](https://img.shields.io/badge/signed-production_key-informational?logo=android&logoColor=white)
[![License](https://img.shields.io/badge/license-MIT-green)](LICENSE)
<br />
[![Stars](https://img.shields.io/github/stars/Lord1Egypt/PocketClaw?style=social)](https://github.com/Lord1Egypt/PocketClaw/stargazers)
[![Last commit](https://img.shields.io/github/last-commit/Lord1Egypt/PocketClaw)](https://github.com/Lord1Egypt/PocketClaw/commits/main)
[![Issues](https://img.shields.io/github/issues/Lord1Egypt/PocketClaw)](https://github.com/Lord1Egypt/PocketClaw/issues)

</div>

---

## Download

**[PocketClaw v0.2.0 — arm64-v8a APK](https://github.com/Lord1Egypt/PocketClaw/releases/latest)**

An `arm64-v8a` Android device on **Android 7.0 (API 24)** or newer, and an API
key for a model provider. Built and physically validated on Android 16.

<details>
<summary><b>Verify what you downloaded</b></summary>

<br />

Both files are on the release page. Check the APK against the checksum file:

```bash
sha256sum -c PocketClaw-v0.2.0-SHA256SUMS.txt
```

```
c8d599517dcaf6b954691634c2eb2c5e5cd8c88200cb77cb21a7f5861439e28c  PocketClaw-v0.2.0-arm64-v8a.apk
```

And confirm it was signed by the PocketClaw release key — a checksum proves the
file is intact, the signature proves who built it:

```bash
apksigner verify --print-certs PocketClaw-v0.2.0-arm64-v8a.apk
```

```
Signer #1 certificate SHA-256 digest: 176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf
```

That fingerprint is the public half of the signing key and is committed to this
repository at [`android/release-signing-cert.sha256`](android/release-signing-cert.sha256).
An APK that does not show it was not built by this project.

</details>

---

## What it is

PocketClaw packages a full agent runtime into a single Android app. It ships its
own Go gateway, a web dashboard, and a set of real command-line tools compiled
for `arm64-v8a`, so the agent can clone a repository, run a Python script, query
a SQLite database or call an API **on the phone**, without a build server or a
developer machine in the loop.

You bring an API key for whichever model provider you prefer. PocketClaw brings
everything around it.

## Capabilities

| | |
| --- | --- |
| **On-device gateway** | A Go binary bound to loopback only. Nothing listens to the network on your behalf. |
| **Managed Runtime** | Eight real arm64 tools, version-pinned and shipped in the APK — no package manager, no network install. |
| **Web dashboard** | A local console for providers, channels, skills, models and logs, in 14 locales. |
| **Telegram** | Guided managed-bot onboarding, or a manual bot token if you prefer. |
| **Chat** | Talk to the agent in the app, or over the built-in web channel. |
| **Workspace** | A real directory you own, on shared storage, that the agent reads and writes. |
| **Skills** | Reusable prompt/tooling bundles the agent can load per task. |
| **12-locale app** | Full app translation, right-to-left included. |

### Managed Runtime — what actually ships

Version-pinned, compiled for `arm64-v8a`, extracted from the APK at install
time. No download step, no toolchain on the device.

| Tool | Version | Tool | Version |
| --- | --- | --- | --- |
| `python` | 3.14.7 | `git` | 2.51.0 |
| `gh` | 2.82.1 | `curl` | 8.11.1 |
| `rg` | 14.1.1 | `sqlite3` | 3.50.4 |
| `jq` | 1.7.1 | `git-remote-http` | (with `git`) |

### Model providers

Anthropic (both the Messages API and the classic surface), OpenAI-compatible
endpoints, OpenAI Responses, Azure OpenAI, AWS Bedrock, and a CLI provider for
locally reachable models.

### Channel transports compiled into the Core

`pocketclaw` (the native realtime channel), `telegram`, `discord`, `slack`,
`matrix`, `irc`, `mqtt`, `whatsapp`, `qq`, `wecom`, `weixin`, `feishu`,
`dingtalk`, `line`, `onebot`, `teams_webhook`, `slack_webhook`, `vk`,
`maixcam`.

> Compiled in is not the same as first-class. Telegram and the built-in web chat
> are the two paths the app itself guides you through; the rest are configured
> from the dashboard.

## How it works

```mermaid
flowchart TD
    U["📱 Android UI<br/><i>Flutter</i>"] --> SVC["PocketClaw Service<br/><i>foreground service</i>"]
    SVC --> GW["Core Gateway<br/><b>libpocketclaw.so</b><br/>127.0.0.1:18790"]
    SVC --> WEB["Web Dashboard<br/><b>libpocketclaw-web.so</b>"]

    WEB -.->|"reads config,<br/>controls gateway"| GW

    GW --> PROV["Model providers<br/><i>Anthropic · OpenAI-compatible ·<br/>Azure · Bedrock</i>"]
    GW --> MR["Managed Runtime<br/><i>python · git · gh · curl<br/>rg · jq · sqlite3</i>"]
    GW --> CH["Channels<br/><i>Telegram · Web chat · …</i>"]

    GW --> WS["Workspace<br/><i>your files, shared storage</i>"]
    GW --> PS["Private Core state<br/><i>app-private, excluded from backup</i>"]

    style U fill:#3DDC84,stroke:#2a9d63,color:#08210f
    style GW fill:#00ADD8,stroke:#0b7fa0,color:#04222b
    style WEB fill:#00ADD8,stroke:#0b7fa0,color:#04222b
    style PS fill:#e9573f,stroke:#b8402d,color:#2b0d08
    style WS fill:#f5c542,stroke:#c69a20,color:#2b2205
```

The gateway is the only component that talks to a model. The UI, the dashboard
and every channel reach it through the same local socket, which is why the agent
behaves identically whether you type in the app, message it on Telegram, or open
the dashboard on your laptop.

## Quick start

1. Install the APK.
2. Open PocketClaw and press **Start**. The gateway comes up on loopback.
3. Open the **Dashboard** and add your provider API key.
4. Chat in the app, or connect Telegram from **Settings → Channels**.

Telegram setup is one tap: PocketClaw creates your own bot for you. There is no
token to copy, and only your own account can talk to it.

## Security and privacy design

This is the part worth reading before you trust an agent with a shell.

- **The gateway is loopback-only.** It binds `127.0.0.1:18790` and `[::1]:18790`.
  Android does not isolate loopback between apps, which is exactly why the
  gateway credential is not where another app could read it.
- **Credentials live in app-private, no-backup storage.** The gateway bearer
  token, the dashboard auth database and the Core logs are outside shared
  storage and excluded from Android backup and device transfer.
- **The GitHub token is sealed by the Android Keystore** under a
  non-exportable key, decrypted only when the service builds the Core
  environment, and never exposed to the agent.
- **The dashboard session cookie is `HttpOnly`, `SameSite=Lax`, host-only**, and
  backed by a server-side session store — the cookie alone is not a credential.
- **Logs are redacted at the source.** Keys, tokens and message text never reach
  a log line, however deeply they are nested in a structured field.
- **Your workspace stays yours.** Seeded once, then owned by you; upgrades never
  rewrite your files.

<details>
<summary><b>Build and verification guarantees</b></summary>

<br />

The two native binaries are reproducible and provenance-stamped:

- One **source fingerprint** covers every input to both shipping binaries — the
  Go gateway *and* the dashboard, including the frontend sources that become its
  embedded bundle. Both binaries carry that fingerprint, and the release gate
  fails if either is stale.
- **Builds are deterministic.** The build timestamp is derived from the last
  commit that touched a real build input, never the wall clock, so the same
  source produces byte-identical binaries.
- **`tool/release_gate.py` decides what "releasable" means**, and the same
  command runs in CI and on a developer machine, so the two cannot disagree.
- **`tool/no_active_pico.py`** enforces the namespace policy on every gate run.
- **Signing is owner-held.** The keystore and its passwords live outside this
  repository and are never in CI; only the public certificate fingerprint is
  committed, and the gate fails closed if an artifact does not carry it.

This release was signed from commit
[`ca984fa`](https://github.com/Lord1Egypt/PocketClaw/commit/ca984fa1780fa4bbf925e93e2fe45614673b4b40),
Core fingerprint `87c320c1…`, Dart AOT `5c0f5825…`.

</details>

## Development

```bash
# Flutter app
flutter pub get
flutter analyze lib/ test/
flutter test

# Core (Go) — the shipped arm64 binaries
./core/build-android-arm64.sh

# Release APK
cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64

# What "releasable" means, in one command
python3 tool/release_gate.py --verify-source --release-class production
```

Never commit generated APKs, tool caches, signing material, or
provider/Telegram/Firebase credentials.

> **Note on branches.** `develop` is the active branch and carries the released
> source; the `v0.2.0` tag points into it. Work from `develop`.

## Project status

**Released.** `v0.2.0` (build 62), physically accepted on a Samsung SM-A165F
running Android 16 and published as a production-signed `arm64-v8a` APK.

Binary hardening is complete: Dart obfuscation with private split debug info,
R8 shrinking and obfuscation, and a native/ELF audit covering every packaged
binary — all enforced by the release gate rather than asserted.

**Not in this release.** Account-login credential management — Google, Claude
subscription and ChatGPT/Codex sign-in — is deliberately deferred. Provider
access in v0.2.0 is configured with API keys through **Models**.

Known open items are tracked in [`docs/DEFECT_LOG.md`](docs/DEFECT_LOG.md); the
phase sequence is in [`docs/ROADMAP.md`](docs/ROADMAP.md). Engineering agents and
maintainers should begin with [`docs/AI_HANDOFF.md`](docs/AI_HANDOFF.md), which
defines the repository read order and milestone protocol.

## Contributing

See [`CONTRIBUTING.md`](CONTRIBUTING.md). For anything security-related, read
[`SECURITY.md`](SECURITY.md) first — please do not open a public issue.

## Upstream and attribution

PocketClaw is an independent product built on **PicoClaw**, an MIT-licensed
project by Sipeed. The Android/Flutter foundation selectively adapts PicoClaw
FUI, and PocketClaw vendors PicoClaw Core into [`core/src/`](core/src/) at
upstream release `v0.3.1`, building the shipped binaries from that tree with its
own modifications applied on top.

This is not a Git fork and it does not merge upstream history. Full attribution,
upstream commits and licence texts are in
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md),
[`UPSTREAM_BASELINE.md`](UPSTREAM_BASELINE.md) and
[`licenses/`](licenses/).

## Licence

PocketClaw's own code is [MIT licensed](LICENSE). Third-party and upstream
components keep their own licences and copyright notices — see
[`THIRD_PARTY_NOTICES.md`](THIRD_PARTY_NOTICES.md) and [`licenses/`](licenses/).

<div align="center">
<br />
<sub>Built for people who would rather their agent ran on their own hardware.</sub>
</div>
