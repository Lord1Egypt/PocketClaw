# PocketClaw review protocol

The reviewer evaluates an implementation report against Git, repository state,
and independently reproducible evidence. Confidence, effort, and a plausible
narrative do not substitute for proof.

## Reviewer checks

- Did the agent remain inside the authorized scope?
- Does Git state support the report, including branch, exact commits, clean
  worktree, remote synchronization, and changed-file list?
- Were version, accepted baseline, Core/runtime state, checkpoint refs, tags,
  releases, and device boundaries preserved where required?
- Are hashes, tests, gates, and artifact inspections meaningful and tied to the
  exact source or artifact claimed?
- Did anything become silently accepted, releasable, installed, published, or
  baseline-advancing?
- Were secrets or private material exposed in prompts, argv, environment dumps,
  files, logs, reports, or Git?
- Were every failure and skip named and explained?
- Were blockers fixed narrowly and fully re-tested?
- Were unrelated defects logged rather than mixed into the milestone?
- Did new evidence invalidate an earlier assumption, and was the record
  corrected rather than defended?
- Is authoritative documentation updated in the same closeout?
- Is the milestone genuinely complete under its own success criteria?

## Valid result

The review result must be exactly one of:

### PASS

All required evidence supports completion, scope and invariants held, and no
unresolved blocker remains. `PASS` authorizes preparation of the next milestone
prompt. It does not authorize executing that milestone.

### BLOCKER

Evidence shows the milestone is incomplete or unsafe, or a required invariant
failed. State the exact blocker and the evidence needed to clear it. Do not
advance the roadmap.

### OWNER ACTION REQUIRED

Completion depends on an owner-only secret, physical action, irreversible
decision, account operation, or other explicitly owner-held authority. State
the exact local action without requesting secret values in chat or repository
files. Resume only after the owner reports completion.

After issuing one result, stop. A reviewer does not begin implementation or
write the next milestone prompt in the same review unless separately authorized.
