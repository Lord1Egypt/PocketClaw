# PocketClaw Session Handoff

## In progress — Resume white-screen recovery (2026-08-30)

Branch `feature/provider-resilience-failover`, on top of `ebf49b4`. Not merged.
**Physical validation PENDING.**

### What not to change

- **Never reload on every resume.** The probe exists precisely so a healthy page
  is left alone; a blanket reload would discard scroll and page state every time
  and hide the real defect.
- **Blankness is reported by the page, not sampled from pixels.** A white area is
  a symptom. `window.__pocketclawReady` plus a non-empty `#root` is the signal.
- **Recovery is capped at one attempt per page load.** A console that is
  genuinely broken must stop being reloaded so the Refresh control stays usable.
- **A resource error during a gateway restart is not a trigger.** The backend is
  briefly unavailable by design; only a resume that finds a dead page acts.

### Still open

Which failure mode actually occurs on the device. `webview_flutter_android`
4.14.0 has no `onRenderProcessGone`, so renderer death cannot be observed
directly — the new `[webview]` logs separate `renderer_gone` (probe threw) from
`blank` (page answered but is empty, i.e. a console crash). Read those on the
next occurrence before drawing a conclusion.

## In progress — Fallback UI and automatic gateway restart (2026-08-30)

Branch `feature/provider-resilience-failover`, on top of `68443c1`. Not merged,
`main` untouched. **Physical validation PENDING.**

### Boundaries worth keeping

- **The fallback chain is `Agents.Defaults.ModelFallbacks`, not
  `config.ModelConfig.Fallbacks`.** The latter serves multi-key expansion within
  one provider and is generated, not user-edited. Editing it from the UI would
  target the wrong mechanism.
- **A fallback is a reference by model name.** The referenced entry supplies its
  own provider, credentials, base URL and headers. Never copy the primary's key
  into a fallback; that would send one provider's secret to another's endpoint.
- **`apply-config` is deliberately separate from `POST /api/gateway/restart`.**
  The manual restart is immediate recovery the user asked for. The automatic one
  is a consequence of saving settings and must wait for the gateway to be idle,
  or it will cut off a Telegram reply mid-sentence or kill a `git push` half way
  through. Do not merge the two.
- **`busy` and `active_requests` are pointers on purpose.** "Not reported" is not
  "idle", and it never authorises a restart. The idle check has four outcomes:
  `not_running` and `idle` permit a restart, `busy_timeout` and `unverified` do
  not. The two-minute limit bounds the *wait*, not the safety of the user's work
  — reaching it leaves the config saved and unapplied. Do not "fix" either case
  by restarting anyway; an earlier revision did, and it was wrong.
- **There is only one restart decision.** It is the backend's existing
  `gateway_restart_required` signature comparison. Do not add a second.
- **Readiness is signature-matched, not HTTP 200.** Success means the gateway is
  running *and* booted the configuration that was just saved.

### Next

Physical validation: configure a failing primary, add a working fallback from
the UI, save, and confirm the gateway restarts by itself and the fallback
answers.

## In progress — Provider Resilience & Automatic Failover (2026-08-30)

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`.
Not merged, not released, `main` untouched. **Physical validation PENDING.**

### The one thing not to undo

The agent loop guarantees that a provider retry does not rewind a completed tool
execution: results are committed to the turn and the session in
`pipeline_execute.go` before control returns, and the loop in `turn_coord.go`
never re-enters a finished tool call. A retry re-sends the committed results.

**Do not add a checkpoint subsystem, tool fingerprinting, or side-effect
classification to defend this.** They were explicitly scoped out because the
structure already provides the property, and each would add surface area and new
ways to be wrong. It is protected by tests and by one guard in
`turn_tool_results.go`.

That guard matches on the provider's `tool_call_id` and nothing else. Do not
"improve" it to match on tool name or arguments: asking for the same command
twice in one turn is legitimate, and collapsing those would silently change what
the agent did. A call with no id is not recorded, for the same reason.

### Other boundaries

- **Unknown capability is not a refusal.** The gate skips a candidate only on an
  explicit unsupported. PocketClaw routes to providers whose model lists it does
  not enumerate; treating unrecognised as unusable would disable failover where
  it matters most.
- **Hard-quota patterns are narrow on purpose.** Widening them until they catch
  ordinary throttling would put healthy providers into long cooldowns.
- **An empty completion is not an outage.** `responseIsUserVisiblyEmpty` is for
  logging and the placeholder only. A tool-call-only response is not empty.
- **Streaming failover stops at first visible output.** Unchanged, and it must
  stay that way or answers duplicate on screen.

### Deferred

Cross-provider context-overflow fallback, for want of reliable per-model context
capacity metadata. See `DECISIONS.md`.

### Next

Physical validation, then merge.

## Next milestone — Provider Resilience & Automatic Failover

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`.
Nothing implemented yet; the branch exists so the work starts from the merged
Runtime v2 baseline rather than from a feature branch.

A provider that rate-limits, times out, or returns nothing should degrade into a
retry or a fallback, not into a failed turn the user has to notice and repeat.

**Detect:** HTTP 429 (honouring the provider's own retry hint where it sends
one), 502/503/504, provider timeouts, and empty model responses *only* where the
emptiness is attributable to provider failure. A model that legitimately returns
nothing must not be retried as though it had errored — that distinction is the
first thing to get right, because getting it wrong turns a quiet answer into a
loop.

**Respond:** bounded retries, provider cooldown so a failing provider is not
hammered by every subsequent request, and automatic fallback to the configured
backup model or provider.

**The hard part is correctness under retry, not detection.** Completed tool-call
results must be preserved across a retry or failover, and a tool that already
succeeded with side effects must never be blindly re-run. A retry that re-sends
a message, re-pushes a commit, or re-writes a file is worse than the failure it
is recovering from. Design for that first and the rest follows.

**Observability:** every retry and failover decision gets a clear lifecycle log
saying what failed, what was decided, which provider was chosen and why. No
provider secrets on any path — the runtime's redaction rules apply here too.

Full scope in `TASKS.md`.

## Shipped — Lean Runtime Pack v2 (2026-08-30, PHYSICAL PASS)

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`, from `develop` at
`fa27ad2`. **Physical validation PASSED** on a real ARM64 device on 2026-08-30,
then merged to `develop`. Not released, `main` untouched. Read `RUNTIME.md`
first.

Physical: Git HTTPS PASS, `git clone` of a public GitHub repository PASS,
**Git helper symlink execution on Android PASS**, git 2.51.0, gh 2.82.1, curl
HTTPS PASS, ripgrep PASS, sqlite3 PASS.

Skills: **7/7** on an existing upgraded workspace, **6/6** on a fresh install.
Both are correct — seeding only writes and never deletes, so an existing device
keeps the GitHub Skill it already had, while a fresh workspace gets the six
seeded skills (`picoclaw-agent` is deliberately unseeded).

git 2.51.0, gh 2.82.1, curl 8.11.1, ripgrep 14.1.1 and sqlite3 3.50.4 now ship
alongside jq. APK 34.7 MB -> 58.3 MB.

### Fixed after the first physical run

`git clone` failed with `unable to find remote helper for 'https'`. That message
names the protocol and sends you to TLS; the actual cause was that git spawns
`git remote-https` and resolves the literal name `git` through PATH, and the
helper directory held only the two remote helpers. git now declares itself as
one of its own helpers. If a tool re-invokes itself by name, it needs an entry
for itself — that is the general lesson.

### The platform question that is now answered

`git clone` over HTTPS depends on **executing through a symlink** in app-private
storage that points at a packaged payload. Android cannot package a file named
`git-remote-https`, so the runtime builds a symlink directory and points
`GIT_EXEC_PATH` at it.

**The device confirmed this works.** A symlink is not an executable: the kernel
resolves it and runs the read-only packaged file, so the API 29+ restriction on
executing writable app storage does not apply. This is the mechanism that makes
multi-executable tools possible on Android at all, and it is now evidence rather
than reasoning.

Do not "simplify" it by copying a helper into app storage. That is the one thing
Android definitely refuses, and the reason this design exists.

### What a next session must not undo

- **No credential ever goes in argv.** git gets its token through
  `GIT_CONFIG_COUNT`/`GIT_CONFIG_KEY_n`/`GIT_CONFIG_VALUE_n`, gh through
  `GH_TOKEN`. Do not "simplify" this to `https://TOKEN@github.com/...`: that
  leaks the credential into the command line, into git's on-disk remote config,
  and into any error quoting the URL.
- **Helper payloads are verified like main payloads.** A tool whose helper fails
  its checksum resolves unavailable on purpose. git with an unverified transport
  helper looks installed and then fails at the first `https://` URL.
- **gh is not a precedent.** Anything else above roughly 10 MB installed needs
  its own justification. yq was rejected at 11.25 MB with jq already present.
- **The build-path privacy check is deliberately narrow.** It tests for this
  build's own home directory plus boundary-anchored developer roots. A bare
  `/root/` search false-positives on Go's trimmed module paths such as
  `pkg/root/trusted_root.go`. Do not widen it back.
- **curl is real curl.** Do not replace it with a lookalike, and do not name any
  in-house HTTP tool `curl`.

### Where things are

- `runtime/android-build-env.sh` — shared cross-build setup and the
  strip/verify/install step every payload goes through.
- `runtime/build-{curl,git,ripgrep,sqlite3,gh}-android-arm64.sh` — one per
  payload. Run curl before git; git links the libcurl it leaves behind.
- `runtime/patches/git-android-pthread-cancel.h` — the bionic shim, and the only
  PocketClaw modification to git's source. It is also what satisfies the GPL
  source-offer obligation alongside the pinned tarball.
- `core/src/pkg/pcruntime/{helpers,environment}.go` — the symlink farm and the
  per-tool environment profiles.

### Known limitation

git ships with its default compiled-in `SHELL_PATH` of `/bin/sh`, which Android
lacks. **Shell-dependent git features — hooks above all, and git's `ENOEXEC`
fallback — are not guaranteed on Android.** Clone, fetch and push do not need a
shell. Overriding `SHELL_PATH` breaks git's own cross-build because its Makefile
uses the same variable for its build recipes; see `DECISIONS.md`.

### Next

Provider Resilience and Automatic Failover, on its own branch from the new
`develop` HEAD. SSH is worth reconsidering now that HTTPS git is proven, and the
deferred release-engineering items in `TASKS.md` still stand, including the
high-priority `extractBinaryFromApk()` dead path.

## Shipped — Managed Runtime Foundation (2026-08-30, PHYSICAL PASS)

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. **Physical validation PASSED** on a real ARM64 device
on 2026-08-30, then merged to `develop`. Not released, `main` untouched. Read
`RUNTIME.md` first; it is the architecture of record.

Physical results: runtime tool registered; jq 1.7.1 executed and processed JSON;
`sha256sum`, `grep`, `sed`, `tar`, `uname`, `df`, `ping` executed; stderr
captured; non-zero exit preserved; timeout terminated a long-running command;
lifecycle logs appeared; no secret leakage; the runtime was driven through
Telegram; Auto-Start and the Gateway PID fix held.

**43 of 44** catalog tools were available. `traceroute` was correctly reported
unavailable — the resolver measuring the device instead of trusting the catalog,
which is exactly what it is for.

**The writable-app-data probe returned INCONCLUSIVE on that device**, and that is
fine. Nothing depends on it: the bundled jq payload executed from
`nativeLibraryDir` on the same run. Do not read an inconclusive probe as licence
to try writable execution — the two supported routes are unchanged.

### Counts, and one that will look wrong

- Tools: **18**
- Skills: **7/7**

**Skills 7/7 is correct and is not a regression.** The user deliberately removed
the incomplete GitHub Skill. Do not restore it, do not write a migration for it,
and do not treat 8/8 as the target.

Note for whoever picks this up: `core/src/workspace/skills/github/` is still in
the repository, so a *freshly seeded* workspace would receive it again. The
device count of 7 and the repo's seeded set are therefore consistent today only
because the device workspace already exists. If 7/7 is meant to hold for new
installs too, the skill needs removing from the seeded tree or adding to
`unseededTemplates` in `cmd/picoclaw/internal/onboard/helpers.go`. That was not
done here because it was not asked for.

### What a next session must not undo

- **There is no provisioning subsystem, and that is deliberate.** PocketClaw
  targets SDK 36; an app targeting API 29+ cannot `execve()` a file in its own
  writable data directory, and `File.setExecutable(true)` does not change it
  because the restriction is enforced on the app's SELinux domain rather than by
  the file mode. Do not add a download-verify-activate pipeline for executables:
  it could not run, and its absence is what makes "no arbitrary binary
  installation" structural instead of a policy. Any future provisioning
  abstraction is limited to non-executable assets.
- **Bundled payloads must be named `lib*.so`** and must appear in both
  `requiredArm64NativeLibraries` and `keepDebugSymbols` in
  `android/app/build.gradle.kts`. The package manager only unpacks
  `lib/<abi>/*.so` into `nativeLibraryDir`; any other name ships a payload that
  can never run. Without `keepDebugSymbols`, Gradle strips the executable during
  packaging, changing its bytes and breaking the catalog's pinned SHA-256, which
  reaches the device as a `checksum_mismatch` that looks like a corrupt install.
- **System tools carry no hash, on purpose.** The OS owns `/system/bin` and
  replaces it on every update, so `security_class: system` reports
  `platform_owned` and the catalog format refuses to record a checksum. Do not
  "improve" this by pinning one.
- **Managed execution never uses `sh -c`.** It is deliberately narrower than the
  existing `exec` tool, which does. Do not relax one to match the other.
- **Flag-directed redaction is scoped per tool.** Applying curl's flag list
  globally would blank the filename in `sort -u notes.txt` and the pattern in
  `grep -E '<expr>' file`. Do not make it global.

### Where things are

- `core/src/pkg/pcruntime` — manifest and catalog, resolver, execution API,
  lifecycle events, redaction, inventory, execution probe, storage policy.
- `core/src/pkg/pcruntime/manifest.json` — the catalog. Adding a bundled tool
  means updating its SHA-256 here; `TestBundledPayloadsMatchTheirPinnedChecksums`
  fails while the catalog and the packaged payload disagree.
- `core/src/pkg/tools/runtime_tool.go` — the `runtime` agent tool, wired in
  `pkg/agent/instance.go` and gated by `cfg.Tools.Runtime`.
- `runtime/build-jq-android-arm64.sh` — the jq payload build.
- `PicoClawService.buildEnvironment()` exports `POCKETCLAW_RUNTIME_LIB_DIR` and
  `POCKETCLAW_RUNTIME_DIR`.

### Known gap for physical validation

`core/src/workspace/AGENT.md` is seeded only when the file is **missing**
(`copyMissingEmbeddedToTarget`). A device with an existing workspace keeps its old
`AGENT.md` and will not see the new runtime guidance. The same applies to the
GitHub Skill. This is why the essential rules are also in the `runtime` tool's
own description, which is always in the prompt when the tool is registered — but
a tester checking the workspace file should use a fresh workspace.

### Next

Lean Runtime Pack v2 on `feature/lean-runtime-pack-v2`: Git and GitHub CLI
first, then an HTTP/TLS capability, ripgrep, sqlite3 and yq. See `TASKS.md` for
the deferred release-engineering items, including the high-priority
`extractBinaryFromApk()` dead path.

## Shipped — v0.2.0-rc2, Auto-Start and Gateway PID ownership (2026-08-30)

`v0.2.0-rc2` is frozen, merged to `develop`, tagged, and published as a GitHub
pre-release. Both commits PASSED physical validation on a real ARM64 device on
2026-08-30, with reference APK SHA-256
`182b85183156428aa93baf3113492484258a3a1eace95f7bccd9ed82035177a3`.

- `75ac0d9` — Auto-Start safe rebuild: Service and Gateway auto-start, manual
  stop and start, no resurrection, internal chat, Telegram, Skills 8/8, Tools
  17, Core bound only to `127.0.0.1:18790` / `[::1]:18790`.
- `90194c8` — Gateway PID ownership fix: the false positive did not recur.

`feature/autostart-foundation` was NOT merged and must not be. The shipped work
was rebuilt from `v0.2.0-rc1`; see `DECISIONS.md`.

The next milestone is Managed Runtime Foundation, per `TASKS.md`.

### Gateway PID ownership on Android

`validateGatewayPidData` proved ownership by running `ps -o command= -p <pid>`
and looking for the bare `gateway` subcommand from the launch line
`<binary> gateway -E --no-color`. On Android the Gateway is executed as
`.../lib/arm64/libpicoclaw.so` and `ps` does not report that argv, so a live,
launcher-spawned Gateway was classified foreign and its pid file deleted.

The log message "pid belongs to another process" is reachable only when `ps`
succeeded with non-empty output lacking the token — that is how the cause was
established rather than guessed.

Startup and status disagreed because the readiness goroutine in
`startGatewayLocked` accepts the pid file on `pd.PID == pid` alone and never
calls `sanitizeGatewayPidData`. Only status, the realtime proxy, and manual
start ran the heuristic.

Two rules now hold, and should not be undone:

- `exec.Cmd` ownership outranks `ps`. If this launcher spawned the exact pid and
  the process is alive, it is owned, full stop. Pid reuse cannot defeat this:
  the child stays a zombie holding its pid until `cmd.Wait()` returns, and the
  monitor goroutine clears `gateway.cmd` at that moment.
- A negative command-line match is decisive only when `ps` actually returned a
  command line. A bare executable name is un-inspected, not foreign, and falls
  through to the health probe. Do not "restore" bare-name rejection.

Dead-process cleanup is independent of all of this — it lives in
`ppid.ReadPidFileWithCheck` and runs first — so relaxing the heuristic cannot
leak stale pid files.

`feature/autostart-foundation` is abandoned. Do not base work on it, merge it,
or cherry-pick from it. It is kept intact for reference only. Its automated
tests were green, but on a device a manual Start Service press could land in a
FAILED state that RC1 cannot even represent — RC1's `ServiceStatus` is
`{ stopped, running, starting }`, with no `failed` member.

The three mechanisms behind that regression, so it is not reintroduced:

1. It changed native `PicoClawService.start()` to return a `Boolean` and mapped
   `false` to `ServiceStatus.failed` in Flutter. `false` also meant "skipped",
   returned whenever the process-lifetime `isStarting` / `isStopping` /
   `manualStopActive` companion flags were set. Those flags outlive the Service
   object and had paths that never cleared them, so one stale flag made every
   later manual Start fail.
2. It set `hasFailed = true` after every child-process exit, including a
   requested stop, from a thread outside the lock that had just cleared it. A
   clean manual Stop therefore left the service marked failed.
3. `inspectServiceState()` latched on FAILED and reported it even when the
   native side said the service was cleanly stopped.

Design rules this rebuild holds to, and the reasons:

- Auto-start evaluates exactly once, from `main()`, at a true app-process
  launch. There is no resume hook. A manual Stop stays stopped until the user
  starts it again or relaunches, and that is guaranteed structurally rather
  than by a flag.
- `LaunchAutoStartPreferences` (Android) is the only store: the existing
  `picoclaw_prefs` file under new keys, synchronous `commit()`, acknowledged
  only from a post-commit readback. RC1's `auto_start` boot-receiver key is a
  different preference and is untouched.
- Gateway auto-start is a preference, not a lifecycle. The Service exports
  `POCKETCLAW_GATEWAY_AUTOSTART` and Core's existing `TryAutoStartGateway()`
  is gated on it. Do not add a Flutter gateway lifecycle, an Android gateway
  bridge, or a second definition of gateway readiness — the experimental
  branch's `gatewayRuntimeReady()` also introduced a new HTTP 400
  `precondition_failed` failure mode on the RC1 web console's own Gateway
  button.
- `START_NOT_STICKY`, and a null restart intent stops the Service. RC1's
  `runWebService` crash restart is intentionally unchanged; it is guarded by
  `stopped` and never fires on a manual stop.

## Previous objective

**v0.2.0-rc1 is frozen, merged to `develop`, tagged, and published as a GitHub
pre-release. No further RC work is pending.** The next objective is the
deferred roadmap in `TASKS.md`, not another release pass.

The DEBUG log-cleanup physical gate that previously blocked release is CLEARED.
The user physically validated
`5760247a17ccff68d188180875f1812c150dd7ae6de45e3f109d7ba07c2186b9`
on a real ARM64 Android device on 2026-08-29, covering startup, Skills 8/8,
Tools 17, Telegram owner-only authorization and final delivery, Web/realtime
owner authorization, dashboard password/session auth, Core staying loopback-only
on 18790, Public Mode OFF/ON with live OFF→ON→OFF→ON rebinding and no manual
service restart, a real `192.168.x.x:18800` LAN URL, authenticated LAN Dashboard
access from a computer, QR/connect refresh, Telegram continuity across the mode
change, Web Logs stability, UTF-8/Arabic/emoji/µs rendering, and every earlier
privacy expectation.

What a later session most needs to know:

- The Go sweep has no accepted failing region any more. Build and test with
  `-tags goolm,stdjson` from `core/src`; the pure-Go Olm implementation removes
  the `olm/olm.h` host dependency entirely. Treat any Go failure as real.
- `go test -race ./web/backend/api` fails in
  `TestStartGatewayLocked_UsesReloadedConfigForBootSignature`. The test's
  cleanup and the production monitor goroutine both call `cmd.Wait()` on the
  same `exec.Cmd`. It is a test-harness defect, it reproduces identically on
  `develop` at `8f861bc`, and the same Kill+Wait pattern appears in several
  other tests in that file. Fixing it is deferred and tracked in `TASKS.md`.
- Core is not byte-reproducible by default because `BuildTime` is stamped via
  `-ldflags`; two builds of identical source differ in exactly 64 bytes at the
  same length. To compare a binary against a reference, re-run
  `make build-android-arm64 BUILD_TIME_RAW='<the reference stamp>'` and the
  hashes match exactly. That is how this RC's native payload was proven to be
  the physically validated payload.
- Core on 18790 is loopback-only structurally, not by configuration.
  `gatewayHostOverride()` returns `localhost` unconditionally and is exported to
  the gateway child as `PICOCLAW_GATEWAY_HOST`, which outranks the config file.
  Do not "restore" config-driven gateway host binding without a security review.
- Dashboard sessions are in-memory, so restarting Core signs everyone out. That
  is intended, not a bug.

Install:

    build/app/outputs/flutter-apk/app-release.apk
    com.lord1egypt.pocketclaw 0.2.0 (4) · arm64-v8a

All of the following were confirmed on the device in this pass and need no
re-verification unless a concrete regression appears:

1. A DEBUG log export containing HTTP durations carries valid `53.616µs`, with
   no U+FFFD introduced by PocketClaw.
2. Arabic, emoji, Unicode punctuation, and ANSI/control removal are correct in
   both Logs and Export.
3. Sustained successful `GET /api/gateway/logs` and `GET /api/gateway/status`
   polling does not consume user-visible history, while failed polls and
   ordinary requests such as config/models stay visible.
4. Basename callers, user-facing debranding, and exactly-once queue/drain
   behavior are correct.
5. Telegram, skills, background/locked operation, providers, Skill Hub,
   startup, and performance all pass.

The Core binaries in this build are the exact ones validated in that run;
rebuilding this source with their timestamps pinned reproduces them byte for
byte.

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

## AGENT DEBUG PRIVACY + TELEGRAM PAYLOAD CANDIDATE — 2026-08-28

Continue on `fix/user-facing-log-privacy`. Physical evidence proved the actual
fresh default model prompt was legacy-branded and that normal Web DEBUG exposed
Agent prompt/message/tool/reasoning/session data plus complete non-empty Telego
result payloads. The default prompt now identifies as PocketClaw; custom prompt
overlays and all internal compatibility values remain unchanged.

Explicit prompt/full-request/raw-reasoning and tool-argument log paths are
removed. A pre-writer exact-field copy redacts internal/session identities and
raw-content fields without mutating runtime maps. Telego request/response data
is normalized before stdout/backend storage to operation/status/count/type
metadata; raw Telegram IDs, profiles and message bodies no longer enter normal
history. Backend, React and Dart guards cover legacy/raw input.

New APK: `46ca983a...2b908e` (34,251,141 bytes). Core hashes:
`8ed15601...9be24a` and `21001004...5fc1d7`. Tagged relevant Go suites,
frontend 46/46/tsc/lint, Flutter analyze/101, 124-file provenance, zero paths,
live endpoint, packaged hashes, and permanent guard pass. Physical device is
the final gate; no merge/release/main change.

## v0.2.0-rc1 freeze record — 2026-08-29

The candidate is merged to `develop`, tagged `v0.2.0-rc1`, and published as a
GitHub **pre-release**. `main` was not touched and `phase2-milestone-d` was not
moved. Nothing was published to Google Play.

Explicitly deferred and NOT in this candidate: Flutter/Dart obfuscation,
R8/ProGuard, symbol stripping, anti-reverse-engineering protection, and
generated Dart source URI cleanup. Those belong to the final production release
and are tracked in `TASKS.md` alongside the Statistics/Runtime page, background
and battery settings, service/gateway auto-start, the managed tool runtime, and
Android Keystore migration for provider secrets.

The candidate is validated on one ARM64 Android device. Do not describe it as
universally compatible.
