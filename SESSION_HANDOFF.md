# PocketClaw Session Handoff

## Current Objective

**Physically re-test the log regression fix. The GitHub milestone release is on
hold until it passes.**

Milestone D itself is closed and verified. After closure, the device showed two
user-facing log defects in the verified APK; both are fixed on
`fix/user-facing-log-privacy` (not merged, not tagged).

Install:

    build/app/outputs/flutter-apk/app-release.apk
    SHA-256 543c759b04b0e4c77dd7831435753aceac0b1e16a7a45fdec3fb37ed2e45479a
    34,227,525 bytes · com.lord1egypt.pocketclaw 0.1.3 (3)   [version unchanged]

Check on the device:

1. The Logs screen shows no `github.com/sipeed`, `picoclaw`, `Sipeed`,
   `/home/lordegypt`, or `.upstream`.
2. Callers read like `gateway.go:298` — no module or repository prefix.
3. One stale-PID cleanup appears **once**, not dozens of times, and real log
   history is no longer evicted.
4. Telegram Managed Bot onboarding still works end to end.
5. Native Settings shows **Connected** with the bot handle — not "Connect
   PocketClaw to Telegram" — and tapping it opens the connected page instead of
   starting a new pairing.
6. Reconnect / Create New Bot asks first, and cancelling leaves the working bot
   configured.
7. Both surfaces still show Connected after a full app restart.
8. No black screen, no slowdown.

**The Core binaries are new again in this build**, so the regression sweep is
required. New pair: `libpicoclaw.so` `24c7df0a...e8029962`,
`libpicoclaw-web.so` `bab16f30...351a90b` — these replace the device-verified
`33f8b4ef...` / `5400cb02...`.

Do not merge, do not tag, and do not create the GitHub release until this
passes. Do not touch `main`.

### What the two bugs actually were

`-trimpath` removed `/home/lordegypt/...` from the caller and left the Go
**module path** in its place; every existing check looked only for `/home/`, so
nothing caught it. Fixed with `zerolog.CallerMarshalFunc` in
`pkg/logger/logger.go` — the earliest structured layer, so all writers and
exports inherit it.

The repeated warning was **not** the backend emitting repeatedly. Native
`lastLog` is a sticky snapshot, and the Flutter three-second poll appended it
every tick, filling the 500-entry buffer from one event and evicting real
history. Fixed by draining: `publishLog`/`takeNewLogs` hand each line out once.

Do not reintroduce `status['lastLog']` as a log source — that is the bug.

The third defect: native Settings hardcoded "Connect PocketClaw to Telegram"
and never read any state, so it disagreed with the console. Both surfaces now
derive from `channel_list.telegram` via `TelegramConnectionReader`. Do not add
a stored "connected" boolean — that is what one source of truth prevents.

## Milestone D — COMPLETE

Phase 2 Milestone D, Telegram Managed-Bot Onboarding, passed its physical
end-to-end test on a real Android device on 2026-08-26 and is closed. Merged
into `develop` with a non-fast-forward merge and tagged `phase2-milestone-d`.
`main` remains deliberately untouched at `100a51d`.

Automatic managed-bot onboarding is now the **default** Telegram setup path.
Manual Bot Token entry is the **advanced fallback**, still complete and
reachable.

### The verified reference artifact

    APK              b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94
                     34,220,929 bytes · com.lord1egypt.pocketclaw 0.1.3 (3)
    libpicoclaw.so       37,224,801 · 33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a
    libpicoclaw-web.so   24,641,889 · 5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd

This supersedes the Milestone C APK `588bbec1...5ff3a90b` and the Milestone C
Core pair. Diff any future regression against it first.

### Live architecture

    PocketClaw Android
      → PocketClaw Telegram Setup  (Vercel)
        → @PocketClawSetupBot      (manager, Bot Management Mode)
          → Telegram Managed Bots
            → Upstash Redis        (pairing state, REST, 600 s TTL)
              → automatic PocketClaw Telegram configuration

Official manager `@PocketClawSetupBot`. Service
<https://pocketclaw-telegram-setup-bot-83ai.vercel.app>. Public service
repository `Lord1Egypt/PocketClaw-Telegram-Setup` — independent, public, MIT,
and not a build input to the APK.

What was verified physically: the full Channels → Telegram onboarding UI, bot
creation through Telegram, automatic pairing detection and token delivery,
automatic owner and channel configuration, the connected state, Open Chat, and
a real Telegram → Core → AI provider → Telegram message round trip with context
persisting across consecutive messages. The full record is in
`PROJECT_STATE.md`.

### Still open, and deliberately so

- **Rotate the manager bot token.** It was pasted into a chat log after
  deployment. The service keeps working; rotate it in BotFather and update the
  Vercel environment variable. This is the one outstanding security item and it
  is not blocked by anything.
- Localize the Telegram onboarding strings — English in all twelve locales.
- Confirm a `claude-*` model on OpenCode, carried over from Milestone C.

### Architecture worth knowing before touching Telegram again

Telegram has two entry points and they must stay converged on one flow:

- Native Settings → Telegram (`lib/src/ui/config_page.dart`)
- Channels → Telegram in the embedded console
  (`core/src/web/frontend/.../channel-config-page.tsx` → `telegram-panel.tsx`)

Both call `TelegramOnboardingLauncher.open`. The console never runs pairing
itself; it asks the Flutter host over the `PocketClawHost` bridge
(`lib/src/ui/webview/pocketclaw_host_bridge.dart`). Do not reimplement pairing
in TypeScript — that split is what made Milestone D ship unreachable the first
time. The bridge is scoped to the console's origin and `openExternal` accepts
only absolute http(s) URLs; both are tested, do not loosen either.

The lesson worth keeping: a Flutter widget test proves the code runs, never
that a user meets it. Any UI claim here needs a test entering through the same
route the app does.

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
- The onboarding service lives in its own public repository,
  `Lord1Egypt/PocketClaw-Telegram-Setup`. It is infrastructure, not part of the
  APK build; `services/README.md` explains why it is not vendored here.
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

1. Revoke the exposed manager token in BotFather.
2. Deploy `Lord1Egypt/PocketClaw-Telegram-Setup` to Vercel with the variables
   above, including a Redis store.
3. Register the webhook and verify the manager from the setup page. The
   `can_manage_bots` assertion happens here.
4. Rebuild the app with `POCKETCLAW_ONBOARDING_BASE_URL` and run the device
   checklist at the end of `PROJECT_STATE.md`.
5. On PASS: merge `feature/telegram-managed-onboarding` into `develop` with
   `--no-ff`, tag the closure point, and record the result. `main` needs
   separate instruction.
6. On FAIL, the likely causes in order: the app was built without
   `POCKETCLAW_ONBOARDING_BASE_URL`; the webhook was never registered, or was
   registered against a preview deployment URL rather than the production one
   (set `PUBLIC_BASE_URL` if so); or the suggested username was edited on
   Telegram's confirmation screen, which by design fails closed and expires the
   pairing.
7. Two open items unrelated to the blocker: localizing the onboarding strings,
   and confirming a `claude-*` model on OpenCode. Both are in `TASKS.md`.
8. Do not start another milestone.
