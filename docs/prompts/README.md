# PocketClaw milestone prompts

This directory preserves the operating contract for future milestones and
evidence-based records of important completed phases.

- [`PROMPT_TEMPLATE.md`](PROMPT_TEMPLATE.md) is the canonical structure for a
  new milestone prompt.
- [`REVIEW_PROTOCOL.md`](REVIEW_PROTOCOL.md) defines the independent reviewer
  role and its only valid outcomes.
- [`history/`](history/) contains reconstructed operating records for completed
  phases.

## Original versus reconstructed

An exact historical prompt may be stored only when its verbatim text is
available in repository evidence and contains no secret or prohibited material.
Otherwise the file must begin with **RECONSTRUCTED OPERATING RECORD**. A
reconstructed record summarizes commits, scope, outcome, defects, and evidence;
it never claims to be the original prompt.

Historical records do not authorize new implementation and do not override
[`PROJECT_STATE.md`](../../PROJECT_STATE.md). The active milestone must arrive
as an explicit owner-authorized prompt, pass pre-flight, and stop at its stated
boundary.

## Reconstructed history index

- [`H1 — Production signing architecture`](history/H1.md)
- [`H1.5 — Canonical build and F-Droid cleanup`](history/H1_5.md)
- [`H1.5D — Source-built runtime adoption`](history/H1_5D.md)
- [`Safety checkpoints before H2`](history/SAFETY_CHECKPOINTS.md)
- [`H2 — Developer production signer enrollment`](history/H2_ENROLLMENT.md)
- [`H2 — Private production-signing validation`](history/H2_PRODUCTION_VALIDATION.md)
