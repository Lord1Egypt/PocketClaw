# PocketClaw Upstream Tracking

This file is the checkpoint of record for upstream review. It answers one
question: what upstream state has already been reviewed, so the next review
starts after that point instead of restarting from the baseline.

The immutable adoption record — which upstream release and commit PocketClaw
originally started from, and when — lives in `UPSTREAM_BASELINE.md` under
"Adoption record". This file records movement; that file never changes.

## Checkpoint

### PicoClaw Core

Upstream repository:
https://github.com/sipeed/picoclaw

Baseline release:
v0.3.1

Baseline commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Last upstream review date:
2026-08-25

Last reviewed upstream release:
v0.3.1

Last reviewed upstream commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Latest upstream release known at review time:
v0.3.1 (published 2026-07-03; verified against the GitHub releases API on
2026-08-25 — `releases/latest` and the full tag list both name v0.3.1 as the
newest version tag)

Latest upstream commit/tag known at review time:
`bbf6893ca7afad27f1d00a0f5a45982a549c6ed6` on `main`, authored 2026-08-19
(`feat(models): add configurable default fallback chain (#3200)`). The rolling
`nightly` release tag still points at the v0.3.1 commit.

Distance from current upstream:
PocketClaw is level with the newest upstream *release* and 19 first-parent
commits behind upstream `main`. The raw `2cf030d2..main` range counts 2583
commits because `main` has merged in a second, unrelated root history; the
first-parent count is the meaningful figure for review planning.

Next review must begin AFTER:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

### PicoClaw FUI

Upstream repository:
https://github.com/sipeed/picoclaw_fui

Baseline commit:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

Last upstream review date:
2026-08-25

Last reviewed upstream release:
picoclaw_fui-v0.1.4 (tag `v0.1.4`)

Last reviewed upstream commit:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

Latest upstream release known at review time:
picoclaw_fui-v0.1.4 (published 2026-06-04; verified against the GitHub
releases API on 2026-08-25)

Latest upstream commit/tag known at review time:
`d689c94c1b67f625f70ec4111a9aa3f01be9cbb3` — `main` HEAD equals the baseline
commit and equals tag `v0.1.4`.

Distance from current upstream:
Zero commits behind. The FUI has not moved since the baseline.

Next review must begin AFTER:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

## Upstream Review History

This history is append-only. Never edit or delete an existing entry; every
review adds a new one below the last.

### Review 1 — 2026-08-25

PocketClaw Core baseline:
v0.3.1

Baseline commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Reviewed upstream from:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Reviewed upstream through:
2cf030d2fd3b871d7ec17e3be34c24688aac76da (Core)
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3 (FUI)

Latest release observed:
Core `v0.3.1` (2026-07-03); FUI `picoclaw_fui-v0.1.4` (2026-06-04). Both
verified live against the GitHub API on 2026-08-25.

Scope of this review:
Establishing the checkpoint. This review verified the adoption record against
real upstream metadata and measured the distance to current upstream. It did
not classify the 19 first-parent commits that landed on Core `main` after the
baseline — those remain unreviewed and are the subject of the next review.

Changes adopted:

- Android active-network DNS integration, carried from the baseline commit
  (`ConnectivityManager.activeNetwork` → `LinkProperties.dnsServers` →
  `PICOCLAW_DNS_SERVER` → the Go runtime).
- Optional device-feedback behavior, carried from the baseline commit.

Changes reviewed but not adopted:

- The 19 first-parent commits on Core `main` after v0.3.1 were counted and
  bounded but not classified. Nothing from that range is adopted.
- Core `main` also carries a merged unrelated root history, which inflates the
  raw commit count; it is not upstream product work to adapt.

Not upstream work at all (PocketClaw-authored, recorded here so a future
review does not mistake it for an adoption):
product identity and original PocketClaw visual assets; the Milestone C
provider catalog, presets, category metadata, and provider-first UI; the
Gemini discovery branch; the OpenCode Zen/Go presets and per-model protocol
routing; and the generic Responses provider. These live in
`core/pocketclaw-core-v0.3.1.patch`.

PocketClaw commit(s):
`950d4a3` (baseline adoption) through `b1fbee3` on `feature/provider-catalog`.

Next review must begin AFTER:
2cf030d2fd3b871d7ec17e3be34c24688aac76da (Core)
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3 (FUI)

## Upstream is an update source, not a build dependency

Since 2026-08-25 the Core source is vendored into this repository at
`core/src/`, which is the canonical build source. Upstream is fetched only to
review it, never to build from. Concretely:

- No build script, Makefile target, or Gradle task reads a checkout outside
  this repository. `core/verify-no-external-source.sh` proves it by hiding the
  historical reference checkout and rebuilding.
- The app never downloads Core source or binaries at runtime. The APK ships the
  approved binaries, and updates are deliberate PocketClaw releases.
- Adapting an upstream change means editing `core/src/` and regenerating
  `core/pocketclaw-core-v0.3.1.patch`, not re-pointing a build at an upstream
  tree.

This does not restrict upstream review in any way — the checkpoint above is
still what bounds it.

## Review Workflow

When PocketClaw is asked "check what is new in PicoClaw", the workflow is
fixed:

1. Read the last reviewed-through commit from this file's Checkpoint section.
2. Fetch current upstream.
3. Compare **only** changes after that checkpoint.
4. Classify the useful changes.
5. Selectively adapt them into PocketClaw's repository-local Core (`core/src/`).
6. Test, build, and physically verify.
7. Regenerate `core/pocketclaw-core-v0.3.1.patch`, update the Checkpoint
   section, and append a new Review entry.

Never restart upstream analysis from v0.3.1 once later commits have been
reviewed. Never overwrite review history.

Upstream changes are classified and manually reviewed before selective
adaptation. Automated merges from upstream are not a maintenance model.

## PocketClaw-Authored Work, Not Upstream

Recorded here so a future upstream review does not mistake PocketClaw's own
work for an adoption, and does not go looking upstream for its origin.

- **Pre-release Android reliability and privacy work (Codex Sol candidate,
  2026-08-26).** This is PocketClaw-authored divergence, not an upstream
  adoption. It adds the loopback-authenticated, write-only Android Telegram
  credential bridge so Core remains the sole owner of its split
  `config.json`/`.security.yml` state; bounded Telegram HTTP requests, safe
  per-request correlation, independently completing Telegram session
  mailboxes, synchronous final delivery, edit-to-send fallback, and terminal
  placeholder cleanup; non-destructive workspace seed repair and additive
  skill-import coverage; and Android-safe plain logging, including basename
  callers, a non-terminal banner, neutral stale-PID wording, UTF-8 export, and
  suppression of only successful high-rate log/status self-polls. Future
  upstream work touching agent session steering, Telegram delivery, onboarding
  helpers, or web API routing must preserve these PocketClaw guarantees.
- **User-visible Web log parity (2026-08-27).** PocketClaw additionally owns the
  shared plain-text gateway log boundary, React plain-log renderer and DOM
  regression, captured no-color text banner, neutral gateway-start event, and
  successful `/pico/ws` visibility filter. The actual route, executable/library
  filenames, Go module identity, and provenance remain upstream-compatible and
  were deliberately not renamed. Future upstream log UI, CLI banner, gateway
  launcher, or HTTP middleware changes must preserve Unicode, error
  observability, and the fixture-backed visibility contract.
  The enabled-channel startup/reload summary also maps only exact internal
  `config.ChannelPico` to display label `pocketclaw`; the upstream-compatible
  channel/config/protocol identity remains `pico` everywhere else.
  Structured user-visible log headers additionally display exact component
  `pico` / caller `pico.go` as `realtime` / `realtime.go`, with line numbers
  preserved. This normalization does not rename upstream source or runtime IDs.
  The Web Logs viewport also uses PocketClaw-owned stable run/offset event IDs,
  memoized plain-text rows, browser-native wrapping, and conditional pre-paint
  bottom following. This replaces upstream-derived array-index rows and
  content-measured `wrap-ansi` hard wrapping; native log delivery is unaffected.
  PocketClaw also replaces Telego-compatible partial token masking with complete
  pre-writer credential redaction, then normalizes Web-stored Bot API URLs to
  operation-only diagnostic wording. Full tokens had already been masked before
  Web storage; this removes the retained fragments without changing Telegram
  credentials, lifecycle, API calls, or public bot metadata.
  Exact structured `channel=pico` / `type=pico`, classified protocol/reasoning
  messages, the internal registration path, and compatibility PID path also
  receive semantic display-only wording before Web storage. This does not alter
  upstream ChannelPico, serialized IDs, packages/files, routes, the actual PID
  file, libraries, or environment variables; substring-negative tests enforce
  that boundary.
  PocketClaw also narrows the upstream-derived orphaned-CSI fallback to numeric
  remnants so Telego's printable `[<nil>]` cannot be mistaken for terminal
  control data. Successful nil fields display semantically as `none`.
  PocketClaw's Telego adapter suppresses only DEBUG `getUpdates` request lines
  and exact successful empty responses before writers; failure/non-empty/API
  error diagnostics and all Telegram runtime behavior remain upstream-compatible.
  PocketClaw additionally changes the freshly generated default product identity
  to PocketClaw and enforces metadata-only normal diagnostics before writers:
  prompt/message/tool/reasoning bodies and tool arguments are omitted, exact
  session/internal fields are redacted on a copy, and Telego result payloads are
  reduced to operation/status/count/type metadata. Runtime session, routing,
  Telegram, ChannelPico, module and user-authored prompt values remain unchanged.
  Backend/React/Dart normalization is retained only as an idempotent legacy/raw
  guard. Future upstream Agent or Telego logger updates must not reintroduce
  payload bodies or personal identifiers into normal user-visible history.
- **Telegram managed-bot onboarding (Milestone D, 2026-08-26).** The
  onboarding service is PocketClaw-authored and depends on no upstream code. It
  lives in its own public repository, `Lord1Egypt/PocketClaw-Telegram-Setup`,
  and no part of it is vendored here. The Flutter onboarding screen is likewise
  PocketClaw's own.
- **The Milestone D UI integration does touch `core/src/`** and therefore does
  appear in `core/pocketclaw-core-v0.3.1.patch`, which grew from 52 to 58 files
  on 2026-08-26. Under `core/src/web/frontend/src/` it adds
  `components/channels/channel-forms/telegram-panel.tsx`,
  `components/channels/channel-forms/telegram-surface.ts`,
  `lib/pocketclaw-host.ts`, `components/channels/channel-config-page.telegram.test.tsx`
  and `test/setup.ts`, and modifies `components/channels/channel-config-page.tsx`,
  `i18n/locales/en.json`, `vite.config.ts` and `package.json`. It also adds
  `jsdom` and `@testing-library/react` as frontend dev dependencies.
  None of this is an upstream adoption: it is PocketClaw's own work living
  inside vendored Core because the channel configuration UI lives there. A
  future upstream review must not go looking upstream for its origin, and must
  expect a conflict here if upstream changes `channel-config-page.tsx`.
- It adopts nothing from PicoClaw. It uses official Telegram Bot API 9.6
  managed-bot support directly, verified against `core.telegram.org` rather
  than inferred from any other implementation.
- Hermes Agent's managed-bot onboarding was read as a **behavioural**
  reference — the shape of the flow: a pairing session, a deep link and QR,
  polling, and a token handed back on completion. No Hermes source was copied,
  and no Hermes or Nous service is contacted at build time or at runtime.
  Behavioural inspiration is not a source dependency, and this milestone
  introduced neither an upstream nor a third-party one.
- The existing Telegram **channel** in `core/src/pkg/channels/telegram/` is
  upstream PicoClaw code and remains so. Milestone D did not modify it: the new
  flow writes the same `channel_list.telegram` configuration manual setup
  writes, so `core/src/` is unchanged by this milestone.

## Verified Upstream-Backed Behaviors

- ClawHub Skill Hub search is available after the Android active-network DNS
  fix. The prior registry-unavailable symptom is resolved by DNS, not pending
  an independent FUI/registry change.
- The Android active-network DNS integration survives vendoring. It was
  re-verified on a physical device on 2026-08-25, after the Core source moved
  into this repository at `core/src/` and was rebuilt with `-trimpath`: Skill
  Hub search and Fetch Models both passed, and both fail closed without working
  DNS. Adapting the source location did not regress the one upstream-derived
  behavior PocketClaw most depends on.

## Security Updates

None currently recorded. Note that upstream `49183d7e`
(`fix: update Go and x/text for govulncheck`) sits in the unreviewed range and
is a candidate for the next review.
