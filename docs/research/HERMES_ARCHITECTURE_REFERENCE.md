# PocketClaw Post-Stable Architecture Reference — Hermes Android Mining

> Status: **research / future architecture only**  
> Created: **2026-09-16**  
> PocketClaw documentation branch: `docs/hermes-architecture-notes`  
> Based from PocketClaw branch: `feature/final-release-hardening` at remote commit `0695a33452f7f80d8c187a419a6c28c5e9ec4722`  
> Reference project: `adybag14-cyber/hermes-agent`, default branch `codex/termux-five-goals`

---

## 0. Why this file exists

This file is a continuation anchor for future ChatGPT / Claude / Codex sessions.

The goal is **not** to turn PocketClaw into Hermes Agent and **not** to copy the Hermes fork feature-for-feature. The goal is to mine the fork for:

- architecture patterns;
- lifecycle/readiness contracts;
- release engineering ideas;
- resource-efficient implementation patterns;
- security boundaries;
- provider/MCP/runtime design ideas;
- test strategies that prevent regressions.

PocketClaw must remain the lighter product.

**Rule:** adopt the brain, not the body.

Hermes is a useful reference because the Android fork has had substantial independent development. At review time GitHub showed the branch as roughly **631 commits ahead** and **3932 commits behind** `NousResearch/hermes-agent:main`. Those numbers are volatile and must be rechecked before any later comparison.

---

## 1. Current PocketClaw checkpoint — read before future work

This section is intentionally transient. Verify it against the repository and the developer machine before acting.

### Stable release state at the time this note was created

- Target product: **PocketClaw v0.2.0 Stable**.
- Current work branch: `feature/final-release-hardening`.
- Remote HEAD when this documentation branch was created: `0695a33`.
- Do **not** merge/tag/publish based only on this document.
- No Google Play publication is planned for this Stable release.
- No public AAB is planned for this Stable release.

### Physically verified items already known from the Samsung run

- Notification permission flow: **PHYSICAL PASS** after moving the per-install "already asked" marker out of backup-restorable preferences.
- Telegram first-message/readiness: **PHYSICAL PASS**.
- First dashboard password / Public Mode first-claim flow: **PHYSICAL PASS**.
- Telegram context compression + multi-image media-group stress: **PHYSICAL PASS**.
- OpenCode Go provider session-header path had already been physically verified in earlier rounds.

### Work that was in progress locally when Claude hit its usage limit

The local working copy may contain **uncommitted changes not present in remote `0695a33`**. Before doing anything else, inspect:

```bash
git status
git log --oneline --decorate -10
git diff
git diff --cached
```

The interrupted work included:

- renaming active runtime `picoclaw_media` writes to PocketClaw-owned naming;
- a full Pico/PicoClaw residue re-audit;
- DEBUG absolute Android private-path normalization/redaction;
- a new logger test similar to `pkg/logger/path_redaction_test.go`.

**Do not reset, clean, checkout, pull destructively, or overwrite the local work until it is inspected.**

Earlier still-pending verification items included PC-DEF-062 and older 050/052/055 checks. Re-read the defect ledger instead of trusting these identifiers forever.

---

## 2. Hermes fork snapshot used for this research

Repository:

- https://github.com/adybag14-cyber/hermes-agent
- Upstream: https://github.com/NousResearch/hermes-agent
- Reviewed default branch: `codex/termux-five-goals`

At review time the fork repository metadata reported approximately **2,060,896 KB** of repository size.

Latest reviewed release:

- Tag: `v0.13.157`
- Full universal Android APK: `hermes-agent-android-v0.13.157-universal.apk`
- Full APK size: **349,507,455 bytes** (~333.3 MiB)
- Full APK SHA-256: `8bd52f375b338c278591a2a265b6ff68add563e6cdb08985235cdc2ad2ed62db`
- Play universal Android APK: `hermes-agent-android-play-v0.13.157-universal.apk`
- Play APK size: **182,739,051 bytes** (~174.3 MiB)

The size difference is important: Hermes proves many ideas, but it also demonstrates why PocketClaw must not blindly bundle every runtime and capability.

Hermes Android documents:

- minimum Android API 24;
- target/compile API 36;
- `arm64-v8a` + `x86_64` release ABIs in a universal artifact;
- embedded Python 3.13 through Chaquopy;
- Java 17;
- remote OpenAI-compatible providers;
- local models through llama.cpp and LiteRT-LM;
- separate Full and Play distributions.

PocketClaw should copy **contracts and architecture**, not this footprint.

---

# 3. PocketClaw architectural identity to preserve

PocketClaw should remain a **Thin Agent Host**:

```text
Flutter UI
    |
Small Android native bridge
    |
PocketClaw Go Core
    |
Channels / Providers / Tools / MCP / Optional external runtimes
```

Core principles:

1. The baseline APK stays small and fast.
2. Heavy capabilities are optional or downloaded on demand.
3. No model weights in the base APK.
4. No bundled Linux root filesystem in the base APK.
5. No embedded Python unless a future feature proves it is necessary and wins an explicit resource-budget review.
6. Keep the Go Core authoritative for agent/runtime state where practical.
7. Native Android code should expose narrowly-scoped capabilities rather than become a second full agent runtime.
8. Prefer one authoritative state machine over duplicated status guesses in Flutter, Android, Go, Telegram, and Web.
9. New features must have a measurable resource budget before acceptance.
10. Do not introduce a dependency on the Hermes fork. It is a **reference repository only**.

### Soft product-size guardrail

Try to keep the baseline PocketClaw APK **below 100 MiB** unless the owner explicitly approves a larger baseline for a concrete reason.

This is a guardrail, not a release law. The important requirement is that heavy functionality does not silently become baseline baggage.

---

# 4. The core pattern to steal: state truth, not optimistic labels

The strongest reusable pattern observed across the Hermes Android work is the separation between configuration, authorization, process state, readiness, and execution.

Use this mental model everywhere:

```text
EXISTS
!= CONFIGURED
!= AUTHORIZED
!= STARTING
!= HEALTHY
!= READY
!= EXECUTABLE
!= SUCCESSFUL
```

Examples:

```text
Telegram configured
!= Telegram polling connected
!= receiver ready
!= commands published
!= first message deliverable
```

```text
MCP server configured
!= transport connected
!= schema loaded
!= server authorized
!= tool still authorized now
!= tool executable now
```

```text
Local model downloaded
!= artifact valid
!= runtime compatible
!= process started
!= health check passed
!= completion canary passed
```

PocketClaw already learned this lesson through the Telegram first-message/readiness bug. Future architecture should generalize it.

### Future PocketClaw runtime state vocabulary

A useful common vocabulary is:

```text
UNCONFIGURED
CONFIGURED
STARTING
HEALTHY
READY
DEGRADED
FAILED
STOPPING
STOPPED
```

Each subsystem may have extra states, but UI labels should derive from authoritative runtime evidence rather than configuration alone.

---

# 5. High-value architecture patterns to adopt later

## 5.1 Unified authoritative readiness snapshot

Relevant Hermes files:

- `android/app/src/main/java/com/mobilefork/hermesagent/backend/HermesRuntimeManager.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/backend/HermesRuntimeService.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/ui/chat/ChatReadinessStrip.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/backend/OnDeviceBackendManager.kt`

PocketClaw direction:

- one authoritative runtime snapshot;
- Flutter reads the snapshot;
- Telegram/Web/API derive status from the same source;
- never infer `Connected` from saved credentials;
- never infer `Ready` from a listening port alone;
- add completion/readiness canaries when the subsystem needs them.

Resource cost: **very low**.

Priority: **high, post-Stable foundation**.

---

## 5.2 Release evidence bound to exact artifacts

Hermes has a strong release-evidence contract that binds tests to exact source/artifact/device identities.

Reference:

- `android/RELEASE_EVIDENCE_V3.md`

Useful PocketClaw concepts:

- source SHA;
- APK SHA-256;
- APK byte count;
- signer certificate digest;
- Core fingerprint/build time;
- device model/API/build fingerprint;
- test case identity;
- explicit PASS / FAIL / NOT PERFORMED;
- immutable historical evidence;
- never reuse an old device/API result as proof for a new build/API.

Possible future layout:

```text
release-evidence/
  v0.2.x/
    candidate.json
    samsung-sm-a165f.json
    telegram.json
    provider-opencode.json
    public-mode.json
    runtime-tools.json
```

A test result should be meaningless unless it is bound to the exact artifact it claims to certify.

Resource cost: **zero runtime cost**.

Priority: **very high**.

---

## 5.3 Process ownership and fail-closed cleanup

Hermes repeatedly distinguishes:

- process exists;
- process belongs to this runtime;
- process can be safely stopped;
- cleanup is proven;
- replacement is allowed.

This is especially visible in its MCP/local-runtime/Linux tooling.

PocketClaw should adopt an explicit process-ownership contract for every managed child runtime:

```text
owner_id
pid/process handle
start timestamp
command fingerprint
working directory
expected listening endpoint
health state
cleanup state
```

If cleanup cannot be proved, fail closed instead of starting a duplicate runtime on top of an unknown process.

Resource cost: **low**.

Priority: **high**.

---

## 5.4 MCP supervisor security contract

Hermes Android's MCP design is one of the best ideas to mine.

Observed documented behavior:

- stdio transport;
- legacy SSE transport;
- Streamable HTTP transport;
- bounded connect/call/cleanup deadlines;
- 1 MiB pre-parse limits for HTTP JSON and individual SSE/stdio frames;
- additional schema/catalog/result limits;
- certificate verification for HTTPS;
- no redirects;
- no compressed responses in the documented implementation;
- credentials require HTTPS outside loopback;
- schema snapshots stored without credentials;
- existing conversations can retain a schema snapshot without granting stale execution authority;
- revoked/changed tools cannot execute merely because an old conversation remembers the schema;
- runtime replacement is blocked when cleanup cannot be proven.

PocketClaw MCP-X should separate:

```text
Schema visibility
from
Live execution authority
```

Possible flow:

```text
Discover schema
    -> sanitize/limit
    -> snapshot schema for conversation
    -> authorize transport/runtime
    -> tool call checks CURRENT authorization
    -> execute with bounded time/result
    -> redact result/log metadata
```

Do not copy Hermes' Python SDK runtime just because it exists. Implement the smallest equivalent contract that fits PocketClaw's Go/Flutter architecture.

Resource cost if implemented natively in Go: **low to moderate**.

Priority: **MCP-X, after the Stable foundation work**.

---

## 5.5 Provider authentication strategies, not API-key-only providers

Relevant Hermes Android files:

- `auth/CodexDeviceCodeAuth.kt`
- `auth/CodexLoopbackOAuthServer.kt`
- `auth/CodexOAuthClient.kt`
- `auth/OpenRouterLoopbackOAuthServer.kt`
- `auth/OpenRouterOAuthClient.kt`
- `auth/XaiLoopbackOAuthServer.kt`
- `auth/XaiOAuthClient.kt`
- `auth/NousDeviceCodeAuth.kt`
- `auth/AuthRuntimeApplier.kt`
- `data/ProviderPresets.kt`

PocketClaw PROVIDER-X should support a provider-auth strategy field such as:

```text
API_KEY
OAUTH_LOOPBACK
DEVICE_CODE
CUSTOM_HEADER
LOCAL_NO_AUTH
```

The provider preset should define the auth flow instead of hard-coding every provider into UI logic.

Resource cost: **low**.

Priority: **Provider-X**.

---

## 5.6 Content-addressed downloaded artifacts

Hermes restricts recommended/quick-start local-model entries to exact artifacts with known immutable identity and size.

Useful contract for PocketClaw downloadable runtimes, model files, tool payloads, or optional components:

```text
source repository / origin
immutable revision
filename
expected byte count
SHA-256
runtime compatibility
minimum platform/ABI
certification status
```

Do not present a moving `latest` artifact as "certified".

This idea is useful even if PocketClaw never ships local models.

Resource cost: **almost zero**.

Priority: **high for any downloadable component**.

---

## 5.7 Stable / Experimental / Custom runtime lanes

Hermes separates stable and experimental model/runtime paths rather than contaminating the stable lane to support every experiment.

PocketClaw future pattern:

```text
Stable
Experimental
Custom
```

A future RUNTIME-X feature should never require destabilizing the Core release path merely to support an experimental engine.

Resource cost: **architectural only**.

Priority: **when RUNTIME-X starts**.

---

## 5.8 Android-native tool bridge — narrow capabilities only

Hermes has a very large native Android device layer. Useful files include:

- `device/HermesAppControlBridge.kt`
- `device/HermesIntentBridge.kt`
- `device/HermesNotificationActionBridge.kt`
- `device/HermesAutomationBridge.kt`
- `device/HermesSystemControlBridge.kt`
- `device/HermesWorkspaceFileBridge.kt`
- `device/HermesClipboardActionBridge.kt`
- `device/HermesToastActionBridge.kt`
- `device/HermesVibrationActionBridge.kt`
- `device/RequestMutationGate.kt`

PocketClaw should **not** import the whole device layer.

If Android-native tools are added later, expose small capability packs:

```text
Basic
Files
Device
Automation
Advanced
```

Each capability should declare whether it is:

```text
READ_ONLY
MUTATING
PRIVILEGED
BACKGROUND
SENSITIVE_PERMISSION
```

Mutation should pass a central policy gate.

Examples of low-cost native tools worth considering later:

- battery/device information;
- open app/intent;
- clipboard read/write with clear permission/UX rules;
- share file;
- toast/vibration;
- safe notification metadata/action where policy allows.

Avoid making Accessibility, Tasker, sensors, watchers, overlays, privileged shells, etc. baseline PocketClaw requirements.

Resource cost: **low if capability-scoped; high if copied wholesale**.

Priority: **TOOLS-X, optional**.

---

## 5.9 Structured runtime activity timeline

PocketClaw should expose structured diagnostic events without exposing chat text, credentials, or full private filesystem paths.

Example:

```text
09:42:10 Telegram message received
09:42:10 Agent turn started
09:42:11 Provider request started
09:42:14 load_image x7
09:42:15 Provider response 200
09:42:16 Telegram reply delivered
```

This should be derived from real events, not UI guesses such as "thinking" when no runtime evidence exists.

Possible future event schema:

```json
{
  "ts": "...",
  "component": "telegram|agent|provider|tool|runtime",
  "event": "...",
  "status": "...",
  "duration_ms": 0,
  "safe_metadata": {}
}
```

Continue current PocketClaw log redaction policy:

- no API keys;
- no raw auth headers;
- no Telegram token;
- no full private Android `/data/user/...` or `/data/app/...` prefixes;
- no unnecessary chat content;
- preserve useful counts, component names, safe filenames, statuses, timings.

Resource cost: **low**.

Priority: **high**.

---

## 5.10 Terminal UX: copy the architecture idea, not the baggage

Hermes includes:

- `ui/terminal/TerminalScreen.kt`
- `device/NativeAndroidShellTool.kt`
- `device/HermesLinuxSandboxBridge.kt`
- `device/HermesLinuxSubsystemBridge.kt`
- `device/HermesTermuxPackageManager.kt`

It also has live Alpine/Debian instrumentation tests.

PocketClaw may benefit from a future terminal, but the default implementation should **not** imply bundling a distro or embedded Python.

Possible lightweight PocketClaw terminal modes:

```text
Runtime
Android (safe subset)
Logs
Optional Sandbox (downloaded separately)
```

Before implementing Terminal-X, determine whether a true PTY is needed:

- persistent interactive session;
- stdin streaming;
- resize events;
- Ctrl+C / signals;
- history;
- ANSI rendering;
- process ownership;
- suspend/resume behavior.

If a simple command runner is sufficient, do not pay the complexity cost of a full terminal emulator.

Resource cost: **low for a command console; moderate/high for full PTY + sandbox**.

Priority: **later**.

---

## 5.11 Memory-X: pluggable memory, not uncontrolled self-modification

Hermes upstream advertises a closed learning loop with:

- persistent memory;
- skill creation/improvement;
- session search;
- summarization;
- user modeling;
- skills standard integration.

Android fork files include:

- `device/HermesHindsightMemoryBridge.kt`
- `device/HermesHyMemoryBridge.kt`
- `ui/settings/LocalMemorySection.kt`

PocketClaw should first design a stable interface:

```text
MemoryBackend
  - SessionSummary
  - LongTermMemory
  - Search
  - Forget/Delete
  - Export
  - Health
```

Possible backend implementations later:

```text
BuiltIn
Hindsight-compatible
HyMemory-compatible
Custom
```

Do not let an autonomous memory/skill loop rewrite critical runtime behavior without explicit versioning, review, and rollback.

Resource cost: **depends on backend; interface itself low**.

Priority: **MEMORY-X**.

---

## 5.12 Full vs Play distribution split

Hermes explicitly builds a Full edition and a Play edition from the same source.

Reference:

- `android/PLAY_EDITION.md`

The Full edition keeps capabilities such as Linux/Python/automation/accessibility that the Play edition removes or disables.

Future PocketClaw direction, only if/when Google Play becomes a real target:

```text
PocketClaw Full
  GitHub / possible F-Droid
  broader tool/runtime capability

PocketClaw Play
  restricted tool/runtime capability
  reduced permissions/components
```

Important: the split must be a **real build/runtime boundary**, not a hidden menu item. Components, permissions, assets, and dispatch authority must actually differ.

Do not add this complexity to v0.2.0 Stable.

Resource cost: **build complexity, minimal runtime cost**.

Priority: **Distribution-X, later**.

---

# 6. What NOT to copy from Hermes by default

The following are useful references, but should not become PocketClaw baseline merely because Hermes has them:

1. Embedded Python runtime.
2. Bundled Linux distro/rootfs.
3. Bundled local-model engines unless RUNTIME-X explicitly requires them.
4. Universal arm64 + x86_64 heavy payload inside every APK if Play/AAB splits can deliver per-device assets later.
5. Accessibility service as a baseline permission.
6. Notification listener as a baseline permission.
7. Location/sensor/calendar/logcat watchers as baseline services.
8. Tasker integration by default.
9. Privileged shell/Shizuku-style surface by default.
10. Huge automation/device bridge monoliths.
11. Multiple always-alive helper processes.
12. Feature accumulation without APK/RAM/startup/battery budgets.
13. Forking Hermes architecture so deeply that PocketClaw becomes dependent on its internal changes.

Hermes has very large implementation units. Examples observed in the tree include very large diagnostics, automation, and native tool-chat files. PocketClaw should keep modules smaller and contracts clearer.

---

# 7. Resource-budget contract for every future feature

Before a future feature is accepted, report **before vs after**:

```text
APK/AAB size delta
installed size delta
idle PSS/RSS delta
active PSS/RSS delta
process count delta
cold-start time delta
background battery implications
new Android permissions/components
new native libraries
new persistent storage
new network listeners
new attack surface
```

Recommended workflow:

```bash
adb shell dumpsys meminfo com.lord1egypt.pocketclaw
adb shell ps -A | grep -i pocketclaw
adb shell dumpsys package com.lord1egypt.pocketclaw
```

Use Android Studio/Perfetto or equivalent only when a deeper measurement is necessary.

No future feature should be approved merely because it "works".

It must also justify its cost.

---

# 8. Suggested post-Stable roadmap

This is a **candidate roadmap**, not a commitment to build everything.

## Phase A — Foundation / Evidence-X

Goal: strengthen architecture without adding heavy features.

- unified readiness/status vocabulary;
- authoritative runtime snapshot;
- structured activity events;
- process ownership contract;
- release-evidence v2 bound to source/APK/signer/device;
- unexpected generated-input/reproducibility gate;
- resource-budget baseline measurements.

Expected footprint: **tiny**.

## Phase B — PROVIDER-X

- provider presets remain data-driven;
- auth strategy abstraction;
- API key / custom header / OAuth loopback / device-code support;
- explicit credential lifecycle and rotation tests;
- provider capability discovery only where useful.

Expected footprint: **small**.

## Phase C — MCP-X

- Go-native or otherwise lightweight MCP supervisor;
- stdio / Streamable HTTP first; add legacy SSE only if needed;
- bounded frame/result/schema limits;
- schema snapshots without credentials;
- current live authorization checked at execution time;
- explicit cleanup/process ownership;
- no automatic download of arbitrary MCP executable dependencies.

Expected footprint: **small/moderate**.

## Phase D — TOOLS-X

- small Android native bridge;
- capability packs;
- centralized mutation gate;
- permissions requested only when the enabled capability needs them;
- no broad privileged baseline.

Expected footprint: **small if scoped**.

## Phase E — TERMINAL-X

Start with a lightweight runtime/log console.

Only add full PTY semantics if a real use case requires them.

Any Linux sandbox should be optional/downloaded, not baseline.

Expected footprint: **small -> moderate depending on scope**.

## Phase F — MEMORY-X

- stable `MemoryBackend` interface;
- conversation/session summary;
- long-term memory/search;
- explicit delete/export semantics;
- optional experimental backends;
- no uncontrolled self-modifying runtime.

Expected footprint: **variable**.

## Phase G — RUNTIME-X / Local models (optional)

Only after resource budgets and real owner demand.

- stable/experimental/custom lanes;
- content-addressed model/runtime artifacts;
- health + completion readiness checks;
- model/runtime compatibility matrix;
- optional downloads;
- never bundle weights in base APK;
- avoid bundling both llama.cpp and LiteRT unless both have a real product case.

Expected footprint: **heavy if enabled, therefore optional**.

## Phase H — Distribution-X

If Play becomes a real requirement:

- Full vs Play product flavors;
- real component/permission/runtime separation;
- AAB delivery strategy;
- store-specific policy/evidence tests.

Expected footprint: mostly **build/release complexity**.

---

# 9. Architecture-mining workflow for future Hermes research

Do not browse Hermes asking "what features can we copy?"

Use this workflow instead:

### Step 1 — State the PocketClaw problem first

Example:

```text
Problem: MCP server cleanup can leave a stale process and duplicate port owner.
```

### Step 2 — Find how Hermes models the problem

Inspect only the relevant files/tests/docs.

### Step 3 — Extract the invariant

Example:

```text
Invariant: runtime replacement is forbidden unless ownership and cleanup of the previous runtime are proven.
```

### Step 4 — Design the smallest PocketClaw-native implementation

Do not port Kotlin/Python code if a small Go contract is enough.

### Step 5 — Define the resource budget

APK/RAM/process/permission/startup impact.

### Step 6 — Add a regression test before feature expansion

Prefer journey/contract tests over isolated implementation tests.

### Step 7 — Record provenance

In the implementation PR/commit or design note, record:

```text
Reference project
Reference commit/branch
Reference files
Pattern adopted
Code copied: YES/NO
PocketClaw-specific differences
```

Default should be:

```text
Code copied: NO
```

---

# 10. Hermes files worth studying next

When the current PocketClaw Stable release is finished, deep-dive these one at a time.

## Runtime/readiness

- `android/app/src/main/java/com/mobilefork/hermesagent/backend/HermesRuntimeManager.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/backend/HermesRuntimeService.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/ui/chat/ChatReadinessStrip.kt`

Questions:

- Who is authoritative for state?
- How are stale processes detected?
- How is readiness different from process existence?
- What is persisted vs recomputed?

## llama.cpp/local runtime controller

- `android/app/src/main/java/com/mobilefork/hermesagent/backend/LlamaCppServerController.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/backend/LlamaCppLaunchConfig.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/backend/GgufArtifactInspector.kt`

Questions:

- How does ownership work?
- What admission checks happen before launch?
- How are ports/workdirs/envs isolated?
- How is failure surfaced?

## MCP

- `android/app/src/main/java/com/mobilefork/hermesagent/data/McpRuntimeBridge.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/data/McpSettingsStore.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/ui/settings/McpSettingsViewModel.kt`
- locate the Python MCP supervisor implementation used by Android before drawing conclusions.

Questions:

- Transport lifecycle?
- Limits?
- Schema snapshot identity?
- Credential handling?
- Revocation semantics?
- Cleanup proof?

## Mutation/security boundary

- `android/app/src/main/java/com/mobilefork/hermesagent/device/RequestMutationGate.kt`
- corresponding mutation-gate tests.

Questions:

- What is considered mutation?
- Where is authorization decided?
- Can UI and model paths bypass the same gate?

## Memory

- `android/app/src/main/java/com/mobilefork/hermesagent/device/HermesHindsightMemoryBridge.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/device/HermesHyMemoryBridge.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/ui/settings/LocalMemorySection.kt`

Questions:

- Data model?
- Retrieval trigger?
- Memory size limits?
- Deletion/export semantics?
- How much of this is useful without adding a heavy service?

## Terminal

- `android/app/src/main/java/com/mobilefork/hermesagent/ui/terminal/TerminalScreen.kt`
- `android/app/src/main/java/com/mobilefork/hermesagent/device/NativeAndroidShellTool.kt`
- Linux subsystem/sandbox bridge files.

Questions:

- Real PTY or command-runner UI?
- stdin streaming?
- resize/signals/history?
- process lifetime?
- what requires the Linux payload versus Android shell only?

---

# 11. Tests worth borrowing conceptually

Hermes has test names that reveal useful contracts:

- `StartupOrderingInstrumentedTest`
- `ReadinessUiInstrumentedTest`
- `AuthSecureStorageInstrumentedTest`
- `FullMcpRuntimeInstrumentedTest`
- `LiveAlpineSandboxInstrumentedTest`
- `LiveDebianSandboxInstrumentedTest`
- `PlayPrivacyInstrumentedTest`
- `LiteRtLmModelMatrixInstrumentedTest`
- `LlamaCppModelMatrixInstrumentedTest`
- `NativeAgentToolAccessInstrumentedTest`
- `ReleaseDeviceEvidenceIdentity.kt`

PocketClaw principle:

> **Test contracts and user journeys, not only functions.**

This is consistent with the bugs already found in PocketClaw where source/unit logic looked correct but the fresh physical journey exposed lifecycle failures.

---

# 12. Reproducibility ideas to add to PocketClaw later

Hermes release work is strict about generated inputs and exact artifact identity.

Future PocketClaw gates can include:

- clean source requirement;
- no unexpected generated source/input files;
- no stale build outputs consumed as inputs;
- exact dependency/runtime versions;
- exact signer assertion;
- exact ABI inventory;
- exact native ELF audit;
- exact asset manifest;
- exact downloadable payload hashes;
- immutable release evidence after publication.

This extends the existing PocketClaw Source/Artifact/ELF/Zero-Pico gate model rather than replacing it.

---

# 13. Sources / references

## Hermes fork / upstream

- Fork: https://github.com/adybag14-cyber/hermes-agent
- Upstream: https://github.com/NousResearch/hermes-agent
- Branch compare UI used during discussion: https://github.com/adybag14-cyber/hermes-agent/compare/NousResearch%3Ahermes-agent%3Amain...codex/termux-five-goals
- Reverse compare: https://github.com/adybag14-cyber/hermes-agent/compare/codex/termux-five-goals...NousResearch%3Ahermes-agent%3Amain

## Key documentation

- Root README: https://github.com/adybag14-cyber/hermes-agent/blob/codex/termux-five-goals/README.md
- Android README: https://github.com/adybag14-cyber/hermes-agent/blob/codex/termux-five-goals/android/README.md
- Play edition: https://github.com/adybag14-cyber/hermes-agent/blob/codex/termux-five-goals/android/PLAY_EDITION.md
- Release evidence v3: https://github.com/adybag14-cyber/hermes-agent/blob/codex/termux-five-goals/android/RELEASE_EVIDENCE_V3.md
- Latest reviewed release: https://github.com/adybag14-cyber/hermes-agent/releases/tag/v0.13.157

## Code tree

- Android app tree: https://github.com/adybag14-cyber/hermes-agent/tree/codex/termux-five-goals/android/app/src/main

Before implementation, pin a specific Hermes commit SHA and use permanent commit URLs instead of relying on a moving branch.

---

# 14. New-chat resume procedure

If the current ChatGPT conversation reaches its context limit, start a new chat and say something equivalent to:

```text
We are continuing PocketClaw.
Read docs/research/HERMES_ARCHITECTURE_REFERENCE.md from the
Lord1Egypt/PocketClaw repository, branch docs/hermes-architecture-notes.
Use it as the architecture-research checkpoint.

Before proposing or changing anything:
1. inspect the current PocketClaw branch/HEAD and defect/release ledger;
2. inspect local git status because the previous Claude session may have
   uncommitted Zero-Pico/path-redaction work;
3. do not modify the Stable release path merely to add future Hermes-inspired features;
4. re-fetch the current Hermes fork state before relying on branch counts or implementation details;
5. continue architecture mining from the "Hermes files worth studying next" section.
```

If this document has later been merged to `main`, use the `main` copy instead of the documentation branch.

---

# 15. Decision summary

The Hermes fork is valuable to PocketClaw mainly as an **architecture laboratory**.

The highest-value ideas are:

1. authoritative readiness/state contracts;
2. artifact-bound release evidence;
3. process ownership and fail-closed cleanup;
4. MCP schema/execution separation and bounded transports;
5. provider auth strategies;
6. content-addressed downloadable artifacts;
7. stable/experimental runtime lanes;
8. narrowly-scoped Android-native capability packs with mutation policy;
9. structured privacy-safe runtime activity;
10. Full-vs-Play build boundaries;
11. memory backends behind a stable interface;
12. journey/contract testing and reproducibility gates.

The main anti-goal is equally important:

> **Do not let PocketClaw become a 300+ MiB all-in-one mobile workstation unless the product actually needs that.**

Keep the base small. Add heavy capability only on demand. Measure every addition.
