# Development Changelog

## 2026-08-25 — Milestone C device PASS, plus the OpenCode completion

- Physical Android device test of
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b` returned
  PASS. That APK is now the verified reference artifact, superseding
  Milestone B's `ba4f067d...70f70a4b8`. Still not merged: the OpenCode
  completion below lands on the same branch and needs its own device test.
- Added the OpenCode Zen (`https://opencode.ai/zen/v1`) and OpenCode Go
  (`https://opencode.ai/zen/go/v1`) presets, using the official endpoints the
  user verified. Both require an API key, both support Fetch Models against
  `{base}/models`, and both keep the base URL hidden in the normal flow.
- These are mixed-protocol gateways, so they are not modelled as plain
  OpenAI-compatible providers. One base URL and one key front OpenAI Responses,
  OpenAI-compatible chat completions, and Anthropic Messages, and the protocol
  is a property of the selected model. All routing lives in
  `pkg/providers/opencode_routing.go`: `ClassifyOpenCodeModel` matches the
  longest model-ID family prefix and returns both a protocol and whether the
  match was known. No model-name conditional was added anywhere else.
- Routing is family-based rather than an enumerated model list, because
  OpenCode changes its lineup frequently and Fetch Models already returns the
  authoritative live list. `gpt-*`/`o*`/`codex*` route to Responses, `claude-*`
  to Messages, and `kimi-*`/`deepseek-*`/`glm-*`/`qwen-*`/`grok-*` and friends
  to chat completions.
- Built the generic Responses provider the Core was missing
  (`pkg/providers/openai_responses`). The two existing Responses paths were not
  reusable: Azure hardcodes its deployment path, and the Codex provider
  hardcodes the ChatGPT backend plus Codex headers and instructions. The new
  package reuses the shared `openai_responses_common` translation and adds only
  transport.
- Extended `anthropic_messages` with an opt-in `WithBearerAuth()` so the
  OpenCode Messages route sends both `X-API-Key` and `Authorization: Bearer`
  with the one OpenCode account key. It is off by default, so Anthropic's own
  endpoint and the existing `anthropic-messages` and `alibaba-coding-anthropic`
  presets are unchanged. Which form OpenCode's Messages surface actually wants
  is the single assumption only a device can settle.
- Model IDs go out bare. The `opencode-go/` style CLI namespace prefix is
  stripped before the request is built and never persisted into the wire model.
- An unrecognized model stays configurable and is still attempted, falling back
  to chat completions, with a warning naming the model, the fallback protocol,
  and what a 404 would imply. The warning never contains the API key.
- Added a catalog-wide invariant test: every HTTP-API provider that can drive a
  chat model must construct from a plain key-plus-base configuration, with a
  floor of 30 providers exercised so the assertion cannot become vacuous. This
  is the general form of the rule the earlier per-preset test only spot-checked.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; 28 frontend tests;
  new Go tests covering OpenCode routing and catalog metadata, the Responses
  transport, the Messages bearer/base-path behavior, and discovery for both
  endpoints; full Go suites for providers, config, web/backend/api, androiddns,
  mqtt, onboard, and commands all pass; frontend `tsc -b` and `pnpm lint` clean.
- Every new test that can touch a credential asserts the key never appears in
  an error, and the routing warning was inspected in real log output.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` present, launcher still 0 `PicoClaw` /
  0 `Sipeed` / 31 `PocketClaw`:
  `libpicoclaw.so` 37,421,409 `e48e8af0...12e78938`;
  `libpicoclaw-web.so` 24,772,961 `5faaf82c...c383fe2abf`.
- One build note worth keeping: running `:app:packageRelease` without
  `-Ptarget-platform=android-arm64` produced a 50 MB universal APK. Caught on
  the size check and rebuilt on the canonical path. The flag is not optional,
  and APK size is the cheapest signal that it was dropped.
- OpenCode completion APK: 34,129,765 bytes, SHA-256
  `785ccd94cfa351ee2996ac340f9a55e828a0c8f736bec67a3edac906a56058c6`,
  `com.lord1egypt.pocketclaw` 0.1.3 (code 3), label PocketClaw, release guard
  passed. NOT merged and NOT physically verified.
- No Telegram work, no UI changes beyond the two presets flowing through the
  existing picker, no release-pipeline changes, no dependency upgrades, and the
  `libdartjni` guard is untouched.

## 2026-08-25 — Phase 2 Milestone C: Provider Catalog + Easy API-Key Setup

Implementation complete on `feature/provider-catalog`, branched from the
verified `develop` @ `14e6991`. Not merged; physical-device testing is the gate.

- Audited the provider architecture before changing anything and recorded it in
  `docs/PROVIDER_ARCHITECTURE.md`. Two findings shaped the work: all AI provider
  configuration lives in the Core web console, not in Flutter, and the provider
  catalog is already backend-owned by `pkg/providers`, so Milestone C extends it
  rather than building a competing catalog.
- Catalog: added `category` and `documentation_url` to `ModelProviderOption` and
  classified all 42 entries as cloud, local, managed, custom, or speech. Added
  the xAI (`https://api.x.ai/v1`), Together AI (`https://api.together.xyz/v1`),
  Fireworks AI (`https://api.fireworks.ai/inference/v1`), and Custom
  OpenAI-Compatible presets, each also registered in the OpenAI-compatible arm of
  `CreateProviderFromConfig`. A catalog-only addition would have saved cleanly
  and then failed at request time with `unknown protocol`.
- Deferred OpenCode Zen and OpenCode GO: their base URL, auth header, and model
  listing endpoint could not be established accurately, and a guessed preset is
  worse than none. Both work today through Custom OpenAI-Compatible.
- Gemini model discovery now works. It needed a dedicated fetch branch, not just
  the `supports_fetch` flag: the shared path sends `Authorization: Bearer` while
  Gemini authenticates with `X-Goog-Api-Key` and returns
  `{"models":[{"name":"models/<id>"}]}`. The branch selects by base URL, so a
  `/openai` compatibility base keeps using Bearer and the standard shape.
  URL construction stays base-relative; a custom Gemini proxy path is unaffected.
- Broadened model-fetch error classification to distinguish invalid key /
  unauthorized, rate limited, DNS and network failure, provider unavailable, a
  missing listing endpoint, and a malformed response. A test asserts no error
  path echoes the API key.
- UX: the Add Model form became a two-step Add Provider flow — choose a provider
  from a searchable, category-grouped card list, paste an API key, fetch or type
  a model, save. The alias is derived from the model ID instead of being the
  first required field. Base URL, alias, and optional keys moved into Advanced;
  local and custom providers keep a visible base URL with an Android-specific
  hint that `localhost` means the phone. Keyless providers are not asked for a
  key. Saving never requires a successful fetch.
- Backward compatibility: stored provider, model ID, and custom base URL are
  untouched. A base that differs from the preset is now labeled as an override
  in the edit sheet rather than silently presented as the default.
- Removed runtime provider-logo fetching from `cdn.simpleicons.org` and Google's
  favicon service. Those requests disclosed which providers a user had
  configured and broke when offline. Provider marks render locally; verified 0
  occurrences of either host in the built binary.
- Localization: new keys added across all five web-console locales, with English
  and Chinese translated and the rest carrying English fallbacks. URLs and model
  IDs are pinned LTR and the picker uses logical properties.
- API key storage is deliberately unchanged. Transport and UI exposure are
  already sound (masked on GET, preserved on PUT when omitted), but at rest on
  Android keys are plaintext because Core encryption needs
  `PICOCLAW_KEY_PASSPHRASE` and an SSH key that no device has. Android Keystore
  is recorded as its own controlled milestone.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; new Go tests (8
  catalog, 5 model discovery) plus full suites for providers, config,
  web/backend/api, androiddns, mqtt, onboard, commands, and agent all pass;
  22 new frontend tests on a newly added vitest runner; frontend `tsc -b` and
  `pnpm lint` clean.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` present, and the launcher still has
  0 `PicoClaw` / 0 `Sipeed` / 31 `PocketClaw` strings:
  `libpicoclaw.so` 37,421,409 `cbe568af...5556468a`;
  `libpicoclaw-web.so` 24,772,961 `86e53457...0cb0a4cd`.
- Built through the canonical arm64 Gradle path; the release guard printed its
  verification line for `libdartjni.so`, `libpicoclaw.so`, and
  `libpicoclaw-web.so`. `libdartjni.so` is byte-identical to Milestone B.
- Milestone C APK: 34,123,401 bytes, SHA-256
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`,
  `com.lord1egypt.pocketclaw` 0.1.3 (code 3), label PocketClaw. NOT merged and
  NOT physically verified. The Milestone B APK `ba4f067d...70f70a4b8` remains
  the verified reference artifact.
- Telegram QR/deep-link onboarding was deliberately not started; it is the next
  milestone. No release hardening or obfuscation was enabled.

## 2026-08-25 — Phase 2 Milestone B COMPLETE and physically verified

- Physical Android device test of `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`
  returned PASS across every check: install, app launch / first frame, no black
  screen, Gateway/Core lifecycle, navigation, PocketClaw branding, workspace
  path, QR/access page, Skill Hub, provider/model flow, and no abnormal
  slowdown.
- This APK is now the verified reference artifact, superseding
  `2717f32e...0278a1de4`.
- Phase 2 Milestone B is CLOSED: independent product identity, the black-screen
  build-pipeline fix and its permanent release guard, full user-facing
  debranding, and the two branding edge cases.
- Merged `recovery/pocketclaw-clean-debrand` into `develop` with a
  non-fast-forward merge so the recovery history stays intact and auditable.
  `main` is intentionally untouched.
- Tagged the closure point as `phase2-milestone-b`.
- Milestone C is NOT started and requires explicit authorization.

## 2026-08-25 — Phase 2 Milestone B final cleanup

- Closed both remaining branding edge cases. No new features; Milestone C not
  started; no dependency, Flutter, Gradle, AGP, or Kotlin changes.
- MQTT: the Core default topic prefix is now `/pocketclaw`, exposed as
  `mqtt.DefaultTopicPrefix`. `topicPrefix()` substitutes the default only for an
  empty value, so an explicitly configured prefix — including the legacy
  `/picoclaw` — is preserved and broker-side topics are never rewritten. The
  frontend topic preview, placeholder, and the localized hint in all five
  locales were updated together. New tests in
  `pkg/channels/mqtt/topic_prefix_test.go` cover fresh default, explicit legacy
  prefix, custom prefix, and normalization.
- Seeded workspace: `skills/picoclaw-agent` is no longer written into a fresh
  workspace. It joins the existing `AGENTS.md` / `IDENTITY.md` exclusions via a
  named `unseededTemplates` list. It was not renamed, because it documents the
  real upstream CLI. Seeding only writes files, so existing user copies survive;
  three new tests in `cmd/picoclaw/internal/onboard/helpers_test.go` cover the
  exclusion, the surviving user copy, and the path matcher.
- Retained factual third-party hardware references (Sipeed, LicheeRV Nano,
  MaixCAM, NanoKVM) in the `hardware` skill rather than falsifying documentation
  for a zero string count. Full classification in `docs/BRANDING_AUDIT.md`.
- Rebuilt both Core binaries through the documented Makefile targets. Stripped,
  0 debug sections, `PICOCLAW_DNS_SERVER` verified present:
  `libpicoclaw.so` 37,421,409 `eb895f08...40bd9c88`;
  `libpicoclaw-web.so` 24,772,961 `6d282df0...1195a5a3`.
  The built frontend contains 0 `PicoClaw` and 0 `/picoclaw` strings.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; Go tests pass for
  `pkg/channels/mqtt`, `cmd/picoclaw/internal/onboard`, `pkg/androiddns`,
  `web/backend/api`, `pkg/commands`; frontend `pnpm lint` clean.
- Built through the canonical arm64 Gradle path; the release guard passed for
  `libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
- Cleanup APK: 34,119,477 bytes, SHA-256 `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`.
  Not merged; physical-device approval is the merge gate.

## 2026-08-25 — Black-screen incident RESOLVED; Stage B physically verified

- Physical Android device test of `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`
  returned PASS across the board: install, app launch, Flutter first frame,
  Gateway/Core startup, navigation, PocketClaw branding, PocketClaw workspace
  path, and the QR/access page, with no abnormal device slowdown.
- The black-screen incident is CLOSED. The missing `lib/arm64-v8a/libdartjni.so`
  diagnosis and the build-pipeline fix are physically confirmed.
- This APK is now the reference physically verified PocketClaw artifact.
- Retained deliberately and not to be removed: the `packageRelease` guard over
  `libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`, and the canonical
  arm64 release command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`.
- Still open, awaiting a product decision: the MQTT `/picoclaw` topic prefix and
  the bundled `picoclaw-agent`/`hardware` seeded skills — see
  `docs/BRANDING_AUDIT.md`.
- Nothing merged to `develop` or `main`.

## 2026-08-25 — Release build guard and user-facing debranding

- Physical device confirmed the black-screen fix: the Stage A replacement APK
  `d8742534...d7e405` launches and reaches a usable Flutter first frame.
- Added a `packageRelease` guard that fails the build when the APK is missing
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, or `libpicoclaw-web.so`.
  It printed the verification line on this build.
- Debranded every normal user-facing surface. The full classification, with the
  retained legal and internal-compatibility occurrences and the two open product
  decisions, is in `docs/BRANDING_AUDIT.md`.
- Rebuilt the embedded web runtime from source, not by patching the binary.
  `libpicoclaw-web.so` contains 0 `PicoClaw`/`Sipeed` strings and 31
  `PocketClaw` strings.
- Rebuilt the gateway too, because the assistant identity (`/start` reply and
  the seeded `AGENT.md`/`SOUL.md`) lives in `libpicoclaw.so`. Verified the
  Android DNS integration survived: `PICOCLAW_DNS_SERVER` is still present.
- Recorded the Core source diff and the exact rebuild commands in `core/`.
- Android workspace default for fresh installs is now `Download/pocketclaw`;
  existing `Download/picoclaw` data is untouched and nothing migrates at startup.
- Removed three stale generated `lib/l10n/app_localizations*.dart` copies that
  still carried the `PicoClaw UI` title. `l10n.yaml` generates into
  `lib/src/generated/l10n`, so they were dead files.
- Validation: `flutter analyze` clean; 27/27 Flutter tests; Go tests pass for
  `pkg/androiddns`, `web/backend/api`, and `pkg/commands`; frontend `pnpm lint`
  clean.
- Candidate APK: `build/app/outputs/apk/release/app-release.apk`,
  34,119,837 bytes, SHA-256
  `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`.
  Not merged to `develop`; awaiting physical test.

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
