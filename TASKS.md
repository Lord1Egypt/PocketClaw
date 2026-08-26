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
- [x] PHYSICAL DEVICE TEST of APK
  `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`: PASS
  (2026-08-25). This is now the verified reference artifact.

## Phase 2 — Milestone C: OpenCode completion

Authorized 2026-08-25 after the Milestone C device PASS, with official endpoints
supplied by the user.

- [x] Add the OpenCode Zen preset: base `https://opencode.ai/zen/v1`, discovery
  at `https://opencode.ai/zen/v1/models`, API key required, base URL hidden in
  the normal flow.
- [x] Add the OpenCode Go preset: base `https://opencode.ai/zen/go/v1`,
  discovery at `https://opencode.ai/zen/go/v1/models`, same key policy.
- [x] Treat both as mixed-protocol gateways rather than assuming
  `/chat/completions`. All routing lives in `pkg/providers/opencode_routing.go`;
  no model-name conditionals were added anywhere else.
- [x] Build the generic OpenAI Responses provider the Core was missing
  (`pkg/providers/openai_responses`), reusing the existing
  `openai_responses_common` translation layer. The Azure and Codex Responses
  implementations are hardcoded to their own endpoints and were not reusable.
- [x] Route Anthropic Messages models through the existing
  `anthropic_messages` provider, extended with an opt-in `WithBearerAuth()`
  so one OpenCode account key satisfies either header convention. Anthropic's
  own endpoint behavior is unchanged.
- [x] Route chat-completions models through the existing OpenAI-compatible
  HTTP provider.
- [x] Send the bare model ID; strip the `opencode-go/` style CLI namespace
  prefix before the request is built and never persist it into the wire model.
- [x] Handle an unknown model safely and non-silently: it stays configurable
  and attemptable, falls back to chat completions, and logs a warning naming
  the model, the fallback protocol, and what a 404 would mean. The API key is
  never logged.
- [x] Do not hardcode the model list: Fetch Models reads the live list from
  OpenCode, and the routing table is family-based rather than an enumeration.
- [x] Tests: 14 OpenCode routing/catalog tests, 6 Responses transport tests,
  4 Messages bearer/base-path tests, 4 discovery tests, 6 frontend preset
  tests. Plus a catalog-wide invariant that every HTTP chat provider in the
  catalog constructs, so a provider can never be offered while being unusable
  at inference time.
- [x] PHYSICAL DEVICE TEST — PASS on 2026-08-25, carried by the
  source-migration APK `588bbec1...5ff3a90b` rather than
  `785ccd94...56058c6`, which was retired untested. Both OpenCode presets
  passed Fetch Models and a real request/response.
- [ ] Confirm a `claude-*` model on OpenCode. The device report does not say
  which model families were exercised, and the Anthropic Messages route sends
  both `X-API-Key` and a bearer header on an unverified assumption. A 401 there
  while `gpt-*` and `kimi-*` succeed points at the header pair, not the
  routing.

## Self-contained source migration (2026-08-25)

- [x] Vendor the pinned Core source into the repository at `core/src/`
  (1,372 files, 16 MB), with no submodule and no second clone required.
- [x] Audit the vendored tree against the real patched source rather than from
  memory: proved it equals upstream `v0.3.1` + `core/pocketclaw-core-v0.3.1.patch`
  + `pkg/androiddns/`, byte-for-byte.
- [x] Confirm every required Core modification is present — Android
  active-network DNS, `PICOCLAW_DNS_SERVER`, PocketClaw wording, provider
  catalog extensions, model-discovery fixes, OpenCode Zen, OpenCode Go, MQTT
  `/pocketclaw` default, seeded workspace, provider routing.
- [x] Secret/junk review of the vendored tree before committing: no keys,
  tokens, caches, `node_modules`, build outputs, or developer state.
- [x] Exclude `assets/` (upstream README media) and `pkg/seahorse/.omc/`
  (upstream developer tool-state that leaks a home path).
- [x] Add `core/build-android-arm64.sh` as the canonical repo-local Core build.
- [x] Add `-trimpath` to the Android arm64 build lines and assert zero
  developer paths in the shipped binaries (was 2,501 and 1,346; now 0).
- [x] Move the Go build/module caches out of the external checkout to
  `/home/lordegypt/PocketCLaw/.tooling/go/`. Moved, not deleted.
- [x] Remove every build/runtime reference to
  `/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1`; the three
  remaining mentions are documentation or the negative test itself.
- [x] Keep the provenance patch. Rewrote `core/regen-upstream-patch.sh`, fixed
  two defects in it, and verified upstream + patch reproduces `core/src/`.
- [x] Confirm no runtime source or binary fetching: the app executes only the
  packaged binaries; no clone, download, or auto-update path exists.
- [x] Rewrite `core/README.md` as the authoritative Core guide.
- [x] Preserve the release guard and document the `.cxx` recovery procedure.
- [x] Prove the external checkout is unnecessary:
  `core/verify-no-external-source.sh` hid it, the Core built, it was restored.
- [x] Validation: Go 92 ok / 0 fail, vitest 28 passed, `pnpm lint` clean,
  `flutter analyze` clean, `flutter test` 27 passed, arm64 release APK built
  with the release guard passing.
- [x] PHYSICAL DEVICE TEST of APK
  `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b` — PASS on
  2026-08-25, all 18 checks. Install/startup, Flutter first frame, Core
  lifecycle, logs free of developer paths and of PicoClaw branding, provider
  catalog, both OpenCode presets with Fetch Models and real request/response,
  Gemini/provider, Skill Hub, Telegram, workspace, no black screen, no
  slowdown. This is now the verified reference artifact.
- [ ] Merge `feature/provider-catalog` into `develop` with `--no-ff` and tag the
  closure point. Awaiting explicit instruction; `main` needs separate
  instruction.

## Phase 2 — Milestone D: Telegram Managed-Bot Onboarding

- [x] Close Milestone C: merge `feature/provider-catalog` into `develop`
  (`36bc88d`), tag `phase2-milestone-c`, branch
  `feature/telegram-managed-onboarding` from the verified `develop`.
- [x] Verify Telegram's managed-bot capability against official documentation
  before implementing: Bot API 9.6 (2026-04-03), `User.can_manage_bots`,
  `Update.managed_bot` / `ManagedBotUpdated`, `Message.managed_bot_created`,
  `getManagedBotToken`, `replaceManagedBotToken`, and the
  `t.me/newbot/{manager}/{suggested}[?name=]` link form.
- [x] Build PocketClaw's own onboarding service in-repo at
  `services/telegram-onboarding/` — a separate Go module, zero external
  dependencies, no third-party onboarding provider at runtime.
- [x] Pairing API: create, poll, and a separate single-use token collection.
- [x] Pairing security: `crypto/rand` IDs and poll tokens that are independent
  of each other, poll tokens stored hashed and compared in constant time,
  wrong-token and unknown-pairing both answering 404, per-client rate limiting,
  10-minute TTL.
- [x] Child naming: `pocketclaw_<random>_bot` with an 8-character
  cryptographically random segment, Telegram username rules enforced, and
  `hermes`/`picoclaw`/`sipeed` rejected in names and usernames.
- [x] Deep-link and QR generation, with `%20` encoding rather than `+`.
- [x] Manager bot verification at startup; the service refuses to run without
  `can_manage_bots`.
- [x] Manager token redaction from every error path, including transport errors
  that quote the request URL.
- [x] Flutter onboarding screen: Connect, Open Telegram, QR, live progress,
  expiry countdown, Connected with Open Chat, and retry.
- [x] Android lifecycle: polling pauses on background and resumes with an
  immediate check; the pairing survives Telegram taking focus and survives the
  app being killed.
- [x] Auto-configuration reuses the existing `channel_list.telegram` entry,
  merges rather than replaces, sets `allow_from` to the creating Telegram user,
  and restarts Core.
- [x] Manual token entry retained behind "Set up manually", writing the same
  configuration.
- [x] No secret embedded in the APK: the endpoint is a build-time
  `--dart-define` that defaults to empty and must be HTTPS.
- [x] Tests: 62 service tests across 6 Go packages; 50 new Flutter tests
  (77 total).
- [x] Regression: Core 92 packages ok, `pnpm lint` clean, `flutter analyze`
  clean, release APK built with the build guard passing.
- [x] Manager bot created and Bot Management Mode enabled by the project owner;
  the managed-bot deep link opened successfully against `@PocketClawSetupBot`.
- [x] Audit pairing storage for serverless: found process-local state, replaced
  it with a Redis-compatible store using atomic `SET NX` and `GETDEL`.
- [x] Replace `getUpdates` long-polling with an authenticated Telegram webhook.
- [x] Extract the service to the public repository
  `Lord1Egypt/PocketClaw-Telegram-Setup` with a Deploy to Vercel button,
  operator status page, `/privacy`, README, PRIVACY.md, SECURITY.md, LICENSE.
- [x] Verify no real credential exists in the public repository.
- [x] Deploy the service to Vercel with an Upstash Redis store and the four
  secrets. Live at `https://pocketclaw-telegram-setup-bot-83ai.vercel.app`.
- [x] Live `getMe` → `can_manage_bots == true` from the deployed service —
  **PASS on 2026-08-26**, the last link neither the BotFather UI nor a manual
  deep link could establish.
- [x] Webhook registered; storage connected; a test pairing created and read
  back through the live service.
- [ ] Revoke the manager bot token again — it was pasted into a chat log after
  deployment. The service keeps working; rotate it in BotFather and update the
  Vercel environment variable.
- [x] Rebuild the app with
  `--dart-define=POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app`
  through the canonical Gradle path (`-Pdart-defines`, base64 `KEY=VALUE`).
- [x] Fix the UI integration defect the first device test exposed: managed
  onboarding was wired only to the native Settings tab, while Channels →
  Telegram is rendered by the Core web console and still opened the raw Bot
  Token form. The console now renders the entry point and asks the Flutter host
  to run the existing Dart flow over the `PocketClawHost` bridge; pairing was
  not reimplemented in TypeScript.
- [x] Add regression tests that enter through the real route
  (`ChannelConfigPage channelName="telegram"`), covering managed-onboarding
  primary, the no-endpoint and no-host fallbacks, the connected summary, and
  the legacy form behind both Advanced entries. Confirmed to fail against the
  pre-fix wiring before being accepted.
- [x] **Physical end-to-end device test: PASS on 2026-08-26.** Full Channels →
  Telegram onboarding UI, bot creation through Telegram, automatic pairing
  detection and token delivery, automatic owner and channel configuration, the
  connected state, Open Chat, and a real Telegram → Core → AI provider →
  Telegram message round trip with context persisting across consecutive
  messages. Verified APK
  `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`.
- [x] Mark the rebuilt Core pair physically verified with that APK:
  `libpicoclaw.so` `33f8b4ef...3470e98a`, `libpicoclaw-web.so`
  `5400cb02...2dbcb3bd`.
- [x] Close Milestone D: merge `feature/telegram-managed-onboarding` into
  `develop` with `--no-ff` and tag `phase2-milestone-d`. `main` untouched.
- [ ] Localize the Telegram onboarding strings. They live in
  `TelegramOnboardingStrings` and are English in all twelve locales, because
  shipping machine-guessed translations into the `.arb` files would put
  unverified text in front of users.
- [ ] Confirm a `claude-*` model on OpenCode, carried over from Milestone C.

## Deferred out of Milestone C, deliberately

- [ ] API key storage on Android is plaintext in the workspace config, because
  Core encryption needs `PICOCLAW_KEY_PASSPHRASE` and an SSH key that no
  Android device has. Android Keystore or an equivalent needs its own
  controlled milestone: it touches config loading, the secret resolver, and
  migration of existing files. Recorded in `docs/PROVIDER_ARCHITECTURE.md` §6.

## Later (not started)

- [ ] Telegram easy-linking design after validating legitimate API capabilities:
  QR code and/or deep link when valid, with manual bot-token entry retained as
  an advanced/fallback option. This is the next milestone and was deliberately
  kept out of Milestone C.

## CODEX SOL HANDOFF — PRE-RELEASE FIX

- [x] Create and push rollback branch/tag at exact pre-Codex commit `e5b88ff`.
- [x] Make native Settings Telegram a neutral shortcut to Core console
  `/channels/telegram`; assert card tap cannot invoke pairing.
- [x] Route managed/manual credential writes through Core's authoritative
  config/security persistence boundary without exposing a read/token API.
- [x] Bound Telegram HTTP operations; propagate edit/send errors; make final
  delivery synchronous; correlate placeholders; queue same-session Telegram
  requests as independent FIFO lifecycles.
- [x] Add deterministic empty-response, idle, sequential/close-arrival,
  edit-fallback, send-failure, and placeholder-cleanup tests.
- [x] Audit tool success/failure/timeout/empty-output/termination paths; no
  unbounded `gh`-specific defect found.
- [x] Determine intended fresh skill baseline: seven; preserve but do not newly
  seed `picoclaw-agent`; repair missing seeds without overwriting user files;
  prove `gh` import is additive.
- [x] Preserve basename callers and exactly-once queue/drain logs; neutralize
  stale-PID text; sanitize terminal controls once for Logs and Export while
  preserving Arabic/emoji/Unicode.
- [x] Regenerate Core patch and record intentional divergence.
- [x] Pass Flutter, frontend, required Go, metadata, guard, path, branding, and
  secret validations.
- [x] Build one successful ARM64 candidate APK:
  `f663d25a...71c621d` (34,239,649 bytes).
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** validate a lone `تسلم` completes
  without any later inbound update; repeat idle/sequential/close-message flows.
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** validate foreground, background, and
  locked-screen completion plus typing/placeholder/streaming combinations;
  permanent Thinking placeholders must be zero.
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** validate native Telegram opens only
  the canonical Core page, explicit reconnect is safe, failed/cancelled pairing
  preserves the old bot, restart persists, and live bot AI replies work.
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** validate fresh/existing seven-skill
  baseline, additive `gh` import, and Core/app restart discovery.
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** validate basename exactly-once logs,
  no upstream branding/ANSI/control boxes, readable Arabic/emoji, and clean
  exported logs.
- [ ] Merge/release only after every physical item passes. `main` needs separate
  authorization; do not create `v0.2.0-rc1` yet.
- [ ] Security hygiene: confirm manager bot token rotation externally. Never
  retrieve or record the token.
- [ ] FINAL RELEASE HARDENING: controlled Dart generated-source URI strategy.
- [ ] Future milestone only: Background & Battery page.
- [ ] Future milestone only: local Runtime / Statistics bottom tab.
