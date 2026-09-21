# OPERATING RECORD — Final release exposure audit

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **BLOCKED.** Two release blockers proven: `PC-DEF-020` and
  `PC-DEF-021`. The audit itself ran to completion; the milestone cannot close.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `25753cef5fa4d956e11d37b5a6176cdef977f015` (PC-DEF-019 closeout).
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.

Audit-first milestone. Six defects were opened and none was fixed: every one
carries a narrow fix plan and waits for its own authorization. No device, no
ADB, no install, no merge, no tag, no release, no publication, no Play upload,
no F-Droid submission. No production signing material was requested or
accessed. `PC-DEF-006` and `PC-DEF-012` are deliberately unchanged.

## Why no production-signed candidate was built

The owner signing ceremony was **not requested**. `PC-DEF-020` and `PC-DEF-021`
are proven release blockers, so any production candidate built now must be
rebuilt after they are fixed, and its hash, its native-support binding and its
gate evidence would all be superseded. Asking the owner to unlock the
production keystore to produce evidence that is already known to be disposable
is not a reasonable use of a one-way ceremony.

Everything that does not depend on the signing identity was completed instead,
against a fresh **LOCAL TEST / NON-RELEASABLE** APK built from this exact
source tree by the canonical helper. H5C established that the production and
development builds of one tree differ only by the signing block — all 18
packaged ELF entries, the Dart AOT, the private DWARF and the R8 mapping were
byte-identical across that pair — so the structural, packaging and exposure
findings here transfer to the production candidate. The signer itself does not,
and is not claimed.

## Artifacts

    FRESH APK — LOCAL TEST / NON-RELEASABLE (structural audit artifact)
      path    build/app/outputs/apk/release/app-release.apk
      bytes   63,560,203
      sha256  5460d86a80d74219a39554e4da2ceb819c708c32050789396c95e394c533346a
      package com.lord1egypt.pocketclaw   version 0.2.0 (62)
      entries 496

    STRUCTURAL AUDIT AAB — debug-signed, NON-PUBLISH AUDIT ARTIFACT
      path    build/app/outputs/bundle/release/app-release.aab
      bytes   74,029,175
      sha256  ea8a3d8239b0ef8fdbb92667fa97717389dc644d2e664cf43aa4d3d5944c617b
      modules base (single module; no dynamic feature or asset packs)

The AAB is **not** Play-ready and must not be described as such: no Google Play
upload key exists, none was created, and the developer app-signing key was not
repurposed as one. The repository has no AAB build or inspection path — the
canonical helper only runs `:app:assembleRelease`, and neither the release gate
nor the native audit accepts a bundle — so this bundle was produced by invoking
`:app:bundleRelease` directly with the same hardening properties the helper
passes, under `-PallowDebugSigning=true`. That absence is recorded in
`PC-DEF-021`.

## Core and runtime identity in the fresh artifacts

Both artifacts carry the post-UI-1 / post-`PC-DEF-019` Core pair, and the
Managed Runtime is byte-unchanged from H5B/H5C:

    libpocketclaw.so       37,724,640  f273b9ced85f4d00cb542df9c2f4c691b4151526cb0ac9c2c7612a1432d7230f
    libpocketclaw-web.so   25,517,952  900c43fcaad2094017c6959eed623d1e2499cfd560f01f2cff36dd34202b86b9
    libapp.so (Dart AOT)    5,702,536  c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77

`artifact.core_provenance_pair` reports source fingerprint
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` from **both**
Core binaries, `artifact.core_matches_staged` reports identical, and
`artifact.build_time` reports `2026-09-12T07:27:12+0000`. All eight Managed
Runtime payloads match their H5B hashes exactly. Packaged ABIs are arm64-v8a
(14 native entries) plus the accepted `armeabi-v7a` / `x86_64` plugin ABI stubs
(2 entries each) — the same shape in the APK and in the AAB's `base/` module.

## Private material: absent from the APK, present in the AAB

The APK is clean. Scanned for Dart private symbols, R8 mapping, native `.debug`
companions, keystores and key files, `.env` files, VCS metadata, CI metadata,
source maps, build logs and crash dumps, shell scripts, test fixtures,
developer documentation and temporary files: **zero hits in all thirteen
categories.**

The AAB is not. `BUNDLE-METADATA/` carries ~39.5 MB of release-support
material, including `proguard.map` byte-identical to the private `mapping.txt`
(`14d49fad46e773e3…`) and native debug symbols for the obfuscated Dart AOT
library. This is AGP's intended bundle design and is correct for a Play upload;
it is wrong for a public release asset, and publishing the AAB as a GitHub
release asset is this project's established practice. Tracked as `PC-DEF-021`.

## Host and build-path leakage

No packaged entry carries a developer path. Two substring hits in
`libpocketclaw-gh.so` resolve to expected upstream content and are already
documented in `tool/native_elf_audit.py`'s own comment: one `/home/runner/work/`
fragment from upstream `cli/cli`'s GitHub Actions build, and `/root/` inside Go
package paths such as `github.com/cli/cli/v2/pkg/cmd/root/`. Zero occurrences of
`lordegypt`, `PocketClaw-App`, `pocketclaw-runtime-build`, `/mnt/c/` or
`/tmp/pocketclaw` in that payload or any other. The AAB adds only `/root/`
inside its JAR signature manifests.

## Source secrets

1,831 tracked text files scanned against 23 credential patterns. 90 candidates,
every one resolved without a real secret: 50 obvious placeholders, 20 test
fixtures, 19 upstream documentation examples using `your-*` values, and one
`SECRET_FIELD_MAP` in the dashboard frontend that maps field names to suffixes
and holds no values. No tracked keystore, key, certificate private half or
`.env` with content; `core/src/.env.example` holds one timezone and
`android/gradle.properties` holds only JVM and AndroidX flags. `.gitignore`
covers `android/local.properties`, `.env`, the private symbol tree, the R8
mapping and `*.jks`. The enrolled certificate digest is public metadata by
design, correctly committed, and is not a secret.

A separate entropy pass over release-relevant source found what the pattern
pass could not: a base64-wrapped Google OAuth **client secret** in
`pkg/auth/oauth.go`, present in both Core binaries in encoded form only.
Tracked as `PC-DEF-023`. The audit records that its own pattern scan missed it
and that the wrapper is why.

## Web dashboard

UI-1 is present, proved by content rather than timestamp:
`__pocketclaw_tour_probe__` occurs in the embedded bundle and in
`libpocketclaw-web.so`; the deleted docs-step copy and its minified locale key
occur zero times in either. Bundle residue: **no source maps**, no
`@vite/client`, no `vite/client`, no `react-refresh`, no `NODE_ENV` or
`process.env`, no `sourceMappingURL`, no `eval(`. Four markers matched and all
four are benign — `localhost:5173` is placeholder text in the "Allowed Origins"
form hint across all fourteen locales, `__vite__mapDeps` is Vite's production
dynamic-import helper, `__REACT_DEVTOOLS_GLOBAL_HOOK__` is in React's
production build, and `debugger` appears inside a minified library's JS keyword
table. No embedded credentials or environment values: the `xoxb-` hit is the
Slack token field's `xoxb-xxxx` placeholder and the `.env` hits are the MCP
server form's `env` / `envFile` field names.

The `__pocketclaw_tour_probe__` marker is a `localStorage` availability probe
that UI-1 introduced as product code, not a test hook. It is safe production
provenance and this audit relies on it as such deliberately, rather than
ignoring it.

## Android manifest

Merged manifest read from the fresh APK with `aapt2 dump xmltree`.
`android:debuggable`, `android:testOnly` and `profileable` are all **absent**.
`usesCleartextTraffic` is absent; `networkSecurityConfig` permits cleartext only
for `127.0.0.1`, `localhost` and `0.0.0.0`. `allowBackup="true"` is paired with
`fullBackupContent` and `dataExtractionRules` that exclude `credentials/`,
`pocketclaw-core/` and `picoclaw/` from cloud backup and device transfer —
`artifact.backup_exclusions` confirms all three in the artifact.
`extractNativeLibs="true"` is required by the Android exec model and expected.

Thirteen permissions, matching `artifact.permissions`, with
`artifact.forbidden_permissions_absent` clean: no `READ_PHONE_STATE`, no
`AD_ID`, no AdServices, no install-referrer. `WRITE_EXTERNAL_STORAGE` and
`READ_EXTERNAL_STORAGE` are bounded by `maxSdkVersion` 28 and 32.
`MANAGE_EXTERNAL_STORAGE` is unbounded and is the broadest thing in the set; it
is a recorded product decision (`DECISIONS.md`, log export) rather than a new
finding, but it is the permission most likely to draw a Play policy review and
is flagged here for that reason.

Exported components were judged on their callable surface, not on the flag:

    MainActivity                  exported=true   LAUNCHER — required.
                                                  Also carries a dead BROWSABLE
                                                  deep link — PC-DEF-024.
    BootReceiver (PocketClaw)     exported=true   BOOT_COMPLETED is system-only.
    ProfileInstallReceiver        exported=true   guarded by permission DUMP.
    WatchdogReceiver              exported=true   plugin-owned, unguarded; can
                                                  only poke a service the app
                                                  starts anyway. Noted, not a
                                                  defect PocketClaw owns.
    flutter_background_service
      .BootReceiver               exported=true   as above.
    PocketClawService             exported=false
    BackgroundService             exported=false
    ShareFileProvider             exported=false  authority .flutter.share_provider
    InitializationProvider        exported=false  authority .androidx-startup
    SharePlusPendingIntent        exported=false
    WebViewActivity               exported=false

## Public Mode and the network boundary

Audited separately for both states with reproducible evidence, and the result
explains `PC-DEF-002` while opening `PC-DEF-020`. Both entries carry the
detail; in summary: Public Mode OFF binds `::1` and `127.0.0.1` only; Public
Mode ON binds `::` and `0.0.0.0`; an explicit host override beats `-public`; the
Core gateway on 18790 is loopback-only in **both** states and is unreachable by
the launcher's public flag. Authentication is mandatory in every state and the
unauthenticated surface is four auth endpoints plus static login assets. What
fails is durability, not enforcement: `launcher-config.json`'s `public` field is
a second persisted authority that the Android OFF path never writes, so the
console can come back LAN-bound after a restart while the native toggle reports
OFF.

## Network endpoints

164 distinct destinations in release-relevant source, 21 of them loopback. The
rest are provider API bases and their console/API-key pages — Anthropic,
OpenAI, Google, Groq, Mistral, DeepSeek, Moonshot, Cerebras, Fireworks,
Together, Perplexity, Tavily, ElevenLabs, Bedrock, Azure, ModelScope and
others — plus `api.github.com` for skills and `clawhub.ai` for the skill
registry. All are HTTPS. No staging endpoint, no developer-specific IP, no
tunnelling or ngrok-style host, and no localhost development server reachable
from production code: the one `localhost:5173` is UI placeholder text. The
intentional LAN HTTP URL is not flagged as insecure transport — it is the
documented local-network Public Mode surface, password-protected, and the
threat model for it is recorded.

Two endpoints are notable rather than clean. `pkg/updater` points at upstream
`api.github.com/repos/sipeed/picoclaw` and at a third-party fork
`api.github.com/repos/sky5454/picoclaw`, and the route that reaches them
accepts an arbitrary URL instead — tracked as `PC-DEF-022`.

## Native, Dart and R8 revalidation

`tool/native_elf_audit.py --enforce-target --native-support-manifest` against
the fresh APK: **194 PASS / 0 FAIL / 0 SKIP**, the same totals as H5B and H5C.
All 18 packaged ELF entries inventoried and category-classified; correct
architecture and type per role, NX everywhere, no writable+executable segment,
no TEXTREL, no RPATH/RUNPATH, 16 KiB-compatible load alignment, no debug
sections, and the Core pair exporting exactly `main.main`. The private native
support manifest was **rebound to this fresh APK** with
`tool/native_support.py bind-apk`; before rebinding, the audit reported the
single expected failure `native.private_support_apk_binding` still naming the
H5C APK, which is the pre-artifact state `PC-DEF-019` documented. The H5C
binding was not falsely retained.

Dart and R8 are regression-free, and the hashes prove no input moved:

    Dart AOT             c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77
    private split DWARF  0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc
    private R8 mapping   14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb

All three are byte-identical to H3A/H3B/H4A/H4B/H5B/H5C. Obfuscation, the
controlled `package:pocketclaw_generated/…` URI, external private
split-debug-info, minification, resource shrinking and the four preserved
manifest entry points all pass, and both private files are external, untracked
and absent from the **APK**. The mapping is not absent from the AAB, which is
`PC-DEF-021`.

`PC-DEF-012` is unchanged by decision and this audit produced no new
reachability evidence: `libdartjni.so` still exports 313 symbols and the CPython
payload 2,261, exactly as H5A recorded. Broad but unchanged exports are not
treated as an audit failure.

## Gates and tests

    tool/release_gate.py --verify-source  (test and production class)   25 PASS / 0 FAIL / 0 SKIPPED
    tool/release_gate.py --full <apk> --release-class test              54 PASS / 0 FAIL / 0 SKIPPED
    tool/native_elf_audit.py --enforce-target                          194 PASS / 0 FAIL / 0 SKIP
    Core Go suite (go test ./...)                                       98 packages ok, 0 failed
    Frontend (vitest)                                                   25 files, 418 tests passed
    Flutter (flutter test)                                             480 passed, 1 FAILED — PC-DEF-025
    tool/ audit-tool suites (6 files)                                   all PASS

No gate was suppressed or reclassified. The artifact gate ran in **test** class
because the artifact is development-signed; it reports `releasable: false` and
classifies signing as LOCAL TEST / NON-RELEASABLE, which is correct and is not
a production result. The one red suite is reported as red.

## Findings

    PC-DEF-020  Public Mode OFF is not durable across a restart      RELEASE BLOCKER
    PC-DEF-021  AAB embeds the private R8 mapping and Dart symbols   RELEASE BLOCKER (publication)
    PC-DEF-022  /api/update fetches an arbitrary URL unverified      open
    PC-DEF-023  third-party Google OAuth client secret embedded      open
    PC-DEF-024  dead analytics deep link exported in the manifest    open
    PC-DEF-025  flutter test red since H5B; no gate runs the suite   open
    PC-DEF-002  explained by design; superseded by PC-DEF-020        resolved as explained

`PC-DEF-006` stays **OPEN**: one artifact was built, not two compared, so
APK-level reproducibility remains unproven and its acceptance criterion is
untouched.

## H5C boundary

The H5C production APK
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7` is
historical evidence for the pre-UI-1 Core and dashboard generation. It is not
relabelled, its recorded hashes and evidence are unaltered, and its device
acceptance does not transfer to the current Core generation. The artifacts in
this record are the first to contain UI-1 and the rebuilt `PC-DEF-019` Core
pair, and neither is production-signed.
