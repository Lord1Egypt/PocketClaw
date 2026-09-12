# OPERATING RECORD — PC-DEF-021 AAB privacy and release-artifact policy

RECONSTRUCTED OPERATING RECORD. Evidence-based closeout, not the original prompt.

## Status and boundary

- **Status:** **RESOLVED.**
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `69479b4a6116c8ddf0ced49a213a177cb160eac6` (PC-DEF-020 closeout).
- **Version/baseline:** `0.2.0+62`; accepted physical baseline vc62 /
  `lastAcceptedVersionCode=62`, untouched.
- **Core fingerprint:** `2692de41b2fe2487475911b62cec519193d581b25cf6d0ebe935fc63973229df`
  before and after. No Core source changed and Core was not rebuilt.

Scope was `PC-DEF-021` only. `PC-DEF-022`, `PC-DEF-023`, `PC-DEF-024`,
`PC-DEF-025`, `PC-DEF-006` and `PC-DEF-012` are untouched. No production signing
credentials were requested, no production candidate was built, no Play upload key
was created, no exports were narrowed, no updater/OAuth/deep-link behaviour
changed, no device or ADB was used, and no release, tag, publication or asset
deletion occurred.

## What the defect actually was

Not that AGP writes the R8 mapping and native debug symbols into a bundle's
`BUNDLE-METADATA/`. That is AGP working correctly: Google Play consumes those
entries to symbolicate crash reports and never delivers `BUNDLE-METADATA/` to an
installed client. Deleting them would remove Play's ability to read a stack trace
and fix nothing.

The defect was that PocketClaw had no way to say what an artifact was **for**,
and three consequences followed from that absence:

1. AABs were attached to public GitHub pre-releases while the mapping and
   symbols were simultaneously treated as private. Both cannot be true.
2. `RELEASE_PROCESS.md` required that material to stay "outside APK/AAB files",
   which AGP cannot satisfy for a bundle — a rule nothing could obey.
3. `artifact.r8_mapping_private` was a hardcoded `True` whose observation read
   "absent from APK". It verified nothing, and there was no AAB build or
   inspection path in the repository at all.

Contents cannot be judged without purpose. So purpose is now declared, and the
judgement follows from it.

## Artifact policy: before and after

    BEFORE
      policy      "private mapping/symbols stay outside APK/AAB files"
                  — unsatisfiable for a bundle, so unenforced
      gate        artifact.r8_mapping_private = True, always, observation wrong
      AAB         no build path, no inspection path, no classification
      publication AAB attached to public pre-releases (rc1, rc2)

    AFTER
      policy      absent from every APK; never a public release asset;
                  expected inside a Play-destined bundle's BUNDLE-METADATA/
      gate        artifact.r8_mapping_private reads the archive and cites
                  what it scanned; runs standalone, before the R8 contract
      AAB         repository-owned build path and inspection mode, both
                  requiring an explicit class
      publication public release asset allowlist forbids *.aab and every
                  mapping/symbol/keystore shape

## Classification model

Three classes, in `tool/artifact_policy.py`:

| Class | Meaning | Android binary |
| --- | --- | --- |
| `public-release` | GitHub Release, or a direct public download | **APK only** |
| `play-upload` | Uploaded to Google Play and nowhere else | AAB |
| `non-publish-audit` | Inspection evidence; published nowhere | APK or AAB |

`--artifact-class` is required for every artifact phase and has **no default**.
`--release-class` is about signing and `--artifact-class` is about purpose; they
answer different questions and both are mandatory.

Two properties are worth stating because they are what make the rule hold:

- **The refusal does not depend on contents.** An AAB with no metadata at all is
  still forbidden as a public asset, because what makes a bundle unpublishable
  is what the format is for. A test builds exactly that bundle and asserts the
  refusal.
- **Detection reads the archive, not the extension.** A bundle renamed to
  `.apk` is still detected as a bundle, so the policy cannot be laundered by
  `mv`. A test covers that too.

## Fail-closed behaviour

    $ python3 tool/release_gate.py --verify-bundle <aab>
    release_gate.py: error: --artifact-class is required with an artifact phase
    and has no default: an unclassified artifact fails closed.
    → exit 2

    $ python3 tool/build_hardened_android.py --signing local-test --package bundle
    build_hardened_android.py: error: --artifact-class is required with
    --package bundle and has no default …
    → exit 2

`distribution_verdict` refuses an empty string, `None`, and any unrecognised
value rather than falling through to the permissive reading. The build helper
does not even offer `public-release` for a bundle: it is not a choice a caller
should be able to express.

## Real `artifact.r8_mapping_private`

It now reads the artifact and reports its evidence, and it was moved out of
`r8_hardening_gates` into its own gate for two reasons: it is a property of the
artifact rather than of the mapping evidence, so it must run even without
`--r8-mapping`; and `inspect_r8_outputs` raises and returns early on a packaged
mapping, which meant the named check was unreachable in precisely the case it
existed for.

Demonstrated against two **real** artifacts, not only fixtures:

    clean fresh APK          PASS  496 archive entries scanned, deobfuscation entries = 0
    same APK + assets/mapping.txt
                             FAIL  assets/mapping.txt

## Hardened AAB build path

`tool/build_hardened_android.py --package bundle --artifact-class <class>` runs
`:app:bundleRelease` through the **same** hardening contract as the APK — arm64
target, `pocketclawDartHardening`, `dart-obfuscation`, private
`split-debug-info`, R8 minification and resource shrinking. The Gradle task is
parameterised rather than duplicated, and the Dart verification is shared:
`verify_private_dart_symbols` and `verify_packaged_dart_aot` are now called by
both paths, so neither can drift from the other.

The bundle inspection holds it to the same Dart contract and a different privacy
contract, because it is not the same artifact for the same audience.

## Evidence from this milestone's artifacts

Both built LOCAL TEST, from the current tree, carrying the PC-DEF-020 Core pair
`602ce034…` / `b5cce071…`:

    FRESH APK — LOCAL TEST / NON-RELEASABLE
      bytes   63,560,039
      sha256  7155de0aa8d77287a0a3ac7f64f0130f40d1963813d8f4074358f242752d503a

    AUDIT AAB — LOCAL TEST / NON-PUBLISH
      bytes   74,028,994
      sha256  00bde2c913b09a3acebaf1646cfb07613a0d597449b3c5918ffb64484b8d3455
      modules base;  native entries 18;  ABIs arm64-v8a, armeabi-v7a, x86_64
      BUNDLE-METADATA  11 entries, 39,584,662 bytes

Dart AOT `c7b2a885…` and private DWARF `0f52873b…` are byte-identical to H3A
onward in both. The R8 mapping moved to `1cf56ffe…` — expected, and explained:
`PocketClawService.kt` changed in PC-DEF-020, so R8's output for it differs.
That is a real input change, not a regression.

### The three classifications, against that real bundle

    --artifact-class public-release       FAIL, exit 1
      artifact.distribution_class: "AAB is a Play-upload/private-distribution
      artifact and is forbidden as a public release asset…"

    --artifact-class play-upload          PASS, exit 0
      bundle.play_metadata_expected: 1 mapping + 5 debug-symbol entries,
      39,584,662 bytes — allowed and expected for a Play upload
      bundle.metadata_recognised: 11 entries, all recognised
      bundle.no_unrelated_private_material: none

    --artifact-class non-publish-audit    PASS, exit 0
      NOT PLAY-READY / NOT PUBLIC-RELEASE-SAFE / NOT A GITHUB RELEASE ASSET

    (no --artifact-class)                 refused, exit 2

The Play-upload run is the point of the whole design: the same bytes that are
forbidden as a download are acceptable as a Play upload, and the gate names the
metadata rather than passing over it in silence.

## Unrelated private material still fails everywhere

The `BUNDLE-METADATA/` exemption covers exactly two known AGP entry shapes,
`obfuscation/proguard.map` and `debugsymbols/<abi>/<lib>.so.sym`.

That narrowing came from this milestone's own test suite. The first
implementation exempted the whole `obfuscation/` and `debugsymbols/`
directories, and the test that injects `…/obfuscation/release.jks` proved a
keystore dropped beside the mapping would pass. Exempting a directory to
accommodate two filenames was the wrong shape; it now matches the entries.

Keystores, key files, `.env`, the private support tree, `.debug`/`.dwarf`
companions, Dart `.symbols`, signing helpers, VCS metadata and credential stores
fail in every class, Play upload included.

## Tests

`tool/test_artifact_policy.py`, **32 tests**, covering all twelve required
cases; six drive the real gate CLI end to end.

| Case | Assertion |
| --- | --- |
| 1 | clean APK: no packaged mapping |
| 2 | APK with a mapping, at four different paths: detected |
| 3 | AAB classified public: FAIL — including an AAB with no metadata at all |
| 4 | AAB classified play-upload: PASS, metadata reported as allowed |
| 5 | AAB classified non-publish-audit: PASS with the notice |
| 6 | missing/unrecognised class: fails closed, four variants |
| 7 | expected `proguard.map`: reported allowed, not leakage |
| 8 | expected native debug metadata: reported allowed |
| 9 | unrelated private artifact in a Play bundle: FAIL, five shapes |
| 10 | asset allowlist rejects `*.aab` |
| 11 | allowlist rejects mapping/usage/`proguard.map`, `.debug`, `.sym`, `.symbols`, symbol and private-support archives, `.jks`, `.p12`, `.env` |
| 12 | `artifact.r8_mapping_private` is content-derived, not constant |

Plus: a bundle renamed to `.apk` is still a bundle; a non-Android zip is
refused; `.dwarf` outside the metadata directory is still private; ordinary
public assets (APK, `SHA256SUMS.txt`, notices, licences, source archives) remain
allowed; and a path is judged by its basename.

Existing suites: `tool/test_release_gate.py` **24 tests** (one added, one
retargeted — it asserted `artifact.r8_mapping_private` from inside
`r8_hardening_gates`, which is no longer where it lives),
`tool/test_build_hardened_android.py` **12 tests**, unchanged and passing.

One existing Flutter assertion needed updating for the same reason:
`android_release_contract_test.dart` matched the literal
`reset_generated_build_outputs(APK, symbols, r8_mapping=R8_MAPPING)`, and
parameterising the reset target for the bundle path changed that spelling. It now
asserts the call and its arguments plus the parameterisation, so the guarantee it
exists for — cached AOT and split symbols invalidated together, for whatever is
being built — is still pinned. That failure was caused by this milestone and was
fixed here rather than deferred.

## The rc1/rc2 bundles

`PocketClaw-v0.2.0-rc1.aab` and `PocketClaw-v0.2.0-rc2.aab` remain attached to
their published GitHub pre-releases. **Nothing was deleted and no release history
was rewritten**: this milestone had no authority to mutate published releases.

They predate Dart obfuscation and R8 minification, so what they disclose is not
the current hardened mapping. They are, however, the practice this policy
retires, and they are the reason `PC-DEF-021` was a live finding rather than a
theoretical one.

The exact owner action, if removal is wanted, is recorded in
`RELEASE_PROCESS.md`:

    gh release delete-asset v0.2.0-rc1 PocketClaw-v0.2.0-rc1.aab
    gh release delete-asset v0.2.0-rc2 PocketClaw-v0.2.0-rc2.aab

And what it buys is recorded with it: it ends ongoing public availability. It
cannot revoke a copy already downloaded, and each asset shows a recorded
download, so any mapping or symbol data in those bundles should be treated as
disclosed regardless.

## Gates

    tool/release_gate.py --verify-source (test and production)   25 PASS / 0 FAIL / 0 SKIPPED
    tool/release_gate.py --full <apk> --artifact-class public-release
                                                                 55 PASS / 0 FAIL / 1 SKIPPED
    tool/release_gate.py --verify-bundle <aab> --artifact-class play-upload         PASS
    tool/release_gate.py --verify-bundle <aab> --artifact-class non-publish-audit   PASS
    tool/release_gate.py --verify-bundle <aab> --artifact-class public-release      FAIL (required)
    tool/release_gate.py --verify-bundle <aab>                                      exit 2 (fail closed)
    tool/test_artifact_policy.py                                 32 tests
    tool/test_release_gate.py                                    24 tests
    tool/test_build_hardened_android.py                          12 tests

The full-APK skip is `repo.clean_worktree` while this work was uncommitted.

`flutter test` remains **one failure**: the stale literal in
`namespace_n3_native_identity_test.dart`, tracked as `PC-DEF-025` and
deliberately out of scope. It predates this milestone by five and is reported
rather than hidden — and it is not the failure this milestone introduced and
fixed, which was `android_release_contract_test.dart`.

## Closeout conditions

1. AAB is explicitly Play/private-only in release policy — `RELEASE_PROCESS.md`,
   "Artifact class".
2. Public release policy forbids AAB — the asset allowlist, machine-checkable
   via `--release-assets`.
3. `artifact.r8_mapping_private` is a real artifact-content check — proved on two
   real APKs and pinned by test.
4. AAB gate/inspection mode exists — `--verify-bundle` with a full inventory.
5. Classification fails closed — exit 2 in both tools, four refused values.
6. Public AAB classification fails — exit 1 on the real bundle.
7. Play AAB classification accepts expected metadata — and names it.
8. Unrelated private leakage still fails — five shapes, including inside the
   metadata directory.
9. Hardened AAB build path is repository-owned and repeatable — shares the APK's
   build contract rather than duplicating it.
10. rc1/rc2 recorded as historical owner-action items, not silently deleted.
