# PocketClaw milestone prompt template

Replace every angle-bracket placeholder with verified milestone-specific facts.
Delete irrelevant optional text rather than leaving ambiguous placeholders.

```text
POCKETCLAW — <PHASE / MILESTONE NAME>

Repository:
<repository path>

Working branch:
<branch>

Expected starting HEAD:
<full commit SHA-1>

Version:
<versionName+versionCode>

Accepted physical baseline:
<lastAcceptedVersionCode and accepted artifact identity>

Relevant fingerprints:
<Core, signer, payload or artifact digests required for this scope>

PURPOSE

<One concrete milestone outcome. State what this milestone proves or changes.>

CURRENT STATE

<Only verified state needed to understand the milestone. Distinguish accepted,
private-validation, candidate and released artifacts.>

PRE-FLIGHT

Independently verify:
- branch and exact HEAD;
- clean worktree and origin synchronization;
- version and accepted baseline;
- relevant fingerprints;
- develop/main, tags/releases and checkpoint refs;
- current milestone state and known defects.

STOP on unexplained drift.

ALLOWED WORK

- <Bounded change 1>
- <Bounded change 2>

REQUIRED VERIFICATION

- <Focused tests>
- <Source/build gate>
- <Exact artifact inspection, if authorized>
- <Hashes and independently derived facts>

INVARIANTS

- <Version/baseline rules>
- <Core/runtime/source rules>
- <Branch/tag/release/device rules>

DEFECT DISCOVERY

Classify every finding as:
A. MILESTONE BLOCKER
B. DIRECTLY RELATED DEFECT
C. UNRELATED / FUTURE DEFECT

Never hide evidence to obtain PASS. Fix only narrow in-scope defects. Record
unrelated findings in docs/DEFECT_LOG.md with evidence and a target milestone.

SECURITY

- <Secret-handling constraints>
- <Private-data and credential constraints>
- <Artifact exposure constraints>

COMMIT / PUSH

- Commit only <authorized file/scope classes>.
- Push only to <authorized remote branch>.
- Do not merge, tag, release or publish unless explicitly authorized here.

DO NOT

- <Named forbidden adjacent milestone work>
- <Forbidden build/device/release/ref operations>
- Begin the next milestone.

FINAL REPORT

Report:
1. starting/final branch and HEAD;
2. exact changes;
3. tests, gates, hashes and inspection evidence;
4. defects and dispositions;
5. secret/privacy result;
6. commit/push result;
7. unchanged invariants;
8. confirmation forbidden operations did not occur.

SUCCESS LINE

<EXACT SUCCESS ENDING>

OWNER ACTION LINE

<EXACT OWNER ACTION REQUIRED ENDING>

BLOCKED LINE

<EXACT BLOCKED ENDING>

STOP

Do not begin the next milestone automatically.
```

The prompt must identify whether an operation is read-only, a private validation,
a candidate build, physical acceptance, or publication. Never let the word
“release” blur those authorities.
