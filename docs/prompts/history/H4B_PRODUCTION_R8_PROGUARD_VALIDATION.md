# RECONSTRUCTED OPERATING RECORD — H4B production-signed R8 / ProGuard validation

This is an evidence-based closeout record, not a claim to reproduce the owner
prompt verbatim.

## Status and refs

- **Status:** CLOSED for production-signed R8/ProGuard validation.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `a6034c065becccc0a01ed7e734dad6b2558a0ef1`.
- **Ending commit:** the H4B public documentation closeout commit containing
  this record.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline remains vc62 / 62.
- **Next action:** native/ELF hardening, symbol policy, and private symbol
  archive under a separate explicit prompt. It has not started.

## Purpose and boundary

Validate the exact H4A R8/ProGuard contract under the enrolled PocketClaw
production signer, compare meaningful payloads against preserved H4A evidence,
keep R8 mapping and Dart split-debug-info private, update authoritative records,
and stop. R8 redesign, native/ELF changes, device access, publication, merging,
tags/releases, Core/runtime rebuilds, version changes, and baseline advancement
were outside scope.

## Owner-secret handling

A password-free helper outside the repository checked Java 17, Python, Gradle,
Git state, repository paths, the production-keystore path, the R8 source
contract, and the exact Android signing-validation route before its first
hidden prompt. The owner entered both passwords through `read -rsp`; they lived
only in that helper process. An exit trap unset all four signing variables.
No value entered the conversation, command arguments, files, logs, docs, or
Git.

The first owner attempt exposed `PC-DEF-R012`: calling the wrapper from the
repository root did not select the Android Gradle project. The corrected helper
uses `android/gradlew -p android` for every Gradle call and probes that exact
route without secrets before prompting. Twelve focused assertions plus an EOF
dry run verified the root and ordering contract. The first attempt produced no
H4B APK; the corrected run completed.

The first successful production-gate manifest exposed `PC-DEF-R013`: its
static pending list still called H4B pending even though all 30 checks passed.
The closeout removes only that completed item, retains the open F-Droid and
bootstrap items, and adds a focused regression test. This metadata-only fix did
not require an artifact rebuild.

## Production validation evidence

- **Canonical invocation:**
  `python3 tool/build_hardened_android.py --signing production --clean`.
- **Signing validation:** `:app:validateReleaseSigning` selected the production
  environment keystore before assembly; no debug fallback was enabled.
- **APK:** `build/app/outputs/apk/release/app-release.apk`.
- **Bytes:** 63,560,431.
- **SHA-256:**
  `14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`.
- **Package/version:** `com.lord1egypt.pocketclaw`, `0.2.0` (62).
- **ABI:** arm64 product payload; accepted plugin stubs for armeabi-v7a and
  x86_64.
- **Signer:** exactly one APK Signature Scheme v2 signer, SHA-256
  `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
  The development signer was absent.
- **DEX:** two entries, 2,283,480 bytes total. `classes.dex` is 2,069,524
  bytes, SHA-256
  `6de32319f187e9dd26d8313a4a44cb8f94f2f08c2e0773bd9d5f0e4e067a386b`;
  `classes2.dex` is 213,956 bytes, SHA-256
  `e16d9e1acfb5180aab18d10942562486710c53636daec37606f4f0d3a18e6e5f`.
- **R8 mapping:** `build/app/outputs/mapping/release/mapping.txt`, 13,630,085
  bytes, SHA-256
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`.
- **Dart AOT SHA-256:**
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`.
- **Dart private symbols:**
  `build/private-symbols/dart/android-arm64/app.android-arm64.symbols`,
  3,513,592 bytes, SHA-256
  `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
- **Production artifact gate:** 30 PASS / 0 FAIL / 0 SKIP.

Both DEX files, the R8 mapping, Dart AOT, Dart symbols, Core, and all eight
Managed Runtime payloads are byte-identical to the preserved H4A evidence.
Production signing changes the whole APK hash and does not change these build
payloads. The six H4A internal descriptors remain absent, the four manifest
components remain preserved, and Dart hardening, controlled generated-source
URI, and snapshot-path checks remain PASS. This is scoped build equivalence,
not full APK reproducibility; `PC-DEF-006` remains open.

The mapping and Dart symbols are ignored and untracked, are absent from the
APK, and were not published or committed. The APK itself is private validation
evidence and was not installed, published, accepted, or committed.

## Tests and invariants

Focused suites passed: 8 R8 contract tests, 12 canonical hardened-build tests,
23 release-gate tests, 486 Flutter tests, and 19 Android release unit tests.
The production source gate passed 23 checks with 0 failures and 0 skips after
the closeout commit.

Core, Managed Runtime, version, baseline, production-signer enrollment,
protected branches, checkpoints, tags, and RC1-RC3 remain unchanged.
`PC-DEF-008` remains deferred to native/symbol hardening, which H4B did not
start.
