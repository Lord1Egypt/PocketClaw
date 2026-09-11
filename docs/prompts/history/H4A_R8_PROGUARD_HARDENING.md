# RECONSTRUCTED OPERATING RECORD — H4A R8 / ProGuard hardening

This is an evidence-based closeout record, not a claim to reproduce the owner
prompt verbatim.

## Status and refs

- **Status:** CLOSED for architecture and LOCAL TEST / NON-RELEASABLE
  validation; production validation remains H4B.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `30ec1951cb91df2d3ab80ce09d3e1176611242f7`.
- **Ending commit:** the H4A implementation and closeout commit containing this
  record.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline remains vc62 / 62.
- **Next action:** H4B production-signed R8/ProGuard validation under a separate
  explicit prompt and owner-local hidden signing input. It has not started.

## Purpose and boundary

Audit and narrow only the Android Java/Kotlin R8/ProGuard layer, prove real
shrinking/obfuscation with a non-releasable local-test artifact, establish the
private mapping policy, preserve H3 Dart hardening, and stop. Production signing,
native/ELF behavior, device access, publication, merging, tags/releases,
Core/runtime rebuilds, version changes, and baseline advancement were outside
scope. `PC-DEF-008` remains deferred to native/symbol hardening.

## Audit and decisions

Release already used `isMinifyEnabled = true`, `isShrinkResources = true`, the
optimized Android defaults, and `android/app/proguard-rules.pro`. The project
file nevertheless kept every PocketClaw, Flutter, plugin, Firebase, Umeng, Tika,
resource, enum, and JSON-constructor target. H3B mapping evidence showed
nonsynthetic PocketClaw implementation classes retained their original names.

The merged configuration independently preserves the four manifest components
through generated aapt rules. Flutter 3.47.1 supplies its conditional
FlutterPlugin rule, generated plugin registration carries `@Keep`, and
dependencies supply consumer rules. The app has no JNI native methods,
class-name reflection, or reflective model serialization requiring a custom
blanket rule. The PocketClaw application rules file therefore contains no active
rule. Dependency-owned broad rules for file-picker/Tika, background service,
and Dart JNI remain visible and unchanged; H4A did not patch third-party caches
or guess around their runtime contracts.

## Implementation and validation evidence

`tool/r8_contract.py` verifies the source configuration and the fresh build
outputs. The canonical helper removes stale R8 reports before assembly and
fails unless `mapping.txt` and `usage.txt` are nonempty, internal implementation
classes are renamed/removed/folded, their original descriptors are absent from
DEX, manifest components remain preserved, and mapping material is external and
untracked. `tool/release_gate.py` accepts explicit `--r8-mapping` evidence.

- **Canonical invocation:**
  `python3 tool/build_hardened_android.py --signing local-test --clean`.
- **Signing:** explicit development certificate; LOCAL TEST /
  NON-RELEASABLE. Production signing variables were removed.
- **APK:** `build/app/outputs/apk/release/app-release.apk`.
- **Bytes:** 63,556,335.
- **SHA-256:**
  `db7fa8cb190fcebc160b2c718d9c120de296efb378d8a1196a5ff722ba3e1f78`.
- **Package/version:** `com.lord1egypt.pocketclaw`, `0.2.0` (62).
- **ABI:** arm64 product payload; accepted plugin stubs for armeabi-v7a and
  x86_64.
- **DEX:** two entries, 2,283,480 bytes total, down 614,820 bytes from the H3B
  2,898,300-byte baseline.
- **R8 mapping:** `build/app/outputs/mapping/release/mapping.txt`, 13,630,085
  bytes, SHA-256
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`.
- **R8 usage:** 1,991,108 bytes, SHA-256
  `3f09340e4163e5cc8fd4981ab1345be3ec8bad831daf87508797fe05430da7af`.
- **R8 effect:** three internal probes renamed, three removed/folded, all six
  original descriptors absent; MainActivity, PocketClawApp, BootReceiver, and
  PocketClawService preserved.
- **Dart AOT:** unchanged SHA-256
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`.
- **Dart private symbols:** unchanged 3,513,592 bytes, SHA-256
  `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
- **Artifact gate:** 30 PASS / 0 FAIL / 0 SKIP.

Focused tests passed: 8 R8 contract, 12 canonical H3 build contract, 22 release
gate, 67 Flutter signing/Android contracts, and 19 Android release unit tests.
The production source gate passed 23 checks with 0 failures and 0 skips after
the closeout commit.

## Defect and closeout

The first post-build check incorrectly treated R8 removal/folding as missing
mapping evidence. Gradle assembly succeeded; the corrected checker classifies
each probe from mapping plus usage evidence and also scans DEX. This directly
related defect is resolved as `PC-DEF-R011` with regression coverage; the same
fresh artifact passes the corrected inspection, so no rebuild was needed.

The APK, mapping, usage report, and Dart symbols remain ignored and untracked.
Nothing was installed, published, accepted, or committed as an artifact. The
production key was not accessed. Core, Managed Runtime, version, baseline,
protected branches, checkpoints, tags, and GitHub prereleases remain unchanged.
Full APK reproducibility remains open as `PC-DEF-006`; native hardening has not
started.
