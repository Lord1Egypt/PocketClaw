# OPERATING RECORD — H5B targeted native hardening and private symbol archive

This is an evidence-based closeout record, not a claim to reproduce the owner
prompt verbatim.

## Status and boundary

- **Status:** IMPLEMENTED under LOCAL TEST / NON-RELEASABLE signing. Owner
  production validation is still required and has not happened.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `6c24f9ac67989a8bb2e08344ef9dcd113cdf7f18` (H5A closeout).
- **Source commit:** `aa24d9e906a28f71eb8231c8bf0f236cb1f96410`, which carries
  the recipes, tooling and the eight rebuilt Managed Runtime payloads. It is
  also the canonical Core build-input commit for the staged Core pair.
- **Ending commit:** the closeout commit containing this record and the staged
  Core pair.
- **Version/baseline:** `0.2.0+62`; the accepted physical baseline remains
  vc62 / 62 and is untouched.

H5B did not access production signing material, install on a device, use ADB,
merge, tag, publish, change the version, or advance the accepted baseline. The
only APK it produced is signed by the development certificate.

## What changed in the shipped bytes

Ten project-owned executables were rebuilt. `libapp.so`, `libflutter.so`,
`libdartjni.so` and `libdatastore_shared_counter.so` are dependency-owned and
were deliberately not touched; their packaged bytes are unchanged.

| Payload | Old bytes / SHA-256 | New bytes / SHA-256 |
| --- | --- | --- |
| `libpocketclaw.so` | 37,749,089 `d86276feb16b4556ebcebb5c0ff8e945d1e9a3501f8e9d5827c1e4680d03bda4` | 37,724,640 `62f741be6f71f7518ba0df8f4457e0f7b666dfe512cde88e79e213ca0907e901` |
| `libpocketclaw-web.so` | 25,559,393 `a44582bd3206567fbc07593895123be17f91dd3c3c90d7cf8d2effa7a1b8e892` | 25,517,952 `42d418bd3e2d1863d2dda4d46357e541a222b831cfffb17e5f36e442e455e9f5` |
| `libpocketclaw-curl.so` | 1,362,800 `0da29feaef9865014c48b3e4d54add375c79693bcb61607096338fbd69a20194` | 1,362,768 `9dd4b75a8cf4ad0e3e078e77884336ae6e84110f3cd48111ababe664bad15a96` |
| `libpocketclaw-gh.so` | 58,610,880 `fe97fb29f2b2036c71a7626b0fd3c9611cd3c746a7911a6e3b074cf7b20fdf96` | 58,610,880 `804ed93f89b0cdfc20ae282466a89be5e2b23bfe9848eb78b7e7c5ca159a44ac` |
| `libpocketclaw-git.so` | 3,386,776 `b3e905b81a46d5903ca22ebb169ee7698984384ff2f30a9ac0009490f8d7b66b` | 3,386,776 `a8a342ac2cff1c39bb9d8d3165921a4b6843b18908e8714b9be051bad68e7681` |
| `libpocketclaw-git-remote-http.so` | 3,121,848 `9790f8179d62c52f3d7a2f757103383a1120537fe7bdc40b9efed49d7a2b5200` | 3,121,736 `baea79ac073e5b3ceff5b14ada00a56931462652604b1c38e0db8b19ec30c260` |
| `libpocketclaw-jq.so` | 807,320 `3c1f61c100d7b8f3a68355f9cd697952bae27579cba516a0a3e43ac54926c997` | 807,032 `843b324de37727e59f9ec88b40c4f347832b91f4bd52719eebb5a7baeb4f097d` |
| `libpocketclaw-python.so` | 11,511,592 `4f98d0e31d0d0d35bf063c28da80b03a202af3c1acfefe3f05bda68f1e9d084d` | 11,511,919 `8b52e36dd4975763af46c9c26ab07b40e25d407e2df4415166e9d3d7d26a5b44` |
| `libpocketclaw-rg.so` | 4,482,472 `7dfcc880667642d9d3f4b5d7955bfae8602e5f76796144c556cba9f0aab1c73a` | 4,483,120 `0a5fc43f8df2fca3dbe70365e0496670df2e904239d9a4b5358fee40c4c70e4e` |
| `libpocketclaw-sqlite3.so` | 1,286,992 `6a5d6c3d9dc6da31235908084b076ce9c02ef51f2c35857f768e2d84a2ab286e` | 1,286,992 `b940785d77bb7c82d43843a721f9135c4b59e641f911ea4406eae67cc201cf43` |

Core source fingerprint
`86369a32a9873715672f7867b31dcd72a7d19088c49cdb1df2b584c548ba4c73`, stamped
into both Core binaries and independently recomputed by
`pkg/coresource`. Core `BuildTime` is `2026-09-12T05:01:21+0000`, the timestamp
of build-input commit `aa24d9e`.

## PC-DEF-009 — build-only RUNPATH: RESOLVED at the link step

git's Makefile turns `CURLDIR` into `-Wl,-rpath,$CURLDIR/lib`. The recipe now
passes `CURL_CFLAGS="-I$DEPS_PREFIX/include"` and an explicit `CURL_LDFLAGS`
library list instead, with `-L$DEPS_PREFIX/lib` in `LDFLAGS`. No finished ELF
was rewritten. `readelf -dW` finds no `DT_RPATH` or `DT_RUNPATH` in any of the
18 packaged entries, and the runtime payloads still resolve only the Android
platform libraries `libz.so`, `libdl.so`, `libc.so`.

`install_payload` now rejects *any* RPATH/RUNPATH rather than matching the one
root that happened to leak.

## PC-DEF-010 — build roots in shipped bytes: RESOLVED

`strings` over all ten payloads finds zero occurrences of
`/tmp/pocketclaw-runtime-build`, `/tmp/pocketclaw-h5b*`, `/home/lordegypt` or
the PocketClaw checkout path. The audit asserts this per entry as
`build_path_privacy`.

Two payloads needed more than a compiler flag, because a prefix map cannot
rewrite a string that the build wrote into generated *source*:

- **jq** records its literal `CFLAGS` in `src/config_opts.inc` and exposes it
  through `$JQ_BUILD_CONFIGURATION`. The recipe replaces that generated file
  with a stable release description after `configure` and before `make`.
- **CPython** compiles its configure-time `VPATH` into `getpath.c` as a C
  string literal. The recipe normalizes that one `Makefile.pre.in` definition
  to `/pocketclaw/cpython`, and only *after* the host build interpreter — which
  still needs the real tree — has been produced.

`ripgrep`'s `--remap-path-prefix` now covers the whole build root rather than
only its own source directory. curl, git, sqlite3 and the CPython dependencies
use shared `-ffile-prefix-map` / `-fdebug-prefix-map` / `-fmacro-prefix-map`
settings.

`install_payload` fails the build if `$BUILD_ROOT` survives in a payload, so a
future recipe cannot regress this silently.

## PC-DEF-011 — private native symbol archive: IMPLEMENTED

Every owned recipe now compiles with debug information, installs and strips the
shipped payload, and then calls `tool/native_support.py`, which derives a
`.debug` companion from the same link with `llvm-objcopy --only-keep-debug`,
refuses a companion without `.debug_*` sections, refuses a shipped file that
still has `.debug_*`/`.symtab`/`.strtab`, requires any build IDs present across
source/shipped/support to agree, resolves a representative symbol through
`llvm-addr2line`, and records the pair in a private manifest.

Python's companion comes from `$OBJ/python` before the standard library is
appended, because that append is exactly why the shipped file cannot be
stripped. Core keeps Go's DWARF by dropping `-s -w` and stripping the installed
copy instead, which is what makes a companion possible at all.

The archive lives at ignored `build/private-symbols/native/android-arm64/`.
Companion files are 0755 and the manifest is 0600; nothing is tracked, packaged
or published.

| Payload | Toolchain | Support bytes | Support SHA-256 | Symbolization | Location |
| --- | --- | ---: | --- | --- | --- |
| `libpocketclaw-curl.so` | ndk-clang | 5,926,504 | `29dbd54359f8f3da462d0a5ccae559ff31215d37fb6ffdd000c6b72d7681016a` | `0x7d8a0` → `main` | `/pocketclaw-runtime/build/curl-8.11.1/src/tool_main.c:229` |
| `libpocketclaw-gh.so` | go | 23,715,952 | `b80904bd20e946570b588dc71e02b24c3552dc78519e2f80aaead9e0be7b1f21` | `0x13dbce0` → `main.main` | `./github.com/cli/cli/v2/cmd/gh/main.go:9` |
| `libpocketclaw-git-remote-http.so` | ndk-clang | 12,926,288 | `876bc19d9148c8d0707225b719f06379ed7b0e7002794dd476ed5161422cb7aa` | `0xef418` → `main` | `/pocketclaw-runtime/build/git-2.51.0/common-main.c:5` |
| `libpocketclaw-git.so` | ndk-clang | 13,316,256 | `8276de71cb76acbb7a8373ff0df901ad897bebe92ad293d84ef8ff1af8ada6ab` | `0x1ae54c` → `main` | `/pocketclaw-runtime/build/git-2.51.0/common-main.c:5` |
| `libpocketclaw-jq.so` | ndk-clang | 1,518,880 | `d6bd5fa85aede8796524df5c493202a09d6815825631da7c9d25fd20eabd506d` | `0x6e5b0` → `main` | `/pocketclaw-runtime/build/jq-1.7.1/src/main.c:310` |
| `libpocketclaw-python.so` | cpython-ndk-clang | 33,952,680 | `83b9d390b2a697e78984ad573b78472146fadd5cf1814b084c1ba22abc58c397` | `0x476a74` → `main` | `.../cpython/Programs/python.c:15` |
| `libpocketclaw-rg.so` | rust | 55,648,928 | `cc9e8c8f5da5ee2911dbd6669a8ecf63bece8b8a88425cf8db4ac968d7658d70` | `0x20c39c` → `main` | no line entry — see below |
| `libpocketclaw-sqlite3.so` | ndk-clang | 4,273,504 | `a26689a954e897a41096ef9f32e589968324d73784f12c2a6b695da03bba8c28` | `0x58400` → `main` | `/pocketclaw-runtime/build/sqlite-amalgamation-3500400/shell.c:33210` |
| `libpocketclaw-web.so` | go | 9,765,608 | `469177f50505fa8248a90528632b1058f2f9cc2c1e2095f7d2a515a54f19c33f` | `0x86d4d0` → `main.main` | `./github.com/sipeed/picoclaw/web/backend/main.go:416` |
| `libpocketclaw.so` | go | 14,545,360 | `509ef59cf83c02bf0f1a75cd5a7f0b99ddfb91cca8f6098f8b413a8ccf3c9dfd` | `0xd3f1c0` → `main.main` | `./github.com/sipeed/picoclaw/cmd/picoclaw/main.go:166` |

ripgrep is the one qualified result. Its `main` is the C-ABI shim rustc emits
for a bin crate and carries no DWARF line entry, while every Rust function in
the same companion is a generic instantiation whose mangled name ends in an
unstable `17h<hash>` suffix and cannot serve as a fixed probe. The 55 MB
companion does contain the crates' DWARF; only the entry-point line record is
absent. This is recorded rather than papered over.

Six payloads — curl, git, git-remote-http, jq, rg and sqlite3 — carry no
`.note.gnu.build-id`, so their support entries bind by SHA-256 alone. That is
the stronger of the two bindings; the build ID is corroboration, not a
substitute.

## PC-DEF-012 — dependency export surfaces: DEFERRED, unchanged by decision

No export map was added. H5A established that `libdartjni.so`'s 313 exports are
bound by JNI name lookup and Dart FFI, that the CPython executable's 2,261
dynamic symbols back statically linked extension modules, and that datastore
and Flutter exports are upstream consumer contracts. H5B produced no new
call/reachability evidence, and narrowing a visibility surface without it is
how a runtime `UnsatisfiedLinkError` ships. The defect stays open with its
target restated: narrowing requires dependency-specific reachability plus
runtime validation, or an explicit acceptance of the retained surface.

## Reproducibility

Payload bytes depend on `SOURCE_DATE_EPOCH`: `libpocketclaw-python.so` embeds
its UTC date literally. `runtime/android-build-env.sh` therefore pins
`RUNTIME_EPOCH=1789157892` as a build input and validates any override, rather
than deriving it from repository state. An epoch derived from `HEAD`, or from
the last commit touching `runtime/`, would mean the very commit that records a
payload checksum in the Core catalog also changes the bytes that checksum
describes; `core/resolve-build-time.sh` documents the same failure for Core and
solves it by path scoping, which cannot help here because the recipes are their
own build input.

- All eight Managed Runtime payloads and all eight companions were produced
  byte-identically in two independent build roots, and then again in a third
  root under different `BUILD_ROOT`, `JNI_LIBS` and `NATIVE_SYMBOL_ROOT` paths.
- The Core pair and both Core companions were produced byte-identically in
  three separate output roots, including the canonical in-repository run.
- `TestBundledPayloadsMatchTheirPinnedChecksums`,
  `TestStagedCoreEmbedsTheCurrentCatalog` and
  `TestStagedCoreWasBuiltFromTheCurrentSource` all pass against the staged tree.

## Candidate artifact and verification

The only artifact is a LOCAL TEST / NON-RELEASABLE APK at
`build/app/outputs/apk/release/app-release.apk`, 63,560,467 bytes, SHA-256
`d4fe2c4a035051e3b6500d2a2fe9bdad639c97323c355b26a3f8ae6f215b9dd8`, built by
`tool/build_hardened_android.py --signing local-test --clean`. Independent
`apksigner` inspection reports exactly one v2 signer, the development
certificate `15cf75f9945d5354e75707e0326b7cffc60ac51a68df38156db318ef4578a27c`.
The enrolled production signer was not used and not accessed.

- `tool/native_elf_audit.py --enforce-target --native-support-manifest`:
  **194 PASS / 0 FAIL / 0 SKIP**. H5A's four target-policy failures are gone.
- Private support manifest checks: ten expected entries, every support
  hash/size and relative path valid, every entry bound to the shipped ELF's
  hash, size and build ID, debug data plus a resolved representative function
  present for each, manifest bound to this exact APK, and no private file
  tracked by Git.
- `tool/release_gate.py --full … --release-class test`: **PASS**, with
  `releasable: false` and `artifact.signing` classified LOCAL TEST /
  NON-RELEASABLE.
- Dart and R8 are regression-free: packaged Dart AOT
  `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77`, private
  split DWARF `0f52873bc712fe0c17d636f5bdb7d08ee80cdacfe633cf6a786dd2c6d93b8acc`
  and private R8 mapping
  `14d49fad46e773e32da69b7b2336b7a968808cd1130f0319f7806ca4d09c1beb` are all
  byte-identical to H3A/H3B/H4A/H4B.
- Focused suites: 16 tests in `tool/test_native_elf_audit.py` and 11 in
  `tool/test_native_support.py`, both wired into the source gate as
  `native.elf_audit_contract` and `native.private_support_contract`.

## Findings corrected during H5B verification

The implementation inherited from the previous session was reviewed rather than
trusted. Four things were changed before adoption:

1. `SOURCE_DATE_EPOCH` was derived from `git show -s --format=%ct HEAD`, which
   would have made the catalog unreproducible from the tree that carries it.
   Pinned as described above; the resolved value is unchanged, so no payload
   byte moved.
2. `tool/native_support.py` derived its output path straight from the
   `--logical-name` argument and would have written outside the private root
   for a name containing a separator or `..`. The name is now constrained to a
   plain file name and the resolved path is required to stay in the root.
3. `tool/native_support.py` and `tool/native_elf_audit.py` both assumed a
   well-formed existing manifest. A malformed one now fails closed with a clear
   message instead of raising, and the audit reports an unparsable support file
   as a finding rather than crashing.
4. Both recipes archived the private companion *before* the privacy, RUNPATH,
   ABI and alignment checks, so a rejected build would have left symbols for
   bytes that were never adopted. The capture now runs last.

One pre-existing build-script defect was also fixed because it blocked this
milestone: the AArch64 header check piped `llvm-readelf` into `grep -q` under
`pipefail`, and a `grep -q` that exits on its match can fail the pipeline by
SIGPIPE for reasons unrelated to the machine type. It misfired once on a python
payload whose bytes were provably correct. The header is now captured and then
matched.

## What still requires the owner

Nothing here was executed on hardware. Stripping Go's symbol table from Core
with `llvm-strip` rather than `-s -w`, and the CPython `VPATH` and jq
configuration normalizations, are source-level changes whose runtime behaviour
only a device can confirm. The golden physical workflow in
[`RELEASE_PROCESS.md`](../../RELEASE_PROCESS.md) is unchanged and still applies.

The next milestone is H5C. It must be separately authorized and must not be
started from this closeout.
