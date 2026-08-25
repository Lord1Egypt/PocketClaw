# PocketClaw Upstream Tracking

Last Review: 2026-08-24

## PicoClaw Core

Pinned: `v0.3.1`  
Last reviewed release: `v0.3.1`  
Reviewed source commit: `2cf030d2fd3b871d7ec17e3be34c24688aac76da`

## PicoClaw FUI

Baseline: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`  
Last reviewed commit: `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`

## Adopted Changes

- Android active-network DNS integration
- Optional feedback behavior

## Verified Upstream-Backed Behaviors

- ClawHub Skill Hub search is available after the Android active-network DNS
  fix. The prior registry-unavailable symptom is resolved by DNS, not pending
  an independent FUI/registry change.

## Pending Review

None. Product identity and original PocketClaw visual assets are maintained in
this repository and are not upstream imports.

## Security Updates

None currently recorded.

Upstream changes are classified and manually reviewed before selective
adaptation. Automated merges from upstream are not a maintenance model.
