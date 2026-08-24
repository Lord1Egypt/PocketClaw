# PocketClaw Session Handoff

## Current Objective

Physically test the Stage A replacement APK. The black-screen regression is
root-caused and fixed in the build environment; no source change was needed.
Do not resume debranding or begin new features until the device test passes.

## Black-screen root cause (resolved)

Every APK built from `/home/lordegypt/PocketClaw-App` since 2026-08-24 02:52
was missing `lib/arm64-v8a/libdartjni.so`. The `jni` package's CMake configure
for `arm64-v8a` failed once with a transient filesystem error and CMake cached
the failure as a valid configure with an empty target list in
`~/.pub-cache/hosted/pub.dev/jni-1.0.3/android/.cxx/RelWithDebInfo/6n1p6673/`.
Because that cache lives outside the project `build/` tree, deleting build
intermediates never cleared it.

`com.github.dart_lang.jni.JniPlugin` calls `System.loadLibrary("dartjni")` from
a static initializer. `GeneratedPluginRegistrant.registerWith` wraps plugin
construction in `catch (Exception e)`, but the failure is an
`ExceptionInInitializerError` — an `Error`, not an `Exception` — so it escapes
during `FlutterActivity.onCreate` and the Flutter view never attaches. The app
process stays alive as a foreground service, which matches the reported black
screen plus device slowdown. `path_provider_android` depends on
`jni`/`jni_flutter`, so the plugin is always registered.

The physically working APK `45be7269...ebe9e` was built from
`/tmp/pocketclaw-runtime-fix`, whose separate cache
(`.cxx/RelWithDebInfo/4l131246`) configured arm64-v8a successfully. That is the
sole material difference; Stage A's source changes are text/UI only.

Fix applied: purged the poisoned `.cxx` cache and rebuilt. Always verify
`lib/arm64-v8a/libdartjni.so` is packaged before releasing.

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

1. Install and launch `build/app/outputs/apk/release/app-release.apk`
   (34,119,437 bytes, SHA-256
   `d87425344cca526afd8ef3eca41ef3648791593e69016a463aeee86693d7e405`) on the
   physical device. Confirm the Flutter first frame, usable UI, Core start/stop,
   active-network DNS, model discovery, Telegram, and Skill Hub.
2. If it launches, Stage A is validated; only then consider merging or resuming
   debranding. `recovery/pocketclaw-clean-debrand`, `feature/pocketclaw-identity`
   (`354fc38` WIP preserved), `develop`, and `main` are all intact and unmerged.
3. If it still black-screens, capture `adb logcat` around
   `GeneratedPluginRegistrant`/`UnsatisfiedLinkError` before changing any source.
