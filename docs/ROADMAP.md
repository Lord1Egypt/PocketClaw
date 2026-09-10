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
| PC-1 repository continuity protocol | **Closed by this documentation closeout** | The commit containing this document; final hash belongs in the PC-1 report |

## Authorized next milestone

### H3 — Dart Binary Hardening

Status: **authorized for prompt preparation and review; implementation not
started**.

Expected scope:

- controlled Dart obfuscation;
- `--split-debug-info` with symbols kept outside Git and release binaries;
- a controlled generated-source URI strategy;
- the controlled strategy and evidence needed for the later
  `artifact.dart_snapshot_paths` closure, without silently waiving the gate;
- an explicit reproducibility assessment for identical inputs.

## Required sequence after H3

Do not collapse these into one milestone. Each receives its own prompt, evidence,
review result, documentation closeout, and `STOP`.

1. R8 / ProGuard review and keep-rule narrowing.
2. Native hardening, symbol policy, and symbol archive.
3. Secrets/configuration and APK/AAB exposure audit.
4. APK and AAB production inspection.
5. Close `artifact.dart_snapshot_paths` on artifact evidence.
6. Run the full production-class release gate.
7. Perform an external-view reverse-engineering exposure audit.
8. Run the final Samsung physical-device smoke test under an explicitly
   authorized device milestone.
9. Create the stable tag and GitHub Release only after owner authorization.

## Distribution targets preserved throughout

- **Direct APK:** developer-signed artifact distributed directly.
- **Google Play:** AAB/upload flow, with Google Play App Signing as its own
  signing lineage.
- **Official F-Droid:** reproducible, developer-signed APK is the target so it
  remains update-compatible with direct APK installs.

The three channels share one canonical source/build configuration where
possible, but their artifact and signing states are not interchangeable. See
[`RELEASE_PROCESS.md`](RELEASE_PROCESS.md).
