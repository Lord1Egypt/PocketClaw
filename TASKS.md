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
- [ ] Milestone B physical test BLOCKED — retest the replacement APK for Flutter
  first frame, usable UI, Core lifecycle, and absence of a service/CPU loop.

## Later (not started)

- [ ] Telegram easy-linking design after validating legitimate API capabilities:
  QR code and/or deep link when valid, with manual bot-token entry retained as
  an advanced/fallback option.
- [ ] Maintainable AI provider catalog and advanced custom-provider flow:
  known provider protocol/endpoints/headers/model discovery plus a manual
  OpenAI-compatible base URL, key, model ID, and safe additional headers.
- [ ] Keep Fetch Available Models optional: discovery failures must still allow
  first-class manual model ID configuration.
