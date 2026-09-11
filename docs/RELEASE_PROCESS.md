# PocketClaw release process

This document defines release states and the high-level lifecycle. Detailed key
handling and implementation remain in [`RELEASE_SIGNING.md`](RELEASE_SIGNING.md);
F-Droid evidence remains in [`FDROID_RELEASE.md`](FDROID_RELEASE.md).

## Artifact states are not interchangeable

| State | Meaning | May be installed/published? | Changes accepted baseline? |
| --- | --- | --- | --- |
| Development/test artifact | Release-shaped build explicitly using the local development signer | Only under an authorized test/device milestone | No |
| Accepted physical baseline | Exact artifact physically validated and recorded with a baseline advance | Already accepted for continued development/testing | Yes, only in its acceptance commit |
| Production validation artifact | Private artifact proving the developer production signer and production artifact gate | No; evidence only | No |
| Hardened production candidate | Production-signed APK/AAB after all hardening and exposure gates | Not until final review and explicit device/release authority | No |
| Direct APK release | Approved developer-signed APK published directly | Yes, after stable release authorization | Release record, not automatic baseline evidence |
| Google Play submission | Approved AAB authenticated by the Play upload key and re-signed under Play App Signing | Through Play after explicit authorization | No automatic physical baseline change |
| Official F-Droid artifact | Reproducible target carrying the developer certificate, or a separately documented fallback lineage | Through F-Droid after explicit authorization | No automatic physical baseline change |

The H2 APK is a **production validation artifact**. Its SHA-256 is
`f0d83298c2ce061c01a9fc931ad29676e4d4b646bb5b204a9bf0002b11a7f46f`.
It was not installed or published and is not the accepted vc62 artifact.

The H3A APK is a **development/test artifact** with hardened Dart payload. Its
SHA-256 is
`23dbaa24f375057faf30b469b3a8cafb1a1c235c9afacea15f944df41ad63894`.
It carries the local development signer and is LOCAL TEST / NON-RELEASABLE. It
was not installed, published, accepted, or signed with the production key.

The H3B APK is a **production validation artifact** with the same hardened Dart
payload and the enrolled developer signer. Its SHA-256 is
`ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`.
Its 25 production artifact checks pass with no failure or skip, but later R8,
native, exposure, APK/AAB, external-view, and physical milestones remain. It is
private, uninstalled, unpublished, unaccepted evidence rather than a stable
release or hardened production candidate.

## Three signing identities

1. **PocketClaw developer app-signing key.** Long-lived Android application
   identity for Direct APK and the target developer-signed F-Droid APK. Public
   certificate SHA-256:
   `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`.
2. **Google Play upload key.** Future replaceable credential used only to
   authenticate uploads to Play. It does not exist yet and must not be created
   without a dedicated milestone.
3. **Google Play App Signing lineage.** Google-held identity applied to artifacts
   distributed by Play. It is treated as separate from direct/F-Droid lineage.

Private key material, keystores, passwords, and recovery secrets remain outside
Git. A certificate digest is public identity evidence; it is not a secret.

## Lifecycle

1. Start from an explicitly authorized milestone prompt and verified Git state.
2. Implement only that scope and classify every defect found.
3. Run source tests/gates and record exact results.
4. Build only artifacts authorized by the milestone.
5. Inspect the exact new artifact: hash, package, version, ABI, native payloads,
   permissions, signer, and relevant exposure/hardening evidence.
6. Run the correct gate classification. A test artifact cannot pass as
   production merely because it uses a release build type.
7. Update authoritative documentation in the closeout commit.
8. Obtain reviewer `PASS`. Stop.
9. Use separate authorization for physical-device acceptance, baseline advance,
   tag creation, publication, or store submission.

## Canonical Dart-hardened Android build

Use one entry point for release-variant Dart compilation:

```text
python3 tool/build_hardened_android.py --signing local-test --clean
```

The local-test mode refuses any declared production-signing environment field
and passes the existing explicit `-PallowDebugSigning=true` opt-in. Production
validation uses `--signing production` after owner-only hidden secret entry;
credentials remain environment-only and never appear in the command.

The helper invokes `:app:assembleRelease` for `android-arm64` and passes the
exact Flutter 3.47.1 Gradle properties `dart-obfuscation=true` and
`split-debug-info=<private directory>`. It also prepares the generated package
mapping required for the stable
`package:pocketclaw_generated/dart_plugin_registrant.dart` URI. Gradle fails
before Dart compilation if the mode marker, flags, arm64 target, private output
location, or mapping is absent or malformed.

Before every hardened assembly, the helper clears only the generated
`.dart_tool/flutter_build` cache. Flutter 3.47.1 tracks cached `app.so` but does
not treat external split DWARF as a required incremental output; invalidating
that cache guarantees AOT and its private symbol companion are regenerated
together. The package configuration remains intact.

The default Dart support artifact is
`build/private-symbols/dart/android-arm64/app.android-arm64.symbols`. It is
ignored private DWARF for crash deobfuscation/symbolization. Preserve the
artifact privately with the exact release it supports. Do not commit it, put it
inside APK/AAB files, attach it to public GitHub releases by default, or submit
it to F-Droid as a public payload. It is sensitive release-support material,
but it is not an application-signing secret.

Artifact inspection supplies the private path explicitly:

```text
python3 tool/release_gate.py --verify-artifact <apk> \
  --release-class <test|production> --dart-symbols <private-symbol-directory>
```

This verifies that application-level names moved out of `libapp.so`, the split
DWARF exists externally, the generated URI is controlled, private symbols are
not packaged/tracked, and host checkout paths are absent from the packaged Dart
AOT payload.

## Channel paths

### Direct APK

Build the canonical hardened APK with the developer app-signing key. Verify the
signer against the enrolled public fingerprint, record the artifact hash, and
publish only after stable-release authorization.

### Google Play

Produce and inspect the production AAB after the APK path is hardened. Create
and enroll a separate upload key only in its authorized milestone. Play App
Signing determines the certificate on user-delivered Play artifacts.

### Official F-Droid

The target is a developer-signed APK that F-Droid can reproduce bit-for-bit and
pin through `AllowedAPKSigningKeys`, preserving update continuity with Direct
APK. Builder compatibility, committed prebuilts, and full-APK reproducibility
remain open. An F-Droid-signed fallback would be a separate installation
lineage and must be an explicit decision, never an accidental result.

## Stable release threshold

A stable tag and GitHub Release occur only after the roadmap's hardening,
APK/AAB inspection, production gate, external-view exposure audit, and final
Samsung physical smoke all pass and the owner explicitly authorizes release.
Creating a candidate, signing it, or passing an artifact gate does not itself
grant publication authority.
