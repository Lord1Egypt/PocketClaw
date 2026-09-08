# PocketClaw Tasks

## Status Dashboard v1 — CLOSED 2026-09-07

Branch `feature/status-dashboard-v1`, head `85eb93a`, merged to `develop` with
`--no-ff`. Physically accepted on SM-A165F / Android 16 as vc53, APK
`6469e9f1…`, Core fingerprint `f4a3f913…`. `main` untouched, no tags moved, no
release. Branch retained.

| Item | Value |
| --- | --- |
| versionName / versionCode | 0.2.0 / 53 |
| APK SHA-256 | `6469e9f103109505c9d05c9b47534e033a633d1da36c636621ecfa550196e857` |
| Core source fingerprint | `f4a3f913014034c974df39a63214c34fb66d4e8b08d848115b58db4723c12104` |
| `libpicoclaw.so` | `bf4fb01faf2741ccc402f3c07b4925e6cd4ebdd7d581d8775ab1744d50b8fc44` |
| `libpicoclaw-web.so` | `bd84eaba705f3d0ec0e71e8cfbefe5cf392668b4a53807b8b9bafb5d89422280` |

Core was built for vc52 and reused byte-for-byte in vc53, which changed only
Dart. The fingerprint is unchanged across both.

- [x] Status is the existing tab-0 destination, not a fifth tab. Navigation
  remains four destinations.
- [x] The access section keeps Start/Stop, Public Mode state, endpoint, QR and
  the QR instructions together in one card.
- [x] SYSTEM, AI, ACTIVITY, CHANNELS, RESOURCES render from one authenticated
  snapshot.
- [x] Counters are in-memory and gateway-process lifetime, incremented at the
  two lifecycle points that already classify the work — `runTurn`'s terminal
  defer and `recordToolExecution`. No database, no metrics file, no history.
- [x] Detail rides the existing 3-second health poll and is requested only while
  tab 0 is selected. No new timer, no WebSocket, no SSE.
- [x] `/health?detail=1` requires the gateway bearer token that already guards
  `/reload` and fails closed. Anonymous `/health` is byte-identical to what it
  always returned.
- [x] `detail.system.uptime_seconds` is whole seconds as a number; Flutter
  renders a localized duration. The legacy `uptime` duration string is
  unchanged for the launcher and the host.
- [x] The obsolete 100px footer reservation is gone; the bottom gap is the
  card margin plus the scroll view's own padding.

### Truthful semantics — do not regress these

These were each corrected after an audit found the first version overstated what
the runtime knows. Changing any of them needs a reason better than convenience.

- **Channels are Running or Stopped, nothing else.** Running requires *both* a
  worker and the channel's running flag. There is no "Failed to start": nothing
  retains a start failure — `StartAll` and `Reload` keep theirs in local
  variables for a log line and a returned error — so a channel that failed and
  one the gateway has not started yet are indistinguishable from here. Naming
  one a failure would necessarily mislabel the other. "Connected", "Healthy" and
  "Reachable" are not claimed at all, because nothing probes the network.
- **Fallbacks is `max(len(candidates)-1, 0)`.** `resolveModelCandidates` builds
  the list as the primary followed by its fallbacks and `ExecuteCandidate` walks
  the whole list in order, so index 0 is the model in use. Reporting the raw
  length credited every agent with one fallback more than it had, including
  agents configured with none.
- **Active turns and active subagents are separate gauges**, split by turn depth
  in `activeTurnStates`. They are never summed into an "operations" number, and
  neither is a tool call or a queued message.
- **`/health`'s `active_requests` counts provider calls, not turns.** It includes
  background summarization. It is right for deciding whether a restart is safe
  and wrong for any user-facing "running" number.
- **Activity counters reset with the gateway process**, which the screen states
  as "Since Gateway start".
- **Uptime is transported as a number with a unit.** Piping Go's
  `Duration.String()` to the screen is what produced an unreadable,
  unlocalizable value; nothing on that path had ever parsed it.
- **Status is read-only about models.** It reports the runtime model, and the
  configured default beside it only when `/switch` has moved them apart. It
  reports the divergence and does not resolve it.

### Deliberately not built

Live CPU percentage, app-process RAM, a recent-activity feed, charts,
sparklines, historical metrics, persistent counters, token or cost accounting,
per-tool breakdowns, tool durations, channel message counts, channel
reachability probes, MCP health, and a web-console Status page.

## Telegram Command UX Phase A — CLOSED 2026-09-06

Branch `feature/telegram-command-ux`, head `896021f`, merged to `develop` with
`--no-ff`. Physically accepted on SM-A165F / Android 16 as vc46, APK
`0d430478…`. `main` untouched, no tags moved, no release. Branch retained.

- [x] Raw CLI grammar removed as the primary no-argument response for `/switch`,
  `/show` and `/list`, at the executor presentation layer rather than per
  command.
- [x] Exact usage retained for genuinely invalid advanced syntax.
- [x] `/subagents` struct leak fixed at the type boundary: `UserMessage`,
  `SessionKey`, `ChatID` and the turn identifiers are unreachable from the
  formatting code, not merely unprinted.
- [x] Zero-value timestamps no longer rendered; unknown status tokens dropped.
- [x] `/help` is a product overview with no command grammar.
- [x] `/btw` answers a bare invocation with a sentence.
- [x] Command descriptions rewritten for people, which also updates Telegram's
  native "/" menu through the existing `RegisterCommands` mechanism.
- [x] `/switch` no longer points at `/list models`, which enumerates nothing to
  switch to.
- [x] Advanced textual forms unchanged; `/stop` lifecycle untouched.
- [x] Deliberately **not** done: interactive buttons, callbacks, `/model`.

## Telegram Model Command — CLOSED 2026-09-07

Branch `feature/telegram-interactive-menus`, head `39450df`, merged to `develop`
with `--no-ff`. Physically accepted on SM-A165F / Android 16 as vc50, APK
`71ac692f…`, Core fingerprint `16a78423…`. `main` untouched, no tags moved, no
release. Branch retained.

What shipped is a fixed informational `/model`:

    🤖 Model selection is managed from PocketClaw Settings.

- [x] `/model` is a real registered command, so Telegram's native "/" menu lists
  it through the existing `RegisterCommands` mechanism.
- [x] Informational by construction: the handler discards the `Runtime`, so the
  switcher and the current-model reader are unreachable rather than merely
  unused, and the reply is a constant that names no model, provider or endpoint.
- [x] No LLM call and no history entry — a handled command returns before
  `runAgentLoop`. Proven with an ordinary message as the control that does both.
- [x] No buttons, no callbacks. A structural test fails if `Request` grows a
  menu, button or callback field.
- [x] `/help` lists it as "Manage models from PocketClaw Settings"; `/switch` is
  "Advanced runtime controls".
- [x] Phase A intact: `/help`, `NoArgsHelp`, `/subagents` privacy boundary, and
  the advanced textual `/switch model to <name>`.

### Phase B interactive menus — ABANDONED BY PRODUCT DECISION 2026-09-06

Built, tested and physically verified as vc47 and vc48. **Not merged, and it
will not be.** The branch was converged back to `develop` instead: every Phase B
file was removed and the only source that survives is noted below.

**This was not a failed implementation.** `/model` worked. Configured-model
eligibility, tap-time revalidation, opaque TTL'd callback handles, chat/sender
binding, single-message editing, cancel and picker retirement all passed their
tests and their physical interaction test on the device.

**What was rejected was the product semantics**, and only physical use exposed
it: the Dashboard and the picker meant different things by "the model".
The Dashboard owns the configured default in `config.json`; the picker moved
only the running `AgentInstance`. A model configured in the Dashboard could be
missing from the picker, and a model chosen in the picker never became the
configured default. That is two sources of truth for one setting, and
synchronizing them is far more machinery than the convenience is worth.

**Model management remains Dashboard-owned.** Telegram stays a conversation and
control surface. See DECISIONS.md, "Model selection belongs to the Dashboard,
not to Telegram".

- [x] Removed: `/model`, the inline picker, the callback registry and TTL
  handles, the menu-action delegate, `bus.InteractiveMenu` and the
  `OutboundMessage.Menu` field, Telegram `callback_query` handling, same-message
  picker editing and retirement, `commands.Menu`/`ReplyMenu`, `Definition.Instant`
  and `Registry.IsInstantCommand`, `Runtime.GetModelPicker`, `BaseChannel.Bus()`,
  `publishResponseWithMenu`, `pkg/modelaccess`, and all their tests.
- [x] `pkg/modelaccess` reverted too, deliberately. It was a verbatim move of
  `hasModelConfiguration`/`requiresRuntimeProbe` out of `web/backend/api`, which
  is their only consumer once the picker is gone — and the move left
  `hasLocalAPIBase` implemented in both packages, which is the drift it claimed
  to prevent. The Dashboard rule is back where it is owned and used.
- [x] Retained: the `harnessWorkspace` fix in
  `pkg/agent/telegram_cancellation_test.go`. `t.TempDir()` races a turn that
  finishes during teardown and writes its session, which turned a passing suite
  red on a filesystem detail. Test-only, independently valuable, latent on
  `develop` before Phase B.
- [x] Retained: `/switch` is described as "Advanced runtime controls" and no
  longer promises a picker. It still performs `/switch model to <name>` against
  the running agent and now says so, and points at the Dashboard for choosing the
  default.
- [x] Deliberately not done: no replacement model command, no persistence, no
  synchronization of runtime selection into configuration, no interactive-menu
  infrastructure kept for later.

### Follow-up — recorded, not started

- [ ] **Stale "Gateway restart required" banner after a successful hot reload.**
  Cosmetic. `gateway.bootConfigSignature` is set only when the launcher starts
  or attaches to a gateway; nothing refreshes it after an in-process reload.
- [ ] **`BaseChannel` typing inconsistency for non-Telegram channels.** Decide
  the defaults before gating them.
- [ ] **`requires.tools` skill metadata is parsed by nothing.** If skill
  eligibility becomes a feature, retire the narrow Android hardware-skill filter
  into it.
- [ ] **`/list models` enumerates nothing.** It reports the current model and
  says to edit config.json. Listing real choices needs a Go-side configured-model
  source; `web/backend/api` has the rule but `pkg/commands` cannot import it.
  Note that listing is not selecting: chat may report what the Dashboard has
  configured, but choosing the default stays in the Dashboard.
- [x] **Release hardening: Core builds are not byte-reproducible by default.**
  Fixed in A3 on `feature/release-hardening-a3`, 2026-09-08:
  `core/resolve-build-time.sh` is the single source of the build timestamp,
  taking `SOURCE_DATE_EPOCH` or the HEAD commit and failing rather than using
  the wall clock, and scoping the default epoch to canonical build inputs so a
  documentation or staging commit cannot redate the build. Two consecutive
  canonical builds produced byte-identical binaries on both the explicit-epoch
  and the default path. **IMPLEMENTED — AWAITING REVIEW.** Original entry:
  ~~**Core builds are not byte-reproducible by default.**~~
  `core/src/Makefile` derives `BUILD_TIME` from `date` unless `BUILD_TIME_RAW`
  is pinned, so two builds of identical source differ. Observed twice on
  2026-09-06: the same tree produced `373c914b…` then `248bc5bd…`, and again
  `5a1eceb4…` then `a2faf15e…`, purely from the timestamp. The source
  fingerprint is unaffected — it is content-addressed and deliberately excludes
  anything time-varying — so staleness detection is sound either way. This
  matters only for independently reproducing a released artifact from its
  source. Recorded, not fixed: do not change build reproducibility as a side
  effect of another task.
- [x] **`TestNoUserFacingWhatsAppSurface` fails on the `develop` baseline.**
  Resolved in A3, 2026-09-08, test-only. It was a guard self-conflict, not a
  product regression: the What's New guard's `forbiddenSubstrings` list names
  WhatsApp in order to forbid it, and the surface scan read that as a violation.
  A file may now declare itself enforcement data with an explicit
  `WHATSAPP-GUARD-ENFORCEMENT-DATA` marker; exactly one file claims it, asserted
  by a test. Original entry: ~~**fails on the `develop` baseline.**~~ The
  guard reports `test/widgets/whats_new_page_test.dart still references WhatsApp
  outside a comment`. It predates the Status milestone, reproduces on `develop`
  unchanged, and has been carried as non-blocking through vc46 to vc53. It is
  the only failure in `pkg/coresource`. Recorded, not fixed: do not modify or
  bypass the guard as a side effect of another task — either the test file's
  reference is legitimate and the guard needs narrowing, or the reference should
  go, and that is its own decision.
- [x] **Release hardening / security review: on Android the gateway bearer token
  lives on shared storage.** Fixed in A2 on `feature/release-hardening-a2`,
  2026-09-08: `PICOCLAW_GATEWAY_TOKEN_FILE` moves the credential to app-private
  no-backup storage and the shared record is written token-free. Core still
  generates it per gateway start, so rotation is unchanged, and non-Android
  deployments are untouched.
  **PHYSICALLY ACCEPTED as vc57 and again as vc58:** the shared record carries
  only pid, version, port and host, with no credential key or credential-shaped
  value.
  The original diagnosis, kept because it explains what the fix had to preserve:
  `pid.WritePidFile` wrote `.picoclaw.pid` — which
  carries the gateway's `token` — into `PICOCLAW_HOME`, and on Android
  `PicoClawService.getWorkspacePath` resolves that to
  `Download/pocketclaw` on shared external storage whenever
  MANAGE_EXTERNAL_STORAGE is granted. The file is created 0600, but external
  storage does not honour POSIX modes, so any app with storage access can read
  it. This is pre-existing and already guards `POST /reload`, which triggers a
  config reload; as of the Status milestone the same token also authenticates
  the read-only `GET /health?detail=1`. Reusing it was the smallest compatible
  design and does not widen the exposure — a reader of that file could already
  reload the gateway — but the storage location itself should be reviewed
  before release. Recorded, not fixed: do not redesign token storage as a side
  effect of another task, and do not weaken detailed-status authentication to
  work around it. A2 did neither: `/reload` and detailed `/health` still share
  one credential and still fail closed; only where it is stored changed.

## Android Hardware Tool Cleanup — CLOSED 2026-09-06

Branch `feature/android-hardware-tool-cleanup`, head `14a88ba`, merged to
`develop` with `--no-ff`. Physically accepted on SM-A165F / Android 16 as vc45,
APK `59ebc6d8…`. `main` untouched, no tags moved, no release. Branch retained.

- [x] The Android Tool Library offers no hardware category; `i2c`, `spi` and
  `serial` are absent rather than shown as three switches that cannot work.
- [x] The status resolvers take the platform as an argument instead of reading
  `runtime.GOOS` inside logic no test could reach; behaviour off Android is
  unchanged and asserted for linux, darwin, windows and freebsd.
- [x] The managed Gateway is launched with the three tools forced off, so a
  persisted, imported or hand-edited config cannot re-enable them.
- [x] They therefore never register into `ToolRegistry` and never reach the
  model's tool definitions — asserted end to end.
- [x] The bundled hardware skill is withheld from discovery on Android, for an
  upgraded workspace as well as a fresh one, because the filter runs at listing
  rather than at onboarding.
- [x] A skill a user wrote and happened to call `hardware` stays visible: the
  bundled one is identified by the two frontmatter fields the loader already
  parses, and a guard fails if the shipped file is reworded out of that match.
- [x] Upstream implementations and embedded skill bytes intact; no new Android
  permission — the installed permission list is identical before and after.

### Follow-up — recorded, not started

- [ ] **Stale "Gateway restart required" banner after a successful hot reload.**
  Non-blocking and cosmetic. `gateway.bootConfigSignature` is set only when the
  launcher starts or attaches to a gateway; nothing refreshes it after an
  in-process reload. Needs a reload-completion signal between two processes that
  do not have one.
- [ ] **`BaseChannel` typing inconsistency for non-Telegram channels.** Discord,
  Slack, Matrix and Feishu start typing through `BaseChannel` without consulting
  `Typing.Enabled`, and `defaultChannels()` leaves the field `false` for them.
  Decide the defaults first.
- [ ] **`requires.tools` skill metadata is parsed by nothing.** The bundled
  hardware skill declares it and the Android filter deliberately does not use
  it, because enforcing it would change what every other skill means. If skill
  eligibility ever becomes a real feature, that is where it belongs — and the
  narrow Android filter should then be retired into it.

## Chat Lifecycle Durability — CLOSED 2026-09-06

Branch `feature/chat-lifecycle-durability`, head `8024094`, merged to `develop`
with `--no-ff`. Physically accepted on SM-A165F / Android 16 across vc42, vc43
and vc44. `main` untouched, no tags moved, no release. Branch retained.

- [x] `/stop` is control-plane traffic and never queues behind the turn it
  cancels; cleanup addresses the cancelled turn's lifecycle, not the command's.
- [x] Ordinary Telegram messages keep strict FIFO, one turn per conversation.
- [x] Configuring fallbacks cannot change the request sent to the first
  candidate; per-candidate providers keyed by model_list identity.
- [x] Every candidate receives the whole turn, media included; each attempt gets
  its own copy so a consuming adapter cannot empty the next one.
- [x] Exhausted-chain and single-model failures both render concise, safe
  errors — no raw bodies, billing URLs or "Original error:".
- [x] A 401 about credits no longer blames the API key.
- [x] Every common `Channel` field participates in the reconcile hash, so saving
  one restarts only that channel.
- [x] `Typing.Enabled` honoured centrally in `Manager.StartTyping`.
- [x] PocketClaw Android's managed Gateway runs with hot reload on, enabled at
  the host boundary; the Core-wide default stays off and is guarded.

### Follow-up — recorded, not started

- [ ] **Stale "Gateway restart required" banner after a successful hot reload.**
  Non-blocking and cosmetic. `gateway.bootConfigSignature` is set only when the
  launcher starts or attaches to a gateway; nothing refreshes it after an
  in-process reload. Fixing it needs a reload-completion signal from gateway to
  launcher that does not exist today — a subsystem change, not a one-liner. Do
  not redesign it opportunistically.
- [ ] **`BaseChannel` typing inconsistency for non-Telegram channels.** Discord,
  Slack, Matrix and Feishu start typing through `BaseChannel` without consulting
  `Typing.Enabled`, and `defaultChannels()` leaves the field `false` for them.
  Gating there today would switch the indicator off for users who never chose
  that. Decide the defaults first.
- [x] **Android hardware tools cleanup (i2c / spi / serial).** Done and
  physically accepted as vc45 — see the milestone below. Hidden at the
  PocketClaw Android boundary; the upstream Core implementations stay.

## Configured Model Discovery — CLOSED 2026-09-05

Branch `feature/configured-model-discovery`, merged to `develop` with `--no-ff`.
Physically accepted on SM-A165F / Android 16 as vc41. `main` untouched, no tags
moved, no release.

The milestone is live model discovery scoped to the providers the user has
actually configured. The reported defect was a Fallback picker offering Azure,
Cerebras, Anthropic, Groq, Ollama and Volcengine to someone who had configured
only OpenCode and Gemini — `config.DefaultConfig()` seeds `model_list` with
thirty keyless provider templates, and that picker was the one selector that
never read the backend's `unconfigured` status.

- [x] One shared source, `lib/configured-model-source.ts`, that decides what any
  routing selector may offer.
- [x] Live discovery per configured provider through the existing secure
  `POST /api/models/fetch` — a `model_index`, never a key.
- [x] Per-provider isolation under `Promise.allSettled`, with refresh and retry.
- [x] Configured and discovered entries distinguished, grouped by provider.
- [x] Provider-scoped identity: normalized provider + normalized API base +
  model id, so two providers exposing the same id stay distinct.
- [x] Materialise-on-selection through `POST /api/models/materialize`, which
  finds or creates the entry and applies a role in one config write.
- [x] Default Model and Fallback Models both drawn from that one source.
- [x] Providers with no `/models` endpoint still offer their configured entries.
- [x] Never any fall back to the global template catalog, including on failure.

### Product decision: no dedicated Vision / Image Model UI

**PocketClaw does not expose a Vision / Image Model selector.** The model
routing surface is Default Model and Fallback Models, and nothing else. A user
who needs image support chooses a multimodal model as their Default Model;
image capability is a provider and model concern rather than something the
product solves with a second routing control.

A dedicated Vision surface was built on this branch and then removed at the
user's decision — commits `bc7c3ff` and `c9c0db2`, reverted. **Vision routing is
not a delivered PocketClaw feature and must not be described as one.**

What that removal did **not** touch: Core has had `agents.defaults.image_model`,
`agents.defaults.image_model_fallbacks` and `routeMediaTurn` since before this
branch existed. Those fields and that routing are upstream, load-bearing and
byte-identical to `develop`. They remain reachable by editing the config file
directly; there is simply no Dashboard or API surface for them.

## APERTURE Visual Redesign — CLOSED 2026-09-05

Branch `feature/visual-redesign-aperture`, merged to `develop` with `--no-ff`.
Physically accepted on SM-A165F / Android 16 as vc38 and vc39. `main` untouched,
no tags moved, no release.

- [x] Replace the stock shadcn token set with the APERTURE layer, remapping the
  existing shadcn variables onto it rather than deleting them, so every
  component inherits the identity without being edited.
- [x] Give every neutral a small chroma at hue 245, rising as the surface
  darkens.
- [x] Reserve Claw (208) for interactive intent and Signal (195) for live
  machine state, and guard that Signal is used nowhere else.
- [x] Emit both Flutter brightnesses from one theme, keeping all six modes, the
  enum order and the persisted preference index.
- [x] Keep `obsidian` true-black for AMOLED and `sakura` light in both slots.
- [x] Draw a vector-first PocketClaw mark that works at 16px, in monochrome and
  in both themes from one asset.
- [x] Delete the visible PICOCLAW wordmark and lobster, and rebrand the favicon
  family, touch icon, PWA icons and web manifest.
- [x] Keep every load-bearing internal Pico identifier; no namespace migration.
- [x] Make `SidebarTrigger` render the icon its caller supplies, give the mobile
  control a 44px hit area, and announce it as opening and closing a menu.
- [x] Collect the low-frequency header utilities behind one labelled, keyboard-
  accessible overflow at phone width, without removing any action.
- [x] Make the desktop sidebar the single brand anchor and remove the second
  wordmark from the top chrome.
- [x] Keep chat deliberately asymmetric: contained user bubble, editorial
  assistant block on a logical-start rail.
- [x] Polish the composer into one control well with a stable footer and a
  single send/stop slot.
- [x] Re-tokenise code wells, tables, inline code, tool rows, the streaming
  indicator and the context ring.
- [x] Apply the component language to Models, Credentials, Channels, Telegram,
  Hub, Skills, Tools, Config and Logs without restructuring any route.
- [x] Refine the Flutter Run screen, Settings, Public Mode, Auto-Start,
  Telegram, Context Memory, GitHub, Models, What's New and About.
- [x] Correct the mark's optics so it stops reading as a crown below 24px, with
  the evidence and the rejected variants kept as an artifact.
- [x] Preserve the D-pad focus chain, the RTL drawer sides and the
  accessibility contract.
- [x] Hold all of it with tests: token integrity, light-mode integrity, token
  discipline, brand guards, mark-geometry consistency, mobile header structure,
  overflow accessibility and focus restoration, chat structure, RTL.

Not done here, and deliberately: the Android launcher icon was not re-rendered
from the refined geometry, and no Sol/GRIPLINE concept was read or implemented.

## Telegram Context Settings + Dashboard i18n — CLOSED 2026-09-04

Branch `feature/telegram-context-settings`, merged to `develop` with `--no-ff`.
Physically accepted on SM-A165F / Android 16 as vc35. `main` untouched, no tags
moved, no release.

- [x] Expose the Telegram context limit in Settings: 10 / 15 (Recommended) /
  20 / 25 / Custom, Custom accepting 5–50.
- [x] Give it an explicit localized Save button, disabled until the value is
  valid and changed, with one action producing one save.
- [x] Persist to `agents.defaults.telegram_recent_context_messages`, default 15.
- [x] Apply a saved limit to the next turn with no app and no Gateway restart.
- [x] Key the config cache on file identity so an atomic same-size replacement
  cannot be missed.
- [x] Prove live apply on the device: Custom = 17 held at `limit=17` across
  `history_total` 26, 28, 30, 32 and 34.
- [x] Resolve all twelve app locales in the embedded dashboard, mapping `pt`
  onto the existing `pt-BR` resource.
- [x] Preserve the pre-existing `bn-IN` and `cs` dashboard resources.
- [x] Take the Flutter locale over `?lng=`, ahead of the cached value.
- [x] Localize Models, Channels, Chat, Credentials, every `pages` namespace,
  Tour, Launcher Setup and Launcher Login.
- [x] Reach 907/907 in all thirteen non-English bundles with zero placeholder
  drift.
- [x] Replace the three hand-written language dropdowns with one shared
  selector listing every app locale by endonym.
- [x] Drive Arabic direction from `i18n.dir()` rather than a test for Arabic.
- [x] Clean up the pt-BR and zh values still byte-identical to English, and
  extend the English-copy gate to both with exact-key allowances.
- [x] Mirror the sidebar drawer in RTL from `i18n.dir()`; logical `border-e`
  for the inner edge.
- [x] Build vc35 (`0.2.0`, SHA-256 `fcec23a5...82077`) and confirm physical
  acceptance of Context Memory live apply, Arabic RTL and the RTL drawer.
- [x] Close the milestone: merge into `develop` with `--no-ff`.

Carried forward, deliberately not done here:

- [x] Rebuild an APK carrying `65dfc02`. Done by the What's New milestone: vc36
  and vc37 both include the pt-BR/zh cleanup, and vc37 is physically accepted.
- [ ] `TelegramOnboardingStrings` in the Flutter layer remains English in all
  twelve `.arb` locales. That is the native onboarding screen, not the embedded
  dashboard; it is tracked under Milestone D and is untouched by this work.

## What's New — CLOSED 2026-09-05

Branch `feature/whats-new`, from `develop` at `09790ac7`, merged with `--no-ff`.
Physically accepted on SM-A165F / Android 16 as vc37. `main` untouched, no tags
moved, no release. The feature branch is retained.

- [x] Add a What's New entry to the Settings header, immediately before About
  and styled to match it.
- [x] Open a full page with `Navigator.push`, not an `AlertDialog`.
- [x] Show a localized NEW badge only while the release notes are unread.
- [x] Key the seen state on `versionName` alone, never on versionCode or
  buildNumber, under `pocketclaw.whats_new.last_seen_version`.
- [x] Write the mark once the route is on screen — not on Settings open, not
  before the push succeeds.
- [x] Inject the seen store and the version loader so the badge is testable
  without a platform preference store or global mutable state.
- [x] Keep the release structure in typed Dart and every user-facing word in
  the ARB bundles.
- [x] Ship New / Improvements / Fixes for 0.2.0, advertising only what actually
  shipped and was accepted.
- [x] Translate all sixteen keys into all twelve app locales, with no English
  copies and no placeholder drift.
- [x] Lay the page out with `EdgeInsetsDirectional` so Arabic mirrors without a
  locale branch.
- [x] Absorb one new focus node into the existing D-pad chain:
  `What's New -> About -> public mode toggle`.
- [x] Build vc36 (`0.2.0`, SHA-256 `7e237ebb...f046`) and confirm physical
  acceptance of the What's New feature.
- [x] Fix the Settings header truncation vc36 exposed: replace the header `Row`
  with a `Wrap` so the title keeps its natural width instead of receiving only
  what the action buttons leave.
- [x] Guard it at 360x800 in four cases — English and Arabic, badge visible and
  badge seen — asserting the title's box is at least its intrinsic width.
- [x] Build vc37 (`0.2.0`, SHA-256 `7e617eb7...24e1`) and confirm physical
  acceptance of the header fix.
- [x] Close the milestone: merge into `develop` with `--no-ff`.

Carried forward, deliberately not done here:

- [ ] `TelegramOnboardingStrings` in the Flutter layer remains English in all
  twelve `.arb` locales. That is the native onboarding screen, not the embedded
  dashboard, and is untouched by this work.

## NEXT MILESTONE: PocketClaw Visual Identity / UI-UX Redesign — NOT STARTED

To be done on a **new branch cut from the updated `develop`**. Nothing in the
What's New milestone implements, stubs or prepares it. The What's New header
`Wrap` is a targeted truncation fix within the current visual style, not the
first step of the redesign.

## BACKLOG: not started, not scheduled

Recorded so they are not rediscovered. None of these is implemented, stubbed or
prepared, and none belongs to a milestone yet.

- [x] Remove the remaining user-visible PicoClaw branding in the Dashboard.
  Done in the APERTURE redesign, 2026-09-05.
- [x] Replace the ambiguous mobile sidebar-toggle icon with a clear menu /
  hamburger icon. Done in the APERTURE redesign, 2026-09-05.
- [ ] A dedicated Vision / Image Model routing option was built and then
  **rejected as out of scope** on 2026-09-05: PocketClaw stays at Default Model
  plus Fallback Models, and a user needing images picks a multimodal default.
  **Not implemented, not stubbed, not prepared.** Core's pre-existing
  `image_model` / `image_model_fallbacks` fields and `routeMediaTurn` are
  untouched and still config-file reachable; only the product surface was
  removed. Do not list Vision routing as a delivered feature.
- [x] Remove the deterministic lobster from the shared kernel identity prompt so
  the model stops signing replies with it. Done on
  `feature/final-user-facing-polish`, 2026-09-07.
  **PHYSICALLY ACCEPTED on SM-A165F / Android 16 as vc55, 2026-09-08.** A fresh
  neutral message produced a reply with no fixed sign-off. No output filter was
  added and normal emoji use is untouched.
- [x] Modernize the About dialog onto Aperture tokens and a responsive version
  layout. Done on `feature/final-user-facing-polish`, 2026-09-07.
  **PHYSICALLY ACCEPTED on SM-A165F / Android 16 as vc55, 2026-09-08**, in both
  English and Arabic.
- [x] Re-render the Android launcher icon from the refined APERTURE mark
  geometry. Done on `feature/final-launcher-icon`, 2026-09-07.
  **PHYSICALLY ACCEPTED on SM-A165F / Android 16 as vc54, then merged to
  `develop` with `--no-ff`.**

## FOLLOW-UP: Telegram request lifecycle durability

Found while auditing the queued-"Thinking…" incident on 2026-09-03. The
placeholder timing itself is fixed; these are the separate defects that audit
surfaced. None is a regression from that fix, and none should be folded into it.

- [ ] `/stop` from Telegram can be queued behind the very turn it is meant to
  cancel. `claimSessionMailbox` returns `queued` for a Telegram message on a
  busy session and the loop `continue`s before `tryHandleStopCommand` is ever
  reached, so the command lands in the FIFO instead of stopping anything.
- [ ] The per-session Telegram FIFO is unbounded and in-memory. There is no
  capacity limit, no backpressure and no TTL, and a Gateway restart loses every
  queued message.
- [ ] The Telegram update offset is not persisted. telego advances it in memory
  the moment an update is read, and confirms it to Telegram only on the next
  `getUpdates`. Depending on when the process dies, a restart either replays
  updates already answered or silently drops queued ones.
- [ ] `placeholderTTL` (10 minutes) deletes the placeholder record without
  editing or deleting the Telegram message, so the bubble is stranded and the
  answer arrives as a new message. Deferring the placeholder shrank the window
  but did not close it.
- [ ] `StopAll` does not reconcile outstanding placeholders, so every one still
  in flight is abandoned in the chat on shutdown.
- [ ] The Telegram long poll asks for a 30s server timeout under a 45s HTTP
  client cap, leaving only 15s of headroom.
- [ ] One turn ran 474,024 ms and hit `iterations_total=50`. That is agent and
  tool efficiency, not a queue defect; no wall-clock deadline and no change to
  `max_tool_iterations` was made for it.
- [ ] `editMessageText` still returns 400 for malformed HTML before the
  plain-text fallback succeeds. Recovered, not fixed.

The typing half of the incident is fixed on
`fix/telegram-queued-typing-lifecycle`; the persistent-placeholder half was
already fixed on `fix/telegram-deferred-thinking-placeholder`. Neither branch
touches anything above.

## FOLLOW-UP: normalize channel list-field serialization

Not part of the WhatsApp removal or the Web branding cleanup. Recorded here
because it was found while testing them, and deliberately left alone so those
branches stayed scoped.

- [x] Normalize channel list-field serialization consistently between the
  console, Go's `FlexibleStringSlice`, and Android provisioning.

The console's `serializeStringArrayForSubmit` joined list fields with newlines
and submitted a bare string, so saving any channel page turned `["a","b"]` into
`"a\nb"`. `2a2f3ce` fixed the console to write arrays and kept Go reading the
legacy string. The Android half is closed by deletion instead: there is no
Android provisioning any more, so only two layers remain and they agree.


## WhatsApp Self-Chat + Chat image attachment

- [x] Audit the two existing WhatsApp implementations before removing anything;
  record that both are load-bearing and keep them.
- [x] Replace the "WhatsApp" and "WhatsApp Native" catalog entries with one
  WhatsApp Self-Chat entry and remove every UI arm for the old two.
- [x] Store one value — the user's own number in canonical international form —
  and validate it identically in Go and TypeScript.
- [x] Build the Connect / Test / Change / Disconnect surface with no bridge URL,
  session store path, WebSocket URL, or QR session database.
- [x] Add the `whatsapp_self_chat` agent tool: configured number required,
  non-empty message required, bounded length, `OPENED` not `SENT`.
- [x] Reach Android from Core over an app-private request/reply directory, with
  a `FileObserver` in the foreground service; no port, no token.
- [x] Prefer regular WhatsApp, fall back to Business, report neither installed;
  declare both packages in `<queries>`.
- [x] Keep the number and the message body out of the logs, and prove it by
  capturing the log file during a real call.
- [x] Trace the Chat attachment failure to its actual cause rather than
  replacing the implementation.
- [x] Register `setOnShowFileSelector` and route the pick through Android's own
  photo picker / document picker, adding no storage permission.
- [x] Handle cancel, unsupported type, unreadable file, and an Activity torn
  down mid-pick without a crash or a phantom attachment.
- [x] Full regression gate green; forced-arm64 APK built with versionCode 14.
- [x] Physical acceptance, first pass: one entry, old entries gone, number
  configuration, Settings Test, prepared text, no auto-send, image picker, and
  the image returning to Chat — all PASS on SM-A165F / Android 16.
- [x] Close the one gap the device found: Connect / Change / Disconnect applied
  automatically through the existing safe config-apply path, never asking for a
  manual restart and never interrupting a turn.
- [x] Physical recheck: Connect, agent tool, Change, Disconnect, image
  attachment, and the GitHub `gh api user` regression — all PASS.
- [x] Merged `feature/whatsapp-self-chat` into `develop` with `--no-ff` and
  pushed. `main` untouched, no tags moved, no release created.


## Secure GitHub Authentication

- [x] Audit the existing architecture before editing: the Android bridge, Core
  launch environment, `Manager.Execute` injection, the gh and git runtime
  profiles, provider secret handling, Settings, and credential lifecycle.
- [x] Encrypt the GitHub token under a non-exportable Android Keystore key
  (AES-256-GCM, provider-chosen nonce, fail closed on any decryption failure).
- [x] Remove Core's `credentials/github_token` plaintext fallback; the runtime
  reads the credential from its environment and nowhere else.
- [x] Validate a candidate through Core's bundled gh over the loopback Android
  bridge, scrubbing the candidate out of anything gh reports.
- [x] Add the GitHub Settings card: connect, test, disconnect, no reveal.
- [x] Declare `allowBackup`, `fullBackupContent` and `dataExtractionRules`, and
  exclude the credential directory from cloud backup and device transfer.
- [x] Make validate-before-save structural: `connect` requires the login that
  only a successful `gh api user` can produce.
- [x] Apply connect and disconnect through the existing stop/start, queueing the
  restart when the service is mid-start rather than interrupting it.
- [x] Classify decryption failures: destroy only on a failed GCM tag, corrupt
  ciphertext, a malformed blob or a permanently invalidated key; preserve the
  ciphertext on transient platform failures. JVM unit tests cover the rule.
- [x] Classify GitHub validation failures (auth, connectivity, timeout,
  unavailable, other) from gh's stderr with `GH_DEBUG=1`, scrubbing the
  candidate and sending the detail to Debug Logs. A connection failure no longer
  reports the token as rejected.
- [x] Prove why gh cannot reach GitHub on Android: no `/etc/resolv.conf`, so
  Go's resolver falls back to `[::1]:53`. Reproduced over adb with `GH_DEBUG=1`
  and `GODEBUG=netdns=2`; curl succeeds in the same environment.
- [x] Fix Go DNS for gh: the payload carries the `pkg/androiddns` resolver,
  copied in by the build so there is one implementation, and the runtime hands it
  `PICOCLAW_DNS_SERVER` on the gh profile only. Verified over adb — a dummy
  credential now returns HTTP 401 from GitHub instead of a DNS failure. gh
  checksum repinned, catalog `2.3.0`, Core rebuilt.
- [x] Correct `install_payload`'s alignment guard: require a multiple of 16 KB
  rather than exactly `0x4000`, which rejected every Go payload including the
  device-verified Core.
- [x] **Physical acceptance** of secure GitHub auth — all twelve steps PASS on
  SM-A165F with a real token and a private repository, including disconnect
  removing auth from both gh and git, and `adb install -r` preserving the
  credential. Merged to `develop`.

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

- [ ] **Full PicoClaw → PocketClaw namespace / branding migration.** Carries the
  load-bearing internal identifiers — `libpicoclaw.so`, `libpicoclaw-web.so`,
  `PICOCLAW_*` placeholders, `/pico/*`, `.picoclaw`, the Go module paths, the
  JNI and method-channel names, the Kotlin `PicoClawApp` / `PicoClawService`
  classes — none of which is user-visible and each of which needs its own
  migration path. It also carries the last user-visible PicoClaw artwork:
  `assets/app_icon.png` and `assets/icon.ico` are still the orange lobster and
  are still the Windows/macOS icon source and the desktop window icon
  (`lib/main.dart`). The Android launcher milestone found them and deliberately
  left them; they are not Android launcher resources.

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
- [x] **Bootstrap architecture PHYSICALLY ACCEPTED as vc59 and CLOSED,
  2026-09-08**, merged to `develop` with `--no-ff`; baseline advanced 58 → 59.
  The upgrade experience remains open, below.
- [ ] **Non-blocking security review: the launcher/web console listens on
  `0.0.0.0:18800`.** Observed during vc59 machine validation. The Core gateway
  is correctly loopback-only on 18790; the console is not. This predates the
  bootstrap branch and was not introduced by vc59, so it was deliberately not
  fixed in that closeout. Decide whether the console should bind loopback-only,
  and what that costs anyone reaching it from another device on the network.
- [~] **Bootstrap architecture implemented 2026-09-08 on
  `feature/bootstrap-architecture`; the upgrade experience is still open.**
  PocketClaw-managed capability guidance no longer lives in `AGENT.md`: it ships
  in the binary as the `capability.managed_runtime` prompt part, so every
  install receives it on upgrade with no file to migrate. `.pocketclaw/
  bootstrap.json` records the digest of what seeding actually wrote for
  `AGENT.md`, `SOUL.md`, `USER.md` and `memory/MEMORY.md`, which is what makes
  "has the user touched this?" answerable later. **Still open:** what to do when
  a tracked template is pristine and its bundled default has since improved —
  offering the delta, or asking. `bootstrap.UserOwns` is the predicate that
  decision must consult; it has no production caller until that work is done.
  `MEMORY.md` is out of scope permanently, by design.
  Original entry, kept for the reasoning:
- [ ] **DEFERRED — Release Hardening / post-stable architecture:** a versioned,
  non-destructive bootstrap update strategy for existing installs. Embedded
  workspace templates are seeded with `keepExisting=true`, so `AGENT.md`,
  `SOUL.md`, `USER.md` and `MEMORY.md` are written once and never refreshed. An
  install created before a default was improved keeps the old text forever: the
  device validated on 2026-09-08 still carries an `AGENT.md` predating the
  Managed Runtime guidance, so that agent has never been told the Managed
  Runtime exists. Any fix must preserve user edits rather than overwrite the
  file — track a template version, merge or offer the delta, or surface it as a
  prompt the user can accept. Do not blindly overwrite, and do not rewrite user
  memory. Found during the vc55 lobster-signature investigation; not fixed
  there.
- [x] **Release Hardening A1: signing fails closed, Core secrets leave Android
  backup, the version is tracked, and unused analytics surface is gone.** Done on
  `feature/release-hardening-a1`, 2026-09-08, with a review follow-up the same
  day correcting the debug-signing wording, making the version floor advance
  from `android/release-baseline.properties`, and removing the Firebase
  advertising surface while keeping the feature.
  **PHYSICALLY ACCEPTED on SM-A165F / Android 16 as vc56, 2026-09-08, then
  merged to `develop` with `--no-ff`.** The upgrade preserved its signing
  identity, install time, dataDir, uid and application data, and dropped six
  permissions without adding any.
- [ ] **Advance `lastAcceptedVersionCode` in `android/release-baseline.properties`
  whenever a build passes physical acceptance**, in the same commit that records
  the acceptance in `PROJECT_STATE.md`. It is the floor every later build is
  checked against; leaving it behind makes the check meaningless. Done twice so
  far: 55 to 56 at the A1 closeout, and 56 to 58 at the A2 closeout — jumping 57,
  which was superseded before acceptance. A candidate that is never accepted
  never becomes the floor.
- [ ] **Produce the production signing key and switch to it.** Deliberately not
  done in A1: the test device runs a debug-signed install and changing signers
  forces an uninstall and a data reset. Until then every artifact must be built
  with `-PallowDebugSigning=true` and is not releasable.
- [x] **Decide whether Firebase Analytics belongs in the default build.**
  Resolved in the A1 follow-up, 2026-09-08: **kept.** `firebase_analytics` and
  `firebase_core` back the device-feedback feature, which has a Settings toggle
  and its own telemetry state machine; it is unconfigured by default rather than
  unused, so removing the plugin would delete a feature to shorten a permission
  list. The advertising surface it contributed — `AD_ID`, both
  `ACCESS_ADSERVICES_*` and the Play install-referrer permission — is removed
  with Google's documented `tools:node="remove"` opt-out instead. Custom event
  logging, which is all this feature does, is unaffected.
- [ ] **If Play campaign attribution ever becomes a requirement**, restore
  `com.google.android.finsky.permission.BIND_GET_INSTALL_REFERRER_SERVICE` in
  `AndroidManifest.xml`. It was removed because PocketClaw does no install-source
  attribution, not because it conflicts with anything.
- [x] **Release Hardening A2: the gateway credential, the diagnostic log and the
  Dashboard credential verifier leave shared storage, secret redaction covers
  provider credentials, and realtime authentication is constant-time.** Done on
  `feature/release-hardening-a2`, 2026-09-08.
  **PHYSICALLY ACCEPTED on SM-A165F / Android 16 as vc58, 2026-09-08, then
  merged to `develop` with `--no-ff`.** The existing Dashboard password survived
  the verifier migration, Status and Logs both pass, and the upgrade preserved
  install time, dataDir, uid and application data. vc57 was superseded before
  acceptance, so the baseline advanced directly 56 → 58.
- [x] **Dashboard credential database on shared storage.** Found by source audit
  after vc57, fixed in A2 before close: `launcher-auth.db` holds only a bcrypt
  verifier, but on shared storage it can be *written*, and replacing the verifier
  yields a working Dashboard login without breaking bcrypt.
  `PICOCLAW_DASHBOARD_AUTH_DIR` moves it to app-private no-backup storage, with a
  crash-safe one-time migration that preserves the existing password.
  **PHYSICALLY ACCEPTED as vc58, 2026-09-08:** the shared database and every
  exact sidecar were retired on first start, and the user's existing Dashboard
  password still authenticated.
- [ ] **RECHECK AFTER THE NAMESPACE MIGRATION: every A2 contract is keyed on a
  compatibility name.** `.picoclaw.pid`, `PICOCLAW_GATEWAY_TOKEN_FILE`,
  `PICOCLAW_LOG_DIR`, `PICOCLAW_DASHBOARD_AUTH_DIR`, the legacy
  `launcher-auth.db` filename the migration matches by name,
  `PICOCLAW_CHANNELS_PICO_TOKEN`, the `picoclaw` private
  directory the backup rules exclude, and the `pico` channel name. Renaming one
  side alone puts the credential, the logs or the backup exclusion back where
  they were without breaking anything visible.
- [x] **Release Hardening A3: deterministic Core builds and a release gate.**
  Done on `feature/release-hardening-a3`, 2026-09-08. `tool/release_gate.py` is
  the single authority on whether an artifact is releasable, with explicit
  `test` and `production` signing classes and a JSON release manifest.
  **CLOSED, 2026-09-08, merged to `develop` with `--no-ff`.** No physical
  candidate and no version bump: A3 changes build and release engineering, not
  runtime behaviour. The first deterministically-built Core is staged
  (fingerprint `0f601437…`, BuildTime `2026-09-08T05:36:42+0000`), and
  `--verify-source` exits 0 with no unexplained failures.
- [ ] **Run the full release gate in CI.** `.github/workflows/release-gate.yml`
  currently invokes only `--verify-source --no-tests`, which needs just Python
  and binutils. The delegated suites need the Go toolchain, the Flutter SDK and
  the Android SDK; installing and pinning those is its own piece of work rather
  than a side effect of adding CI. The workflow must keep invoking the
  repo-local gate rather than reimplementing any of it.
- [ ] FINAL RELEASE HARDENING: controlled Dart generated-source URI strategy.
  Confirmed present in the vc58 artifact by the A3 release gate: the packaged
  `libapp.so` embeds one generated-source URI. The gate reports it as
  `PENDING_FINAL_HARDENING` rather than failing on it, so it stays visible
  without blocking every build.
- [ ] Future milestone only: Background & Battery page.
- [ ] Future milestone only: local Runtime / Statistics bottom tab.

## DEBUG log cleanup micro-pass

- [x] Trace exported `53.616�s` to Android Export Logs truncating Dart UTF-16
  `codeUnits` into bytes; confirm Go output, Kotlin string transport, sanitizer,
  queue, and Logs UI are not the corruption layer.
- [x] Encode Android export content as UTF-8 and test the real MethodChannel
  transport with `µ`, Arabic, emoji, mixed text, punctuation, and surrounding
  ANSI sequences; strict decoding introduces no U+FFFD.
- [x] Suppress only exact successful 2xx `GET /api/gateway/logs` and
  `GET /api/gateway/status` middleware DEBUG events; preserve failures,
  unexpected methods, unknown routes, config/models, and other diagnostics.
- [x] Preserve and rerun exactly-once queue, caller sanitization, branding, and
  terminal-control regressions.
- [x] Pass Flutter analyze/97 tests, frontend 36 tests/tsc/lint, tagged Go
  logger/gateway/API/middleware suites, Core provenance/build, APK guard, and
  artifact scans.
- [x] Build replacement ARM64 candidate `eacbbc86...423f9a8` (34,239,073 bytes).
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** export one new DEBUG log and verify
  valid `µ`, zero PocketClaw-introduced U+FFFD, intact Arabic/emoji/punctuation,
  continued terminal cleanup, and no routine successful self-poll noise.
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** verify failed polls and real API
  requests remain visible, then continue every standing device checklist item.

## Web Console log parity / internal identifier visibility

- [x] Trace Web Console startup boxes to the Core CLI's block-art no-color
  banner and the React Logs route's SGR-only renderer over a raw log ring.
- [x] Store the shared Unicode-safe plain-text representation at
  `LogBuffer.Append`; apply the same canonical fixtures to native/export and
  the browser's idempotent legacy/raw guard.
- [x] Replace captured no-color block art with one `PocketClaw` text line and
  force captured gateway launches to `--no-color`.
- [x] Remove the gateway executable/library path from the startup event.
- [x] Suppress exact successful `GET /pico/ws` 101/2xx events; retain failures
  and unexpected methods with `/internal realtime connection` wording.
- [x] Prove the real Logs page preserves Arabic, emoji, and `53.616µs`; removes
  ANSI/control data; hides routine compatibility traffic/path leakage; and
  still renders a genuine 500.
- [x] Pass Flutter analyze/99 tests, frontend 37 tests/tsc/lint, tagged Go
  logger/gateway/API/middleware/CLI suites, Core build/provenance, and APK guard.
- [x] Build ARM64 candidate `3e138b4a...170adc` (34,241,381 bytes).
- [ ] **PRE-RELEASE BLOCKER — PHYSICAL:** verify Native Logs, Export Logs, and
  Core Web Console Logs all show the same clean Unicode representation, with
  no boxes/ANSI, routine `/pico/ws`, internal library path, or unintended
  upstream branding, while genuine errors remain visible under neutral wording.

## Final enabled-channel brand micro-fix

- [x] Confirm internal `pico` means the singleton authenticated Web Console
  WebSocket/media transport, not a user-facing product label.
- [x] Map only exact `config.ChannelPico` to `pocketclaw` at the startup/reload
  enabled-channel summary; preserve all internal identifiers and input data.
- [x] Add targeted regression coverage for `[telegram pocketclaw]`, internal
  identity preservation, and no substring/global replacement.
- [x] Pass relevant tagged Go suites, regenerate the 110-file provenance patch,
  rebuild Core, verify zero paths, and build guarded ARM64 APK
  `aab3c565...25b3582`.
- [ ] **FINAL PHYSICAL CHECK:** restart Core and confirm the enabled-channel
  summary shows `pocketclaw`, with the physically passed Web log cleanup and
  8/8 skills / 17 tools unchanged.

## Final user-visible caller brand fix

- [x] Keep internal package/file/channel/routes/config identity unchanged.
- [x] Map exact structured logger component `pico` to `realtime` and caller
  basename `pico.go` to `realtime.go`, preserving line numbers.
- [x] Prove Native/Export, Web `LogBuffer`, and real React Logs page parity via
  the canonical fixture; prove substring names and `pico_client` are unchanged.
- [x] Pass Flutter analyze/99 tests, frontend 37/tsc/lint, relevant tagged Go
  suites, Core provenance/build, zero-path scan, endpoint/hash checks, and guard.
- [x] Build candidate `1eeca7c9...ad089f7` (34,242,865 bytes).
- [ ] **FINAL PHYSICAL GATE:** verify all three user-visible log surfaces show
  `realtime realtime.go:<original line>` and preserve every prior physical PASS.

## Web Console Logs scroll/jitter micro-pass

- [x] Trace the real page to passive post-paint bottom correction and a
  content-height-driven JavaScript hard-wrap measurement loop.
- [x] Preserve live polling while following the bottom only when the user was
  within 24 px; perform the correction before paint.
- [x] Preserve a scrolled-up `scrollTop` without programmatic writes.
- [x] Give events stable `run_id:absolute_offset` identities and memoize rows.
- [x] Replace content-measured hard wrapping with deterministic browser-native
  wrapping of the unchanged sanitized text.
- [x] Cover bottom, scrolled-up, long-row node identity, and repeated no-new-log
  cases in the real Logs page DOM suite.
- [x] Pass frontend 41/41, tsc/lint, relevant Go and Native/Export regressions,
  regenerate 113-file provenance, rebuild zero-path Core, and build guarded APK
  `be5d7cbb...5070fc96`.
- [ ] **FINAL PHYSICAL GATE:** verify several minutes at bottom and scrolled up
  show no Web Logs shake, rewrap, or forced scrolling; preserve all prior PASS.

## Telegram token Web-log redaction security pass

- [x] Confirm case A: the full Telego URL was partially masked before stdout
  and Web storage; only retained fragments reached stored/rendered Web logs.
- [x] Replace partial masking at the same pre-stdout boundary with complete
  credential redaction, including encoded/bare and Authorization forms.
- [x] Normalize Web Bot API URLs before `LogBuffer` storage to operation-only
  wording while preserving method, status/error, timeout, and latency.
- [x] Add an idempotent React guard for raw/historical partial lines.
- [x] Cover standard/arbitrary methods, successes, 5xx, timeouts, stored
  history, real DOM rendering, public metadata, and fragment-negative checks.
- [x] Leave Native/Export source, Telegram lifecycle/onboarding, and credentials
  unchanged; pass the existing Native/Export 7/7 regression.
- [x] Pass relevant Go suites, frontend 42/42/tsc/lint, regenerate 115-file
  provenance, rebuild zero-path Core, and build guarded APK
  `8257e9f0...7c7050fe`.
- [ ] **FINAL PHYSICAL GATE:** exercise Telegram DEBUG calls and failures; Web
  Logs must show useful operations/errors with zero credential fragments.

## Final legacy brand visibility sweep

- [x] Trace each physical string to structured ChannelPico fields, the exact
  protocol lifecycle message, the exact internal webhook path field, or the
  compatibility PID filename.
- [x] Normalize exact channel/type fields and classified messages at the shared
  Web pre-storage boundary with an idempotent React guard.
- [x] Preserve the security warning and genuine realtime failures with neutral
  semantic wording.
- [x] Prove all internal channel/config/routes/files/libraries remain unchanged
  and reject substring/global rewrites with explicit negative fixtures.
- [x] Scan a representative stored full startup and real Logs DOM: unintended
  user-visible `pico`/`picoclaw`/`sipeed` occurrences = 0.
- [x] Pass frontend 44/44/tsc/lint, relevant tagged Go including Skills,
  Flutter analyze/99 tests, 115-file provenance, zero-path builds, and guard.
- [x] Build ARM64 candidate `309f6d7a...a5f3030` with Core hashes
  `49f89ae2...be656f` and `98f3fa9d...08bae`.
- [ ] **FINAL PHYSICAL GATE:** restart Core in DEBUG and confirm the observed
  startup/security/realtime/PID lines are brand-safe while all earlier physical
  passes remain intact.

## Telegram DEBUG final cleanup

- [x] Prove Telego and the pre-stdout redactor preserve raw `Err: [<nil>]`.
- [x] Trace corruption to the Web pre-storage orphaned-CSI fallback removing
  `[<n` and persisting `il>]`.
- [x] Restrict orphaned control cleanup to numeric CSI remnants and preserve
  ordinary angle brackets, Unicode, Arabic, emoji, and HTML-safe text rendering.
- [x] Display successful Telego nil fields as `Err: none` consistently.
- [x] Suppress routine `getUpdates` calls and exact successful empty responses
  before stdout; retain failures, non-empty updates, API errors, send/edit, and
  lifecycle events.
- [x] Prove repeated empty polls add zero stored history and all retained lines
  remain credential-fragment-free.
- [x] Pass frontend 45/45/tsc/lint, relevant tagged Go, Flutter analyze/101,
  115-file provenance, zero-path builds, and permanent APK guard.
- [x] Build ARM64 candidate `2c00720a...2da27c` with Core hashes
  `7c1d3918...38ef1f` and `1611b6e1...d09256`.
- [ ] **FINAL PHYSICAL GATE:** observe several empty long polls, one non-empty
  update, and a recoverable failure in DEBUG; then reconfirm all prior passes.

## Agent DEBUG privacy + final branding + Telegram payload privacy

- [x] Prove the legacy assistant name was actual freshly generated system-prompt
  content and update only the PocketClaw default identity.
- [x] Remove normal logs of prompt bodies, full messages/tools JSON, raw model
  reasoning, and tool arguments while retaining useful lifecycle metadata.
- [x] Redact exact session/internal structured fields before writers without
  mutating runtime routing values or using global/substr replacement.
- [x] Normalize Telego requests/results before stdout and backend storage;
  retain operation/status/count/type/error metadata without Telegram PII or
  message bodies.
- [x] Add backend, real Logs DOM, Core logger/Agent/Telegram, shared fixture and
  Native/Export defense regressions.
- [x] Pass relevant tagged Go suites, frontend 46/46/tsc/lint, Flutter
  analyze/101, regenerate 124-file provenance, rebuild zero-path Core, and pass
  the permanent APK guard.
- [x] Build ARM64 candidate `46ca983a...2b908e` with Core hashes
  `8ed15601...9be24a` and `21001004...5fc1d7`.
- [x] **FINAL PHYSICAL GATE:** run a fresh realtime plus Telegram conversation
  in DEBUG and confirm only safe metadata appears while Skills 8/8, Tools 17+,
  delivery, viewport, Unicode, branding and earlier polling fixes remain PASS.
  PASSED on a real ARM64 device on 2026-08-29 against APK
  `5760247a...2c186b9`.

## Phase 2 — v0.2.0-rc1 Release Candidate

- [x] Make owner authorization server-derived on the internal realtime channel
  so payload fields and a stale allowlist cannot select the effective identity.
- [x] Replace the single process-wide dashboard cookie with a revocable
  server-side session store and require a same-origin realtime upgrade.
- [x] Fail closed on Telegram unless exactly one paired numeric owner exists,
  across the Android bridge and manual onboarding.
- [x] Fail closed on credential generation when the CSPRNG is unavailable and
  give the Android host a per-installation realtime credential.
- [x] Pin the managed Core gateway to loopback unconditionally.
- [x] Add the Dashboard listener supervisor so Public Mode rebinds only 18800,
  with rollback on bind failure and persistence only after success.
- [x] Advertise a real LAN address from an active Wi-Fi/Ethernet link and
  refresh the connect URL and QR without a service restart.
- [x] Physically validate the candidate on a real ARM64 Android device.
- [x] Prove both Core binaries reproduce byte for byte from the committed
  source with their build timestamps pinned.
- [x] Pass Flutter analyze/114, frontend 46/46 + tsc + lint, and the complete
  Go suite, build, and vet under `-tags goolm,stdjson`.
- [x] Bump the app version to `0.2.0+4` for the candidate.
- [x] Merge to `develop`, tag `v0.2.0-rc1`, and publish a GitHub pre-release.

### Deferred — recorded, not implemented

- [ ] Statistics / Runtime page: uptime, agent active/idle/working state, input
  and output token counts, session/day/lifetime totals only where truthful,
  CPU/RAM/Core usage, request counts, success/failure, average latency,
  provider/model, channel states, Core restart count and last restart, optional
  charts. Local telemetry and provider-reported tokens only — never invent a
  statistic.
- [ ] Background and battery settings: battery-optimization detection, a route
  into Android battery settings, Samsung Sleeping/Deep Sleeping Apps guidance,
  and a plain reliability explanation. No silent battery-exemption claims;
  Play policy review comes later.
- [x] Service/Gateway auto-start: configurable "start the service
  automatically" and "start the Gateway automatically". Shipped in
  `v0.2.0-rc2`, PHYSICAL PASS on 2026-08-30. Built on `v0.2.0-rc1` rather than
  on the abandoned `feature/autostart-foundation` experiment, which was never
  merged. The optional restart of the Gateway when it stops unexpectedly is
  deliberately NOT part of this work and remains deferred — a restart policy
  belongs to a reliability milestone, and mixing it into auto-start is what made
  the experimental branch fail physical acceptance.
- [x] Managed runtime and tool dependencies. The foundation is implemented,
  physically validated and merged; see the milestone section below. The
  original plan here — an app-private `runtime/bin` — is **invalid** and was
  replaced: PocketClaw targets SDK 36, and an app targeting API 29+ cannot
  execute a file in its own writable storage. `gh` and `git` remain unshipped;
  see the milestone section for why each is hard.
- [ ] Secure API-key storage: migrate provider secrets to the Android Keystore.
- [ ] Final production hardening, deferred until the final release: Flutter and
  Dart obfuscation, R8/ProGuard, symbol stripping, release-only hardening,
  anti-reverse-engineering protection, config/secret exposure minimization,
  APK/AAB inspection, and generated Dart source URI cleanup.
- [ ] Fix the shared Kill+Wait cleanup pattern in `gateway_test.go` so the
  `-race` gate on `web/backend/api` is green. Pre-existing on `develop`; the
  race is in the test harness, not in production code.

## Phase 2 — Managed Runtime Foundation

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. **PHYSICAL PASS** on a real ARM64 device 2026-08-30,
then merged to `develop`.

Physical catalog result: **43 of 44** tools available; `traceroute` correctly
reported unavailable. Bundled: jq 1.7.1. The writable-app-data probe returned
INCONCLUSIVE on that device, which changes nothing — the architecture never used
writable executable storage. Tools 18, Skills 7/7.

- [x] Establish the Android execution model. Executables reach the device only
  through `/system/bin` or through APK payloads the installer unpacks into
  `nativeLibraryDir`. Writable app-private storage is not executable on
  targetSdk 36, so no provisioning subsystem exists; see `DECISIONS.md`.
- [x] Runtime registry and versioned manifest in `core/src/pkg/pcruntime`, with
  per-tool metadata, integrity class, timeout profile and output bounds.
- [x] Tool resolver that measures the device instead of trusting the catalog,
  with SHA-256 and ELF/ABI verification for bundled payloads and per-tool
  resolution locks so concurrent callers share one verification pass.
- [x] Bounded execution API: direct argv with no shell, constructed environment,
  workspace-confined working directory, timeout ceiling, cancellation with
  process-group termination and reaping, bounded stdout/stderr with truncation
  markers.
- [x] Structured lifecycle logging for resolve, verify, exec, cleanup, probe and
  inventory, with redaction inside the emitter and tool output never persisted.
- [x] Read-only runtime inventory API, suitable for a future Runtime/Statistics
  page. The page itself is still deferred.
- [x] On-device execution probe that measures whether writable app storage is
  executable and reports the verdict in the Debug Logs.
- [x] `runtime` agent tool (list / info / run). Agent tool count moves 17 -> 18;
  all 17 existing tools are unchanged.
- [x] Runtime Pack v1: Tier 1 catalogued as system-provided and probed;
  jq 1.7.1 cross-built and bundled as `libpocketclaw-jq.so`, proving the
  packaging contract end to end.
- [x] Agent guidance in `RUNTIME.md`, `core/src/workspace/AGENT.md`, the
  `runtime` tool description, and the GitHub Skill.
- [x] PHYSICAL validation on a real ARM64 device. **PASS**, 2026-08-30.

### Not in this milestone, deliberately

- [ ] `curl`, `wget`, `openssl`: no system binary exists and each needs an NDK
  cross-build with a TLS stack.
- [ ] `git`: C, and it `exec`s helpers from a `libexec` layout that
  `nativeLibraryDir`'s flat `lib*.so` namespace cannot represent.
- [ ] `gh`: pure Go and easy to cross-compile, but roughly 40 MB and of little
  use without `git`.
- [ ] Runtime / Statistics UI. The inventory API exists; the page does not.

### Deferred release engineering

- [ ] Version source-of-truth cleanup. `android/local.properties` holds
  `flutter.versionCode` and is gitignored, so a stale value there ships an
  apparently successful build with the wrong version. `pubspec.yaml` does not
  reach the canonical Gradle release.
- [ ] The arm64 native-payload guard runs from a `packageRelease` `doLast` block
  and may not execute when Gradle considers packaging UP-TO-DATE, so a bad
  payload could pass unchecked on an incremental build.
- [ ] `PicoClawService.extractBinaryFromApk()` copies a payload into `filesDir`
  and calls `setExecutable(true)`. **High priority.** On targetSdk 36 that path
  cannot work: the file is written, the mode is set, and `ProcessBuilder.start()`
  then fails with `EACCES`, so the failure reads as a mysterious launch error
  rather than an unsupported delivery model. It is unreachable today because
  `nativeLibraryDir` always resolves first. Do not fix it inside the Managed
  Runtime milestone unless it becomes a blocker.
- [ ] Production release signing. RC2 and this branch are debug-signed for
  sideload pre-release use.
- [ ] Final obfuscation and hardening.

## Phase 2 — Lean Runtime Pack v2

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`, from `develop` at
`fa27ad2`. **PHYSICAL PASS** on a real ARM64 device 2026-08-30, then merged to
`develop`.

Physical: Git HTTPS PASS, `git clone` of a public GitHub repository PASS, **Git
helper symlink execution on Android PASS**, git 2.51.0, gh 2.82.1, curl HTTPS
PASS, ripgrep PASS, sqlite3 PASS. Skills 7/7 on an existing upgraded workspace,
6/6 on a fresh install.

Six bundled tools ship: git 2.51.0 with its transport helper, gh 2.82.1,
curl 8.11.1, ripgrep 14.1.1 and sqlite3 3.50.4, alongside the jq from v1.
APK 34.7 MB -> 58.3 MB. Six more system entries were added at zero cost.

The goal is maximum Agent capability per megabyte. PocketClaw is not becoming a
Linux distribution: no apt, no proot, no Python or Node runtime, no compiler
toolchain, no background package manager. Android's own system tools plus a small
number of high-value bundled executables, with native Go capability preferred
wherever it is lighter than a binary.

### Tier A

- [x] Git over HTTPS. Packaged as `libpocketclaw-git.so` plus
  `libpocketclaw-git-remote-http.so`; the runtime presents the helper under its
  logical name through a symlink directory pointed at by `GIT_EXEC_PATH`.
- [x] GitHub CLI (`gh`). 55.9 MB installed, accepted as a sanctioned exception.
- [x] HTTP/TLS: real curl 8.11.1 against mbedTLS, 1.30 MB. The native-Go
  alternative was rejected once curl proved this cheap.
- [x] ripgrep 14.1.1, 4.27 MB.
- [x] sqlite3 3.50.4, 1.23 MB, built with `SQLITE_OMIT_LOAD_EXTENSION` so it
  cannot load a shared library the runtime never verified.
- [ ] `yq` — measured at 11.25 MB, above the size policy, and jq already covers
  JSON. Left out; revisit only if YAML handling proves to matter.

### Tier B — probed rather than bundled

- [x] `zip`, `unzip`, `diff`, `patch`, `file`, `tree` added as `system` catalog
  entries at zero cost. Shipping the entry *is* the probe: the physical run
  reports which the platform provides. Bundle one only if a device is shown not
  to have it.

### Tier C — deferred

OpenSSH suite (reconsider after HTTPS Git is stable), `rsync`, Python, Node,
npm, a full bash runtime, `make`, compilers, ffmpeg, ImageMagick.

### Known limitation carried forward

- Git features that depend on a shell — hooks in particular, and git's `ENOEXEC`
  fallback — are **not guaranteed on Android**. git compiles in `/bin/sh`, which
  the platform does not have, and overriding `SHELL_PATH` breaks git's own
  cross-build. Clone, fetch and push do not need a shell.

### Physical validation results

- [x] `git clone` over HTTPS failed on the first physical run with `unable to
  find remote helper for 'https'`. Root cause: git spawns `git remote-https` and
  resolves the literal name `git` through PATH; the helper directory held only
  the two remote helpers. Fixed by declaring `git` as its own helper. Not TLS,
  not symlinks.
- [x] **`symlink_exec` — PASS on hardware.** Android permits executing through a
  symlink in app-private storage that points at a packaged payload in
  `nativeLibraryDir`. The kernel resolves the link and runs the read-only
  packaged file, so the API 29+ restriction on executing writable storage does
  not apply. This is what makes multi-executable tools possible on Android, and
  it is now evidence rather than reasoning. The probe still reports it, so a
  device that behaves differently will say so.
- [x] GitHub Skill removed from the seeded workspace. Note that the seeded set is
  now **6** skills, not 7: `picoclaw-agent` is deliberately unseeded. An existing
  device keeps its 7 because seeding only writes and never deletes.
- [ ] Everything else in the physical acceptance list below.

### Rules that carry over

- Delivery classes stay `system`, `bundled`, `unavailable`. No executable
  downloading or provisioning, no package marketplace, no URL-to-executable flow.
- Every bundled executable needs an exact version, trusted source, license,
  ARM64 build method, and SHA-256 recorded before it ships — licensing is
  reviewed first, not after.
- Any single tool adding more than roughly 10 MB installed needs explicit
  justification. Git and `gh` are the deliberate exceptions.
- Per-tool timeout profiles, not one global timeout: `git clone` and
  `gh release upload` are not utility commands.
- Bounded output stays mandatory; Git logs and diffs are unbounded by nature.
- Never put a credential in argv. No `https://TOKEN@github.com/...`.

## Phase 2 — Provider Resilience & Automatic Failover

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`.
**PHYSICAL PASS** on the target ARM64 device 2026-08-30, then merged to
`develop`. Commits `68443c1`, `7b67493`, `ebf49b4`, `812a003`, `3446b0f`.

Physical: automatic failover PASS, Fallback Models UI and ordered selection PASS,
failing primary answered by its configured fallback PASS, one user-visible final
answer PASS, automatic gateway restart after model and fallback changes PASS,
active-turn safety PASS, white-screen resume recovery PASS with no loop.

Deliberately not built: checkpoint subsystem, semantic tool fingerprinting,
side-effect classification framework. The loop already held the invariant; it is
protected by tests plus one exact-`toolCallID` reuse guard.

A provider that rate-limits, times out, or returns nothing should degrade into a
retry or a fallback, not into a failed turn the user has to notice and repeat.

### Detection

- [x] HTTP 429, with `Retry-After` parsed in both legal forms and honoured.
- [x] HTTP 502, 503 and 504, classified by what each actually means rather than collapsed into timeout.
- [x] Provider timeouts.
- [ ] Empty model responses, but **only** where the emptiness is attributable to
  provider failure. A model that legitimately returns nothing must not be
  retried as though it had errored.

### Response

- [x] Bounded retries, capped at one same-candidate attempt when a fallback exists.
- [x] Provider cooldown, now fed by single-candidate failures too.
- [x] Automatic fallback to the configured backup model or provider.

### Correctness under retry — the hard part

- [x] Preserve completed tool-call results across a retry or failover.
- [ ] **Never blindly re-run a tool that already succeeded and had side effects.**
  A retry that re-sends a message, re-pushes a commit, or re-writes a file is
  worse than the failure it is recovering from. This constraint, not the
  detection, is what makes the milestone non-trivial.
- [x] Done without a checkpoint subsystem, fingerprinting or side-effect
  classification: the loop already provided the property. Guarded by tests and a
  `tool_call_id` reuse check.

### Observability

- [x] `provider.*` lifecycle events, distinguishing configured name, provider, upstream model and protocol.
- [x] No provider secrets in logs; redaction lives in the emitter, not at call sites.

### Deferred out of Provider Resilience, deliberately

- [ ] Cross-provider context-overflow fallback. Choosing a fallback for a
  context overflow needs the alternate model's context capacity, and no reliable
  per-model metadata exists — `ContextWindow` is an agent default, not a model
  property. Failing over on a guess would overflow again having paid the
  latency. Compact-and-retry on the current candidate is unchanged and still
  correct.
- [ ] Automatic-failover user settings (on/off, ordered fallback list, retry
  toggle, maximum fallback attempts). The config shape already supports ordered
  fallbacks; only the UI is deferred.

## Phase 2 — Fallback models UI and automatic gateway restart

Branch `feature/provider-resilience-failover`, on top of `68443c1`.
Not merged; physical validation PENDING.

- [x] Fallback Models section on the Models page: add, remove, reorder, save,
  reload. Candidates selected from configured model entries.
- [x] `POST /api/models/fallbacks` with validation: unknown entry, duplicate,
  virtual model, non-chat model, and self-reference all rejected. Empty list
  valid; existing configs unaffected.
- [x] Fallbacks stored as references by model name, so each keeps its own
  provider, credentials and base URL.
- [x] Automatic gateway restart after a restart-requiring save, reusing the
  existing `gateway_restart_required` signature decision.
- [x] Restart deferred while the gateway is busy, via new `active_requests` and
  `busy` fields on Core's `/health`.
- [x] **Unknown busy state never forces a restart.** A running gateway that will
  not report its state is retried for five seconds, then the change is left
  saved and unapplied.
- [x] **The two-minute cap bounds the wait, not the user's work.** Reaching it
  never restarts a busy gateway; the change stays saved and the manual Restart
  Gateway action applies it.
- [x] Restart coalescing at both the frontend and the launcher.
- [x] Readiness confirmed by signature match, not by the restart call returning
  200. Failure keeps the saved config and leaves the manual control available.
- [x] PHYSICAL validation. **PASS**, 2026-08-30.

### Deferred, deliberately

- [ ] Retry-count sliders, cooldown controls, Retry-After settings, per-error
  policy, provider health dashboard and fallback statistics. The milestone needs
  only presence-of-list plus ordered selection.
- [ ] Automatic re-application once the gateway later goes idle. It would mean a
  background worker restarting the gateway at a moment the user did not choose,
  which is the surprise the Auto-Start milestone was built to avoid. The
  restart-required indicator stays visible and the next save or the manual
  action applies the change.
- [ ] `raw-config-page`, `config-page` and `channel-config-page` still show
  "restart required" rather than applying automatically. Out of scope here.

## Phase 2 — Resume white-screen recovery

Branch `feature/provider-resilience-failover`, on top of `ebf49b4`.
Not merged; physical validation PENDING.

- [x] Resume lifecycle observer on the Android WebView, which had none.
- [x] Liveness probe: readiness flag plus non-empty `#root`, asked on resume.
- [x] Conditional recovery only — a healthy page is never reloaded.
- [x] Route preserved across recovery rather than returning to home.
- [x] Recovery capped at one attempt per page load; no loop.
- [x] React error boundary so a crash shows a reload affordance instead of an
  empty page, and clears the readiness flag.
- [x] `[webview]` lifecycle logging that separates renderer death from a console
  crash. No page contents or secrets.
- [x] PHYSICAL validation. **PASS**, 2026-08-30: normal resume does not
  reload, automatic recovery works, no recovery loop observed. The underlying
  cause remains unconfirmed — recovery works for both candidates, and the
  `probe_failed` / `page_unresponsive` split is what would settle it.

### Known limitation

`webview_flutter_android` 4.14.0 exposes no `onRenderProcessGone` callback, so
renderer death cannot be observed directly. The probe is the substitute. If the
physical logs confirm renderer death is the cause, a plugin upgrade or a native
`WebViewClient` override would allow reacting at the moment it happens rather
than at the next resume.

## Phase 2 — Python Lite Runtime

Branch `feature/python-lite-runtime`, from `develop` at `b46921e`.
**Architecture review only — nothing implemented.**

The appeal is capability per megabyte: scripting, parsing, JSON, CSV, XML,
regex, calculation, file transformation, SQLite scripting, archives, and
automation logic, from one interpreter.

### Hard boundaries for the review to assume

No Linux distribution, no PRoot, no apt, no compiler toolchain, no GCC/Clang, no
make, no Node or npm, no arbitrary executable downloads, no pip installation by
default, no native wheel compilation, no shell environment emulation. v1 targets
an interpreter plus a selected standard library under PocketClaw-controlled
execution, with no unrestricted package ecosystem.

### Android execution model

Python must respect what Managed Runtime already proved physically: executables
ship in the APK and run from `nativeLibraryDir`; **writable executable storage is
not used**. Writable Python data may live in app-private storage, and stdlib
resources may ship as non-executable assets. Do not write a native Python binary
into `filesDir` and try to exec it.

### The review must answer

Distribution route for Android ARM64; interpreter, stdlib and dynamic-module
sizes; compressed APK contribution and installed size; idle and per-script RAM;
startup latency; `nativeLibraryDir` packaging feasibility; stdlib asset layout;
`PYTHONHOME`, `PYTHONPATH`, `HOME`, `TMPDIR`; subprocess behaviour on Android and
its security implications; `ctypes`; dynamic extension modules; SSL; `sqlite3`;
`json`/`csv`/`xml`/`re`/`hashlib`/`zipfile`/`tarfile`; multiprocessing and signal
limitations; `/bin/sh` assumptions (PocketClaw's git already ships without a
usable one); pip feasibility and whether it should ship at all initially;
licensing and redistribution; provenance and build reproducibility; runtime
integration, timeout, cancellation, output bounds, redaction and workspace
boundaries; and the Agent-facing interface.

Compare a minimal bundled CPython, embedding CPython in Core, and any lighter
runtime that offers a real advantage. Do not pick an exotic runtime for size
alone if compatibility suffers.

### Security, stated honestly

The review must address filesystem access, subprocess execution, environment
access, secret exposure, network access, `ctypes`, dynamic libraries, native
extension loading, process creation, output limits, timeouts, cancellation, and
runaway CPU and memory. **Python must not become an escape hatch around Runtime
security**, and the review must not claim a sandbox the architecture cannot
provide. State the real boundary.

### Size gate

The APK is currently ~55.6 MB. Exact projections are required before any
inclusion: APK before, interpreter, stdlib, dynamic modules, APK after, installed
increase. **A 100+ MB addition needs explicit approval.**

- [x] Python Lite architecture review (2026-08-30, approved for Phase A only).
- [x] Python Lite **Phase A** build and measurement (2026-08-31). CPython 3.14.7,
  NDK 28.2, static modules, appended `.pyc` stdlib. Payload 11,591,387 bytes,
  APK delta +5,814,942 — both gates pass. See `runtime/PYTHON_LITE_PHASE_A.md`.
- [x] Python Lite Phase A **physical device run** (2026-08-31, SM-A165F,
  Android 16 / API 36). 52 passed, 0 failed. Harness:
  `runtime/python-lite-device-tests.sh`.
- [ ] Correct the architecture review's shell claim: Android 11+ ships
  `/bin/sh` (`/bin` -> `/system/bin`), so `subprocess(shell=True)` and
  `os.system()` work on API 30+. minSdk is 24, so document it as conditional.
  The same assumption underlies the recorded git `SHELL_PATH` limitation.
- [x] Python Lite provenance closed (2026-08-31): bzip2 1.0.8 and XZ 5.4.7 are
  built from pinned source; no upstream prebuilt binary is used.
- [x] Python Lite **Phase B** Runtime integration — **PHYSICAL PASS** and merged
  (2026-08-31). Catalog 2.1.0, 56 tools, 7 bundled. Verified inside the
  installed app: python resolves and runs, Python 3.14.7, json/arithmetic/
  Arabic/emoji/sqlite3 all correct through the Managed Runtime.
- [x] Fix the stale-Core packaging defect (2026-08-31): editing the embedded
  catalog requires `./core/build-android-arm64.sh`; guarded by
  `TestStagedCoreEmbedsTheCurrentCatalog`.
- [x] Python Lite Phase C — the Agent-facing `python` tool implemented on
  `feature/python-lite-agent-tool`. Code on stdin, never argv; reuses
  `Manager.Execute`; no duplicate execution path. Automated gates green.
- [x] Python Lite Phase C physical validation of statistics, JSON, Unicode,
  timeout and uncaught exception — all PASS. One gap found: the model could not
  report the traceback from a single `raise ValueError("TEST-ERROR")` call.
- [x] Expose the real `ExecResult` stderr to the Agent. `formatPythonResult` now
  names `exit_code`, `timed_out`, `cancelled`, `stdout_truncated` and
  `stderr_truncated` and prints both streams, an empty one included. End-to-end
  tests run a real interpreter through `Manager.Execute`.
- [x] Core staleness beyond the catalog: `pkg/coresource` fingerprints the Core's
  build inputs, `core/build-android-arm64.sh` stamps it into the binary, and the
  gate fails if the staged Core does not carry the current value. A Go-only
  change is now detected with the catalog unchanged.
- [x] Python Lite Phase C physical stderr recheck — **FAILED**. `print()` gave
  exit 0 with empty stdout and `raise` gave exit 1 with empty stderr; the timeout
  still passed. Not merged.
- [x] Prove which boundary loses the bytes: an empty `ExecResult.Stdout` can only
  mean the capture layer received nothing, because a bounded buffer that kept no
  bytes returns a truncation marker. Formatter, serialisation and the agent
  message are ruled out.
- [x] Close the test seam: a production-path test through
  `ToolRegistry.ExecuteWithContext` to `ContentForLLM()`, with a real child
  process staged as a checksum-verified bundled payload.
- [x] Instrument the boundary: `ExecResult.StdoutBytes`/`StderrBytes`, printed in
  the Python result and logged at INFO.
- [x] Root cause found and confirmed on device: Android's CPython replaces
  `sys.stdout`/`sys.stderr` with `TextLogStream`, which writes to the Android
  system log rather than to descriptors 1 and 2.
- [x] Ship `pocketclaw_bootstrap` inside the payload's appended stdlib and run
  the tool through `-m pocketclaw_bootstrap`. It rebinds both streams over
  `os.dup(1)`/`os.dup(2)`, then reads the program from stdin as `python -` does.
  Source transport, argv contract and traceback filenames unchanged. CPython not
  patched.
- [x] Repin the payload checksum, move the catalog to `2.2.0`, and add an
  entry-point guard to the Gradle release and the Go tests.
- [x] Python Lite Phase C **physical PASS** on all four tests, one call each, no
  retries: `stdout_bytes=18`/`PYTHON-FINAL-PASS`, `stderr_bytes=96` with a real
  traceback ending `ValueError: TEST-ERROR` at `exit_code=1`, `stdout_bytes=16`
  for `مرحبا 🐍`, and `timed_out=true` at 2 s. Merged to `develop`.
- [ ] Git `SHELL_PATH`: the recorded "Android has no /bin/sh" limitation is
  wrong on Android 11+, which ships `/bin` -> `/system/bin`. Re-evaluate whether
  git hooks and the ENOEXEC fallback can be supported on API 30+ devices.
