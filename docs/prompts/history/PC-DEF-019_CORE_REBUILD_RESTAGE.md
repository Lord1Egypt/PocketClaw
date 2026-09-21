# OPERATING RECORD — PC-DEF-019 Core rebuild and re-stage

RECONSTRUCTED OPERATING RECORD. This is an evidence-based closeout record, not a
claim to reproduce the owner prompt verbatim.

## Status and boundary

- **Status:** RESOLVED. Prerequisite repair only.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `ea43369289c8b6c618faa08f7b91355882fc050c` (UI-1 closeout).
- **Canonical Core build-input commit:** `ea43369289c8b6c618faa08f7b91355882fc050c`.
  `core/resolve-build-time.sh --print-commit` resolves to it, so the Core
  `BuildTime` is the UI-1 source state's timestamp and not this record's.
- **Ending commit:** the following staging/closeout commit that carries the
  rebuilt Core pair and this record, and touches no Core build input.
- **Version/baseline:** `0.2.0+62`; the accepted physical baseline remains
  vc62 / `lastAcceptedVersionCode=62` and is untouched.

This was not a feature milestone, the final release exposure audit, a signing
milestone, a version bump or a baseline advance. No device or ADB was used, no
production signing material was accessed, no APK or AAB was built, nothing was
merged, tagged, published or released, and the next release milestone is
unchanged.

## Why the rebuild was required

`libpocketclaw-web.so` embeds the compiled dashboard, so everything under
`core/src/web/frontend` is a Core build input. UI-1's guided-tour repair moved
the Core source fingerprint from
`86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73` to
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`, and the
staged pair still carried the H5B/H5C generation. UI-1 had no authority to
rebuild Core, so it recorded the consequence as `PC-DEF-019` instead of papering
over it.

## The two-commit rule, observed

`ea43369` is the source/build-input commit for this Core generation; no second
source-input commit was created before the rebuild. The rebuilt pair, the
regenerated private-support entries for it and this documentation land in one
following commit that changes no Core build input, so
`GOOS= GOARCH= go run ./cmd/corefingerprint .` still reports
`bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec`
after staging, and `core/resolve-build-time.sh --print-commit` still resolves to
`ea43369`.

## What changed in the shipped bytes

Only the Core pair. `curl`, `gh`, `git`, `git-remote-http`, `jq`, `python`,
`rg`, `sqlite3`, Dart AOT and the Flutter/JNI dependencies were not rebuilt and
their packaged bytes are unchanged; `git status` over
`android/app/src/main/jniLibs/` shows exactly two modified files. No export map
changed, and `PC-DEF-012` is untouched.

| Payload | Old bytes / SHA-256 | New bytes / SHA-256 |
| --- | --- | --- |
| `libpocketclaw.so` | 37,724,640 `62f741be6f71f7518ba0df8f4457e0f7b666dfe512cde88e79e213ca0907e901` | 37,724,640 `f273b9ced85f4d00cb542df9c2f4c691b4151526cb0ac9c2c7612a1432d7230f` |
| `libpocketclaw-web.so` | 25,517,952 `42d418bd3e2d1863d2dda4d46357e541a222b831cfffb17e5f36e442e455e9f5` | 25,517,952 `900c43fcaad2094017c6959eed623d1e2499cfd560f01f2cff36dd34202b86b9` |

Both files are the same size as the generation they replace; the sizes match by
coincidence of content, not by reuse, and the SHA-256 values are what identify
them.

    libpocketclaw.so       build ID 25e206ab402f8cd44a766bc03935468beebd8633
    libpocketclaw-web.so   build ID 84afbe2439b779722b22c2b4c6aa1300cd3ef199
    both                   ELF 64-bit LSB pie executable, ARM aarch64, DYN,
                           interpreter /system/bin/linker64, stripped
    BuildTime              2026-09-12T07:27:12+0000 (timestamp of ea43369)
    source fingerprint     bd4a8629a2682e2f05aa3859a400be8a77fb4954ad14994e5703ccbe365d05ec,
                           stamped into and read back from both binaries

`libpocketclaw.so` changed even though no Go source did: `BuildTime` and the
source-fingerprint stamp are `-X` values in both links, so a fingerprint move
moves both binaries. That is the point of stamping both — see the comment in
`core/build-android-arm64.sh` about the N4K-A gap.

## Reproducibility

The canonical `./core/build-android-arm64.sh` run, plus two further runs in
independent output roots with different `CORE_BUILD_DIR`, `JNI_LIBS` and
`NATIVE_SYMBOL_ROOT` paths — the third also with a cold `GOCACHE`, so the result
does not depend on a warm build cache. `cmp` reports both binaries and both
private companions byte-identical across all three roots.

`go test -tags "stdjson goolm" ./pkg/coresource/` passes whole: 46 tests, 0
failures, including `TestStagedCoreWasBuiltFromTheCurrentSource`, the
path-scoping tests and the web-bundle freshness tests.
`TestBundledPayloadsMatchTheirPinnedChecksums` and
`TestStagedCoreEmbedsTheCurrentCatalog` also pass, so the Managed Runtime
catalog and the embedded catalog still agree with the staged tree.

## Frontend embedding evidence

Timestamps prove nothing here, so the embedded dashboard is identified by
content, in both directions:

- `__pocketclaw_tour_probe__`, a string literal introduced by UI-1 in
  `core/src/web/frontend/src/store/tour.ts`, appears in the new
  `libpocketclaw-web.so` and appears **zero** times in the previous staged one.
- `Need more help? Click the documentation button in the top right corner`, the
  copy UI-1 deleted with the removed docs step, appears **zero** times in the
  new binary and once in the previous staged one. The minified locale key
  `"docs":{"title"` is likewise absent.
- The bundle asset names this build emitted — `index-BrDTVAj8.js`,
  `lib-CO0i6ntk.js` — appear in the new binary and in neither case in the
  binary staged at `ea43369`, which ties the embedded bundle to this build
  rather than to a bundle left over in `web/backend/dist`.

The structural half of the guarantee is already enforced by tests:
`TestTheLauncherBuildDependsOnTheFrontendBuild`,
`TestTheFrontendBundleIsAlwaysRebuilt` and
`TestTheBundleIsWrittenCleanIntoTheEmbeddedDirectory` pin the web Makefile so
the canonical build cannot embed a stale `dist/`.

## Native hardening contract

`tool/native_elf_audit.py` is APK-bound by design and no APK was built here, so
its own `inspect_elf()` and `evaluate_record()` were driven directly over the
two staged files under their packaged entry names. No check was redefined,
added or relaxed. **22 PASS / 0 FAIL** across the pair:

    identity                  64-bit AArch64 DYN, non-zero entry point
    load_alignment            minimum PT_LOAD alignment 0x10000 — 16 KiB compatible
    nx_stack                  GNU_STACK=RW
    no_wx_segments            no writable+executable PT_LOAD
    no_textrel                no DT_TEXTREL
    no_runtime_search_path    no RPATH/RUNPATH
    no_debug_sections         none
    no_static_symbol_table    no .symtab, no .strtab
    build_path_privacy        zero prohibited path strings
    relocation_hardening      GNU_RELRO present; zero imports, no lazy relocations
    required_exports          exactly {main.main}

The build script's own guards ran as part of the build: zero `/home/`, `/Users/`
or `/root/` strings in either payload, and the source fingerprint grep-verified
present in both.

## Private native support companions

Both Core companions were rebuilt from the same links by
`tool/native_support.py`, as the canonical build script does, and are bound to
the new shipped hashes, sizes and build IDs.

| Payload | Support bytes | Support SHA-256 | Symbolization |
| --- | ---: | --- | --- |
| `libpocketclaw.so` | 14,545,360 | `ce0d041d5421aa9a01e2cf21918a7415fa35c4515d4de77c960dc221ca95cecb` | `0xd3f1c0` → `main.main` at `./github.com/sipeed/picoclaw/cmd/picoclaw/main.go:166` |
| `libpocketclaw-web.so` | 9,765,608 | `55f077cd1b201fe67b64731c51126e2221988f838d80843867605ef775ef99b4` | `0x86d4d0` → `main.main` at `./github.com/sipeed/picoclaw/web/backend/main.go:416` |

The policy is unchanged: the archive is `build/private-symbols/native/android-arm64/`,
ignored and untracked (`git ls-files build/private-symbols` is empty), companions
0755 and the manifest 0600, nothing packaged — no `.debug` file exists under
`jniLibs/`. The eight Managed Runtime companions were not touched; their files
still carry their H5B mtimes and hashes.

**Pre-artifact manifest binding.** The manifest's `apk` field still names the
H5C production APK
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7`. That is the
established pre-artifact state rather than a defect: `native_support.py bind-apk`
is a separate explicit step run against a candidate APK, and
`native.private_support_apk_binding` is only evaluated in the APK audit phase,
which did not run. No APK was invented to satisfy it. The binding is
re-established at the next artifact build, against the artifact that actually
contains this Core pair.

## Gates

Both gates that `PC-DEF-019` was failing now pass:

    core.staged_freshness          FAIL → PASS
    build.reproducibility_tests    FAIL → PASS

`tool/release_gate.py --verify-source` reports **25 PASS / 0 FAIL / 0 SKIPPED**
in both the test and production release classes, against the staging commit with
the full toolchain on `PATH`. Before the rebuild the same run was 22 PASS /
2 FAIL / 1 SKIPPED. `core.staged_build_time` also moved SKIPPED → PASS: both
binaries are stamped `2026-09-12T07:27:12+0000`, the timestamp the gate expects
from `ea43369`. Full Core Go suite: 98 packages ok, 0 failed, 19 with no test
files.

## H5C historical evidence is untouched

The H5C production APK
`3774202ef9832c70ffa376e663db1da69e17ae9318df4cc8cb31156fc0c7eae7` remains
historically valid **for the dashboard and source tree it was built from**. It
is not relabelled as containing UI-1, none of its recorded hashes or evidence
were altered, and no history was rewritten. The Core pair recorded here belongs
to the post-UI-1 source state and will be included in the next artifact build.
