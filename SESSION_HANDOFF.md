# PocketClaw Session Handoff

## Current Objective

**Physically verify the DEBUG log cleanup candidate. The GitHub release remains
on hold until it and the broader pre-release checklist pass.**

Milestone D is closed and verified. Candidate `f663d25a...71c621d` physically
confirmed the neutral Telegram card, basename callers, log debranding,
terminal-control cleanup, and exactly-once queue/drain behavior. Preserve those
fixes. Its exported DEBUG file exposed two smaller issues, now fixed on
`fix/user-facing-log-privacy` (not merged, not tagged).

Install:

    build/app/outputs/flutter-apk/app-release.apk
    SHA-256 eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8
    34,239,073 bytes · com.lord1egypt.pocketclaw 0.1.3 (3)   [version unchanged]

Check on the device:

1. Export a DEBUG log containing HTTP durations. It must contain valid
   `53.616µs`, not `53.616�s`, with no U+FFFD introduced by PocketClaw.
2. Arabic, emoji, Unicode punctuation, and ANSI/control removal remain correct
   in both Logs and Export.
3. Several minutes of successful `GET /api/gateway/logs` and
   `GET /api/gateway/status` polling do not consume user-visible history.
4. A failed poll and ordinary requests such as config/models remain visible.
5. Basename callers, user-facing debranding, and exactly-once queue/drain
   behavior remain physically correct.
6. Continue the standing Telegram, skills, background/locked, provider, Skill
   Hub, startup, and performance sweep.

**The Core binaries are new again in this build**, so the regression sweep is
required. New pair: `libpicoclaw.so` `5c09eb72...4d763bc`,
`libpicoclaw-web.so` `cb6b10cc...4b03b52`.

Do not merge, do not tag, and do not create the GitHub release until this
passes. Do not touch `main`.

### What the DEBUG-export bugs were

The sanitizer was already Unicode/rune-safe. Corruption happened later in
Android Export Logs: Dart `content.codeUnits` (UTF-16 code units) was truncated
to bytes, making `µ` a lone invalid `B5`. The MediaStore bridge now receives
real UTF-8 bytes from `utf8.encode`; the actual MethodChannel export boundary
has a strict-decode regression test.

The polling entries were genuine DEBUG middleware events, not the old sticky
snapshot bug. Middleware now omits only exact successful 2xx `GET` requests to
`/api/gateway/logs` and `/api/gateway/status`. Errors, unexpected methods,
redirects, unknown routes, and other API requests remain logged. Do not alter
`publishLog`/`takeNewLogs` or reintroduce `status['lastLog']`.

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

## CODEX SOL HANDOFF — PRE-RELEASE FIX

Continue on `fix/user-facing-log-privacy`. The exact pre-Codex state is
recoverable from pushed branch `checkpoint/pre-codex-sol-prerelease-fix` or
annotated tag `pre-codex-sol-prerelease-fix-20260826`, both at
`e5b88ff1a4c8f76e321c07af97eff5ca23d59d78`. Historical Milestone D remains
`phase2-milestone-d` / `8f861bca1c82b43b306e95b14e277269260bbab0`.

Automated work is complete and one candidate exists:

- APK: `build/app/outputs/apk/release/app-release.apk`
- Size: 34,239,649 bytes
- SHA-256: `f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
- Core: `87653601023974156be1b1a255387ec3c93a0c154254a7b8d9e1012ac2e926e5`
  and `a8328f1d66932ba8da544060905965278898744c04c2ece5a993717e361400c5`.
- Guard, metadata, endpoint, `-trimpath`, and secret scans: PASS.
- Flutter analyze/test (94), frontend Vitest (36)/tsc/lint, and required Go
  suites: PASS.

Native Telegram is now only a shortcut to Core's `/channels/telegram` page;
Core owns credential persistence through a narrow authenticated loopback write;
Telegram outbound HTTP is deadline-bounded; request/placeholder state is
correlated and same-session Telegram requests are independent FIFO lifecycles;
edit/send failures are no longer swallowed; seven fresh skills are repaired
non-destructively; Android logs are sanitized once for display/export and use a
plain banner/neutral PID warning. Full root causes, file groups, tests, and exact
physical symptoms are in the matching section of `PROJECT_STATE.md`.

Physical testing is still **PENDING**, so release is **BLOCKED**. Do not merge,
touch `main`, move `phase2-milestone-d`, or create a GitHub release. Test a lone
`تسلم` with no wake-up message, empty-response recovery, sequential/close
messages, foreground/background/locked screen, placeholder/typing/streaming
variants, neutral native Telegram navigation, reconnect preservation, skills
repair/import, Unicode logs/export, provider, Skill Hub, startup, and speed.

Deferred only: Background & Battery UX; Runtime/Statistics tab; controlled Dart
source-URI hardening. Manager-token rotation remains pending unless explicitly
confirmed externally; never retrieve or print the token.

## WEB CONSOLE LOG PARITY CANDIDATE — 2026-08-27

Continue on `fix/user-facing-log-privacy`. The previous UTF-8 export,
successful log/status poll suppression, exactly-once drain, basename callers,
and neutral Telegram card all remain passing and must not be reopened.

The current automated candidate is
`3e138b4a53a0389b76dbef045649d906fe2db785cd2af606826f7a0f87170adc`
(34,241,381 bytes). Core hashes are
`c9c348e9c637a7552e810396bfba5460ad4a3f65ab506d933ac5d05ea68d236e`
and `c891ca033ededb6b8941d997fbc8e0a8d69eb18f44a6ed2176f878f65d34840a`.
The three-library ARM64 guard passed, Core paths are clean, the onboarding
endpoint is present once, and package/version remain 0.1.3 (3).

Root causes and fixes: the gateway's no-color banner was still Unicode block
art; captured launches now force `--no-color` and render one `PocketClaw` line.
The Web log ring stored raw output and React only interpreted SGR; the ring now
stores the shared fixture-backed plain-text contract and the real Logs page has
an idempotent parity guard instead of an ANSI renderer. Startup no longer logs
the executable/library path. Routine successful `/pico/ws` events are omitted;
failures remain under `/internal realtime connection`. No compatibility
identifier was renamed.

Automated validation is complete: Flutter analyze plus 99 tests, frontend 37
tests/tsc/lint, and tagged Go logger/gateway/API/middleware/Core CLI suites all
pass. Physical validation remains mandatory and release remains blocked. Next:
install only this APK, open Core Web Console Logs for several minutes, export a
DEBUG log, and compare all three surfaces for intact `53.616µs`, Arabic, emoji,
zero controls/boxes/ANSI, zero routine `/pico/ws`, zero internal library path or
unintended upstream branding, and a visible neutral genuine-error line. Do not
merge, release, touch `main`, move `phase2-milestone-d`, or start a milestone.

## FINAL LOG BRAND MICRO-FIX CANDIDATE — 2026-08-27

Physical validation of `3611ca1` passed the Web terminal cleanup, banner,
UTF-8/`µs`, library-path visibility, and skills checks (8/8, 17 tools). The sole
remaining line was `Channels enabled: [telegram pico]`.

Internal meaning: `pico` is Core's singleton authenticated Web Console
WebSocket/media transport and compatibility/config ID. It remains unchanged
everywhere internal. Only the startup/reload summary copies the enabled names
and maps exact `config.ChannelPico` to display label `pocketclaw`.

New APK: `aab3c565bd6bec2eb714443756e16be5d2ebe8d4496d94e64a6b8ecac25b3582`
(34,241,857 bytes). Core hashes: `10446d81...71b45f1` and
`683463df...e5f5ce4`. Tagged relevant Go suites, Core provenance/build,
zero-path checks, endpoint check, packaged hashes, and permanent guard pass.
Physical action: restart Core and confirm `[telegram pocketclaw]` while all
previously passed log and 8/8 skills behavior remains intact. No merge/release.

## FINAL USER-VISIBLE CALLER BRAND CANDIDATE — 2026-08-27

The last physical leak was structured logger header `INF pico
pico.go:<line>`. Internal package/file/channel/routes/config remain unchanged.
At the shared Go/Dart/React normalization contract, only exact logger component
`pico` displays as `realtime` and exact caller basename `pico.go` as
`realtime.go`, preserving the numeric line. Substrings and `pico_client` are
unchanged.

New APK: `1eeca7c993d657f0e6e763d94691a054e584592d0989445454144ac15ad089f7`
(34,242,865 bytes). Core hashes: `a379ae45...07eafb9` and
`584dd9ae...1fa4a01`. Flutter analyze/99 tests, frontend 37/tsc/lint, relevant
tagged Go suites, provenance, zero paths, endpoint, packaged hashes, and guard
pass. Physical device remains the final gate; no merge or release.

## WEB CONSOLE LOG VIEWPORT STABILITY CANDIDATE — 2026-08-27

Continue on `fix/user-facing-log-privacy`. The previous candidate's complete
log sanitization/branding behavior passed physically. This micro-pass changes
only the Core React Logs viewport.

Root cause: bottom correction ran after paint in `useEffect`, and long lines
were hard-wrapped from a `ResizeObserver` measurement of content whose height
changes on every append. That could expose an old-offset frame and then rewrite
all long-row layout. Index keys also lacked the API's real event identity.

Fix: conditional pre-paint `useLayoutEffect` bottom following, no writes while
scrolled up, stable `run_id:absolute_offset` keys, memoized rows, and native CSS
wrapping of the unchanged plain-text entry. No native queue, logger sanitizer,
poll filter, Telegram, Skills, Provider, or internal route logic changed.

New APK: `be5d7cbb18c0378dad0a3d53d2d3a4e71001411121fa056f7a6606645070fc96`
(34,239,873 bytes). Core hashes: `715cd790...6143cf1` and
`e3930ae2...f5f5da14`. Frontend 41/41, tsc/lint, relevant Go, Native/Export
regression, provenance, zero paths, packaged hashes, and guard pass. Physical
device is the final gate; no merge/release/main change.

## TELEGRAM WEB LOG REDACTION CANDIDATE — 2026-08-27

Physical clarification: token fragments appeared only in Core Web Console
Logs; Native Android Logs remained clean. This is case A. Telego's full Bot API
URL was partially masked in PocketClaw's logger before stdout and before Web
`LogBuffer` storage. The full token was not persisted through this path, but
the old bot-ID/first-four/last-four fragments were stored and rendered.

The pre-stdout logger now retains no credential fragment. Web pre-storage
normalization turns Bot API URLs into `Telegram API call: <operation>` and
keeps method/status/error/timeout/latency; React has an idempotent legacy guard.
Native/Export sources and all Telegram credentials/lifecycle code are
unchanged. No rotation occurred.

New APK: `8257e9f091039f2c29332b8f14e2d397d36bbb735593596a4caa0e567c7050fe`
(34,240,641 bytes). Core hashes: `0e914550...8b555e9` and
`7d7b254b...d9898c7`. Relevant Go suites, frontend 42/42/tsc/lint, unchanged
Native log regression, 115-file provenance, zero paths, packaged hashes, and
guard pass. Physical Web Console validation remains mandatory; no merge,
release, main change, or credential mutation.

## FINAL LEGACY BRAND VISIBILITY SWEEP CANDIDATE — 2026-08-27

The physically observed `.picoclaw.pid`, structured `channel=pico` /
`type=pico`, `/pico/` registration path, and Pico Protocol lifecycle wording
were traced to gateway compatibility details entering the Web ring. The shared
pre-storage normalizer now gives only these exact classified identities semantic
PocketClaw/realtime/internal/gateway display wording; React keeps the same
idempotent guard for raw or historical lines. The security warning remains
fully visible and only its exact channel field changes.

All runtime compatibility identities and files remain unchanged. Substring
negative cases remain byte-for-byte unchanged. A representative full startup
stored in `LogBuffer` and the real Logs DOM have zero unintended legacy brand
occurrences. New APK: `309f6d7a...a5f3030`. Core hashes:
`49f89ae2...be656f` and `98f3fa9d...08bae`. Frontend 44/44/tsc/lint,
relevant tagged Go including Skills/Pico/Telegram, Flutter analyze/99 tests,
115-file provenance, zero paths, and guard pass. Physical device remains the
final gate; no merge/release/main change.

## TELEGRAM DEBUG FINAL CLEANUP CANDIDATE — 2026-08-27

Telego emits correct `Err: [<nil>]`; pre-stdout redaction preserved it. The Web
backend's orphaned-CSI fallback then misclassified `[<n` as a control suffix,
so `LogBuffer` stored `il>]`. The fallback is now restricted to numeric CSI
remnants. Known successful Telego nil fields display as `Err: none`, while
ordinary angle-bracket text is preserved and React continues safe text-node
rendering.

Exact DEBUG `getUpdates` request lines and successful empty responses are now
discarded before stdout, with idempotent shared-boundary guards. Failures,
non-empty updates, API errors, send/edit operations, lifecycle diagnostics, and
credential redaction remain. Four repeated empty polls produce zero stored
events. No Telegram lifecycle/onboarding/delivery, Skills, Provider, Pico,
viewport, or native queue logic changed.

New APK: `2c00720a...2da27c` (34,243,585 bytes). Core hashes:
`7c1d3918...38ef1f` and `1611b6e1...d09256`. Frontend 45/45/tsc/lint,
relevant tagged Go including Telegram/Skills, Flutter analyze/101 tests,
115-file provenance, zero paths, and build guard pass. Physical device remains
the final gate; no merge/release/main change.
