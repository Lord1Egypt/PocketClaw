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
hardening and H5A native/ELF audit are closed. **H5B targeted native hardening
and private native-symbol archive is implemented** under LOCAL TEST /
NON-RELEASABLE signing; it rebuilt the ten project-owned native executables and
resolved `PC-DEF-009`, `PC-DEF-010` and `PC-DEF-011`. `PC-DEF-012` stays open by
decision — no reachability evidence, so no export narrowing.

Every H5B artifact is private validation evidence. None was installed,
published, or accepted, and the accepted physical baseline remains vc62 /
versionCode 62. Do not treat the H5B candidate APK as a release or as an
accepted physical baseline.

**H5B is not physically validated.** The native changes are source-level and
all gates are green, but nothing ran on hardware and Core now relies on
`llvm-strip` rather than Go's `-s -w`. Owner production-signed validation on the
Samsung device is the next action.

Full H5B evidence — old and new hashes for all ten payloads, the private
support manifest, reproducibility proof, and the four findings corrected in the
inherited implementation — is in
[`prompts/history/H5B_NATIVE_HARDENING.md`](prompts/history/H5B_NATIVE_HARDENING.md).
H5A's 18-entry ELF inventory and category policy remain in
[`prompts/history/H5A_NATIVE_ELF_AUDIT.md`](prompts/history/H5A_NATIVE_ELF_AUDIT.md).

The next engineering milestone is **H5C**, which has not started and requires a
separate explicit prompt.

Do not create a stable tag, publish a release, access a device, rebuild Core or
Managed Runtime, or advance the accepted baseline without explicit milestone
authority.
