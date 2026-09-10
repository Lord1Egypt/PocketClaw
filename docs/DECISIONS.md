# PocketClaw durable decision register

This is the concise authoritative register for decisions that constrain future
work. The root [`DECISIONS.md`](../DECISIONS.md) remains the detailed historical
decision journal and supplies the underlying engineering narratives.

## PC-D001 — Maintain an independent PocketClaw repository

- **Status:** Accepted.
- **Decision:** PocketClaw keeps independent Git history. PicoClaw FUI is a
  manually reviewed reference, while the pinned Core source is vendored under
  `core/src/` and built from this repository.
- **Rationale:** A permanent FUI merge path and an external Core source checkout
  would make review and reproducibility depend on unrelated history or local
  machine state.
- **Consequences:** Upstream changes are classified and selectively adapted;
  upstream attribution and licenses remain intact.
- **Evidence:** Root `DECISIONS.md`, “Independent repository, not a permanent
  FUI fork” and “The PocketClaw repository is the canonical source-of-truth”;
  [`UPSTREAM_BASELINE.md`](../UPSTREAM_BASELINE.md).

## PC-D002 — Preserve three distribution targets

- **Status:** Accepted.
- **Decision:** The release plan supports Direct APK, Google Play, and Official
  F-Droid.
- **Rationale:** All three are confirmed product channels, and distribution
  constraints must be designed into signing and reproducibility before the
  first stable release.
- **Consequences:** One canonical build is preferred; artifacts remain subject
  to channel-specific signing and inspection rules.
- **Evidence:** [`RELEASE_SIGNING.md`](RELEASE_SIGNING.md), “Three distribution
  channels, three distinct keys”; [`FDROID_RELEASE.md`](FDROID_RELEASE.md).

## PC-D003 — Keep the developer app-signing key separate from the Play upload key

- **Status:** Accepted.
- **Decision:** The PocketClaw developer app-signing key signs Direct APK and
  the target developer-signed F-Droid artifact. A future Google Play upload key
  is a separate credential.
- **Rationale:** An upload key is replaceable and serves authentication to Play;
  it should not expose the long-lived identity governing direct/F-Droid update
  continuity.
- **Consequences:** No Play upload key exists yet. Creating one requires its own
  explicitly authorized milestone.
- **Evidence:** [`RELEASE_SIGNING.md`](RELEASE_SIGNING.md), sections 8A and 8B.

## PC-D004 — Treat Google Play App Signing as a separate lineage

- **Status:** Accepted.
- **Decision:** Under Google Play App Signing, the certificate on Play-delivered
  installs is Google's app-signing lineage, separate from direct/F-Droid.
- **Rationale:** Giving Google the developer key merely to force cross-channel
  equality would expand custody of the identity governing every other channel.
- **Consequences:** Play installs are not assumed mutually updatable with
  direct/F-Droid installs. Direct and F-Droid target continuity with each other.
- **Evidence:** [`RELEASE_SIGNING.md`](RELEASE_SIGNING.md), section 8C.

## PC-D005 — Keep private signing material outside Git

- **Status:** Accepted and enforced.
- **Decision:** Keystores, private keys, signing passwords, and recovery secrets
  never enter the repository, command arguments, project properties, logs, or
  reports. Only public certificate and artifact fingerprints are tracked.
- **Rationale:** A secret committed once remains in history; build defaults also
  previously allowed plausible but wrongly signed artifacts.
- **Consequences:** Production signing uses environment input, fails closed, and
  requires owner-controlled hidden input or secret storage.
- **Evidence:** Root `DECISIONS.md`, “A release states its signer, its version
  and its dependencies, or it fails”; [`RELEASE_SIGNING.md`](RELEASE_SIGNING.md);
  `492759b` and `78d33fd`.

## PC-D006 — Advance the accepted baseline only on physical acceptance

- **Status:** Accepted and enforced.
- **Decision:** vc62 / `lastAcceptedVersionCode=62` remains the accepted physical
  baseline until a later candidate passes an explicitly authorized physical
  acceptance milestone.
- **Rationale:** The baseline is an install floor representing observed device
  acceptance, not the newest artifact built.
- **Consequences:** Private validation, hardening, and superseded candidates do
  not advance it. The advance occurs in the same commit that records acceptance.
- **Evidence:** `android/release-baseline.properties`; `54740f2`; root
  `DECISIONS.md`, “A release states its signer, its version and its dependencies,
  or it fails.”

## PC-D007 — Make source builds and reproducibility release requirements

- **Status:** Accepted; APK-level proof remains open.
- **Decision:** Core and Managed Runtime inputs must be buildable from pinned
  source. Identical-input reproducibility is required for the target
  developer-signed F-Droid path and must be rechecked through hardening.
- **Rationale:** F-Droid can preserve the developer signature only when it can
  reproduce the submitted APK bit-for-bit.
- **Consequences:** Hardening outputs, mapping/symbol files, timestamps, APKs,
  and AABs require explicit deterministic-build evidence. Committed native
  prebuilts and F-Droid builder compatibility remain open submission issues.
- **Evidence:** [`FDROID_RELEASE.md`](FDROID_RELEASE.md); commits `6f117d3`,
  `51ed822`, `e62d67c`, `dea826c`, and `fb38c7d`.

## PC-D008 — Use the milestone/evidence/STOP workflow

- **Status:** Accepted by PC-1.
- **Decision:** Every engineering change follows the scoped milestone loop in
  [`WORKING_METHOD.md`](WORKING_METHOD.md), receives an independent review, and
  stops before the next milestone.
- **Rationale:** Existing work repeatedly found defects by verifying assumptions
  against code, artifacts, devices, and Git rather than carrying them forward.
- **Consequences:** Only reviewer `PASS` permits preparation of the next prompt;
  new implementation still requires explicit authorization.
- **Evidence:** Historical milestone records under [`prompts/history/`](prompts/history/);
  repository closeout records; PC-1 owner instruction.

## PC-D009 — Treat repository documentation as authoritative project memory

- **Status:** Accepted by PC-1.
- **Decision:** The current snapshot and protocol documents in the repository,
  corroborated by Git, are authoritative. Chat history and model memory are
  advisory.
- **Rationale:** Future engineers and agents must recover state without access
  to a particular conversation or model session.
- **Consequences:** Documentation updates are part of milestone closeout;
  historical archives are preserved but cannot override the current snapshot.
- **Evidence:** [`AI_HANDOFF.md`](AI_HANDOFF.md),
  [`WORKING_METHOD.md`](WORKING_METHOD.md), and the PC-1 closeout commit.

## PC-D010 — Create immutable checkpoints before high-risk hardening

- **Status:** Accepted.
- **Decision:** Preserve named branch/tag checkpoints before signer ceremonies
  and high-risk final hardening work.
- **Rationale:** Rollback must resolve to an exact, reviewed source state rather
  than a remembered point in a moving branch.
- **Consequences:** Existing checkpoints are inspected, not moved. New ones
  require explicit milestone authority and a factual annotation.
- **Evidence:** `checkpoint/vc62-accepted` and annotated tag
  `checkpoint-vc62-accepted` at `47cde00`; `checkpoint/pre-h2` at `fb38c7d`.
