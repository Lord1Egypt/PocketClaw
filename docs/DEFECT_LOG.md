# PocketClaw defect log

This log contains defects and deferred engineering work with reliable repository
evidence. It is intentionally not an invented inventory of every issue ever
found. The operating method has surfaced and helped resolve dozens of defects;
only reconstructable examples belong here.

## Open / deferred





### PC-DEF-012 — Broad dependency export surfaces need reachability evidence

- **Discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Dart JNI plugin, embedded Python, and dependency-native ELF.
- **Severity:** Hardening review item; no demonstrated functional or security
  failure.
- **Description:** Arm64 `libdartjni.so` exports 313 symbols (214
  `globalEnv_*`, 42 Dart DL, seven Java/JNI, and 50 other); packaged Dart AOT
  contains 11 matching names. The Python executable exports 2,261 dynamic
  symbols. Datastore exports four required Java methods plus five C++ helpers.
  These are wider surfaces than the app-owned entry points, but FFI lookups,
  JNI name binding, statically linked modules, and upstream consumer contracts
  make blind visibility changes unsafe.
- **Evidence:** Dynamic-symbol and packaged-AOT comparison from the exact H4B
  APK; required Java/JNI symbols and Dart snapshot exports are asserted by the
  H5A audit tool.
- **Reason deferred:** Static counts do not establish that an export is safe to
  remove. Narrowing requires dependency-specific call/reachability evidence and
  runtime validation.
- **H5C disposition:** Unchanged again. H5C narrowed no export and produced no
  new reachability evidence; the packaged export counts in the production APK
  are identical to H5B's.
- **H5B disposition:** Evaluated and deliberately unchanged. H5B produced no
  call/reachability evidence for any of these surfaces, and narrowing a
  visibility surface without it is how a runtime `UnsatisfiedLinkError` ships.
  No export map was added and the packaged export counts are unchanged. The
  automated audit continues to assert the required boundary rather than hide
  the rest.
- **Target milestone:** A later dependency-focused milestone. Narrowing requires
  dependency-specific reachability plus runtime validation, or an explicit
  owner acceptance of the retained surface.
- **Status:** OPEN / FUTURE EVIDENCE REQUIRED.

### PC-DEF-002 — Web console listens on `0.0.0.0:18800`

- **Discovered:** vc59 machine validation, 2026-09-08.
- **Component:** Launcher/web console network exposure.
- **Severity:** Unrated security review item.
- **Description:** The Core gateway is loopback-only on 18790, while the web
  console was observed listening on all interfaces on port 18800.
- **Evidence:** `TASKS.md`, “Non-blocking security review”; historical
  `PROJECT_STATE.md` vc59 evidence.
- **Reason deferred:** It predated the milestone that found it and requires a
  product decision about cross-device console access.
- **Target milestone:** Secrets/configuration and exposure audit, before stable.
- **Audited 2026-09-12, and the observation is explained.** The product now has
  an explicit Public Mode whose stated purpose is LAN access to the
  password-protected dashboard. Reproducible evidence from the shipped
  `pkg/netbind`, driven with the same default-mode selection
  `openLauncherListeners` applies:

      PUBLIC OFF (no -public, no host)   bindHosts=[::1 127.0.0.1]
      PUBLIC ON  (-public, no host)      bindHosts=[:: 0.0.0.0]
      host override 127.0.0.1 + -public  bindHosts=[127.0.0.1]   (host wins)
      gateway (host=localhost)           bindHosts=[::1 127.0.0.1]  in BOTH states

  `0.0.0.0:18800` is Public Mode ON and nothing else. The Core gateway on 18790
  is loopback-only in both states and is not reachable by the launcher's public
  flag at all — `openGatewayListeners` always passes `netbind.DefaultLoopback`.
  Authentication is mandatory whenever Public Mode is on: the unauthenticated
  surface is only `POST /api/auth/{login,logout,setup}`, `GET /api/auth/status`
  and GET/HEAD of the login/setup SPA routes, `/assets/`, the favicons,
  `site.webmanifest` and `robots.txt`; there is no unauthenticated health,
  config or control endpoint. The realtime WebSocket requires a live session
  **and** an origin check, and auth-path canonicalization blocks
  `/assets/../` traversal. `/api/auth/setup` is unauthenticated only while no
  password exists and requires a session once one does. The Android
  local-auto-login grant cannot exist on Android at all, because the host passes
  `--no-browser` and `shouldEnableLocalAutoLogin` requires its absence. The
  empty default `allowed_cidrs` means the IP allowlist is a documented no-op and
  the password is the boundary.
- **Disposition:** the original wording — "the console was observed listening on
  all interfaces" — is **resolved as explained by design**. Its one remaining
  actionable residue is that "Public Mode off" is not reliably enforced across a
  service restart, which is tracked precisely as `PC-DEF-020` rather than left
  inside this entry.
- **Status:** RESOLVED AS EXPLAINED / superseded by `PC-DEF-020`.

### PC-DEF-003 — Restart-required banner can remain after hot reload

- **Discovered:** Telegram interactive-menu/model-selection closeout.
- **Component:** Dashboard/launcher state presentation.
- **Severity:** Low / cosmetic, as recorded at discovery.
- **Description:** `gateway.bootConfigSignature` is not refreshed after a
  successful in-process reload, so the Dashboard can continue showing
  “Gateway restart required.”
- **Evidence:** `TASKS.md` and `SESSION_HANDOFF.md` entries named “Stale
  Gateway restart required banner.”
- **Reason deferred:** Unrelated to the milestone and non-blocking.
- **Target milestone:** Dedicated dashboard state-correctness maintenance.
- **Status:** OPEN.

### PC-DEF-004 — `BaseChannel` typing defaults are inconsistent

- **Discovered:** Telegram interactive-menu/model-selection closeout.
- **Component:** Non-Telegram channel typing behavior.
- **Severity:** Unrated.
- **Description:** Non-Telegram channels do not share settled typing defaults.
- **Evidence:** Repeated open entries in `TASKS.md`; historical
  `PROJECT_STATE.md` explicitly says defaults must be decided before gating.
- **Reason deferred:** Product semantics were undecided and unrelated to the
  completed Telegram scope.
- **Target milestone:** Dedicated channel behavior milestone.
- **Status:** OPEN.

### PC-DEF-005 — Versioned non-destructive bootstrap updates are unresolved

- **Discovered:** vc55 investigation; architecture implemented during Bootstrap
  Architecture on 2026-09-08.
- **Component:** Workspace template lifecycle.
- **Severity:** Deferred architecture work.
- **Description:** Binary-owned capability guidance now updates safely, but no
  production behavior offers or merges improvements to a pristine historical
  user template. User edits and `MEMORY.md` must never be overwritten.
- **Evidence:** `TASKS.md`, “Bootstrap architecture implemented”; root
  `DECISIONS.md`, “Product guidance ships in the binary; workspace files belong
  to the user”; current release-gate pending list.
- **Reason deferred:** Safe product behavior requires a separate decision and
  cannot be inferred from file equality alone.
- **Target milestone:** Post-stable architecture unless explicitly reprioritized.
- **Status:** OPEN.

### PC-DEF-006 — Full APK reproducibility is not yet proven

- **Discovered:** H1/H1.5 F-Droid readiness audit.
- **Component:** Release build / Official F-Droid path.
- **Severity:** Blocks the target developer-signed F-Droid publication path.
- **Description:** Core, frontend, runtime recipes, and individual payloads have
  reproducibility evidence, but the final hardened APK has not been rebuilt
  twice and compared bit-for-bit.
- **Evidence:** [`FDROID_RELEASE.md`](FDROID_RELEASE.md), sections 3, 4, 6, and
  8; current release-gate pending list.
- **Reason deferred:** Final APK inputs are still changing during H3 and later
  hardening.
- **Target milestone:** Production artifact hardening/reproducibility proof.
- **Not closed by the exposure-audit closure re-run, 2026-09-13.** That audit
  built one APK and did not build and compare two independent final builds, so
  this acceptance criterion is untouched. One observation recorded without
  overstating it: the APK built at the re-run (`113a8382…`) and the one built at
  `PC-DEF-021` (`7155de0a…`) differ while their Dart AOT and R8 mapping are
  byte-identical. The earlier APK had been overwritten by `--clean`, so no
  byte-level comparison was possible; differing archive hashes with identical
  compiled inputs are consistent with packaging non-determinism, which is
  precisely what this defect is about. It gates the F-Droid path, not the
  GitHub / direct APK release.
- **Status:** OPEN.

### PC-DEF-007 — F-Droid builder compatibility and committed prebuilts remain open

- **Discovered:** H1.5D source-build audit.
- **Component:** Official F-Droid submission.
- **Severity:** Submission blocker, not an application runtime defect.
- **Description:** All eight Managed Runtime payloads can be built from source,
  but F-Droid builder acceptance of NDK/Rust/Go recipes and the repository's
  committed native prebuilts is unresolved.
- **Evidence:** [`FDROID_RELEASE.md`](FDROID_RELEASE.md), “Remaining F-Droid
  question” and summary.
- **Reason deferred:** It requires final build metadata and external F-Droid
  policy validation.
- **Target milestone:** F-Droid submission preparation.
- **Status:** OPEN.

## Resolved

### PC-DEF-069 — A bot already owned by another service fought it or died silently

- **Discovered:** owner requirement, 2026-09-17, v0.2.0 release hardening.
- **Component:** `core/src/pkg/channels/telegram/{telegram.go,polling.go,command_registration.go}`,
  new `conflict_test.go`; `core/src/web/backend/api/{telegram_credentials.go,telegram_readiness.go,android_bridge.go,telegram_onboarding.go}`,
  new `telegram_credentials_test.go`; `core/src/web/frontend/...` and 14 locale bundles.
- **Two conflict classes, one root cause.** `getMe` proves only that a token is
  valid; it says nothing about whether PocketClaw can own the update stream. So a
  manually entered token already attached to another application could pass
  `getMe` and then either (a) face an active webhook, which makes `getUpdates`
  answer 409, or (b) be owned by another long poller, which also answers 409.
  Before this fix the 409 fell into Telego's generic eight-second retry loop and
  PocketClaw fought the other service indefinitely; and a candidate could be
  persisted over a working bot before the conflict surfaced.
- **Resolution — runtime, non-destructive.** `Start` now asks `getWebhookInfo`
  before polling. A configured webhook is refused; PocketClaw **never calls
  `deleteWebhook` or `setWebhook`**, and mutations of the other service are
  impossible by construction. The `telegramIntakeCaller` classifies a `getUpdates`
  409 (the reliable signal is the status code; the description only refines the
  subtype to `webhook_active` vs `bot_in_use`) and returns a cancellation-shaped
  error so Telego stops immediately. The exact generation is revoked, its command
  registration cancelled, and it is retired once — no retry loop, and no
  generation left that could authorize a handoff.
- **Resolution — replacement transaction.** Candidate validation is a pre-commit
  gate: `getMe`, then the non-destructive `getWebhookInfo`, then a
  non-consuming `getUpdates` probe (`offset -1`). A candidate owned elsewhere is
  rejected with `ErrTelegramWebhookConflict` / `ErrTelegramBotInUse` **before any
  config mutation**, so the previously committed bot stays authoritative and
  recoverable. A transport or unexpected probe answer is not treated as a
  conflict, so a valid token is never refused for a network hiccup; the runtime
  409 path remains the backstop.
- **Failure states stay distinct.** 401 → `invalid_credentials`; 409 webhook →
  `telegram_conflict` / `webhook_active`; 409 poller → `telegram_conflict` /
  `bot_in_use`; missing owner → `setup_required` / `owner_missing`. None collapse
  into one "connection failed". Readiness returns generation 0 for a conflict.
- **Logging.** The conflict path logs the generation, the safe subtype and the
  state transition only — never the token, the webhook URL, an owner id or a chat
  id. The webhook URL is never returned to a client.
- **Verification:** channel `conflict_test.go` (webhook refused non-destructively
  with no `getUpdates`/`deleteWebhook`/`setWebhook`; 409 retires the generation
  with exactly one ownership attempt and no retry; 409 is not 401; a webhook-
  described 409 classifies as `webhook_active`; a webhook appearing after the
  preflight is still caught). Backend `telegram_credentials_test.go` (healthy
  accept; webhook rejected without delete; another poller rejected; poll-409
  webhook classification; 401 stays 401; probe-401 stays invalid; an unexpected
  probe answer is not a conflict).
  `TestAndroidTelegramBridgeRejectsOwnedCandidateWithoutReplacingOldBot` proves
  A is preserved for both conflict kinds; `TestTelegramReadinessReportsWebhookConflict`,
  `...ReportsBotInUse`, `...Keeps401And409Distinct` prove the state mapping.
  Frontend: `case 3d`/`case 3e` conflict cards and two managed-connect error-kind
  tests.
- **Status:** FIXED IN SOURCE — physical confirmation required (disposable bots
  only, per the consolidated checklist).

### PC-DEF-068 — A valid Telegram token with no owner was a silent dead bot

- **Discovered:** owner report, 2026-09-17, during v0.2.0 release hardening.
- **Component:** `core/src/pkg/channels/telegram/telegram.go`, new
  `owner_missing_test.go`, `core/src/pkg/channels/{interfaces,manager}.go`,
  `core/src/pkg/status/status.go`, `core/src/web/backend/api/telegram_readiness.go`,
  `core/src/web/frontend/src/components/channels/channel-forms/telegram-panel.tsx`.
- **Symptom, and why it was invisible.** A user could save a valid bot token with
  an empty owner from the desktop manual form — `PATCH /api/config` never ran the
  Telegram owner contract, while the full-config PUT and the Android/managed
  writers always injected exactly one owner. `NewTelegramChannel` then refused the
  channel with "telegram requires exactly one paired numeric owner", the Manager
  logged and skipped it, and **the bot polled nothing at all**. Because
  `telegramIsConfigured` only checks enabled + token, the Dashboard still called
  it configured while readiness sat at `gateway_starting` forever. A user who
  messaged the bot got silence and no explanation.
- **The contract that was missing.** Zero owners is not a construction error to be
  swallowed; it is an explicit lifecycle state. It must be defended at runtime
  even when UI validation would normally prevent it.
- **Resolution, in the channel.** `NewTelegramChannel` now accepts zero owners as
  the **owner-missing** state instead of failing, and still refuses several,
  blank, wildcard, non-numeric and non-positive owners. `handleMessages` handles
  that state **before the allowlist**: a private sender gets deterministic setup
  guidance exactly once, including their own numeric id (the one fact the inbound
  update carries that would otherwise need a third-party id bot); a group or
  channel is left unanswered rather than leaking setup guidance publicly. Nothing
  else runs: no agent turn, no provider, no tool, no session write, no built-in
  command (so a `/start` cannot execute as if the sender were the owner), no media
  download, no config mutation and no auto-claim. Because an empty `AllowFrom`
  makes `BaseChannel.IsAllowedSender` permissive, the base allowlist is seeded
  with a non-matching sentinel — a false "allows everyone" warning and a
  permissive base layer are both avoided.
- **Resolution, in state and readiness.** `status.Channel` gained an
  `owner_missing` boolean via a new `OwnerMissingReporter`. `/api/telegram/readiness`
  gained `setup_required` (detail `owner_missing`), derived from the persisted
  configuration so it is the same contract whichever writer saved it, and it never
  names a polling generation, so it cannot authorize a handoff. The Dashboard
  shows "Telegram setup incomplete" instead of Connected and opens the Allowed
  From field directly.
- **What is deliberately unchanged.** The one-owner contract still authorizes the
  agent; a non-owner is still rejected silently; Disconnect/Replace, generation
  ownership, intake ordering and the 401 fail-fast are untouched. Core has no
  locale, so the Telegram reply is English like every other Core reply; the
  machine-readable state is what the Dashboard localizes.
- **Verification:** `owner_missing_test.go` — private text gets guidance once with
  the sender's id and no bus publish; `/start` does not run the authorized start
  reply; a group gets nothing; two senders each see only their own id; with an
  owner configured a non-owner still gets no guidance; the constructor accepts
  zero owners and refuses every invalid list. Backend:
  `TestTelegramReadinessReportsOwnerMissingAsSetupRequired`,
  `TestTelegramReadinessEndpointReportsSetupRequired`,
  `TestTelegramOwnerMissingIsFalseWhenAnOwnerIsConfigured`,
  `TestTelegramOwnerStateIsDerivedFromConfigurationRegardlessOfWriter`. Also
  pinned: `TestOwnerMissingStateIsNotAutoClaimed` (a stranger's message never
  makes them the owner) and `TestConfiguringAnOwnerRestoresNormalRouting` (after
  an owner is configured, the same sender reaches the agent). Frontend:
  `case 3c: a valid token with no owner shows setup incomplete, not connected`.
- **Status:** FIXED IN SOURCE — physical confirmation required.

### PC-DEF-067 — Desktop "Copy link" failed on a plain-HTTP origin

- **Discovered:** owner physical browser observation, 2026-09-17.
- **Component:**
  `core/src/web/frontend/src/components/channels/channel-forms/telegram-desktop-connect.tsx`,
  `core/src/web/frontend/src/lib/clipboard.ts`.
- **Symptom:** pressing "Copy link" on the desktop Telegram onboarding card
  produced the failure toast ("Could not copy the link") instead of copying.
- **Root cause, proven by reading every copy path.** The component was the **only**
  copy surface in the Dashboard that called `navigator.clipboard.writeText`
  directly. The async Clipboard API exists only in secure contexts, and the
  launcher serves plain HTTP: a desktop browser reaching the console from a LAN
  address is not a secure context, and desktop WebViews that do not expose the API
  behave the same way. `copyText` in `src/lib/clipboard.ts` already handles this
  with an `execCommand("copy")` fallback and is used by every other surface; this
  one did not. The bare `catch` converted the `TypeError` into the failure toast.
  No test covered the copy path, and jsdom defines no `navigator.clipboard`, so CI
  never saw it.
- **Resolution.** `copyLink` now goes through `copyText`. On success it shows
  "Link copied."; when **both** paths fail it does not dead-end: the canonical
  Telegram link is rendered in a read-only, selectable field with guidance.
  `copyText` was also made to return `false` rather than throw when `execCommand`
  is missing (jsdom, some WebViews). Only a link that passed the
  Telegram-destination guard can be copied or shown; no backend, hosting or
  callback URL can reach the clipboard.
- **Verification:** 6 cases in `src/lib/clipboard.test.ts` (Clipboard success,
  rejected promise falls back, absent API falls back, both unavailable, execCommand
  refuses, execCommand missing); 2 in `telegram-desktop-connect.test.tsx`
  (`copies the trusted Telegram link through the shared helper`,
  `shows a selectable trusted link when copying is impossible`) plus the existing
  non-Telegram test now asserts nothing was ever offered to the clipboard.
- **Status:** FIXED IN SOURCE — physical browser confirmation required.

### PC-DEF-030 — A saved Telegram configuration did not become live

- **Discovered:** Samsung physical testing of verification APK
  `b6d6e6f9bd6316e24308a63265dcb1e4c15ef7bd1b7e04ba921a5d6b0a1e63b7`,
  2026-09-13.
- **Component:** `web/backend/api/gateway_config_restart.go`,
  `web/backend/api/android_bridge.go`,
  `android/.../service/PocketClawService.kt`, `lib/src/core/service_manager.dart`.
- **Description:** managed Telegram onboarding completed, the bot was created
  and the configuration was saved, and the bot did not answer until the owner
  restarted the PocketClaw Service **and** the Gateway by hand. After that
  restart the runtime log shows the whole path healthy: channel initialised,
  `getMe` succeeded, long polling started, the owner accepted, the model
  answered, the reply delivered. The auto-apply added in `a7533bd` is present in
  the tested APK — `libpocketclaw-web.so` contains `telegram_configured`,
  `/api/pocketclaw/internal/gateway-idle` and the configuration-restart log
  strings — so the fix shipped and did not work.
- **Root cause:** three independent holes, each sufficient on its own.
  1. `gatewayProcessRunning` read only the launcher's in-memory gateway state,
     which is populated when this process starts the gateway or when a client
     polls `GET /api/gateway/status`. The Android Telegram bridge does neither,
     so a live gateway read as "not running": the busy check was skipped
     entirely, `stopGatewayProcessForRestart(nil)` stopped nothing, and a second
     gateway was started against a port the first still held. The original
     process went on serving the previous configuration.
  2. A change parked with outcome `unverified` waited for an idle notification
     that cannot arrive. The gateway reports only the in-flight transition
     N>0 → 0, and a gateway that is already idle never crosses it.
  3. Flutter's reload was `stop()` then `start()`. On Android those are two
     service intents with an unconditional `stopSelf()` between them, which
     Android honours even though the start request has already been queued, so
     the freshly started service is destroyed and Core is left stopped.
- **Resolution:** the restart path reconciles against the pid file before
  deciding anything and adopts a gateway it is not tracking; a bounded
  supervisor re-asks the idle question for a parked change and applies it the
  first time the answer is trustworthy; the Android service gained one
  `ACTION_RESTART` that stops and starts Core in order, and
  `ServiceManager.restartCore()` is the only way Flutter asks for a restart. The
  Android Telegram bridge no longer blocks its response on the two-minute idle
  wait: it parks the change, which both the notification and the supervisor pick
  up. "Unknown is not idle" is unchanged — nothing interrupts a running answer.
- **Verification:** `TestGatewayProcessRunningAdoptsAnUntrackedGateway`,
  `TestGatewayProcessRunningStaysFalseWithoutAPidFile`,
  `TestWaitForGatewayIdleDoesNotInterruptAnAdoptedGateway`,
  `TestPendingApplySupervisorAppliesOnceTheGatewayIsSafeToRestart`,
  `TestPendingApplySupervisorIsSingular`, and the Flutter
  `service_restart_single_intent_test.dart` suite.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS**, Samsung, 2026-09-13. The
  managed Telegram setup activates and the bot answers with no manual Service or
  Gateway restart.

### PC-DEF-032 — OpenCode inference failed with an unexplained HTTP 400

- **Discovered:** Samsung physical testing, 2026-09-13.
- **Component:** `pkg/providers/opencode_routing.go`, the three OpenCode
  transport arms, `pkg/agent/pipeline_llm.go`, `pkg/agent/error_format.go`.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS**, Samsung, 2026-09-13.
  `deepseek-v4.1-flash` produced a real Chat response on the device, so the
  `x-opencode-session` implementation, the OpenCode Go endpoint, the API key and
  model inference are all confirmed in the tested flow. Not to be reopened
  without contradictory evidence.

- **Root cause — PROVEN, and not what this log first recorded.**
  OpenCode Go requires an `x-opencode-session` header and PocketClaw never sent
  one. Owner's direct external test against the live service, 2026-09-13:

  `POST https://opencode.ai/zen/go/v1/chat/completions`, model
  `deepseek-v4.1-flash`, same key, same body.

  | Request | Result |
  | --- | --- |
  | without `x-opencode-session` | HTTP 400 `{"error":{"type":"MissingSessionID","message":"Error from provider (Console Go): Request is missing x-opencode-session and cannot be routed efficiently..."}}` |
  | with `x-opencode-session: <uuid>` and `User-Agent: PocketClaw/0.2.0` | HTTP 200, assistant content `OK` |

  So the service, the API key, the model and the Go endpoint are all PROVEN
  WORKING. The incompatibility was PocketClaw's.

- **Correction to the earlier entry.** This log previously recorded the cause as
  a model configured against the wrong OpenCode endpoint, on the evidence that
  `deepseek-v4.1-flash` appears in Go's inventory and not in Zen's. That
  observation is still true and still a real hazard — Zen answers that model id
  with `ModelError: Model deepseek-v4.1-flash is not supported` — but it was not
  the cause of the owner's failure, which was on Go. The endpoint-provenance
  work done for it stands on its own merits and is retained.

- **Resolution.**
  - `common.StableSessionID` maps a conversation scope to a stable, opaque,
    UUID-shaped identifier through a process-lifetime random salt. The scope is
    never sent: a PocketClaw session key can be a token-shaped identifier and a
    chat id can be the owner's Telegram account, and neither belongs to a third
    party. A restarted gateway re-salts, so a conversation spanning a restart
    continues under a new id — a routing inefficiency, never an error.
  - `turnConversationScope` composes the scope from the turn's session key **and**
    chat id. The session key alone was the wrong granularity: under the default
    session policy several chats on one channel share one key, which a test
    caught by finding two conversations collapsing into one session.
  - The scope rides on the per-turn options every request is built from, so each
    turn, each streamed turn, each tool-call continuation, each retry and each
    fallback candidate carry the same value. Summarisation and compaction, which
    run outside any conversation, fall back to a process identifier rather than
    sending no header — "no header" is the 400 this exists to prevent.
  - Applied to all three OpenCode transport arms (chat completions, responses,
    Anthropic messages) and to both gateways, with `User-Agent: PocketClaw/0.2.0`.
    It is opt-in per provider and reaches nothing else.
  - Separately, the provider's own explanation now reaches the user. The
    `MissingSessionID` message names the problem exactly, and PocketClaw was
    replacing it with "request rejected (400)". `providerErrorDetail` extracts
    one known message field, redacts it, collapses it to one line and caps it.

- **Verification:** `pkg/providers/common/session_test.go` (identity, stability,
  per-conversation distinctness, no scope or credential in the output, the
  process fallback, opt-in); `pkg/providers/opencode_session_test.go` (the
  header on the wire for non-streaming and streaming turns, one value across a
  turn/tool-continuation/retry sequence, a different value for a new
  conversation, and the header reaching none of five unrelated providers, each
  against a stub that refuses a request without it exactly as the real gateway
  does); `pkg/agent/session_scope_test.go` (the agent actually puts the scope on
  the turn options); `pkg/agent/provider_detail_test.go` (the error detail).
- **Outstanding:** owner's regression items 10 — a real physical request from
  the Samsung. Until then this defect is FIXED IN SOURCE and OPEN.

### Identifier note for the 2026-09-13 Samsung results

The owner's report allocated **PC-DEF-047, 048 and 049** to provider CRUD, the
API-key update and the Telegram command menu. All three numbers were already in
use in this log — 047 is the model Delete affordance, 048 is the hardened build
skipping Flutter AOT. The three new defects are therefore recorded as
**PC-DEF-049, PC-DEF-050 and PC-DEF-051**, in the owner's order, and each entry
names the number the owner used. This is the same renumbering the PC-DEF-047
entry below already carries a note about.

A third collision followed on 2026-09-14: the owner allocated **PC-DEF-054** to
the Telegram readiness race, but 054 was already the signature-plaintext defect.
That race is recorded as **PC-DEF-056**, and the logging-hardening work beside it
as **PC-DEF-057**. The owner's later report refers to the race as PC-DEF-054
again — **owner's 054 = PC-DEF-056 here**, and it is now physically verified.

The two UX defects from that same report took the next free numbers:
**PC-DEF-058** (first run never requested Android notification permission) and
**PC-DEF-059** (authentication discarded the requested Dashboard destination). The
desktop Telegram observation that followed is **PC-DEF-060**.

Two later owner requirements, and two items this session disclosed and was told
to act on, took the next free identifiers: **PC-DEF-052** (Telegram onboarding
exposed the hosting origin), **PC-DEF-053** (actionable user-facing runtime and
configuration errors), **PC-DEF-054** (the config-change signature carried
credentials in plaintext) and **PC-DEF-055** (Set Default was the last
tooltip-only control). The owner named 052 and 053 directly; 054 and 055 were
allocated here because they are defects with their own evidence and resolution,
not sub-points of another entry.

Two corrections from the same report are applied to this log rather than argued
with:

- **PC-DEF-032 (OpenCode Go) is PHYSICALLY VERIFIED PASS.** `deepseek-v4.1-flash`
  produced a real Chat response on the device. The `x-opencode-session` work, the
  Go endpoint, the key and model inference are all confirmed in the tested flow.
  Not to be reopened without contradictory evidence.
- **PC-DEF-033 (the amber block) and PC-DEF-030 (Telegram automatic runtime
  apply) are PHYSICALLY VERIFIED PASS.** The managed bot activates and answers
  with no manual Service or Gateway restart.
- **Model Delete exists.** The Model screen carries Delete Model, Edit API Key
  and the default-model state. PC-DEF-047's fix is confirmed by the screenshots;
  nothing below claims model deletion is missing. The management gap is at the
  **provider** level.

### PC-DEF-066 — Telegram /help opened with pre-Aperture branding

- **Discovered:** owner physical Telegram observation, 2026-09-15.
- **Component:** `pkg/commands/cmd_help.go`, new `pkg/commands/branding_test.go`.
- **Symptom:** `/help` began with a lobster emoji before the product name — legacy
  branding from before the Aperture visual system, and the first thing the command
  a new user runs showed them.
- **Audit, since the owner asked for the surface and not just the one string.** The
  lobster existed in exactly two places. `pkg/commands/cmd_help.go` is the one a
  chat user reaches. `pkg/env.go`'s `Logo` constant is upstream terminal branding
  for the PicoClaw CLI and is **not referenced from `pkg/` or `web/` at all**, so it
  cannot reach a PocketClaw channel, the gateway's user-visible log or the
  Dashboard; it is left alone deliberately rather than forking the upstream
  baseline for no user-visible gain. Every other emoji in the command surface
  (`🤖`, `📋`, `👁`, `🔄`) is functional decoration, not legacy identity, and none
  was replaced — the instruction was not to introduce new ones. `/start`, `/show`,
  `/list`, `/check`, `/switch`, every command description and every subcommand
  description were checked for legacy identity text and are clean.
- **Resolution:** the header is `PocketClaw`. Removed rather than substituted: the
  product's mark is not an emoji, and picking a different one would be inventing
  identity in a help string instead of using the product's own.
- **Why nothing caught it, and what does now.** Zero-Pico is lexical and an emoji
  is not a Pico identity; the i18n parity suites cover the Dashboard bundles and
  the Flutter ARB files, not Core's command text; the command-menu tests assert
  names and counts. So the branding could sit in `/help` with every gate green.
  `branding_test.go` now checks the assembled `/help` output and every command and
  subcommand description, usage and no-args help for the mascot and for legacy
  product identity in any casing — and asserts `/help` still names the product, so
  it cannot be satisfied by deleting the header. **Proven to fire:** restoring the
  lobster fails it with the offending line quoted.
- **Command behaviour is untouched**, and the 14-command registration the owner
  physically verified is unchanged.
- **Sweep update, 2026-09-17.** The branding check covered
  `BuiltinDefinitions()` only, while the handler prefers the runtime's own
  `ListDefinitions()` — the list that actually reaches a chat user. Added
  `TestRuntimeHelpOutputCarriesNoLegacyBranding`, which drives the real handler
  with a runtime-supplied command, asserts the assembled reply is clean and that
  the runtime command reached it. The header assertion is now exact
  (`PocketClaw\n\n`) rather than a substring, so a substituted emoji or tagline
  cannot satisfy it.
- **Status:** RESOLVED.

### PC-DEF-065 — The first Dashboard password could not be created

- **Discovered:** owner physical fresh install on the Samsung, 2026-09-15.
- **Component:** `web/backend/launcher_http_runtime.go`, `web/backend/main.go`,
  `web/backend/api/android_bridge.go` (interface), new
  `web/backend/journey_fresh_install_test.go`, `tool/release_gate.py`.
- **Symptom:** on a fresh install the first-run Dashboard password page accepted a
  password and confirmation and answered **"must be authenticated to change
  password"** — which is impossible for a first setup, because there is nothing to
  authenticate with yet. First-time Dashboard initialization was impossible.

**Root cause, reproduced in a test before anything was changed.** The exact path:

1. `POST /api/auth/setup` reaches `handleSetup` with the store uninitialized, so
   the first-claim branch is taken. The request is loopback, so PC-DEF-039 permits
   it. **The password is written.**
2. `handleSetup` then calls `onClaimed()` — PC-DEF-040's reconciliation — **on the
   request's own goroutine**, after `w.Write` but *before the handler returns*. The
   response is still in `net/http`'s buffer at that point; written is not flushed.
3. `ReconcileAfterDashboardClaimed` → `ApplyPublicMode(true)` → `closeLocked()` →
   `server.Close()`, plus an explicit `Close()` of every tracked connection —
   **including the connection carrying this very request**.
4. The browser gets an aborted request. The user sees setup fail.
5. The retry now finds `initialized == true` and is refused with the
   change-password rule: **401 "must be authenticated to change password"**.

So the two flows the owner asked about were never conflated: the initial-claim
branch and the authenticated-change branch are correctly distinct, and the report
was the *second* attempt hitting the second branch after the first had silently
succeeded. It only reproduces when Public Mode is already requested, which is what
makes the reconciliation run at all — and that is the documented fresh-install
sequence.

- **Resolution, in two parts, because either alone is insufficient.**
  **Widening drains instead of cutting off:** a swap that only *widens* access
  revokes nothing, so the old listener group is now `Shutdown` with a bounded wait
  and in-flight requests finish. Narrowing still closes hard — there a remote
  client is being revoked and must not be allowed to finish.
  **The reconciliation is asynchronous, and must be:** it is invoked from the very
  request whose listener it replaces, so draining inline would deadlock against
  `Shutdown` waiting for that handler. It now runs on its own goroutine and logs
  its own outcome, so the handler returns, the response flushes, and the drain
  completes. `ReconcileAfterDashboardClaimed` therefore no longer returns an error
  — there is no caller left to return one to.
- **Security is unchanged, and asserted.** First claim is still loopback-only and
  still refuses spoofed `Host`/`X-Forwarded-For`; an initialized dashboard still
  refuses an unauthenticated password change; `/launcher-setup` is still not a
  reset path. PC-DEF-037 and PC-DEF-039 hold.
- **Verification:** `journey.fresh_install`, a new gate row, drives the ordered path
  against the real HTTP server, the real listener swap, the real middleware and the
  real bcrypt store. **Proven to catch the regression:** with the inline
  hard-closing behaviour restored, the journey fails with
  `POST /api/auth/setup: the response never arrived: EOF` — the user-visible
  failure, in a test. It also covers login with the newly created password, the
  refusal of an anonymous password change (and that the original password still
  verifies afterwards), the refusal of a remote first claim from three addresses
  with spoofed headers, and ownership surviving a restart.
- **The process change the owner required.** The journey is a named gate row rather
  than one more test in a package, because every isolated test passed while the
  ordered path was broken. Any change to auth, launcher setup, first claim, Public
  Mode, Android permissions, the Service lifecycle or Dashboard middleware runs it.
- **Status:** **PHYSICALLY VERIFIED PASS**, 2026-09-15. A fresh install created its
  first Dashboard password on the first attempt, Public Mode reconciled to LAN
  without a toggle, and login with that password succeeded. Not to be reopened
  without contradictory evidence.

### PC-DEF-064 — What's New described a release that had moved on

- **Discovered:** owner physical verification, 2026-09-15.
- **Component:** `lib/l10n/app_*.arb`, `lib/src/whats_new/whats_new_release.dart`,
  new `tool/release_notes.py`, `docs/RELEASE_NOTES.md`, `tool/release_gate.py`.
- **Symptom:** the What's New screen had not been audited against the final 0.2.0
  candidate. One bullet was actively wrong -- "Telegram integration, set up from
  Settings" -- because setup had moved: the managed flow is offered in the app
  *and* from the Dashboard in a browser, and the owner pairing is the part worth
  saying. Several verified capabilities were missing entirely.
- **Audit result.** Every existing bullet was checked against the source. The
  bundled-tool claim holds: `pkg/pcruntime/manifest.json` carries `git`, `gh`,
  `curl`, `rg`, `jq` and `sqlite3`. The Python claim holds: 3.14.7. The three
  "one-time effects of upgrading" bullets stay last and stay in Improvements,
  because they say what to expect after installing rather than what is new.
- **Resolution:** the inaccurate bullet reworded, and seven added, each for
  something physically verified in an earlier round -- managed Telegram setup and
  its owner pairing, Dashboard Telegram management, provider and model management
  with key rotation, the automatic Telegram apply (PC-DEF-030), the `PC-E-AI`
  configuration messages (PC-DEF-053), the post-sign-in destination
  (PC-DEF-059), the logging redaction (PC-DEF-057), and the unclaimed-Dashboard
  rule (PC-DEF-039). **Deliberately not claimed:** the notification permission,
  which the owner named as conditional on physical verification and which
  PC-DEF-058 has not yet had.
- **The drift itself is now a gate.** `tool/release_notes.py` renders
  `docs/RELEASE_NOTES.md` from the app's own release structure and English
  strings, so the published notes cannot say something different from the screen;
  `release.notes_match_whats_new` fails the source gate on a stale file. Proven
  to fire: an extra hand-written bullet in the rendered file fails the check.
- **Status:** RESOLVED.

### PC-DEF-063 — Unknown was used as the About screen's loading placeholder

- **Discovered:** owner physical observation, 2026-09-15.
- **Component:** `android/.../PocketClawService.kt`,
  `android/.../PocketClawMethodChannel.kt`, `lib/src/core/pocketclaw_channel.dart`,
  `lib/src/core/service_manager.dart`, `lib/src/native/core_service_adapter.dart`
  and both adapters, `lib/src/ui/config_page.dart`.
- **Symptom:** About showed the Core version as unknown and only later as
  `0.3.1`.
- **Cause, which is not a loading-state bug but a value doing two jobs.** Reading
  the Core version means *running the Core binary*, which can fail transiently.
  Every layer answered the literal string `"unknown"` for a failure: the Kotlin
  probe, the method channel's catch, and both Dart adapters. `ServiceManager`
  then cached it, because `"unknown"` passes a non-empty test -- so one transient
  failure became the Core version and stayed displayed until something happened
  to re-probe. The About dialog compounded it by deciding "loading" from whether
  it had a value rather than from whether the future had completed.
- **Resolution:** a failed probe is an absence. `readCoreVersion` returns
  `String?`, the channel reports `null` (still mapping a literal `unknown`,
  because the host reads the version out of a binary whose output it cannot
  assume), the adapter interface is `Future<String?>`, and **a failure is never
  cached** -- the cache stays empty so the next read retries. The three states the
  owner asked for are now distinct: the future still running renders Loading, a
  value renders the version, and only a *completed* failed probe renders
  Unavailable. There is still one source: the version comes from the binary, not
  a second hardcoded constant.
- **Verification:** 4 cases in `test/unit/core_version_probe_test.dart` -- a
  failing probe, a silent probe, a failed probe not being cached so the next read
  succeeds, and a successful probe being cached. The two existing channel tests
  that pinned the `"unknown"` sentinel were rewritten to the new contract.
- **Sweep update, 2026-09-17.** Added the widget case that was missing:
  `renders Unavailable when the Core probe returned null` pins the real
  completed-failure representation (`AboutInfo(coreVersion: null)`) to
  Unavailable and asserts no spinner, which is the distinction PC-DEF-063 is
  about. The app version still comes from `package_info_plus` and the Core
  version from the binary probe; no hardcoded release number was added.
- **Status:** RESOLVED.

### PC-DEF-062 — Desktop could pair a Telegram bot but never remove one

- **Discovered:** owner physical verification, 2026-09-15.
- **Component:** new `web/backend/api/telegram_lifecycle.go`, new
  `web/frontend/src/api/telegram-lifecycle.ts`, new
  `channel-forms/telegram-disconnect-dialog.tsx`, `channel-forms/telegram-panel.tsx`.
- **Symptom:** having created a bot from a desktop browser, the owner had to pick
  the phone up to remove it.
- **Cause:** the connected card's only actions were Open chat and Reconnect, and
  both are gated on the Android host -- `openTelegramOnboarding` and
  `openExternal` are host calls. With no host, a configured Telegram channel
  rendered as a read-only summary with no lifecycle at all.
- **Resolution:** removal is Core's operation, so it lives in Core.
  `clearTelegramCredentials` is the deliberate mirror of
  `writeTelegramCredentials` -- same loader, same save, same apply -- and clears
  the token, the owner allowlist and the enabled flag **together**. A partial
  removal would be worse than none: a disabled channel still holding a token and
  an owner reads as connected to every surface that asks, and a retained owner
  would silently authorise the next bot paired there. Going through
  `applyTelegramConfigChange` is what stops the old bot polling; a client-side
  field edit would have left it running. The Dashboard gains an explicit
  confirmation that says what is cleared, and Replace bot reveals the managed
  flow that was already verified rather than reimplementing pairing.
- **Verification:** 6 backend cases -- the three fields cleared together, the
  runtime outcome reported honestly, idempotence, re-pairing afterwards leaving
  exactly one *new* owner, readiness agreeing that nothing is configured, and no
  credential in the response -- plus 4 UI cases covering confirmation, the single
  authoritative call, a parked removal reported as information, and a failure not
  telling the page it succeeded.
- **Sweep update, 2026-09-17 — the direct-replacement gap closed.** The earlier
  round covered delete-then-pair and rejected-candidate-preserves-old, but not a
  pair-over-pair with no intervening removal, which is what Replace actually
  does. `TestDirectReplacementLeavesExactlyOneOwnerAndToken` now asserts exactly
  one owner and the new token remain. The Telegram generation/intake/401 fixes
  are untouched. The generation-overlap and stale-identity invariants remain
  covered by `TestTelegramReadinessRejectsThePreviousGenerationWhileConfigApplies`,
  `TestChannelNamesItsPollingGenerationAndRetiresIt` and
  `TestStartCleanupRequiresTheSameActiveGeneration`.
- **Status:** FIXED IN SOURCE — physical confirmation required.

### PC-DEF-061 — The first owner message after a managed pairing was not received

- **Discovered:** owner physical desktop + Samsung verification, 2026-09-15.
- **Component:** `pkg/channels/telegram/telegram.go` (`Start`), new
  `pkg/channels/telegram/polling.go`.
- **Symptom, physically reproduced:** immediately after the newly created bot's
  chat became available the owner sent `/start` and got no answer; a second
  `/start` was answered normally. The device log shows one `/start`-sized update
  at 21:51:59 and no earlier one, so the first was lost before the agent
  pipeline. Everything else in the managed desktop flow worked, including 14/14
  command registration.

**The audit, because the owner's instruction was to prove where the update went
rather than add a delay. Each of these is a read of the real path, not an
inference:**

- **PocketClaw persists no Telegram update offset.** There is no `update_id`,
  offset or last-update state anywhere in `pkg/channels/telegram`. The offset
  exists only inside Telego's polling loop, which copies the params it is given,
  and every `Start` passes an unset offset -- which asks Telegram for everything
  it still holds. **A replaced or reconnected bot therefore cannot inherit an
  offset, and cannot skip its own first updates.** That is the owner's
  bot-identity question answered by construction, and it is now pinned by a test.
- **PocketClaw makes no webhook call at all** -- no `setWebhook`,
  `deleteWebhook` or `getWebhookInfo` -- and never passes
  `drop_pending_updates`. It sets no `allowed_updates`, so it asks for every
  update type.
- **The onboarding service does not consume the child bot's updates.** Its
  Telegram client is constructed once, with the *manager* bot's token; its
  `setWebhook` (which does carry `drop_pending_updates=true`) is therefore the
  manager bot's webhook. The child token is retrieved by `getManagedBotToken`,
  stored, and delivered -- the service never builds a client with it, so it
  issues no `getUpdates` and sets no webhook on the new bot.
- **The long-poll timeout is not racing the HTTP client.** `telegramHTTPTimeout`
  is 45s against a 30s long poll, so a poll is never aborted client-side.
- **Telego does not discard pending updates.** Its loop sends on a 100-deep
  buffered channel and blocks rather than dropping, and it calls no cleanup
  method before polling.

**The defect this found, which is real, narrow and proven by test:** long polling
is at-least-once only while the client behaves. `getUpdates` returns a batch and
the *next* call, carrying the advanced offset, is what makes Telegram delete that
batch permanently. Telego's loop issues that next call immediately. So the gap
between the poller starting and the handler consuming is the one place an update
that already arrived can still be lost -- and `Start` put a **blocking `getMe`
inside that gap**: `c.bot.Username()`, used to name the bot in the connect log,
performs a `getMe` on first use, which the device measured at **four seconds**
(21:51:42 to 21:51:46). `SetRunning(true)` and the lifecycle-started event both
fired inside that window, so the channel reported **Running while nothing could
receive**, and anything that ended the channel in those four seconds destroyed
whatever polling had already fetched and had already told Telegram to forget.

- **Resolution:** the consumer goes live first. `bh.Start()` is launched
  immediately after the handler is wired, `Running` is not reported until the
  handler confirms it is consuming (a yield loop on the handler's own state, not
  a delay), and the identity call moved off the intake path into its own
  goroutine -- the bot is still named in the log, since a replaced managed bot
  has to be tellable from the one before it. Only allocation now separates the
  poller from the consumer. A new `observeUpdates` forwarder, deliberately
  unbuffered so it adds no second place an update can sit, logs
  `polling.update_delivered` with `update_id` and the offset that delivery
  confirms, and logs `polling.update_dropped` as a **WARN** when an update is
  lost to shutdown -- the silent version of that was indistinguishable from
  Telegram never having sent it.
- **Verification:** 11 cases in `polling_test.go`, driving the real `Start`
  against a stubbed Telegram. Proven to catch the regression: with the old
  ordering restored, `TestFirstPollUpdateIsDeliveredWhileGetMeIsStillBlocked`
  does not merely fail, it hangs until the test timeout, because `Start` never
  returns while `getMe` is blocked. The bounds in those tests are generous on
  purpose -- what proves intake is independent of `getMe` is that `getMe` stays
  blocked until after the assertion, not a short deadline. Also pinned: a
  pending update delivered in the first poll is processed; it is processed
  exactly once and the following poll carries `update_id+1`; a replaced bot polls
  from an unset offset and receives an update id far below the previous bot's;
  a restarted channel receives the next message; an unpaired sender still never
  reaches the agent; and the new log fields survive the redaction layer
  (`pkg/logger/field_redaction_test.go`), without which the instrumentation would
  prove nothing.
- **What this does NOT claim.** It does not explain the observed loss on its
  own. The device log shows Telegram returning no update at all before 21:51:59,
  and at 21:51:42 the first poll asked with an unset offset; by elimination the
  update was already gone from Telegram's queue, and neither repository issues a
  call that could have consumed it. The one hop neither repository can audit is
  Telegram's own queueing across managed-bot token issuance. **The next physical
  run settles it:** with DEBUG on, `polling.started` followed by
  `polling.update_delivered … first_update=true` for the owner's *second* message
  and none for the first proves polling was live and consuming while Telegram
  returned nothing -- a Telegram-side drop, not a PocketClaw one.
- **Also found, reported rather than built:** the desktop pairing reports
  `applied` when the gateway process has been restarted, not when Telegram is
  receiving, so the UI's "connected" can precede intake. It did not cause this
  failure -- the owner's first `/start` preceded even the completion response,
  since the bot's chat exists in Telegram before PocketClaw has the token -- and
  gating it needs the authenticated health-detail token plumbed into the backend.
  Named here for an owner decision rather than bundled into this fix.
- **Update, 2026-09-15 — the ordering fix is confirmed and the readiness gap is
  now the blocker.** The new instrumentation shows the intended order on the
  device: `polling.started` and `polling.ready` at 02:58:42, with
  `Telegram bot identity resolved` only at 02:58:48 -- so the getMe call is
  provably off the intake path and the four-second window is gone. A first
  `/start` was still unanswered, and the minute-resolution Telegram timestamp
  cannot prove whether it was sent just before or just after `polling.ready`, so
  **no claim is made about a Telegram-side drop.**
  What that boundary *does* settle is that the product contract was unmet either
  way: the Dashboard said Connected when the gateway had been restarted, not when
  Telegram was receiving, so a user could be invited to send the first message
  into a channel that was still starting. Readiness is now authoritative.
  `status.Channel` carries a **three-valued** `commands_registered` -- absent
  means this channel publishes no menu, which is not the same as a menu that has
  not landed, and a gate that could not tell those apart would wait forever --
  and `GET /api/telegram/readiness` maps the gateway's own snapshot to the
  lifecycle states the UI renders. Completion no longer announces anything: the
  managed flow waits, names the stage it is waiting on, and announces Connected
  only on `ready`. The wait is bounded at 90s and offers Check again rather than
  claiming anything on expiry. The credential for the authenticated detail probe
  is the gateway's own bearer token, read from the private token file on Android
  and from the pid record on desktop -- nothing new is minted.
  **A second real bug surfaced while testing the restart case:** `Stop` returned
  while Telego still held its long-polling lock, so `Start` on a stopped channel
  failed with "long polling already running" and left Telegram down. `Stop` now
  waits for the poller to unwind, bounded, and says so if it does not.
- **Status:** **PHYSICALLY VERIFIED PASS**, 2026-09-15. On a fresh install the log
  shows `polling.started` and `polling.ready` at 06:18:13 and the first delivered
  update at 06:18:33 with `first_update=true` and `message_chars=6` — the owner's
  `/start`, answered on the **first** send. Both halves are therefore confirmed on
  the device: the intake ordering and the readiness gate that made "Connected" mean
  receiving. **The Telegram intake and readiness logic is not to be modified without
  new contradictory evidence.**

- **Regression reopened, 2026-09-17 — contradictory physical Android evidence.**
  A newly paired bot again ignored its first native Telegram `/start` and answered
  the second; replacing it with another bot answered the first send. The audit found
  an Android-only ordering race rather than a command-definition failure. Core's
  authoritative credential writer already saved the token/owner and applied the
  change by restarting the Gateway, but Flutter then queued a second whole-service
  `ACTION_RESTART`. That Android call returns when the intent is submitted, not when
  stop/start settles, so the readiness poll could observe the first Gateway as READY
  and open the final `t.me/<bot>` handoff immediately before the queued restart tore
  that receiver down. An update fetched in that shutdown window follows the existing
  `polling.update_dropped` path and cannot be redelivered. Desktop does not perform
  this second service restart and therefore does not have this ordering.
- **Second violated edge found in the same audit:** Telegram `Start` logged
  `polling.ready_unconfirmed` after its bounded handler check but nevertheless set
  `Running=true`. That contradicted both the source comment and this entry's earlier
  claim that Running meant consumption. Start now fails closed and cleans up the
  unconfirmed poller; no readiness consumer can authorize a handoff from that state.
- **Regression resolution:** managed Android onboarding performs exactly one
  authoritative config apply, then reads Core's authenticated readiness endpoint
  through a narrow loopback bridge instead of inferring readiness from Android's
  generic health snapshot. That endpoint rejects the previous bot generation while
  configuration is pending or applying, and requires the final Gateway snapshot to
  report both handler-backed Running and `commands_registered=true` before the bot
  chat opens. No delay was added. Safe timestamped lifecycle events now cover
  `config_applied`, `polling_live`, `consumer_attached`, `handler_ready`,
  `telegram_ready`, `handoff_opened`, `first_update_received`, and
  `first_start_replied` without recording bot, owner, chat, message, token or path
  data.
- **Regression verification:** deterministic Core tests make unconfirmed handler
  readiness fail closed and drive the real polling, built-in `/start`, and Telegram
  send paths for both fresh and replacement bot identities. Each bot starts from an
  unset offset, its first `/start` produces exactly one `Hello! I am PocketClaw.`
  send, and no duplicate inbound or reply appears. Flutter tests pin
  `config_applied → telegram_ready → handoff_opened` for fresh and replacement flows
  and prove no bot-chat URL opens while readiness is false.
- **Current status:** **REOPENED / FIXED IN SOURCE — physical confirmation required.**

- **Second reopen, 2026-09-17 — physical still fails; the receiver that answered
  was not the owner that acknowledged the first `/start`.** The failing run's log
  shows the active process becoming ready and its **first** delivered update being
  the owner's *second* message (`message_chars=3`, `first_update=true`,
  `next_offset` one past it). The first `/start`, sent a second earlier, was never
  returned to that process with an unset offset, so Telegram had already forgotten
  it: some getUpdates owner advanced past it before the surviving receiver's poll.
  Two in-repo facts were then proven, and both are generation-ownership defects:
  - **A replaced channel lost its worker.** `compareChannels` reports a changed
    channel as both removed and added. `Reload` stopped the old instance, built the
    replacement and its worker, and then ran the *outgoing* instance's deferred
    `UnregisterChannel(name)` — which closed the **replacement's** worker. The
    reloaded generation was left configured, started and polling, with no worker to
    send a reply through: exactly a generation that can acknowledge an update and
    never answer it. Proven by a new deterministic test
    (`TestReloadChangedChannelKeepsExactlyOneWorker`); fixed by releasing the
    outgoing instance synchronously and only while it is still the registered one.
  - **Readiness did not prove intake.** `polling_live` is emitted when Telego's
    poller goroutine is launched, and Running was set once the handler goroutine
    started. Telego returns from `UpdatesViaLongPolling` **before** its first
    `getUpdates` request is issued, so a handoff could be authorized while the Bot
    API was holding no poll for that generation. Readiness now requires the
    generation's first `getUpdates` request to be observed at the Bot API caller
    (`getUpdates_intake_established`); an unconfirmed generation fails closed and
    tears both halves down.
- **Generation identity is now end-to-end.** Every activation is a distinct local
  polling generation (`telegram_generation=N`, a process-local counter carrying no
  bot, owner, chat or credential identity). `status.Channel` carries the active
  generation; readiness refuses to report `ready` for a Running channel that names
  no generation (`generation_unconfirmed`); the readiness answer includes the
  generation; and the Android client requires the **same** generation on two
  consecutive reads before it opens the final bot chat. A superseded generation's
  success can no longer authorize the handoff that belongs to its replacement.
- **What this does not yet claim.** The log alone cannot say whether the owner that
  acknowledged the first `/start` was a retired in-process generation or the remote
  onboarding service (which is not in this repository). The generation-tagged
  lifecycle events — `generation_created`, `poller_created`,
  `getUpdates_intake_established`, `first_update_received`,
  `generation_retiring`, `poller_cancel_requested`, `poller_exit_confirmed`,
  `generation_retired` — plus the generation in every readiness answer are what the
  next physical run needs to name that owner definitively.
- **Second-reopen status:** **FIXED IN SOURCE — physical confirmation required.**

### PC-DEF-060 — Desktop Dashboard had no managed Telegram onboarding

- **Discovered:** owner UI observation, 2026-09-14.
- **Component:** new `pkg/telegramonboarding`,
  `web/backend/api/telegram_onboarding.go`, `web/backend/api/android_bridge.go`
  (writer extracted), `web/frontend/src/api/telegram-onboarding.ts`,
  `channel-forms/telegram-desktop-connect.tsx`, `telegram-surface.ts`,
  `telegram-panel.tsx`, `android/app/build.gradle.kts`, `service/PocketClawService.kt`.
- **Audit first, as the owner asked. The cause is the host-bridge check — not
  responsive CSS, not a user-agent test, not a different route or component.**
  `Channels → Telegram` is the same route and component on both clients. It resolves
  a surface from two facts: `configured`, and
  `isTelegramOnboardingAvailable(host)` = `host !== null && host.onboardingConfigured`.
  `getPocketClawHost()` reads `window.__pocketclawHost`, which only the Android
  WebView injects. In a desktop browser there is no host, so the surface is
  `manual-only`.

  Managed pairing genuinely cannot run in a browser: it needs the native flow to
  launch Telegram and write the token into Core's config. The feature is not
  mobile-only by preference; it is mobile-only by mechanism.

- **What the owner's report gets right, and where it overstates.** Desktop does
  **not** silently omit Telegram management: the manual token form is the whole page
  there, with an explanation above it, and an already-connected channel shows its
  connected summary on desktop because `configured` outranks the host check. So
  there is a working management path.

  The real defect is narrower and worth fixing: the explanation said **"One-tap bot
  creation is not available in this build."** On a desktop browser that is simply
  untrue — the build has the feature, this client cannot run it — and it sends the
  user looking for a different build instead of telling them where one-tap setup
  lives. One sentence was serving two different causes.

- **Resolution, kept to that.** `resolveTelegramManualReason` separates `no-host`
  (a browser) from `host-without-endpoint` (a build compiled without the onboarding
  URL), and each gets its own wording. The desktop copy names the actual route:
  open PocketClaw on the phone for one-tap setup, or create a bot with BotFather and
  paste its token below — with a BotFather link, because the manual path was
  otherwise a form with no starting point. The surface itself is unchanged, and no
  Android-only action was copied to desktop.
- **Deliberately not done:** a browser-side managed flow. Pairing needs an HTTPS
  conversation with the onboarding service and a write into Core's config; doing it
  from the console would mean either cross-origin calls the service does not permit
  or a second implementation of the whole pairing state machine in Go. That is its
  own milestone, not a copy of an Android action, and the owner's instruction was
  explicitly not to blindly copy one.
- **Unchanged:** the owner contract. Nothing here touches `AllowFrom`, the
  OwnerUserID pairing, or the token path.
- **Verification:** 9 cases in `telegram-surface.test.ts` — a connected channel
  first whatever the client, managed onboarding offered only where it can run, the
  manual form staying visible when it is the only path, the browser reason not
  blaming the build, the build blamed only when a host is present without an
  endpoint, and an existing configuration manageable without a host as its own
  stated rule. Two i18n keys added in all 14 locales; parity suite green.
- **Resolution, second pass — managed onboarding now runs from desktop.** The owner's
  architecture, built as specified. Core performs the pairing and the browser only ever
  calls same origin:
  - `pkg/telegramonboarding` is a Go client for the onboarding service, matching the Dart
    client's wire contract field for field, because both speak to the same deployment and
    a drift between them would be a pairing that works from one client and not the other.
  - `GET /api/telegram/onboarding` reports availability;
    `POST …/pairings` starts one; `GET …/pairings/{id}` polls;
    `POST …/pairings/{id}/complete` collects and configures;
    `DELETE …/pairings/{id}` forgets it. None is in the launcher auth allowlist, so all
    require a Dashboard session like every other `/api` path.
  - Completion goes through **`writeTelegramCredentials`**, extracted from
    `handleAndroidTelegramConfigure` so the Android bridge and the desktop flow share one
    writer. The owner contract lives there and is not reimplemented: `AllowFrom` becomes
    exactly one positive numeric owner, and PC-DEF-030's apply runs as part of saving.
- **What the browser never receives, asserted rather than assumed.**
  - **The poll token.** It authorises status reads and the single token collection, so
    Core holds it against the pairing id and the browser only ever names the id. Two
    tests assert it appears in no response, and the fake service 404s without it — so a
    passing status poll proves Core supplied it.
  - **The onboarding service's URL**, keeping PC-DEF-052's no-visible-hosting-origin rule
    for this client too.
  - **The bot token**, at any point. It goes from the service into Core's configuration
    and is never serialised back.
- **The blocker resolved, with one source of truth.** Core did not know where the
  onboarding service lives. The URL now reaches it as `POCKETCLAW_ONBOARDING_BASE_URL` in
  the service environment: Gradle already decodes the dart-defines, so
  `android/official-onboarding.properties` → dart-define → `BuildConfig` → Core's
  environment, with no second place to set it. A deployment without it reports managed
  onboarding unavailable and the manual form remains. Only `https` is accepted — plain
  HTTP would put the poll token, and once the bot token, on the wire in the clear.
- **Lifecycle:** cancellation drops the stored poll token, an unmounted tab cancels its
  own unfinished pairing, completion is guarded so a second poll tick cannot retry a
  token the service delivers exactly once, expiry and an unrecognised state both end the
  flow rather than polling forever, and a parked change is reported as information
  because PC-DEF-030 says that is not a failure.
- **Deliberately not built: the QR code.** The owner listed it as one option among
  several. Rendering one needs a new frontend dependency, and `pnpm` is not on PATH in
  this environment, so the other-device case is served by an openable *and copyable*
  Telegram link instead. Adding QR is a small follow-up once the dependency question is
  settled; it is named here rather than silently skipped.
- **Verification:** 10 cases in `web/backend/api/telegram_onboarding_test.go` —
  availability both ways, the poll token absent from create and status responses, the
  service URL absent, the status poll proving Core supplied the stored token, completion
  configuring Telegram with exactly one owner and never returning the bot token,
  completion not replayable, an unknown pairing reported as expired rather than probed,
  cancellation dropping the token, and retry. Plus 13 in
  `telegram-desktop-connect.test.tsx` covering the UI lifecycle, that only the Telegram
  link is ever opened, and that nothing credential-shaped is rendered. 19 i18n keys added
  in all 14 locales.
- **Status:** **PHYSICALLY VERIFIED PASS**, 2026-09-15, on the desktop Dashboard and the
  Samsung. Confirmed on the device: managed pairing is offered from a desktop browser, the
  suggested bot is generated, Telegram opens directly, **no `*.vercel.app` page is ever
  shown**, bot creation succeeds, PocketClaw receives and configures the bot, the Telegram
  channel starts, the bot answers, and command registration reached Telegram — the log
  shows `getMyCommands ok=true`, `setMyCommands ok=true`, `defined=14 sent=14`. The
  command menu is **not** a defect: it was reached by typing `/`, and the log proves
  registration succeeded. Not to be reopened without contradictory evidence.
  The one remaining observation from that run — the owner's **first** `/start` going
  unanswered while the second was answered — is split out as **PC-DEF-061** rather than
  reopening this entry, because every step of the pairing itself passed.

### PC-DEF-058 — First run never requested Android notification permission

**STILL PHYSICALLY FAILED, 2026-09-15 (third attempt). Audited before patching, as
instructed, and one input is provably wrong.**

Ruled out from the built artifact rather than from source: `targetSdkVersion` is
**36**, so `POST_NOTIFICATIONS` is a runtime permission and the dialog is
requestable; and `aapt2 dump permissions` on the packaged APK confirms
`android.permission.POST_NOTIFICATIONS` is declared. Neither is the cause.

**What is provably wrong: the "have we asked" record was backup-eligible.** It lived
in `shared_prefs/pocketclaw_prefs.xml`, and this app ships `allowBackup="true"` with
only three *file*-domain paths excluded — `credentials/`, `pocketclaw-core/` and
`picoclaw/`. The preference store is not among them, so a reinstall can restore
`notification_permission_asked = true` from a **previous** install. The state
machine then resolves `DENIED`, whose action is "offer Settings, do not ask", and a
genuinely fresh install never sees the dialog. That is the behaviour the device
showed, and it is consistent with the second attempt's resume trigger being correct
and still silent.

"Have we asked *this install*" is per-install state by definition, so it is now a
marker file under `noBackupFilesDir`, which a restore cannot reach. The legacy
preference key is deliberately **not** migrated: reading it would carry the restored
value straight back. The cost is that an upgrade from an older build may raise the
dialog once more, and only for someone who has not already granted it — a granted
permission short-circuits before the record is consulted.

**Instrumented, because the source has now looked correct twice.** One DEBUG line at
the decision and one at the result, carrying exactly the safe state the owner
specified and nothing else: `android_api_level`,
`notification_permission_declared` (read from the installed manifest via
`PackageManager`, so it answers the declaration question on the device),
`notification_permission_granted`, `should_show_rationale`, `asked_marker`,
`resolved_state`, `activity_lifecycle_state`, `returned_from_all_files_settings`,
`prompt_shown_this_launch`, `notifications_enabled`, `should_request`, then
`notification_request_attempted` and `request_result`. No chat, account or
credential value appears in any of it.

- **Status:** FIXED IN SOURCE (third attempt), with instrumentation — physical
  fresh-install confirmation required. If the dialog still does not appear, the new
  log line names which input is wrong.

**REOPENED 2026-09-15 — physically failed again on a fresh install, and the cause
was placement, not policy.** The manifest entry, the SDK gate, the
`NotificationPermissionPolicy` rules and the platform call were all correct and all
**unreached**: the only trigger was `ConfigPage.initState`, and a fresh install
never opens Settings — `MainShell` starts on the Dashboard at index 0, and Settings
is index 3. So the dialog was never raised, exactly as reported: the storage screen
appeared and the notification dialog did not.

Asking now happens on the resume path, which every launch takes.
`MainActivity.onResume` asks after the all-files-access check, and
`requestAllFilesAccessIfNeeded` reports whether *this* resume sent the user to
Settings — when it did, the ask waits for the resume that comes back rather than
stacking a dialog behind a screen the user is being sent to, which is the order the
owner requires. The per-launch guard and the persisted asked-record are shared with
the Flutter-initiated path, so there is one decision and one record; the Settings
page keeps its request as a second chance and keeps showing the state and the
recovery action.

`NotificationPermissionPolicy.shouldRequestOnResume` holds the rule and is tested:
a fresh install asks; a resume that just launched the storage screen does not, and
the next one does; one dialog per launch; and granted, refused or inapplicable never
ask. The placement itself is not unit-testable — that is what the physical check is
for, and why the journey gate exists.

- **Status:** FIXED IN SOURCE (second attempt) — physical fresh-install
  confirmation required.

#### Original entry


- **Discovered:** Samsung physical testing, 2026-09-14, owner-reported.
- **Component:** `android/.../NotificationPermission.kt`,
  `PocketClawMethodChannel.kt`, `MainActivity.kt`, `PocketClawPreferences.kt`,
  `lib/src/core/pocketclaw_channel.dart`, `lib/src/ui/config_page.dart`.
- **Problem:** `POST_NOTIFICATIONS` **was already declared** in the manifest and
  **never requested at runtime**, so on a fresh install the persistent "PocketClaw
  Running (PID …)" notification simply never appeared and the owner had to enable
  notifications by hand in Android Settings. The storage flow beside it jumps
  straight to Settings, which is correct for `MANAGE_EXTERNAL_STORAGE` — that
  permission has no runtime dialog — and that shape was applied to a permission
  which does have one.
- **Resolution.** The rules live in `NotificationPermissionPolicy`, separate from
  the platform calls, because they are what is worth testing:
  - Android 13 (API 33) is the boundary. Below it there is no runtime permission,
    and `NOT_REQUIRED` is reported rather than a false denial.
  - PocketClaw keeps **its own record of having asked**. Android's
    `shouldShowRequestPermissionRationale` cannot answer this — it returns false
    both before the first ask and after a permanent refusal — so it cannot tell
    "never asked" from "refused for good".
  - Asked once. A refusal yields `OFFER_SETTINGS`, never a second dialog: Android
    stops showing it after a refusal, so re-requesting is a silent no-op that
    makes the app look broken.
  - "Permission granted" and "the notification will appear" are kept as separate
    facts. On every Android version the user can switch notifications off in
    Settings, and below API 33 that is the only control there is, so visibility is
    reported from `areNotificationsEnabled()` as well as the permission.
  - The ask is recorded **before** the dialog, not after: the callback does not
    fire if the activity is recreated mid-dialog, and an unrecorded ask would
    re-prompt on the next launch.
  - The request runs after the first frame, so the app is on screen behind the
    system dialog rather than the dialog being the first thing a fresh install
    shows.
  - Settings gains a tile showing the state, with an **Open notification
    settings** action that appears only when one is needed, and re-reads the state
    on return. Labelled, 40px, for the touch reasons in PC-DEF-047/055.
- **Denial is not a failure:** nothing gates on the permission. PocketClaw keeps
  running; only the notification is absent, which is what Android decided.
- **Verification:** 7 cases in
  `android/app/src/test/kotlin/.../NotificationPermissionPolicyTest.kt` — the
  API-33 boundary across four SDK levels, fresh install, granted with and without
  a prior ask, asked-and-refused, the dialog offered exactly once, visibility
  following the notification switch on every version, and the wire names the Dart
  side matches on. Android unit suite 26 tests, 0 failures. Four l10n keys added
  in all 12 locales.
- **Status:** FIXED IN SOURCE. **Physical confirmation required** — fresh install,
  the system dialog appears, grant, start the service, "PocketClaw Running" is
  visible with no trip through Android app settings.

### PC-DEF-059 — Authentication discarded the requested Dashboard destination

- **Discovered:** Samsung physical testing, 2026-09-14, owner-reproduced.
- **Component:** `web/frontend/src/lib/post-auth-destination.ts` (new),
  `routes/__root.tsx`, `routes/launcher-login.tsx`.
- **Problem:** the native Settings cards open the console at
  `canonicalModelsConsolePath` = `/models` and `canonicalTelegramConsolePath` =
  `/channels/telegram`. When the Dashboard session had expired, the root auth
  guard redirected to `/launcher-login`, and on success the login page called
  `globalThis.location.assign("/")` — unconditionally Chat/Home. The destination
  the user asked for was discarded, so they had to navigate inside the Dashboard
  or go back to native Settings and tap the same thing again.
- **Resolution:** the destination travels through the auth flow as `?next=`. The
  guard builds the login URL with `launcherLoginUrlFor(...)`, and the login page
  resolves it once on mount — captured in state rather than re-read per submit, so
  a wrong password followed by the right one still lands on the original
  destination.
- **Security: `next` is untrusted input that ends in a navigation, so it is
  matched against the route set rather than sanitised.** An allowlist cannot be
  talked into an external host and does not depend on completing a denylist.
  Rejected: any scheme (`https:`, `javascript:`, `data:`, `mailto:`, `vbscript:`,
  `file:`), protocol-relative `//host`, backslash smuggling (`/\evil.example` —
  a browser can read `\` as `/`), control characters and header-injection
  newlines, traversal, unrooted relative paths, unknown internal paths, a channel
  segment that is not one plain segment, and the auth pages themselves (returning
  to login after logging in is a loop). Anything rejected falls back to home.
  The routes were read from the generated route tree rather than invented, and no
  new route was added.
- **Unchanged:** a direct login with no request still goes to home, so the
  ordinary path behaves exactly as before, and dashboard authentication itself is
  untouched.
- **First attempt: PHYSICALLY FAILED**, Samsung, 2026-09-14. It was frontend-only,
  and it did nothing on the device.

- **Why it failed, traced through the real path rather than the route tests.**
  `lib/main.dart`'s `_webUrl` loads the WebView at
  `http://127.0.0.1:18800/models?lng=en`, and for an unauthenticated request
  `rejectLauncherDashboardAuth`
  (`web/backend/middleware/launcher_dashboard_auth.go`) answered
  **`http.Redirect(w, r, "/launcher-login", 302)`** — server-side, before one line
  of JavaScript loaded. So:
  - the exact destination URL before auth was `/models?lng=en`;
  - `next` was **absent**, because the server never put one there;
  - the login page therefore received nothing and its `readPostAuthDestination`
    correctly fell back to `/`;
  - the router guard in `__root.tsx` that built the `?next=` URL **never ran**: it
    lives on the `/models` page, and that page was never served.

  The frontend half was not wrong, it was unreachable. The whole previous round's
  coverage was route-level, and a route-level test cannot see a redirect that
  happens before the routes exist.

- **Second resolution:** the destination now survives the server redirect.
  `web/backend/middleware/post_auth_destination.go` builds
  `/launcher-login?next=<path>` for a rejected page request whose path is a
  Dashboard route, and the bare login path for anything else — so an ordinary
  unauthenticated visit is unchanged. API and websocket rejections keep their 401
  shapes; a 302 to an HTML page is not a useful answer to a fetch.

  The frontend half is kept and is now reachable: the login page reads
  `globalThis.location.search` **once on mount into state**, so a wrong password
  followed by the right one still lands on the original destination, and a later
  router rewrite cannot take it away. `/launcher-login` has no `validateSearch` or
  `beforeLoad`, so the query is not stripped before that read.

- **Security, restated for the server side.** The value goes into a `Location` a
  browser follows and the path comes from the request, so anyone may ask for
  anything. It is matched against an allowlist, never sanitised: rejected are any
  scheme or colon-bearing first segment, protocol-relative `//host`, backslashes,
  control characters and CR/LF (header splitting), traversal, unrooted paths,
  unknown routes, a `/channels/<name>` segment that is not one plain lowercase
  segment, and the auth pages. A **request-supplied `next` is never reflected** —
  the middleware decides the destination from the path it rejected, which is
  asserted directly.
- **Drift is guarded mechanically.** The backend cannot import the frontend's
  allowlist, so it is duplicated — and
  `TestPostAuthDestinationsMatchTheFrontendAllowlist` parses
  `post-auth-destination.ts` and fails if the two diverge. That guard was confirmed
  to fail on a deliberately removed route, so it is not a vacuous pass.
- **Verification:** 8 middleware cases driven by the URLs the native app actually
  launches, including `?lng=en` exactly as `_webUrl` builds it — the two native
  destinations surviving, the full unauthenticated request through the real
  middleware, home and auth pages staying bare, API/websocket rejections
  unchanged, 20 hostile destinations refused, every `Location` same-origin
  relative, a request-supplied `next` ignored, and the drift guard. Plus the 18
  frontend cases from the first attempt, which still hold. Middleware package 40
  tests.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS**, Samsung, 2026-09-14. Native
  Settings → Manage Telegram / Manage Models → authentication → the requested
  Dashboard destination. Not to be reopened without contradictory evidence.

  The lesson is recorded because it generalises: the first attempt passed every test
  it had and failed on the device, because all of that coverage was route-level and
  the redirect that discarded the destination happened before the routes existed. A
  navigation fix has to be driven by the URL the native app actually launches.

### PC-DEF-040 — REOPENED: Public Mode was never reconciled after the dashboard was claimed

- **Recorded here for the first time.** The original fix (`66e211f`) lived only in
  commit messages and code comments, which is part of why its structural gap went
  unexamined.
- **Reopened:** Samsung + desktop, 2026-09-14, **physically reproduced**. Public Mode
  desired ON, dashboard protection active, desktop cannot reach the LAN Dashboard;
  toggling Public Mode OFF then ON makes it reachable immediately. So the desired
  state existed and the effective binding was never reconciled to it.
- **Component:** `web/backend/launcher_http_runtime.go`, `web/backend/api/auth.go`,
  `web/backend/main.go`.
- **Root cause — the original fix detected the wrong event.** `PC-DEF-039` narrows an
  unclaimed dashboard to loopback whatever the user asked for, so "Public Mode ON" and
  "bound to the LAN" necessarily diverge until something re-applies the preference.
  The only thing that ever did was **the Android app noticing its embedded WebView
  navigate away from `/launcher-setup`** (`_maybeReconcilePublicMode` →
  `reconcilePublicModeAfterSetup` → `applyPublicMode(true)`).

  That is an inference from one client's UI navigation, not the event itself. Any
  route to a first claim that does not pass through exactly that transition — and
  there are several, including a claim made in a session where the WebView never
  reaches the setup page — leaves the listener on loopback with a manual Public Mode
  toggle as the only recovery. The network-mode bridge and the live rebind were never
  at fault, which is exactly why OFF→ON worked.
- **Audited at the owner's request: the recent launcher-login / destination changes
  are NOT the cause.** `PC-DEF-059`'s work touched `launcher-login.tsx`,
  `__root.tsx` and `rejectLauncherDashboardAuth`. The last commits to
  `launcher-setup.tsx`, `webview_android.dart`, `service_manager.dart` and
  `public_mode_reconciliation.dart` all predate this session — `66e211f` and
  `aa44135`. The setup path and the reconciliation hook were not modified, and the
  `?next=` redirect does not bypass them: an uninitialized dashboard still lands on
  `/launcher-setup`, and `Uri.path` ignores the query so the hook's URL match is
  unaffected. This is the original fix being incomplete, not a regression I introduced.
- **Resolution — react to the claim, not to a navigation.** The authoritative moment
  is `POST /api/auth/setup` succeeding on a **first** claim, and it happens inside
  Core, which already owns both the listener and the desired preference. The runtime
  now remembers `desiredPublic` alongside the effective value and exposes
  `ReconcileAfterDashboardClaimed()`, which the setup handler calls after a first
  claim — after the response, because applying it replaces the listener carrying that
  request.
- **PC-DEF-039 is intact, and that mattered more than the fix.** The hook fires only
  for a **first** claim, and that path already refuses any non-loopback request, so
  this only ever re-applies the owner's own preference to a dashboard a local owner
  has just taken. A remote claim is rejected before the hook, a failed or malformed
  claim never reaches it, and a password change on an already-owned dashboard does not
  fire it. An unclaimed dashboard is still narrowed to loopback with Public Mode
  desired.
- **Also fixed while here:** an explicit `ApplyPublicMode` now updates
  `desiredPublic` too. Without that, turning Public Mode off and then claiming a
  dashboard would have re-widened it from a desire the user had retracted.
- **The Android-side hook is kept**, unchanged, as a second path. Two independent
  detectors is the same posture PC-DEF-039 takes about the takeover itself.
- **Verification:** 6 cases in `web/backend/public_mode_reconciliation_test.go` —
  the owner's case B (claim applies the requested LAN binding with no toggle), case A
  (a private dashboard is not exposed by being claimed), idempotence, a retracted
  desire not resurrected, case C (desired and effective stay coherent across explicit
  changes), and PC-DEF-039's narrowing asserted beside it. Plus 5 in
  `web/backend/api/auth_claim_hook_test.go` — a first claim fires it exactly once, a
  password change does not, a remote first claim is refused and fires nothing, four
  shapes of failed claim fire nothing, and a host with no controller still succeeds.
- **Status:** FIXED IN SOURCE. **Physical confirmation required** — case D (fresh
  install, Public Mode ON, remote first-claim still impossible) needs the device, and
  so does the headline case.

### PC-DEF-056 — The final bot chat opened before the Telegram runtime was ready

- **Owner's number:** PC-DEF-054, which was already in use for the signature
  plaintext defect below. Recorded as 056, the next free identifier.
- **Discovered:** Samsung physical testing, 2026-09-14, owner-reproduced.
- **Component:** `lib/src/ui/telegram_onboarding_page.dart`,
  `lib/src/telegram/telegram_onboarding_controller.dart`.
- **Symptom, exactly as reproduced:** Connect → Telegram opens → bot creation
  completes → Telegram opens the new bot chat → **the bot does not answer and has
  no command menu**. Returning to PocketClaw, opening the Telegram tab and
  pressing Open Chat a second time made everything work. No manual Service or
  Gateway restart was needed, which is what distinguishes this from PC-DEF-030.
- **Root cause — PROVEN in source, two independent faults.**
  1. **Polling stopped while the app was backgrounded.**
     `telegram_onboarding_page.dart` called `controller.pausePolling()` on
     `paused`/`hidden`/`detached` — and that window is precisely the time the user
     spends in Telegram confirming the bot. So `fetchStatus` was never called
     while the pairing became ready, `collectCredentials` never ran, the token was
     never written, and Core had no Telegram channel at all. The bot chat Telegram
     navigated to was therefore a bot PocketClaw had not finished creating. On
     return, `resumePolling()` polled immediately, collected the token, wrote the
     config and reloaded Core — which is why the second Open Chat worked and why
     no manual restart was involved.
  2. **`connected` was declared on config-applied, not on runtime-running.**
     `_completePairing` set the `connected` stage as soon as `_reloadCore()`
     returned, and that returns when the restart has been *requested*:
     `ServiceManager.restartCore` hands Android one intent and comes back. Even
     once the token was consumed, the flow could present a ready bot before the
     channel had started.
- **Resolution.**
  - Polling continues across the handoff. It stays bounded by the pairing's own
    `expiresAt`, and if Android kills the timer or the process instead, the
    existing `restore()` plus the resume catch-up still picks it up — so nothing
    now depends on the foreground loop being alive at the moment the result
    lands.
  - A new `startingRuntime` stage, and `connected` is reached only when Core
    reports the Telegram channel running. Readiness is asked through
    `resolveTelegramRuntimeState` — PC-DEF-027's single definition of "Telegram is
    running" — rather than a second definition invented here. The wait is
    **bounded** (45 s in production) and polls; it is not a sleep. A status read
    that fails or a Core that is not reporting yet counts as silence, not failure,
    which is PC-DEF-027's rule applied to a restart in progress.
  - On reaching `connected` the flow **opens the bot chat itself**, exactly once,
    so reaching a working bot takes no second action. A failure to open is not an
    error: the configuration is live and Open Chat remains on the screen.
  - A runtime that never comes up produces `runtimeNotReady`: "your bot is saved,
    but Telegram has not started yet". The token and owner are persisted and
    sound, so the user is told to retry the start, never that setup failed — and
    is never sent into a dead chat.
- **A real bug the tests found before the device could.** `reset()` during the
  readiness wait did not abort it, so a cancelled pairing whose runtime came up
  later went on to declare itself connected and open a bot chat the user had
  walked away from. The wait now re-checks the stage every iteration, and a
  cancelled wait is not reported as a runtime failure either.
- **What did not change:** PC-DEF-052's no-visible-hosting-page fix is intact —
  the readiness gate sits after the launch, not in place of it. The owner contract
  (exactly one positive numeric owner in `AllowFrom`), token handling, reconnect,
  replace and PC-DEF-030's automatic apply are all untouched.
- **Verification:** 11 cases in `telegram_onboarding_controller_test.dart` under
  "PC-DEF-056 runtime readiness" — connected only once running is reported, the
  chat opened automatically, never opened before running, opened exactly once
  across resumes, a never-starting runtime reported rather than waited on, a
  throwing status read treated as not-yet-running, configuration applied exactly
  once across the wait, reconnect re-running the gate, cancellation opening
  nothing, the owner identity unchanged, and an already-running runtime not
  delayed. Plus the rewritten background test (polling continues) and the widget
  test at page level. Flutter suite 561 passed.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS** in the tested normal
  onboarding path, Samsung, 2026-09-14. Managed onboarding completes without the
  old second "Open Chat in Telegram" workaround.

  **Performance observation, not a defect:** the first Telegram reply can take
  about 15-25 s during initial runtime activation, after which the bot responds
  normally. Deliberately **not** optimised — no sleeps, no speculative change. The
  stages are instrumented instead, so a later session can attribute the wait
  rather than guess: `TelegramOnboardingController` now records
  `runtimeReadyLatency` and `onboardingLatency` and emits
  `pocketclaw.onboarding stage=telegram_running runtime_wait_ms=… onboarding_total_ms=…`
  at the moment Core first reports the channel running. Both marks reset when a new
  flow begins, so a retry never reports the previous flow's timing.

  The owner's number for this defect was PC-DEF-054, which was already the
  signature-plaintext entry; see the identifier note.

### PC-DEF-057 — Structured log fields were not recursively redacted

- **Discovered:** owner logging-hardening requirement, 2026-09-14.
- **Component:** `pkg/logger`, new `pkg/logger/field_redaction.go`.
- **Problem — two shapes a secret survived in.** `sanitizeFieldsForLog` matched a
  short list of **exact** field names and pattern-redacted string values. So:
  1. A field *named* for a credential whose value no pattern recognises went
     through verbatim. `token: "hunter2"` is not `sk-…`, has no vendor prefix and
     is not `KEY=value`, so every pattern declined it. The sensitive-name list
     held only `session_key`, `scope_key` and `route_main_session` — not
     `api_key`, `authorization`, `bot_token`, `password` or any of the rest.
  2. A secret nested inside a map, slice or struct never reached the string case
     at all: the `default:` branch handed the value straight to the encoder.
- **Resolution:** a central layer that classifies field **names** and walks
  values recursively. Names are normalised (`X-Api-Key`, `x_api_key`, `apiKey`
  compare equal) and matched on substrings, because credentials arrive under
  compound names nobody can enumerate — `bot_token`, `proxy_password`,
  `crypto_passphrase`, `channel_access_token`, `x-opencode-session`.
  Non-primitives are walked by JSON shape, so structs, nested structs and typed
  collections all go through the same key-aware pass and what is checked is
  exactly what the encoder would have written. The walk is depth-bounded.
- **What is deliberately kept, because DEBUG has to stay worth reading.** A bool
  is never redacted whatever it is called — `authorization_present=true` and
  `api_key_changed=true` are the point of logging them. Names ending `_present`,
  `_set`, `_configured`, `_changed`, `_count`, `_length`, `_digest`, `_hash`,
  `_type`, `_kind` are facts *about* a credential, and `auth_method`,
  `changed_fields`, `token_type` are metadata. Provider, model, protocol, method,
  sanitised endpoint, status, duration, counts, trace id and error class all
  survive.
- **`session` is treated as sensitive**, including bare. This codebase already
  redacted every session identifier it named, `x-opencode-session` is a secret
  header value, and the facts about a session survive through the suffix rule.
- **Also added:** `provider.request` / `provider.response` /
  `provider.transport_failed` DEBUG lines in the OpenAI-compatible provider — the
  owner's exemplar, and the line missing when an inference failure had to be
  reconstructed. Endpoints are reduced to scheme, host and path, so a key in
  `?key=`, a signed URL's signature, userinfo credentials and an auth fragment
  are all dropped rather than inspected. Header facts are booleans.
- **Verification:** 46 redaction cases in `pkg/logger/field_redaction_test.go`,
  driven by a canary (`POCKETCLAW_TEST_SECRET_123456`) chosen so that only the
  *name* rule can catch it — 23 credential field names, 8 nesting shapes
  (nested map, twice-nested, header map, slice of maps, struct, pointer to
  struct, struct in a map, string slice), pattern redaction still applying under
  an innocent name, useful metadata surviving, presence booleans surviving,
  numeric credentials redacted, existing omit/internal rules unregressed and now
  applied when nested, bounded depth, and an end-to-end pass through the real
  emit path at DEBUG/INFO/WARN/ERROR reading the writer's bytes. Plus 3 endpoint
  sanitisation cases. `pkg/logger` 50 tests pass.
- **Status:** **OPEN / PARTIAL**, at the owner's instruction. Redaction is proven
  safe and the two follow-ups below are fixed in source; it stays open until the
  Samsung DEBUG output is read again.

**Physically verified portion**, Samsung, 2026-09-14: **safe token metrics and
provider DEBUG fidelity PASS.** The device log shows `max_tokens=32768`,
`provider.request` with `authorization_present=true`, `custom_header_count=4`,
`endpoint=https://opencode.ai/zen/go/v1/chat/completions`,
`model=deepseek-v4-flash-vision-exp`, `request_bytes`, `session_header_present=true`,
`stream=false`, `tools=19`, and `provider.response` with `status=200`,
`content_type=application/json` and `duration_ms`. `prompt_tokens`,
`completion_tokens` and `total_tokens` are all visible, and no sensitive value
appears. PC-DEF-057 stays OPEN overall until the remaining logging-hardening
acceptance criteria are checked on the device.

#### Follow-up 1 — redaction was too aggressive, proven by the physical log

The device log showed `max_tokens=<redacted>`. `max_tokens` is a model parameter,
not a credential: the substring rule matched `token` inside it.

`token` is a substring of every credential worth hiding **and** of every usage
metric worth keeping, so it cannot be resolved by substring alone. Explicit safe
metadata is now evaluated **before** the broad secret match:

- an enumerated set of token *measurements* — `max_tokens`, `prompt_tokens`,
  `completion_tokens`, `total_tokens`, `reasoning_tokens`, `cached_tokens`,
  `input_tokens`/`output_tokens` and their `_details` breakdowns, `tokens_before`
  /`_after`, `used_tokens`, `token_count`, `prompt_token_count`,
  `summarize_token_percent`, and the generic `_tokens` / `_token_count` /
  `_token_percent` shapes;
- **the value's type decides where the name cannot.** `tokens` is a count as a log
  field (`pkg/seahorse`) and a **map of credentials** as a struct field
  (`pkg/channels/weixin`, `pkg/providers/cli`), so a numeric `tokens` is a metric
  and a string or string-map `tokens` stays redacted;
- `max_tokens_field` names a config field, so it joins `changed_fields` as a name
  rather than a value;
- a bool is never redacted whatever it is called, stated in the predicate as well
  as the sanitizer so the two cannot disagree.

**A hole found while doing this and closed:** the generic "facts about a
credential" suffixes had included `_hash` and `_digest`, which made
`dashboard_password_hash` read as safe. A hash of a secret is a verifier and is
offline-crackable, so both suffixes are gone and such names redact.

Verification: the owner's six named cases pass exactly
(`api_token`/`bot_token`/`refresh_token` → `<redacted>`; `max_tokens=32768`,
`prompt_tokens=123`, `completion_tokens=45` → visible), plus 17 metrics asserted
visible in one line with no `<redacted>` anywhere in it, 18 credential names
asserted redacted, the `tokens` type discrimination in both directions, the
password-hash case, and every real field name this codebase logs classified.
`pkg/logger` 177 assertions pass.

#### Follow-up 2 — a configuration block was logged as a runtime failure

The device log showed `PC-E-AI-004` — "every configured AI model is disabled" —
arriving as `ERR agent > LLM call failed`, `severity=error`. Telegram and the
gateway were healthy; the product was waiting on the owner.

`ErrorPayload` gained a `Classification`, and
`runtimeSeverityForAgentEvent` returns **warning** severity for
`configuration_blocked`. The log line is now
`WARN agent > Turn blocked by configuration` carrying `reason` and `code`, and the
provider-failover-exhausted event is **not** emitted — nothing was attempted, so
there is no exhausted chain to report.

The **event kind is deliberately unchanged**: it is still what ended the turn, and
every consumer that routes on kind keeps working. The zero-value classification
means "ordinary failure", so every existing caller is unaffected.

**The user-facing reply is byte-identical**, which the owner required, and a test
asserts the full text and code.

Verification: 6 cases in `pkg/agent/configuration_block_severity_test.go` —
warning for a configuration block, error for an unclassified payload, the zero
value meaning ordinary failure, the other error kinds unaffected, the user-facing
reply unchanged, and the user-facing identity surviving being returned and wrapped.

### PC-DEF-052 — Managed Telegram onboarding exposed the hosting origin

- **Discovered:** owner requirement after the 2026-09-13 physical round.
- **Component:** `lib/src/telegram/telegram_onboarding_controller.dart`, new
  `lib/src/telegram/telegram_deep_link.dart`.
- **Symptom:** starting managed Telegram setup opened a browser that showed a
  `*.vercel.app` page for a moment before redirecting into Telegram. The
  intermediate page does nothing for the user and names infrastructure that is
  not theirs to think about.
- **Where it came from, established by reading every launch site.** The
  onboarding screen can open exactly two URLs: `openTelegram()` →
  `pairing.deepLink`, and `openBotChat()` → `https://t.me/{username}`. The
  second is Telegram by construction, and `qr_payload` is rendered, never
  navigated to. So `deep_link` is the only candidate, and the deployed service
  returns its own redirect endpoint there. **The service is in a separate
  repository** (`Lord1Egypt/PocketClaw-Telegram-Setup`), so the fix could not be
  "make the service return a `t.me` link".
- **Resolution:** the app stops opening whatever URL it is handed.
  `TelegramDeepLinkResolver` resolves the setup link to a Telegram destination
  **in the background** before anything is shown — a link that is already
  Telegram passes through with no request at all, and anything else is followed
  by reading `Location` without rendering the intermediate page. Only
  `https://t.me`, `telegram.me`, `telegram.dog` and `tg://` are accepted as
  destinations.
  A link that does not resolve to Telegram is **refused, not opened**: opening it
  is the defect. That surfaces as the new `telegramLinkUnavailable` kind, distinct
  from `telegramUnavailable` (Telegram missing) because the two need different
  actions from the user, and manual token entry stays available.
- **Security posture this also buys:** a URL from a network response was being
  handed straight to the OS. Resolution is now bounded to 5 hops, refuses any
  non-`https` hop including a downgrade to `http://t.me`, dereferences nothing
  that is not `https` to begin with, and attaches no `Authorization` or `Cookie`
  to a host it has not vetted. The setup link is public by construction, so there
  is nothing to authenticate with.
- **What did not change:** the owner contract (exactly one positive numeric owner
  in `AllowFrom`), token handling, reconnect and replace behaviour, and the
  automatic runtime apply from PC-DEF-030. The backend may stay on Vercel; the
  requirement was that it not be user-visible navigation.
- **Verification:** 13 cases in `test/unit/telegram_deep_link_test.dart`
  (Telegram-target classification including `https://t.me.evil.invalid` and
  `?next=` smuggling, pass-through with no request, hosting-redirect resolution,
  relative `Location`, refusal of a chain that never reaches Telegram, refusal of
  a plaintext downgrade, refusal of a non-https start, hop cap, no-redirect
  endpoint, no credential sent, transport failure, blank input) and 6 in
  `telegram_onboarding_controller_test.dart` under "PC-DEF-052 direct Telegram
  launch" (opens the resolved link and never the `vercel.app` one, refuses an
  unresolvable link, passes a Telegram link through, distinguishes Telegram
  missing, proves no bot or poll token appears in any opened URI, and re-resolves
  on reconnect).
- **Status:** FIXED IN SOURCE. **Physical confirmation required** — connect a bot
  on the device and watch for any intermediate page.
- **Sweep update, 2026-09-17 — the browser path was the remaining hole.** The
  Android native launch was guarded, but the audit found two render paths that
  still accepted the service's value verbatim: the Dashboard anchor/`copyLink`
  (`telegram-desktop-connect.tsx`) and the Android QR
  (`TelegramPairingQr(payload: pairing.qrPayload)`), and the Go handler forwarded
  `deep_link`/`qr_payload` unvalidated despite comments claiming otherwise. Core
  now returns only a Telegram destination
  (`telegramDestinationOrEmpty`, mirroring the Dart resolver); the desktop
  component refuses to render a non-Telegram link even if Core had returned one;
  and the QR is omitted when its payload is not a Telegram destination. Tests:
  `TestCreatePairingDropsANonTelegramDeepLink`,
  `TestTelegramDestinationOrEmptyAdmitsOnlyTelegram` (Go); the
  `drops a non-Telegram deep link` and `classifies only real Telegram
  destinations` Vitest cases; and the `drops a QR whose payload is not a
  Telegram link` widget case. The rejected-pairing behaviour is unchanged: the
  manual form remains.
- **Note for the service repository:** returning a `t.me` link directly as
  `deep_link` would make the background resolution a no-op and remove the round
  trip. The app is correct either way now, so this is an optimisation, not a
  prerequisite.

### PC-DEF-053 — A first-run Telegram message got an internal error, not advice

- **Discovered:** owner requirement after the 2026-09-13 physical round.
- **Component:** `pkg/gateway/gateway.go` (`startupBlockedProvider`), new
  `pkg/agent/user_error.go` and `pkg/agent/ai_readiness.go`.
- **Root cause — PROVEN by reading the path end to end.** With no model
  configured, the gateway starts in limited mode and installs
  `startupBlockedProvider`, whose `Chat` returned
  `fmt.Errorf("no default model configured; gateway started in limited mode")`.
  That error is not an `*common.HTTPError`, so `providerErrorDetail` declines it;
  it is not classifiable, so `formatProviderFailure` declines it; and it lands in
  `formatProcessingError`'s last branch as **"Error processing message: no
  default model configured; gateway started in limited mode"**. Implementation
  jargon, no instruction, and nothing separating "Telegram works" from "no AI
  configured" — so a silent-looking bot sends the owner to re-pair a bot that was
  never at fault.
- **Resolution:** a `UserFacingError` carrying a stable code and a safe,
  actionable sentence, checked first by `formatProcessingError`. The blocked
  provider carries one instead of a bare string, and
  `agent.AIConfigurationProblem` picks which: nothing added (`PC-E-AI-001`),
  everything disabled (`PC-E-AI-004`), nothing selected (`PC-E-AI-002`), or a
  selection whose `model_list` entry is gone (`PC-E-AI-003`, which names the
  model). The first-run wording states that PocketClaw is connected before saying
  what is missing.
- **A wrong first attempt, recorded because it is the instructive part.** The
  check was first written as a precondition at the top of `processMessage`. Two
  existing tests failed, and they were right to: `NewAgentLoop` takes an
  **injected** provider, so an empty `model_list` does not imply there is nothing
  to send a request to, and a config-only precondition there refuses turns for a
  loop that has a perfectly good provider. The check belongs where the gateway
  already decides it cannot build one.
- **Scope boundary, deliberate.** `AIConfigurationProblem` does **not** judge
  whether a credential is usable. That needs the OAuth credential store and the
  local-endpoint probe that `web/backend/api`'s `hasModelConfiguration` owns, and
  a second copy of it is exactly the drift that had `pkg/modelaccess` reverted —
  an ambient-credential provider (a local Ollama, an OAuth provider) legitimately
  has no `api_key`. A missing or rejected credential is already reported
  accurately at request time by the provider's own 401 through
  `AuthErrorMissingAPIKey`.
- **The other categories the owner listed were already covered** by
  `pkg/agent/error_format.go` and `provider_detail.go` (PC-DEF-032): invalid /
  missing / expired credentials, rate limit, hard quota, billing, provider 5xx,
  network, timeout, request rejection, context overflow, overloaded, and the one
  sanitised sentence of the provider's own message. The configuration category
  was the gap.
- **Web UI: already localised, verified rather than assumed.**
  `chat-empty-state.tsx` already renders `chat.empty.noConfiguredModel` /
  `noSelectedModel` with a "Go to Models" action, and all five keys are present in
  all 14 locale bundles. No new web strings were needed for this category.
- **Telegram replies are English, because Core has no locale to localise
  against.** There is no `locale`/`language` field anywhere in `pkg/config`, and
  no i18n layer in Core. The stable codes exist so a future layer can localise
  them without touching Core. Recorded as an open item rather than guessed at.
- **Verification:** 14 cases in `pkg/agent/ai_readiness_test.go` (each
  configuration state, the message stating "connected" and telling the user what
  to do, no message blaming the channel, codes stable and distinct, no secret /
  path / stack-trace shapes in any message, the wrapped cause kept out of the user
  message, and a model with no API key deliberately **not** blocked) and 5 in
  `pkg/gateway/startup_blocked_provider_test.go` (limited mode produces a
  user-facing problem, the Chat failure is actionable and no longer says "limited
  mode" or "Error processing message", nothing-selected is distinguished from
  nothing-configured, a reasonless blocked provider still fails closed, and
  limited mode stays opt-in).
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS**, Samsung, 2026-09-14. The
  device replied over Telegram with:

  > PocketClaw is connected, but every configured AI model is disabled.
  > Open PocketClaw, enable a model, then send this again.
  > (PC-E-AI-004)

  The runtime log confirms `PC-E-AI-004` was classified and delivered. That is the
  disabled-model arm; the never-configured arm (`PC-E-AI-001`) and the
  stale-selection arm (`PC-E-AI-003`) remain source-verified only.

### PC-DEF-054 — Config-change signature carried credentials in plaintext

- **Discovered:** disclosed by this session while fixing PC-DEF-050, then
  investigated at the owner's instruction.
- **Component:** `web/backend/api/gateway.go`, new
  `web/backend/api/signature_digest.go`.
- **Scope — wider than first disclosed.** The original note named the `webcfg:`
  component. The mechanism is `canonicalizeSignatureValue`, which resolves
  `SecureString`/`SecureStrings` to plaintext, and it is used for channel settings
  too. Proven by test: a Brave web-search key, a proxy URL password and a Telegram
  bot token were all present verbatim in the signature string. By inspection the
  same applies to every channel credential (Slack bot/app tokens, Matrix access
  token and crypto passphrase, Feishu app secret, DingTalk client secret, LINE
  channel secret, OneBot/WeCom/WeiXin/QQ secrets) and every web-search key, plus
  the `normalizeRawJSON` fallback which dumps an undecodable settings subtree
  verbatim.
- **Exposure assessment — answered per the owner's list.**
  - *Logged?* **No.** No logger call takes a signature.
  - *Status/debug output?* **No.** Only the derived boolean
    `gateway_restart_required` reaches a client.
  - *Persisted?* **No.** It lives in `gateway.bootConfigSignature`, package
    memory, and is never written.
  - *In error text?* **No.** No error or panic embeds it.
  - *Crash reporting?* **No crash reporter exists** — Firebase/Crashlytics were
    removed under PC-DEF-R005.
  So this was **not a disclosure**. It was unnecessary plaintext secret material
  retained in long-lived package state, one careless
  `logger.Debugf("signature=%s")` away from becoming one.
- **Resolution — not deferred.** `computeChannelSignatures` and the `webcfg:`
  component now embed a SHA-256 digest of their payload instead of the payload.
  Digesting the whole subtree rather than classifying fields is deliberate: a
  field-by-field secret list would need updating every time a channel gains a
  credential, and missing one is silent. The model-credential digests added for
  PC-DEF-050 now share the same helper, which length-prefixes each part so a
  shifted boundary cannot collide. Plaintext still exists transiently inside the
  call — detecting that a secret changed requires reading it — but nothing
  retained carries it, which is the owner's stated invariant.
- **Verification:** `signature_secrets_test.go` — one case asserting that none of
  11 distinctive markers across web-search keys, a proxy password, four channel
  credentials, a raw-fallback secret, a model key and a model header appears in
  the signature; 9 sub-cases proving the signature still moves when each of those
  changes, so no sensitivity was traded away; stability across repeated
  computation; and digest unambiguity across part boundaries. Every pre-existing
  signature test still passes, so no restart decision changed.
- **Gotcha found while testing, worth keeping.** `Channel.GetDecoded` decodes
  lazily and **caches** into `extend`, so mutating raw `Settings` after a decode
  reads back the stale typed value. Production is unaffected —
  `handleGatewayStatus` calls `config.LoadConfig` on every poll — but a test must
  compare two freshly built configs rather than mutate one in place.
- **Status:** **RESOLVED.**

### PC-DEF-055 — Set Default was the last tooltip-only control on the model card

- **Discovered:** disclosed by this session under PC-DEF-047, audited at the
  owner's instruction as a low-severity mobile UX item.
- **Component:** `web/frontend/src/components/models/model-card.tsx`.
- **Audit result against the three things the owner asked to verify:**
  - *Accessible semantic label* — **PASS.** `aria-label` and `title` were already
    present on the control and on its disabled wrapper.
  - *Default state visually obvious* — **PASS.** A model that is default carries a
    visible "Default" badge, a start-edge accent (`border-s-pc-claw`) and a filled
    star. Not a glyph alone.
  - *Discoverability on touch* — **FAIL, on two counts.** The control was
    `size="icon-sm"`, smaller than the 40px Edit and Delete use on the same card,
    and the reason a disabled star was disabled lived only in a Radix tooltip,
    which never opens on a touch screen — the same PC-DEF-047 pattern.
- **Resolution, kept to the two failures.** A 40px touch target, and the disabled
  reason printed under the row exactly as Delete's already is. The tooltip is
  gone, and with it the wrapper `<span>` that existed only to carry `tabIndex`,
  `role` and `aria-disabled` so Radix could trigger on a disabled button. No
  redesign: the badge, the accent edge and the icon are unchanged.
- **Verification:** 6 cases in `model-card.test.tsx` under "ModelCard
  set-default affordance".
- **Sweep update, 2026-09-17 — a real data-loss bug beside the UX one.** The add
  and edit request shapes carry no `enabled` field, and both handlers wrote Go's
  zero value back: a model added through the API started disabled, and editing
  one disabled it. An omitted field now means enabled on add and preserves the
  stored value on edit (`models.go`). Added handler-level tests for the default
  contract the earlier round left unproven: `TestHandleSetDefaultModelPersistsAnEnabledModel`,
  `TestHandleSetDefaultModelSwitchesBetweenModels`,
  `TestHandleAddModelEnablesAnOmittedEnabledField`,
  `TestHandleUpdateModelPreservesEnabledWhenOmitted`, and
  `TestUpdatingTheDefaultModelsKeyKeepsItDefault`. **Deliberate limitation,
  recorded rather than hidden:** the product still exposes no enable/disable
  control, so the default-selectability gate remains `available` +
  `default_model_allowed` + non-virtual; `validateDefaultModelSelection` was not
  widened to require `Enabled`, because `Enabled` is not a user-facing control in
  this build and doing so would reject ordinary models. The only removal path is
  delete, which clears the default to the empty string (documented canonical
  fallback, `model_references.go`), already tested.
- **Status:** FIXED IN SOURCE. **Physical confirmation required.**

### PC-DEF-049 — A configured provider had no management path at all

- **Owner's number:** PC-DEF-047.
- **Discovered:** Samsung physical testing, 2026-09-13, owner-reported.
- **Component:** `web/frontend/src/components/models/provider-section.tsx`,
  `provider-picker.tsx`; new `web/backend/api/providers.go`.
- **Problem:** the console could add a provider and add or edit a model, but the
  provider object itself had no lifecycle. `ProviderSection` was a divider, an
  icon and a label wrapped in a collapse toggle — no action on it of any kind.
  The picker marked an existing provider "Already configured" with a checkmark
  and, when tapped, went to Add Model. So a provider could be created and seen
  but never viewed, edited, re-keyed, or removed, and the owner was left editing
  every model to change one thing that belongs to the provider.
- **Root cause of the shape, established by audit and not assumed:** there is no
  provider record in the configuration schema. `config.Config` holds only
  `model_list`, and each entry carries its own `provider` label, `api_base`,
  `api_keys`, `proxy` and `custom_headers`. A "provider" is a derived grouping
  over that string. There was nothing for a management screen to be a screen
  *of*. The full audit is `docs/PROVIDER_ARCHITECTURE.md` section 13.
- **Resolution:** provider management as an explicit view over `model_list`,
  introducing no provider object — a derived view cannot disagree with the models
  it is derived from, a stored one can.
  - `GET /api/providers`, `GET /api/providers/{provider}`,
    `PUT /api/providers/{provider}`, `DELETE /api/providers/{provider}`.
  - Provider-scoped state is reported as what the models *agree on*.
    Disagreement returns `credential_state: "mixed"` / `api_base_mixed: true`
    rather than picking one of several keys — presenting one as "the provider
    key" is exactly what would let a rotation update one model and leave its
    siblings on an old credential.
  - A labelled **Manage** control on the provider heading, `min-h-10`, separate
    from the collapse toggle so it stays reachable while collapsed. Not an icon
    with a tooltip: a touch screen has no hover, which is what made the model
    controls unfindable in PC-DEF-047.
  - A Manage Provider sheet carrying provider identity, credential state,
    Replace API Key, the base URL where the models agree on one, the provider's
    models by name, and Delete Provider. The primary action is in the header as
    well as the footer, for the PC-DEF-045 reason: the soft keyboard covers the
    footer the moment the credential field is focused.
  - Delete reports dependent models **by name**, calls out the default chat
    model, removes the models, and purges all seven reference sites. Recreating
    the provider afterwards carries nothing over.
- **Also fixed while in the same contract:** the single-model delete path covered
  only three of the seven reference sites (`PC-DEF-043` widened it to two
  fallback chains). It now uses the same complete purge, so `image_model`,
  `routing.light_model` and per-agent and per-subagent model references can no
  longer keep a deleted name.
- **Verification:** 17 backend cases in `web/backend/api/providers_test.go`
  (view, alias/case lookup, shared vs mixed credentials, rotation reaching every
  sibling, whole-key-list replacement, unsent fields untouched, blank key
  refused, delete removing models and clearing default/fallback/light-model and
  agent references, delete reporting what it removed, clean recreate);
  `TestDeleteModelClearsTheLightModelAndAgentReferences` and
  `TestPurgeModelReferencesListSemantics`; 4 cases in
  `delete-provider-dialog.test.tsx`, 9 in `manage-provider-sheet.test.tsx`, 4 in
  `provider-section.test.tsx`. Full Go backend suite and the 469-test frontend
  suite green.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS in the tested flow**, Samsung,
  2026-09-14. The owner can open Manage Provider and the provider-level
  management UI is present, including provider deletion and credential
  management. Credential rotation reaching the runtime (`PC-DEF-050`) is a
  separate check and is still outstanding.

### PC-DEF-050 — A replaced API key was saved but never reached the running gateway

- **Owner's number:** PC-DEF-048.
- **Discovered:** Samsung physical testing, 2026-09-13, owner-reported: "the
  owner cannot confidently change the API key and have the new key saved and
  used."
- **Component:** `web/backend/api/gateway.go`, `computeConfigSignature`.
- **Root cause — PROVEN, not inferred.** `gateway_restart_required` is the
  difference between a saved configuration and a live one, and it is a comparison
  of the signature the gateway booted with against the signature of the config on
  disk. That signature covered the default model name, the *streaming flag* of
  referenced entries, the tool set, the web-search config and the channels. It
  covered **no credential, no endpoint and no header** of any `model_list` entry.

  So the whole flow succeeded and changed nothing that answers a request:
  `PUT /api/models/{index}` persisted the new key correctly; the console then
  asked whether a restart was required and was told **false**; it reported the
  save as applied; and the running gateway went on using the previous credential
  until some unrelated change or a manual restart happened to reload it.

  Proven by computing the signature either side of a rotation before the fix: the
  string was byte-identical after replacing the key, after changing `api_base`,
  and after changing `custom_headers`. The owner's report is accurate, and the
  editable field was never the problem — the apply step was.
- **Resolution:** `computeModelCredentialSignatures`
  (`web/backend/api/model_credential_signature.go`) adds one entry per
  `model_list` index covering provider, model, `api_base`, `proxy`,
  `auth_method`, `enabled`, a digest of the whole key list, and a digest of the
  custom headers. Every entry is covered, not only those reachable from the
  default and fallback chains: a key belonging to any configured model is
  material the booted process holds, and narrowing it would restore the same
  silent staleness for the rest. Secrets are reduced to SHA-256 digests — the
  value is compared in process and never serialised, but a comparison does not
  need the plaintext, and the whole key list is digested because a multi-key
  entry fails over between them.
- **Verification:** 8 cases in `model_credential_signature_test.go` — rotation,
  `api_base`, header add and header change, a non-default sibling's rotation,
  disabling a model, stability across repeated computation of one config, and
  that no raw key or header credential appears in the signature. Existing gateway
  signature tests still pass, so no restart-decision behaviour regressed.
- **Note:** the provider-level rotation added for PC-DEF-049 goes through the same
  apply path, so one rotation now both persists and becomes live for every model
  of the provider.
- **Status:** FIXED IN SOURCE. **Physical confirmation required** — real
  inference with a replaced key on the device.
- **Sweep update, 2026-09-17.** Reconfirmed that the credential material is
  covered by `computeModelCredentialSignatures` and that the apply path goes
  through `RestartGatewayForConfigChange`, so a rotation that changes the
  signature reaches a restarted runtime. The end-to-end physical proof (next
  request uses K2, K1 never reused) is still a device check. Added
  `TestUpdatingTheDefaultModelsKeyKeepsItDefault`, which pins that a rotation of
  the model that is currently default does not move the selection. No production
  change was needed for this defect beyond the shared `enabled` preservation
  recorded under PC-DEF-055.

### PC-DEF-051 — The Telegram command menu is empty; registration evidence was never trustworthy

- **Owner's number:** PC-DEF-049.
- **Discovered:** Samsung physical testing, 2026-09-13, owner-reported. Message
  round-trip **PASS**; the command menu, previously about 14 commands, shows
  nothing, `/start` included.
- **Component:** `pkg/channels/telegram/command_registration.go`,
  `pkg/commands/builtin.go`.
- **Classification, kept separate as the owner instructed:**
  - TELEGRAM MESSAGE ROUND-TRIP = **PASS**.
  - TELEGRAM BOT COMMAND REGISTRATION = **BROKEN on the device. Root cause
    UNKNOWN — device evidence required.**
- **What the source audit rules out, and by what.** The registry and the wiring
  are intact, so this is not a narrowed command set and not removed
  registration:
  - `commands.BuiltinDefinitions()` returns **14** definitions, every one with a
    non-empty name and description, `/start` first. None is filtered out on the
    way to Telegram. Pinned by `pkg/commands/telegram_command_menu_test.go`.
  - `TelegramChannel.Start` calls `startCommandRegistration(c.ctx,
    commands.BuiltinDefinitions())` after the bot connects, and the goroutine
    retries with backoff until it succeeds or the channel shuts down.
  - The `/model` picker revert (`5c92d9b`) and its replacement (`bc6ad04`) did not
    touch the registration wiring. `/model` is still in the set.

  None of that makes the bot's menu correct — it means the explanation is not in
  the command list or the call site, and must not be guessed at.
- **What *is* provably wrong, and is fixed: the success log was never evidence.**
  The completion log printed `"count": len(defs)` — the number of definitions
  *received*, never the number of commands Telegram accepted. It printed
  `count=14` whether fourteen commands were published, none were because every
  definition had been filtered out, or the call was skipped because Telegram
  already agreed. The historical `Telegram commands registered count=14` the
  owner cites therefore never proved the menu was populated, and its absence now
  is the only real signal in either direction.

  `RegisterCommands` now logs what it actually did — `sent` versus `defined`, or
  "already current" with the count Telegram reported — plus the bot username, so
  a replaced or reconnected managed bot is distinguishable from the one before
  it. Any definition that cannot be published is named in a warning instead of
  dropped in silence.
- **Verification:** 6 cases in `command_registration_test.go`, including that
  registration receives the complete set with `/start` present, that it is
  attempted again on a reconnected channel, and that an empty definition list
  touches the menu not at all; 6 in `telegram_command_menu_test.go` pinning the
  expected 14-command set, `/start`'s presence, publishability, and Telegram's
  name and description limits — a single over-long description or invalid name
  fails `setMyCommands` for *every* command, which is one mechanism that would
  produce exactly this symptom.
- **Owner security contract: unchanged.** `NewTelegramChannel` still requires
  exactly one positive numeric owner in `AllowFrom` and refuses empty, wildcard,
  username and multiple entries — `TestNewTelegramChannelRejectsOpenAuthorization`.
  Command registration does not read or write it.
- **Next step is diagnostic, not a fix.** On the device, after connecting or
  reconnecting the managed bot, the Core log must be read for
  `Telegram command menu set` / `already current` with its `sent` count and bot
  username, or for `Telegram command registration failed; will retry` with the
  API error. That single line separates "never called", "called and rejected by
  Telegram", and "called, accepted, and the menu is a client-side view problem".
  **No further change may be made to registration by guess.**
- **Status:** **RESOLVED — command registration is PHYSICALLY VERIFIED WORKING**,
  Samsung, 2026-09-14. The device log shows `getMyCommands → ok=true`,
  `Telegram command menu already current`, `registered=14`, and the Telegram UI
  exposes the menu including `/start`, `/help`, `/stop`, `/show`, `/list`, `/use`,
  `/btw`, `/switch`, `/model` and `/check`.

  So this was never a missing-registration or missing-definition defect. The
  owner's original observation — an empty menu — is explained by `PC-DEF-056`:
  onboarding had not finished applying the configuration, so the Telegram channel
  had not started, so nothing had registered anything yet. The fix that mattered
  was the readiness race, not the command set.

  **Do not re-debug Telegram API connectivity or command definitions.** The
  observability improvement made here stands on its own merit: the success log now
  reports what was actually published rather than the number of definitions
  received, which is what made the device evidence above readable in the first
  place.

### PC-DEF-045 — The Save/Update action was not reachable in the real flow

- **Discovered:** Samsung physical testing, 2026-09-13, after PC-DEF-033 was
  confirmed fixed on the device.
- **Component:** the Add and Edit model sheets.
- **Description:** the owner reports, from the device, that the Edit Model
  screen still has no clear Save/Update control available in the actual flow.
- **Assessment:** the control exists, is enabled when the form is dirty, and is
  now labelled "Update Model" — so its absence is a matter of where it is, not
  whether it is there. The sheet is a fixed-height panel whose footer sits at the
  bottom edge; the soft keyboard opens the moment the API key field is focused,
  which is precisely when there is something to save. A footer below the
  keyboard is a control that does not exist as far as the user is concerned.
  This is INFERRED: it fits the report and the layout, and no device was
  available to this session to confirm it.
- **Resolution deliberately chosen to be robust to the diagnosis being wrong:**
  the primary action is now also in the sheet header, which no keyboard can
  cover and which is visible without scrolling. If the cause was the keyboard,
  this fixes it; if the cause was that the owner never reached the bottom of a
  long form, this fixes that too.
- **Status:** FIXED IN SOURCE. **Physical confirmation required.**

### PC-DEF-047 — No obvious Delete/Remove action for a model

- **Discovered:** Samsung physical testing, 2026-09-13.
- **Component:** `web/frontend/src/components/models/model-card.tsx`.
- **Root cause:** Edit and Delete were 32px (`size-8`) icon-only ghost buttons,
  packed with a 2px gap into the card's top-right corner beside a truncating
  model name, and explained only by a Radix tooltip. A tooltip opens on hover;
  a touch screen has no hover, so on the device the two controls were unlabelled
  grey glyphs and the delete one was `text-pc-muted` until a hover that never
  came. Reporting it as missing was accurate to the experience.
- **Resolution:** both are now labelled text controls on their own row, each
  `min-h-10` and half the card's width, and the reason a disabled Delete is
  disabled is printed on the card instead of hidden in a tooltip.
- **Note:** the set-default star is still a 32px icon with a tooltip. It is
  left alone deliberately — adding a third control to the row would recreate the
  crowding this fixes, and its meaning is carried by the visible "Default"
  badge — but it is the same affordance pattern and is recorded here rather than
  left unmentioned.
- **Status:** FIXED IN SOURCE. **Physical confirmation required.**
- **Note on the identifier:** this was allocated PC-DEF-046 earlier in the same
  session and renumbered when the owner allocated 046 to the navigation stall
  below. Commit `806c083`'s message predates the renumbering and still says 046.

### PC-DEF-046 — UI navigation stalls during rapid screen switching

- **Discovered:** Samsung physical testing, 2026-09-13, owner-reported.
- **Component:** `lib/main.dart` navigation shell, `lib/src/ui/webview_page.dart`,
  `lib/src/ui/webview/webview_android.dart`.
- **Status:** **UNKNOWN — INVESTIGATION REQUIRED. Android-client-specific or
  device-specific.** Owner classification, 2026-09-13. No fix has been applied
  and none may be applied by guess.
- **Symptom:** rapidly switching between Chat, Models, Credentials and Settings
  sometimes stops responding; the UI stays on Chat although another destination
  was tapped. No crash.

- **What is ruled out, and by what.**
  - *Core/Gateway.* The owner established that the supplied Core log does not
    explain it: the failing OpenCode turn completed in ~621 ms. The OpenCode 400
    is explicitly **not** the cause and must not be offered as one.
  - *The console web app itself.* The PocketClaw dashboard in a desktop browser
    is fast and shows no stall (owner, 2026-09-13). The same HTML, JavaScript and
    Core serve both, so whatever is slow is not the page.

  Neither of those makes the remaining hypothesis true. They narrow where to
  look: the Android client, its WebView integration, or the device.

- **The discriminator that decides this, owner-specified and not yet run.**

  | Test | Reading |
  | --- | --- |
  | Same physical phone, console in the phone's **browser**, rapid switching | responsive → the defect is in the PocketClaw Android application |
  | Same physical phone, console in the **app** | both stall similarly → investigate device CPU/RAM pressure before changing any application architecture |

  Until that runs, everything below is a hypothesis with a known code basis,
  not a cause.

- **Audit findings: code properties that are proven, whose causal role is not.**

  These are facts about the source, verified by reading it. They would produce
  work on exactly the path the owner is exercising, and they need no network —
  which is consistent with the Core log showing nothing. That consistency is not
  proof, and none of it is offered as the answer.
  Chat, Models and Credentials are not separate Flutter routes. They are one
  IndexedStack child, index 1, the WebView, with a different `_webPath`.
  Switching between them is therefore not navigation; it is a change of one
  string. Two keys turn that string into a full teardown.

  1. **`WebViewPage(key: ValueKey<String>(_webPath))`** — `lib/main.dart:330`.
     A changed key unmounts the element and mounts a new one, so every switch
     between Chat, Models and Credentials **destroys and recreates the
     WebView**. `WebViewAndroid` has no `didUpdateWidget`: it loads its URL once
     in `initState` (line 171), so remounting is in fact the only way a URL
     change is honoured today. Each switch is a fresh `WebViewController` and a
     cold boot of the console single-page app, whose main bundle is ~1.36 MB of
     JavaScript, followed by host-bridge re-injection. The file says as much:
     "the console is a single-page app served fresh on each navigation into the
     tab".
  2. **`IndexedStack(key: ValueKey<int>(_selectedIndex))`** — `lib/main.dart:320`.
     The same rule applies one level up, so every tab change **destroys and
     recreates all four pages**: DashboardPage, the WebView, LogPage and the
     2095-line ConfigPage. Three lines below it the code states the opposite
     intent — "The pages live in an IndexedStack and are never disposed on a tab
     change, so the flag — not the widget lifecycle — is what stops the extra
     work." The key contradicts the design it sits inside.

  Under rapid tapping this composes exactly into the reported symptom: a tap
  arriving while the previous WebView is still initialising has nothing settled
  to act on and is effectively lost, and what stays on screen is whatever last
  finished loading — Chat.

- **Secondary finding, same audit.** `getGitHubStatus` runs
  `GitHubCredentialStore.status()` **on the Android platform main thread**:
  a file read plus an Android Keystore key load and an AES-GCM decrypt, all
  under a `@Synchronized` class lock (`PocketClawMethodChannel.kt:351`,
  `GitHubCredentialStore.kt:153`). Platform-channel handlers run on the main
  thread, so this blocks the UI for as long as the Keystore takes. It is
  reachable from the Settings screen, one of the four the owner was switching
  between. Every other I/O-bearing handler in that file (`configureTelegram`,
  `checkHealth`, `connectGitHub`) correctly uses a `Thread`; this one and
  `saveConfig` do not.

- **Context, not yet implicated.** Two `Timer.periodic` loops run at 3-second
  intervals — `_nativePollingTimer` (status + health over the platform channel)
  and `_lanAddressPollingTimer` — each ending in `notifyListeners()`. They are
  a steady background cost rather than a stall, and are recorded so the next
  measurement can rule them in or out rather than rediscover them.

- **Why no patch.** Two reasons, and the first is sufficient on its own.

  The cause is not established. Changing the navigation architecture to fix a
  hypothesis would be exactly the guess the owner ruled out, and if the stall is
  device pressure it would be a rewrite that fixes nothing while removing a
  shipped behaviour.

  Second, even once confirmed, both primary findings are one-line keys whose
  removal changes product behaviour, and the choice is the owner's:
  - Removing the IndexedStack key keeps page state and ends the remounting, and
    also **removes the cross-tab `PageTransitionSwitcher` animation**, which
    only fires because the child's identity changes. Keep the animation or keep
    the state; the current code pays for the animation with a full remount.
  - Removing the WebView key requires `WebViewAndroid` to gain a
    `didUpdateWidget` that navigates the existing controller instead, and the
    console is an SPA, so the real question is whether to `loadRequest` the new
    path or push it through the page's own router. That is a design decision,
    not a key deletion.

- **Evidence that would confirm or refute the audit, in order.**
  1. The browser-versus-app discriminator above. It decides whether this is an
     application defect at all, and costs one minute on the device.
  2. If it is the application: `adb logcat` during rapid switching, looking for
     repeated WebView or renderer creation, and a Flutter timeline showing
     element rebuilds. The audit predicts both.
  3. If neither appears, the findings above are not the cause, and the 3-second
     polling loops and the main-thread Keystore read are the next candidates.
  4. If both surfaces stall on the same phone: device CPU and memory pressure
     first, and no application change until that is excluded.

- **Acceptance contract (owner's, recorded verbatim in intent):** rapid repeated
  switching between Chat, Models, Credentials and Settings stays responsive — no
  lost taps, no navigation deadlock, no multi-second freeze, and no waiting on
  an unrelated network or provider request. Not closed until reproduced on the
  Samsung and physically verified after a fix.

### PC-DEF-048 — The hardened build silently skipped Flutter AOT

- **Discovered:** building the verification APK for this milestone, 2026-09-13.
- **Component:** `tool/build_hardened_android.py`.
- **Description:** the build failed with
  `Hardened Android build FAILED: split debug info was not produced at ...`.
  Nothing in that message says why, and the APK it had just assembled was
  otherwise complete.
- **Root cause:** this milestone's second round changed Go, Kotlin and
  web-console source and no Dart source. `reset_generated_build_outputs` clears
  `.dart_tool/flutter_build`, which is Flutter's own incremental cache and the
  fix for the sibling defect PC-DEF-R009 — but Gradle decides **separately**
  whether to run the Flutter AOT task at all, and its up-to-date check watches
  the Dart sources, not the private symbol file, which lives outside the project
  tree by design. With no Dart change the task was skipped, no symbols were
  emitted, and the post-build assertion failed. `454 actionable tasks: 29
  executed` against 38 on the previous, Dart-touching build.
- **Resolution:** the same reset now also drops Gradle's own
  `build/app/intermediates/flutter`, removing the answer its up-to-date check
  was relying on, with the same symlink refusal the Dart-side clear already had.
- **Workaround used for this milestone's artifact:** `--clean`.
- **Status:** FIXED IN SOURCE; the fix has not yet been exercised by a build
  that changes no Dart source, which is the only condition that reproduces it.

### PC-DEF-033 — An empty amber rectangle on every configuration screen

- **Discovered:** Samsung physical testing, 2026-09-13.
- **Component:** `web/frontend/src/components/config-change-notice.tsx`.
- **Root cause:** the notice rendered `text-pc-warning` on `bg-pc-warning` —
  the same design token for the surface and the label, so the text and the icon
  were painted in the colour of the box behind them. `--pc-warning` is fully
  opaque in both themes; `--pc-warning-soft` is the 14%-alpha variant the rest
  of the codebase uses for exactly this. The notice appears in the footer of the
  Add Model and Edit Model sheets, the settings and channel-config pages and the
  web-search tab, which is why it was reproduced on several screens; on a phone
  the footer is a single column, so it read as a large empty block near the
  bottom. The `kind` ternary had two identical branches, which is how it
  survived review.
- **Note:** the notice is what appears the moment a form becomes dirty. On the
  Edit Model sheet that is the moment a new API key is typed, so the one
  affordance telling the owner there was something to save was invisible.
- **Resolution:** soft background, solid foreground, matching the established
  pattern. Three further `bg-pc-warning/70`-with-`text-pc-warning` collisions in
  the Agent hub and tools screens were fixed with it.
- **Verification:** `config-change-notice.test.tsx` asserts the token identity
  rather than a rendered colour, so the regression cannot return under a
  different class name.
- **Status:** **RESOLVED — PHYSICALLY VERIFIED PASS**, Samsung, 2026-09-13. The
  warning renders visible text on the device.

### PC-DEF-041 — Model chips did not say where they came from

- **Discovered:** Samsung physical testing, 2026-09-13.
- **Component:** the Add and Edit model sheets.
- **Root cause:** three sources of model ids rendered as three consecutive,
  unlabelled rows of chips: the provider preset's curated `common_models` (a
  static list compiled into the build), a cached earlier fetch from
  `model_catalogs.json`, and this session's live Fetch Models result. Nothing
  distinguished a name this build happens to know from one the provider had just
  confirmed it serves.
- **Resolution:** one `ModelChipGroup` used by both sheets, with an explicit
  label and hint per origin — Suggestions, Previously fetched, Verified
  available — in all fourteen locales. A suggestion never renders with the
  confirmed-selection variant. Fetched ids are exact: nothing normalises or
  aliases them.
- **Status:** FIXED IN SOURCE. **Not physically verified.**

### PC-DEF-042 — A chat model selection appeared to be ignored

- **Discovered:** Samsung physical testing, 2026-09-13.
- **Component:** `web/frontend/src/hooks/use-chat-models.ts`.
- **Root cause:** the selector is a fully controlled `Select` whose value is
  server state, and that state was written only after `POST /api/models/default`,
  a second `GET /api/models` and a gateway restart had all resolved. Until then
  the trigger went on showing the previous model with nothing to indicate work in
  progress, so the selection read as having been ignored. Navigating to Settings
  and back remounts the hook, which re-reads the persisted default — which is
  exactly the workaround the owner found.
- **Resolution:** the picked model is held separately and shown immediately, the
  trigger is disabled while the save is in flight, the pending value yields to
  the persisted one on success, and a failed save reverts it and reports the
  error rather than pretending it succeeded.
- **Verification:** `hooks/use-chat-models.test.ts`.
- **Status:** FIXED IN SOURCE. **Not physically verified.**

### PC-DEF-043 — Removing a model left references behind and failed silently

- **Discovered:** provider-settings audit during this milestone, 2026-09-13.
- **Component:** `web/backend/api/models.go`,
  `web/frontend/src/components/models/delete-model-dialog.tsx`.
- **Description:** three defects in one removal path.
  1. `handleDeleteModel` cleared `agents.defaults.model_name` when the deleted
     entry was the default, but left the deleted name in
     `agents.defaults.model_fallbacks` and `image_model_fallbacks`. Those are
     lists of `model_list` names, so a deleted name stays as a candidate the
     router will try and cannot resolve.
  2. The dialog's confirm handler returned without doing anything when the model
     was the default: the user pressed Delete, the dialog closed, and the model
     was still there with no reason given.
  3. A failed delete was caught and discarded with a comment saying the user
     could retry — the user was told nothing, and a failed delete looked
     identical to a stale list.
  Removal was also never applied to the gateway, so a running gateway went on
  holding the deleted entry.
- **Resolution:** the deleted name is purged from both fallback chains; the
  default model is refused in the dialog with a reason and a disabled action;
  a failure is reported and leaves the dialog open; and the removal is applied
  through the same gateway path every other configuration change uses.
- **Verification:** `web/backend/api/model_delete_references_test.go` and
  `delete-model-dialog.test.tsx`.
- **Status:** FIXED IN SOURCE. **Not physically verified.**

### PC-DEF-044 — A replaced Telegram token skipped the owner contract

- **Discovered:** source audit of the historical
  "telegram requires exactly one paired numeric owner" failure, 2026-09-13.
- **Component:** `web/backend/api/telegram_owner_contract.go`.
- **Description:** `telegramSemanticKey` recorded `token_set=true|false` and a
  comment claiming that "a changed token still reads as an edit". It does not:
  replacing one token with another leaves the flag `true` both times, so the key
  is identical, `telegramSubtreeChanged` returns false and `PUT /api/config`
  skips the owner check entirely. A Replace Bot save could therefore persist a
  fresh token alongside a stale or malformed owner — which Core then refuses at
  startup with exactly the historical message.
- **Assessment of the historical failure:** **INFERRED**, not proven. This is a
  demonstrated path to that message, and it is the only writer-side gap found.
  No log or configuration from the original broken build survives to establish
  that this is the path it actually took.
- **Resolution:** the key carries a SHA-256 digest of the token instead of a
  presence flag. The digest is built, compared and discarded inside the call;
  it is never persisted, logged or returned past the comparison.
- **Verification:** four cases in `telegram_owner_contract_test.go`.
- **Status:** FIXED IN SOURCE.


### PC-DEF-023 — A third-party Google OAuth client secret is embedded in both Core binaries

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `core/src/pkg/auth/oauth.go`, `GoogleAntigravityOAuthConfig`.
- **Severity:** Third-party credential reuse and availability risk. Not a
  disclosure of any PocketClaw or user secret; not a release blocker.
- **Description:** the Google Cloud Code Assist ("Antigravity") OAuth
  configuration carries a hardcoded client ID **and client secret**, stored
  base64-encoded and decoded at runtime by a local `decodeBase64` helper. The
  encoded form is present in both `libpocketclaw.so` and `libpocketclaw-web.so`;
  the decoded form is not, so a plain string scan for the credential's prefix
  finds nothing. The source comment states these are "the same client
  credentials used by the OpenCode antigravity plugin" — that is, a credential
  registered to another project's Google Cloud account, not PocketClaw's.
  `web/backend/api/oauth.go:552` reaches it, so it is live product surface, not
  dead code.
- **Assessment:** for an installed application this class of secret is not
  confidential — RFC 8252 and Google's own desktop-client model assume it cannot
  be kept — so shipping it does not leak anything that was ever protected. The
  real exposures are different: PocketClaw depends on a credential a third party
  can revoke at any time, which would break the provider for every user; and the
  base64 wrapper means the credential is invisible to routine secret scanning,
  including this audit's own pattern pass. It was found by entropy review.
- **Owner decision, 2026-09-13: do not ship Google Antigravity in v0.2.0.**
  PocketClaw stable will not depend on another project's OAuth client. The
  provider may return later under a PocketClaw-owned integration; see
  `DECISIONS.md`.
- **Fix applied, 2026-09-13 — the provider is removed from the product.**
  Surface by surface rather than by deleting one function:
  `GoogleAntigravityOAuthConfig`, both embedded credentials and the orphaned
  `decodeBase64`; the `googleapis.com` token-URL inference that produced the
  `google-antigravity` provider name; the credential-store alias; the provider
  implementation and its test; the facade type aliases and fetch wrappers; the
  factory construction arm; the product-facing catalogue entry; the keyless
  `model_list` template; the legacy-import protocol mapping; the OAuth API's
  constant, order, methods, labels, config arm, project-ID fetch and default
  model; the implicit-OAuth handling in `models.go` and `model_status.go`; the
  CLI login arm, `authLoginGoogleAntigravity`, `authModelsCmd` and the
  `auth models` subcommand that existed only to list Antigravity models; and on
  the frontend the credential card, the union-type member, the hook status and
  label, and the locale key in all fourteen bundles.

  **Retained deliberately.** `OAuthProviderConfig.ClientSecret` stays: the
  generic token exchange supports confidential clients and that is shared
  infrastructure. `canonicalProvider` keeps its trim/lower-case normalisation,
  which every credential-store caller goes through — only the alias went.
  Gemini is untouched and is a different provider entirely, with its own
  catalogue entry, API-key auth and `generativelanguage.googleapis.com` base.
  OpenAI OAuth, the Anthropic token flow, PKCE, state, the callback and session
  handling are unchanged.

  **Backend behaviour:** `antigravity` and `google-antigravity` fall through to
  the existing unsupported-provider error rather than being special-cased, and
  the provider is absent from the catalogue — not a hidden callable provider
  behind a removed UI.
- **Verification:** `core/src/web/backend/api/no_antigravity_test.go` pins that
  five spellings are rejected as unsupported, that the OAuth surface is OpenAI
  and Anthropic only across order/methods/labels, that no catalogue entry or
  alias mentions it, that Gemini and the `google` → `gemini` alias survive, and
  that neither the encoded nor the decoded third-party credential exists in any
  Go, TS, TSX, JSON, Dart or Kotlin source file. `NormalizeProvider` is
  deliberately *not* the absence assertion: it is a string normaliser that
  echoes an unknown id back and says nothing about registration, which a first
  attempt at this test got wrong.

  Both staged Core binaries carry **zero** occurrences of `google-antigravity`,
  `antigravity`, `Antigravity`, `Google Code Assist`, `antigravity.google` and
  every encoded or decoded credential marker, while Gemini's endpoint and
  display name are still present. The first rebuild still showed three
  `antigravity` strings in the gateway binary: they came from the **embedded**
  agent skill document, which advertised the provider and the removed
  `auth models` command to the agent. That is a shipped product surface, so it
  was corrected and the pair rebuilt.
- **Core impact:** `core/src` changed, so the pair was rebuilt and re-staged
  under the two-commit rule from build-input commit
  `54ff2525fa555744d017aae56c9a26e2049812e1`. Source fingerprint moved from
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` to
  `bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9`.

      libpocketclaw.so       37,658,976  0a28bd5e1d6e33dc35b808039571b6683e4d47dec021941650f916641b859f6e
                                         build ID c657e80da54549a3bcc9a8bdba0a576d7b7273a0
      libpocketclaw-web.so   25,319,424  9ae1d2d9e7ac26d602db722649ebec4ac9cd982fa166c50ded303685d2abce50
                                         build ID 45355d87ba5ec042740675f82bc3d3940265318a
      BuildTime              2026-09-13T00:24:56+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise — `libpocketclaw.so.debug` 14,525,160 bytes
  `709cf86397dcb365b545eed00a84aede20a20ce77b1663562b1803cc4f77e546` and
  `libpocketclaw-web.so.debug` 9,687,312 bytes
  `a8c8f3d421fdce710a709cb330c79b63fe9aa68d7e0e5d55d58d0dd152006ebe`. Native
  contract 22 PASS / 0 FAIL. No Managed Runtime payload was rebuilt. The support
  manifest still names the PC-DEF-024 audit APK `f580cadc…`, which no longer
  contains this pair — the established pre-artifact state, rebound at the next
  artifact build.
- **Status:** RESOLVED, 2026-09-13.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:** still live
  product surface; the encoded form appears once in each Core binary and the
  decoded form zero times; the upstream-origin comment is intact. An
  installed-app OAuth client class, so the real exposure remains third-party
  revocation/dependency rather than secrecy, and no PocketClaw or user secret is
  disclosed. Classification unchanged; **not a release blocker.**
- **Status:** OPEN.

### PC-DEF-024 — A dead analytics deep link is exported in the release manifest

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `android/app/src/main/AndroidManifest.xml`; `MainActivity`.
- **Severity:** Unnecessary externally reachable entry point. Low.
- **Description:** `MainActivity` is exported and carries a second intent filter
  with `VIEW` + `DEFAULT` + `BROWSABLE` on scheme
  `${POCKETCLAW_UMENG_LINK_SCHEME}`. In the shipped configuration
  `POCKETCLAW_UMENG_APP_KEY` is empty, so `build.gradle.kts:42-46` resolves the
  scheme to the literal `um.placeholder`, and the merged release manifest
  confirms `android:scheme="um.placeholder"` reaches the artifact. Any web page
  can therefore launch PocketClaw with `um.placeholder://…`, and
  `MainActivity.logIncomingIntent` writes the full attacker-supplied URI to
  logcat at INFO. The Umeng SDK it exists for is not packaged
  (`POCKETCLAW_UMENG_PACKAGED=false`, and H1.5 removed the SDK entirely).
- **Assessment:** `MainActivity` is already launchable by any app through its
  LAUNCHER filter, so the added capability is web-originated launch plus
  attacker-controlled text in a log other apps cannot read on current Android.
  No injection surface: the URI is logged and otherwise unused.
- **Fix applied, 2026-09-13 — removed, not made conditional.** The repository
  had already decided this shape for the same integration: the manifest's
  advertising-permission comment records that "an analytics build gets whatever
  the analytics SDK's own AAR manifest declares" and that an app-level
  declaration it genuinely needs belongs to that build's own manifest. A filter
  that is only correct for a build PocketClaw does not ship should not sit in
  the manifest it does, and making it conditional would have added complexity to
  preserve dead code.

  Removed: the `VIEW` + `DEFAULT` + `BROWSABLE` intent filter from
  `AndroidManifest.xml`; `umengLinkScheme`, its `POCKETCLAW_UMENG_LINK_SCHEME`
  `buildConfigField` and its `manifestPlaceholders` entry from
  `build.gradle.kts`; and `MainActivity.logIncomingIntent` with both its call
  sites, plus the `TAG` constant and `android.util.Log` import that existed only
  for it. `onCreate` became an override that only called `super`, so it and the
  then-unused `Bundle` import went too. `setIntent(intent)` in `onNewIntent`
  **stays** — FlutterActivity and plugins read `getIntent()`, and removing the
  logging must not remove real intent handling.

  `POCKETCLAW_UMENG_APP_KEY`, `_CHANNEL` and `_PACKAGED` are **kept**: they are
  consumed by `AnalyticsReporter.kt` and the two `meta-data` entries, so they are
  live plumbing rather than residue, and they are not an exported surface.
- **Verification:** the packaged merged manifest of a fresh LOCAL TEST APK
  (`f580cadc…`) contains **zero** occurrences of `um.placeholder`, `BROWSABLE`,
  `android:scheme` and `action.VIEW`, while `category.LAUNCHER` and
  `.MainActivity` are still present and `debuggable`/`testOnly` remain absent.

  Seven tests in `test/unit/android_release_contract_test.dart` guard it: the
  source manifest declares no scheme at all; no `BROWSABLE` or `VIEW` survives;
  the `MAIN`/`LAUNCHER` contract and the exported launcher activity remain; the
  Gradle link-scheme plumbing is gone so no stale placeholder can survive merge
  processing; the logging branch is gone while `setIntent` remains; the default
  provider still packages no SDK; and the **merged** release manifest carries
  none of it when one has been built. Assertions strip XML comments first and
  check declarations, because the comment documenting the removal necessarily
  names what was removed — running the merged-manifest assertion for real caught
  exactly that, since Gradle carries comments through and only aapt2 strips them.

  Mutation-tested: reintroducing the filter fails the contract.
- **Core impact:** none. Android product source and tests only; the Core source
  fingerprint is unchanged at
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00`, Core was
  not rebuilt and the staged pair is untouched.
- **Status:** RESOLVED, 2026-09-13.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:**
  `android:scheme="um.placeholder"` is present in the fresh merged release
  manifest; `POCKETCLAW_ANALYTICS_PROVIDER` still defaults to `none` so the SDK
  is not packaged; `logIncomingIntent` still only writes the URI to logcat.
  `MainActivity` is already LAUNCHER-exported. Classification unchanged; **not a
  release blocker.**
- **Status:** OPEN.

### PC-DEF-022 — `/api/update` fetches and extracts an arbitrary URL with no provenance check

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `core/src/web/backend/api/update.go`; `core/src/pkg/updater`.
- **Severity:** Authenticated attack surface. Not a release blocker.
- **Description:** `POST /api/update` takes a caller-supplied `url`, hands it to
  `updater.UpdateSelfFromRelease`, which downloads the named asset, extracts the
  archive, `chmod 0755`s the binary it finds and calls `selfupdate.Apply` on the
  running executable. The SHA-256 it computes comes from the same
  caller-controlled release document, so it authenticates nothing; there is no
  host allowlist and no signature check. The route is registered unconditionally
  for every platform, including Android.
- **Mitigations that bound it:** the route is behind the dashboard session wall
  (it is absent from `isPublicLauncherDashboardPath`); archive extraction is
  guarded against path traversal in both the zip and tar paths
  (`updater.go:558-562`, `635-638`); and on Android `os.Executable()` is inside
  the read-only install directory, so the replace step cannot succeed. What
  remains is an authenticated arbitrary-URL fetch with archive extraction to a
  temporary directory. Nothing in PocketClaw's own UI calls it — no Flutter,
  Kotlin or dashboard code references `/api/update` — so it is inherited
  upstream desktop surface with no product use.
- **Interaction with `PC-DEF-020`:** in Public Mode the route is LAN-reachable.
  `PC-DEF-020` is now resolved, so Public Mode can no longer be active while the
  UI reports it off; the route is reachable only when the user has deliberately
  enabled LAN access.
- **Re-confirmed by the exposure-audit closure re-run, 2026-09-13:** still
  registered unconditionally, still absent from the unauthenticated allowlist,
  still takes a caller-supplied URL, both traversal guards intact, still called
  by no PocketClaw UI, and the Android apply step still cannot succeed against
  the read-only install directory. Classification unchanged; **not a release
  blocker for the GitHub APK path.**
- **Fix applied, 2026-09-13 — removed.** Securing an unused self-update
  subsystem would have been the wrong repair, so the exposed route is gone
  rather than hardened.

  `core/src/web/backend/api/update.go` is deleted and `router.go` no longer
  calls `registerUpdateRoutes`. No special response was invented: `embed.go`
  already answers an unknown `/api/` path with `http.NotFound`, so
  `POST /api/update` is now an ordinary 404 with no information-leaking envelope
  and no misleading fake success.

  **`pkg/updater` stays.** `cmd/picoclaw/main.go` registers its CLI update
  command, a legitimate non-HTTP consumer, and the library's archive-traversal
  guards and tests are untouched. The acceptance criterion was removal of the
  exposed product route, not maximum source deletion.

  Everything was re-proved before the change rather than taken from the audit
  report: the route was registered at `api/update.go:12`, it was absent from the
  unauthenticated allowlist so a live session was required, and a search across
  Dart, Kotlin, TypeScript and TSX found **zero** callers anywhere outside the
  backend package.
- **Verification:** `core/src/web/backend/api/no_update_route_test.go` — the
  route resolves to no registered pattern under five HTTP methods; an
  authenticated request driven through the routed mux, past any auth wall and so
  matching this route's actual threat model, returns 404 with no handler
  envelope; six plausible renames (`/api/updates`, `/api/self-update`,
  `/api/system/update`, `/api/upgrade`, `/api/download` and others) are checked
  so an arbitrary-download surface cannot reappear under a different name; the
  auth middleware is asserted not to name the path, so removal cannot have
  widened the unauthenticated surface; and the handler file's absence is
  asserted so a revert cannot restore dead code that reads as live product.
  Mutation-tested: restoring `update.go` and its registration fails them.

  The shipped binaries confirm it independently — `/api/update` occurs **zero**
  times in both `libpocketclaw.so` and `libpocketclaw-web.so`, and the web
  binary shrank by 132,864 bytes as the linker dropped the now-unreachable
  paths.

  No regression in the network or auth boundary: `web/backend`, `api`,
  `middleware`, `dashboardauth`, `launcherconfig`, `netbind`, `gateway` and
  `updater` all pass. Public Mode semantics, the dashboard session wall, the
  WebSocket session-plus-origin check and the gateway's loopback pin were not
  touched.
- **Core impact:** `core/src` changed, so the source fingerprint moved from
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` to
  `6f00359dc9e8bf7ee24f9d170754b2792a41fb880d9da4f34a8600dd8f99df00` and the
  pair was rebuilt and re-staged under the two-commit rule from build-input
  commit `a0be2a705c1b255c5bd2fe1d8c9f44c094019627`:

      libpocketclaw.so       37,724,640  7ebeebd1…  build ID 512ed36a…
      libpocketclaw-web.so   25,385,088  b682b76d…  build ID 4eeb1385…
      BuildTime              2026-09-12T23:25:34+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise. Native contract 22 PASS / 0 FAIL. No Managed
  Runtime payload was rebuilt. The private support manifest's `apk` field still
  names the exposure-audit APK `113a8382…`, which no longer contains this pair —
  the established pre-artifact state, rebound at the next artifact build.
- **Status:** RESOLVED, 2026-09-13.

### PC-DEF-025 — `flutter test` has been failing since H5B and no gate runs it

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** `test/unit/namespace_n3_native_identity_test.dart`;
  `tool/release_gate.py` Flutter coverage.
- **Severity:** Test and gate integrity. No product impact.
- **Description:** `flutter test` reports **480 passed, 1 failed**. The failure
  is "the build script still consumes the upstream artifact names", which
  asserts that `core/build-android-arm64.sh` contains the literal
  `build/picoclaw-android-arm64`. H5B made the output root overridable, so the
  script now builds that path from `CORE_BUILD_DIR` / `CORE_OUTPUT_ROOT` and the
  literal no longer appears. The assertion is stale; the behaviour it guards —
  that the install step still consumes the upstream artifact names — is intact.
- **Why it went unnoticed:** the release gate never runs the suite. It runs three
  named files only — `android_release_contract_test.dart`,
  `android_backup_exclusion_test.dart` and
  `android_runtime_secret_placement_test.dart`
  (`tool/release_gate.py:621-633`) — so a red Flutter suite has passed every
  gate since `aa24d9e`, through H5B, H5C, UI-1 and `PC-DEF-019`.
- **Evidence:** the literal is present in `core/build-android-arm64.sh` at
  `6c24f9a` and absent from `aa24d9e` onward; the test file has not been touched
  since `0f0332d`, which predates H5B.
- **Fix applied, 2026-09-13.** Both halves.

  **The assertion.** It now pins the *rename boundary* rather than a path: the
  install step must take upstream `picoclaw-android-arm64` and ship it as
  `libpocketclaw.so`, and `picoclaw-launcher-android-arm64` as
  `libpocketclaw-web.so`, matched as a regex over the real `install -m 0755`
  lines through `$CORE_OUTPUT_ROOT`. It additionally asserts the private-support
  step consumes the same two upstream names — a rename that missed it would
  archive symbols for the wrong binary — and asserts the *absence* of a
  hard-coded `build/` root, which is the shape that went stale. So the test now
  fails if the rename boundary breaks and also fails if someone reintroduces the
  fixed root, while surviving legitimate output-root relocation.

  Mutation-tested both ways against the real script: renaming the upstream
  artifact fails it, and hard-coding `build/picoclaw-android-arm64` fails it.
  The script was restored byte-for-byte afterwards.

  **The gate.** `tool/release_gate.py` now runs the **complete** Flutter suite
  as `flutter.suite`, using a deterministic `find_flutter()` that prefers the
  repository toolchain, then `FLUTTER_ROOT`, and only then `PATH` — a gate that
  answers differently depending on the caller's shell is not a gate. The suite
  runs **once**, through the JSON reporter, and the three named contract items
  (`a1.contracts`, `signing.production_contract`, `a2.placement_guards`) are
  derived from that single run rather than being the whole of it. They are kept
  because a record that says only "the suite passed" loses which guarantee was
  checked.

  A non-zero exit can never be reported as PASS, and exit 0 with no parsed
  results is a FAIL rather than a pass — a suite that did not run must not look
  like a suite that passed. Failure output is bounded and names the failing
  suite and test.
- **Verification:** `flutter analyze` clean; `flutter test` **490 passed, 0
  failed**. Proven against a real red suite: a deliberately failing test placed
  in a file none of the three named contracts covers made the gate exit 1 and
  report `490 passed, 1 failed — …/pc_def_025_probe_test.dart: deliberate
  failure outside the three named contract files`, while the three named items
  stayed PASS. That is precisely the scenario that went unnoticed for five
  milestones. The probe was removed. `tool/test_release_gate.py` grew from 24 to
  **35 tests**, covering full-suite success, a failure outside the named files,
  failing-test identity in the output, bounded output under 100 failures,
  non-zero exit never passing, exit-0-with-no-results failing, the command not
  being the three named files, a missing toolchain skipping rather than passing,
  a named contract failing when its own file fails, and PATH being last in
  resolution order.
- **Gate composition:** the source gate gains one item (25 → 26) and the full
  artifact gate 56 → 57. Measured at `3e3941f` before this change: **56 PASS / 0
  FAIL / 0 SKIPPED**, 56 items on a clean tree.
- **Core impact:** none. Tests, release tooling and documentation only; the
  source fingerprint is unchanged at
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and Core
  was not rebuilt.
- **Status:** RESOLVED, 2026-09-13.

### PC-DEF-021 — The AAB embeds the private R8 mapping and native debug symbols

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** Android App Bundle packaging; release-asset policy; release gate.
- **Severity:** **RELEASE BLOCKER for any path that publishes the AAB.** No
  impact on the APK or on installed devices.
- **Description:** `:app:bundleRelease` writes release-support material that the
  APK correctly excludes into `BUNDLE-METADATA/`:

      18,776,264  BUNDLE-METADATA/com.android.tools.build.debugsymbols/arm64-v8a/libflutter.so.sym
      13,630,085  BUNDLE-METADATA/com.android.tools.build.obfuscation/proguard.map
       6,763,120  BUNDLE-METADATA/com.android.tools.build.debugsymbols/arm64-v8a/libapp.so.sym
         159,016  .../debugsymbols/arm64-v8a/libdartjni.so.sym
         134,544  .../debugsymbols/x86_64/libdartjni.so.sym
         109,380  .../debugsymbols/armeabi-v7a/libdartjni.so.sym

  `proguard.map` is SHA-256
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb` —
  byte-identical to the private `mapping.txt`. `libapp.so.sym` is native debug
  data for the obfuscated Dart AOT library. Roughly 39.5 MB of the bundle is
  material H3A/H3B/H4A/H4B exist to keep out of distributed artifacts. This is
  AGP's intended design — Play consumes `BUNDLE-METADATA/` for crash
  symbolication and strips it from delivered splits — so it is correct for a
  Play upload and wrong for anything else.
- **Why it is not theoretical:** attaching the AAB to a public GitHub
  pre-release is this project's established practice.
  `PocketClaw-v0.2.0-rc1.aab` and `PocketClaw-v0.2.0-rc2.aab` are published
  assets today. Repeating that for a hardened release would publish the complete
  Java/Kotlin deobfuscation map and the Dart AOT symbols.
- **Why no gate caught it:** `RELEASE_PROCESS.md` states this material stays
  "outside APK/AAB files", which AGP cannot satisfy for a bundle, and
  `artifact.r8_mapping_private` in `tool/release_gate.py:864` is a hardcoded
  `True` whose observation reads "absent from APK" — it verifies nothing and is
  scoped to the APK. The repository also has no AAB build or inspection path at
  all: `tool/build_hardened_android.py` only runs `:app:assembleRelease`, and
  neither the release gate nor the native audit accepts a bundle.
- **Fix applied, 2026-09-12.** All four parts of the plan, and the framing
  changed: the defect is not that AGP writes those entries, it is that
  PocketClaw had no way to say what an artifact was *for*. Contents cannot be
  judged without purpose, so purpose is now declared.

  **(1) Policy corrected.** `RELEASE_PROCESS.md` no longer claims the mapping
  and symbols stay "outside APK/AAB files", which AGP cannot satisfy for a
  bundle and was therefore a rule nothing could obey. It now says: absent from
  every APK, never a public release asset, and expected inside a Play-destined
  bundle's `BUNDLE-METADATA/`. It also says explicitly not to strip them —
  that would remove Play's ability to symbolicate a crash and fix nothing.

  **(2) Distribution classes.** `tool/artifact_policy.py` defines
  `public-release`, `play-upload` and `non-publish-audit`. An AAB may never be
  `public-release`, and the refusal does not depend on what the bundle contains:
  a bundle with no mapping at all is still forbidden, because the prohibition is
  about what the format is for. Detection reads archive contents, not the file
  extension, so renaming a bundle to `.apk` does not launder it — a test covers
  that. A new public release asset allowlist rejects `*.aab`, mapping and usage
  reports, `.debug`/`.sym`/`.dwarf` companions, `.symbols`, symbol and
  private-support archives, keystores and key files, and `.env`, while leaving
  the APK, checksums, notices and source archives permitted.

  **(3) Real gate checks.** `artifact.r8_mapping_private` reads the archive and
  reports what it scanned — `"N archive entries scanned, deobfuscation entries =
  0"` — instead of the hardcoded `True` whose observation read "absent from
  APK". A test asserts the vacuous form cannot come back and that the same code
  answers differently for two different archives. `--verify-bundle` inventories
  modules, ABIs, manifests, native entries and every `BUNDLE-METADATA/` entry by
  name, size, category and Play-acceptability; `--artifact-class` is required for
  any artifact phase and has no default, so an unclassified artifact fails
  closed rather than being assumed public-safe. `--release-assets` checks a
  proposed asset list.

  **(4) rc1/rc2 left in place**, deliberately, and recorded — see below.

  Unrelated private material still fails in **every** class including a Play
  upload. The `BUNDLE-METADATA/` exemption covers exactly two known AGP entry
  shapes, `obfuscation/proguard.map` and `debugsymbols/<abi>/<lib>.so.sym`. That
  narrowing came from this milestone's own test suite: the first implementation
  exempted the whole directory, so a keystore dropped beside the mapping passed.
  It now fails.

  A repository-owned hardened bundle path was added rather than left to an ad-hoc
  Gradle invocation: `tool/build_hardened_android.py --package bundle` runs
  `:app:bundleRelease` through the same hardening contract as the APK — same
  obfuscation, split-debug-info, controlled generated URI, R8, shrinking and
  arm64 target — sharing the Dart verification helpers instead of duplicating
  them. It requires `--artifact-class` and does not offer `public-release`.
- **Verification:** `tool/test_artifact_policy.py`, 32 tests, covering all
  twelve required cases: a clean APK passes and an APK carrying a mapping fails;
  an AAB classified public fails, including one with no metadata at all; a
  Play-upload AAB passes and its expected `proguard.map` and native debug
  metadata are reported as allowed rather than as leakage; a non-publish audit
  AAB passes with the classification notice; a missing or unrecognised class
  fails closed; a Play AAB carrying unrelated private material fails; the asset
  allowlist rejects `*.aab`, mapping files, `.debug`, symbol archives and
  keystore-like material while permitting ordinary assets; and
  `artifact.r8_mapping_private` is shown to be content-derived. Six of those run
  the real gate CLI end to end. `tool/test_release_gate.py` (23 tests) still
  passes unchanged.
- **Historical exposure, not remediated here:** `PocketClaw-v0.2.0-rc1.aab` and
  `PocketClaw-v0.2.0-rc2.aab` remain attached to their published pre-releases.
  They predate Dart obfuscation and R8 minification, so what they disclose is
  not the current hardened mapping, but they are the practice this policy
  retires. This milestone had no authority to mutate published releases, so
  nothing was deleted and no release history was rewritten. The exact owner
  action, if removal is wanted, is recorded in `RELEASE_PROCESS.md` along with
  what it does and does not achieve: it ends ongoing public availability and
  cannot revoke a copy already downloaded, and each asset shows a recorded
  download.
- **Core impact:** none. Only tooling, tests and documentation changed; the Core
  source fingerprint is unchanged at
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and Core
  was not rebuilt.
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-020 — Public Mode OFF does not guarantee a loopback-only console

- **Discovered:** final release exposure audit, 2026-09-12.
- **Component:** Dashboard listener; Android Public Mode toggle; `launcher-config.json`.
- **Severity:** **RELEASE BLOCKER.** The product's stated network posture can
  differ from the listener it actually opens.
- **Description:** The dashboard's public/loopback decision has two persisted
  authorities that are never reconciled. Android stores the user's choice in
  SharedPreferences `public_mode` and passes `-public` only when it is on. When
  it is off the flag is absent, so `web/backend/main.go` takes
  `effectivePublic = launcherCfg.Public` — the `public` field of
  `launcher-config.json`. Nothing on the Android OFF path ever writes that file:
  `handleAndroidNetworkModeApply` and `launcherHTTPRuntime.ApplyPublicMode`
  rebind the live listener and update in-memory state only, and
  `PUT /api/system/launcher-config` is the single writer of the file. The
  console's own Config page does send `public`, so saving that page while LAN
  access is on persists `public: true`. After that, turning Public Mode off in
  the native UI rebinds the listener to loopback for the life of the process and
  leaves the file saying `true`; the next service start — app restart, service
  kill, device reboot — binds the console to all interfaces while the native
  toggle still reports OFF.
- **Evidence:** `core/src/web/backend/main.go:550-556` (`if !explicitPublic {
  effectivePublic = launcherCfg.Public }`);
  `core/src/web/backend/api/android_bridge.go:154-212` (no config write);
  `core/src/web/backend/launcher_http_runtime.go:134-161` (no config write);
  `core/src/web/backend/api/launcher_config.go:84-96` (the only writer);
  `core/src/web/frontend/src/components/config/config-page.tsx:701-706` (sends
  `public`). Reproducible bind evidence from the shipped `pkg/netbind` with the
  same default-mode selection `openLauncherListeners` applies:

      PUBLIC OFF (no -public, no host)   bindHosts=[::1 127.0.0.1]
      PUBLIC ON  (-public, no host)      bindHosts=[:: 0.0.0.0]
      host override 127.0.0.1 + -public  bindHosts=[127.0.0.1]
      gateway (host=localhost)           bindHosts=[::1 127.0.0.1]  in BOTH states

  The dashboard password wall is unaffected in every state, so this is exposure
  of a password-protected surface, not an unauthenticated one. The Core gateway
  on 18790 stays loopback-only regardless: `openGatewayListeners` always passes
  `netbind.DefaultLoopback` and never sees the launcher's public flag.
- **Fix applied, 2026-09-12** — option (b), plus the display half of (a).
  `PocketClawService` now passes `-public=true` or `-public=false` rather than
  the flag or nothing, so `flag.Visit` always reports the decision as supplied
  and the persisted field is never consulted on Android. `main.go`'s inline
  resolution moved into `resolveLauncherPublicMode` and its `flag.Visit` block
  into `launcherExplicitFlags`, so the state matrix is testable rather than
  arguable. Desktop is unchanged by construction: with no flag supplied the
  stored field is still the authority, which is the only way Public Mode can be
  set where there is no native toggle.

  The Config page was the second half. Making the host authoritative for the
  listener left the page reading and writing the stored field directly, so it
  could display a value the running listener contradicts and saving it rewrote
  the stale value. Where the host owns the decision the page now reports the
  effective mode and persists that instead of the submitted one, which also
  repairs a file that had already drifted. `effectiveLauncherPublic` already
  encoded the precedence and had no product caller; it is wired up rather than
  duplicated, and extended to prefer a runtime rebind over the startup flag
  because `ApplyPublicMode` replaces the listeners without rewriting
  `serverPublic`. The frontend is untouched — the field is informational under a
  host-owned decision by virtue of what the API reports, not by a redesign of
  Settings.

  Contract: `-public=<bool>`, always supplied by the Android host. An explicit
  `false` and an omitted flag are distinguishable because `flag.Visit` reports
  only flags that were `Set`; a test pins that, since the whole fix is inert
  without it.
- **Verification:** the required state matrix is covered by
  `core/src/web/backend/public_mode_authority_test.go` (fresh install off;
  native on; native on with a stored true; native off with a stored true for
  every subsequent process; a stored true predating startup; a stored false with
  native on; host-override precedence in all three public states; the gateway
  loopback for every gateway host value; and the end-to-end case that a stale
  stored true with an explicit off opens loopback sockets and nothing else),
  `core/src/web/backend/api/launcher_config_authority_test.go` (the page reports
  the effective mode, follows a runtime rebind, cannot override a host-owned
  decision, repairs a stale stored true on save, reports off under an explicit
  host, and stays writable on desktop), and
  `test/unit/android_public_mode_authority_test.dart` (the host always states
  the decision and never emits a bare `-public`). The live ON→OFF rebind was
  already covered by
  `TestLauncherHTTPRuntimeAppliesPublicModeWithoutReplacingHandler` and is not
  duplicated. No change to authentication, session handling, the WebSocket
  origin check, the unauthenticated path allowlist, the gateway's loopback pin,
  or Public Mode ON semantics; the middleware, dashboardauth, api,
  launcherconfig, netbind and gateway suites all pass.
- **Core impact:** `core/src` changed, so the source fingerprint moved from
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` to
  `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df` and the
  pair was rebuilt and re-staged under the two-commit rule from build-input
  commit `f8bc52a0757f7b0a9f6c0704d2a3586db929e33f`:

      libpocketclaw.so       37,724,640  602ce034…  build ID ed130bed…
      libpocketclaw-web.so   25,517,952  b5cce071…  build ID f61a369f…
      BuildTime              2026-09-12T18:54:26+0000

  Byte-identical in three independent output roots, one with a cold Go cache;
  both private companions likewise. Native contract 22 PASS / 0 FAIL for the
  pair. No Managed Runtime payload was rebuilt.
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-019 — Staged Core predates the guided-tour dashboard change

- **Discovered:** guided-tour hardening repair, 2026-09-12.
- **Component:** staged `libpocketclaw.so` / `libpocketclaw-web.so`.
- **Severity:** Release blocker for the next artifact build; no runtime defect.
- **Description:** `libpocketclaw-web.so` embeds the compiled dashboard, so
  everything under `core/src/web/frontend` is a Core build input. Repairing the
  tour moved the Core source fingerprint from
  `86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73` to
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`, so the
  staged pair no longer matches the source it is supposed to be built from.
- **Evidence:** `cmd/corefingerprint` recomputes the new value; the test-class
  source gate reports `core.staged_freshness` and `build.reproducibility_tests`
  FAIL with 22 PASS / 2 FAIL / 1 SKIPPED. Every other gate is unaffected.
- **Reason deferred:** This repair had no authority to rebuild Core. The H5C
  production artifact and its evidence remain historically valid; they simply
  describe the previous dashboard.
- **Resolution:** Core was rebuilt and re-staged on 2026-09-12 from canonical
  build-input commit `ea43369289c8b6c618faa08f7b91355882fc050c` — the UI-1
  source commit — under the two-commit rule, so the staged pair landed in a
  following commit that changes no build input and the source fingerprint stayed
  `bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`.

      libpocketclaw.so       37,724,640 bytes
                             f273b9ced85f4d00cb542df9c2f4c691b4151526cb0ac9c2c7612a1432d7230f
                             build ID 25e206ab402f8cd44a766bc03935468beebd8633
      libpocketclaw-web.so   25,517,952 bytes
                             900c43fcaad2094017c6959eed623d1e2499cfd560f01f2cff36dd34202b86b9
                             build ID 84afbe2439b779722b22c2b4c6aa1300cd3ef199
      BuildTime              2026-09-12T07:27:12+0000

  Byte-identical in three independent output roots, one with a cold Go cache.
  `core.staged_freshness` and `build.reproducibility_tests` are PASS; the whole
  `pkg/coresource` package passes, 46 tests. The embedded dashboard is identified
  by content rather than timestamp: UI-1's `__pocketclaw_tour_probe__` is present
  and the deleted docs-step copy is absent, both reversed in the binary staged at
  `ea43369`. The pair holds the H5B/H5C native contract at 22 PASS / 0 FAIL under
  the repository's own ELF audit logic, and both private-support companions were
  rebuilt and rebound. Only the Core pair changed; no Managed Runtime payload,
  export map or `PC-DEF-012` disposition was touched, and no APK or AAB was
  built. Evidence:
  [`docs/prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md`](prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md).
- **Status:** RESOLVED, 2026-09-12.

### PC-DEF-013 — Guided tour placed its card outside the viewport in RTL

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `core/src/web/frontend/src/components/tour/tour-guide.tsx`.
- **Problem/root cause:** Steps declared a physical placement (`"left"` /
  `"right"`). The sidebar anchors to the right edge in Arabic — see
  `sidebar-direction.test.tsx` — so the Models step computed
  `left = rect.right + 12` and put the card, and the only Next button, past the
  viewport edge. There was no flip and no clamp, so the step could not be
  advanced, dismissed or reached at all.
- **Resolution:** Placement is now logical (`start`/`end`/`above`/`below`),
  resolved against the document direction, and the finished rectangle is clamped
  inside the viewport with an 8px margin. The clamp is the guarantee: no layout
  can put a tour control out of reach.
- **Verification:** Measured in real Chrome and Brave at 1440x900. In Arabic the
  live target is `left 1193 / right 1432`; the pre-fix formula would have placed
  the card at `left 1444`, right edge `1764`, off-screen. It now renders at
  `861..1181`, fully inside, with the primary control hit-testable at its own
  centre. Both directions, every step, both browsers.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-014 — Guided tour left focus on the control it spotlighted

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`; the visible symptom surfaced on
  `SidebarMenuButton`.
- **Problem/root cause:** Reported as a highlight the tour failed to clear. It
  was not: the audit proved every tour node unmounts on every close path and
  that the tour never touches the target's classes, attributes or inline style.
  The spotlight was `pointer-events-none`, so a click on the highlighted item
  passed through to the real `<Link>`; the app navigated and the anchor kept DOM
  focus. `sidebarMenuButtonVariants` carries `outline-hidden focus-visible:ring-2`,
  so that anchor then painted a persistent ring almost identical to the tour's
  own spotlight, and it outlived the tour.
- **Resolution:** Two explicit policies instead of an accident. The spotlight now
  captures the click and swallows it, so the tour's own buttons own progression.
  And the tour captures the previously focused element when it opens and restores
  it on every termination path — finish, skip, Escape, click-outside, unmount —
  blurring instead when the opener is gone.
- **Verification:** In Chrome and Brave, clicking the spotlight leaves the path
  at `/`, keeps the tour open, and leaves no focused `[data-tour]` element. After
  the tour closes: `0` tour nodes, `activeElement` is `BODY`, no focused tour
  target. Route-active styling is untouched and still correct.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-015 — Guided tour declared a step for a control that does not exist

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx` step table; `tour.docs.*` translations.
- **Problem/root cause:** The `docs` step targeted `[data-tour='docs-button']`.
  No component has ever rendered that attribute, and the console has no
  documentation control for it to point at — the translated copy described a
  button in the top-right corner that does not exist. The step highlighted
  nothing in every layout and every language, and never failed loudly because
  a missing target fell back to a centred card.
- **Resolution:** The step is removed rather than answered with a new control
  invented to satisfy the tour. The now-unreferenced `tour.docs.*` keys are
  removed from all 14 locales and the i18n assertion re-pointed at a live step.
  The tour is three steps; the counter reads `1 / 3` through `3 / 3`.
- **Verification:** A structural test walks the step table and fails if any
  declared selector names a `data-tour` attribute no component renders, so a
  renamed target now breaks CI instead of shipping. Confirmed in-browser: the
  counter reads `1 / 3` in Chrome and Brave.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-016 — Guided tour never re-measured its target

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`.
- **Problem/root cause:** Geometry was read during the render a click produced
  and never again — the component had no effect, listener or observer of any
  kind. Scrolling, resizing, a breakpoint swap or the header finishing its first
  data load all left the spotlight stranded at coordinates that no longer meant
  anything. An all-zero rect from an unlaid-out element was also accepted as
  valid, producing a 16px spotlight at (-8,-8) that dimmed the whole screen from
  the corner.
- **Resolution:** A target is eligible only when connected, not `display:none`,
  not `visibility:hidden` and of non-zero size. Resolution retries on
  `requestAnimationFrame` under a finite 90-frame budget and then degrades to a
  centred card — never a fixed delay, never an unbounded wait. While a step is
  live, a `ResizeObserver`, capture-phase `scroll`, `resize` and a
  `MutationObserver` keep it synchronized, and the target is brought into view
  through its own scroll container with `scrollIntoView({block:"nearest"})`
  rather than scrolling the page. All of it is released on step change, close
  and unmount.
- **Verification:** Regression tests move the target, resize, remove it from the
  DOM and swap it for a replacement, asserting the overlay follows or degrades;
  a listener-balance test proves `scroll` and `resize` counts return to zero
  after close.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-017 — Guided tour backdrop blocked the app with no way out

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `tour-guide.tsx`.
- **Problem/root cause:** The no-target backdrop was a full-screen
  `fixed inset-0` layer without `pointer-events-none`; it absorbed every click
  aimed at the application. Escape did nothing, clicking outside did nothing,
  and nothing in the app could reopen the tour once dismissed. On a large
  desktop screen a dimmed, blurred, click-dead page with one small card reads as
  a freeze.
- **Resolution:** The dimmer is an explicit dismissal surface: clicking it ends
  the tour. Escape ends it too. The spotlight still blocks its target, by
  design, but it is the only blocking region and the card is always reachable.
- **Verification:** Tests assert Escape and a dimmer click each leave zero tour
  nodes and record completion, and that the centred fallback is dismissible.
  Confirmed in Chrome and Brave.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-018 — Guided tour state had no schema version

- **Phase discovered:** Guided-tour read-only audit, 2026-09-12.
- **Component:** `core/src/web/frontend/src/store/tour.ts`.
- **Problem/root cause:** `localStorage["pocketclaw-tour-state"]` held
  `{currentStep, isActive}` with no version, so a changed step list would strand
  users on a step that no longer exists and the tour could never be replayed
  deliberately. It also meant the reported "the tour comes back after updates"
  had no versioning explanation: the real cause is that the embedded WebView
  (`http://127.0.0.1:<port>`) and Public Mode (`http://<device-ip>:18800`) are
  different origins with independent storage, so a changed LAN IP presents a
  fresh origin. That is browser behaviour and is left alone.
- **Resolution:** State is `{version, currentStep, isActive}` with
  `TOUR_VERSION = 1` and a migration that reads an absent version as 0. Anyone
  who finished stays finished; a step this build no longer defines completes
  rather than stranding; a corrupt value falls back to the default instead of
  throwing; state written by a newer build is left alone. `localStorage` is
  probed and degrades to memory where it is unavailable.
- **Verification:** Migration tests cover completed state, a resumable
  unversioned step, the removed `docs` step, corrupt values and future versions.
  Confirmed in-browser: a stored completed state renders zero tour nodes, and
  legacy `{currentStep:"docs", isActive:true}` renders zero tour nodes rather
  than hanging.
- **Commit:** the guided-tour hardening commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-009 — Managed Git HTTP helper carried a build-only RUNPATH

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Managed Runtime `libpocketclaw-git-remote-http.so`.
- **Problem:** The packaged PIE carried `DT_RUNPATH`
  `/tmp/pocketclaw-runtime-build/deps/lib`, a build-host search path with no
  runtime purpose on Android.
- **Resolution:** H5B found the cause in git's own Makefile, which turns
  `CURLDIR` into `-Wl,-rpath,$CURLDIR/lib`. The recipe now passes
  `CURL_CFLAGS="-I$DEPS_PREFIX/include"` and an explicit `CURL_LDFLAGS` library
  list, with `-L$DEPS_PREFIX/lib` in `LDFLAGS`. No finished ELF was rewritten.
- **Evidence:** Candidate APK SHA-256
  `d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`; the
  enforced audit reports `no_runtime_search_path` PASS for all 18 packaged
  entries, and the payload still resolves only `libz.so`, `libdl.so`,
  `libc.so`. `install_payload` now fails on any RPATH/RUNPATH, not on one known
  root.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-010 — Three runtime payloads retained the neutral build root

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** Managed Runtime curl, Git HTTP helper, and Python payloads.
- **Problem:** curl and the Git HTTP helper each carried ten mbedTLS source
  paths under `/tmp/pocketclaw-runtime-build/`, and Python one CPython build
  root, despite an existing `-ffile-prefix-map`.
- **Resolution:** H5B added shared `-ffile-prefix-map` / `-fdebug-prefix-map` /
  `-fmacro-prefix-map` settings and widened ripgrep's `--remap-path-prefix` to
  the whole build root. Two cases needed more, because a prefix map cannot
  rewrite a string the build wrote into generated *source*: jq records its
  literal `CFLAGS` in `src/config_opts.inc`, and CPython compiles its
  configure-time `VPATH` into `getpath.c` as a C string literal. Both generated
  inputs are normalized before compilation, the CPython one only after the host
  build interpreter is complete.
- **Evidence:** `strings` over all ten payloads finds zero build roots,
  `/home/lordegypt` or checkout paths; the audit asserts `build_path_privacy`
  per entry, and `install_payload` fails the build if `$BUILD_ROOT` survives.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-011 — Native private symbol companions are now preserved

- **Phase discovered:** H5A native/ELF audit, 2026-09-11.
- **Component:** PocketClaw Core and Managed Runtime build recipes.
- **Problem:** All packaged payloads were stripped, correctly, but no recipe
  preserved a symbol-capable precursor, so a native crash address from a
  shipped build could not be resolved.
- **Resolution:** H5B builds every owned payload with debug information, strips
  the shipped copy, and derives a `.debug` companion from the same link through
  `tool/native_support.py`. Core drops Go's `-s -w` and strips the installed
  copy instead, which is what makes a Go companion possible; Python's companion
  comes from the interpreter before its standard library is appended, because
  that append is why the shipped file cannot be stripped.
- **Evidence:** Ignored `build/private-symbols/native/android-arm64/` holds ten
  companions and a 0600 manifest binding each to its shipped hash, size and
  build ID, plus a resolved representative function, and bound to the exact
  candidate APK. The audit's five `native.private_support_*` checks pass and
  nothing is tracked by Git. ripgrep's entry point is a qualified result and is
  recorded as such in the H5B operating record.
- **Status:** RESOLVED. Confirmed on the Samsung SM-A165F in the H5B
  physical smoke and again under the enrolled production signer in H5C, whose
  packaged native payloads are byte-identical to the artifact that ran on the
  device.

### PC-DEF-008 — Dart intermediate strip boundary verified

- **Phase discovered:** Post-H3B packaged-DWARF inspection.
- **Component:** Dart AOT intermediate / Android native-library packaging.
- **Problem:** Flutter warned that `gen_snapshot` emitted unobfuscated DWARF,
  raising the question whether source-level debug data reached the APK.
- **Resolution/conclusion:** H5A rechecked the exact H4B APK. Its 5,702,536-byte
  `libapp.so` is byte-identical to the H3/H4 Dart AOT evidence and is stripped:
  it has no `.debug_*`, `.zdebug_*`, `.symtab`, `.strtab`, source path, or
  application-name exposure. Its only dynamic exports are the three Flutter
  snapshot symbols; its 45-byte `.eh_frame` is unwind metadata. The ignored
  intermediate may contain DWARF, while the required private Dart split-debug
  file remains external.
- **Verification:** APK SHA-256
  `14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`;
  packaged `libapp.so` SHA-256
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`;
  `readelf`, `file`, `strings`, and the H5A automated audit agree.
  `gen_snapshot --strip` would not reduce distributed exposure because AGP
  already produces the desired packaged result; enabling it could interfere
  with the established external symbol/reproducibility contract without a
  demonstrated release benefit.
- **Commit:** H5A audit/closeout commit containing this record.
- **Status:** RESOLVED / VERIFIED NON-BLOCKING. Future Flutter/AGP changes must
  retain the packaged-DWARF regression check.

### PC-DEF-R018 — Runtime payload epoch was derived from `HEAD`

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `runtime/android-build-env.sh`.
- **Problem/root cause:** `SOURCE_DATE_EPOCH` defaulted to
  `git show -s --format=%ct HEAD`. `libpocketclaw-python.so` embeds that date
  literally, so any commit — documentation included — changed the bytes the
  Core catalog had just pinned. The commit recording a checksum would have
  invalidated it, and the catalog could never be reproduced from the tree
  carrying it. `core/resolve-build-time.sh` documents this exact failure for
  Core and solves it by path scoping, which cannot help here because the
  recipes are their own build input.
- **Resolution:** Pin `RUNTIME_EPOCH=1789157892` as a build input alongside the
  tarball checksums, still overridable by an explicit `SOURCE_DATE_EPOCH`,
  which is now validated as Unix seconds.
- **Verification:** A regression test asserts no `HEAD`-derived derivation
  remains, that sourcing the script resolves to the pinned value, and that the
  staged Python payload actually contains that epoch's UTC date. The pinned
  value equals the one the payloads were built with, so no byte moved.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R017 — Private companion path was taken from an unvalidated argument

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `tool/native_support.py`.
- **Problem/root cause:** The companion path was built directly from
  `--logical-name`. A name containing a separator or `..` would have written
  outside the private root, and the manifest's relative `supportPath` would
  then have been wrong about where the file is. The audit reads that name back
  out of the manifest, so the value is not purely internal.
- **Resolution:** Constrain the logical name to a plain file name and require
  the resolved path to stay directly inside the private root.
- **Verification:** A focused test rejects `../escape`, `nested/name.so`,
  `/absolute`, empty, `.` and `..`, and accepts a real payload name. The audit
  side has its own test that a manifest naming a support file outside its root
  fails `native.private_support_hashes`.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R016 — A malformed private manifest raised instead of failing closed

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `tool/native_support.py`, `tool/native_elf_audit.py`.
- **Problem/root cause:** Both tools assumed a well-formed manifest. A truncated
  or hand-edited file made `update_manifest` raise an opaque `KeyError`, and the
  audit crashed with an unhandled `CalledProcessError` when a `supportPath`
  named something `readelf` cannot parse. An audit must report on whatever the
  private root actually contains.
- **Resolution:** `update_manifest` reports a malformed manifest as an
  actionable error and leaves the file untouched; the audit treats a malformed
  manifest, a non-object artifact list and an unparsable support file as
  findings.
- **Verification:** Tests cover truncated JSON, a non-list `artifacts`, a list
  of non-objects, a top-level array, a non-ELF support file and a non-object
  `symbolization`; every case fails closed and the malformed file is unchanged.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R015 — Private symbols were archived before the release checks ran

- **Phase discovered:** H5B review of the inherited native implementation.
- **Component:** `runtime/android-build-env.sh`, `core/build-android-arm64.sh`.
- **Problem/root cause:** Both recipes captured the companion and wrote its
  manifest entry immediately after stripping, before the build-path privacy,
  RUNPATH, ABI and page-alignment checks. A build rejected by any of those would
  have left a support file and a manifest entry describing bytes that were
  never adopted.
- **Resolution:** Move the capture to the end of both recipes, after every
  check.
- **Verification:** A test asserts the capture appears after the `-trimpath`
  and source-fingerprint guards in the Core recipe, and the full third-root
  rebuild produced identical payloads and companions under the new order.
- **Commit:** H5B source commit `aa24d9e`.
- **Status:** RESOLVED.

### PC-DEF-R014 — ABI check could fail by SIGPIPE rather than by machine type

- **Phase discovered:** H5B third-root reproducibility build.
- **Component:** `runtime/android-build-env.sh`, `install_payload`.
- **Problem/root cause:** The check piped `llvm-readelf -h` into `grep -q` under
  `pipefail`. `grep -q` exits on its match, which can leave the reader writing
  into a closed pipe; the resulting SIGPIPE fails the pipeline for a reason
  unrelated to the machine type. It misfired once on a python payload whose
  bytes were provably correct and byte-identical to two other roots. This
  predates H5B and was fixed because it blocked the milestone.
- **Resolution:** Capture the header into a variable and match it with `case`.
- **Verification:** The same payload re-ran through `install_payload` and passed
  with the identical SHA-256 it had already produced in three roots.
- **Commit:** H5B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R013 — Release manifest still listed H4B as pending after validation

- **Phase discovered:** H4B production artifact inspection.
- **Component:** Release-gate status metadata.
- **Problem/root cause:** All 30 production artifact checks passed, but the
  manifest's static pending-hardening list still said R8 production-signed
  validation was pending. That completed-milestone text would make a valid H4B
  closeout internally contradictory.
- **Resolution:** Remove only the completed H4B item. Keep the independently
  open APK reproducibility and bootstrap-strategy items unchanged.
- **Verification:** A focused regression test requires the H4B item to be
  absent and the F-Droid reproducibility item to remain. The exact existing APK
  then passes the production artifact gate again without a rebuild.
- **Commit:** H4B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R012 — H4B helper selected the repository root as the Gradle project

- **Phase discovered:** H4B owner production-signing validation.
- **Component:** Temporary owner-local signing helper.
- **Problem/root cause:** The first helper invoked `android/gradlew` while its
  working directory remained the repository root. A Gradle wrapper locates its
  distribution but does not make its own directory the project root, so
  `:app:validateReleaseSigning` failed before assembly because the repository
  root is not a Gradle build.
- **Resolution:** Every helper Gradle invocation uses the canonical Android
  project explicitly with `android/gradlew -p android`. Before the first hidden
  prompt, the helper now verifies the Android settings and app build files and
  executes that exact validation route without signing variables, requiring
  the expected missing-material failure from `:app:validateReleaseSigning`.
- **Verification:** Twelve focused structural/order assertions and an EOF dry
  run proved the repository root is not selected, the Android project root is
  canonical, the validation task reaches `:app`, and all non-secret checks run
  before either password prompt. The corrected owner run then completed the
  production build and the exact APK passed 30/30 production artifact checks.
- **Commit:** H4B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R011 — H4A output assertion mistook R8 removal for missing obfuscation

- **Phase discovered:** H4A local-test validation.
- **Component:** Canonical hardened-build R8 evidence checker.
- **Problem/root cause:** The first H4A post-build assertion required at least
  four exact internal class entries in `mapping.txt`. R8 correctly renamed
  three probe classes and removed or folded three others, so the Gradle build
  and APK were valid but the new assertion reported only three mappings.
- **Resolution:** Account for each probe through its exact renamed mapping or
  through `usage.txt`/nested mapping evidence of removal or folding, and require
  every original clear DEX descriptor to be absent. Manifest components remain
  a separate exact-name preservation check.
- **Verification:** The already-built fresh APK passes the corrected output
  inspection and 30/30 local-test artifact gates. Focused fixtures cover both
  renaming and removal/folding, missing mapping/usage output, packaged mapping,
  blanket rules, and disabled minification/resource shrinking.
- **Commit:** H4A closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R009 — Cached Dart AOT could outlive its deleted split debug info

- **Phase discovered:** H3B owner production-signing validation.
- **Component:** Canonical Dart-hardened Android build helper / Flutter 3.47.1
  incremental build cache.
- **Problem/root cause:** The helper deleted the expected private DWARF before
  the build, but `:app:clean` did not invalidate `.dart_tool/flutter_build`.
  Flutter reused cached `app.so` because signing does not change Dart AOT inputs
  and the external split-debug-info file is not a tracked cache output. The APK
  assembled correctly while the required private symbol file was not recreated.
- **Resolution:** Clear only Flutter's generated `.dart_tool/flutter_build`
  cache before every hardened assembly so `gen_snapshot` must regenerate AOT
  and private DWARF as one pair. Refuse a symlinked cache path.
- **Verification:** The first diagnostic APK was production-signed and carried
  H3A-identical AOT but had no symbol file; its gate was 23 PASS / 2 FAIL / 0
  SKIP. After the fix, the owner rerun produced APK SHA-256
  `ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`
  with H3A-identical AOT and DWARF. The production artifact gate passed 25 / 25.
  Focused tests cover stale-cache removal, package-config preservation, symlink
  refusal, and cwd-independent symbol resolution.
- **Commit:** H3B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R010 — Owner signing helper prompted before Java preflight

- **Phase discovered:** H3B owner production-signing validation.
- **Component:** Temporary owner-local signing helper.
- **Problem/root cause:** The first temporary helper collected both owner
  passwords before checking `JAVA_HOME` and Java availability. Its trap still
  cleared the environment and no value was printed or stored, but the secret
  prompts occurred before all non-secret prerequisites had passed.
- **Resolution:** The corrected external helper validates JDK 17, Python,
  Gradle, repository/helper paths, and keystore presence before its first hidden
  prompt. The durable signing policy now requires this ordering for every future
  owner-secret helper.
- **Verification:** A missing-Java dry run exited before any prompt; a
  correctly configured EOF-only dry run completed all non-secret checks and did
  not begin a build. The subsequent owner rerun completed the production build,
  and the helper's exit trap cleared all four signing variables.
- **Commit:** H3B closeout commit containing this record.
- **Status:** RESOLVED.

### PC-DEF-R008 — Dart snapshot exposed an absolute generated-source URI

- **Phase discovered:** H2 artifact inspection; resolved in H3A.
- **Component:** Flutter/Dart release artifact and Gradle build path.
- **Problem/root cause:** `libapp.so` embedded
  `file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
  Flutter 3.47.1's Gradle plugin reads `filesystem-roots` and
  `filesystem-scheme` but does not forward those task fields to `flutter
  assemble`; direct and extra-frontend trials therefore left the absolute URI
  unchanged. The generated registrant also sits outside every package URI root
  in Pub's normal package config.
- **Resolution:** The canonical helper adds a deterministic generated-only
  package mapping before Gradle configuration. Flutter's own
  `toPackageUriForWorkspace` path then emits
  `package:pocketclaw_generated/dart_plugin_registrant.dart`. Gradle rejects a
  hardened compile without that exact mapping.
- **Verification:** Two clean local-test builds with different split-info roots
  produced identical Dart AOT SHA-256
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`;
  the H3A artifact gate records `artifact.dart_snapshot_paths` PASS and 25 PASS
  / 0 FAIL / 0 SKIPPED overall.
- **Commit:** H3A closeout commit containing this record.
- **Status:** RESOLVED; H3B subsequently validated the same contract under the
  enrolled production signer.

### PC-DEF-R001 — Keystore helper could report a blank fingerprint as success

- **Phase discovered:** H2 signer enrollment.
- **Component:** `tool/create_release_keystore.sh`.
- **Problem/root cause:** The helper discarded `keytool` stderr, hiding its
  password prompt and integrity warning. The remaining pipeline found no digest
  but still exited successfully through `tr`.
- **Resolution:** Preserve stderr, pass passwords by environment-variable name,
  share one fingerprint implementation, and validate the output shape.
- **Verification:** `tool/test_create_release_keystore.py` creates a disposable
  keystore, verifies the digest, and proves the passwordless path fails with
  empty stdout.
- **Commit:** `78d33fd5b179dd52c7cd8118a5d23ec19c8368ec`.
- **Status:** RESOLVED.

### PC-DEF-R002 — Signing pending-state text outlived completed work

- **Phase discovered:** H2 closeout.
- **Component:** Release-gate state and signing documentation.
- **Problem/root cause:** After enrollment, the gate correctly narrowed “key not
  created” to “no artifact signed”; after private validation, that second state
  also became stale.
- **Resolution:** Remove the resolved pending item and update the authoritative
  H2 state without changing signing logic.
- **Verification:** Production artifact gate: 20 PASS, 0 FAIL, 1 expected SKIP;
  production source gate: 21/21 PASS.
- **Commits:** `78d33fd`, `7f19309`, `0be6afb`.
- **Status:** RESOLVED.

### PC-DEF-R003 — `gh` source recipe broke after canonical environment import

- **Phase discovered:** H1.5 source-build proof.
- **Component:** Managed Runtime `gh` build recipe.
- **Problem/root cause:** The Android DNS resolver gained a `canonicalenv`
  import, while the vendoring guard permitted standard-library imports only.
- **Resolution:** Vendor the stdlib-only `canonicalenv` leaf beside the resolver
  and allow exactly that import.
- **Verification:** Two source builds produced identical adopted bytes; manifest
  checksum and packaged payload match.
- **Commits:** `6f117d3`, adopted by `51ed822`–`fb38c7d`.
- **Status:** RESOLVED.

### PC-DEF-R004 — Python payload embedded wall-clock ZIP timestamps

- **Phase discovered:** H1.5 source-build proof.
- **Component:** `runtime/python-lite-stdlib.py`.
- **Problem/root cause:** Appended standard-library ZIP entries used wall-clock
  timestamps, making each payload different.
- **Resolution:** Use `SOURCE_DATE_EPOCH`, fixed permissions, and a sorted walk.
- **Verification:** Consecutive pinned-epoch builds were byte-identical; adopted
  payload, manifest checksum, and packaged payload match.
- **Commits:** `6f117d3`, adopted by `51ed822`–`fb38c7d`.
- **Status:** RESOLVED.

### PC-DEF-R005 — Canonical APK packaged Firebase/GMS and fetched fonts at runtime

- **Phase discovered:** H1/H1.5 F-Droid audit.
- **Component:** Flutter dependencies, Android manifest, and app typography.
- **Problem/root cause:** Proprietary SDK dependencies were packaged regardless
  of runtime use, and `google_fonts` defaulted to network fetching.
- **Resolution:** Remove Firebase/GMS from the canonical build; bundle Inter and
  Fira Code with their license texts and remove `google_fonts`.
- **Verification:** Source/artifact gates find no Firebase/GMS/AdMob/measurement
  surface; font contract tests prove local packaged assets.
- **Commits:** `612ce96` and `6f117d3`.
- **Status:** RESOLVED.

### PC-DEF-R006 — Artifact-only Core provenance check had no comparison value

- **Phase discovered:** vc62 Zero-Pico artifact validation.
- **Component:** `tool/release_gate.py`.
- **Problem/root cause:** `artifact.core_provenance_pair` read a fingerprint fact
  populated only by source mode, so correct artifact-only verification failed.
- **Resolution:** Resolve and cache the Core fingerprint on demand for both
  source and artifact paths.
- **Verification:** vc62 artifact gate and later H2 production artifact gate
  both pass the provenance-pair check.
- **Commit:** `84080a5`.
- **Status:** RESOLVED.

### PC-DEF-R007 — H2 closeout initially counted the summary as a gate item

- **Phase discovered:** H2 documentation closeout.
- **Component:** Evidence reporting.
- **Problem/root cause:** A temporary count included the final `PASS — ...`
  summary line in addition to the named checks.
- **Resolution:** Count only named gate rows and correct all recorded totals.
- **Verification:** 20 named PASS rows, 0 FAIL rows, 1 named SKIPPED row.
- **Commit:** `0be6afbd92209953d918d5c0516662bc54f0d081`.
- **Status:** RESOLVED.
