# PocketClaw Session Handoff

## Current Objective

None outstanding. Stage B is complete and physically verified. Do not start new
features and do not merge anything without the user's explicit instruction.

## Status: black-screen incident RESOLVED

APK `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4` passed a
physical Android device test on 2026-08-25: install, app launch, Flutter first
frame, Gateway/Core startup, navigation, PocketClaw branding, PocketClaw
workspace path, and the QR/access page all PASS, with no abnormal device
slowdown. This is the reference physically verified PocketClaw artifact.

Root cause, now confirmed: builds from this workspace were missing
`lib/arm64-v8a/libdartjni.so`. The `jni` package's arm64 CMake configure failed
once with a transient filesystem error, and CMake cached that failure as a
successful configure with an empty target list under `~/.pub-cache` — outside
the project `build/` tree, so no amount of cleaning `build/` cleared it.
`JniPlugin` loads that library from a static initializer, and
`GeneratedPluginRegistrant` catches only `Exception`, so the resulting `Error`
escaped `FlutterActivity.onCreate` and no frame was ever drawn.

## Permanent release requirements — do not remove

- `packageRelease` fails the build unless the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
  It prints "Verified arm64-v8a native payload" on success.
- The canonical release command is
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`, run with
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`. A universal
  `flutter build apk --release` is not a PocketClaw release path.
- If the guard reports `libdartjni.so` missing, purge
  `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and rebuild.

## Stage B debranding

Every normal user-facing surface reads PocketClaw. The embedded web runtime was
rebranded at source and rebuilt, never binary-patched, and the gateway was
rebuilt too because the assistant identity lives in it; `PICOCLAW_DNS_SERVER` is
verified still present. `docs/BRANDING_AUDIT.md` holds the full classification
and the two remaining open product decisions. `core/` holds the Core source
patch and the exact rebuild commands.

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

The Milestone B physical test then found a black startup screen and noticeable
device slowdown. The root cause is the branded Android launch drawable: both
`launch_background.xml` variants used `<item android:color=...>` in a
`layer-list`; `LayerDrawableItem` requires `android:drawable` (or a child
drawable), so Android cannot inflate the launch window before Flutter's first
frame. The replacement changes only those two items to reference
`@color/pocketclaw_splash_background` and adds a regression test. The 1254 px
RGBA splash mark is 654,845 bytes on disk and about 6.0 MiB decoded, which is
within a reasonable startup budget and was not the root cause.

The fixed validation results are: `flutter analyze` clean; all 29 Flutter
tests pass; focused Core `pkg/androiddns` and `web/backend/api` tests pass.
Debug and release APKs built successfully. The replacement release is
`build/app/outputs/flutter-apk/app-release.apk`, 33,308,653 bytes, SHA-256
`45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`.
`aapt2` confirms `com.lord1egypt.pocketclaw`, label PocketClaw, launchable
`com.lord1egypt.pocketclaw.MainActivity`, and a compiled splash layer with a
drawable-backed first item. Embedded Core hashes remain
`3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d` and
`252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.
The service/Core code was unchanged: a fresh Flutter launch does not issue a
Core start request, and the native service has its bounded three-restart guard.
No connected ADB device was available to measure the reported slowdown or
capture final device logs, so that symptom must be confirmed during retest.

## Next Exact Steps

1. Decide the two open branding items in `docs/BRANDING_AUDIT.md`: the MQTT
   `/picoclaw` topic prefix (a broker-side protocol identifier, visible only in
   the MQTT channel config screen) and the bundled `picoclaw-agent` / `hardware`
   skills seeded into the onboarding workspace.
2. Await the user's instruction before merging or starting new work.
   `recovery/pocketclaw-clean-debrand`, `feature/pocketclaw-identity`
   (`354fc38` WIP preserved), `develop`, and `main` are all intact and unmerged.
