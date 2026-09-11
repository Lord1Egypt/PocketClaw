# RECONSTRUCTED OPERATING RECORD — H3B production-signed Dart hardening

This is an evidence-based closeout record, not a claim to reproduce the
original owner prompt verbatim.

## Status and refs

- **Status:** CLOSED for private production-signed Dart-hardening validation.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `6491ccc6f611c7513506d622dbb1ad4a75c93a43`.
- **Ending commit:** the H3B defect-fix and closeout commit containing this
  record.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline remains vc62 / 62.
- **Next action:** R8 / ProGuard final hardening review under a separate explicit
  prompt. It has not started.

## Purpose and boundary

Validate H3A's canonical Dart-obfuscation, split-debug-info, generated-package
URI, and arm64 build contract under the enrolled developer production signer.
The artifact remained private and uninstalled. R8/ProGuard changes, native
hardening, device access, publication, merging, tags, releases, Core/runtime
rebuilds, version changes, and baseline advancement were outside scope.

## Owner-secret handling

The owner entered both passwords through hidden terminal input in a temporary
password-free helper outside the repository. Values existed only in that
process environment, never appeared in argv, output, logs, documentation, or
Git, and were unset by an exit trap. After the first attempt showed that Java
preflight occurred too late, the corrected helper validated JDK 17, Python,
Gradle, repository/helper paths, and keystore presence before its first hidden
prompt.

## Directly related defects

The first owner build validated production signing and completed assembly, but
the post-build assertion found no external DWARF. Flutter 3.47.1 had reused
cached `.dart_tool/flutter_build/.../arm64-v8a/app.so` after the canonical helper
deleted the expected symbol file; external split debug info is not a tracked
incremental-cache output. This was not a cwd error: the Python resolver, Gradle
guard, and pinned Flutter task all resolve the relative directory against the
Flutter source root.

The helper now clears only `.dart_tool/flutter_build` before hardened assembly,
forcing `gen_snapshot` to emit AOT and private DWARF together. Focused tests
cover stale-cache removal, package-config preservation, symlink refusal,
cwd-independent symbol resolution, and the canonical call. The corrected owner
rerun regenerated both outputs and passed all artifact checks. These defects are
recorded as `PC-DEF-R009` and `PC-DEF-R010`.

## Production validation evidence

- **Canonical invocation:**
  `python3 tool/build_hardened_android.py --signing production --clean`.
- **APK:** `build/app/outputs/apk/release/app-release.apk`.
- **Bytes:** 63,787,907.
- **SHA-256:**
  `ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`.
- **Package/version:** `com.lord1egypt.pocketclaw`, `0.2.0` (62).
- **ABI:** arm64 product payload; existing plugin stubs for armeabi-v7a and
  x86_64.
- **Signer:** exactly one v2 signer; certificate SHA-256
  `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
- **Dart AOT SHA-256:**
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`.
- **Private DWARF:** 3,513,592 bytes; SHA-256
  `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`.
- **Production artifact gate:** 25 PASS / 0 FAIL / 0 SKIP.

The AOT and private DWARF are byte-identical to H3A. All APK ZIP entry names and
payload bytes also match H3A; the whole-file APK hash differs through the
signing block. The controlled generated URI remains
`package:pocketclaw_generated/dart_plugin_registrant.dart`, sampled application
names remain absent from packaged AOT, and no owner-home or checkout path is
present. Core and all eight Managed Runtime payloads match H3A byte for byte.

## Closeout and invariants

Private symbols remain outside the APK and Git. The APK was not installed,
published, uploaded, accepted, or committed. Core, Managed Runtime, version,
baseline, protected branches, checkpoints, tags, and GitHub prereleases remain
unchanged. Full APK reproducibility is not claimed; `PC-DEF-006` remains open.
R8/ProGuard and native hardening have not started.
