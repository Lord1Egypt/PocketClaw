# PocketClaw working method

PocketClaw work is milestone-driven. This protocol governs implementation,
review, closeout, and handoff for humans and AI agents.

## Mandatory milestone loop

```text
PLAN
→ VERIFY CURRENT STATE
→ EXECUTE ONE SMALL SCOPE
→ DISCOVER DEFECTS
→ CLASSIFY DEFECTS
→ FIX ONLY IN-SCOPE DEFECTS
→ LOG DEFERRED DEFECTS
→ TEST / GATE / HASH / INSPECT
→ REPORT EVIDENCE
→ REVIEW PASS / BLOCKER / OWNER ACTION REQUIRED
→ STOP
```

`STOP` means stop. An implementation agent never begins the next milestone
automatically.

## Before every milestone

Independently verify, rather than copying the prompt's claims:

- branch and exact starting HEAD;
- clean worktree and synchronization with origin;
- tracked version and accepted physical baseline;
- relevant source, binary, artifact, and signer fingerprints;
- `develop`, `main`, current tags/releases, and applicable checkpoint refs;
- existing milestone state and open defects in the authoritative docs.

Unexplained drift blocks the milestone. A documentation-only continuation is
acceptable only when every intervening commit is explained and in scope.

## Required prompt contract

Every implementation prompt defines:

- `PURPOSE`;
- `CURRENT STATE` and `PRE-FLIGHT`;
- `ALLOWED WORK`;
- `REQUIRED VERIFICATION`;
- `INVARIANTS`;
- `FORBIDDEN WORK` / `DO NOT`;
- `SECURITY RULES`;
- `COMMIT / PUSH RULES`;
- `FINAL REPORT FORMAT`;
- exact `SUCCESS`, `BLOCKED`, and `OWNER ACTION REQUIRED` endings;
- mandatory `STOP`.

Use [`prompts/PROMPT_TEMPLATE.md`](prompts/PROMPT_TEMPLATE.md) rather than
creating a different structure for each agent or model.

## Scope and authority rules

1. Work in small, explicitly scoped milestones.
2. Do not combine unrelated cleanup with release work.
3. Never silently bump the version, advance the accepted baseline, rebuild
   Core, modify Managed Runtime payloads, merge branches, move tags, create or
   modify releases, publish artifacts, or access a physical device.
4. Those actions require explicit authorization even when they seem like the
   obvious next step.
5. Historical acceptance records are immutable facts. A new artifact on a
   branch does not retroactively change the accepted baseline.
6. Prefer exact commits, hashes, tests, gates, artifact inspection, and
   reproducible commands over assumptions or narrative confidence.
7. Treat skips as named states. Never silently convert a skip or known limit
   into a pass.

## Defect discovery is part of the work

This operating method has already surfaced and helped resolve dozens of defects
during PocketClaw development. That record does not authorize opportunistic
refactoring.

Every discovered defect is classified:

### A. Milestone blocker

The defect prevents safe completion of the current milestone. Fix it only when
the fix is narrowly required, then fully re-test the affected path and the
milestone gate.

### B. Directly related defect

The defect is caused by, or tightly coupled to, the current milestone. A narrow,
low-risk fix may remain in scope. Report the finding, root cause, fix, and
verification explicitly.

### C. Unrelated / future defect

The defect is real but outside the milestone. Do not refactor around it. Add it
to [`DEFECT_LOG.md`](DEFECT_LOG.md) with evidence, the reason for deferral, and a
target phase.

Never hide a defect to obtain `PASS`. Never ignore evidence that invalidates a
previous assumption. Never use discovery as permission for uncontrolled
refactoring.

## Implementation and review split

After implementation, the implementation agent returns concrete evidence. A
reviewer evaluates the scope, Git state, invariants, tests, hashes, gates,
secrets handling, defects, and unresolved skips under
[`prompts/REVIEW_PROTOCOL.md`](prompts/REVIEW_PROTOCOL.md).

The review result is exactly one of:

- `PASS`
- `BLOCKER`
- `OWNER ACTION REQUIRED`

Only `PASS` authorizes preparation of the next milestone prompt. It does not
authorize that milestone's implementation by itself.

## Documentation closeout contract

A milestone is not fully closed until its authoritative documentation is
updated in the same closeout. Normally update:

- [`PROJECT_STATE.md`](../PROJECT_STATE.md);
- [`ROADMAP.md`](ROADMAP.md).

When relevant, also update:

- [`DEFECT_LOG.md`](DEFECT_LOG.md);
- [`DECISIONS.md`](DECISIONS.md);
- [`RELEASE_PROCESS.md`](RELEASE_PROCESS.md);
- the milestone record under [`prompts/`](prompts/README.md).

The closeout must say what changed, what passed, what stayed invariant, what is
still open, and what is authorized next. Then stop.
