# PocketClaw Project State

## Python Lite — Phase B PHYSICAL PASS and merged, 2026-08-31

Branch `feature/python-lite-runtime`, commits `bfe47e6`, `c376837`, `c60b15f`,
`bfc2074`. **Physically validated inside the installed application** on
SM-A165F / Android 16 / API 36, then merged to `develop`. Not released, `main`
untouched, no tags moved. Phase A was a physical PASS in its own right.

Observed in the real app: catalog **2.1.0**, **56** tools, 53 available, `python`
present. `runtime {tool: python, args: ["--version"]}` returned
**Python 3.14.7**, and a script through the Managed Runtime produced
`PYTHON-PASS`, `SUM=5`, `مرحبا 🐍`, `SQLITE-PASS`. No shell was required.

| | |
|---|---|
| CPython | 3.14.7, NDK 28.2.13676358, API 24, arm64-v8a |
| Payload | `libpocketclaw-python.so`, 11,509,517 bytes, `a302c990…ad1b` |
| Core | `95a9b33b…`, embeds catalog 2.1.0 |
| Provenance | bzip2 1.0.8, XZ 5.4.7, SQLite 3.50.4 — all built from pinned source |

Static extension modules, `lib-dynload` empty, standard library appended to the
ELF as a `.pyc` zip. No pip, no ctypes, no direct Python sockets, no writable
executable storage.

**Python is not a sandbox.** The boundary is the Android app UID; `subprocess`
remains a Runtime-observability bypass and that guidance is advisory, not
enforcement. Shell availability is version-dependent: Android 11+ ships
`/bin/sh`, API 24-29 does not.

Two packaging defects were found and are permanently guarded: Gradle stripping
the appended stdlib (`keepDebugSymbols` plus an EOCD check in the build guard),
and a stale Core shipping beside a new payload
(`TestStagedCoreEmbedsTheCurrentCatalog`). **Core must be rebuilt whenever the
embedded Runtime catalog changes.**

Phase C — the Agent-facing Python tool — is open on
`feature/python-lite-agent-tool` and not started.

## Python Lite — Phase A COMPLETE (build + measurement), 2026-08-31

Branch `feature/python-lite-runtime`. Architecture review approved for Phase A
only. **Phase A is host-side build and measurement. Nothing is integrated:** no
catalog entry, no tool count change, no `python_tool.go`, no production APK
payload, no merge.

CPython **3.14.7** cross-built for `aarch64-linux-android` API 24 on PocketClaw's
own NDK **28.2.13676358**, using upstream `Android/android.py` with a single
patched line (the NDK version). Extension modules linked statically, so the
interpreter is one self-contained PIE ELF and `lib-dynload` is empty.

Measured, not estimated:

| | bytes |
|---|---|
| Interpreter, stripped, LTO | 9,123,056 |
| Payload (interpreter + `.pyc` stdlib) | 11,591,387 |
| APK increase, measured against the shipped APK | +5,814,942 |
| Projected APK | 64,147,589 |

Both Phase A gates pass: APK increase 5.55 MiB (limit 7.5 MB), installed
11.06 MiB (limit 13.0 MB).

The stdlib is appended to the ELF as a zip and imported by `zipimport` with
`PYTHONHOME`/`PYTHONPATH` set; verified functionally on the host. `.pyc` is
kept over `.py` despite costing 943,381 bytes because it starts ~4.5x faster.

`hashlib`, `hmac` and `secrets` work with no OpenSSL. `socket`, `ssl`, `ctypes`,
`multiprocessing`, `email` and `http` are absent by construction.

**PHYSICAL VALIDATION PASSED (2026-08-31)** on Samsung SM-A165F, Android 16,
API 36, arm64-v8a: 52 checks passed, 0 failed. Standalone CPython runs from
`nativeLibraryDir` under the app uid, the appended-zip stdlib imports on
hardware, `sqlite3` 3.50.4 works with FTS5 and JSON1, `hashlib` works with no
OpenSSL, Arabic and emoji round-trip, a runaway loop dies in 25 ms with no
orphan. Startup: bare 90 ms median, typical imports 121 ms median. RSS 11.2 MB
bare, 20.8 MB for a 20k-object JSON workload.

Correction carried out of the run: **Android 11+ does have `/bin/sh`** (a
symlink `/bin` -> `/system/bin`, mksh), so `subprocess(shell=True)` and
`os.system()` work on API 30+ and the architecture review was wrong to call them
unusable. PocketClaw's minSdk is 24, so shell availability is conditional on the
device. This sharpens the existing "subprocess is a bypass, guidance is
advisory" conclusion rather than changing it.

Second gap: upstream's Android tooling downloads prebuilt dependency binaries
with no checksum verification. The Phase A build script pins them by SHA-256,
but bzip2 and xz remain third-party binaries. Phase B must build them from
pinned source, as SQLite already is.

Full record: `runtime/PYTHON_LITE_PHASE_A.md`.

## Next milestone — Python Lite Runtime

Branch `feature/python-lite-runtime`, from `develop` at `b46921e`.
**Not started. Architecture review only — nothing to be compiled or bundled
until that review is approved.**

The appeal is capability per megabyte: one interpreter buys scripting, parsing,
JSON, CSV, XML, regex, calculation, file transformation, SQLite scripting,
archives and automation logic.

The constraint is that PocketClaw must not become a Linux distribution. No
PRoot, no apt, no compiler toolchain, no Node or npm, no arbitrary executable
downloads, no pip by default, no native wheel compilation, no shell emulation.
v1 targets an interpreter plus a selected standard library under
PocketClaw-controlled execution.

Python must respect what Managed Runtime already proved on hardware:
**executables ship in the APK and run from `nativeLibraryDir`; writable
executable storage is not used.** Writable Python data may live app-private, and
stdlib resources may ship as non-executable assets.

The APK is currently ~55.6 MB. Exact size projections are required before any
inclusion, and a 100+ MB addition needs explicit approval. Full scope in
`TASKS.md`.

## Provider Resilience & Automatic Failover — PHYSICAL PASS

Branch `feature/provider-resilience-failover`, from `develop` at `0a0b3fa`.
**Physical validation PASSED** on the target ARM64 device on 2026-08-30 and
merged to `develop`. Not released; `main` untouched and no tags moved.

Implementation commits: `68443c1` (resilience), `7b67493` (fallback UI and
automatic gateway restart), `ebf49b4` (restart safety invariants), `812a003`
(resume white-screen recovery), `3446b0f` (diagnostic naming).

### Physical results

| Check | Result |
|---|---|
| Provider automatic failover | **PASS** |
| Fallback Models UI, ordered selection | **PASS** |
| Primary unavailable → configured fallback answered | **PASS** |
| Single user-visible final answer | **PASS** |
| Automatic gateway restart after model config | **PASS** |
| Automatic gateway restart after fallback config | **PASS** |
| Active-turn safety | **PASS** |
| White-screen resume recovery | **PASS** |
| No recovery loop; healthy pages untouched | **PASS** |

The observed restart sequence was: active request running → model configuration
saved → UI entered "Restarting Gateway" → the request finished → gateway
restarted → configuration became active. **The active request was not
interrupted.**

### What was built, and what deliberately was not

The existing `FallbackChain`, `CooldownTracker` and `ClassifyError` were reused
rather than rewritten. There is **no checkpoint subsystem, no semantic tool
fingerprinting and no side-effect classification framework**: the agent loop
already guaranteed that a provider retry does not rewind completed tool
execution, so that property is protected by tests plus one exact-`toolCallID`
result-reuse guard.

Delivered: single-candidate cooldown, `Retry-After` support, hard-quota
distinction from transient throttling, 502/503/504 classified by what each
actually means, a conservative fallback capability gate, streaming failover only
before first visible output, steering and cancellation preserved across retries,
a `provider.*` event family with emitter-level redaction, the Fallback Models UI,
and automatic safe gateway config apply.

**Two restart invariants hold absolutely.** A busy gateway is never force
restarted when the two-minute wait expires, and an unverified busy state is never
treated as idle. Both leave the configuration saved and unapplied, and the manual
Restart Gateway control remains available.

### Counts

Tools **18**. Skills **7/7** on an existing upgraded workspace, **6/6** on a
fresh install. The removed GitHub Skill stays removed, and `picoclaw-agent` stays
unseeded — the count is not a target.

## Lean Runtime Pack v2 — PHYSICAL PASS

Branch `feature/lean-runtime-pack-v2`, fix commit `7ebd254`, from `develop` at
`fa27ad2`. **Physical validation PASSED** on a real ARM64 device on 2026-08-30
and merged to `develop`. Not released. `main` untouched; `v0.2.0-rc1`,
`v0.2.0-rc2` and `phase2-milestone-d` not moved.

### Physical results

| Check | Result |
|---|---|
| Git HTTPS | **PASS** |
| `git clone` of a public GitHub repository | **PASS** |
| **Git helper symlink execution on Android** | **PASS** |
| `git --version` | 2.51.0 |
| `gh` | 2.82.1 |
| curl HTTPS | PASS |
| ripgrep | PASS |
| sqlite3 | PASS |

### Skills

- Existing upgraded workspace: **7/7**
- Fresh install: **6/6**

The two differ legitimately. Seeding only ever writes and never deletes, so a
device that already had the GitHub Skill keeps its seven; a fresh workspace gets
the six seeded skills, because `picoclaw-agent` is deliberately unseeded.

### Known limitation

git is built with its default compiled-in `SHELL_PATH` of `/bin/sh`, which
Android does not have. **Git features that depend on a shell — hooks in
particular, and git's `ENOEXEC` fallback — are not guaranteed on Android.**
Nothing on the clone, fetch or push path needs a shell: the transport helpers
are ELF executables. See `DECISIONS.md` for why overriding it is not possible
without breaking git's own cross-build.

Six bundled tools now ship. Full architecture in `RUNTIME.md`.

| Tool | Version | Installed | License |
|---|---|---:|---|
| git + git-remote-http | 2.51.0 | 6.21 MB | GPL-2.0-only |
| gh | 2.82.1 | 55.9 MB | MIT |
| curl (mbedTLS) | 8.11.1 | 1.30 MB | curl + Apache-2.0 |
| ripgrep | 14.1.1 | 4.27 MB | MIT / Unlicense |
| sqlite3 | 3.50.4 | 1.23 MB | public domain |
| jq (from v1) | 1.7.1 | 0.77 MB | MIT |

APK 34,727,724 -> 58,302,215 bytes. Catalog 44 -> 55 tools.

### Two things a reader should know

**git found its helpers only after it could find itself.** The v2 physical run
failed `git clone` with `unable to find remote helper for 'https'`. The cause was
not TLS or symlinks: git spawns `git remote-https` and resolves the literal name
`git` through PATH, and the helper directory did not contain it. Fixed by
declaring `git` as one of its own helpers; the payloads are unchanged.

**git's transport helper is presented by symlink.** Android cannot package a
file named `git-remote-https`, so the runtime builds a directory of symlinks to
the packaged payloads and points `GIT_EXEC_PATH` at it. Nothing is written into
app storage and executed. This was the **one platform assumption v2 rested on**,
and the device has now settled it: symlink execution works, confirmed end to end
by a real clone. The probe continues to report `symlink_exec` so a device that
behaves differently says so in its own Debug Logs.

**gh's 55.9 MB is a sanctioned exception, not a precedent.** The user accepted it
explicitly because GitHub capability is core to the agent. The size policy still
binds everything else: yq was measured at 11.25 MB and left out on that basis,
since jq already covers JSON.

### Verification

Automated: `go vet` and `go test` green across Core, `flutter analyze` clean,
Flutter tests passing, all seven bundled payloads extracted from the built
release APK hashing to their catalog pins, and the arm64 guard listing every one.

Physical: **PASS**, as recorded above.

## Managed Runtime Foundation — PHYSICAL PASS

Branch `feature/managed-runtime-foundation`, commit `ee236da`, based on
`v0.2.0-rc2` / `404ef44`. Not released. `main` untouched. `v0.2.0-rc1`,
`v0.2.0-rc2` and `phase2-milestone-d` not moved.

**Physical validation: PASS** on a real ARM64 Android device, 2026-08-30.

That run covered the runtime tool registering; jq 1.7.1 executing and processing
JSON; `sha256sum`, `grep`, `sed`, `tar`, `uname`, `df` and `ping` executing;
stderr captured; a non-zero exit code preserved; a timeout terminating a harmless
long-running command; runtime lifecycle events appearing in the logs; no secret
leakage observed; the runtime driven end to end through Telegram; Service and
Gateway Auto-Start still working; and no recurrence of the Gateway PID ownership
false positive.

### Physical runtime catalog

- **43 of 44** catalog tools available on the tested device.
- Unavailable: `traceroute`, correctly reported as such rather than assumed
  present. This is the resolver doing its job: availability is measured per
  device, never read from the catalog.
- Bundled: jq 1.7.1 (`libpocketclaw-jq.so`), resolved and executed from
  `nativeLibraryDir`.

### Writable-app-data probe: INCONCLUSIVE on the tested device

The probe could neither execute its staged copy nor observe a clean permission
refusal, so it reported `inconclusive` with its reason, which is the honest
outcome rather than a guess in either direction.

This changes nothing. **The architecture does not depend on writable executable
app storage.** Executable delivery remains exactly two routes, both read-only to
the app:

1. Android system executables in `/system/bin`.
2. APK payloads the package manager unpacks into `nativeLibraryDir`.

An inconclusive probe is therefore information, not a blocker: the bundled jq
payload executed from `nativeLibraryDir` on the same device, which is the path
the runtime actually uses.

### Counts

- Tools: **18**
- Skills: **7/7**

Skills 7/7 is the current expected value and **not a regression**. The
incomplete GitHub Skill was intentionally removed by the user. It must not be
restored and 8/8 must not be treated as the target.

The PocketClaw Managed Runtime gives the Agent a controlled, observable, verified
local tool environment: `core/src/pkg/pcruntime` plus a `runtime` agent tool.
Architecture, storage layout, observability contract and the tool catalog are
documented in `RUNTIME.md`.

### The constraint this milestone established

PocketClaw targets Android SDK 36, and an app targeting API 29+ cannot execute a
file in its own writable storage — `setExecutable` does not change that. The
earlier plan for an app-private `runtime/bin` is invalid and has been replaced.
Executables reach the device only through `/system/bin` or through APK payloads
the installer unpacks into `nativeLibraryDir`, so the runtime contains no
download-and-execute path at all. See `DECISIONS.md`.

### Runtime Pack v1

- Tier 1 catalogued as system-provided and probed per device; Android already
  ships toybox, so BusyBox is deliberately not bundled.
- jq 1.7.1 bundled as `libpocketclaw-jq.so`, cross-built from the pinned official
  release tarball, proving the APK/`nativeLibraryDir` packaging contract.
- `curl`, `wget`, `openssl`, `git` and `gh` are not shipped; `TASKS.md` records
  why each is hard.

### Agent tool count

17 -> 18. All 17 existing tools are unchanged; the addition is `runtime`.

### Verification

Automated: `go vet` and `go test` green across Core, `flutter analyze` clean,
Flutter tests passing, and the jq payload extracted from the built release APK
hashing to its catalog pin, proving Gradle packaging leaves it byte-identical.

Physical: PASS, as recorded above.

## v0.2.0-rc2 — Auto-Start and Gateway PID ownership (PHYSICAL PASS)

Release candidate 2, published as a GitHub pre-release. Not a production
release and not published to Google Play.

Physical validation: **PASS** on a real ARM64 Android device, 2026-08-30.
Physical reference APK SHA-256:
`182b85183156428aa93baf3113492484258a3a1eace95f7bccd9ed82035177a3`

That run covered fresh install, Service Auto-Start, Gateway Auto-Start, manual
Service stop and start, Gateway starting automatically after the Service, no
immediate Service resurrection, internal PocketClaw chat, Core startup, Skills
8/8, Tools 17, Core bound only to `127.0.0.1:18790` / `[::1]:18790`, a working
Dashboard, and no recurrence of the Gateway PID ownership false positive.

- Auto-Start Safe Rebuild (`75ac0d9`): **PHYSICAL PASS**
- Gateway PID ownership fix (`90194c8`): **PHYSICAL PASS**
- `feature/autostart-foundation`: **NOT MERGED**, reference only. The shipped
  work was rebuilt from `v0.2.0-rc1` rather than salvaged from that branch.

Project: PocketClaw  
Current Phase: Phase 2 — Independent Product Repository  
Current Milestone: Phase 2 Milestone D — Telegram Managed-Bot Onboarding.
**COMPLETE.** Physically verified end to end on a real Android device on
2026-08-26, including a live Telegram → PocketClaw → AI provider → Telegram
message round trip against the production onboarding service.
Milestone C — Provider Catalog + Easy API-Key Setup, the OpenCode completion,
and the self-contained source migration — is complete and PASSED
physical-device testing on 2026-08-25, merged to `develop` as `36bc88d` and
tagged `phase2-milestone-c`.
Git Branch: `fix/user-facing-log-privacy`. `feature/telegram-managed-onboarding`
was merged with a non-fast-forward merge and is retained intact.
Last verified milestone: Phase 2 Milestone D (tag `phase2-milestone-d`).
Milestone C (merge `36bc88d`, tag `phase2-milestone-c`) is the fallback
reference state, with Milestone B (merge `225be3c`, tag `phase2-milestone-b`)
retained below it.
Recovery Branch: `recovery/pocketclaw-clean-debrand` @ `f25d38e`, retained intact
Foundation Bootstrap Commit: `950d4a3`  
Origin: `https://github.com/Lord1Egypt/PocketClaw.git` (private)  
Upstream FUI Baseline: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`  
PicoClaw Core: `v0.3.1`, source `2cf030d2fd3b871d7ec17e3be34c24688aac76da`,
vendored into this repository at `core/src/` and built from there — see
`core/README.md` and `UPSTREAM_BASELINE.md`  
Source-of-Truth: this repository. A clone contains all application and runtime
source; no external checkout is a build dependency. Proven by
`core/verify-no-external-source.sh`.  
Build Status: arm64 release APK built through the canonical Gradle path; the
release guard verified the arm64 native payload.
APK Status: the Milestone D APK
`b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`
PASSED physical-device testing on 2026-08-26 and remains the verified
milestone reference artifact. The broader pre-release candidate
`f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
then physically confirmed the neutral native Telegram shortcut, basename-only
callers, user-facing log debranding, terminal-control removal, and exactly-once
queue/drain behavior. A DEBUG export exposed two smaller defects; their
replacement candidate
`eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8`
is BUILT, AUTOMATED PASS, and NOT yet physically verified.
Verified Core binaries (device-verified 2026-08-26, in the reference APK):
`libpicoclaw.so` 37,224,801
`33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a`;
`libpicoclaw-web.so` 24,641,889
`5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd`.
The DEBUG-cleanup candidate carries a NEW, not-yet-verified pair:
`libpicoclaw.so`
`5c09eb72a1faa6dd6f8a0e6b66bcbc028f070d3eff9b8c5b5f87716c04d763bc`;
`libpicoclaw-web.so`
`cb6b10cc2a951d2307959e9effbdbd052762fe827b9943c22a9cdcdd54b03b52`.
Current Blocker: physical DEBUG-export validation and the remaining broader
pre-release device sweep. The GitHub release is **on hold** until both pass.
Next Exact Action: install `eacbbc86...423f9a8`, export a new DEBUG log, and
confirm valid `53.616µs` plus no successful `/api/gateway/logs` or
`/api/gateway/status` poll noise. `main` remains deliberately at `100a51d`.

## Completed

- Phase 1 baseline APK and all physical-device checks.
- Created a separate PocketClaw directory without upstream Git history.
- Selectively adapted the Android/Flutter foundation, tests, tools, and Core
  packaging from the reviewed FUI baseline.
- Preserved the Android DNS and optional-feedback behaviors.
- Recorded provenance, upstream tracking, and third-party notices.
- Audited and recorded the FUI/Core MIT notices and component classification.
- Ran `flutter analyze` (clean), `flutter test` (28 passed), and focused Core
  Android DNS/model API tests (passed).
- Built and inspected the independent arm64 foundation APK. Its embedded Core
  hashes match the pinned replacement binaries; Firebase build values are absent.
- Completed a Git diff/stat/check review and credential scan. No credentials,
  signing files, Firebase config, generated APKs, or caches are eligible for
  commit; the standard Gradle Wrapper files are intentionally retained.
- Created the private GitHub repository `Lord1Egypt/PocketClaw`, pushed
  `main`, and created/pushed the `develop` integration branch.
- Physical-device verification confirmed the independent foundation APK,
  launch, Core/Gateway, active-network DNS, model discovery/manual model,
  AI requests, Telegram, restart/persistence, and optional feedback behavior.
- Physical-device verification confirmed ClawHub Skill Hub search: a `Crypto`
  query returned 20 results with metadata, URLs, and install actions. The
  prior registry-unavailable observation is classified as resolved by the
  Android DNS fix, not as an independent Skill Hub defect.
- Started Milestone B on `feature/pocketclaw-identity`: product naming,
  independent package identity, original visual direction, Android icon/splash
  treatments, design tokens, and the required audit records are complete.
- Passed Milestone B `flutter analyze`, all 28 Flutter tests, and focused
  pinned-Core Android DNS/model API tests. Built and inspected the new APK:
  package/label are `com.lord1egypt.pocketclaw`/PocketClaw, Android branding
  resources are bundled, the Core hashes match the pin, and no Firebase config
  values are present.
- Committed Milestone B as `34b0f6b` and pushed
  `feature/pocketclaw-identity` to the private `origin`; `develop` and `main`
  remain untouched pending the user's physical-device approval.
- Root-caused the Milestone B black-screen regression to the two branded Android
  `launch_background.xml` resources. A `layer-list` `<item android:color>` is
  not a valid drawable layer: Android must inflate a drawable-backed item before
  Flutter can replace `LaunchTheme`. The corrected resources use
  `@color/pocketclaw_splash_background` through `android:drawable`.
- Added a source-level regression test for both launch-background variants.
  `flutter analyze` is clean; all 29 Flutter tests and focused Core
  `pkg/androiddns`/`web/backend/api` regressions pass. A debug and a replacement
  arm64 release APK were built and the release's compiled layer-list was
  inspected to confirm the first item has a drawable reference.

- Milestone C: audited the provider architecture end to end and recorded it in
  `docs/PROVIDER_ARCHITECTURE.md`. Established that all AI provider
  configuration lives in the Core web console, not in Flutter, and that the
  provider catalog is already backend-owned by `pkg/providers`.
- Milestone C: extended the backend-owned catalog with `category` and
  `documentation_url`, added the xAI, Together AI, Fireworks AI, and
  Custom OpenAI-Compatible presets, and registered all four in the protocol
  switch so they actually dispatch at runtime.
- Milestone C: enabled Gemini model discovery with a dedicated fetch branch
  that uses `X-Goog-Api-Key` for the native base and Bearer for the
  OpenAI-compatible base, and broadened fetch error classification to cover
  rate limiting, provider outage, and a missing listing endpoint.
- Milestone C: replaced the Add Model form with a two-step provider-first flow
  — choose provider, paste API key, fetch or type a model, save — deriving the
  model alias automatically and moving base URL, alias, and optional keys into
  Advanced. Local and custom providers keep a visible base URL.
- Milestone C: removed runtime logo fetching from `cdn.simpleicons.org` and
  Google's favicon service; provider marks are now rendered locally.
- Milestone C: added 22 frontend tests (new vitest runner), 8 Go catalog tests,
  and 5 Go model-discovery tests. `flutter analyze` clean, 27 Flutter tests,
  Go suites for providers/config/api/androiddns/mqtt/onboard/commands/agent all
  pass, and the frontend type-checks and lints clean.

## Constraints

- Do not modify the Phase 1 workspace or its verified APK.
- Do not merge upstream repositories automatically.
- Preserve PicoClaw Core protocol/binary/environment identifiers and the
  compatible `Downloads/picoclaw` workspace path while product identity changes.
- Do not commit credentials, signing material, generated APKs, or caches.
- Do not merge `feature/provider-catalog` to `develop` before the user's
  physical-device approval, and do not touch `main`.
- Do not start Telegram QR/deep-link onboarding: it is the next milestone.
- Do not enable obfuscation or anti-reverse-engineering during active feature
  development; release hardening is a later pre-release milestone.

## Independent Foundation APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 03:06:47 +03:00
- Size: 32,619,416 bytes
- SHA-256: `207a5e4623b6c6ae295a29a874c7a7d6ac9a17552e2ddb2511daa093b085fa4d`
- Package/version: `com.sipeed.picoclaw`, `0.1.3` (version code `3`)
- Label: `PicoClaw` (intentionally unchanged for this foundation milestone)
- Architecture: functional application/Core payload is `arm64-v8a`

## Milestone B PocketClaw APK

- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-24
- Size: 34,096,308 bytes
- SHA-256: `0e440d2804978e6f94550a9d0cab563328d03cd32312bd4e33ec1ecdfc6d4883`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Architecture: universal APK; the PicoClaw Core payload is `arm64-v8a` and
  its two pinned hashes match `UPSTREAM_BASELINE.md`.

## Milestone B Runtime-Regression Replacement APK

- Status: BLOCKED — physical-device retest pending.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (ignored; not committed)
- Built: 2026-08-24 04:34:40 +03:00
- Size: 33,308,653 bytes
- SHA-256: `45be7269af920df4a36eb4eb37171770bbcfa242ed7c071da28874d9c27ebe9e`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: `PocketClaw`
- Core payload hashes: unchanged — gateway
  `3b849072a7c2858b0d2c0db5cbcfa42b542353e834f4c473399eda571ab16f3d`, web
  `252b38c64cbc4dc52277c206ca1b069cc7c3bb97b8a9c276e23f8edc3aaf95e3`.

## Stage B Debranded APK — PHYSICALLY VERIFIED (previous reference)

- Status: PASS on a physical Android device, 2026-08-25. Not merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 from `recovery/pocketclaw-clean-debrand` with
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=.tooling/gradle-stage-a-clean`, Flutter 3.47.1 / Dart 3.13.1.
- Size: 34,119,837 bytes
- SHA-256: `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- arm64 payload verified by the release guard: `libdartjni.so` (131,248),
  `libpicoclaw.so` (37,421,409), `libpicoclaw-web.so` (24,772,961).
- Embedded Core hashes: gateway
  `1f239a827c8562d6ac2ffdf63c1354ce0d28396cdab7d4f240f3866cbb525fed`, web
  `94bb6319bbac08e1aa0fa43e8093b4dd00bad512cb67ca94a6a57d666f4bc716`.
- Physical-device results (2026-08-25): install PASS, app launch PASS, Flutter
  first frame PASS, black-screen regression FIXED, Gateway/Core startup PASS,
  navigation PASS, PocketClaw branding PASS, PocketClaw workspace path PASS,
  QR/access page PASS, no abnormal device slowdown observed.
- This is the reference physically verified PocketClaw artifact. Compare any
  future build against it.

## Pre-release APK check (mandatory)

`packageRelease` now fails the build if `lib/arm64-v8a/` is missing
`libdartjni.so`, `libpicoclaw.so`, or `libpicoclaw-web.so`, and prints a
"Verified arm64-v8a native payload" line when it passes. If `libdartjni.so` is
reported missing, purge `~/.pub-cache/hosted/pub.dev/jni-*/android/.cxx/` and
rebuild — that cache is outside the project `build/` tree, so cleaning build
intermediates does not clear it.

`./gradlew :app:assembleRelease -Ptarget-platform=android-arm64` is the
canonical release path. Do not release a universal `flutter build apk --release`.

## Milestone B Final Cleanup APK — VERIFIED REFERENCE ARTIFACT

- Status: PASS on a physical Android device, 2026-08-25. Merged to `develop`.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Also copied to `build/app/outputs/flutter-apk/app-release.apk` (identical).
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`.
- Size: 34,119,477 bytes
- SHA-256: `ba4f067df9811bd0e4af713343bdba632abbf96a41e3a5b47cf154740f70a4b8`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `eb895f0892509b76242f572515c26f56530ec417bdedc0bb9ec1486f40bd9c88` |
| `libpicoclaw-web.so` | 24,772,961 | `6d282df06680869a0aca25a976b123bce8e793d2f08708e79386a1761195a5a3` |

- Contents of this cleanup: MQTT fresh default is `/pocketclaw` while any
  explicitly configured prefix (including the legacy `/picoclaw`) is preserved;
  `skills/picoclaw-agent` is no longer seeded into a fresh workspace while
  existing user copies are untouched; factual Sipeed hardware references are
  retained deliberately.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Physical-device results (2026-08-25): install PASS, app launch / first frame
  PASS, no black screen, Gateway/Core lifecycle PASS, navigation PASS,
  PocketClaw branding PASS, workspace path PASS, QR/access page PASS, Skill Hub
  PASS, provider/model flow PASS, no abnormal slowdown.
- This is the current verified reference artifact. Compare any future
  regression against it before forming new hypotheses.

## Milestone C Provider Catalog APK — PHYSICALLY VERIFIED (superseded)

- Status: PASS on a physical Android device, 2026-08-25. This is the verified
  reference artifact, superseding Milestone B's `ba4f067d...70f70a4b8`.
  Not merged to `develop`: the OpenCode completion below rides on the same
  branch and must pass its own device test first.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,123,401 bytes
- SHA-256: `b3dd892bdea86e8dfe7d1c2eb87e89f4e2832b1d1dbe39fc1decf20dabce569b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Flutter and Core
  payloads are `arm64-v8a`; the other ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `cbe568af0d6e0a1e3e4e48f7ab53fa00300509dc04f5d6ee07d0465e5556468a` |
| `libpicoclaw-web.so` | 24,772,961 | `86e53457468c6c53f6c8814b4345fcfe1ec7026e3ded388d2ab305c10cb0a4cd` |

- `libdartjni.so` is byte-identical to the Milestone B verified build.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, which is the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.
- What to test on the device: the Add Provider picker opens and lists providers
  by category; selecting a cloud provider asks only for an API key and a model;
  Fetch Models succeeds against a real provider; a failed fetch still allows a
  manually typed model ID; Custom OpenAI-Compatible accepts a base URL, key, and
  model ID; an existing provider still opens with its stored values intact; and
  startup, DNS, Core lifecycle, Telegram, Skill Hub, workspace, MQTT, and
  branding are all unregressed.

## Milestone C OpenCode Completion APK — RETIRED, NEVER TESTED

- Status: RETIRED without ever being tested. Its functionality is contained in
  the physically verified source-migration APK `588bbec1...5ff3a90b`, which
  supersedes it. Kept here only as a record of what was built.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,129,765 bytes
- SHA-256: `785ccd94cfa351ee2996ac340f9a55e828a0c8f736bec67a3edac906a56058c6`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- ABIs advertised: `arm64-v8a`, `armeabi-v7a`, `x86_64`. Only `arm64-v8a`
  carries `libapp.so`, `libflutter.so`, and the two Core payloads; the other
  ABIs carry plugin JNI libs only.
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,421,409 | `e48e8af073d6e7dfdb46ba8268785780d1f900888b82dba41747ef9212e78938` |
| `libpicoclaw-web.so` | 24,772,961 | `5faaf82ccbcd2fbad27d7ffc336f240fd5c08a48a1abbb2bd4eff7c383fe2abf` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped with 0 debug sections, and
  `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw`. `google_app_id` is present but empty, the expected
  `cleanupFirebaseResources` outcome; no Firebase credential value ships.

### What to test on the device

The rest of the app is unchanged from the verified Milestone C build, so the
regression sweep can be brief. The new surface is the two OpenCode presets:

1. OpenCode Zen appears in Add Provider, asks only for an API key, and Fetch
   Models returns the live list from `https://opencode.ai/zen/v1/models`.
2. OpenCode Go does the same against `https://opencode.ai/zen/go/v1/models`.
3. Run one real inference on each protocol family, per provider, because the
   protocol is chosen per model and only a live request proves the route:
   - a Responses-family model (`gpt-*`, `*codex*`),
   - an Anthropic Messages-family model (`claude-*`),
   - a chat-completions-family model (`kimi-*`, `deepseek-*`, `glm-*`).
4. Confirm the model ID saved and sent is the bare ID (`kimi-k3`), not the
   namespaced OpenCode CLI form.

The one assumption that only a device can settle is the Messages
authentication form — see the OpenCode routing decision in `DECISIONS.md`.
If a `claude-*` model returns 401 while `gpt-*` and `kimi-*` succeed, that is
the bearer-versus-`X-API-Key` question, not a routing failure.

## Self-Contained Source Migration APK — PHYSICALLY VERIFIED REFERENCE ARTIFACT

- Status: PHYSICALLY VERIFIED on 2026-08-25. This is the current verified
  reference artifact, superseding Milestone C's `b3dd892b...bce569b`. Bisect or
  diff any future regression against it before forming new hypotheses. It
  carries the OpenCode completion, so `785ccd94...56058c6` is retired untested.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-25 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`,
  after building Core from `core/src/` with `core/build-android-arm64.sh`.
- Size: 34,123,225 bytes
- SHA-256: `588bbec144fe0c84b8429f4f053a73b44b9b3e8d9f24e31dab04b2165ff3a90b`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Label: PocketClaw; launchable `com.lord1egypt.pocketclaw.MainActivity`
- Release guard PASS for all three required libraries:

| Packaged library | Size | SHA-256 |
| --- | --- | --- |
| `libdartjni.so` | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` |
| `libpicoclaw.so` | 37,224,801 | `cb9b2cdea1ccd7ddbbda723ddd3bed1d8c3a931638b1952dd767f62efb895818` |
| `libpicoclaw-web.so` | 24,641,889 | `b6b356f75eb348933e6cb1049890bb8be4d2bd20b605f55b23a5487656db9ba5` |

- `libdartjni.so` is byte-identical to every verified build since Milestone B.
- Both Core binaries are stripped, `ARM aarch64` PIE, and are the first built
  from repository-local source and the first built with `-trimpath`. They are
  roughly 197 KB and 131 KB smaller than the previous pair for that reason.
- Developer-machine paths in the packaged Core binaries: 0 and 0. The previous
  pair carried 2,501 and 1,346. `core/build-android-arm64.sh` fails the build
  if this regresses.
- `PICOCLAW_DNS_SERVER` is verified present in the rebuilt gateway.
- Branding invariants hold in the rebuilt launcher: 0 `PicoClaw`, 0 `Sipeed`,
  31 `PocketClaw` — identical to the verified Milestone C counts.
- Pre-build validation: Go suites 92 ok / 0 failed, frontend `vitest` 28
  passed, `pnpm lint` clean, `flutter analyze` no issues, `flutter test` 27
  passed.

### Physical-device results — 2026-08-25, all PASS

| Check | Result |
| --- | --- |
| Install / startup | PASS |
| Flutter first frame | PASS |
| Gateway/Core lifecycle | PASS |
| User-facing logs free of developer absolute paths | PASS |
| User-facing logs free of PicoClaw product branding | PASS |
| Provider catalog | PASS |
| OpenCode Zen preset | PASS |
| OpenCode Zen Fetch Models | PASS |
| OpenCode Zen real request/response | PASS |
| OpenCode Go preset | PASS |
| OpenCode Go Fetch Models | PASS |
| OpenCode Go real request/response | PASS |
| Gemini / provider regression | PASS |
| Skill Hub regression | PASS |
| Telegram regression | PASS |
| Workspace regression | PASS |
| No black screen | PASS |
| No abnormal slowdown | PASS |

Three results carry more weight than the rest.

The logs check is the first device confirmation of the `-trimpath` change. The
build-time assertion proved the strings were absent from the binaries; this
proves nothing surfaces them on the screen a user actually reads.

Skill Hub search and Fetch Models both working is the end-to-end proof that the
Android active-network DNS integration survived being rebuilt from a relocated
source tree. Those paths fail closed without working DNS, so a PASS on both
means the integration is intact — not merely present as a string in the binary.

The four OpenCode results are the first live confirmation of per-model protocol
routing. Zen and Go each returned a real response, so the routing table in
`pkg/providers/opencode_routing.go` is exercised rather than assumed.

One question stays open, and this PASS does not close it. `DECISIONS.md`
records that OpenCode's Anthropic Messages surface is sent both `X-API-Key` and
a bearer header because the correct form could not be established offline. That
is only settled by a `claude-*` model returning a real response, and the device
report does not say which model families were exercised. Treat the Messages
route as unconfirmed until a `claude-*` inference is observed; if one 401s while
`gpt-*` and `kimi-*` succeed, the header pair is the cause, not the routing.

## Post-Milestone-D device regressions — FIXED, AWAITING PHYSICAL DEVICE

Three defects found on the device after Milestone D closed, on branch
`fix/user-facing-log-privacy` (not merged, not tagged). All three are
pre-release blockers; the GitHub milestone release is on hold until the
replacement APK passes a device re-test.

### Bug 1 — the caller leaked the upstream module path

The Logs screen showed:

    WRN api github.com/sipeed/picoclaw/web/backend/api/gateway.go:298 >
    removed stale pid file for PID 12302

Root cause: `pkg/logger/logger.go` builds its zerolog logger with `.Caller()`,
which reports the path the **compiler** recorded. The earlier `-trimpath` work
did exactly what it was asked — it removed `/home/lordegypt/...` — but what
replaces an absolute path under `-trimpath` is the Go **module path**. So the
fix for one leak created another, and no check caught it because every existing
assertion looked for `/home/`.

Fixed at the structured layer, in `logger.init()`, by setting
`zerolog.CallerMarshalFunc` to `ShortCallerLocation`, which reduces any
recorded caller to `file.go:line`. This is the earliest point that sees caller
metadata, so every writer, every level, and every exported log file inherits
the short form. No message text is rewritten and no arbitrary file path inside
a message is touched.

    WRN api gateway.go:298 > removed stale pid file for PID 12302

`ShortCallerLocation` normalises `\` as well as `/`. `filepath.ToSlash` only
rewrites the *host* separator, so a Windows-recorded caller would have passed
straight through a Linux build — the test caught this.

Eight user-visible message strings that named the project were also reworded
(not blind-replaced; each was read and edited individually):
`pkg/pid/pidfile.go`, `web/backend/api/gateway.go`,
`web/backend/utils/runtime.go`, `web/backend/main.go`, and four provider
credential errors that told users to run a CLI command that does not exist on
Android.

### Bug 2 — one event rendered as hundreds

Not repeated backend emission, and not a lifecycle bug. The device screenshot
was decisive: every duplicate carried the **identical timestamp** `02:23:44`
and the identical PID, and the event counter read the full 500.

Root cause: `PicoClawService.lastLog` is a **sticky snapshot** of the most
recent line that never clears. `ServiceManager._syncNativeServiceStatus`, which
runs on a three-second timer, appended that snapshot on every tick. One warning
was therefore re-added every three seconds until it filled the 500-entry buffer
and **evicted the entire real log history** — the damage was not cosmetic.

`RemovePidFileIfPID` was audited and is correct: it returns true only after
actually reading, matching, and removing the file, so it cannot have fired
repeatedly for one PID.

Fixed by making the producer match the contract the consumer needs, rather than
by deduplicating text. `PicoClawService` now funnels every line through
`publishLog`, which appends to a bounded pending queue; `takeNewLogs` drains
it, so each line is handed out exactly once. This also fixes a second defect the
snapshot hid: lines emitted **between** polls used to be lost, because only the
most recent one was ever read.

### Bug 3 — the two surfaces disagreed about whether Telegram was connected

The embedded console correctly showed Connected with the bot handle, while the
native Settings card still read "Connect PocketClaw to Telegram" and opened a
page whose primary action was a brand-new pairing. Same class of defect as the
Milestone D UI bug: two surfaces, no shared source of truth.

Root cause: the native card's subtitle was a **hardcoded constant**
(`TelegramOnboardingStrings.introHeadline`) and its tap handler went straight
to onboarding. It never read any state. The console, by contrast, derived
`configured` from Core's own `detectConfiguredSecrets`.

Fixed by giving both surfaces one canonical source. `TelegramConnectionReader`
(`lib/src/telegram/telegram_connection_status.dart`) reads the persisted
`channel_list.telegram` entry — a non-empty `settings.token`, plus `allow_from`
for owner scoping — which is exactly the state Core reports to the console. No
"connected" boolean is stored anywhere, so nothing can drift. It fails closed:
unreadable or malformed configuration reports disconnected.

- `TelegramSettingsCard` renders from that state and re-reads after the flow
  returns and on app resume, so a change made in the console is picked up.
- `TelegramConnectedPage` is what an already-connected user now opens: bot
  handle, owner, Open Chat, Reconnect / Create New Bot, Advanced / Manual.
- `TelegramOnboardingLauncher.open` is state-aware; `startPairing` is the
  explicit pairing entry. The console's Connect and Reconnect buttons call
  `startPairing`, since both are deliberate user requests.
- Reconnect asks first, and the confirmation states plainly that the current
  bot keeps working until a new one is ready.

Replacement is already safe and was verified rather than assumed:
`TelegramConfigWriter.apply` is the only thing that mutates
`channel_list.telegram`, and the controller calls it only after the new token
has been received. A cancelled or expired pairing therefore never reaches the
writer and the existing bot stays configured.

The bot handle is not stored by Core, so a manually configured bot — or one
paired on another device — shows as Connected without a handle and hides Open
Chat, rather than inventing one.

14 tests in `test/widgets/telegram_state_sync_test.dart` cover cases A–G
against real configuration JSON and the real widgets. Four of them fail against
the old hardcoded card, which was confirmed before they were accepted.

### Tests — confirmed to fail against the old code

- `core/src/pkg/logger/caller_sanitize_test.go` — the exact device caller plus
  absolute, `.upstream`, Windows, already-short and degenerate inputs, and an
  end-to-end case that logs through the real logger into a real file and
  asserts the emitted `caller` field carries no module path and no separator.
- `test/unit/service_manager_log_stream_test.dart` — drives the real polling
  path. Against the old code 25 polls produced **25 copies** of one event; it
  now produces one. Also asserts a genuinely repeated line is *not* collapsed,
  that bursts between polls all arrive, and that no rendered line carries a
  module or developer path.

`ServiceManager` is a singleton, so these tests assert on the lines each test
appended rather than on the whole list.

### Verification

`flutter analyze` clean, 93/93 Flutter tests, 36/36 frontend, `tsc -b` clean,
`pnpm lint` clean, and Go tests green for `pkg/logger`, `pkg/pid`,
`pkg/providers/...` and `web/backend/...`.

Core rebuilt (`-trimpath` verified, zero developer paths in both binaries) and
`core/pocketclaw-core-v0.3.1.patch` regenerated — now 67 files.

Candidate APK `543c759b04b0e4c77dd7831435753aceac0b1e16a7a45fdec3fb37ed2e45479a`,
34,227,525 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (3), guard PASS. Secret scan
clean: 0 tokens, 0 webhook/pairing secrets, 0 Redis credentials. The onboarding
endpoint and the Telegram onboarding UI are both present and unchanged.

### Known, pre-existing, not a regression

`libapp.so` contains one developer path:
`file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
It is the Flutter-generated plugin registrant's source URI baked into the Dart
AOT snapshot, it is present in the **device-verified** `b6fea5d8...` APK and in
every earlier one, and it reaches a user only inside a Dart stack trace, not
the Logs screen. Go's `-trimpath` has no Dart equivalent. Recorded rather than
fixed, because it is out of this fix's scope and needs its own decision.

`Run: picoclaw auth login --provider <name>` remains in `libpicoclaw.so`, in
`cmd/picoclaw/internal/auth/helpers.go`. It is printed by the CLI `auth list`
subcommand, which the Android app never invokes — the app runs the gateway —
and on desktop the binary genuinely is named `picoclaw`, so rewording it would
make the instruction wrong. Left accurate deliberately.

The compiler's embedded source-path table still contains
`github.com/sipeed/picoclaw/...` inside the binaries. That is unavoidable Go
metadata for stack traces and is explicitly permitted; what matters is that it
no longer reaches rendered caller metadata.

## Phase 2 Milestone D — Telegram Managed-Bot Onboarding — COMPLETE

**Status: COMPLETE. Physical end-to-end test PASSED on a real Android device,
2026-08-26.**

Automatic managed-bot onboarding is now the default Telegram setup path in
PocketClaw. Manual Bot Token entry remains available as the advanced fallback.

### Verified artifacts

| | |
| --- | --- |
| Verified APK | `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94` |
| APK size | 34,220,929 bytes, `com.lord1egypt.pocketclaw` 0.1.3 (version code 3) |
| `libpicoclaw.so` | 37,224,801 bytes, `33f8b4efbc88333747c5df30b3ddc6864c924b35dba99e9c8c3b91df3470e98a` |
| `libpicoclaw-web.so` | 24,641,889 bytes, `5400cb02322ece5c7035356595355bd3c116adfc6a5bb6b78f3e2bd22dbcb3bd` |
| Official manager | `@PocketClawSetupBot` |
| Onboarding service | `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` |
| Public service repository | `Lord1Egypt/PocketClaw-Telegram-Setup` (public, MIT) |

The Core pair above was rebuilt for the UI integration fix and is now
**physically verified with this APK**, superseding the Milestone C pair.

### Live architecture

    PocketClaw Android
      → PocketClaw Telegram Setup  (Vercel)
        → @PocketClawSetupBot      (Telegram manager, Bot Management Mode)
          → Telegram Managed Bots
            → Upstash Redis        (pairing state, REST, 600 s TTL)
              → automatic PocketClaw Telegram configuration

The service repository is independent and public. PocketClaw's Android and Core
source stays in this repository; the APK builds from nothing in the service
repo and carries only its public base URL.

### Telegram UI integration — verified on device

- Channels → Telegram opens the managed onboarding flow: PASS
- The legacy Bot Token form is no longer the primary first-run flow: PASS
- QR code displayed: PASS
- Open Telegram action: PASS
- Suggested bot username uses the PocketClaw prefix: PASS
- Manual setup remains available: PASS
- Advanced Settings exposes the legacy Telegram configuration: PASS

### Managed-bot end to end — verified on device

Pairing created; QR/deep link generated; the Telegram managed-bot creation
screen opened; the bot was created. Created identity: display name
**PocketClaw Agent**, username pattern `@pocketclaw_<random>_bot`.

Returning to PocketClaw detected the pairing automatically, the child bot token
was delivered automatically with no BotFather copy/paste, the owner was
configured automatically, and the Telegram configuration saved itself. The
connected state and bot username rendered correctly, and Open Chat opened the
Telegram conversation. All PASS.

### Real message end to end — verified on device

    Telegram user → PocketClaw bot → PocketClaw Core
      → configured AI provider → AI response → Telegram

PASS. Real messages were sent to the newly created PocketClaw Agent bot and AI
replies were received. Multiple consecutive messages were tested and
conversation context persisted across them. The first response was slightly
slower, consistent with initial channel/provider startup; subsequent replies
were prompt.

### Live service state

Manager authentication PASS as `@PocketClawSetupBot`; `can_manage_bots` true;
Bot Management Mode enabled; Telegram webhook registered; shared pairing
storage PASS on Upstash Redis REST with a 600-second pairing TTL; Create Test
Pairing PASS; privacy page PASS.

### Security — confirmed

Embedded in the APK: manager bot token NO, Telegram webhook secret NO, pairing
secret NO, Redis credentials NO. Child bot token included in the QR: NO. Raw
token shown in normal UI: NO. Raw token required from the user: NO. Manager
token committed to Git: NO. Production secrets committed to Git: NO.

No secret value is recorded in this repository, and none ever should be.

### Core regression observed during physical testing

PocketClaw startup and Flutter first frame PASS; Gateway/Core lifecycle PASS;
Telegram channel lifecycle PASS; PocketClaw branding PASS; no black screen;
no abnormal runtime slowdown; user-facing log path privacy intact.

Preserved automated regression coverage, not re-tested physically in this
round: Android DNS/model discovery, Provider Catalog, Gemini and other
providers, OpenCode Zen, OpenCode Go, Skill Hub, Workspace, MQTT.

## Milestone D UI Integration Fix — the change that made it reachable

### Root cause

PocketClaw shows Telegram on two different surfaces, and Milestone D wired the
managed-bot flow to the wrong one.

The app's four tabs are Dashboard, the embedded Core console in a WebView,
Logs, and native Settings (`lib/main.dart:288-299`). Milestone D added the
onboarding entry to the **native Settings** page
(`lib/src/ui/config_page.dart`, `_buildTelegramEntry`). The **Channels** list —
where a user naturally goes to add a channel — lives inside the Core web
console, is served from `core/src/web/frontend`, and knew nothing about
onboarding.

So the flow existed, was fully tested, and was unreachable from the path users
take. Every Milestone D test passed because every one of them entered through
the Flutter widget directly; none entered through Channels.

### Actual Telegram entry component

    WebView tab (lib/main.dart:290)
      → Core web console
        → routes/channels/$name.tsx        (TanStack route /channels/$name)
          → components/channels/channel-config-page.tsx
            → channel-forms/telegram-form.tsx      ← what the device showed

`telegram-form.tsx` renders exactly the fields reported from the device: Bot
Token, API Base URL, HTTP Proxy, allow_from, Typing Indicator, Streaming
Output, Placeholder Message. "Enable channel" and "Save" come from the
`channel-config-page.tsx` wrapper around it.

### The fix — one journey, one implementation

The pairing flow stays in Dart, where it is already written and tested; the
console renders the entry point and asks the host to run it. Nothing was
reimplemented in TypeScript.

- `lib/src/ui/telegram_onboarding_launcher.dart` (new) — the single way into
  onboarding. Both the native settings list and the console now call it, so
  there is one implementation rather than one per surface. It also remembers
  the paired bot's public `@username` in SharedPreferences, because Core does
  not report the handle back and the connected summary needs it for Open Chat.
- `lib/src/ui/webview/pocketclaw_host_bridge.dart` (new) — the host contract.
  Builds the injected `window.__pocketclawHost` script and parses messages
  coming back. It carries no secret: only whether an endpoint was compiled in,
  and the bot's public handle.
- `lib/src/ui/webview/webview_android.dart` — registers the `PocketClawHost`
  JavaScript channel, injects the contract on page load, runs the native flow
  on request, then re-injects and fires `pocketclaw:telegram-updated` so the
  console re-renders as connected instead of showing a stale state.
- `core/src/web/frontend/src/components/channels/channel-forms/telegram-panel.tsx`
  (new) — the surface the Channels route now renders.
- `core/src/web/frontend/src/lib/pocketclaw-host.ts` (new) — typed access to
  the host, returning null in an ordinary browser.
- `channel-config-page.tsx` — the `telegram` branch renders `TelegramPanel`
  instead of the bare `TelegramForm`.

### The three surfaces

| Condition | Surface |
| --- | --- |
| No token, host can pair | **Managed onboarding first** — Connect PocketClaw to Telegram, Open Telegram, status, then "Having trouble? / Advanced / Manual setup" |
| No token, no host or no endpoint | **Manual setup**, shown in full with a short explanation and nothing to expand |
| Token already set | **Connected** — bot handle, owner, Open Chat, Reconnect / Create New Bot, then Advanced Settings |

The legacy form is never deleted or altered. It moved behind a disclosure, and
in the manual-only case it is still the whole page. Every field the manual path
depends on is asserted present by test.

### Security review of the new bridge

- The injected script carries no token or secret, and the bot handle is emitted
  through `jsonEncode` so it cannot break out of its string literal.
- `openExternal` accepts only absolute `http`/`https` URLs. `javascript:`,
  `file:`, `intent:`, `content:` and relative paths are dropped, so the bridge
  cannot be turned into an arbitrary-launch primitive.
- Both injection and message handling are scoped to the console's own origin
  (`PocketClawHostBridge.isSameOrigin`). The WebView will follow an outbound
  link if a user taps one, and a third-party page holding the host object could
  otherwise drive the app. Unparseable input fails closed.

### Tests

The point of these is that they fail against the old wiring. Reverting
`channel-config-page.tsx` to render `TelegramForm` was tried, and all 8 web
tests failed; restoring the panel made them pass. They enter through
`ChannelConfigPage channelName="telegram"` — the same component the APK renders
— not through an isolated onboarding widget.

- `channel-config-page.telegram.test.tsx` — 8 tests: case 1 managed onboarding
  primary and the handoff firing, case 2 fallback for both "no endpoint" and
  "no host at all", case 3 connected summary with Open Chat, case 4 the legacy
  form revealed from both Advanced entries, plus a host that injects late.
- `test/unit/pocketclaw_host_bridge_test.dart` — 11 tests over the script
  payload, origin scoping, scheme rejection, and malformed input.
- Frontend suite 36/36, Flutter 88/88, `flutter analyze` clean, `tsc -b` clean,
  `pnpm lint` clean.

`jsdom` and `@testing-library/react` are new frontend dev dependencies. They
had to be added: the whole failure was that no test rendered the real route,
and there was no DOM renderer in the project to do it with.

### The APK

- Path: `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,220,929 bytes (~34 MB arm64 band)
- SHA-256: `b6fea5d8ec5c3c66ba8a1320b0a217afcca322e75b5b26cc4082bbbb08a57f94`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (3)
- Release guard PASS, all three libraries present.
- **Core WAS rebuilt** — `core/src/web/frontend` changed, so the console binary
  had to be regenerated. New pair: `libpicoclaw.so` 37,224,801
  `33f8b4ef...3470e98a`; `libpicoclaw-web.so` 24,641,889 `5400cb02...2dbcb3bd`.
  They replace the device-verified `cb9b2cde...`/`b6b356f7...`, so the Core half
  of this APK is no longer device-proven and the regression sweep matters.
  `-trimpath` verified: zero developer paths in either binary.
  `core/pocketclaw-core-v0.3.1.patch` regenerated (58 files).
- Endpoint present once in `libapp.so`; the new console strings present in
  `libpicoclaw-web.so` and absent from the previous build's copy.
- Secret scan over all 425,247 printable strings: 0 Telegram tokens, 0
  `TELEGRAM_MANAGER_BOT_TOKEN`/`TELEGRAM_WEBHOOK_SECRET`/`PAIRING_SECRET`, 0
  `KV_REST_API_*`/`UPSTASH_*`/`REDIS_URL`, 0 `upstash`, 0 `redis://`. All 23
  64-hex strings in `libapp.so` are google_fonts asset checksums.

### What to check on the device

1. Channels → Telegram opens **Connect PocketClaw to Telegram**, not Bot Token.
2. Open Telegram launches the native pairing screen with QR and live status.
3. Completing pairing returns to a **Connected** summary without a manual
   reload, showing the new `@pocketclaw_..._bot` handle.
4. Open Chat opens Telegram outside the app.
5. Advanced / Manual setup reveals the full legacy form, and saving from it
   still works.
6. The native Settings → Telegram entry still reaches the same flow.
7. Regression: startup, DNS, provider catalog, OpenCode, Skill Hub, workspace,
   MQTT, Core lifecycle, branding — the Core binaries are new.

## Milestone D Live-Endpoint APK — DEVICE-FAILED (UI never reachable)

This is the artifact to install for the end-to-end Telegram test. It is the
first PocketClaw build that carries a real onboarding endpoint.

- Status: FAILED on a physical device 2026-08-26. Everything asserted about
  this build was true — endpoint compiled in, guard passed, no secrets — and it
  still did not work, because none of those checks covered whether a user could
  reach the flow. Superseded by `b6fea5d8...08a57f94`.
- Path: `build/app/outputs/flutter-apk/app-release.apk` (also written to
  `build/app/outputs/apk/release/app-release.apk`; both ignored, not committed)
- Built: 2026-08-26 with the canonical command plus the supported dart-define
  mechanism, which the Flutter Gradle plugin forwards to `flutter assemble` as
  `--DartDefines`:

      cd android && ./gradlew :app:assembleRelease \
        -Ptarget-platform=android-arm64 \
        -Pdart-defines=$(printf '%s' \
          'POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app' \
          | base64 -w0)

  with `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17` and
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
  `-Pdart-defines` takes a comma-separated list of base64-encoded `KEY=VALUE`
  pairs. It is the Gradle-path equivalent of `--dart-define`, so the endpoint
  does not require leaving the canonical release command.
- Size: 34,211,833 bytes — in the ~34 MB arm64 band, not the ~50 MB universal
  band, so `-Ptarget-platform=android-arm64` was honoured.
- SHA-256: `9a0f74070f0129b2180b4b3237fbfacaf001ee6c8808a26e128d7ae06bb1be7f`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`),
  minSdk 24, targetSdk 36.
- Release guard PASS. The build printed "Verified arm64-v8a native payload in
  app-release.apk: lib/arm64-v8a/libdartjni.so, lib/arm64-v8a/libpicoclaw.so,
  lib/arm64-v8a/libpicoclaw-web.so", and `unzip -l` confirms all three are
  packaged (131,248 / 37,224,801 / 24,641,889 bytes).
- Both Core binaries are byte-identical to the device-verified pair
  (`libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`; `libpicoclaw-web.so`
  24,641,889 `b6b356f7...656db9ba5`). No Core source changed, so the Core half
  is already device-proven — and, decisively for the secret scan, those
  binaries were compiled before the service was deployed and therefore cannot
  contain any of its secrets.
- Endpoint present: `https://pocketclaw-telegram-setup-bot-83ai.vercel.app`
  appears exactly once, in `lib/arm64-v8a/libapp.so`, and in no other file in
  the APK. The previous candidate had zero occurrences, so the string is there
  because of the define and nothing else.
- No Milestone D feature behavior was changed to produce this build. The only
  difference from `7c34ab12...c4911178e` is that the define is now set.

### Secret scan — explicit result

Performed over the printable strings of every file in the APK (425,035 lines).

| Checked for | Result |
| --- | --- |
| Telegram bot token pattern `<digits>:<35 chars>` (manager or child) | **0 matches** |
| `TELEGRAM_MANAGER_BOT_TOKEN` | **0 matches** |
| `TELEGRAM_WEBHOOK_SECRET` | **0 matches** |
| `PAIRING_SECRET` | **0 matches** |
| `KV_REST_API_URL` / `KV_REST_API_TOKEN` | **0 matches** |
| `UPSTASH_REDIS_REST_URL` / `UPSTASH_REDIS_REST_TOKEN` / `REDIS_URL` | **0 matches** |
| `upstash` (any case) | **0 matches** |
| `redis://` or `rediss://` | **0 matches** |

The 64-hex scan — the shape of `openssl rand -hex 32`, used for both the
webhook secret and the pairing secret — returns 1,432 hits, and every one is
accounted for:

- 23 unique in `libapp.so`, all of them `google_fonts` 8.2.1 font-asset SHA-256
  checksums, each traced back to that package's `google_fonts_parts/*.dart`.
  Nothing in `libapp.so` is unexplained.
- 209 in `libpicoclaw.so` and 201 in `libpicoclaw-web.so`, inside binaries
  byte-identical to the pre-deployment device-verified pair.
- 0 in `libflutter.so`; the remaining hits are repeats across those files.

Every HTTPS host reachable from the Dart layer was enumerated as well:
`api.flutter.dev`, `docs.flutter.dev`, `fonts.gstatic.com`, `github.com`,
`pub.dev`, `t.me`, and the onboarding base URL. No credential-bearing host is
present.

### Regression validation run before this build

- `flutter analyze` — clean, no issues (Flutter 3.47.1, Dart 3.13.1).
- `flutter test` — 77/77 passed, including the Telegram onboarding config,
  stage, lifecycle, configuration-merge, and manual-fallback tests.
- Core was deliberately not rebuilt or revalidated: no Core source changed and
  the committed binaries hash-match the device-verified pair.

### Device checklist for this APK

Unchanged from the checklist recorded under the superseded candidate below.
Run it, plus the standing regression sweep, and report back before any merge.

## Milestone D Telegram Onboarding APK — SUPERSEDED CANDIDATE (no endpoint)

- Status: SUPERSEDED by `9a0f7407...6bb1be7f` above. Retained as the record of
  the build that proved the app ships no endpoint when the define is unset.
  It was built before the service existed and has no endpoint compiled in, so
  it cannot run the flow. Do not install it for the device test.
- Path: `build/app/outputs/apk/release/app-release.apk` (ignored; not committed)
- Built: 2026-08-26 with the canonical command
  `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`,
  `JAVA_HOME=/home/lordegypt/PocketCLaw/.tooling/jdk-17`,
  `GRADLE_USER_HOME=/home/lordegypt/PocketClaw-App/.tooling/gradle-stage-a-clean`.
- Size: 34,211,793 bytes
- SHA-256: `7c34ab12b544e585981c46632a5246a3a3fe66da24a84fce2c0b831c4911178e`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (version code `3`)
- Release guard PASS for all three required libraries.
- Both Core binaries are byte-identical to the device-verified pair
  (`libpicoclaw.so` 37,224,801 `cb9b2cde...fb895818`; `libpicoclaw-web.so`
  24,641,889 `b6b356f7...656db9ba5`). Milestone D changed no Core source, so
  the Core half of this APK is already device-proven. `libapp.so` grew from
  6,751,112 to 6,947,720 bytes, which is the new Dart code.
- No secret is embedded: 0 occurrences of a manager token pattern, and 0
  occurrences of a baked-in onboarding endpoint. The endpoint is a build-time
  `--dart-define` that is unset in this build.

### Why this cannot be device-tested yet

The flow depends on a Telegram bot that PocketClaw owns and that has Bot
Management Mode enabled. That bot does not exist, so there is nothing to point
the app at. Inventing a token or falling back to another project's setup
service was not an option, so the app in this build reports that automatic
setup is unavailable and offers manual token entry.

What is verifiable today, and was verified:

- 62 service tests across six Go packages, all against a fake Telegram.
- 50 new Flutter tests covering every stage, the lifecycle handling, the
  configuration merge, and the manual fallback. 77 Flutter tests in total.
- Core regression: 92 packages ok, `pnpm lint` clean.
- `flutter analyze` clean, release APK built, build guard passed.

### What to test on the device, once the operator setup is done

1. Settings shows a Telegram entry; opening it offers Connect Telegram, not a
   token field.
2. Connect issues a pairing; Open Telegram lands on Telegram's creation screen
   with the name and username already filled in.
3. Confirming in Telegram and returning shows Bot created, then Connected, with
   the new `@pocketclaw_..._bot` username.
4. The QR path works from a second device.
5. Backgrounding PocketClaw mid-flow and returning resumes the same pairing.
6. The bot answers its owner in Telegram and its `/start` reply is
   PocketClaw-branded, with no PicoClaw, Hermes, or Sipeed wording.
7. `allow_from` contains the creating user's Telegram ID, and the bot ignores
   other users.
8. Letting a pairing expire shows the expiry with a working retry.
9. Manual setup still works from the same screen.
10. Regression: startup, DNS, provider catalog, OpenCode, Skill Hub, workspace,
    MQTT, Core lifecycle, branding.

## Telegram Manager Bot — OPERATOR STATE (2026-08-26)

Recorded because these facts are external to this repository and cannot be
derived from it.

| | |
| --- | --- |
| Official Telegram manager | `@PocketClawSetupBot` |
| Display name | PocketClaw Setup |
| Manager created | **YES**, by the project owner |
| Bot Management Mode enabled | **YES**, manually in the BotFather mini app |
| Managed-bot deep link manually tested | **YES** — Telegram opened the managed-bot creation flow against `@PocketClawSetupBot` |
| Live `getMe` → `can_manage_bots` verification | **PASS (2026-08-26)** — the deployed service reports `Connected as @PocketClawSetupBot`, `can_manage_bots = true` |
| Public service repository | `Lord1Egypt/PocketClaw-Telegram-Setup` (public, MIT) |
| Deployment clone | `Lord1Egypt/pocketclaw-telegram-setup-bot` (private), kept in sync with upstream |
| Service deployment | **LIVE** at `https://pocketclaw-telegram-setup-bot-83ai.vercel.app` |
| Telegram webhook | **REGISTERED** at `…/telegram/webhook` |
| Pairing storage | **CONNECTED** — Upstash Redis over REST, via the Vercel Storage integration |
| Test pairing | **PASS** — created and read back through the live service |

The manager bot token is **not** recorded here, in any other document, in the
repository, or in the APK. It is entered directly into Vercel's environment
variables.

**The previously issued token is considered exposed** — it appeared in a
screenshot — and must be revoked with `/revoke` in BotFather. The production
deployment uses the regenerated token and only that.

The live `can_manage_bots` check is now **done**. It was the last link that
neither the BotFather UI nor a manually-opened deep link could establish, since
only the running service makes that API assertion. The setup page's **Verify
Telegram** button performed it against the production token and returned
`can_manage_bots = true`.

Every server-side prerequisite for Milestone D is therefore satisfied. What
remains is entirely on the app side: rebuild the APK with
`--dart-define=POCKETCLAW_ONBOARDING_BASE_URL=https://pocketclaw-telegram-setup-bot-83ai.vercel.app`
and run the device checklist.

## CODEX SOL HANDOFF — PRE-RELEASE FIX (2026-08-26)

### Status and rollback

- Candidate status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE
  BLOCKED**. Do not merge or create `v0.2.0-rc1` until the device checklist
  passes.
- Work branch: `fix/user-facing-log-privacy`.
- Exact starting commit: `e5b88ff1a4c8f76e321c07af97eff5ca23d59d78`.
- Pushed rollback branch: `checkpoint/pre-codex-sol-prerelease-fix`.
- Pushed annotated rollback tag:
  `pre-codex-sol-prerelease-fix-20260826`.
- Both rollback refs resolve to the starting commit. Historical
  `phase2-milestone-d` remains at
  `8f861bca1c82b43b306e95b14e277269260bbab0`; it was not moved.

### Physical findings that triggered this candidate

- **PRE-RELEASE BLOCKER — PHYSICAL DEVICE REPRODUCED:** "Telegram may send
  Thinking placeholder for message A and stall the final response indefinitely
  until message B arrives; message B then triggers/delivers the response
  belonging to A."
- A broader reproduction returned "The model returned an empty response",
  then accepted several later Telegram updates and emitted several
  `Thinking... 💭` placeholders without final responses before recovering.
- Native Settings disagreed with the correctly connected Core console. A later
  installation showed only the imported `gh` skill. Android Logs showed the
  upstream-branded PID message and ANSI/control/block glyphs.
- The basename-caller and queue/drain fixes at the starting commit had already
  passed physically and were preserved.

### Root causes and fixes

- Native Telegram state was a fragile second interpretation of Core's split
  config. Native Settings is now the neutral `Telegram / Manage Telegram
  connection` shortcut to the authoritative Core route `/channels/telegram`.
  Tapping it has no pairing callback; pairing begins only from an explicit Core
  page action.
- Core secure fields are split between `config.json` and `.security.yml`; raw
  native JSON writes are invalid because secure values are redacted and the
  security file wins. Managed/manual setup now writes through Core's
  loopback-only, per-process-authenticated, write-only
  `PUT /api/pocketclaw/android/telegram` bridge. It returns no credential.
  Failed/cancelled replacement pairing performs no write, so the old bot stays
  active until a new pairing succeeds.
- Telego's default `fasthttp` path had no whole-request deadline when the
  gateway context had none. A lost mobile connection could block one outbound
  operation indefinitely while other update handlers accepted messages and
  sent placeholders. Telegram now always uses a proxy-preserving
  `net/http.Client` with a 45-second deadline.
- `TelegramChannel.EditMessage` swallowed post-connect errors and Manager
  ignored exhausted final delivery. Edit errors now propagate, normal send is
  the fallback, final delivery is synchronous, and failures propagate. Failed
  placeholder edits remain correlated: successful fallback send deletes the
  stale placeholder (or edits if deletion fails); exhausted normal send makes
  one final correlated edit attempt.
- Placeholder, typing, and reaction state was keyed only by channel/chat, so
  close arrivals could overwrite one another. Every accepted update now gets a
  safe random process-local lifecycle ID. Same-session Telegram messages are
  independent FIFO inbound requests instead of steering. Trace fields contain
  only event, correlation ID, channel, durations/counts, and tool name — no
  content, user/chat ID, token, arguments, session key, or credential.
- No causal link was found between the exactly-once Android log queue and the
  Telegram stall. The queue/drain fix remains intact.
- Tool audit found Android `exec` already has a 60-second default timeout,
  kills process trees, collects output without a pipe-drain deadlock, and
  returns `(no output)` for empty output. Skill HTTP clients are bounded.
  Success/failure/timeout/empty-output tests pass; there is no evidence `gh`
  caused the stall. Safe lifecycle events will locate any future device stall.
- Android Core is a sticky foreground service with a partial wake lock;
  Flutter pause/resume does not stop it. No source evidence ties the incident
  to backgrounding, but foreground/background/screen-locked behavior still
  requires physical validation. No battery hack was added.
- Both Milestone D and current source bundle eight templates:
  `agent-browser`, `github`, `hardware`, `picoclaw-agent`, `skill-creator`,
  `summarize`, `tmux`, `weather`. `picoclaw-agent` is deliberately unseeded, so
  the intended fresh baseline is **seven**. Import writes only the new skill
  directory and cannot replace others. The exact device reason for "only gh"
  is unknowable without its filesystem, but a real repair gap existed: existing
  config skipped seed paths. `onboard ensure-workspace` now fills only missing
  embedded files at Core-console startup, never overwrites user files, never
  newly seeds `picoclaw-agent`, and preserves an existing user copy.
- Logs now store one sanitized plain-text representation before display and
  export. CSI/SGR (RGB/256 included), OSC, DCS/SOS/PM/APC, cursor/erase, CR,
  backspace, C0/C1, DEL, and orphaned CSI fragments are removed while Arabic,
  emoji, ordinary Unicode, tabs, and newlines survive. Android sets
  `NO_COLOR=1`, `TERM=dumb`, and prints plain `PocketClaw`. Stale PID text is
  `pid belongs to another process; ignoring stale pid file`. No global string
  replacement was used.

### Changed source areas

- Android host: `PicoClawMethodChannel.kt`, `PicoClawService.kt`.
- Flutter: `lib/main.dart`; Core channel/service/log sanitizer; Telegram config
  writer; config/onboarding/settings widgets. The obsolete native status reader
  and connected page were deleted.
- Core lifecycle: `pkg/bus/types.go`, agent lifecycle/mailbox/pipeline files,
  channel base/interfaces/manager, and Telegram transport.
- Core config/skills/log/UI: onboard helpers/command, Android bridge API, web
  startup/onboarding/banner/gateway, Telegram route test, DingTalk and Teams
  titles.
- Tests changed across Flutter, agent lifecycle, Android bridge, skills,
  onboarding, frontend route, logger, and log sanitization.
- `core/pocketclaw-core-v0.3.1.patch` was regenerated (93 changed files) and
  intentional divergence recorded in `UPSTREAM_TRACKING.md`.

### Validation

- `flutter analyze`: PASS, no issues. `flutter test`: PASS, 94 tests.
- Frontend: Vitest PASS (2 files / 36 tests), `pnpm exec tsc -b` PASS,
  `pnpm lint` PASS.
- Go PASS: logger, PID, providers, all web backend packages, agent, skills,
  androiddns, MQTT, onboard, commands, channels, Telegram, and tools. The final
  agent/channel/Telegram run included long timeout cases.
- Deterministic lifecycle tests prove: empty provider A finalizes explicitly
  while idle; close B/C arrivals finalize independently in FIFO order; failed
  edit sends normally and cleans the placeholder; exhausted send can retry the
  placeholder as final delivery.
- Canonical Core build used repository-local `core/src/`, root Make targets,
  `-trimpath`, `-s`, and `-w`; both binaries contain zero developer paths.

### Candidate artifact

- APK: `build/app/outputs/apk/release/app-release.apk`
- Size: `34,239,649` bytes
- SHA-256:
  `f663d25a2fffb0ce969ad4a9ce3405c1e563b6263c7af37e90768eef471c621d`
- Package/version: `com.lord1egypt.pocketclaw`, `0.1.3` (3), minSdk 24,
  target/compile SDK 36.
- Onboarding URL occurs once:
  `https://pocketclaw-telegram-setup-bot-83ai.vercel.app`.
- Build guard PASS for the three required arm64 libraries.
- `libpicoclaw.so`: 37,224,801 bytes,
  `87653601023974156be1b1a255387ec3c93a0c154254a7b8d9e1012ac2e926e5`.
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `a8328f1d66932ba8da544060905965278898744c04c2ece5a993717e361400c5`.
- Secret scan PASS: zero Telegram-token shapes, secret environment names,
  Redis URLs, private-key blocks, or full provider-key shapes. Two hits are
  only seven-character provider-format prefixes compiled into Core, not keys.
  Zero `.upstream` paths and zero raw full gateway caller strings.
- Known deferred artifact issue remains once in `libapp.so`:
  `file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart`.
  It is not normal UI/log output. Keep under **FINAL RELEASE HARDENING**; do
  not rush obfuscation changes into this candidate.

### Deferred roadmap — record only

- Background & Battery page: optimization state, user-initiated settings,
  Samsung Sleeping/Deep Sleeping guidance, reliability information. PocketClaw
  cannot silently grant Unrestricted mode.
- Runtime / Statistics tab: local-by-default uptime, agent state, token counts,
  CPU/RAM/Core resources, request outcomes/latency, provider/model/channel
  state, restart count, and later charts.
- Manager-token rotation remains separate security hygiene. A token appeared
  in screenshots/chat; do not mark rotation complete without explicit external
  confirmation, and never retrieve or print it.

### Required next action

Install only the APK above. Run the full physical checklist, especially send
`تسلم` and send nothing else until its final reply arrives; then idle and send
another message repeatedly. Check foreground/background/locked screen,
placeholder/typing/streaming combinations, native Telegram navigation,
reconnect preservation, fresh/existing skills plus `gh`, clean exactly-once
Unicode logs/export, provider, Skill Hub, startup, and performance. Do not merge
or release on anything short of physical PASS.

## DEBUG log cleanup micro-pass — AUTOMATED PASS, PHYSICAL PENDING

Physical export from candidate `f663d25a...71c621d` contained valid Go timing
text such as `53.616µs` as `53.616�s`, and 185 routine successful requests to
the two UI polling endpoints. The prior basename caller, branding cleanup,
terminal sanitizer, and exactly-once queue/drain fixes had physically passed
and are unchanged.

The UTF-8 corruption was not in Go, Kotlin ingestion, the Dart sanitizer, the
in-memory queue, or the Logs UI. Android Export Logs used
`Uint8List.fromList(content.codeUnits)`, truncating Dart UTF-16 code units to
single bytes before the Kotlin MediaStore writer. `µ` became lone byte `B5`,
which is invalid UTF-8; Arabic and emoji were vulnerable for the same reason.
The Android export boundary now uses `utf8.encode`, while the rune-safe
terminal sanitizer remains unchanged. The real MethodChannel export transport
is tested by strict UTF-8 decode after sanitizing ANSI-wrapped Arabic, `µ`,
emoji, mixed-language text, and Unicode punctuation; it introduces no U+FFFD.

The polling noise was legitimate middleware output, not duplicate delivery:
DEBUG HTTP logging recorded each successful UI request to its own log and
status endpoints. The HTTP middleware now suppresses only expected `GET`
requests to exact paths `/api/gateway/logs` and `/api/gateway/status` with a
2xx response. Failures, redirects, unexpected methods, unknown routes, and all
other API requests retain normal DEBUG logging. No deduplication or endpoint
behavior changed.

Validation: `flutter analyze` clean; 97 Flutter tests; frontend 36 Vitest tests,
`tsc -b`, and lint; Go `pkg/logger`, `pkg/gateway`, `web/backend/api`, and
`web/backend/middleware` pass with the shipped `goolm,stdjson` tags. Core
provenance patch regenerated at 95 files. Canonical Core build passed
`-trimpath` with zero developer paths; the APK guard passed all three required
arm64 libraries. Artifact scan found zero Telegram token shapes, service secret
names, Redis/Upstash values, or raw full Go callers; the single known generated
Dart source URI remains deferred to final hardening.

Replacement candidate:

- APK: `build/app/outputs/apk/release/app-release.apk` and identical
  `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,239,073 bytes
- SHA-256: `eacbbc86b99429f114aba9ba1dca57224122fa176f6b4d99edf350454423f9a8`
- Package/version: `com.lord1egypt.pocketclaw` 0.1.3 (3), minSdk 24,
  target/compile SDK 36
- `libpicoclaw.so`: 37,224,801 bytes,
  `5c09eb72a1faa6dd6f8a0e6b66bcbc028f070d3eff9b8c5b5f87716c04d763bc`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `cb6b10cc2a951d2307959e9effbdbd052762fe827b9943c22a9cdcdd54b03b52`
- Onboarding endpoint occurs once in `libapp.so`; version unchanged.
- Status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE BLOCKED**.

Install only this replacement. Export a DEBUG log after several minutes and
verify valid `µ`, Arabic, emoji, punctuation, zero newly introduced U+FFFD,
continued ANSI/control removal, and no routine successful polling noise. Failed
polls and real API requests must remain visible. Continue the broader physical
pre-release checklist before any merge or release.

## Web Console log parity / internal identifier visibility — AUTOMATED PASS, PHYSICAL PENDING

The native/export cleanup remains intact. The remaining Web Console startup
boxes came from a separate path: the captured Core gateway CLI treated
`--no-color` as “remove RGB” but still emitted a six-line Unicode block-art
banner, while the React Logs page deliberately parsed SGR into styled spans and
did not handle OSC, cursor/erase, carriage return, backspace, or other terminal
protocols. The gateway log ring also stored raw child output, so the Web API
had no user-visible normalization boundary.

The earliest safe Web boundary is now `LogBuffer.Append`: it stores only the
shared brand-safe plain-text representation. The native Dart boundary and the
browser's idempotent legacy/raw-line guard are locked to the same canonical
JSON fixtures. Valid Unicode, Arabic, emoji, punctuation, `µs`, tabs, and
newlines survive; terminal protocols are removed. Literal box drawing is not
globally stripped because it can be legitimate Unicode. Instead, the captured
gateway is always started with `--no-color`, whose banner is now the single
plain line `PocketClaw`.

The launcher startup event is now `Starting gateway process` and never prints
the `libpicoclaw.so` path. Exact successful `GET /pico/ws` connection/close
events (recorder 200/2xx or upgrade 101) are suppressed at HTTP middleware.
Failures and unexpected methods remain visible as `/internal realtime
connection`, so the compatibility route itself is not exposed. The real
`/pico/ws` endpoint, library filenames, module paths, environment variables,
and other compatibility identifiers were not renamed.

Regression results: `flutter analyze` clean; 99 Flutter tests; frontend 3 files
/ 37 tests, `tsc -b`, and lint; tagged Go `pkg/logger`, `pkg/gateway`,
`web/backend/api`, `web/backend/middleware`, and `cmd/picoclaw` all pass. The
real React Logs page test consumes the shared contract and proves Arabic,
emoji, and `53.616µs` render with no ANSI/control fragments, routine
`/pico/ws`, internal library path, or unintended PicoClaw/Sipeed branding,
while a 500 remains visible under neutral wording. Core provenance is now 108
files; its generator was made deletion-safe after the intentionally removed
ANSI renderer exposed a stale tracked-file assumption.

Replacement candidate:

- APK: `build/app/outputs/apk/release/app-release.apk` and identical
  `build/app/outputs/flutter-apk/app-release.apk`
- Size: 34,241,381 bytes
- SHA-256: `3e138b4a53a0389b76dbef045649d906fe2db785cd2af606826f7a0f87170adc`
- Package/version: `com.lord1egypt.pocketclaw` 0.1.3 (3), minSdk 24,
  target/compile SDK 36
- `libpicoclaw.so`: 37,224,801 bytes,
  `c9c348e9c637a7552e810396bfba5460ad4a3f65ab506d933ac5d05ea68d236e`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `c891ca033ededb6b8941d997fbc8e0a8d69eb18f44a6ed2176f878f65d34840a`
- Both Core libraries contain zero developer paths; packaged hashes match.
- Live onboarding endpoint occurs once in `libapp.so`; metadata unchanged.
- Build guard: PASS for `libdartjni.so`, `libpicoclaw.so`, and
  `libpicoclaw-web.so`.
- Status: **AUTOMATED PASS; PHYSICAL DEVICE PENDING; RELEASE BLOCKED**.

Install only this replacement. In Native Logs, Export Logs, and Core Web
Console Logs, verify zero normal visible `picoclaw`/`PicoClaw`/`sipeed`/`Sipeed`,
no terminal boxes or ANSI, intact Arabic/emoji/`µs`, no successful
`/pico/ws` noise, and visible neutral wording for genuine failures. Do not
merge or release before the complete standing physical checklist passes.

## Final enabled-channel brand micro-fix — AUTOMATED PASS, PHYSICAL PENDING

Physical validation of commit `3611ca1` substantially passed: Web Console
terminal/control boxes are gone, the PocketClaw banner and UTF-8/`µs` are
correct, the internal library path is absent, and all eight skills are
available with 17 tools loaded. One raw startup summary remained:
`✓ Channels enabled: [telegram pico]`.

`pico` is the exact internal singleton channel ID for the Core Web Console
transport. It owns the authenticated `/pico/ws` connection, browser chat
sessions, streaming/tool-feedback protocol, and `/pico/media` delivery. Its
config key, factory registration, token handling, channel identity, routes,
session IDs, and compatibility behavior remain unchanged.

Only the two user-facing startup/reload summary print sites now map exact
`config.ChannelPico` to display label `pocketclaw`. The internal slice is copied
and left untouched; there is no substring or global replacement. Targeted
coverage proves `[telegram pico]` renders as `[telegram pocketclaw]`, while the
original internal value remains `pico` and unrelated names are unchanged.

Tagged Go `pkg/gateway`, `pkg/logger`, `web/backend/api`,
`web/backend/middleware`, and `cmd/picoclaw` tests pass. Core provenance is 110
files. Canonical Core build has zero developer paths, and the APK guard passed.

Replacement candidate:

- APK size: 34,241,857 bytes
- APK SHA-256: `aab3c565bd6bec2eb714443756e16be5d2ebe8d4496d94e64a6b8ecac25b3582`
- `libpicoclaw.so`: 37,224,801 bytes,
  `10446d8156b0a33a920f31c568b25c9ae59f96ea5f8db576f9fe40dce71b45f1`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `683463dfee287b7cd88f592b3ec534e2f8458e3bace77ca9bfee4dd7ce5f5ce4`
- Packaged Core hashes match; live onboarding endpoint occurs once.
- Status: **AUTOMATED PASS; FINAL LABEL PHYSICAL CHECK PENDING; RELEASE BLOCKED**.

Next physical check: restart Core and confirm the Web Console line is exactly
`✓ Channels enabled: [telegram pocketclaw]` (order may follow configured
channel order), with every already-passed log and skill behavior preserved.

## Final user-visible caller brand fix — AUTOMATED PASS, PHYSICAL PENDING

Physical testing found the remaining structured WebSocket logger header as
`INF pico pico.go:1013 > ...` / `pico.go:1071`. The internal Go package,
source filename, `ChannelPico`, routes, media protocol, config keys, and channel
identity remain unchanged.

The shared user-visible normalization contract now recognizes only structured
logger-header fields. Exact component token `pico` displays as `realtime`, and
exact caller basename `pico.go` displays as `realtime.go`; the numeric line
suffix is captured unchanged. The mapping is implemented at the Go Web log
buffer boundary, Dart native/export boundary, and idempotent React Logs guard,
using the same canonical fixtures. It is not a source rename or substring
replacement: `picometer`, `picophone.go`, and `pico_client` remain unchanged.

Expected example:

`INF pico pico.go:1013 > WebSocket client connected`

becomes:

`INF realtime realtime.go:1013 > WebSocket client connected`

Validation: `flutter analyze` clean; 99 Flutter tests including actual stored
and exported representation; frontend 37 tests/tsc/lint including the real Logs
page; tagged Go API/middleware/logger/gateway/CLI suites including `LogBuffer`;
110-file Core provenance; zero developer paths; packaged hashes and live
endpoint verified; permanent APK guard passed.

Replacement candidate:

- APK size: 34,242,865 bytes
- APK SHA-256: `1eeca7c993d657f0e6e763d94691a054e584592d0989445454144ac15ad089f7`
- `libpicoclaw.so`: 37,224,801 bytes,
  `a379ae4531d1c64bf653d503f9f20fa627d497389e037346815525f6407eafb9`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `584dd9ae2e9c11e46d50f879020a07758ff612b744ae69448ac3f3cb71fa4a01`
- Status: **AUTOMATED PASS; PHYSICAL DEVICE FINAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: confirm Native Logs, Export Logs, and Web Console Logs all
show `realtime realtime.go:<original line>` for these WebSocket events, with
genuine failures visible and every previously passed Unicode, control-cleanup,
exactly-once, polling, channel-label, skill, Telegram, and provider behavior
unchanged.

## Web Console Logs scroll/jitter micro-pass — AUTOMATED PASS, PHYSICAL PENDING

Physical testing of the preceding candidate confirmed the complete
user-visible log cleanup, but the Web Console Logs viewport moved between
one-second polls. Native Logs did not reproduce it.

The Web page had two layout timing defects. Following users were corrected with
a passive `useEffect`, so an appended row could paint at the old scroll offset
before the browser was snapped to the new bottom. Long lines were also hard
wrapped in JavaScript using a column count recomputed by a `ResizeObserver` on
the entire content element; every appended row changed that element's height,
causing another measurement and potentially rewriting all long-row text/layout.
Rows additionally used array-index keys rather than the event identity already
available from the incremental API.

The Logs page now records follow state from the viewport's scroll events and
applies bottom correction in `useLayoutEffect`, before paint, only when the user
was within 24 px of the bottom. A scrolled-up viewport receives no programmatic
scroll. Log IDs are `run_id:absolute_offset`; memoized rows use those IDs as
keys. JavaScript hard wrapping and the content resize observer were removed;
the unchanged sanitized string is rendered once and wraps natively with CSS.
Browser anchoring is disabled inside the explicit-policy log content.

The real Logs page DOM suite covers bottom following, scrolled-up preservation,
unchanged long Telegram-style row node/text identity with Arabic, emoji and
`53.616µs`, and repeated no-new-log rerenders. Frontend Vitest is 41/41, `tsc
-b` and lint pass. Tagged Go API/middleware tests and the Native/Export
exactly-once/sanitization regression pass. Core provenance is 113 files; both
Core binaries have zero developer paths; the permanent APK guard passed.

Replacement candidate:

- APK size: 34,239,873 bytes
- APK SHA-256: `be5d7cbb18c0378dad0a3d53d2d3a4e71001411121fa056f7a6606645070fc96`
- `libpicoclaw.so`: 37,224,801 bytes,
  `715cd790d1af3f295a50a87a64d5ac8b2fbc0454c8e2268f55896538c6143cf1`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `e3930ae24e5d7e7185a470af65caf3c8d46a97335daecc3590ab64f9f5f5da14`
- Status: **AUTOMATED PASS; WEB VIEWPORT PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: leave Web Console Logs untouched at the bottom through
multiple updates, then scroll upward and wait through multiple polls. Confirm
no shake, rewrap, or forced bottom jump while every already-passed log
sanitization and branding behavior remains intact.

## Telegram token log-redaction security pass — AUTOMATED PASS, PHYSICAL PENDING

Physical evidence narrowed the disclosure to Core Web Console DEBUG Logs;
Native Android Logs remained clean. Investigation confirmed **case A**: Telego
constructs a full Bot API URL, but PocketClaw's compatible third-party logger
called its masking function before `logMessage`, stdout writers, and the Web
backend `LogBuffer`. The full token did not enter persisted Web history through
this path. The old mask deliberately retained the bot-ID prefix plus the first
and last four secret characters, and those fragments did enter the Web stream
and stored ring. This is a fragment-disclosure defect, not evidence that the
complete credential was persisted or compromised; no credential was rotated
or modified.

The same pre-stdout logger boundary now removes the complete Bot API credential,
including normal/percent-encoded forms and bare credentials, without retaining
an ID, prefix, or suffix. Bearer/Basic Authorization values are also removed.
Before `LogBuffer.Append` persists a Web line, Telegram Bot API URLs—whether
raw, fully redacted, or using the historical partial mask—normalize to
`Telegram API call: <operation>`. Failures retain HTTP method, operation,
status/error, timeout, and latency text. The React normalizer provides an
idempotent legacy/raw guard. Native/Export source and queue code were not
changed.

Synthetic-only regressions cover `getMe`, `getUpdates`, `sendMessage`,
`editMessageText`, timeout/error/5xx cases, arbitrary methods and custom API
servers, percent encoding, bare tokens, Authorization credentials, public bot
metadata preservation, actual `LogBuffer` storage, and the real Logs page.
Go logger/Telegram/gateway/API/middleware/CLI suites pass; frontend Vitest is
42/42 with tsc/lint; the unchanged Native/Export log regression is 7/7. Core
provenance is 115 files; zero developer paths and the permanent APK guard pass.

Replacement candidate:

- APK size: 34,240,641 bytes
- APK SHA-256: `8257e9f091039f2c29332b8f14e2d397d36bbb735593596a4caa0e567c7050fe`
- `libpicoclaw.so`: 37,224,801 bytes,
  `0e914550208fc7a77ce9319b32554be802108207789f992098d2186dc8b555e9`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `7d7b254b04b33919b6ebbfeac9147a06a4472de1c4fe6354b6ecb6fe6d9898c7`
- Status: **AUTOMATED PASS; WEB CONSOLE PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: enable DEBUG, exercise Telegram polling and message/edit
calls plus a recoverable failure, and confirm Web Console Logs show operation
names and useful failure details with zero credential fragments. Reconfirm the
already-passed viewport stability and Native Logs behavior.

## Final legacy brand visibility sweep — AUTOMATED PASS, PHYSICAL PENDING

Physical validation confirmed every preceding logging, viewport, Telegram,
Unicode, and Skills fix, then identified remaining structured compatibility
details in normal Core Web Console startup logs. The shared Web pre-storage
normalizer now maps only exact `channel=pico` and `type=pico` fields to
`pocketclaw`, hides exact `/pico/` only when attached to that internal channel,
uses PocketClaw realtime wording for exact protocol lifecycle messages, and
replaces the compatibility PID path with a semantic gateway PID message.
Exact realtime-subsystem failure messages remain visible with neutral wording.

Runtime `ChannelPico`, config/serialized IDs, Go packages/files, `/pico` routes,
the actual `.picoclaw.pid`, libraries, environment variables, and provenance
were not changed. There is no blanket or substring replacement. Negative tests
prove `picometer`, `picophone.go`, `pico_client.go`, `topic=pico-test`,
`my-pico-notes.txt`, and `.picoclaw.pid.backup` remain unchanged. A
representative stored startup plus real Logs-page DOM test reports zero
unintended legacy occurrences while preserving the security warning.

Regression results: frontend Vitest 44/44, TypeScript, lint; tagged Go logger,
gateway, channel, Pico, Telegram, Skills, API, middleware, and CLI suites;
Flutter analyze and 99/99 tests. The 115-file Core provenance patch was
regenerated, both binaries contain zero developer paths, and the permanent APK
payload guard passed.

Replacement candidate:

- APK SHA-256: `309f6d7ac47206015e3c3c9a5d5f1cf1903b5fb8b2c07783d21131e67a5f3030`
- `libpicoclaw.so`: 37,224,801 bytes,
  `49f89ae22f5020425ff9346fd579705bd2a44e43aa42018985cf1702d3be656f`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `98f3fa9df6b89bb745181793da2351adc7f8086ea63b20e2507d46bfaaf08bae`
- Status: **AUTOMATED PASS; WEB CONSOLE PHYSICAL GATE PENDING; RELEASE
  BLOCKED**.

Next physical check: restart Core with DEBUG logging and confirm channel
initialization, security, webhook, realtime, and PID events use only the
semantic PocketClaw display while all previously passed behavior remains intact.

## Telegram DEBUG final cleanup — AUTOMATED PASS, PHYSICAL PENDING

Telego's raw successful response is correct: `Err: [<nil>]`. PocketClaw's
pre-stdout credential redactor also preserved it correctly. Corruption occurred
when the Web backend normalized captured stdout before `LogBuffer` storage: the
orphaned-CSI fallback accepted `<` as a parameter byte and `n` as a final byte,
removed `[<n`, and persisted the malformed remainder `il>]`. React was already
rendering that stored string as safe text, not HTML.

The orphaned fallback now recognizes only numeric/private-numeric CSI remnants;
real ESC/C1 CSI handling remains unchanged. Ordinary `<` and `>` text,
including Arabic and emoji, stays intact. Exact successful Telego nil fields
display semantically as `Err: none`; generic `<nil>` text remains unchanged.
The real Logs DOM proves `<tag>` remains text and creates no element.

At the pre-stdout Telego logger boundary, routine `getUpdates` request lines and
only exact successful empty responses are omitted. Failures have their separate
`Execution error getUpdates` line, and non-empty or unsuccessful responses stay
visible. Web pre-storage, React, and Native/Export normalizers enforce the same
idempotent policy for legacy/raw lines. Four repeated empty poll pairs add zero
stored entries. Telegram lifecycle/delivery/onboarding and native queue logic
were not changed.

Regression results: frontend Vitest 45/45, TypeScript, lint; tagged Go logger,
gateway, channels, Pico, Telegram, Skills, API, middleware, and CLI suites;
Flutter analyze and 101/101 tests. The 115-file Core patch was regenerated,
both Core binaries contain zero developer paths, and the permanent APK guard
passed.

Replacement candidate:

- APK size: 34,243,585 bytes
- APK SHA-256: `2c00720a44a2b2ac5de2c82c202c172129e778392eabf5434c36883a322da27c`
- `libpicoclaw.so`: `7c1d3918e30ff673e62a963822fcdd00e327805221cea1b963cb222d1138ef1f`
- `libpicoclaw-web.so`: `1611b6e102fbcff726213bef659cbb05509efc12e63592a373df64fe14d09256`
- Status: **AUTOMATED PASS; PHYSICAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: leave Telegram polling in DEBUG for several minutes;
routine empty `getUpdates` must stay silent. Exercise a non-empty update and a
recoverable failure, verify useful details remain, and reconfirm every prior
physical pass.

## Agent DEBUG privacy, Telegram payload privacy, and default identity — AUTOMATED PASS, PHYSICAL PENDING

Physical Web DEBUG evidence exposed two source-level payload paths. First, the
freshly generated system prompt itself—not merely its log preview—still used
the legacy lowercase product identity in `ContextBuilder.getIdentity`. Fresh
PocketClaw defaults now send `# PocketClaw 🦞` and `You are PocketClaw, a
helpful AI assistant.` to the model. Existing user-authored prompt overlays and
all internal/upstream compatibility identities remain unchanged.

Second, normal Agent logs explicitly emitted a system-prompt preview, complete
message/tool JSON, raw reasoning text, complete tool-call argument previews,
and structured routing/session identifiers. The explicit payload logs and dead
raw formatters are removed. Normal logs retain model, iteration, message/tool
counts, prompt length, content/reasoning lengths, token usage, tool names,
status, duration, and failures. An exact-field pre-writer copy redacts session
keys and internal identifiers, omits raw-content fields, and does not mutate the
runtime maps or routing values.

Telego's `Response.String()` also included complete successful Telegram result
JSON (chat/user IDs, names, usernames, language and message bodies). Before
this pass that raw result entered stdout and Web `LogBuffer`; it was not a
frontend-only leak. The Telego adapter now converts request/response data to
operation/status metadata before any writer. Empty successful `getUpdates`
remains silent, non-empty updates retain count/type only, successful operations
retain operation plus `ok=true`, and failures retain safe operation/error-code
metadata. Backend, React, and Native/Export normalization provide idempotent
legacy/raw guards. Credentials remain fully redacted.

Regression results: tagged Go Agent/logger/Telegram/Pico/gateway/API/
middleware/CLI suites; frontend Vitest 46/46, TypeScript and lint; Flutter
analyze and 101/101 tests. Core provenance is 124 files, both binaries contain
zero developer paths, the live onboarding endpoint is packaged, and the
permanent APK guard passed.

Replacement candidate:

- APK size: 34,251,141 bytes
- APK SHA-256: `46ca983a380d1a1b69f71f01cf18b840b5007054d732d8430fed5d11cd2b908e`
- `libpicoclaw.so`: 37,224,801 bytes,
  `8ed15601f3312c034e21df55bcbaa980b5c1c98ff1b15caa1a404be3129be24a`
- `libpicoclaw-web.so`: 24,641,889 bytes,
  `2100100454a42b0c7517084a9b52a069ad0153476da6d9e3787582bc7f5fc1d7`
- Status: **AUTOMATED PASS; PHYSICAL GATE PENDING; RELEASE BLOCKED**.

Next physical check: generate a fresh realtime/Telegram conversation under
DEBUG and confirm only lifecycle/count/length/tool-name metadata appears; no
prompt, message, reasoning, tool arguments/schemas, session/internal IDs, or
Telegram payload values may appear. Reconfirm Skills 8/8, Tools 17+, Telegram
delivery, Web viewport stability, and all prior Unicode/branding passes.

## 2026-08-29 — v0.2.0-rc1 release candidate frozen

**PHYSICAL DEVICE: PASS.** The candidate
`5760247a17ccff68d188180875f1812c150dd7ae6de45e3f109d7ba07c2186b9`
(34,260,709 bytes) was validated by the user on a real ARM64 Android device.
That run cleared app/Core startup, Skills 8/8, Tools 17, Telegram owner-only
authorization and final delivery, the no-second-message stall fix, Web/realtime
owner authorization, password/session dashboard auth, Core Gateway remaining
loopback-only on 18790, Public Mode OFF and ON, live OFF→ON→OFF→ON rebinding
without a manual service restart, a real LAN URL of the `192.168.x.x:18800`
form, authenticated Dashboard access from a computer on the same LAN, Core
18790 staying off the LAN, QR/connect URL refresh, Telegram continuity across a
Public Mode change, Web Logs stability, UTF-8/Arabic/emoji/µs rendering, and
every privacy expectation from the preceding passes. The DEBUG log-cleanup
physical gate that previously blocked release is therefore CLEARED.

Source provenance for that artifact is proven, not assumed. The Core build
stamps `BuildTime` through `-ldflags`, so consecutive builds differ in exactly
64 bytes at identical length. Rebuilding this workspace's Core source with the
validated binaries' own timestamps pinned reproduced both of them byte for
byte:

- `libpicoclaw.so` 37,224,801 bytes,
  `4e8c23c70bd77fbdce96d04004dd13b3ba4cac8e1e03164296cc47f7ead1ffb6`
- `libpicoclaw-web.so` 24,707,425 bytes,
  `45427e0d48c53c7611625a8b621e4a4f565bfec11702d12e66c94ada2c900917`

Those exact binaries are what the repository now carries and what the RC APK
packages, so the released native payload is the physically validated payload.

Release-candidate build identity:

- Package: `com.lord1egypt.pocketclaw`
- Version: `0.2.0`, version code `4` (was `0.1.3`/`3`, the inherited FUI
  baseline; a candidate tagged `v0.2.0-rc1` must not report `0.1.3`)
- ABI: `arm64-v8a`; the permanent guard verified `libdartjni.so`,
  `libpicoclaw.so`, and `libpicoclaw-web.so` under `lib/arm64-v8a/`

Automated verification at the freeze: `flutter analyze` clean and 114/114
Flutter tests; frontend 46/46 Vitest, `tsc -b`, and lint clean; the **complete**
Go suite green under `-tags goolm,stdjson`, along with `go build ./...` and
`go vet ./...`. The `goolm` tag selects the pure-Go Olm implementation, so the
previously reported `olm/olm.h` host dependency is not required at all and is
no longer an accepted exception. Core provenance regenerated to 141 files and
reproduced the committed patch byte for byte; both binaries carry zero
developer paths; `core/verify-no-external-source.sh` passed.

One known, non-blocking condition: `go test -race ./web/backend/api` fails in
`TestStartGatewayLocked_UsesReloadedConfigForBootSignature`. The race is
between that test's own cleanup calling `cmd.Wait()` and the production monitor
goroutine's `cmd.Wait()` on the same `exec.Cmd` — a test-harness defect, not a
production data race; production calls `Wait` once. It reproduces identically on
`develop` at `8f861bc`, so it is pre-existing and not a regression from this
work. Every other test in that package passes under `-race`, as do the race
runs for `pkg/channels`, `pico`, `telegram`, `logger`, `netbind`, `config`,
`skills`, and the rest of `web/backend`. Fixing the shared Kill+Wait cleanup
pattern in `gateway_test.go` is deferred; it is out of scope for RC closure.
