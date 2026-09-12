# PocketClaw release roadmap

This is the authoritative phase sequence. Evidence and exact refs live in
[`PROJECT_STATE.md`](../PROJECT_STATE.md); detailed historical narratives remain
in the repository archives. A checked phase means its recorded scope completed,
not that PocketClaw is a final release.

## Completed foundations and hardening

| Milestone | Status | Evidence |
| --- | --- | --- |
| vc62 / Zero-Pico namespace closeout | **Accepted / closed** | `54740f2b3601544fb108f54790f441fb60ca184e`; merged by `15550719cede1fd606460b6ad5199ede067cc97e`; accepted APK `1470e02d43039e78c6507a1fb12e1c9368033903a8b99c2e2f56014eb0c002e2` |
| Repository polish | **Closed** | `8a60ef5`, `fb70181`, `bbe743c`; merged by `47cde00cecb44cb672fa1ae5d8633a3283a1e830` |
| H1 production-signing architecture | **Closed** | `492759bceca159f624d6f74810cbe10d6ad94124` |
| H1.5 canonical/F-Droid cleanup | **Closed** | `612ce969d587d902b338ad198aae9c52279ec887` and `6f117d38000bdba4f70cc7c87740494ba80727fc` |
| H1.5D source-built runtime adoption | **Closed** | `51ed82280ad25c49fc151a69e06d52b150987cc3` through `fb38c7d3c3fd31420c45a1977e1c1425ede3a378` |
| Safety checkpoints | **Closed** | `checkpoint/vc62-accepted`, annotated tag `checkpoint-vc62-accepted`, and `checkpoint/pre-h2` |
| H2 signer enrollment | **Closed** | `78d33fd5b179dd52c7cd8118a5d23ec19c8368ec` |
| H2 private production-signing validation | **Closed** | `7f19309fdf4261224fd9b323a7f2663f59b37545`, corrected by `0be6afbd92209953d918d5c0516662bc54f0d081` |
| PC-1 repository continuity protocol | **Closed** | `9a5a5dd9fd4a0697451d27948efe2c5be6e5c028` |
| H3A Dart hardening architecture and local validation | **Closed by this H3A closeout** | Canonical helper and Gradle guard; non-releasable APK `23dbaa24f375057faf30b469b3a8cafb1a1c235c9afacea15f944df41ad63894`; commit containing this record |
| H3B production-signed Dart-hardening validation | **Closed by this H3B closeout** | Production APK `ceef6640d8abd9d084c3ff37d8e903aaf3c82b287de65ec15a37d91124bdebe6`; 25/25 production artifact checks PASS; closeout commit containing the H3B record |
| H4A R8/ProGuard hardening and local validation | **Closed by this H4A closeout** | Project blanket keeps removed; non-releasable APK `db7fa8cb190fcebc160b2c718d9c120de296efb378d8a1196a5ff722ba3e1f78`; private mapping `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`; 30/30 artifact checks PASS |
| H4B production-signed R8/ProGuard validation | **Closed by this H4B closeout** | Production-validation APK `14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`; H4A-identical DEX/mapping/Dart payloads; enrolled signer; 30/30 production artifact checks PASS |
| H5A native/ELF audit and symbol policy | **Closed by this H5A closeout** | Exact H4B APK; 18 ELF entries classified and inspected; category-specific policy and read-only audit; `PC-DEF-008` resolved; four target-policy findings assigned to H5B |
| H5B targeted native hardening and private native-symbol archive | **Implemented by this H5B closeout; owner production validation required** | Ten owned executables rebuilt; `PC-DEF-009`/`010`/`011` resolved and `PC-DEF-012` deferred by decision; LOCAL TEST APK `d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`; enforced native audit 194/0/0; ten-entry private support manifest bound to that APK |

## Authorized next milestone

### H5B owner production validation

Status: **implementation complete; physical validation not started**.

H5B rebuilt the ten project-owned native executables, resolved `PC-DEF-009`,
`PC-DEF-010` and `PC-DEF-011`, and deliberately left `PC-DEF-012` open for want
of reachability evidence. Its only artifact is a LOCAL TEST / NON-RELEASABLE
APK signed by the development certificate; the enrolled production signer was
not used and no device was touched.

What the owner still has to do:

- rebuild under the enrolled production signer and run the production-class
  gate;
- confirm on the Samsung device that Core, the Core launcher and all eight
  Managed Runtime payloads start and run. Core now relies on `llvm-strip`
  rather than Go's `-s -w`, and jq and CPython had generated source inputs
  normalized; those are the changes a device can falsify and a gate cannot.

The full evidence, including the four findings corrected in the inherited
implementation, is in
[`prompts/history/H5B_NATIVE_HARDENING.md`](prompts/history/H5B_NATIVE_HARDENING.md).

## Required sequence after H5B

Do not collapse these into one milestone. Each receives its own prompt, evidence,
review result, documentation closeout, and `STOP`.

1. H5B owner production-signed validation and physical device confirmation.
   `PC-DEF-008` through `PC-DEF-011` are resolved; `PC-DEF-012` stays open.
2. Secrets/configuration and APK/AAB exposure audit.
3. APK and AAB production inspection.
4. Run the full production-class release gate after all hardening phases.
5. Perform an external-view reverse-engineering exposure audit.
6. Run the final Samsung physical-device smoke test under an explicitly
   authorized device milestone.
7. Create the stable tag and GitHub Release only after owner authorization.

## Distribution targets preserved throughout

- **Direct APK:** developer-signed artifact distributed directly.
- **Google Play:** AAB/upload flow, with Google Play App Signing as its own
  signing lineage.
- **Official F-Droid:** reproducible, developer-signed APK is the target so it
  remains update-compatible with direct APK installs.

The three channels share one canonical source/build configuration where
possible, but their artifact and signing states are not interchangeable. See
[`RELEASE_PROCESS.md`](RELEASE_PROCESS.md).
