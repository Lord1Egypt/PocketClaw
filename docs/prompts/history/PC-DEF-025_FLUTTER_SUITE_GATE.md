# OPERATING RECORD — PC-DEF-025 Flutter suite and release-gate integrity

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `3e3941f0c3764e713096f1207d8a5808a81b2273` (PC-DEF-021 closeout).
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.
- **Core fingerprint:** `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df`
  before and after. No Core build input changed and Core was not rebuilt.

Tests, release tooling and documentation only. No product behaviour changed.
`PC-DEF-022`, `PC-DEF-023`, `PC-DEF-024`, `PC-DEF-006` and `PC-DEF-012` are
untouched; the updater, OAuth, manifest/deep-link, native export surfaces and
the AAB policy closed in `PC-DEF-021` were not modified. No production signing,
no production candidate, no Play key, no device, no ADB, no merge, tag, release
or publication.

## Two defects, one root

A stale assertion is ordinary. A stale assertion that stayed red for five
milestones without any gate noticing is the real finding, and the second half of
this milestone is about that.

### Defect 1 — the assertion went stale

`test/unit/namespace_n3_native_identity_test.dart` asserted:

    expect(script, contains('build/picoclaw-android-arm64'));
    expect(script, contains('build/picoclaw-launcher-android-arm64'));

H5B (`aa24d9e`) made the Core output root overridable — `CORE_BUILD_DIR` and
`CORE_OUTPUT_ROOT`, which is what lets reproducibility runs build into
independent roots. The script stopped containing that composed literal, and the
test went red while the guarantee it stood for was entirely intact.

The assertion was wrong in a specific way worth naming: it pinned a **path**
when the contract is a **rename boundary**. Upstream emits two artifacts under
its own names; PocketClaw's identity is applied at the install step, not in the
upstream recipe. Where the build happens to write them is not part of that
contract and never was.

### Defect 2 — the gate could not see it

`tool/release_gate.py` ran three named files:

    android_release_contract_test.dart
    android_backup_exclusion_test.dart      → a1.contracts
    production_signing_contract_test.dart   → signing.production_contract
    android_runtime_secret_placement_test.dart → a2.placement_guards

Twenty-five other test files were outside the gate entirely. The suite could be
red and the gate green, which is what happened through H5B, H5C, UI-1,
`PC-DEF-019`, the exposure audit, `PC-DEF-020` and `PC-DEF-021`. It was found by
running `flutter test` by hand during the exposure audit, not by any gate.

## The corrected assertion

It now pins the rename boundary and the relocatability that broke it:

    for (final pair in const [
      ('picoclaw-android-arm64', 'libpocketclaw.so'),
      ('picoclaw-launcher-android-arm64', 'libpocketclaw-web.so'),
    ]) { … install -m 0755 "$CORE_OUTPUT_ROOT/<upstream>" "$JNI_LIBS/<shipped>" … }

Matched as a regex over the real `install` lines, so the pairing is what is
asserted rather than two independent substrings that could both appear for
unrelated reasons. It additionally requires:

- the private-support step consumes the same two upstream names
  (`source_binary=picoclaw-android-arm64`, `…-launcher-…`, and
  `--source "$CORE_OUTPUT_ROOT/$source_binary"`) — a rename that missed this
  would archive symbols for the wrong binary;
- `CORE_OUTPUT_ROOT=` exists, so the root stays relocatable;
- **no** `install -m 0755 "build/picoclaw-…"` — the hard-coded shape that went
  stale cannot come back.

It is therefore strictly stronger than what it replaced: it fails if the rename
boundary breaks, and it fails if someone reintroduces a fixed root, while
surviving legitimate output-root relocation.

**Mutation-tested against the real script**, both ways:

    rename the upstream artifact to pocketclaw-android-arm64   → test FAILS
    hard-code install -m 0755 "build/picoclaw-android-arm64"   → test FAILS

The script was restored byte-for-byte afterwards and `git diff` is empty for it.

## The new Flutter gate

`flutter.suite` runs the complete suite and is the acceptance criterion. Design
points that matter:

**Deterministic toolchain.** `find_flutter()` mirrors the existing `find_jdk()`:
the repository toolchain first, then `FLUTTER_ROOT`, then `PATH` last. A gate
that answers differently depending on which shell invoked it is not a gate, and
the old code went straight to `shutil.which("flutter")`.

**One run, many items.** The suite runs **once**, through
`flutter test --reporter json`, and every Flutter gate item is read from that
run. Four invocations would be four chances for the gate to disagree with itself
about the same tree, and the suite is the expensive part.

**The named contracts survive.** `a1.contracts`,
`signing.production_contract` and `a2.placement_guards` are still reported
individually, derived from the same run by matching failing suite paths. A
release record that said only "the suite passed" would lose which guarantee was
checked. What changed is that they are no longer *the whole* of the Flutter
gate.

**Failure cannot be laundered.** A non-zero exit is never PASS, even if every
parsed test succeeded. Exit 0 with no parsed results is a FAIL, not a pass — a
suite that did not run must not look like a suite that passed. Output is bounded
and names the failing suite and test.

## Proof against a real red suite

A deliberately failing test was placed in a file **none of the three named
contracts covers**, and the gate was run for real:

    gate exit = 1
    FAIL  flutter.suite   490 passed, 1 failed —
          …/test/unit/pc_def_025_probe_test.dart: deliberate failure outside
          the three named contract files
    PASS  a1.contracts
    PASS  signing.production_contract
    PASS  a2.placement_guards

The three named items stayed green, which is exactly the blind spot: before this
milestone that state was a passing gate. The probe was removed and left no
trace.

## Results

    flutter analyze                          No issues found
    flutter test                             490 passed, 0 failed — All tests passed
    tool/test_release_gate.py                35 tests (was 24)
    tool/test_artifact_policy.py             32 tests
    tool/test_build_hardened_android.py      12 tests
    tool/test_native_elf_audit.py            16 tests
    tool/test_native_support.py              11 tests
    tool/test_r8_hardening.py                 8 tests
    tool/test_no_active_pico.py              19 tests
    tool/test_create_release_keystore.py      6 tests — see below

The eleven new release-gate tests cover: full-suite success passing every
Flutter item; one failure outside the named files failing the gate; the failing
test's identity appearing in the output; bounded output under 100 failures;
non-zero exit never reported as PASS; exit 0 with no results failing; the
command being `flutter test` rather than the three named files, invoked exactly
once; a missing toolchain skipping rather than passing; a named contract failing
when its own file fails; `PATH` being last in resolution order; and resolution
being stable across calls.

`tool/test_create_release_keystore.py` fails in a shell without `keytool` on
`PATH` and passes 6/6 with the repository JDK on `PATH`. It is an environment
prerequisite rather than a defect, it is identical with this milestone's changes
stashed, and it is not part of the release gate. Recorded rather than "fixed" or
omitted.

## Gate totals and the 55-vs-56 reconciliation

The `PC-DEF-021` owner report said the full APK gate was **56 PASS / 0 FAIL / 0
SKIP**; its commit message said **55 PASS / 0 FAIL**. Rather than assume either,
the gate was re-run at that exact commit.

Measured at `3e3941f` on a clean tree: **56 PASS / 0 FAIL / 0 SKIPPED**, 56
items. Measured with this milestone's work uncommitted: **55 PASS / 1 SKIPPED**,
also 56 items.

So both numbers were real runs of the same 56-item gate. `repo.clean_worktree`
reports SKIPPED on a dirty tree and PASS on a clean one; the commit message was
written from a run made before the commit existed, and the owner report from the
run after it. **No inconsistency in gate execution or evidence, and no defect
is warranted** — the composition never changed, one item's status did.

This milestone does change composition, by one: `flutter.suite` is new.

    source gate      25 → 26 items
    full APK gate    56 → 57 items

Final totals at this closeout are recorded in the owner report.

## Closeout conditions

1. The stale assertion is corrected without weakening its guarantee — it is
   strictly stronger, and mutation-tested both ways.
2. `flutter test` is completely green — 490 passed, 0 failed.
3. The release gate runs the complete Flutter suite — `flutter.suite`.
4. A deliberately failing suite fails the gate — proven for real, and covered by
   eleven unit tests.
5. `flutter analyze` is clean.
6. Source release gates remain green in both classes.
7. Core fingerprint unchanged; Core not rebuilt.
8. No unrelated defect hidden or fixed; the one pre-existing environment-
   dependent test failure is characterised rather than silently passed over.
