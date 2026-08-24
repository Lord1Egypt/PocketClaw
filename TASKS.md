# PocketClaw Tasks

## Phase 2 — Milestone A: Independent Foundation

- [x] Confirm Phase 1 completion and preserve the verified baseline workspace.
- [x] Create a standalone PocketClaw directory with no upstream Git history.
- [x] Classify and selectively adapt the required Flutter/Android foundation (`docs/fui-component-classification.md`).
- [x] Pin the reviewed PicoClaw Core `v0.3.1` with Android DNS integration.
- [x] Record provenance, upstream policy, and initial license notices.
- [x] Restore dependencies and run Flutter analysis/tests (28 tests passed).
- [x] Run relevant Core/package validation and build an arm64 foundation APK.
- [x] Record the independent APK metadata and hashes.
- [x] Safety review Git contents for secrets and generated files.
- [x] Initialize, commit, and push the private repository; create `develop`.
- [x] Complete physical-device smoke testing of the independent foundation APK.

## Phase 2 — Milestone B: Product Identity & Branding Foundation

- [x] Create the isolated `feature/pocketclaw-identity` branch from `develop`.
- [x] Record the physical Skill Hub/ClawHub verification and classify the prior
  registry symptom as resolved by the Android DNS fix.
- [x] Complete branding/package and visual-asset audits.
- [x] Implement PocketClaw product-visible identity and the independent
  `com.lord1egypt.pocketclaw` package identity while retaining Core contracts.
- [x] Select the original second PocketClaw mark as the primary direction and
  derive launcher, adaptive, splash, and monochrome notification treatments.
- [x] Establish centralized Material 3 design tokens/theme foundations.
- [x] Run Flutter/Core regressions and build/inspect the arm64 PocketClaw APK.
- [x] Commit/push the feature branch after safety review; do not merge to `develop`.
- [x] Root-cause the physical black-screen regression to the invalid branded
  Android `layer-list` item; replace it with a drawable-backed splash layer.
- [x] Add the launch-background regression test; rebuild and inspect debug and
  replacement release APKs.
- [x] Root-cause the persistent black screen: builds from this workspace were
  missing `lib/arm64-v8a/libdartjni.so` because the `jni` package's arm64 CMake
  configure failure was cached in `~/.pub-cache`, outside `build/`.
- [x] Purge the poisoned `.cxx` cache and rebuild Stage A (`642dc63`) with the
  documented arm64 Gradle command and pinned toolchain; verify the packaged
  `libdartjni.so`, package identity, Core hashes, analyze, and tests.
- [x] Physical test of the Stage A replacement APK: launch, Flutter first
  frame, and no black screen all PASS. Root cause physically confirmed.

## Phase 2 — Stage B: User-facing debranding

- [x] Fail the release build when the arm64 native payload is incomplete.
- [x] Debrand the Flutter UI, Android resources, and the embedded web runtime;
  rebuild the web runtime from source and record its hashes.
- [x] Point fresh installs at `Download/pocketclaw` without any startup migration.
- [x] Run Flutter analyze/tests, Go tests, frontend lint, and rebuild the APK
  through the verified arm64 Gradle path.
- [x] Physical test of the debranded APK
  (`2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`): PASS.
  Install, launch, Flutter first frame, Gateway/Core startup, navigation,
  PocketClaw branding, PocketClaw workspace path, and the QR/access page all
  pass, with no abnormal device slowdown. The black-screen incident is RESOLVED.
- [x] Decide the two open branding items: both closed in the Milestone B final
  cleanup below.

## Phase 2 — Milestone B final cleanup

- [x] MQTT fresh default is `/pocketclaw`; explicitly configured prefixes,
  including the legacy `/picoclaw`, are preserved. Go and frontend changed
  together, with tests for fresh/legacy/custom/normalized values.
- [x] `skills/picoclaw-agent` is no longer seeded into a fresh workspace, is not
  renamed, and existing user copies survive. Tests cover all three.
- [x] Factual Sipeed/LicheeRV Nano/MaixCAM/NanoKVM hardware references retained.
- [x] Re-ran the branding audit with full classification into product branding,
  protocol/compatibility, legal attribution, factual third-party, internal
  implementation, and developer documentation.
- [x] Rebuilt both Core binaries stripped via the documented Makefile targets;
  `PICOCLAW_DNS_SERVER` verified present.
- [x] Flutter analyze/tests, Go tests, frontend lint, canonical arm64 release
  build, and the native payload guard all pass.
- [ ] Physical test of the Milestone B final cleanup APK
  (`ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`).
  Physical approval is the gate for merging Milestone B into `develop`.

## Later (not started)

- [ ] Telegram easy-linking design after validating legitimate API capabilities:
  QR code and/or deep link when valid, with manual bot-token entry retained as
  an advanced/fallback option.
- [ ] Maintainable AI provider catalog and advanced custom-provider flow:
  known provider protocol/endpoints/headers/model discovery plus a manual
  OpenAI-compatible base URL, key, model ID, and safe additional headers.
- [ ] Keep Fetch Available Models optional: discovery failures must still allow
  first-class manual model ID configuration.
