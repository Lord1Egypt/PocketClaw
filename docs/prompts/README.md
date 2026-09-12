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
- [`H3A — Dart hardening architecture and non-releasable validation`](history/H3A_DART_BINARY_HARDENING.md)
- [`H3B — Production-signed Dart-hardening validation`](history/H3B_PRODUCTION_DART_HARDENING_VALIDATION.md)
- [`H4A — R8/ProGuard hardening and non-releasable validation`](history/H4A_R8_PROGUARD_HARDENING.md)
- [`H4B — Production-signed R8/ProGuard validation`](history/H4B_PRODUCTION_R8_PROGUARD_VALIDATION.md)
- [`H5A — Native/ELF audit and symbol policy`](history/H5A_NATIVE_ELF_AUDIT.md)
- [`H5B — Targeted native hardening and private symbol archive`](history/H5B_NATIVE_HARDENING.md)
- [`H5C — Production-signed native/ELF validation`](history/H5C_PRODUCTION_NATIVE_VALIDATION.md)
- [`UI-1 — Guided tour hardening`](history/UI-1_GUIDED_TOUR_HARDENING.md)
- [`PC-DEF-019 — Core rebuild and re-stage`](history/PC-DEF-019_CORE_REBUILD_RESTAGE.md)
- [`Final release exposure audit`](history/EXPOSURE_AUDIT.md)
- [`PC-DEF-020 — Public Mode authority fix`](history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md)
- [`PC-DEF-021 — AAB privacy and release-artifact policy`](history/PC-DEF-021_AAB_RELEASE_POLICY.md)
- [`PC-DEF-025 — Flutter suite and release-gate integrity`](history/PC-DEF-025_FLUTTER_SUITE_GATE.md)
