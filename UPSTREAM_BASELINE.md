# PocketClaw Upstream Baseline

## Verified Phase 1 provenance

- PicoClaw FUI baseline commit: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`
- PicoClaw Core release: `v0.3.1`
- PicoClaw Core source commit: `2cf030d2fd3b871d7ec17e3be34c24688aac76da`
- Known-good baseline APK SHA-256: `d673acea94a8d9a38f610afaf54888e731deba33a0978f15996271c6e2510624`

The known-good Phase 1 APK is preserved outside this repository at
`/home/lordegypt/PocketCLaw/.upstream/picoclaw_fui/build/app/outputs/apk/release/app-release.apk`.
It is a regression reference and is not a PocketClaw build artifact.

## Required preserved behaviors

- Android DNS integration is verified on a physical device: Android
  `ConnectivityManager.activeNetwork` → `LinkProperties.dnsServers` →
  `PICOCLAW_DNS_SERVER` → Go PicoClaw runtime. Never use the Go Android
  localhost fallback (`[::1]:53`) and never hardcode public resolvers.
- Model discovery is verified on a physical device. Manual model entry remains
  a first-class path when discovery is unavailable.
- Telegram send/receive is verified on a physical device.
- Device feedback is optional: without complete explicit configuration it is
  disabled without retries or log spam. No Firebase credentials are included.
- Configuration persistence after a Core restart is verified on a physical
  device.
- Skill Hub/ClawHub registry search is verified on a physical device after the
  DNS fix: `Crypto` returned 20 results with metadata, URLs, and install
  actions. The earlier registry-unavailable symptom is DNS-resolved.

## Core binary pin

The initial Android foundation bundles the reviewed arm64 replacement binaries
produced from the Core source commit above with the Android DNS integration:

| File | SHA-256 |
| --- | --- |
| `libpicoclaw.so` | `3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d` |
| `libpicoclaw-web.so` | `252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3` |

## Current PocketClaw Core binaries (2026-08-25, Milestone C)

Rebuilt from the same pinned source with the Milestone C provider-catalog
changes applied (`pkg/providers`, `web/backend/api/models.go`, and the web
frontend). The Android active-network DNS integration is unchanged and verified
present in the gateway. Both are `android/arm64`, PIE, and stripped (`-s -w`).

| File | Size | SHA-256 |
| --- | --- | --- |
| `libpicoclaw.so` | 37,421,409 | `cbe568af0d6e0a1e3e4e48f7ab53fa00300509dc04f5d6ee07d0465e5556468a` |
| `libpicoclaw-web.so` | 24,772,961 | `86e53457468c6c53f6c8814b4345fcfe1ec7026e3ded388d2ab305c10cb0a4cd` |

Milestone B binaries, superseded by the above but still the last physically
verified pair until the Milestone C device test passes:
`libpicoclaw.so` `eb895f08...40bd9c88`, `libpicoclaw-web.so` `6d282df0...1195a5a3`.

## Milestone B PocketClaw Core binaries (2026-08-25)

Rebuilt from the same pinned source with the PocketClaw user-facing wording
applied. The Android active-network DNS integration is unchanged and verified
present in the gateway. Both are `android/arm64`, PIE, and stripped (`-s -w`).
The reproducible source diff and build commands are in `core/`.

| File | Size | SHA-256 |
| --- | --- | --- |
| `libpicoclaw.so` | 37,421,409 | `eb895f0892509b76242f572515c26f56530ec417bdedc0bb9ec1486f40bd9c88` |
| `libpicoclaw-web.so` | 24,772,961 | `6d282df06680869a0aca25a976b123bce8e793d2f08708e79386a1761195a5a3` |

Superseded Stage B binaries (2026-08-25, before the Milestone B final cleanup):
`libpicoclaw.so` `1f239a82...bb525fed`, `libpicoclaw-web.so` `94bb6319...6f4bc716`.

Version stamp: `Version=v0.3.1`, `GitCommit=2cf030d2`, `GoVersion=go1.25.11`.
