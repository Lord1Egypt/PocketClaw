# PocketClaw Project State

Project: PocketClaw  
Current Phase: Phase 2 — Independent Product Repository  
Current Milestone: none in progress. Phase 2 Milestone C — Provider Catalog +
Easy API-Key Setup — is complete, including the OpenCode provider completion
and the self-contained source migration. All three PASSED physical-device
testing on 2026-08-25.
Git Branch: `feature/provider-catalog`, branched from `develop` @ `14e6991`.
Last verified milestone: Phase 2 Milestone B (merge `225be3c`, tag
`phase2-milestone-b`), which remains the fallback reference state.
Recovery Branch: `recovery/pocketclaw-clean-debrand` @ `f25d38e`, retained intact
Foundation Bootstrap Commit: `950d4a3`  
Origin: `https://github.com/Lord1Egypt/PocketClaw.git` (private)  
Upstream FUI Baseline: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`  
PicoClaw Core: `v0.3.1`, source `2cf030d2fd3b871d7ec17e3be34c24688aac76da`,
vendored into this repository at `core/src/` and built from there — see
`core/README.md` and `UPSTREAM_BASELINE.md`  
Source-of-Truth: this repository. A clone contains all application and runtime
source; no external checkout is a build dependency. Proven by
`core/verify-no-external-source.sh`.  
Build Status: arm64 release APK built through the canonical Gradle path; the
release guard verified the arm64 native payload.
APK Status: the source-migration APK
`588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b`
PASSED physical-device testing on 2026-08-25 and is the current verified
reference artifact, superseding Milestone C's `b3dd892b...bce569b`. It carries
the OpenCode completion, so the never-tested `785ccd94...56058c6` is retired.
Current Blocker: none. `feature/provider-catalog` is verified and not yet
merged; merging it into `develop` and tagging the closure point awaits
explicit instruction.
Next Exact Action: install the OpenCode completion APK and configure OpenCode
Zen and OpenCode Go with a real OpenCode API key. Run at least one inference on
each of the three protocol families so the routing is proven end to end:
a Responses-family model (gpt/codex), an Anthropic Messages-family model
(claude), and a chat-completions-family model (kimi/deepseek/glm). Do not merge
`feature/provider-catalog` into `develop` before that passes. `main` is
intentionally untouched.

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

- Milestone C: audited the provider architecture end to end and recorded it in
  `docs/PROVIDER_ARCHITECTURE.md`. Established that all AI provider
  configuration lives in the Core web console, not in Flutter, and that the
  provider catalog is already backend-owned by `pkg/providers`.
- Milestone C: extended the backend-owned catalog with `category` and
  `documentation_url`, added the xAI, Together AI, Fireworks AI, and
  Custom OpenAI-Compatible presets, and registered all four in the protocol
  switch so they actually dispatch at runtime.
- Milestone C: enabled Gemini model discovery with a dedicated fetch branch
  that uses `X-Goog-Api-Key` for the native base and Bearer for the
  OpenAI-compatible base, and broadened fetch error classification to cover
  rate limiting, provider outage, and a missing listing endpoint.
- Milestone C: replaced the Add Model form with a two-step provider-first flow
  — choose provider, paste API key, fetch or type a model, save — deriving the
  model alias automatically and moving base URL, alias, and optional keys into
  Advanced. Local and custom providers keep a visible base URL.
- Milestone C: removed runtime logo fetching from `cdn.simpleicons.org` and
  Google's favicon service; provider marks are now rendered locally.
- Milestone C: added 22 frontend tests (new vitest runner), 8 Go catalog tests,
  and 5 Go model-discovery tests. `flutter analyze` clean, 27 Flutter tests,
  Go suites for providers/config/api/androiddns/mqtt/onboard/commands/agent all
  pass, and the frontend type-checks and lints clean.

## Constraints

- Do not modify the Phase 1 workspace or its verified APK.
- Do not merge upstream repositories automatically.
- Preserve PicoClaw Core protocol/binary/environment identifiers and the
  compatible `Downloads/picoclaw` workspace path while product identity changes.
- Do not commit credentials, signing material, generated APKs, or caches.
- Do not merge `feature/provider-catalog` to `develop` before the user's
  physical-device approval, and do not touch `main`.
- Do not start Telegram QR/deep-link onboarding: it is the next milestone.
- Do not enable obfuscation or anti-reverse-engineering during active feature
  development; release hardening is a later pre-release milestone.

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

## Stage B Debranded APK — PHYSICALLY VERIFIED (previous reference)

- Status: PASS on a physical Android device, 2026-08-25. Not merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 from `recovery/pocketclaw-clean-debrand` with
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=.tooling/gradle-stage-a-clean`, Flutter 3.47.1 / Dart 3.13.1.
- Size: 34,119,837 bytes
- SHA-256: `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- arm64 payload verified by the release guard: `libdartjni.so` (131,248),
  `libpicoclaw.so` (37,421,409), `libpicoclaw-web.so` (24,772,961).
- Embedded Core hashes: gateway
  `1f239a827c8562d6ac2ffdf63c1354ce0d28396cdab7d4f240f3866cbb525fed`, web
  `94bb6319bbac08e1aa0fa43e8093b4dd00bad512cb67ca94a6a57d666f4bc716`.
- Physical-device results (2026-08-25): install PASS, app launch PASS, Flutter
  first frame PASS, black-screen regression FIXED, Gateway/Core startup PASS,
  navigation PASS, PocketClaw branding PASS, PocketClaw workspace path PASS,
  QR/access page PASS, no abnormal device slowdown observed.
- This is the reference physically verified PocketClaw artifact. Compare any
  future build against it.

## Pre-release APK check (mandatory)

`packageRelease` now fails the build if `lib/arm64-v8a/` is missing
`libdartjni.so`, `libpicoclaw.so`, or `libpicoclaw-web.so`, and prints a
"Verified arm64-v8a native payload" line when it passes. If `libdartjni.so` is
reported missing, purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and
rebuild — that cache is outside the project `build/` tree, so cleaning build
intermediates does not clear it.

`./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` is the
canonical release path. Do not release a universal `flutter build apk --release`.

## Milestone B Final Cleanup APK — VERIFIED REFERENCE ARTIFACT

- Status: PASS on a physical Android device, 2026-08-25. Merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`.
- Size: 34,119,477 bytes
- SHA-256: `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `eb895f0892509b76242f572515c26f56530ec417bdedc0bb9ec1486f40bd9c88` |
| `libpicoclaw-web.so` | 24,772,961 | `6d282df06680869a0aca25a976b123bce8e793d2f08708e79386a1761195a5a3` |

- Contents of this cleanup: MQTT fresh default is `/pocketclaw` while any
  explicitly configured prefix (including the legacy `/picoclaw`) is preserved;
  `skills/picoclaw-agent` is no longer seeded into a fresh workspace while
  existing user copies are untouched; factual Sipeed hardware references are
  retained deliberately.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Physical-device results (2026-08-25): install PASS, app launch / first frame
  PASS, no black screen, Gateway/Core lifecycle PASS, navigation PASS,
  PocketClaw branding PASS, workspace path PASS, QR/access page PASS, Skill Hub
  PASS, provider/model flow PASS, no abnormal slowdown.
- This is the current verified reference artifact. Compare any future
  regression against it before forming new hypotheses.

## Milestone C Provider Catalog APK — PHYSICALLY VERIFIED (superseded)

- Status: PASS on a physical Android device, 2026-08-25. This is the verified
  reference artifact, superseding Milestone B's `ba4f067d...70f70a4b8`.
  Not merged to `develop`: the OpenCode completion below rides on the same
  branch and must pass its own device test first.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,123,401 bytes
- SHA-256: `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `cbe568af0d6e0a1e3e4e48f7ab53fa00300509dc04f5d6ee07d0465e5556468a` |
| `libpicoclaw-web.so` | 24,772,961 | `86e53457468c6c53f6c8814b4345fcfe1ec7026e3ded388d2ab305c10cb0a4cd` |

- `libdartjni.so` is byte-identical to the Milestone B verified build.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, which is the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.
- What to test on the device: the Add Provider picker opens and lists providers
  by category; selecting a cloud provider asks only for an API key and a model;
  Fetch Models succeeds against a real provider; a failed fetch still allows a
  manually typed model ID; Custom OpenAI-Compatible accepts a base URL, key, and
  model ID; an existing provider still opens with its stored values intact; and
  startup, DNS, Core lifecycle, Telegram, Skill Hub, workspace, MQTT, and
  branding are all unregressed.

## Milestone C OpenCode Completion APK — RETIRED, NEVER TESTED

- Status: RETIRED without ever being tested. Its functionality is contained in
  the physically verified source-migration APK `588bbec1...5ff3a90b`, which
  supersedes it. Kept here only as a record of what was built.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,129,765 bytes
- SHA-256: `785ccd94cfa351ee2996ac340f9a55e828a0c8f736bec67a3edac906a56058c6`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Only `arm64-v8a`
  carries `libapp.so`, `libflutter.so`, and the two Core payloads; the other
  ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `e48e8af073d6e7dfdb46ba8268785780d1f900888b82dba41747ef9212e78938` |
| `libpicoclaw-web.so` | 24,772,961 | `5faaf82ccbcd2fbad27d7ffc336f240fd5c08a48a1abbb2bd4eff7c383fe2abf` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.

### What to test on the device

The rest of the app is unchanged from the verified Milestone C build, so the
regression sweep can be brief. The new surface is the two OpenCode presets:

1. OpenCode Zen appears in Add Provider, asks only for an API key, and Fetch
   Models returns the live list from `https://opencode.ai/zen/v1/models`.
2. OpenCode Go does the same against `https://opencode.ai/zen/go/v1/models`.
3. Run one real inference on each protocol family, per provider, because the
   protocol is chosen per model and only a live request proves the route:
   - a Responses-family model (`gpt-*`, `*codex*`),
   - an Anthropic Messages-family model (`claude-*`),
   - a chat-completions-family model (`kimi-*`, `deepseek-*`, `glm-*`).
4. Confirm the model ID saved and sent is the bare ID (`kimi-k3`), not the
   namespaced OpenCode CLI form.

The one assumption that only a device can settle is the Messages
authentication form — see the OpenCode routing decision in `DECISIONS.md`.
If a `claude-*` model returns 401 while `gpt-*` and `kimi-*` succeed, that is
the bearer-versus-`X-API-Key` question, not a routing failure.

## Self-Contained Source Migration APK — PHYSICALLY VERIFIED REFERENCE ARTIFACT

- Status: PHYSICALLY VERIFIED on 2026-08-25. This is the current verified
  reference artifact, superseding Milestone C's `b3dd892b...bce569b`. Bisect or
  diff any future regression against it before forming new hypotheses. It
  carries the OpenCode completion, so `785ccd94...56058c6` is retired untested.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`,
  after building Core from `core/src/` with `core/build-android-arm64.sh`.
- Size: 34,123,225 bytes
- SHA-256: `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,224,801 | `cb9b2cdea1ccd7ddbbda723ddd3bed1d8c3a931638b1952dd767f62efb895818` |
| `libpicoclaw-web.so` | 24,641,889 | `b6b356f75eb348933e6cb1049890bb8be4d2bd20b605f55b23a5487656db9ba5` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped, `ARM aarch64` PIE, and are the first built
  from repository-local source and the first built with `-trimpath`. They are
  roughly 197 KB and 131 KB smaller than the previous pair for that reason.
- Developer-machine paths in the packaged Core binaries: 0 and 0. The previous
  pair carried 2,501 and 1,346. `core/build-android-arm64.sh` fails the build
  if this regresses.
- `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw` — identical to the verified Milestone C counts.
- Pre-build validation: Go suites 92 ok / 0 failed, frontend `vitest` 28
  passed, `pnpm lint` clean, `flutter analyze` no issues, `flutter test` 27
  passed.

### Physical-device results — 2026-08-25, all PASS

| Check | Result |
| --- | --- |
| Install / startup | PASS |
| Flutter first frame | PASS |
| Gateway/Core lifecycle | PASS |
| User-facing logs free of developer absolute paths | PASS |
| User-facing logs free of PicoClaw product branding | PASS |
| Provider catalog | PASS |
| OpenCode Zen preset | PASS |
| OpenCode Zen Fetch Models | PASS |
| OpenCode Zen real request/response | PASS |
| OpenCode Go preset | PASS |
| OpenCode Go Fetch Models | PASS |
| OpenCode Go real request/response | PASS |
| Gemini / provider regression | PASS |
| Skill Hub regression | PASS |
| Telegram regression | PASS |
| Workspace regression | PASS |
| No black screen | PASS |
| No abnormal slowdown | PASS |

Three results carry more weight than the rest.

The logs check is the first device confirmation of the `-trimpath` change. The
build-time assertion proved the strings were absent from the binaries; this
proves nothing surfaces them on the screen a user actually reads.

Skill Hub search and Fetch Models both working is the end-to-end proof that the
Android active-network DNS integration survived being rebuilt from a relocated
source tree. Those paths fail closed without working DNS, so a PASS on both
means the integration is intact — not merely present as a string in the binary.

The four OpenCode results are the first live confirmation of per-model protocol
routing. Zen and Go each returned a real response, so the routing table in
`pkg/providers/opencode_routing.go` is exercised rather than assumed.

One question stays open, and this PASS does not close it. `DECISIONS.md`
records that OpenCode's Anthropic Messages surface is sent both `X-API-Key` and
a bearer header because the correct form could not be established offline. That
is only settled by a `claude-*` model returning a real response, and the device
report does not say which model families were exercised. Treat the Messages
route as unconfirmed until a `claude-*` inference is observed; if one 401s while
`gpt-*` and `kimi-*` succeed, the header pair is the cause, not the routing.
