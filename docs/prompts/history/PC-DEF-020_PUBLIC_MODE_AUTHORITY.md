# OPERATING RECORD — PC-DEF-020 Public Mode authority fix

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `76064c91033860653de1e11a08e29d7245061ec1` (exposure-audit closeout).
- **Source / build-input commit:** `f8bc52a0757f7b0a9f6c0704d2a3586db929e33f`.
- **Staging / closeout commit:** the following commit carrying the rebuilt Core
  pair and this record, touching no Core build input.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.

Scope was `PC-DEF-020` only. `PC-DEF-021`, `PC-DEF-022`, `PC-DEF-023`,
`PC-DEF-024` and `PC-DEF-025` are untouched and remain open. No production
signing material was requested or accessed, no APK or AAB was built, no Play key
was created, no device or ADB was used, and nothing was merged, tagged,
published, versioned or advanced.

## The defect, stated exactly

Two persisted authorities, no rule for which won.

The Android host keeps the user's Public Mode choice in SharedPreferences
`public_mode` and passed `-public` **only when it was on**. With the flag
absent, `web/backend/main.go` took `effectivePublic = launcherCfg.Public` — the
`public` field of `launcher-config.json`. Nothing on the Android OFF path ever
wrote that file: the network-mode bridge and `launcherHTTPRuntime.ApplyPublicMode`
rebind the live listener and update in-memory state only, and
`PUT /api/system/launcher-config` is its single writer. The dashboard's own
Config page does send `public`, so saving it while LAN access was on persisted
`true`.

The sequence that broke: enable LAN access → save the Config page → switch the
native toggle off. The live listener moved to loopback for the life of that
process and the stored `true` remained, so the next service start — app restart,
service kill, device reboot — bound the console to every interface while the
native toggle still read OFF.

Enforcement was never the failure. The dashboard password wall held in every
state and the Core gateway on 18790 was loopback-only throughout. Durability
failed, and that is all this milestone changed.

## Authority model

    BEFORE
      listener  = -public if supplied, else launcher-config.json `public`
      Android   = supplies -public only when ON; OFF is silence
      => OFF falls through to a field the Config page can set to true
      Config page reads and writes that field directly and unconditionally

    AFTER
      listener  = -public if supplied, else launcher-config.json `public`   (unchanged rule)
      Android   = always supplies -public=true or -public=false
      => the persisted field is never consulted on Android
      Config page reports the EFFECTIVE mode and persists that, not the
      submitted value, whenever the host owns the decision

The resolution rule itself is deliberately unchanged. What changed is that the
host now always exercises it, so the fallback branch is unreachable on Android
rather than merely discouraged.

## Contract

`-public=<bool>`, always supplied by the Android host.

An explicit `false` and an omitted flag are distinguishable because Go's
`flag.Visit` reports only flags that were `Set`; the parsed value is `false` in
both cases, so a value comparison could not tell them apart. The existing
backend already used `flag.Visit`, so no new mechanism was needed and no
tri-state was introduced — the smallest change that satisfies the requirement
was on the Android side. A test pins the distinction, because the whole fix is
inert without it.

`-public` bare still means on, so an existing desktop command line keeps
working.

## What changed

    android/.../service/PocketClawService.kt          always states the decision
    core/src/web/backend/main.go                      launcherExplicitFlags,
                                                      resolveLauncherPublicMode
    core/src/web/backend/api/gateway_host.go          launcherPublicDecisionIsHostOwned,
                                                      livePublicMode, and
                                                      effectiveLauncherPublic now
                                                      prefers a runtime rebind
    core/src/web/backend/api/launcher_config.go       reportedLauncherPublic on
                                                      GET and PUT

`main.go`'s inline resolution and its `flag.Visit` block became named functions
so the state matrix could be tested rather than argued about; the behaviour they
encode is identical.

`effectiveLauncherPublic` already encoded the precedence — host override, then
explicit flag, then stored value — and had **no product caller**, only a test.
It was wired up rather than duplicated, and extended to prefer the live rebind
controller over the startup flag, because `ApplyPublicMode` replaces the
listeners without rewriting `serverPublic`; without that the page would report a
value that went stale the moment the toggle was used.

The frontend was not touched. Under a host-owned decision the Config page's
field is informational by virtue of what the API reports, which is narrower than
disabling a control in the UI and does not redesign Settings.

## State matrix

| Case | Inputs | Result | Where proved |
| --- | --- | --- | --- |
| A | fresh install, native off, no launcher config | loopback | `TestResolveLauncherPublicMode_StateMatrix` |
| B | native on | public bind | same |
| C | native on, console saved `public:true`, restart still on | public bind | same |
| D | native on with stored `true`, native switched off — live | loopback | `TestLauncherHTTPRuntimeAppliesPublicModeWithoutReplacingHandler` (pre-existing) |
| E | same, then service restart | loopback | `TestResolveLauncherPublicMode_StateMatrix` |
| F | same, then process recreation | loopback | same |
| G | stored `public:true` already on disk before startup, native off | loopback | same, and `TestStaleStoredPublicCannotOpenAWildcardListener` |
| H | stored `public:false`, native on | public bind | `TestResolveLauncherPublicMode_StateMatrix` |
| I | explicit host override, all three public states | host wins, deterministic | `TestExplicitHostOverridesEveryPublicDecision` |
| J | gateway 18790, every Public Mode state | loopback | `TestGatewayStaysLoopbackInEveryPublicModeState` |

E and F are the same question — what does a fresh process decide from these
inputs — because the resolution is a pure function of the flag and the stored
value. Answering it once answers it for a service restart, an app restart and a
process recreation alike; the test says so explicitly rather than leaving a
reader to infer it.

G is also proved end to end rather than only on the resolved boolean:
`TestStaleStoredPublicCannotOpenAWildcardListener` resolves the decision and
then opens real listeners, asserting every bind host and every bound socket
address is loopback.

Config-page consistency is covered separately in
`core/src/web/backend/api/launcher_config_authority_test.go`: the page reports
the effective mode when the host owns it, follows a runtime rebind in both
directions, cannot override a host-owned decision, **repairs** a stale stored
`true` on save, reports off under an explicit host, and stays fully writable on
desktop where no host owns the decision.

The host's half is covered by
`test/unit/android_public_mode_authority_test.dart`: the service always states
the decision, never emits a bare `-public`, does not branch on `publicMode` when
building the argv, and carries the reason in place.

## No security regression

Nothing in the authentication or transport boundary was touched. Unchanged:
session handling and the dashboard password wall, the unauthenticated path
allowlist, the realtime WebSocket's session-plus-origin requirement, the
auth-path canonicalization that blocks `/assets/../` traversal, the Android
bridge's loopback-plus-token gate, the gateway's unconditional loopback pin in
`gatewayHostOverride`, and Public Mode ON semantics.

`gateway_host.go` is in the diff, but only the launcher-public helpers; the
gateway's own host resolution and its loopback pin are byte-unchanged.

Suites re-run: `web/backend/middleware`, `web/backend/dashboardauth`,
`web/backend/api`, `web/backend`, `web/backend/launcherconfig`, `pkg/netbind`
and `pkg/gateway` — all pass.

## Core rebuild and re-stage

`core/src` changed, so this milestone triggered the two-commit rule.

    source fingerprint   bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec
                      →  2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df
    build-input commit   f8bc52a0757f7b0a9f6c0704d2a3586db929e33f
    BuildTime            2026-09-12T18:54:26+0000

    libpocketclaw.so       37,724,640  602ce03439f0038e0b827d75a6a652e5ef7d82da3f1fe49064d095e111890292
                                       build ID ed130bedb68592ba924e4f5ca43db0f75321e2de
    libpocketclaw-web.so   25,517,952  b5cce071f0fef2b03c58d2a4e9c26572b567171adeb27238bc481792b284283c
                                       build ID f61a369ffa4df0afcb5a7c2017cc98218a6441cb

Both produced byte-identically in the canonical in-repository run and two
further independent output roots under different `CORE_BUILD_DIR`, `JNI_LIBS`
and `NATIVE_SYMBOL_ROOT` paths, the third with a cold `GOCACHE`. Both private
companions likewise:

    libpocketclaw.so.debug       14,545,360  a42fb4310dd1666122545909abfa2cf8dab2ae0352a4241ba08eca9ca4a45d4f
    libpocketclaw-web.so.debug    9,766,312  096f50c22211a31e22f18fa21957c06cddc06d71fca3f1b5974b582876be5353

Native contract for the pair: **22 PASS / 0 FAIL** under the repository's own
`inspect_elf` / `evaluate_record`, driven directly because no APK was built.
`PT_LOAD` alignment `0x10000`, `GNU_STACK=RW`, no W+X, no TEXTREL, no
RPATH/RUNPATH, stripped, exports exactly `main.main`, zero prohibited path
strings. No Managed Runtime payload was rebuilt and their eight companions still
carry their H5B mtimes and hashes.

The private support manifest's `apk` field still names the exposure audit's
local-test APK, which no longer contains this pair. That is the established
pre-artifact state — `bind-apk` is a separate step against a candidate artifact,
and no artifact was built here — exactly as in `PC-DEF-019`.

## Gates and tests

    tool/release_gate.py --verify-source (test and production class)   25 PASS / 0 FAIL / 0 SKIPPED
    Core Go suite (go test ./...)                                      98 packages ok, 0 failed
    pkg/coresource (freshness + reproducibility)                       46 tests, 0 failed
    TestBundledPayloadsMatchTheirPinnedChecksums                        PASS
    TestStagedCoreEmbedsTheCurrentCatalog                               PASS
    TestStagedCoreWasBuiltFromTheCurrentSource                          PASS
    Native contract, Core pair                                         22 PASS / 0 FAIL
    Flutter, this milestone's new file                                   4 tests passed

`flutter test` as a whole remains **480 passed, 1 failed** — the stale assertion
in `namespace_n3_native_identity_test.dart` tracked as `PC-DEF-025`. It is
unrelated to this fix, predates it by four milestones, and is deliberately not
touched here. It is reported rather than hidden.

## Closeout conditions

1. Native OFF is authoritative across restart — matrix E, F, G.
2. A stale `public: true` cannot reopen LAN binding — matrix G plus
   `TestStaleStoredPublicCannotOpenAWildcardListener`, and a save now repairs it.
3. Native ON still enables LAN binding — matrix B, C, H.
4. Gateway remains loopback-only — matrix J.
5. Authentication behaviour unchanged — no file in that boundary touched; suites pass.
6. Automated restart and persistence coverage exists — three new test files.
7. Staged Core is fresh — `TestStagedCoreWasBuiltFromTheCurrentSource` passes.
