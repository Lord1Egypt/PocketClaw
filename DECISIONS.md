# PocketClaw Decisions

## Independent repository, not a permanent FUI fork

- Date: 2026-08-24
- Decision: PocketClaw keeps an independent Git history and uses PicoClaw FUI
  as a manually reviewed reference.
- Consequence: No `git merge` maintenance path from FUI; upstream changes must
  be classified as security, bug fix, Android runtime, Core compatibility,
  Telegram, provider, MCP, performance, useful feature, UI-only, or not
  relevant before adoption.

## Preserve the physical-device-proven runtime behavior

- Date: 2026-08-24
- Decision: Keep the Android active-network DNS bridge, pinned Core `v0.3.1`,
  and optional feedback defaults in the independent foundation.
- Consequence: Never revert to `[::1]:53`, hardcode public DNS, include
  Firebase credentials, or make analytics mandatory.

## Delay package identity and product branding

- Date: 2026-08-24
- Decision: Keep the inherited technical package identity until the independent
  foundation builds and is smoke-tested.
- Consequence: Product branding and package-ID changes are a later isolated
  milestone.

## No final license for PocketClaw-authored code yet

- Date: 2026-08-24
- Decision: Preserve upstream MIT notices and defer choosing a license for new
  PocketClaw code until product-owner approval.

## Bound the local Gradle JVM for reproducible foundation builds

- Date: 2026-08-24
- Decision: Set the inherited Gradle JVM maximum heap to 4 GiB and metaspace to
  2 GiB.
- Reason: The inherited 8 GiB/4 GiB settings exceed the available headroom in
  the supported local build environment and can terminate the daemon before
  APK assembly.
- Consequence: This changes only build-process memory limits; Android runtime
  behavior and the pinned Core binaries are unaffected.

## Product identity is PocketClaw; PicoClaw remains the credited engine

- Date: 2026-08-24
- Decision: Use PocketClaw for all product-visible identity and
  `com.lord1egypt.pocketclaw` for the independent Android/Dart integration
  identity. Retain PicoClaw Core names, binaries, environment variables,
  protocol identifiers, and attribution where they are integration contracts.
- Consequence: The PocketClaw and reference APKs can coexist; Android private
  app data is intentionally not migrated across package identities. The
  compatible `Downloads/picoclaw` external workspace remains until a separately
  designed migration exists.

## Skill Hub regression is DNS-resolved, not a separate product defect

- Date: 2026-08-24
- Decision: Keep the existing Skill Hub/ClawHub implementation unchanged.
- Evidence: On a physical device after the active-network DNS fix, Skill Hub
  opened and `Crypto` search returned 20 results with metadata, URLs, and
  visible install actions.
- Consequence: Treat Skill Hub as regression-sensitive; do not install skills
  automatically and do not change its registry configuration without new
  source/runtime evidence.

## Use the selected original PocketClaw mark consistently

- Date: 2026-08-24
- Decision: The second generated PocketClaw mark is the primary visual
  direction. Launcher, adaptive, splash, and monochrome notification variants
  must simplify that identity rather than introduce a cartoon lobster/mascot.
- Consequence: Preserve a restrained professional Android/developer-tool
  appearance and keep future visual work non-derivative of PicoClaw artwork.

## Android splash layers must contain drawables

- Date: 2026-08-24
- Decision: Keep the branded PocketClaw launch mark, but express the splash
  background as an explicit Android color drawable reference rather than
  `android:color` on a `layer-list` item.
- Evidence: `LayerDrawableItem` accepts `android:drawable`; the Milestone B
  resource supplied neither a drawable attribute nor a child drawable, blocking
  launch-theme inflation before Flutter's first frame on the physical device.
- Consequence: `launch_background.xml` and its v21 variant reference
  `@color/pocketclaw_splash_background`; a Flutter regression test guards both
  files. This is not a package-ID migration or Core integration change.

## The `jni` CMake configure cache lives outside the project and can poison builds

- Date: 2026-08-25
- Decision: Treat `~/.pub-cache/hosted/pub.dev/jni-<version>/android/.cxx/` as a
  build input that must be purged when an Android native library goes missing
  from a release APK. Verify `lib/arm64-v8a/libdartjni.so` is packaged before
  releasing any PocketClaw APK.
- Evidence: On 2026-08-24 02:52 the CMake configure of the `jni` package for
  `arm64-v8a` failed inside this workspace with a transient filesystem error
  (`unable to open output file '...CMakeCCompilerABI.c.o': No such file or
  directory`). CMake concluded the C compiler was unusable for that ABI and
  cached a configure result with an empty target list
  (`"libraries": {}`, `"cFileExtensions": []`). Every later build reported the
  JSON "up-to-date", built no arm64 target, and shipped an APK with no
  `lib/arm64-v8a/libdartjni.so`.
- Consequence: `com.github.dart_lang.jni.JniPlugin` calls
  `System.loadLibrary("dartjni")` from a **static initializer**, and
  `GeneratedPluginRegistrant.registerWith` only catches `Exception`. The
  resulting `ExceptionInInitializerError`/`UnsatisfiedLinkError` is an `Error`,
  so it escapes the registrant during `FlutterActivity.onCreate` and the
  Flutter view never attaches — a permanent black screen. `path_provider_android`
  pulls in `jni`/`jni_flutter`, so this affects every build of this app.
- Why it survived cleanups: the poisoned cache is in `~/.pub-cache`, not in the
  project `build/` tree, so deleting build intermediates never cleared it.

## Fresh installs use `Download/pocketclaw`; old data is left alone

- Date: 2026-08-25
- Decision: The Android workspace directory for fresh PocketClaw installs is
  `Download/pocketclaw`, including the no-permission fallback and the
  log-export directory. This supersedes the earlier decision to keep the
  compatible `Downloads/picoclaw` path.
- Consequence: Any existing `Download/picoclaw` tree is left untouched and no
  migration runs at startup. A migration, if it is ever wanted, is a separate
  designed feature and must not block the first frame.

## Debrand the embedded web runtime at source, never by binary patching

- Date: 2026-08-25
- Decision: PocketClaw wording in the embedded web console comes from edits to
  the Core web frontend/backend source, rebuilt through the Core Makefile
  targets. The compiled `.so` files are never patched.
- Consequence: `core/pocketclaw-core-v0.3.1.patch` and `core/README.md` carry
  the exact source diff and build commands. Build only via
  `make build-launcher-android-arm64`; calling `make -C web build-android-arm64`
  directly drops the root `LDFLAGS` and yields an unstripped binary, which is
  what produced the earlier oversized `libpicoclaw-web.so`.
- Consequence: use `pnpm lint` on the frontend, not `pnpm check` — the latter
  runs `prettier --write` across the whole tree and rewrites unrelated files.

## The release build fails closed on an incomplete arm64 payload

- Date: 2026-08-25
- Decision: `packageRelease` verifies that the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`,
  and fails the build otherwise.
- Reason: the black-screen regression shipped for days because a missing
  `libdartjni.so` is silent at build time and fatal at launch.
- Consequence: `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`
  is the canonical PocketClaw release path. Do not use a universal
  `flutter build apk --release` for releases.

## The black-screen incident is closed on physical evidence

- Date: 2026-08-25
- Decision: Treat the black-screen incident as RESOLVED and treat APK
  `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4` as the
  reference physically verified PocketClaw artifact.
- Evidence: a physical Android device passed install, app launch, Flutter first
  frame, Gateway/Core startup, navigation, PocketClaw branding, PocketClaw
  workspace path, and the QR/access page, with no abnormal device slowdown.
  This confirms the missing `lib/arm64-v8a/libdartjni.so` diagnosis and the
  build-pipeline fix.
- Consequence: the release guard and the canonical arm64 Gradle command are
  permanent parts of the release process, not temporary debugging aids. Removing
  either reopens the failure mode that caused this incident. Any future
  regression should be compared against this artifact before new hypotheses are
  formed.

## MQTT defaults to `/pocketclaw`, but never rewrites a configured prefix

- Date: 2026-08-25
- Decision: `mqtt.DefaultTopicPrefix` is `/pocketclaw`. `topicPrefix()`
  substitutes it only when the configured value is empty, so any explicitly
  configured prefix — including the legacy `/picoclaw` — is returned unchanged.
- Reason: the topic prefix is a broker-side contract shared with every other
  subscriber. A fresh PocketClaw install should not advertise the upstream name,
  and an existing deployment must not have its topics moved underneath it.
- Consequence: the Go default and the frontend preview/placeholder/hint must be
  changed together or the preview will misreport the topic an unconfigured
  channel publishes on. Because `topic_prefix` is `omitempty`, a config that
  relied on the *implicit* old default is indistinguishable from a fresh one and
  will move to `/pocketclaw`; pinning it would need a config migration, which is
  deliberately out of scope. Set `topic_prefix` explicitly to keep the old topic.

## The upstream agent skill is not seeded, and not renamed

- Date: 2026-08-25
- Decision: `skills/picoclaw-agent` is excluded from a freshly seeded workspace
  through the `unseededTemplates` list in
  `cmd/picoclaw/internal/onboard/helpers.go`.
- Reason: the skill documents the real upstream `picoclaw` CLI and repository
  internals. Rebranding its text would make its instructions technically wrong,
  so the choice was to seed it or not, and PocketClaw does not.
- Consequence: seeding only ever writes files, so a user who already has that
  skill keeps it, and it can still be installed later from a registry or by
  hand. The general skill loading/install mechanism is untouched.

## Factual third-party hardware references are kept accurate

- Date: 2026-08-25
- Decision: Sipeed, LicheeRV Nano, MaixCAM, and NanoKVM stay in the `hardware`
  skill, as do the real `picoclaw` binary name and `~/.picoclaw/workspace` path
  in shell examples.
- Reason: these are factual references to third-party hardware and to the real
  executable, not PocketClaw product branding.
- Consequence: do not rewrite documentation to reach a superficial zero string
  count. The branding audit classifies occurrences rather than merely counting
  them.

## Milestone B closes on physical evidence and merges non-destructively

- Date: 2026-08-25
- Decision: Treat APK
  `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8` as the
  verified reference artifact and Phase 2 Milestone B as COMPLETE.
- Evidence: a physical Android device passed install, launch/first frame,
  Gateway/Core lifecycle, navigation, branding, workspace path, QR/access page,
  Skill Hub, and the provider/model flow, with no black screen and no abnormal
  slowdown.
- Decision: merge `recovery/pocketclaw-clean-debrand` into `develop` with
  `--no-ff`, keeping the recovery branch and its history intact. `main` is not
  updated yet, and no branch is deleted or rewritten.
- Consequence: `phase2-milestone-b` tags the closure point, so any later
  regression can be bisected against a known-good, physically verified state.
  Milestone C stays unstarted until the user authorizes it.

## The provider catalog stays backend-owned; PocketClaw extends it

- Date: 2026-08-25
- Decision: Milestone C extends `pkg/providers.modelProviderOptionsByName` and
  the `provider_options` payload rather than creating a PocketClaw-side catalog
  in the UI or in Flutter.
- Evidence: the Core already exposes a 42-entry catalog through
  `ModelProviderOptions()`, and `web/frontend/.../provider-registry.ts` is a
  pure projection of that payload with no provider URLs of its own.
- Consequence: there is exactly one place to add or correct a provider. A
  second catalog in the UI would drift from the one the Core actually dispatches
  on. The full audit is in `docs/PROVIDER_ARCHITECTURE.md`.

## A provider preset is only real when the protocol switch knows it

- Date: 2026-08-25
- Decision: every preset added to the catalog is added in the same change to
  the OpenAI-compatible arm of `CreateProviderFromConfig`, and a Go test asserts
  that each PocketClaw preset actually constructs a provider.
- Reason: `CreateProviderFromConfig` ends in `default: unknown protocol %q`. A
  catalog-only addition looks correct in the UI, saves successfully, and then
  fails on the first message — the worst possible failure shape for a preset
  whose entire purpose is to remove guesswork.
- Consequence: `TestPocketClawPresetsResolveToAProvider` fails the build rather
  than shipping a preset that cannot run.

## Only presets that can be specified accurately are shipped

- Date: 2026-08-25
- Status: SUPERSEDED for OpenCode on 2026-08-25 — the user supplied the
  verified official endpoints, so both now ship. See "OpenCode Zen and Go are
  mixed-protocol gateways" below. The principle itself stands.
- Decision: xAI, Together AI, and Fireworks AI ship as presets. OpenCode Zen and
  OpenCode GO do not.
- Reason: the first three are Bearer-auth OpenAI-shaped APIs whose base URLs are
  well established. For the two OpenCode entries, the base URL, authentication
  header, and model-listing endpoint could not be established from the pinned
  Core or anything else in this workspace.
- Consequence: a guessed base URL would produce a preset that fails with a
  confusing error, which is worse for the user than not offering it. Both remain
  configurable today through Custom OpenAI-Compatible, and can be promoted to
  presets once their configuration is verified.

## Custom OpenAI-Compatible has no default base URL

- Date: 2026-08-25
- Decision: the `custom-openai` catalog entry deliberately ships with an empty
  `DefaultAPIBase`, and its base URL is a required field in the normal flow.
- Reason: falling back to a default would silently point a user's "custom"
  endpoint at OpenAI and produce an authentication error that describes the
  wrong service.
- Consequence: an empty base surfaces as an explicit configuration error. This
  is the supported path for self-hosted, VPS, local gateway, and unlisted
  OpenAI-compatible providers, and it is why the previous workaround — provider
  `openai` plus a custom `api_base` — is no longer the only option. Existing
  entries configured that older way keep working untouched.

## Gemini discovery needs its own fetch branch, not a flag

- Date: 2026-08-25
- Decision: `gemini` gains `SupportsFetch`, together with a dedicated branch in
  `fetchUpstreamModels`.
- Evidence: the shared fetch path sends `Authorization: Bearer`, but the Gemini
  provider authenticates with `X-Goog-Api-Key`
  (`pkg/providers/httpapi/gemini_provider.go`), and Google's native listing
  returns `{"models":[{"name":"models/<id>"}]}` rather than the OpenAI shape.
- Consequence: the branch keys off the base URL. A base ending in `/openai` is
  treated as Google's OpenAI-compatibility surface and uses Bearer with the
  standard response shape; anything else uses `X-Goog-Api-Key` and strips the
  `models/` prefix. URL construction stays base-relative, so a custom Gemini
  proxy path keeps working. Merely flipping the flag would have produced 401s.

## The alias is derived, not demanded

- Date: 2026-08-25
- Decision: `model_name` is no longer a required field in the normal Add
  Provider flow. When left blank it is derived from the model ID, with a numeric
  suffix on collision, and it stays editable under Advanced.
- Reason: the alias is a local label with no protocol meaning, but it was the
  first required field in the old form — asking the user to invent a name before
  they had even chosen a model.
- Consequence: the Core contract is unchanged; `model_name` is still required
  and unique at the API layer. Only who supplies it has changed.

## Provider marks are rendered locally, never fetched

- Date: 2026-08-25
- Decision: `provider-icon.tsx` renders a local text mark. The runtime requests
  to `cdn.simpleicons.org` and `https://www.google.com/s2/favicons` are removed.
- Reason: on a mobile device those requests disclose to two third parties which
  AI providers a user has configured, and they leave a broken mark whenever the
  device is offline or behind a restrictive network.
- Consequence: no provider logo assets are bundled and no licensing question is
  raised. Verified absent from the built binary: 0 occurrences of either host.

## Android API key storage is a separate milestone

- Date: 2026-08-25
- Decision: Milestone C does not change how API keys are stored.
- Evidence: `SaveConfig` encrypts keys to `enc://` only when
  `PICOCLAW_KEY_PASSPHRASE` and an SSH key are present, neither of which exists
  on an Android device, so keys are written as plaintext in the workspace
  config. Transport and UI exposure are already sound: `GET /api/models` returns
  `maskAPIKey(...)`, and `PUT` preserves the stored key when `api_key` is empty.
- Consequence: moving to Android Keystore touches config loading, the secret
  resolver, and migration of existing config files. Folding that into a UX
  milestone would put the verified Core config path at risk. It is recorded in
  `docs/PROVIDER_ARCHITECTURE.md` §6 and deferred to its own controlled
  milestone. Milestone C must not make the posture worse, and does not.

## OpenCode Zen and Go are mixed-protocol gateways, routed per model

- Date: 2026-08-25
- Decision: `opencode_zen` and `opencode_go` ship as presets, and the request
  protocol is resolved per model rather than fixed per provider. Every routing
  decision lives in `pkg/providers/opencode_routing.go`.
- Evidence: both gateways expose OpenAI Responses (`/responses`),
  OpenAI-compatible chat (`/chat/completions`), and Anthropic Messages
  (`/messages`) behind one base URL and one API key. Modelling either as a
  plain OpenAI-compatible provider would produce a preset that saves cleanly
  and then fails on the first inference — precisely the failure the catalog
  work exists to prevent.
- Consequence: the catalog entry carries only the base URL and key policy. The
  factory arm calls `ClassifyOpenCodeModel`, which matches the longest model-ID
  family prefix and returns both a protocol and whether the match was known.
  Model-name conditionals appear in that one file and nowhere else.
- Consequence: routing is family-based (`gpt-`/`codex` to Responses, `claude`
  to Messages, `kimi`/`deepseek`/`glm`/etc. to chat completions) rather than an
  enumerated model list. OpenCode adds and retires models frequently, and Fetch
  Models already returns the authoritative live list, so a pinned enumeration
  would be wrong within weeks.

## An unknown OpenCode model falls back, but never silently

- Date: 2026-08-25
- Decision: a model that matches no known family is still configurable and
  still attempted, using chat completions as the fallback, and the attempt logs
  a warning naming the model, the chosen protocol, and what a 404 would imply.
- Reason: erroring out would strand every future OpenCode model until PocketClaw
  ships an update, which is worse than a documented best guess. Routing it
  quietly would violate the rule against sending a knowingly-unverified request
  with no trace.
- Consequence: `ClassifyOpenCodeModel` returns `known bool` alongside the
  protocol, so no caller can mistake a fallback for a confirmed route. The
  warning never contains the API key.

## The Core needed a generic Responses provider, and now has one

- Date: 2026-08-25
- Decision: added `pkg/providers/openai_responses`, a Responses-over-HTTP
  provider driven by a caller-supplied base URL and bearer key.
- Evidence: the Core already spoke Responses twice, but neither was reusable —
  `azure` hardcodes Azure's deployment path, and `oauth/codex_provider`
  hardcodes `https://chatgpt.com/backend-api/codex` plus Codex-specific
  headers and instructions.
- Consequence: the new package reuses the shared
  `openai_responses_common` translation and adds only transport, so request and
  response handling stay in one place. It is the first Responses path in
  PocketClaw usable against a third-party gateway.

## The OpenCode Messages route sends both authentication headers

- Date: 2026-08-25
- Decision: for OpenCode's Anthropic Messages surface, the provider sends both
  `X-API-Key` and `Authorization: Bearer` with the same OpenCode key, via a new
  opt-in `anthropicmessages.WithBearerAuth()`.
- Reason: OpenCode issues one account key for every surface, and its OpenAI
  surfaces use the bearer form, but which form the Messages surface expects
  could not be established from anything available in this workspace. Sending
  both satisfies either convention. The option is off by default, so
  `api.anthropic.com` and the existing `anthropic-messages` and
  `alibaba-coding-anthropic` presets are byte-for-byte unchanged.
- Consequence: this is the one part of the OpenCode work that a green test
  suite cannot settle. If a `claude-*` model returns 401 on a device while
  `gpt-*` and `kimi-*` succeed, the answer is this header pair, not the routing.
- Consequence: `common.NormalizeBaseURL` strips and re-appends `/v1`, which is
  a no-op for both OpenCode bases. That round trip is asserted directly, since
  a base of `.../zen/go/v1` silently becoming `.../zen/v1` would be a
  hard-to-spot production failure.

## A catalog entry must never outlive its ability to run

- Date: 2026-08-25
- Decision: `TestEveryHTTPChatProviderInCatalogIsConstructible` asserts that
  every HTTP-API provider in the catalog that can drive a chat model
  constructs from a plain key-plus-base configuration, and fails if fewer than
  30 providers were exercised so the assertion cannot quietly become vacuous.
- Reason: the per-preset test only covers presets someone remembered to list.
  This one covers the catalog itself, which is the actual contract: anything
  offered in the picker has to work at inference time.

## The upstream adoption timeline is recorded, not remembered

- Date: 2026-08-25
- Decision: PocketClaw permanently records which upstream release and commit it
  started from, when it adopted them, and what upstream state was known at each
  review. `UPSTREAM_BASELINE.md` holds the immutable adoption record
  (adopted 2026-08-24; Core `v0.3.1` at
  `2cf030d2fd3b871d7ec17e3be34c24688aac76da`; historical FUI reference
  `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`, tag `v0.1.4`).
  `UPSTREAM_TRACKING.md` holds a moving Checkpoint plus an append-only Upstream
  Review History. Existing review entries are never edited or deleted.
- Reason: without a durable checkpoint, every future "what is new in PicoClaw?"
  restarts from v0.3.1 and re-reviews ranges that were already classified. The
  checkpoint makes each review begin exactly where the last one ended.
- Consequence: the review workflow is fixed — read the reviewed-through commit,
  fetch upstream, compare only what came after it, classify, selectively adapt,
  then update the checkpoint and append a new review entry.

## Upstream version claims are verified against upstream, or marked NOT VERIFIED

- Date: 2026-08-25
- Decision: the "latest upstream release known at review time" field is only
  ever filled from real upstream metadata. If it cannot be checked during a
  review, it is recorded as `NOT VERIFIED` rather than guessed.
- Evidence: for Review 1 it was verified live on 2026-08-25 against the GitHub
  API. Core `releases/latest` and the full tag list both name `v0.3.1`
  (published 2026-07-03) as the newest version tag, and its commit is the
  PocketClaw baseline commit; the rolling `nightly` release tag points at that
  same commit. FUI `main` HEAD equals the baseline commit and equals tag
  `v0.1.4`. Core `main` has moved to
  `bbf6893ca7afad27f1d00a0f5a45982a549c6ed6` (2026-08-19).
- Consequence: PocketClaw is level with the newest upstream release and 19
  first-parent commits behind Core `main`. That range is bounded and recorded
  as unreviewed, so the gap is a known quantity rather than an open question.
  The raw `2cf030d2..main` count of 2583 is not that gap — Core `main` has
  merged an unrelated root history — so first-parent count is the figure
  review planning uses.

## The PocketClaw repository is the canonical source-of-truth

- Date: 2026-08-25
- Decision: the Core source is vendored into this repository at `core/src/` and
  is the only source the build reads. The external checkout at
  `/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1` survives as a
  historical reference for upstream review and is no longer a build dependency.
- Reason: a build that reaches outside the repository is not reproducible by
  anyone but the machine that has that directory. A `git clone` of PocketClaw
  now contains every line of application and runtime source needed to build the
  APK. External toolchains — Go, Flutter, JDK, Android SDK/NDK, Node/pnpm — stay
  external, because they are tools, not source.
- Deliberately not a submodule: a submodule would reintroduce "clone that other
  repository first" as a precondition, which is the exact failure being fixed.
- Evidence: `core/verify-no-external-source.sh` renames the external checkout
  away, runs the full Core build, and restores it. It passed on 2026-08-25.
- Consequence: adapting an upstream change now means editing `core/src/` and
  regenerating the provenance patch. Upstream is an update source that is
  fetched for review only.

## Vendoring moves the source, not the authorship

- Date: 2026-08-25
- Decision: `core/src/` keeps upstream copyright headers untouched, keeps
  `LICENSE` in place, and `core/pocketclaw-core-v0.3.1.patch` is retained and
  regenerated as the record of exactly what PocketClaw changed.
- Reason: physically storing MIT-licensed upstream source in our repository
  changes where it lives and nothing else. The patch is what keeps the two
  readable apart — 52 changed files, of which 13 are PocketClaw-authored.
- Evidence: upstream `v0.3.1` plus the regenerated patch reproduces `core/src/`
  byte-for-byte, verified on 2026-08-25.
- Consequence: `core/regen-upstream-patch.sh` must be run after any change under
  `core/src/`, or the divergence record goes stale. Running it and getting an
  empty `git diff` is the verification.

## Two upstream paths are deliberately not vendored

- Date: 2026-08-25
- Decision: `assets/` and `pkg/seahorse/.omc/` are excluded from `core/src/`,
  and excluded from both sides of the provenance patch so they never appear as
  PocketClaw deletions.
- Reason: `assets/` is 13 MB of upstream README screenshots and marketing GIFs
  with no build role, and carrying PicoClaw marketing imagery inside PocketClaw
  works against the branding position. `pkg/seahorse/.omc/state/` is an upstream
  developer's tool-state file committed by accident; it contains no code and
  leaks an upstream contributor's home-directory path. Vendoring developer
  machine state was explicitly out of scope.
- Consequence: neither is referenced by any build target, and `core/README.md`
  records the omissions so a future reviewer does not read them as loss.

## Release binaries are built with `-trimpath`

- Date: 2026-08-25
- Decision: `-trimpath` is added to the four Android arm64 `go build` lines in
  `core/src/Makefile` and `core/src/web/Makefile`, and
  `core/build-android-arm64.sh` fails the build if the shipped binaries contain
  any `/home/`, `/Users/`, or `/root/` string.
- Evidence: this was not theoretical. Before the change the shipped
  `libpicoclaw.so` contained 2,501 absolute `/home/lordegypt/...` paths and
  `libpicoclaw-web.so` contained 1,346 — every compiled file's full path on the
  build machine, reachable from the user-facing Logs screen. Both are now zero.
- Reason: `-s -w` strips symbols but does not remove the file paths Go records
  for tracebacks. Moving the source into the repository would have replaced one
  set of developer paths with another, so the fix had to be the compiler flag,
  not the directory layout.
- Consequence: the binaries shrank by roughly 197 KB and 131 KB, and their
  hashes will not match any previously recorded pair. Build provenance now
  lives in `core/README.md` and `UPSTREAM_BASELINE.md`, which is where it
  belongs.

## Upstream's `onboard` ignore rule is anchored in the vendored tree

- Date: 2026-08-25
- Decision: `core/src/.gitignore` changes the bare `onboard` rule to `/onboard`.
- Reason: the bare rule sits in upstream's "Secrets & Config" block and was
  plainly meant for a generated root-level `onboard` artifact, but an unanchored
  pattern matches at every depth. It swallowed
  `cmd/picoclaw/internal/onboard/`, which carries PocketClaw changes to
  `helpers.go` and `helpers_test.go`. Upstream never noticed because those files
  were already tracked there; a fresh vendored copy would silently have dropped
  four source files.
- Consequence: this also had to be handled in `core/regen-upstream-patch.sh`,
  which force-adds both sides. Without that, the upstream baseline commit
  excluded the same files and the patch misreported unmodified upstream source
  as PocketClaw-authored additions.

## Go caches are toolchain, and live outside the repository

- Date: 2026-08-25
- Decision: the Go build and module caches moved from
  `.upstream/picoclaw-core-v0.3.1/.cache/` to
  `/home/lordegypt/PocketCLaw/.tooling/go/`, alongside the JDK, Flutter, pnpm,
  and Android SDK. `core/build-android-arm64.sh` points `GOCACHE`/`GOMODCACHE`
  there and accepts a `GO_CACHE_ROOT` override.
- Reason: 3.3 GB of cache had been living inside the external Core checkout,
  which made that directory a build prerequisite for a second, non-obvious
  reason. It also holds the Go 1.25.11 toolchain that `core/src/go.mod`
  requires and the system Go 1.22.2 cannot satisfy. Caches are regenerable
  build state, so they belong with the tools, not in the repository and not in
  a source checkout.
- Consequence: the caches were moved, not deleted, and a fresh machine can
  instead let Go use its defaults and download from `core/src/go.sum`.
