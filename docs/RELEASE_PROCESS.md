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
