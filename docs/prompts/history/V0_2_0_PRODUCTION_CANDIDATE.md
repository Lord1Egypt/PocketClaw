# OPERATING RECORD — final v0.2.0 production-signed release candidate

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **PASS.** A production-signed v0.2.0 candidate exists and passes the
  full production artifact gate with zero failures and zero skips.
- **Working branch:** `feature/final-release-hardening`.
- **Source tree:** `1c477e601c86be0cd5690343c23fbf452fce1d93`, clean and
  origin-synchronized at build time.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, unchanged.

Build, inspect and gate only. **Nothing was installed, merged, tagged, published
or submitted.** No device was touched and ADB was not invoked. No AAB was built —
public GitHub assets are APK-only. `main` stays `100a51de…` and `develop` stays
`47cde00c…`. A production build is not physical acceptance.

## Provenance clarification — read this before trusting a commit message

Two intermediate literals survive in commit messages and are **superseded**.
They are recorded here so a future maintainer cannot mistake them for artifact
provenance:

    superseded  fingerprint  f9a2d2a8…              in the message of 54ff252
    superseded  BuildTime    2026-09-13T00:23:11+0000   in the message of 1c477e6

Both predate an amend of the source commit and the Core rebuild that followed
it. The **authoritative** values, and the ones stamped into the shipped
binaries, are:

    fingerprint  bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9
    BuildTime    2026-09-13T00:24:56+0000

Verified rather than asserted: both superseded literals occur **zero** times in
either staged Core binary and **zero** times in any tracked document; they exist
only in the two commit messages. Neither commit was rewritten — history is
evidence, and amending it to tidy a superseded number would destroy more than it
fixes.

## The candidate

    path    build/app/outputs/apk/release/app-release.apk
    bytes   63,472,307
    sha256  4d4bc33a63059450383c4eedb34e2902486fbbc8c0d85b91413ab9654b4f3dac

Built by the canonical entry point, `tool/build_hardened_android.py --signing
production --clean`, through an owner-run hidden-input helper kept outside the
repository. No password was requested in conversation, placed on a command line,
written to a repository file, put in a Gradle property or logged. The helper
completed every non-secret prerequisite before its first prompt, proved the
keystore and alias open by piping the password to `keytool` on stdin, and unset
all four signing variables on exit.

**The hash above is frozen.** Every check below was run against that exact file,
and no rebuild occurred after it was recorded.

## Signing

    apksigner verify        Verifies
    schemes                 v2 only (v1/v3/v3.1/v4 false) — as H5C established
    number of signers       1
    signer SHA-256          176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf
    subject                 CN=PocketClaw, OU=PocketClaw Release, O=PocketClaw

That digest is exactly the certificate enrolled in
`android/release-signing-cert.sha256`. The local development certificate
`15cf75f9…` occurs **zero** times in the artifact. `artifact.signing` reports
`enrolled production signer`.

## Core generation and payloads

    Core fingerprint     bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9
    build-input commit   54ff2525fa555744d017aae56c9a26e2049812e1
    BuildTime            2026-09-13T00:24:56+0000

    libpocketclaw.so       37,658,976  0a28bd5e1d6e33dc35b808039571b6683e4d47dec021941650f916641b859f6e
                                       build ID c657e80da54549a3bcc9a8bdba0a576d7b7273a0
    libpocketclaw-web.so   25,319,424  9ae1d2d9e7ac26d602db722649ebec4ac9cd982fa166c50ded303685d2abce50
                                       build ID 45355d87ba5ec042740675f82bc3d3940265318a

`artifact.core_provenance_pair` reads `bc35a598…` from **both** binaries,
`artifact.core_matches_staged` reports identical, and `artifact.build_time`
reports the authoritative timestamp. Packaged ABI is arm64-v8a for the product
payload, with the accepted `armeabi-v7a` and `x86_64` plugin stubs.

All eight Managed Runtime payloads match their pinned checksums exactly:

    curl 9dd4b75a…   gh 804ed93f…   git-remote-http baea79ac…   git a8a342ac…
    jq 843b324d…     python 8b52e36d…   rg 0a5fc43f…   sqlite3 b940785d…

## Dart, R8 and private material

    Dart AOT (libapp.so)        c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77
    private split DWARF          0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc  (3,513,592 bytes)
    private R8 mapping           736ee88d14ff8a8a45d04aa565214d1ce7d7e43709525234c2be9e3e711319c1  (13,629,573 bytes)
    R8 usage report              0238790992ded1b81eb264341697580c06c11f1f71cecfaa339ae6224485a807  (1,990,863 bytes)

Dart AOT and the private DWARF are byte-identical to H3A and every hardened
build since. Obfuscation, the controlled `package:pocketclaw_generated/…` URI,
external split-debug-info, minification, resource shrinking and the four
preserved manifest entry points all pass.

`artifact.r8_mapping_private` reports `496 archive entries scanned,
deobfuscation entries = 0` — read from this artifact, not asserted.
`artifact.dart_symbols_private` reports external and untracked.

## Native audit and support manifest

`tool/native_elf_audit.py --enforce-target --native-support-manifest`:
**194 PASS / 0 FAIL / 0 SKIP** over 18 packaged ELF entries.

The private native support manifest was **rebound to this exact candidate**,
`4d4bc33a…`, replacing the intentionally stale PC-DEF-024 audit binding. No
private support material is packaged and none is committed.

## Packaged-content and exposure checks

Thirteen private/residue categories, all **CLEAN**: Dart private symbols, R8
mapping, native debug companions, keystores and key files, `.env`, VCS metadata,
CI metadata, source maps, build logs and dumps, shell scripts, test fixtures,
developer documentation and temporary files.

Absence checks over every packaged entry:

    um.placeholder              0      /api/update                 0
    google-antigravity          0      Antigravity                 0
    Google Code Assist          0      antigravity.google          0
    R09DU1BYLU / GOCSPX-        0      apps.googleusercontent.com  0
    lordegypt                   0      PocketClaw-App              0
    pocketclaw-runtime-build    0      /mnt/c/                     0

**One `antigravity` occurrence, resolved and not a finding.** It is in
`libpocketclaw-python.so`, in CPython's embedded module-name table beside `dis`,
`ftplib`, `subprocess`, `tty` and `zlib` — the standard library's own
`antigravity` module, the `import antigravity` easter egg. That payload is
byte-identical to its pinned checksum `8b52e36d…`, so the string provably
predates the `PC-DEF-023` removal and cannot have been introduced by it. This is
exactly why the absence rules are specific rather than a blanket "no
antigravity" grep.

Gemini survives as it must: `generativelanguage.googleapis.com` and
`Google Gemini` each occur three times.

## Merged manifest of the candidate

    android:debuggable       absent      android:testOnly        absent
    profileable              absent      usesCleartextTraffic    absent
    allowBackup              present     networkSecurityConfig   present
    fullBackupContent        present     dataExtractionRules     present
    extractNativeLibs        present     category.LAUNCHER       present
    um.placeholder           absent      BROWSABLE               absent
    android:scheme           absent

    package  com.lord1egypt.pocketclaw
    version  versionCode 62, versionName 0.2.0

13 permissions as expected with the forbidden set absent, and backup exclusions
intact for `credentials/`, `pocketclaw-core/` and `picoclaw/`.

## Public Mode and Gateway

Unchanged, and re-proved with a live bind probe through the shipped
`pkg/netbind`:

    PUBLIC OFF (no -public, no host)    bindHosts=[::1 127.0.0.1]
    PUBLIC ON  (-public, no host)       bindHosts=[:: 0.0.0.0]
    host override 127.0.0.1 + -public   bindHosts=[127.0.0.1]
    gateway (host=localhost)            bindHosts=[::1 127.0.0.1]  in BOTH states

The Android host still states the decision unconditionally
(`-public=` + the toggle), `resolveLauncherPublicMode` still lets a supplied
flag win, and the gateway still passes `netbind.DefaultLoopback` with
`gatewayHostOverride` pinned to `localhost`.

## Gates and tests

    flutter analyze                                   No issues found
    flutter test                                      497 passed, 0 failed
    frontend (vitest)                                 25 files, 418 tests passed
    Core Go suite                                     98 packages ok, 0 failed
    pkg/coresource                                    46 tests, 0 failed
    Android contract tests                            87 passed
    release-tool suites (8 files)                    139 tests, all OK
    source gate, test class                           26 PASS / 0 FAIL / 0 SKIPPED
    source gate, production class                     26 PASS / 0 FAIL / 0 SKIPPED
    native ELF audit, enforced                       194 PASS / 0 FAIL / 0 SKIP
    FULL ARTIFACT GATE, production + public-release   57 PASS / 0 FAIL / 0 SKIPPED

The artifact gate prints **`PASS — production release candidate.`** Its standing
"pending final hardening" line still names APK-level reproducibility
(`PC-DEF-006`, F-Droid path) and the versioned bootstrap update strategy; both
are known, open and outside this milestone.

## Public release asset policy

Checked, not published. The intended future asset set — the hardened APK,
`SHA256SUMS.txt`, changelog, third-party notices and licence — passes
`release.public_asset_allowlist`. The forbidden set is rejected with each reason
named: `.aab` as an android app bundle, `mapping.txt` as R8 mapping,
`app.android-arm64.symbols` as Dart split debug info, `libpocketclaw.so.debug`
as a native debug companion, `pocketclaw-release.jks` as signing material and
`.env` as an environment file.

The historical `v0.2.0-rc1` and `-rc2` AAB assets were **not** touched. Removing
them is a separate owner authorization.

## What this candidate is not

It is a **private production validation artifact**. It has not been installed,
published, accepted or released, and it does not advance the accepted physical
baseline.

**Do not install it over the current device state.** The Samsung carries a
development-signed lineage; the production certificate is a different identity,
so a cross-signer `adb install -r` is forbidden and would fail. The physical
transition needs its own milestone with a migration, clean-install and
data-safeguard plan.
