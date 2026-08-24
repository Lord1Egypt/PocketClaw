# Development Changelog

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
