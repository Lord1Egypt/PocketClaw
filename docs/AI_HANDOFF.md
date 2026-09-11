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

H2 developer production signing, H3A/H3B Dart hardening, and H4A/H4B
R8/ProGuard hardening are closed. H5A native/ELF audit and symbol policy is
also closed; it made no native binary change.
Their artifacts are private validation evidence; none was installed, published,
or accepted. The accepted physical baseline remains vc62 / versionCode 62.

The next authorized engineering milestone is **H5B targeted native hardening
and private native-symbol archive**, but implementation must wait for an
explicit prompt. H5A's exact 18-entry ELF inventory, category policy, four
target-policy findings, and ordered H5B plan are in
[`prompts/history/H5A_NATIVE_ELF_AUDIT.md`](prompts/history/H5A_NATIVE_ELF_AUDIT.md).
`PC-DEF-008` is resolved as verified non-blocking; `PC-DEF-009` through
`PC-DEF-012` define the remaining native targets. Do not treat the H4B
validation APK as a final release or accepted physical baseline.

Do not create a stable tag, publish a release, access a device, rebuild Core or
Managed Runtime, or advance the accepted baseline without explicit milestone
authority.
