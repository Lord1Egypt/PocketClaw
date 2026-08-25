# PocketClaw Upstream Tracking

This file is the checkpoint of record for upstream review. It answers one
question: what upstream state has already been reviewed, so the next review
starts after that point instead of restarting from the baseline.

The immutable adoption record — which upstream release and commit PocketClaw
originally started from, and when — lives in `UPSTREAM_BASELINE.md` under
"Adoption record". This file records movement; that file never changes.

## Checkpoint

### PicoClaw Core

Upstream repository:
https://github.com/sipeed/picoclaw

Baseline release:
v0.3.1

Baseline commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Last upstream review date:
2026-08-25

Last reviewed upstream release:
v0.3.1

Last reviewed upstream commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Latest upstream release known at review time:
v0.3.1 (published 2026-07-03; verified against the GitHub releases API on
2026-08-25 — `releases/latest` and the full tag list both name v0.3.1 as the
newest version tag)

Latest upstream commit/tag known at review time:
`bbf6893ca7afad27f1d00a0f5a45982a549c6ed6` on `main`, authored 2026-08-19
(`feat(models): add configurable default fallback chain (#3200)`). The rolling
`nightly` release tag still points at the v0.3.1 commit.

Distance from current upstream:
PocketClaw is level with the newest upstream *release* and 19 first-parent
commits behind upstream `main`. The raw `2cf030d2..main` range counts 2583
commits because `main` has merged in a second, unrelated root history; the
first-parent count is the meaningful figure for review planning.

Next review must begin AFTER:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

### PicoClaw FUI

Upstream repository:
https://github.com/sipeed/picoclaw_fui

Baseline commit:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

Last upstream review date:
2026-08-25

Last reviewed upstream release:
picoclaw_fui-v0.1.4 (tag `v0.1.4`)

Last reviewed upstream commit:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

Latest upstream release known at review time:
picoclaw_fui-v0.1.4 (published 2026-06-04; verified against the GitHub
releases API on 2026-08-25)

Latest upstream commit/tag known at review time:
`d689c94c1b67f625f70ec4111a9aa3f01be9cbb3` — `main` HEAD equals the baseline
commit and equals tag `v0.1.4`.

Distance from current upstream:
Zero commits behind. The FUI has not moved since the baseline.

Next review must begin AFTER:
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3

## Upstream Review History

This history is append-only. Never edit or delete an existing entry; every
review adds a new one below the last.

### Review 1 — 2026-08-25

PocketClaw Core baseline:
v0.3.1

Baseline commit:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Reviewed upstream from:
2cf030d2fd3b871d7ec17e3be34c24688aac76da

Reviewed upstream through:
2cf030d2fd3b871d7ec17e3be34c24688aac76da (Core)
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3 (FUI)

Latest release observed:
Core `v0.3.1` (2026-07-03); FUI `picoclaw_fui-v0.1.4` (2026-06-04). Both
verified live against the GitHub API on 2026-08-25.

Scope of this review:
Establishing the checkpoint. This review verified the adoption record against
real upstream metadata and measured the distance to current upstream. It did
not classify the 19 first-parent commits that landed on Core `main` after the
baseline — those remain unreviewed and are the subject of the next review.

Changes adopted:

- Android active-network DNS integration, carried from the baseline commit
  (`ConnectivityManager.activeNetwork` → `LinkProperties.dnsServers` →
  `PICOCLAW_DNS_SERVER` → the Go runtime).
- Optional device-feedback behavior, carried from the baseline commit.

Changes reviewed but not adopted:

- The 19 first-parent commits on Core `main` after v0.3.1 were counted and
  bounded but not classified. Nothing from that range is adopted.
- Core `main` also carries a merged unrelated root history, which inflates the
  raw commit count; it is not upstream product work to adapt.

Not upstream work at all (PocketClaw-authored, recorded here so a future
review does not mistake it for an adoption):
product identity and original PocketClaw visual assets; the Milestone C
provider catalog, presets, category metadata, and provider-first UI; the
Gemini discovery branch; the OpenCode Zen/Go presets and per-model protocol
routing; and the generic Responses provider. These live in
`core/pocketclaw-core-v0.3.1.patch`.

PocketClaw commit(s):
`950d4a3` (baseline adoption) through `b1fbee3` on `feature/provider-catalog`.

Next review must begin AFTER:
2cf030d2fd3b871d7ec17e3be34c24688aac76da (Core)
d689c94c1b67f625f70ec4111a9aa3f01be9cbb3 (FUI)

## Review Workflow

When PocketClaw is asked "check what is new in PicoClaw", the workflow is
fixed:

1. Read the last reviewed-through commit from this file's Checkpoint section.
2. Fetch current upstream.
3. Compare **only** changes after that checkpoint.
4. Classify the useful changes.
5. Selectively adapt them into PocketClaw.
6. Update the Checkpoint section and append a new Review entry.

Never restart upstream analysis from v0.3.1 once later commits have been
reviewed. Never overwrite review history.

Upstream changes are classified and manually reviewed before selective
adaptation. Automated merges from upstream are not a maintenance model.

## Verified Upstream-Backed Behaviors

- ClawHub Skill Hub search is available after the Android active-network DNS
  fix. The prior registry-unavailable symptom is resolved by DNS, not pending
  an independent FUI/registry change.

## Security Updates

None currently recorded. Note that upstream `49183d7e`
(`fix: update Go and x/text for govulncheck`) sits in the unreviewed range and
is a candidate for the next review.
