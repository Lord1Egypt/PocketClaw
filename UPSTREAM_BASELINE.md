# PocketClaw Upstream Baseline

## Adoption record

This section is permanent. It records exactly which upstream state PocketClaw
started from and when that state was adopted, so a future maintainer never has
to reconstruct it from commit archaeology. Values here are never edited to
track upstream movement — movement is recorded in `UPSTREAM_TRACKING.md`.

Adopted by PocketClaw:
2026-08-24

Baseline release:
v0.3.1

Baseline commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Baseline commit authored:
2026-06-30

Baseline release published:
2026-07-03

Upstream Core repository:
https://github.com/sipeed/picoclaw

Historical FUI reference commit:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

Historical FUI reference release:
picoclaw_fui-v0.1.4 (tag `v0.1.4`, published 2026-06-04)

Upstream FUI repository:
https://github.com/sipeed/picoclaw_fui

PocketClaw adoption commit:
`950d4a3` — `chore: bootstrap PocketClaw independent Android foundation`
(2026-08-24)

The FUI is a historical reference only. PocketClaw is an independent Android
application and is not a maintained fork of `picoclaw_fui`; see
`DECISIONS.md`, "Independent repository, not a permanent FUI fork".

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

## Current PocketClaw Core binaries (2026-08-25, Milestone C OpenCode completion)

Rebuilt from the same pinned source with the OpenCode Zen/Go presets, the
per-model routing table, and the new generic Responses provider added on top of
the Milestone C provider-catalog changes. The Android active-network DNS
integration is unchanged and verified present in the gateway. Both are
`android/arm64`, PIE, and stripped (`-s -w`).

| File | Size | SHA-256 |
| --- | --- | --- |
| `libpicoclaw.so` | 37,421,409 | `e48e8af073d6e7dfdb46ba8268785780d1f900888b82dba41747ef9212e78938` |
| `libpicoclaw-web.so` | 24,772,961 | `5faaf82ccbcd2fbad27d7ffc336f240fd5c08a48a1abbb2bd4eff7c383fe2abf` |

Physically verified Milestone C pair, superseded by the above but still the
last device-proven binaries until the OpenCode completion is tested:
`libpicoclaw.so` `cbe568af...5556468a`, `libpicoclaw-web.so` `86e53457...0cb0a4cd`.

Milestone B pair: `libpicoclaw.so` `eb895f08...40bd9c88`,
`libpicoclaw-web.so` `6d282df0...1195a5a3`.

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
