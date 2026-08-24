# PocketClaw Session Handoff

## Current Objective

Complete Phase 2 Milestone B on `feature/pocketclaw-identity`: establish the
PocketClaw product identity and Android package migration, validate/build its
APK, then await physical-device approval. Do not begin Milestone C features.

## Exact State

This is a standalone Git repository at `/home/lordegypt/PocketClaw-App`.
It has no inherited PicoClaw FUI history. The initial source foundation is a
selective adaptation from the FUI baseline documented in `UPSTREAM_BASELINE.md`.
It is private at `https://github.com/Lord1Egypt/PocketClaw`, with `main` and
`develop` tracking `origin`. The initial commit is `950d4a3`.

The Android service and bundled Core retain the physically verified DNS bridge:
Android active-network DNS servers are passed as `PICOCLAW_DNS_SERVER`; public
resolvers are not hardcoded. Optional Firebase/Umeng feedback defaults to
disabled and requires explicit complete configuration.

Validation completed: `flutter analyze` is clean; `flutter test` passed all
28 tests; focused Core `pkg/androiddns` and `web/backend/api` tests passed.
The Milestone A independent arm64 APK was
`build/app/outputs/apk/release/app-release.apk`, 32,619,416 bytes, SHA-256
`207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`.
Its package remains `com.sipeed.picoclaw` version `0.1.3` (code `3`) by
design. Embedded Core hashes match the
pinned records. The release build reported no Firebase app ID, API key, or
project ID and removed generated Firebase resources after packaging.

The foundation physical regression was subsequently confirmed PASS, including
AI request/response, Telegram send/receive, Core restart/configuration
persistence, and no feedback log spam. Skill Hub/ClawHub was also verified:
searching `Crypto` returned 20 results with metadata, URLs, and visible install
actions. The earlier unavailable-registry symptom is resolved by the Android
DNS fix; do not rewrite Skill Hub without new source/runtime evidence.

Milestone B changes product-visible naming to PocketClaw and migrates only the
Android/Dart integration identity to `com.lord1egypt.pocketclaw`. Core names,
binary names, `PICOCLAW_DNS_SERVER`, protocol identifiers, and the compatible
external `Downloads/picoclaw` workspace remain deliberately intact. The second
original PocketClaw mark is the selected primary visual direction; launcher,
adaptive, and monochrome variants derive from it. Do not replace it with a
cartoon lobster/mascot.

Milestone B validation is complete: `flutter analyze` is clean, all 28 Flutter
tests pass, and focused pinned-Core Android DNS/model API tests pass. The new
release APK is `build/app/outputs/apk/release/app-release.apk`, 34,096,308
bytes, SHA-256
`0e440d2804978e6f94550a9d0cab563328d03cd32312bd4e33ec1ecdfc6d4883`.
`aapt` confirms `com.lord1egypt.pocketclaw`, label PocketClaw, and packaged
launcher/adaptive/splash/notification resources. Embedded Core hashes remain
`3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d` and
`252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.
The release build received no Firebase app ID, API key, or project ID and
cleaned the generated Firebase resource source after packaging.

## Next Exact Steps

1. Give the APK to the user for side-by-side physical regression testing with
   the preserved `com.sipeed.picoclaw` reference app.
2. `34b0f6b` is pushed to the private feature branch
   `origin/feature/pocketclaw-identity`; do not merge it to `develop` before
   the user's approval.
