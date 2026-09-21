# OPERATING RECORD — Final release exposure audit, closure re-run

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

This record does not replace
[`EXPOSURE_AUDIT.md`](EXPOSURE_AUDIT.md); that remains the evidence for the
first, blocked run. This is the second pass against the repaired tree.

## Status and boundary

- **Status:** **PASS — CLOSED. No release blocker remains** for the intended
  GitHub / direct APK stable release.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `60318980e86144052d96bf5e59cfd611e4b84470` (PC-DEF-025 closeout).
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.
- **Core fingerprint:** `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df`,
  build-input commit `f8bc52a0757f7b0a9f6c0704d2a3586db929e33f`.

Audit-first. No defect was fixed, no production signing material was requested
or accessed, no production candidate was built, no Play upload key was created,
no device or ADB was used, and nothing was merged, tagged, published or
released. `PC-DEF-006` and `PC-DEF-012` are deliberately unchanged.

## What the first run blocked on, and what changed

    PC-DEF-020  Public Mode OFF not durable across a restart   RESOLVED (f8bc52a / 69479b4)
    PC-DEF-021  AAB embeds the R8 mapping and Dart symbols     RESOLVED (3e3941f)
    PC-DEF-025  flutter test red; no gate ran the suite        RESOLVED (6031898)

All three were re-validated from the current tree rather than taken on trust.

## Fresh artifacts

Both built from this tree by the canonical helper, both development-signed:

    APK — LOCAL TEST / NON-RELEASABLE, artifact-class public-release
      bytes   63,560,039
      sha256  113a8382e2014ad19461a0898f3bfe208a7e1473742483099a028f741b11001b

    AAB — LOCAL TEST / NON-PUBLISH, artifact-class non-publish-audit
      bytes   74,028,994
      sha256  d45efcbf9d8b8d6d7a38893771e9a93f348415b217e7064293935b3345d94612
      notice  NOT PLAY-READY; NOT PUBLIC-RELEASE-SAFE; NOT A GITHUB RELEASE ASSET

Both carry the current staged Core, and the Managed Runtime is byte-unchanged
from H5B/H5C:

    libpocketclaw.so       37,724,640  602ce034…  build ID ed130bed…
    libpocketclaw-web.so   25,517,952  b5cce071…  build ID f61a369f…
    libapp.so (Dart AOT)    5,702,536  c7b2a885…

`artifact.core_provenance_pair` reports fingerprint `2692de41…` from **both**
Core binaries and `artifact.core_matches_staged` reports identical. All eight
Managed Runtime payloads match their established hashes exactly.

## PC-DEF-020 re-validated

Source: the Android host still states the decision unconditionally
(`cmdList.add("-public=" + publicMode)`), `resolveLauncherPublicMode` still
returns the flag whenever it was supplied, and the gateway still passes
`netbind.DefaultLoopback` with `gatewayHostOverride` pinned to `localhost`.

Live bind evidence, re-run through the shipped `pkg/netbind` with the same
default-mode selection `openLauncherListeners` applies:

    PUBLIC OFF (no -public, no host)    bindHosts=[::1 127.0.0.1]
    PUBLIC ON  (-public, no host)       bindHosts=[:: 0.0.0.0]
    host override 127.0.0.1 + -public   bindHosts=[127.0.0.1]
    gateway (host=localhost)            bindHosts=[::1 127.0.0.1]  in BOTH states

94 matching tests pass across `web/backend`, `web/backend/api`,
`web/backend/middleware`, `web/backend/dashboardauth`,
`web/backend/launcherconfig`, `pkg/netbind` and `pkg/gateway`; all seven
packages green.

The authentication boundary is unchanged. The unauthenticated surface is still
exactly `POST /api/auth/{login,logout,setup}`, `GET /api/auth/status`, and
GET/HEAD of the login/setup SPA routes, `/assets/`, the favicons,
`site.webmanifest` and `robots.txt`. The realtime WebSocket still requires a
live session **and** an origin check. **PC-DEF-020 remains CLOSED.**

## PC-DEF-021 re-validated

APK: `artifact.r8_mapping_private` reports `496 archive entries scanned,
deobfuscation entries = 0` — derived from the real archive.
`artifact.dart_symbols_private` external and untracked. No native `.debug`
companions packaged. All thirteen private/residue categories CLEAN.

Bundle classification, against the fresh AAB:

    --artifact-class public-release       exit 1   FAIL (required)
    --artifact-class play-upload          exit 0   PASS, metadata inventoried
    --artifact-class non-publish-audit    exit 0   PASS, notice emitted
    (no --artifact-class)                 exit 2   fail closed

Public release asset allowlist: `PocketClaw-v0.2.0-arm64.apk` + `SHA256SUMS.txt`
accepted (exit 0); `PocketClaw-v0.2.0.aab` rejected (exit 1).
**PC-DEF-021 remains CLOSED.**

## PC-DEF-025 re-validated

`flutter analyze`: No issues found. `flutter test`: **490 passed, 0 failed**,
"All tests passed". The release gate reports
`PASS flutter.suite  490 passed, 0 failed` — the full suite, not the historical
three-file path. **PC-DEF-025 remains CLOSED.**

## Secrets and entropy

1,840 tracked text files scanned against 23 credential patterns: 90 candidates,
resolving to 72 placeholders, 17 test fixtures and **one** needing review —
`channel-config-fields.ts:9`, the `SECRET_FIELD_MAP` that maps field names to
suffixes and holds no values. Unchanged from the first run and still clean.

Entropy pass over release-relevant source at ≥ 4.6 bits/char found **four**
literals, identical to the first run and with no new candidates: two random-ID
alphabet constants (`wecom.go:891`, `antigravity_provider.go:653`) and the two
`pkg/auth/oauth.go` values that are `PC-DEF-023`.

## Artifact residue and path leakage

APK: zero hits across Dart private symbols, R8 mapping, native debug
companions, keystores/keys, `.env`, VCS metadata, CI metadata, source maps,
build logs, shell scripts, test fixtures, developer documentation and temporary
files.

AAB: the same, with two expected exceptions correctly classified by the
artifact-class policy rather than by an ad-hoc scanner —
`BUNDLE-METADATA/com.android.tools.build.obfuscation/proguard.map` (AGP Play
metadata, allowed for `play-upload`, and the bundle is never publishable) and
`base/res/drawable/abc_vector_test.xml` (an AndroidX AppCompat resource matching
a `_test.` pattern).

Path leakage: **zero** occurrences of `lordegypt`, `PocketClaw-App`,
`pocketclaw-runtime-build`, `/mnt/c/` or `/tmp/pocketclaw` in any packaged
entry. The only path strings are in `libpocketclaw-gh.so` — one
`/home/runner/work/` fragment from upstream `cli/cli`'s own GitHub Actions
build, and `/root/` inside Go package paths such as
`github.com/cli/cli/v2/pkg/cmd/root/`. Both are upstream content, carry no
PocketClaw builder identity, and are already documented in
`tool/native_elf_audit.py`'s own comment.

## Dashboard

UI-1 present by content: `__pocketclaw_tour_probe__` occurs once in
`libpocketclaw-web.so`; the removed docs-step copy and its minified locale key
occur zero times. No source maps, no `@vite/client`, no `vite/client`, no
`react-refresh`, no `sourceMappingURL`, no `process.env`, no `NODE_ENV`. No
embedded credentials (`GOCSPX`, `sk-ant`, `ghp_`, `AIza`, PEM headers all zero)
and no developer paths. The one marker present is
`__REACT_DEVTOOLS_GLOBAL_HOOK__`, which React's production build contains.

## Android manifest

From the fresh merged manifest via `aapt2 dump xmltree`:

    android:debuggable      absent
    android:testOnly        absent
    profileable             absent
    usesCleartextTraffic    absent (networkSecurityConfig governs; loopback only)
    allowBackup             true, with fullBackupContent + dataExtractionRules
    extractNativeLibs       true (required by the Android exec model)

Thirteen permissions, matching `artifact.permissions`, with
`artifact.forbidden_permissions_absent` clean and
`artifact.backup_exclusions` confirming `credentials/`, `pocketclaw-core/` and
`picoclaw/`. `MANAGE_EXTERNAL_STORAGE` remains the broadest and remains a
recorded product decision.

One deep link is present: `android:scheme="um.placeholder"` — `PC-DEF-024`,
reported explicitly and not fixed here.

## Native, Dart, R8

`tool/native_elf_audit.py --enforce-target --native-support-manifest`:
**194 PASS / 0 FAIL / 0 SKIP**, 18 packaged ELF entries inventoried and
category-classified. Correct architecture and type per role, NX everywhere, no
writable+executable segment, no TEXTREL, no RPATH/RUNPATH, 16 KiB-compatible
alignment, no debug sections, stripped payloads, Core exporting exactly
`main.main`.

The private native support manifest was **rebound to the fresh audit APK**
(`113a8382…`) and is not left on an older artifact.

Dart and R8 unchanged: AOT `c7b2a885…` and private DWARF `0f52873b…` are
byte-identical to H3A onward; the R8 mapping is `1cf56ffe…`, which moved at
`PC-DEF-020` because `PocketClawService.kt` changed and has been stable since.
Obfuscation, controlled generated URI, external split-debug-info, minification,
shrinking and the four preserved entry points all pass.

## Network exposure

164 distinct endpoints in release-relevant source, 21 of them loopback. Every
plain-`http://` destination is loopback, a format-string template, or the
`http://www.apple.com/DTDs/PropertyList-1.0.dtd` **XML doctype identifier** in
the macOS LaunchAgent plist writer — a doctype string, not a fetch. There is no
plain-HTTP Internet destination, no staging host, no tunnel
(ngrok/localtunnel/serveo/trycloudflare), and no developer or private-range IP.

The remaining endpoints are provider API bases and their console pages, plus
`api.github.com` for skills and `clawhub.ai` for the skill registry, all HTTPS.
The `pkg/updater` endpoints pointing at upstream `sipeed/picoclaw` and the
third-party fork `sky5454/picoclaw` remain, and are `PC-DEF-022`.

## Open defects, re-confirmed

**`PC-DEF-022` — `/api/update`.** Registered unconditionally
(`router.go:120`); **not** in the unauthenticated allowlist, so a live dashboard
session is required; still takes a caller-supplied `req.URL`; both zip and tar
extraction paths still carry their traversal guards; and **no** PocketClaw
UI calls it — zero references across the frontend, Flutter and Kotlin. On
Android `os.Executable()` is in the read-only install directory, so the apply
step still cannot succeed. What remains is an authenticated arbitrary-URL fetch
with extraction to a temporary directory, on a route the product never uses.
**Classification unchanged; not a release blocker** — it requires the dashboard
password and cannot achieve code execution on Android.

**`PC-DEF-023` — third-party Google OAuth credential.** Still live product
surface (`web/backend/api/oauth.go:552`,
`providers/oauth/antigravity_provider.go:464`). Still base64-wrapped: the
encoded form appears once in each Core binary and the decoded form zero times.
The source comment still records its origin as the OpenCode antigravity plugin,
i.e. a credential registered to a third party's Google Cloud project. It is an
installed-app OAuth client, a class RFC 8252 and Google's desktop-client model
treat as non-confidential, so the real exposure is third-party
revocation/dependency rather than secrecy. No PocketClaw or user secret is
disclosed. **Classification unchanged; not a release blocker.**

**`PC-DEF-024` — dead analytics deep link.** `um.placeholder` confirmed in the
fresh merged manifest. `POCKETCLAW_ANALYTICS_PROVIDER` still defaults to
`none`, so the Umeng SDK is not packaged, and `logIncomingIntent` still only
writes the URI to logcat. `MainActivity` is already LAUNCHER-exported, so the
added capability is web-originated launch plus attacker-controlled text in a log
other apps cannot read. **Classification unchanged; not a release blocker.**

**`PC-DEF-006` stays OPEN.** This audit built one APK and did not build and
compare two independent final builds, so its acceptance criterion is untouched.
One observation worth recording without overstating it: the APK built here
(`113a8382…`) and the one built at `PC-DEF-021` (`7155de0a…`) differ, while the
Dart AOT and R8 mapping between them are byte-identical. The earlier APK was
overwritten by `--clean`, so no byte-level comparison was possible; the differing
hashes with identical compiled inputs are consistent with packaging
non-determinism and are exactly why `PC-DEF-006` remains open.

**`PC-DEF-012` stays OPEN.** No new reachability evidence was produced and no
export was narrowed; `libdartjni.so` still exports 313 symbols and the CPython
payload 2,261, as H5A recorded.

## Gates and tests

    tool/release_gate.py --verify-source (test class)         26 PASS / 0 FAIL / 0 SKIPPED
    tool/release_gate.py --verify-source (production class)   26 PASS / 0 FAIL / 0 SKIPPED
    tool/release_gate.py --full <apk> --artifact-class public-release
                                                              57 PASS / 0 FAIL / 0 SKIPPED
    tool/native_elf_audit.py --enforce-target                194 PASS / 0 FAIL / 0 SKIP
    flutter analyze                                           No issues found
    flutter test                                             490 passed, 0 failed
    frontend (vitest)                                         25 files, 418 tests passed
    Core Go suite (go test ./...)                             98 packages ok, 0 failed
    release-tool suites (8 files)                            139 tests, all OK

Nothing was suppressed, reclassified or skipped. The artifact gate ran in
**test** class because the artifact is development-signed; it reports
`releasable: false` and classifies signing as LOCAL TEST / NON-RELEASABLE, which
is correct and is not a production result.

## Release-blocker decision

**No release blocker remains for the intended GitHub / direct APK stable
release.**

The four open defects were each assessed against that target rather than waved
through:

- `PC-DEF-022` requires dashboard authentication and cannot execute code on
  Android. Real attack surface that should be removed, not a blocker.
- `PC-DEF-023` discloses no PocketClaw or user secret; the risk is dependency on
  a third party's credential.
- `PC-DEF-024` adds a web-originated launch vector to an activity that is
  already externally launchable.
- `PC-DEF-006` is an F-Droid reproducibility requirement, a different
  distribution path from the GitHub APK. It is not down-ranked because the
  release is GitHub-only — it simply does not gate this path, and it stays open
  for the path it does gate.
- `PC-DEF-012` is an unchanged, evidence-pending hardening item with no
  demonstrated failure.

One thing this audit does **not** establish, stated plainly: the candidate
inspected here is development-signed. H5C proved that the production and
development builds of one tree differ only by the signing block, so the
structural, packaging and exposure findings transfer — but the production-signed
candidate itself has not been built or gated under `--release-class production`.
That is the next milestone, not a defect this audit found.

## New defects

None. No new defect was opened by this re-run.
