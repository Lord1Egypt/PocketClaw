# RECONSTRUCTED OPERATING RECORD — H3A Dart binary hardening

This is an evidence-based closeout record, not a claim to reproduce the
original owner prompt verbatim.

## Status and refs

- **Status:** CLOSED for architecture and non-releasable validation.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `9a5a5dd9fd4a0697451d27948efe2c5be6e5c028`.
- **Ending commit:** the H3A closeout commit containing this record.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline remains vc62 / 62.
- **Next action:** H3B owner production-signed validation under a separate
  explicit prompt and hidden local password input.

## Purpose and boundary

Establish one fail-closed Dart-obfuscation/split-debug-info path, remove the
generated checkout URI from the packaged Dart snapshot, validate it twice with
local development signing, assess Dart-output determinism, update the gates and
authoritative docs, and stop. Production signing, installation, publication,
R8/ProGuard redesign, native hardening, Core/runtime rebuilding, merging, tags,
and releases were outside scope.

## Pinned toolchain evidence

Flutter tag `3.47.1`, commit
`6655482ec06e547f90abf8ae7590466f4415978d`, consumes Gradle properties
`dart-obfuscation` and `split-debug-info` in `FlutterPlugin.kt` and forwards
them as `DartObfuscation` and `SplitDebugInfo` build defines in
`BaseFlutterTaskHelper.kt`. `base/build.dart` translates those defines to
`gen_snapshot --obfuscate` and external DWARF through
`--dwarf-stack-traces`, `--resolve-dwarf-paths`, and
`--save-debugging-info`.

The same plugin reads `filesystem-roots` and `filesystem-scheme` into task
fields but this pinned task helper does not forward those fields to `flutter
assemble`. Direct trials with the dedicated properties and equivalent extra
frontend flags left the generated absolute URI unchanged. Flutter's
`PackageConfigWorkspaceExtension.toPackageUriForWorkspace`, however, converts
an additional source under a package URI root before the frontend invocation.
H3A uses that supported compiler path.

## Baseline and resolution

The exact H2 APK hash matched
`f0d83298c2ce061c01a9fc931ad29676e4d4b646bb5b204a9bf0002b11a7f46f`.
Its `libapp.so` contained one host-specific generated URI:

```text
file:///home/lordegypt/PocketClaw-App/.dart_tool/flutter_build/dart_plugin_registrant.dart
```

The canonical helper adds a deterministic generated-only package entry to the
ignored Pub package config, mapping `.dart_tool/flutter_build/` to
`pocketclaw_generated`. Flutter then emits:

```text
package:pocketclaw_generated/dart_plugin_registrant.dart
```

Gradle validates that mapping and the complete hardening property set before
release compilation. It refuses raw/malformed release compiles.

## Canonical command and private-symbol policy

```text
python3 tool/build_hardened_android.py --signing local-test --clean
```

The helper calls `:app:assembleRelease` with `android-arm64`, explicit Dart
obfuscation, split debug info, the H3A mode marker, and the existing explicit
debug-signing opt-in. Default DWARF output is
`build/private-symbols/dart/android-arm64/app.android-arm64.symbols`.

Split debug info is private release-support material. It stays outside the APK,
Git, public GitHub release assets, and public F-Droid payloads, and should be
preserved privately with the release it supports. No upload or storage service
was introduced.

## Non-releasable artifact evidence

- **APK:** `build/app/outputs/apk/release/app-release.apk`.
- **Bytes:** 63,783,811.
- **SHA-256:** `23dbaa24f375057faf30b469b3a8cafb1a1c235c9afacea15f944df41ad63894`.
- **Package/version:** `com.lord1egypt.pocketclaw`, `0.2.0` (62).
- **ABI:** arm64 product payload; existing plugin stubs for armeabi-v7a and
  x86_64.
- **Signer:** development certificate
  `15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`;
  LOCAL TEST / NON-RELEASABLE.
- **Dart AOT SHA-256:**
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`.
- **Private DWARF:** 3,513,592 bytes; SHA-256
  `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
- **Artifact gate:** 25 PASS / 0 FAIL / 0 SKIPPED.

Four sampled application identifiers present in the private DWARF were absent
from packaged `libapp.so`. The private symbol file was not packaged. The
generated URI was stable, and `artifact.dart_snapshot_paths` passed with no
host-specific path.

## Focused reproducibility result

Two clean equivalent builds used different split-info roots: the ignored
default above and `/tmp/pocketclaw-h3a-repro-b-20260911`. Both produced the
same Dart AOT hash and the same split-DWARF hash byte for byte. Absolute
split-output location therefore did not affect either scoped Dart output in
this toolchain. The APK hashes differed, so this evidence does not claim or
close full-APK reproducibility; `PC-DEF-006` remains open for the later F-Droid
proof.

## Defect disposition and invariants

`PC-DEF-001` is resolved as `PC-DEF-R008` on actual H3A artifact evidence. No
unrelated new defect was found. The production key and passwords were not
accessed. The APK was not installed, published, accepted, or committed. Core,
Managed Runtime, version, baseline, protected branches, checkpoints, tags, and
GitHub prereleases remained unchanged. H3B, R8/ProGuard review, and native
hardening were not started.
