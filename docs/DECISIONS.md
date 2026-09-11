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

## PC-D011 — Build Dart-hardened Android artifacts through one fail-closed contract

- **Status:** Accepted by H3A and production-validated by H3B.
- **Decision:** `tool/build_hardened_android.py` is the canonical Android
  release entry point. It couples arm64 release compilation, Dart obfuscation,
  external split debug info, and Flutter's generated-source package mapping.
  Gradle refuses release compilation when that contract is absent or malformed.
- **Rationale:** Flutter 3.47.1 accepts `dart-obfuscation` and
  `split-debug-info` through its Gradle plugin, but its filesystem root/scheme
  task fields do not reach `flutter assemble`. Pub's normal package config also
  leaves the generated Dart plugin registrant outside a package URI root,
  producing an absolute checkout URI in `libapp.so`.
- **Consequences:** The generated registrant is identified as
  `package:pocketclaw_generated/dart_plugin_registrant.dart`. Dart DWARF lives
  by default under ignored `build/private-symbols/dart/android-arm64/`, remains
  private release-support material, and is verified beside the APK rather than
  packaged into it. It is preserved privately for crash symbolization and is
  not a signing secret, Git input, or public release asset. Any future Flutter
  upgrade must revalidate the generated-package mechanism and artifact checks.
  The helper clears only Flutter's generated build cache before assembly because
  Flutter 3.47.1 does not track external split DWARF as an incremental output;
  this keeps the AOT and its private symbol companion from separating.
- **Evidence:** H3A helper and Gradle guard; focused Python/Dart tests; two clean
  builds with byte-identical `libapp.so` and split DWARF across different output
  roots; H3A local-test and H3B production artifact gates each 25 PASS / 0 FAIL
  / 0 SKIPPED.

## PC-D012 — Keep application R8 rules narrow and mappings private

- **Status:** Accepted by H4A; production validation pending in H4B.
- **Decision:** Release builds keep R8 minification, resource shrinking, and
  optimized defaults enabled. PocketClaw adds no application-wide keep rule;
  manifest/aapt rules, Flutter's pinned embedding contract, annotations, and
  dependency consumer rules retain actual entry points. Every future custom
  keep must cite a concrete reflection, JNI, or framework need and have focused
  regression coverage. Fresh `mapping.txt` and `usage.txt` are mandatory
  hardened-build evidence.
- **Rationale:** The former project rule retained all PocketClaw, Flutter, and
  plugin classes, leaving application implementation names unchanged even
  though R8 executed. The merged H4A configuration proved the four Android
  components already have generated constructor keeps and found no app-owned
  reflective serialization or JNI entry point needing a blanket rule.
- **Consequences:** The R8 mapping and sibling reports live under ignored
  `build/app/outputs/mapping/release/`, remain outside APK/AAB files and public
  assets, and must be preserved privately with a production release for
  Java/Kotlin deobfuscation. Dependency-owned broad consumer rules remain
  visible audit boundaries and are not overridden without dependency-specific
  evidence.
- **Evidence:** H4A mapping SHA-256
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb`;
  DEX reduction of 614,820 bytes from H3B; six sampled internal descriptors
  renamed/removed/folded and absent from DEX; four manifest components
  preserved; focused R8 tests and 30/30 local-test artifact gate.
