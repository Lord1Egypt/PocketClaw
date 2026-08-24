# PocketClaw Project State

Project: PocketClaw  
Current Phase: Phase 2 — Independent Product Repository  
Current Milestone: Stage A recovery — awaiting physical-device test of the
black-screen replacement APK
Git Branch: `recovery/pocketclaw-clean-debrand` (from the working `8eeef15`)
Foundation Bootstrap Commit: `950d4a3` — `chore: bootstrap PocketClaw independent Android foundation`  
Origin: `https://github.com/Lord1Egypt/PocketClaw.git` (private)  
Upstream FUI Baseline: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`  
PicoClaw Core: `v0.3.1`, source `2cf030d2fd3b871d7ec17e3be34c24688aac76da`  
Build Status: Stage A arm64 release APK rebuilt after the black-screen root
cause was found and fixed in the build environment
APK Status: Awaiting physical-device verification
Current Blocker: None known. The black screen is root-caused to a missing
`lib/arm64-v8a/libdartjni.so` caused by a poisoned `jni` CMake configure cache
in `~/.pub-cache`; see `DECISIONS.md` and `CHANGELOG_DEV.md`.
Next Exact Action: Install and launch
`build/app/outputs/apk/release/app-release.apk`
(SHA-256 `d87425344cca526afd8ef3eca41ef3648791593e69016a463aeee86693d7e405`) on
the physical device and confirm a usable Flutter first frame.

## Completed

- Phase 1 baseline APK and all physical-device checks.
- Created a separate PocketClaw directory without upstream Git history.
- Selectively adapted the Android/Flutter foundation, tests, tools, and Core
  packaging from the reviewed FUI baseline.
- Preserved the Android DNS and optional-feedback behaviors.
- Recorded provenance, upstream tracking, and third-party notices.
- Audited and recorded the FUI/Core MIT notices and component classification.
- Ran `flutter analyze` (clean), `flutter test` (28 passed), and focused Core
  Android DNS/model API tests (passed).
- Built and inspected the independent arm64 foundation APK. Its embedded Core
  hashes match the pinned replacement binaries; Firebase build values are absent.
- Completed a Git diff/stat/check review and credential scan. No credentials,
  signing files, Firebase config, generated APKs, or caches are eligible for
  commit; the standard Gradle Wrapper files are intentionally retained.
- Created the private GitHub repository `Lord1Egypt/PocketClaw`, pushed
  `main`, and created/pushed the `develop` integration branch.
- Physical-device verification confirmed the independent foundation APK,
  launch, Core/Gateway, active-network DNS, model discovery/manual model,
  AI requests, Telegram, restart/persistence, and optional feedback behavior.
- Physical-device verification confirmed ClawHub Skill Hub search: a `Crypto`
  query returned 20 results with metadata, URLs, and install actions. The
  prior registry-unavailable observation is classified as resolved by the
  Android DNS fix, not as an independent Skill Hub defect.
- Started Milestone B on `feature/pocketclaw-identity`: product naming,
  independent package identity, original visual direction, Android icon/splash
  treatments, design tokens, and the required audit records are complete.
- Passed Milestone B `flutter analyze`, all 28 Flutter tests, and focused
  pinned-Core Android DNS/model API tests. Built and inspected the new APK:
  package/label are `com.lord1egypt.pocketclaw`/PocketClaw, Android branding
  resources are bundled, the Core hashes match the pin, and no Firebase config
  values are present.
- Committed Milestone B as `34b0f6b` and pushed
  `feature/pocketclaw-identity` to the private `origin`; `develop` and `main`
  remain untouched pending the user's physical-device approval.
- Root-caused the Milestone B black-screen regression to the two branded Android
  `launch_background.xml` resources. A `layer-list` `<item android:color>` is
  not a valid drawable layer: Android must inflate a drawable-backed item before
  Flutter can replace `LaunchTheme`. The corrected resources use
  `@color/pocketclaw_splash_background` through `android:drawable`.
- Added a source-level regression test for both launch-background variants.
  `flutter analyze` is clean; all 29 Flutter tests and focused Core
  `pkg/androiddns`/`web/backend/api` regressions pass. A debug and a replacement
  arm64 release APK were built and the release's compiled layer-list was
  inspected to confirm the first item has a drawable reference.

## Constraints

- Do not modify the Phase 1 workspace or its verified APK.
- Do not merge upstream repositories automatically.
- Preserve PicoClaw Core protocol/binary/environment identifiers and the
  compatible `Downloads/picoclaw` workspace path while product identity changes.
- Do not commit credentials, signing material, generated APKs, or caches.
- Do not merge Milestone B to `develop` before the user's physical-device
  approval, and do not begin Milestone C features.

## Independent Foundation APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 03:06:47 +03:00
- Size: 32,619,416 bytes
- SHA-256: `207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`
- Package/version: `com.sipeed.picoclaw`, `0.1.3` (version code `3`)
- Label: `PicoClaw` (intentionally unchanged for this foundation milestone)
- Architecture: functional application/Core payload is `arm64-v8a`

## Milestone B PocketClaw APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24
- Size: 34,096,308 bytes
- SHA-256: `0e440d2804978e6f94550a9d0cab563328d03cd32312bd4e33ec1ecdfc6d4883`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Architecture: universal APK; the PicoClaw Core payload is `arm64-v8a` and
  its two pinned hashes match `UPSTREAM_BASELINE.md`.

## Milestone B Runtime-Regression Replacement APK

- Status: BLOCKED — physical-device retest pending.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 04:34:40 +03:00
- Size: 33,308,653 bytes
- SHA-256: `45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Core payload hashes: unchanged — gateway
  `3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d`, web
  `252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.

## Stage A Black-Screen Replacement APK

- Status: Awaiting physical-device test.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 01:10 from `642dc63` with
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=.tooling/gradle-stage-a-clean`, Flutter 3.47.1 / Dart 3.13.1.
- Size: 34,119,437 bytes
- SHA-256: `d87425344cca526afd8ef3eca41ef3648791593e69016a463aeee86693d7e405`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- Native code advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and
  Core payloads are `arm64-v8a`; the other ABIs carry only plugin JNI libs,
  matching the previously working artifact.
- `lib/arm64-v8a/libdartjni.so`: PRESENT, 131,248 bytes, AArch64. Its absence
  was the black-screen root cause.
- Core payload hashes unchanged: gateway
  `3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d`, web
  `252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.

## Pre-release APK check (mandatory)

Before shipping any PocketClaw Android release, confirm the packaged
`lib/arm64-v8a/` contains `libdartjni.so`:

```bash
unzip -l build/app/outputs/apk/release/app-release.apk | grep arm64-v8a
```

If it is missing, purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and
rebuild. The cache is outside the project `build/` tree, so cleaning build
intermediates does not clear it.
