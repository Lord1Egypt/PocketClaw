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

H2 developer production signing, H3A/H3B Dart hardening, H4A/H4B R8/ProGuard
hardening, H5A native/ELF audit, H5B targeted native hardening and **H5C
production-signed native/ELF validation** are all closed.

H5B was confirmed on hardware: the exact LOCAL TEST APK was installed in place
on the Samsung SM-A165F as a same-identity upgrade with all application data
preserved, and the owner's manual native/runtime smoke returned PASS with no
failing item. H5C then rebuilt the same tree under the enrolled production
signer and proved all 18 packaged ELF entries byte-identical to that physically
validated artifact, so the device result transfers to the production APK.

The production artifact is private validation evidence:
63,564,563 bytes, SHA-256
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`, one v2
signer `176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`. The
production release gate passes 55 PASS / 0 FAIL / 0 SKIPPED with
`releasable: true`. It was not installed, published, accepted, or committed, and
the accepted physical baseline remains vc62 / versionCode 62.

**Do not install the production APK over the current device state.** The
installed path is development-signed; the production certificate is a different
identity and a cross-signer `adb install -r` is forbidden. That transition is a
separately authorized migration / clean-install / data-safeguard milestone.

UI-1, an isolated guided-tour repair authorized between H5C and the exposure
audit, then fixed `PC-DEF-013` through `PC-DEF-018` in the console. Because
`libpocketclaw-web.so` embeds the dashboard, that moved the Core source
fingerprint to
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec` and left the
staged pair stale — tracked as `PC-DEF-019` and now **resolved**. Core was
rebuilt and re-staged from canonical build-input commit `ea43369` under the
two-commit rule:

    libpocketclaw.so       37,724,640  f273b9ce…  build ID 25e206ab…
    libpocketclaw-web.so   25,517,952  900c43fc…  build ID 84afbe24…
    BuildTime              2026-09-12T07:27:12+0000

Byte-identical in three independent output roots; `core.staged_freshness` and
`build.reproducibility_tests` PASS; 22 PASS / 0 FAIL on the native contract for
the pair. No APK or AAB was built, so the private native support manifest is
still bound to the H5C candidate and is rebound at the next artifact build.

**The H5C production APK is evidence for the pre-UI-1 dashboard.** It is not
relabelled; the next artifact build is the first to contain the current Core
pair, and it needs its own validation.

`PC-DEF-009` through `PC-DEF-011`, `PC-DEF-013` through `PC-DEF-022`,
`PC-DEF-024` and `PC-DEF-025` are resolved.

**`PC-DEF-024` was resolved by removing the filter, not making it conditional.**
The `um.placeholder` `VIEW`/`DEFAULT`/`BROWSABLE` filter, its Gradle
link-scheme plumbing and `MainActivity.logIncomingIntent` are gone; the packaged
merged manifest has zero `BROWSABLE`, `android:scheme` and `action.VIEW` while
`MAIN`/`LAUNCHER` still works. Do not reintroduce a placeholder scheme — an
analytics build declares what it needs in its own manifest.
`POCKETCLAW_UMENG_APP_KEY`, `_CHANNEL` and `_PACKAGED` stay: `AnalyticsReporter`
and two `meta-data` entries consume them.

**`PC-DEF-023` is RESOLVED: Google Antigravity is not shipped in v0.2.0**
(owner decision `PC-D014`). The provider is gone from the OAuth API, the
catalogue, the factory, the default `model_list`, the CLI — including the
`auth models` subcommand — the dashboard, all fourteen locale bundles and the
embedded agent guidance. Both credentials are absent from source and from both
Core binaries.

**Do not reintroduce it with a third party's OAuth client.** That is the thing
the decision forbids; it may return only under a PocketClaw-owned integration.
**Gemini is a different provider and was not touched** — API-key auth against
`generativelanguage.googleapis.com`, with a test pinning it and its `google`
alias. Shared OAuth infrastructure stays, including `ClientSecret` and the
confidential-client token exchange.

Every cleanup item from the exposure audit is now closed. `PC-DEF-006` gates the
F-Droid reproducibility path only and `PC-DEF-012` awaits reachability evidence;
neither blocks the GitHub / direct APK release.

**`PC-DEF-022` was resolved by removing the route, not by hardening it.**
`POST /api/update` is gone from source and from both Core binaries; `embed.go`'s
existing not-found path makes it an ordinary 404. `pkg/updater` stays because
`cmd/picoclaw` registers its CLI update command — do not delete the library, and
do not reintroduce an HTTP download endpoint under any name. The current Core generation is fingerprint
`bc35a598d3a836e0a0c95afc73314fe49a38877b985b5b0f15bab11460184fa9`, staged pair
`0a28bd5e…` / `9ae1d2d9…`, BuildTime `2026-09-13T00:24:56+0000`, from
build-input commit `54ff2525fa555744d017aae56c9a26e2049812e1`. `PC-DEF-012` stays open by decision — no reachability evidence, so no
export narrowing; the exposure audit produced none and did not narrow anything.

Full evidence is in
[`prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md`](prompts/history/H5C_PRODUCTION_NATIVE_VALIDATION.md)
and [`prompts/history/H5B_NATIVE_HARDENING.md`](prompts/history/H5B_NATIVE_HARDENING.md);
H5A's 18-entry ELF inventory and category policy remain in
[`prompts/history/H5A_NATIVE_ELF_AUDIT.md`](prompts/history/H5A_NATIVE_ELF_AUDIT.md).
The Core rebuild/re-stage evidence is in
[`prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md`](prompts/history/PC-DEF-019_CORE_REBUILD_RESTAGE.md).

The **final release exposure audit has run and is BLOCKED.** It completed every
part that does not depend on the production signing identity, opened six
defects and fixed none.

    PC-DEF-020  Public Mode OFF is not durable across a restart      RESOLVED
    PC-DEF-021  AAB embeds the private R8 mapping and Dart symbols   RESOLVED
    PC-DEF-022  /api/update fetches an arbitrary URL unverified      RESOLVED (removed)
    PC-DEF-023  third-party Google OAuth client secret embedded      RESOLVED (provider removed)
    PC-DEF-024  dead analytics deep link exported in the manifest    RESOLVED (removed)
    PC-DEF-025  flutter test red since H5B; no gate runs the suite   RESOLVED

**`PC-DEF-020` is now RESOLVED.** The Android host always states the Public Mode
decision (`-public=true` / `-public=false`), so the persisted
`launcher-config.json` `public` field is never consulted there and "off"
survives every restart. The Config page reports the effective mode and a save
repairs a stale stored `true`. That fix moved the Core source fingerprint to
`2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df`; the staged
pair is `602ce034…` / `b5cce071…`, BuildTime `2026-09-12T18:54:26+0000`, rebuilt
and re-staged under the two-commit rule.

**`PC-DEF-021` is now RESOLVED.** Every artifact phase declares what the
artifact is FOR — `--artifact-class public-release | play-upload |
non-publish-audit` — and has no default, so an unclassified artifact fails
closed. **An AAB may never be `public-release`**, regardless of what it
contains, and detection reads the archive rather than the file extension.

    # Publish this
    python3 tool/release_gate.py --full <apk> --release-class production \
      --artifact-class public-release
    # Upload this to Play, and nowhere else
    python3 tool/release_gate.py --verify-bundle <aab> --artifact-class play-upload
    # Never valid
    python3 tool/release_gate.py --verify-bundle <aab> --artifact-class public-release

**Never attach an AAB to a public release.** The hardened APK is the only
public Android binary. `tool/release_gate.py --release-assets <names…>` checks a
proposed asset list, and `tool/build_hardened_android.py --package bundle
--artifact-class …` is the repository-owned hardened bundle path.

`PocketClaw-v0.2.0-rc1.aab` and `-rc2.aab` are still attached to their published
pre-releases. That was left alone deliberately — mutating published releases was
not authorized — and the owner action for removal is recorded in
`RELEASE_PROCESS.md`.

**The production-signed v0.2.0 candidate exists and passes every gate.**

    path    build/app/outputs/apk/release/app-release.apk
    bytes   63,472,307
    sha256  4d4bc33a63059450383c4eedb34e2902486fbbc8c0d85b91413ab9654b4f3dac
    signer  one v2 signer, 176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf
    gate    57 PASS / 0 FAIL / 0 SKIPPED — "PASS — production release candidate"

Core generation `bc35a598…`, BuildTime `2026-09-13T00:24:56+0000`, native audit
194/0/0, and the private native support manifest is bound to this exact APK.
Evidence:
[`prompts/history/V0_2_0_PRODUCTION_CANDIDATE.md`](prompts/history/V0_2_0_PRODUCTION_CANDIDATE.md).

**Do not rebuild it casually.** That hash is the frozen candidate; any rebuild
supersedes it and every artifact check must restart against the new hash,
including the native-support rebinding.

**Provenance warning.** Two superseded literals live in commit messages —
fingerprint `f9a2d2a8…` in `54ff252`, BuildTime `00:23:11` in `1c477e6`. The
authoritative values are `bc35a598…` and `2026-09-13T00:24:56+0000`, which the
binaries carry, and neither superseded literal appears in any binary or tracked
document. Do not quote the commit-message values as artifact provenance, and do
not rewrite those commits.

**The next milestone is Samsung physical acceptance**, under its own explicit
prompt and with a migration / clean-install / data-safeguard plan.

**Do not install the production APK over the current device state.** The
installed path is development-signed; that transition needs its own migration /
clean-install / data-safeguard milestone.

**Do not publish an AAB as a release asset.** `PocketClaw-v0.2.0-rc1.aab` and
`-rc2.aab` already are, and a hardened bundle carries the complete R8
deobfuscation map and the Dart AOT symbols. `PC-DEF-021` covers the policy and
tooling correction; whether to remove the existing published bundles is an
owner decision recorded there.

`PC-DEF-002` is resolved as explained — `0.0.0.0:18800` is Public Mode ON and
nothing else, and the Core gateway stays loopback-only in both states — with
`PC-DEF-020` as its one actionable residue. `PC-DEF-006` stays open: one
artifact was built, not two compared.

**THE FINAL RELEASE EXPOSURE AUDIT IS CLOSED / PASS.** The re-run at `6031898`
found **no remaining release blocker** for the intended GitHub / direct APK
stable release and opened no new defect. `PC-DEF-020`, `PC-DEF-021` and
`PC-DEF-025` were each re-validated from the current tree, not taken on trust.
Evidence:
[`prompts/history/EXPOSURE_AUDIT_RERUN.md`](prompts/history/EXPOSURE_AUDIT_RERUN.md).

**`PC-DEF-025` is RESOLVED.** `flutter test` is green — 490 passed, 0 failed —
and the **complete** suite is now a release gate (`flutter.suite`), resolved
through a deterministic `find_flutter()` that prefers the repository toolchain
and puts `PATH` last. The gate previously ran three named files, so the suite
stayed red for five milestones while the gate reported green; that blind spot is
closed, and a non-zero exit or an unrun suite can no longer be reported as PASS.

Gate composition grew by one item: source 25 → 26, full APK 56 → 57. The earlier
55-vs-56 reporting difference was reconciled — same 56 items, with
`repo.clean_worktree` reporting SKIPPED on a dirty tree and PASS on a clean one.
No defect.

The next milestone is a **re-run of the Final Release Exposure Audit to
closure**, under its own explicit prompt, against the current Core generation.
Full evidence is in
[`prompts/history/EXPOSURE_AUDIT.md`](prompts/history/EXPOSURE_AUDIT.md),
[`prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md`](prompts/history/PC-DEF-020_PUBLIC_MODE_AUTHORITY.md)
[`prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md`](prompts/history/PC-DEF-021_AAB_RELEASE_POLICY.md)
and [`prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md`](prompts/history/PC-DEF-025_FLUTTER_SUITE_GATE.md).

Do not create a stable tag, publish a release, access a device, rebuild Core or
Managed Runtime, or advance the accepted baseline without explicit milestone
authority.
