# Development Changelog

## 2026-08-25 — Black-screen root cause: missing arm64 `libdartjni.so`

- Diagnosed the persistent PocketClaw black screen by differential forensics on
  build artifacts and the AGP/CMake configure caches. It is a build-environment
  defect, not a Stage A source defect.
- Root cause: the `jni` package's CMake configure for `arm64-v8a` failed inside
  this workspace on 2026-08-24 02:52 with a transient filesystem error and was
  cached as a valid-but-empty configure in
  `~/.pub-cache/hosted/pub.dev/jni-1.0.3/android/.cxx/RelWithDebInfo/6n1p6673/`.
  Every subsequent build from `/home/lordegypt/PocketClaw-App` therefore
  packaged **no** `lib/arm64-v8a/libdartjni.so`.
- `JniPlugin`'s static initializer calls `System.loadLibrary("dartjni")`, and
  `GeneratedPluginRegistrant` catches only `Exception`; the resulting `Error`
  escapes during `FlutterActivity.onCreate`, so Flutter never renders a frame.
- The physically working APK `45be7269...ebe9e` was built from a different
  directory (`/tmp/pocketclaw-runtime-fix`), which produced a separate cache
  (`.cxx/RelWithDebInfo/4l131246`) whose arm64-v8a configure succeeded. That is
  the only material difference between the working and failing artifacts.
- Fix: purged the poisoned `.cxx` cache and rebuilt with the documented
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` command and
  the pinned Flutter 3.47.1 / Dart 3.13.1 / Java 17 / AGP 8.11.1 toolchain.
  No source change was required.
- Replacement APK: `build/app/outputs/apk/release/app-release.apk` (also copied
  to `build/app/outputs/flutter-apk/app-release.apk`), 34,119,437 bytes,
  SHA-256 `d87425344cca526afd8ef3eca41ef3648791593e69016a463aeee86693d7e405`.
  Verified: `lib/arm64-v8a/libdartjni.so` present (131,248 bytes, AArch64),
  package `com.lord1egypt.pocketclaw`, label PocketClaw, launchable
  `com.lord1egypt.pocketclaw.MainActivity`, pinned Core hashes unchanged,
  drawable-backed splash layer intact.
- `flutter analyze` clean; all 27 Flutter tests pass.
- Closed hypotheses: stale `libpicoclaw-web.so`, workspace migration, splash
  resources, and Stage A wording changes are all excluded by this evidence.

## 2026-08-24 — Milestone B runtime-regression fix (physical retest pending)

- Recorded Milestone B physical validation as BLOCKED after the branded APK
  installed and launched but remained on a black screen with no usable Flutter
  UI.
- Root-caused the regression to invalid Android splash `layer-list` syntax:
  both launch backgrounds used `android:color` on an item instead of supplying
  a drawable. Replaced the invalid item with the named
  `@color/pocketclaw_splash_background` drawable reference.
- Added `test/unit/android_launch_theme_resource_test.dart` to reject the
  invalid color-only layer in both resource variants.
- Passed `flutter analyze`, all 29 Flutter tests, focused Core
  `pkg/androiddns`/`web/backend/api` tests, debug APK build, and release APK
  build. Inspected the compiled release resource and verified package, label,
  MainActivity, and unchanged pinned Core hashes.
- Replacement APK: `build/app/outputs/flutter-apk/app-release.apk`,
  33,308,653 bytes, SHA-256
  `45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`.
- No merge to `develop`; physical-device retest is required before Milestone B
  can be approved.

## 2026-08-24 — Phase 2 Milestone B identity foundation

- Started `feature/pocketclaw-identity` from the validated `develop` state.
- Implemented the PocketClaw product-visible naming and planned independent
  Android package identity `com.lord1egypt.pocketclaw`, while preserving
  PicoClaw Core integration identifiers and compatible external workspace path.
- Added branding, package-migration, asset, image-generation, and Skill Hub
  regression documentation; established centralized Material 3 design tokens.
- Selected the second original PocketClaw mark as the primary visual direction
  and created matching Android launcher/adaptive, splash, and monochrome
  notification treatments. No cartoon lobster/mascot was adopted.
- Recorded physical Skill Hub/ClawHub success after the DNS fix: the prior
  registry-unavailable symptom is resolved by Android DNS, not a separate hub
  defect.
- Passed `flutter analyze`, all 28 Flutter tests, and focused pinned-Core
  Android DNS/model API tests; built the new PocketClaw APK.
- Inspected the APK: package `com.lord1egypt.pocketclaw`, label PocketClaw,
  branding resources, and pinned Core hashes are correct. The release build
  had no Firebase app ID/API key/project ID and cleaned generated resources.
- Committed as `34b0f6b` and pushed `feature/pocketclaw-identity` to private
  `origin`; `develop` and `main` remain unchanged pending physical approval.

## 2026-08-24 — Phase 2 Milestone A bootstrap

- Created the independent PocketClaw Android/Flutter foundation directory.
- Selectively adapted the reviewed FUI Android, Flutter, test, asset, and tool
  foundation without copying upstream Git history, build outputs, or caches.
- Preserved the pinned Core `v0.3.1` replacement binaries, Android
  active-network DNS integration, and optional feedback behavior.
- Added provenance, upstream-tracking, decision, handoff, task, and
  third-party-license records.
- Passed `flutter analyze`, all 28 Flutter tests, and focused Core Android
  DNS/model API tests.
- Built and inspected the independent arm64 foundation APK; it retains the
  pinned Core hashes and has no Firebase app ID, API key, or project ID.
- Created private GitHub repository `Lord1Egypt/PocketClaw` and pushed initial
  commit `950d4a3` to both `main` and `develop`.
