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
- [x] Physical test of the Milestone B final cleanup APK
  (`ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`): PASS.
  Install, launch/first frame, Gateway/Core lifecycle, navigation, branding,
  workspace path, QR/access page, Skill Hub, and the provider/model flow all
  pass, with no black screen and no abnormal slowdown.
- [x] Phase 2 Milestone B COMPLETE. Merged into `develop` with a non-fast-forward
  merge; `main` intentionally untouched.

## Phase 2 — Milestone C: Provider Catalog + Easy API-Key Setup

Authorized 2026-08-25. Branch `feature/provider-catalog` from `develop` @ `14e6991`.

- [x] Audit the existing provider architecture — config/model schema, Core
  provider abstraction, web provider pages, model discovery, custom
  OpenAI-compatible handling, auth behavior, and API key storage — and record
  it in `docs/PROVIDER_ARCHITECTURE.md`.
- [x] Establish that provider configuration lives in the Core web console, not
  in Flutter, and that the catalog is already backend-owned by `pkg/providers`.
  Extend that catalog rather than creating a competing one.
- [x] Add `category` and `documentation_url` to `ModelProviderOption` and
  classify every one of the 42 catalog entries.
- [x] Add the xAI, Together AI, Fireworks AI, and Custom OpenAI-Compatible
  presets, and register all four in the `CreateProviderFromConfig` protocol
  switch. A catalog entry alone fails at runtime with `unknown protocol`.
- [x] Classify every requested candidate as SUPPORTED NOW, OPENAI-COMPATIBLE,
  REQUIRES CORE ADAPTER, or DEFERRED. OpenCode Zen and OpenCode GO are DEFERRED
  because their endpoint and auth could not be established accurately.
- [x] Enable Gemini model discovery with a dedicated fetch branch:
  `X-Goog-Api-Key` plus `models/` prefix stripping for the native base, Bearer
  for a `/openai` compatibility base. Base-relative path handling unchanged.
- [x] Broaden fetch error classification: invalid key/unauthorized, rate
  limited, DNS/network failure, provider unavailable, missing listing endpoint,
  and malformed response. No error path echoes the API key.
- [x] Replace the Add Model form with a two-step provider-first flow: choose
  provider, paste API key, fetch or type a model, save. The alias is derived
  from the model ID; base URL, alias, and optional keys move to Advanced.
- [x] Keep the base URL visible in the normal flow for local and custom
  providers, and do not ask keyless providers for an API key.
- [x] Keep manual model entry always available; saving never requires a
  successful fetch.
- [x] Keep Custom OpenAI-Compatible first-class, with no default base URL so an
  empty endpoint is a clear error rather than a silent fall back to OpenAI.
- [x] Preserve existing configurations: stored provider, custom base URL, and
  model IDs are untouched, and a base that differs from the preset is surfaced
  as an override in the edit sheet rather than reverted.
- [x] Stop downloading provider logos from `cdn.simpleicons.org` and Google's
  favicon service at runtime; render local text marks instead.
- [x] Add localization keys for every new user-facing string across all five
  web-console locales; keep URLs and model IDs LTR-readable.
- [x] Tests: 22 frontend tests on a new vitest runner, 8 Go provider-catalog
  tests, 5 Go model-discovery tests. `flutter analyze` clean; 27 Flutter tests;
  Go suites for providers, config, api, androiddns, mqtt, onboard, commands,
  and agent all pass; frontend `tsc -b` and `pnpm lint` clean.
- [x] Rebuild both Core binaries through the documented Makefile targets and
  build the Milestone C APK through the canonical arm64 Gradle path with the
  release guard passing.
- [ ] PHYSICAL DEVICE TEST of APK
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`.
  Not merged to `develop` until this passes.

## Deferred out of Milestone C, deliberately

- [ ] API key storage on Android is plaintext in the workspace config, because
  Core encryption needs `PICOCLAW_KEY_PASSPHRASE` and an SSH key that no
  Android device has. Android Keystore or an equivalent needs its own
  controlled milestone: it touches config loading, the secret resolver, and
  migration of existing files. Recorded in `docs/PROVIDER_ARCHITECTURE.md` §6.
- [ ] OpenCode Zen and OpenCode GO presets, pending a verified base URL,
  authentication header, and model-listing endpoint.

## Later (not started)

- [ ] Telegram easy-linking design after validating legitimate API capabilities:
  QR code and/or deep link when valid, with manual bot-token entry retained as
  an advanced/fallback option. This is the next milestone and was deliberately
  kept out of Milestone C.
