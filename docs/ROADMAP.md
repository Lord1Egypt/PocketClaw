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

## Authorized next milestone

### R8 / ProGuard final hardening review

Status: **not started; implementation must wait for an explicit milestone
prompt**.

Expected scope:

- inspect the current Java/Kotlin shrinker configuration and actual artifact;
- review broad keep rules against pinned plugin/runtime requirements;
- narrow only rules supported by focused evidence and regression coverage;
- preserve the validated H3B Dart, signing, Core, runtime, version, and baseline
  contracts.

H3B proved controlled Dart obfuscation, regenerated external split debug info,
the stable `package:pocketclaw_generated/dart_plugin_registrant.dart` strategy,
and a 25/25 production artifact gate under the enrolled production signer. It
does not authorize R8 / ProGuard work automatically.

## Required sequence after H3B

Do not collapse these into one milestone. Each receives its own prompt, evidence,
review result, documentation closeout, and `STOP`.

1. R8 / ProGuard review and keep-rule narrowing.
2. Native hardening, symbol policy, and symbol archive.
3. Secrets/configuration and APK/AAB exposure audit.
4. APK and AAB production inspection.
5. Run the full production-class release gate after all hardening phases.
6. Perform an external-view reverse-engineering exposure audit.
7. Run the final Samsung physical-device smoke test under an explicitly
   authorized device milestone.
8. Create the stable tag and GitHub Release only after owner authorization.

## Distribution targets preserved throughout

- **Direct APK:** developer-signed artifact distributed directly.
- **Google Play:** AAB/upload flow, with Google Play App Signing as its own
  signing lineage.
- **Official F-Droid:** reproducible, developer-signed APK is the target so it
  remains update-compatible with direct APK installs.

The three channels share one canonical source/build configuration where
possible, but their artifact and signing states are not interchangeable. See
[`RELEASE_PROCESS.md`](RELEASE_PROCESS.md).
