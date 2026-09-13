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
| H5B targeted native hardening and private native-symbol archive | **Closed; owner Samsung physical native smoke PASS** | Ten owned executables rebuilt; `PC-DEF-009`/`010`/`011` resolved and `PC-DEF-012` deferred by decision; LOCAL TEST APK `d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`; enforced native audit 194/0/0; ten-entry private support manifest bound to that APK |
| H5C production-signed native/ELF validation | **Closed by this H5C closeout** | Production APK `3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`, one v2 signer `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`; all 18 packaged ELF entries byte-identical to the physically validated H5B APK; native audit 194/0/0; production gate 55/0/0 with `releasable: true` |

## Authorized next milestone

### Final release exposure audit

Status: **CLOSED / PASS.** First run at `25753ce` was blocked; the closure
re-run at `6031898` found no remaining release blocker for the GitHub / direct
APK release and opened no new defect. Evidence:
[`prompts/history/EXPOSURE_AUDIT_RERUN.md`](prompts/history/EXPOSURE_AUDIT_RERUN.md).

Historical first-run status: **blocked, not closed.** Two release blockers proven
(`PC-DEF-020` and `PC-DEF-021`, both since resolved) plus four further defects
(`PC-DEF-022` through `PC-DEF-025`; `PC-DEF-025` is also resolved). The audit is
ready to be re-run to closure. `PC-DEF-002` resolved as
explained. Evidence:
[`prompts/history/EXPOSURE_AUDIT.md`](prompts/history/EXPOSURE_AUDIT.md).
Re-running it to closure requires a fresh prompt after the blockers are fixed.

Expected scope: secrets and configuration exposure, then full APK and AAB
inspection. H5C closed the native layer — the production-signed artifact carries
byte-identical native payloads to the build the owner exercised on the Samsung —
so the remaining questions are about what the package discloses, not how it was
compiled.

Carried forward, none of them blocking this audit:

- `PC-DEF-012`: dependency and JNI export surfaces stay as they are until
  reachability evidence exists.
- APK-level reproducibility for the F-Droid path.
- The versioned non-destructive bootstrap update strategy.
- The production-signer physical transition, which needs its own migration,
  clean-install and data-safeguard milestone because the installed device path
  is development-signed.

## Required sequence after H5C

Do not collapse these into one milestone. Each receives its own prompt, evidence,
review result, documentation closeout, and `STOP`.

1. Secrets/configuration and APK/AAB exposure audit. **Run; BLOCKED.**
1a. Fix `PC-DEF-020` (Public Mode durability) — **DONE**, resolved 2026-09-12;
    evidence in
    [`prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md`](prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md).
1b. Fix `PC-DEF-021` (bundle mapping/symbol policy and gate) — **DONE**,
    resolved 2026-09-12; evidence in
    [`prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md`](prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md).
1c-pre. Fix `PC-DEF-025` (stale Flutter assertion; gate runs three files, not
    the suite) — **DONE**, resolved 2026-09-13; evidence in
    [`prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md`](prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md).
1c. Re-run the exposure audit to closure — **DONE**, CLOSED / PASS 2026-09-13.
1d. `PC-DEF-022` update-surface removal — **DONE**, resolved 2026-09-13;
    evidence in
    [`prompts/history/PC-DEF-022_UPDATE_SURFACE_REMOVAL.md`](prompts/history/PC-DEF-022_UPDATE_SURFACE_REMOVAL.md).
1e. `PC-DEF-024` dead analytics deep-link removal — **DONE**, resolved
    2026-09-13; evidence in
    [`prompts/history/PC-DEF-024_DEAD_DEEP_LINK_REMOVAL.md`](prompts/history/PC-DEF-024_DEAD_DEEP_LINK_REMOVAL.md).
1f. `PC-DEF-023` — **DONE**, resolved 2026-09-13. Owner decision `PC-D014`:
    Google Antigravity is not shipped in v0.2.0, because PocketClaw stable will
    not depend on a third party's OAuth client. Evidence in
    [`prompts/history/PC-DEF-023_ANTIGRAVITY_REMOVAL.md`](prompts/history/PC-DEF-023_ANTIGRAVITY_REMOVAL.md).
    It may return under a PocketClaw-owned OAuth integration.
2. APK and AAB production inspection. Blocked until 1c closes: no
   production-signed candidate may be built before then, and the owner signing
   ceremony must not be requested for a candidate that will be superseded.
3. Production-signer physical transition: migration, clean install and data
   safeguard. `PC-DEF-008` through `PC-DEF-011` are resolved; `PC-DEF-012`
   stays open.
5. Perform an external-view reverse-engineering exposure audit.
6. Run the final Samsung physical-device smoke test under an explicitly
   authorized device milestone.
7. Create the stable tag and GitHub Release only after owner authorization.

## Distribution targets preserved throughout

- **Direct APK:** developer-signed artifact distributed directly.
- **Google Play:** AAB/upload flow, with Google Play App Signing as its own
  signing lineage. The bundle is a Play-upload artifact only and is never a
  public release asset — `PC-DEF-021`.
- **Official F-Droid:** reproducible, developer-signed APK is the target so it
  remains update-compatible with direct APK installs.

The three channels share one canonical source/build configuration where
possible, but their artifact and signing states are not interchangeable. See
[`RELEASE_PROCESS.md`](RELEASE_PROCESS.md).
