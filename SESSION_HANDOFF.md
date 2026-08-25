# PocketClaw Session Handoff

## Current Objective

**Operator setup, then one end-to-end Telegram test.**

Milestone D — Telegram managed-bot onboarding — is implemented, tested, and
built. It cannot be verified on a device yet, and the blocker is not code: the
real PocketClaw manager bot does not exist. No credentials were invented and no
third-party setup service is used as a stand-in.

Candidate APK
`7c34ab12b544e585981c46632a5246a3a3fe66da24a84fce2c0b831c4911178e`. The
verified reference artifact remains `588bbec1...5ff3a90b`.

### What the operator must do

1. **Create the manager bot** with [@BotFather](https://t.me/BotFather).
   Intended identity: display name **PocketClaw Setup**, username
   **@PocketClawSetupBot**. If that username is taken, pick any other
   PocketClaw-owned one — nothing hardcodes it.
2. **Enable Bot Management Mode** in BotFather's mini app for that bot. This is
   what makes Telegram set `can_manage_bots`. Without it the deep links open
   Telegram but Telegram will not offer to create a bot.
3. **Deploy `services/telegram-onboarding/`** behind HTTPS, with
   `TELEGRAM_MANAGER_BOT_TOKEN` and `TELEGRAM_MANAGER_BOT_USERNAME` set from a
   secret store. Exactly one instance per manager bot token. The service
   verifies itself at startup and refuses to run if management mode is off, so
   a successful start is itself confirmation that step 2 worked.
   `services/telegram-onboarding/README.md` has the full deployment notes.
4. **Rebuild the app** pointing at the deployment:

       flutter build apk --release --target-platform=android-arm64 \
         --dart-define=POCKETCLAW_ONBOARDING_BASE_URL=https://your-host

   or pass the same define through the canonical Gradle path. Without it the
   app correctly reports that automatic setup is unavailable and offers manual
   token entry, which is what the current APK does.

The manager bot token is a **server secret**. It must never reach the APK, the
repository, this file, or a log line.

### Then test

The device checklist is at the end of `PROJECT_STATE.md`. The short version:
Settings → Telegram → Connect → Open Telegram → confirm → back to PocketClaw →
Connected → Open Chat, plus the QR path from a second device, a
background/resume mid-flow, an expiry, and the manual fallback.

Do not merge into `develop` until that passes. Do not touch `main`. Do not
start another milestone.

## Milestone C state

Merged into `develop` as `36bc88d` and tagged `phase2-milestone-c` on
2026-08-26, after passing on a device. The Core source lives in this repository
at `core/src/`, and `core/pocketclaw-core-v0.3.1.patch` (52 files) is its
divergence from upstream `v0.3.1`; the arm64 binaries are committed under
`android/app/src/main/jniLibs/arm64-v8a/`.

What it delivers: a provider-first Add Provider flow (choose provider → API key
→ fetch or type a model → save) with the alias derived automatically; the
backend catalog extended with categories, documentation links, and the xAI,
Together AI, Fireworks AI, and Custom OpenAI-Compatible presets, all registered
in the protocol switch; working Gemini model discovery; broader fetch error
classification; and no runtime logo fetching from third-party hosts.

Read `docs/PROVIDER_ARCHITECTURE.md` before touching provider code. Two facts in
it will save a wrong turn: AI provider configuration lives in the Core web
console and not in Flutter, and a provider added to the catalog without a
matching arm in `CreateProviderFromConfig` saves cleanly and then fails at
request time with `unknown protocol`.

Deliberately not done: Telegram QR/deep-link onboarding (next milestone), any
release hardening or obfuscation, and any change to API key storage. The last
is not an oversight — see the deferral decision in `DECISIONS.md`.

## OpenCode completion state

OpenCode Zen (`https://opencode.ai/zen/v1`) and OpenCode Go
(`https://opencode.ai/zen/go/v1`) are mixed-protocol gateways: one base URL and
one API key front OpenAI Responses, OpenAI-compatible chat completions, and
Anthropic Messages, and the protocol is a property of the model. All routing is
in `pkg/providers/opencode_routing.go` — if you need to change how an OpenCode
model is dispatched, that is the only file to touch.

Supporting changes: `pkg/providers/openai_responses` is a new generic
Responses-over-HTTP provider (the Azure and Codex ones are hardcoded to their
own endpoints and were not reusable), and `anthropic_messages` gained an opt-in
`WithBearerAuth()` used only by the OpenCode Messages route.

The one thing tests cannot settle: whether OpenCode's Messages surface wants
`X-API-Key` or `Authorization: Bearer`. Both are sent. If a `claude-*` model
401s on the device while `gpt-*` and `kimi-*` succeed, that is the header
question, not the routing.

Build gotcha found the hard way: `:app:packageRelease` without
`-Ptarget-platform=android-arm64` silently produces a ~50 MB universal APK
instead of ~34 MB. Check the size before trusting any release build.

## Milestone B closure

APK `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8` passed a
full physical Android device test on 2026-08-25: install, app launch / first
frame, no black screen, Gateway/Core lifecycle, navigation, PocketClaw branding,
workspace path, QR/access page, Skill Hub, provider/model flow, and no abnormal
slowdown. It is the verified reference artifact — compare any future regression
against it before forming new hypotheses.

`recovery/pocketclaw-clean-debrand` (@ `f25d38e`) was merged into `develop` with
a non-fast-forward merge, `225be3c`, and the closure point is tagged
`phase2-milestone-b`. The merged `develop` tree is byte-identical to the tested
recovery tip. The recovery branch, `feature/pocketclaw-identity` (`354fc38`
WIP), and `main` are all intact; nothing was deleted or rewritten.

What Milestone B delivered: independent PocketClaw product and package identity;
the black-screen root cause and its permanent fail-closed release guard; full
user-facing debranding including the embedded web runtime rebuilt from source;
`Download/pocketclaw` for fresh installs; the MQTT `/pocketclaw` default with
backward compatibility; and the upstream agent skill dropped from fresh
workspace seeding.

## Permanent release requirements — do not remove

- `packageRelease` fails the build unless the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`.
  It prints "Verified arm64-v8a native payload" on success.
- Canonical release command:
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` with
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
  A universal `flutter build apk --release` is not a PocketClaw release path.
- If the guard reports `libdartjni.so` missing, do not change application code:
  purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and rebuild.
- Rebuild Core with `core/build-android-arm64.sh`, which builds from
  `core/src/` through the root Makefile targets. Calling
  `make -C web build-android-arm64` directly drops the root `LDFLAGS` and
  produces an unstripped binary. Use `pnpm lint`, never `pnpm check`.
- Never point a build at a Core checkout outside this repository. `core/src/`
  is the source-of-truth; `core/verify-no-external-source.sh` proves the build
  needs nothing else. Upstream is fetched for review only.
- `-trimpath` on the Android arm64 build lines is a release requirement, not a
  nicety. Without it the shipped binaries carry thousands of build-machine
  paths that reach the user-facing Logs screen.
  `core/build-android-arm64.sh` fails the build if the count is not zero.
- Regenerate `core/pocketclaw-core-v0.3.1.patch` with
  `core/regen-upstream-patch.sh` after any change under `core/src/`, or the
  divergence record goes stale.
- Never commit the manager bot token, and never add it to a `--dart-define`.
  The app receives only a public HTTPS base URL; the child bot token reaches it
  once, over TLS, and goes straight into Core's config.
- Telegram onboarding must keep writing the existing `channel_list.telegram`
  entry. Do not add a second Telegram runtime; both the automatic and the
  manual path converge on the same channel.
- Core binaries currently committed and DEVICE-VERIFIED (2026-08-25):
  `libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`;
  `libpicoclaw-web.so` 24,641,889 `b6b356f7...656db9ba5`. Both stripped, PIE
  `ARM aarch64`, `-trimpath`, with `PICOCLAW_DNS_SERVER` present and zero
  developer-machine paths. They supersede Milestone C's `cbe568af...5556468a`
  and `86e53457...0cb0a4cd`.
  Core hashes are not reproducible across rebuilds — `BuildTime` is stamped in
  via `-ldflags` — so compare sizes and the zero-path count, not hashes.

## Exact State

This is a standalone Git repository at `/home/lordegypt/PocketClaw-App`.
It has no inherited PicoClaw FUI history. The initial source foundation is a
selective adaptation from the FUI baseline documented in `UPSTREAM_BASELINE.md`.
It is private at `https://github.com/Lord1Egypt/PocketClaw`, with `main` and
`develop` tracking `origin`. The initial commit is `950d4a3`.

The Android service and bundled Core retain the physically verified DNS bridge:
Android active-network DNS servers are passed as `PICOCLAW_DNS_SERVER`; public
resolvers are not hardcoded. Optional Firebase/Umeng feedback defaults to
disabled and requires explicit complete configuration.

Validation completed: `flutter analyze` is clean; `flutter test` passed all
28 tests; focused Core `pkg/androiddns` and `web/backend/api` tests passed.
The Milestone A independent arm64 APK was
`build/app/outputs/apk/release/app-release.apk`, 32,619,416 bytes, SHA-256
`207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`.
Its package remains `com.sipeed.picoclaw` version `0.1.3` (code `3`) by
design. Embedded Core hashes match the
pinned records. The release build reported no Firebase app ID, API key, or
project ID and removed generated Firebase resources after packaging.

The foundation physical regression was subsequently confirmed PASS, including
AI request/response, Telegram send/receive, Core restart/configuration
persistence, and no feedback log spam. Skill Hub/ClawHub was also verified:
searching `Crypto` returned 20 results with metadata, URLs, and visible install
actions. The earlier unavailable-registry symptom is resolved by the Android
DNS fix; do not rewrite Skill Hub without new source/runtime evidence.

Milestone B changes product-visible naming to PocketClaw and migrates only the
Android/Dart integration identity to `com.lord1egypt.pocketclaw`. Core names,
binary names, `PICOCLAW_DNS_SERVER`, protocol identifiers, and the compatible
external `Downloads/picoclaw` workspace remain deliberately intact. The second
original PocketClaw mark is the selected primary visual direction; launcher,
adaptive, and monochrome variants derive from it. Do not replace it with a
cartoon lobster/mascot.

The Milestone B physical test then found a black startup screen and noticeable
device slowdown. The root cause is the branded Android launch drawable: both
`launch_background.xml` variants used `<item android:color=...>` in a
`layer-list`; `LayerDrawableItem` requires `android:drawable` (or a child
drawable), so Android cannot inflate the launch window before Flutter's first
frame. The replacement changes only those two items to reference
`@color/pocketclaw_splash_background` and adds a regression test. The 1254 px
RGBA splash mark is 654,845 bytes on disk and about 6.0 MiB decoded, which is
within a reasonable startup budget and was not the root cause.

The fixed validation results are: `flutter analyze` clean; all 29 Flutter
tests pass; focused Core `pkg/androiddns` and `web/backend/api` tests pass.
Debug and release APKs built successfully. The replacement release is
`build/app/outputs/flutter-apk/app-release.apk`, 33,308,653 bytes, SHA-256
`45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`.
`aapt2` confirms `com.lord1egypt.pocketclaw`, label PocketClaw, launchable
`com.lord1egypt.pocketclaw.MainActivity`, and a compiled splash layer with a
drawable-backed first item. Embedded Core hashes remain
`3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d` and
`252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.
The service/Core code was unchanged: a fresh Flutter launch does not issue a
Core start request, and the native service has its bounded three-restart guard.
No connected ADB device was available to measure the reported slowdown or
capture final device logs, so that symptom must be confirmed during retest.

## Next Exact Steps

1. Operator setup, above. Nothing else in Milestone D can proceed without it.
2. Run the end-to-end Telegram test from the checklist at the end of
   `PROJECT_STATE.md`.
3. On PASS: merge `feature/telegram-managed-onboarding` into `develop` with
   `--no-ff`, tag the closure point, and record the result in the state
   documents. `main` needs separate instruction.
4. On FAIL, the likely causes in order: Bot Management Mode not actually
   enabled (the service would have refused to start, so check it is the running
   build); the user edited the suggested username on Telegram's confirmation
   screen, which by design fails closed and expires the pairing; or the app was
   built without `POCKETCLAW_ONBOARDING_BASE_URL`.
5. Two open items unrelated to the blocker: localizing the onboarding strings,
   and confirming a `claude-*` model on OpenCode. Both are in `TASKS.md`.
6. Do not start another milestone.
