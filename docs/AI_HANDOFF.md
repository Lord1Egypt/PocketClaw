# PocketClaw AI handoff

> **IMPORTANT FOR ANY AI AGENT**
>
> Do not invent a new project-management workflow. PocketClaw already has an
> established engineering workflow.
>
> Repository documentation and Git evidence are authoritative. Chat history and
> model memory are advisory only.

## Read in this order

1. [`docs/AI_HANDOFF.md`](AI_HANDOFF.md)
2. [`PROJECT_STATE.md`](../PROJECT_STATE.md)
3. [`docs/WORKING_METHOD.md`](WORKING_METHOD.md)
4. [`docs/ROADMAP.md`](ROADMAP.md)
5. [`docs/DECISIONS.md`](DECISIONS.md)
6. [`docs/DEFECT_LOG.md`](DEFECT_LOG.md)
7. [`docs/RELEASE_PROCESS.md`](RELEASE_PROCESS.md)
8. The relevant record or authorized prompt under [`docs/prompts/`](prompts/README.md)

Detailed historical evidence remains in [`SESSION_HANDOFF.md`](../SESSION_HANDOFF.md),
[`TASKS.md`](../TASKS.md), [`DECISIONS.md`](../DECISIONS.md), and
[`CHANGELOG_DEV.md`](../CHANGELOG_DEV.md). Those are evidence archives. A
phase-scoped statement in them does not override the current snapshot.

## Operating boundary

- Do not begin implementation merely because the next task appears obvious.
- Wait for an explicitly authorized milestone prompt.
- Verify the prompt's branch, starting commit, version, baseline, relevant
  fingerprints, checkpoints, tags, releases, and clean/synchronized Git state.
- Do not broaden scope or combine milestones.
- Do not continue after `STOP`.
- Never assume a milestone is complete unless repository state and evidence
  agree.
- Historical acceptance is immutable. A newer branch artifact does not replace
  an accepted baseline without explicit physical acceptance and a recorded
  baseline advance.

## Current handoff

H2 developer production signing, H3A/H3B Dart hardening, H4A/H4B R8/ProGuard
hardening, H5A native/ELF audit, H5B targeted native hardening and **H5C
production-signed native/ELF validation** are all closed.

H5B was confirmed on hardware: the exact LOCAL TEST APK was installed in place
on the Samsung SM-A165F as a same-identity upgrade with all application data
preserved, and the owner's manual native/runtime smoke returned PASS with no
failing item. H5C then rebuilt the same tree under the enrolled production
signer and proved all 18 packaged ELF entries byte-identical to that physically
validated artifact, so the device result transfers to the production APK.

The production artifact is private validation evidence:
63,564,563 bytes, SHA-256
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`, one v2
signer `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`. The
production release gate passes 55 PASS / 0 FAIL / 0 SKIPPED with
`releasable: true`. It was not installed, published, accepted, or committed, and
the accepted physical baseline remains vc62 / versionCode 62.

**Do not install the production APK over the current device state.** The
installed path is development-signed; the production certificate is a different
identity and a cross-signer `adb install -r` is forbidden. That transition is a
separately authorized migration / clean-install / data-safeguard milestone.

UI-1, an isolated guided-tour repair authorized between H5C and the exposure
audit, then fixed `PC-DEF-013` through `PC-DEF-018` in the console. Because
`libpocketclaw-web.so` embeds the dashboard, that moved the Core source
fingerprint to
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` and left the
staged pair stale — tracked as `PC-DEF-019` and now **resolved**. Core was
rebuilt and re-staged from canonical build-input commit `ea43369` under the
two-commit rule:

    libpocketclaw.so       37,724,640  f273b9ce…  build ID 25e206ab…
    libpocketclaw-web.so   25,517,952  900c43fc…  build ID 84afbe24…
    BuildTime              2026-09-12T07:27:12+0000

Byte-identical in three independent output roots; `core.staged_freshness` and
`build.reproducibility_tests` PASS; 22 PASS / 0 FAIL on the native contract for
the pair. No APK or AAB was built, so the private native support manifest is
still bound to the H5C candidate and is rebound at the next artifact build.

**The H5C production APK is evidence for the pre-UI-1 dashboard.** It is not
relabelled; the next artifact build is the first to contain the current Core
pair, and it needs its own validation.

`PC-DEF-009` through `PC-DEF-011`, `PC-DEF-013` through `PC-DEF-019` and
`PC-DEF-020` are resolved. `PC-DEF-012` stays open by decision — no reachability evidence, so no
export narrowing; the exposure audit produced none and did not narrow anything.

Full evidence is in
[`prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md`](prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md)
and [`prompts/history/H5B_NATIVE_HARDENING.md`](prompts/history/H5B_NATIVE_HARDENING.md);
H5A's 18-entry ELF inventory and category policy remain in
[`prompts/history/H5A_NATIVE_ELF_AUDIT.md`](prompts/history/H5A_NATIVE_ELF_AUDIT.md).
The Core rebuild/re-stage evidence is in
[`prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md`](prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md).

The **final release exposure audit has run and is BLOCKED.** It completed every
part that does not depend on the production signing identity, opened six
defects and fixed none.

    PC-DEF-020  Public Mode OFF is not durable across a restart      RESOLVED
    PC-DEF-021  AAB embeds the private R8 mapping and Dart symbols   RELEASE BLOCKER (publication)
    PC-DEF-022  /api/update fetches an arbitrary URL unverified      open
    PC-DEF-023  third-party Google OAuth client secret embedded      open
    PC-DEF-024  dead analytics deep link exported in the manifest    open
    PC-DEF-025  flutter test red since H5B; no gate runs the suite   open

**`PC-DEF-020` is now RESOLVED.** The Android host always states the Public Mode
decision (`-public=true` / `-public=false`), so the persisted
`launcher-config.json` `public` field is never consulted there and "off"
survives every restart. The Config page reports the effective mode and a save
repairs a stale stored `true`. That fix moved the Core source fingerprint to
`2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df`; the staged
pair is `602ce034…` / `b5cce071…`, BuildTime `2026-09-12T18:54:26+0000`, rebuilt
and re-staged under the two-commit rule.

**Do not build a production-signed candidate yet, and do not request the owner
signing ceremony.** `PC-DEF-021` is still a blocker, and a candidate built
before it is fixed would be superseded along with its hash, native-support
binding and gate evidence. The audit's structural evidence came from a fresh
LOCAL TEST / NON-RELEASABLE APK (`5460d86a…`), which was sound because H5C
proved the production and development builds of one tree differ only by the
signing block — note that APK predates the `PC-DEF-020` Core generation and is
no longer current.

**Do not publish an AAB as a release asset.** `PocketClaw-v0.2.0-rc1.aab` and
`-rc2.aab` already are, and a hardened bundle carries the complete R8
deobfuscation map and the Dart AOT symbols. `PC-DEF-021` covers the policy and
tooling correction; whether to remove the existing published bundles is an
owner decision recorded there.

`PC-DEF-002` is resolved as explained — `0.0.0.0:18800` is Public Mode ON and
nothing else, and the Core gateway stays loopback-only in both states — with
`PC-DEF-020` as its one actionable residue. `PC-DEF-006` stays open: one
artifact was built, not two compared.

The next milestone is `PC-DEF-021` under its own explicit prompt, then a re-run
of the exposure audit to closure. Full evidence is in
[`prompts/history/EXPOSURE_AUDIT.md`](prompts/history/EXPOSURE_AUDIT.md) and
[`prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md`](prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md).

Do not create a stable tag, publish a release, access a device, rebuild Core or
Managed Runtime, or advance the accepted baseline without explicit milestone
authority.
