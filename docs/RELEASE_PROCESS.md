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

The H4A APK is a **development/test artifact** with Dart hardening and narrowed
project-owned R8 rules. Its SHA-256 is
`db7fa8cb190fcebc160b2c718d9c120de296efb378d8a1196a5ff722ba3e1f78`.
It carries the local development signer and is LOCAL TEST / NON-RELEASABLE. It
was not installed, published, accepted, or signed with the production key.
H4B subsequently validated the same R8 contract under production signing.

The H4B APK is a **production validation artifact** with Dart and R8 hardening.
Its SHA-256 is
`14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`.
Its DEX payload, private mapping, Dart AOT, private Dart symbols, Core, and
Managed Runtime payloads match H4A byte for byte; its one v2 signer is the
enrolled developer certificate. All 30 production artifact checks pass. It is
private, uninstalled, unpublished, unaccepted evidence. Native and later
hardening milestones remain, so it is not a final hardened production
candidate.

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

Before every hardened assembly, the helper clears the generated
`.dart_tool/flutter_build` cache and the prior R8 mapping reports. Flutter
3.47.1 tracks cached `app.so` but does not treat external split DWARF as a
required incremental output; invalidating that cache guarantees AOT and its
private symbol companion are regenerated together. Removing the R8 reports
ensures a prior mapping cannot satisfy the post-build assertion. The package
configuration remains intact.

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

Release builds keep `isMinifyEnabled = true`, `isShrinkResources = true`, and
the optimized Android defaults. PocketClaw's application rules file contains no
active blanket keep: generated aapt rules preserve manifest components, Flutter
supplies its embedding/plugin contract, and dependencies supply their consumer
rules. The helper requires the fresh R8 reports at
`build/app/outputs/mapping/release/`, then proves sampled internal PocketClaw
classes were renamed, removed, or folded while the four manifest entry points
remain preserved.

`mapping.txt` and its sibling reports are private release-support material.
Preserve the mapping privately with the exact release for Java/Kotlin stack
deobfuscation. They remain ignored by Git, outside APK/AAB files, absent from
public release assets by default, and are not F-Droid payloads. Supply mapping
evidence to artifact inspection with `--r8-mapping <private-mapping.txt>`.

## Native ELF and private-symbol policy

Run the read-only packaged-ELF inventory before accepting a native-hardened
candidate:

```text
python3 tool/native_elf_audit.py --apk <exact-apk> \
  --manifest <private-or-temporary-json> --enforce-target
```

The final native policy is category-specific. Packaged Dart AOT carries only
the required snapshot exports and unwind data; JNI/plugin libraries retain
their proven Java/FFI/engine entry points; Core and Managed Runtime entries are
PIE executables packaged with `.so` names and must not be treated as ordinary
shared libraries; upstream dependency ELFs retain their pinned contracts unless
source and runtime evidence supports a narrower one. Every shipped ELF must
match its ABI/type role, use at least 16 KiB-compatible load alignment, have a
non-executable stack, have no writable+executable segment or TEXTREL, and expose
no source DWARF, static symbol table, developer checkout path, prohibited build
root, or RPATH/RUNPATH. Dynamic imports require full RELRO; static PIEs with no
lazy-binding relocations treat BIND_NOW as not applicable.

Native support material lives under ignored
`build/private-symbols/native/android-arm64/`. Preserve a symbol-capable
unstripped twin or separate debug companion when technically possible, grouped
by shipped payload. The private per-build manifest records the shipped payload
SHA-256/build ID, support-file path/size/SHA-256, source and toolchain inputs,
and final APK hash. Native support artifacts follow the same policy as Dart
split-debug-info and R8 mapping: private, untracked, outside APK/AAB files, not
public release assets by default, and not F-Droid payloads. Python support data
must be captured before stripping and before appending its standard-library
ZIP. Build IDs aid association but byte hashes remain authoritative.

The H5A audit record in
[`prompts/history/H5A_NATIVE_ELF_AUDIT.md`](prompts/history/H5A_NATIVE_ELF_AUDIT.md)
contains the exact H4B inventory and ordered H5B implementation targets.

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

## Golden final physical-validation workflow

This workflow is documentation for a future explicitly authorized final-device
milestone. It is not authority to access or change a device.

- Stage in WSL at `/mnt/c/temp/pocketclaw`, corresponding to Windows
  `C:\temp\pocketclaw`.
- Use Windows ADB only for this device workflow:
  `/mnt/c/Users/MohamedMounir/AppData/Local/Microsoft/WinGet/Packages/Google.PlatformTools_Microsoft.Winget.Source_8wekyb3d8bbwe/platform-tools/adb.exe`.
- Expected physical target: device `RK8Y6016N5V`, Samsung `SM-A165F`.
- Hash the exact artifact before and after Windows staging.
- The implementation agent installs only when a future milestone explicitly
  authorizes it. The owner performs manual UI and functional acceptance.
- Do not use screenshots or UI automation unless explicitly requested. Do not
  uninstall or clear data during a same-signer upgrade test.
- Never use `adb install -r` across different signing identities.

The accepted vc62 installation and current developer production certificate are
different signing identities. A future production-signer device transition
therefore requires its own authorized migration/clean-install and data-safeguard
plan before any installation attempt.
