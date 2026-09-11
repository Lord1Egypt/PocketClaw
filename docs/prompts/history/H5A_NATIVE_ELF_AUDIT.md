# RECONSTRUCTED OPERATING RECORD — H5A native/ELF audit and symbol policy

This is an evidence-based closeout record, not a claim to reproduce the owner
prompt verbatim.

## Status and boundary

- **Status:** CLOSED for audit, policy, and read-only automation. No native
  binary implementation or rebuild was performed.
- **Working branch:** `feature/final-release-hardening`.
- **Starting commit:** `9aa7066cb30ca00800291b980de34e0ebccdce89`.
- **Ending commit:** the H5A audit/closeout commit containing this record.
- **Version/baseline:** `0.2.0+62`; accepted physical baseline remains vc62 / 62.
- **Artifact audited:** exact private H4B production-validation APK
  `build/app/outputs/apk/release/app-release.apk`, 63,560,431 bytes, SHA-256
  `14ba7d138a4092aefe264c7e2af6240c97fc1b782ded69918cbf545351eb5eb2`.
- **Next action:** H5B targeted native hardening and private native-symbol
  archive under a separate explicit prompt. H5B has not started.

H5A did not access signing material, build an APK, change any native binary,
rebuild Core or Managed Runtime, access a device, publish, merge, tag, change
version, or advance the accepted baseline.

## Complete packaged ELF inventory

All entries are `ET_DYN`. Category C/D entries are PIE executables with a
nonzero entry point and `/system/bin/linker64`; their `.so` suffix is Android's
existing executable-payload packaging mechanism. Category A/B/E entries have
entry point zero. Hashes are of the exact uncompressed APK entries.

| APK entry | Category | Bytes | SHA-256 | Entry | Align | Exports | Build ID |
| --- | --- | ---: | --- | ---: | ---: | ---: | --- |
| `arm64-v8a/libapp.so` | A Dart AOT | 5,702,536 | `c7b2a885ff843a20c57097a0d16ba07c728bd64cf17455a1ce61e6f463a5ae77` | `0` | `0x10000` | 3 | `811b4b6dc880579fa974eed1794edda5` |
| `arm64-v8a/libdartjni.so` | B JNI bridge | 131,248 | `47dae44db1c6202d164c0bb2ff25cc661023ba2904a6679abad4f3dcf3fcb5cd` | `0` | `0x4000` | 313 | `758897b0a64b066d87856c6e1be96c3eb8a410ed` |
| `arm64-v8a/libpocketclaw.so` | C Core | 37,749,089 | `d86276feb16b4556ebcebb5c0ff8e945d1e9a3501f8e9d5827c1e4680d03bda4` | `0x8da50` | `0x10000` | 1 | `f546bbb119d3188dd7de5491a52473925465f5a5` |
| `arm64-v8a/libpocketclaw-web.so` | C Core launcher | 25,559,393 | `a44582bd3206567fbc07593895123be17f91dd3c3c90d7cf8d2effa7a1b8e892` | `0x8d3e0` | `0x10000` | 1 | `fe7ca692e4a07bdac04ead455e0857dd999755d6` |
| `arm64-v8a/libpocketclaw-curl.so` | D runtime | 1,362,800 | `0da29feaef9865014c48b3e4d54add375c79693bcb61607096338fbd69a20194` | `0x75450` | `0x4000` | 0 | absent |
| `arm64-v8a/libpocketclaw-gh.so` | D runtime | 58,610,880 | `fe97fb29f2b2036c71a7626b0fd3c9611cd3c746a7911a6e3b074cf7b20fdf96` | `0x90e10` | `0x10000` | 1 | `f92880e9093e3eaa55fe3d9edefc2cca60e98ce0` |
| `arm64-v8a/libpocketclaw-git-remote-http.so` | D runtime | 3,121,848 | `9790f8179d62c52f3d7a2f757103383a1120537fe7bdc40b9efed49d7a2b5200` | `0xe5e00` | `0x4000` | 1 | absent |
| `arm64-v8a/libpocketclaw-git.so` | D runtime | 3,386,776 | `b3e905b81a46d5903ca22ebb169ee7698984384ff2f30a9ac0009490f8d7b66b` | `0xf84c8` | `0x4000` | 1 | absent |
| `arm64-v8a/libpocketclaw-jq.so` | D runtime | 807,320 | `3c1f61c100d7b8f3a68355f9cd697952bae27579cba516a0a3e43ac54926c997` | `0x6e5c0` | `0x4000` | 0 | absent |
| `arm64-v8a/libpocketclaw-python.so` | D runtime | 11,511,592 | `4f98d0e31d0d35bf063c28da80b03a202af3c1acfefe3f05bda68f1e9d084d` | `0x33ebe0` | `0x4000` | 2,261 | `bcb5c6e3a1bec8c75f15699d203a658d2fdf935c` |
| `arm64-v8a/libpocketclaw-rg.so` | D runtime | 4,482,472 | `7dfcc880667642d9d3f4b5d7955bfae8602e5f76796144c556cba9f0aab1c73a` | `0x1b0130` | `0x4000` | 0 | absent |
| `arm64-v8a/libpocketclaw-sqlite3.so` | D runtime | 1,286,992 | `6a5d6c3d9dc6da31235908084b076ce9c02ef51f2c35857f768e2d84a2ab286e` | `0x50834` | `0x4000` | 0 | absent |
| `arm64-v8a/libflutter.so` | E Flutter engine | 11,747,528 | `d12fd96d667d5bac111f9c7d5fed6cf37805d896b9f69fa30c09198d3f6eb594` | `0` | `0x10000` | 70 | `edfb9140c6b45651bf2f7c04752bc8e1dac58097` |
| `arm64-v8a/libdatastore_shared_counter.so` | E dependency | 7,112 | `d3e48717c9aa147e0ab21063ba0e8e0211cabf8bf40b222640829519edbf58e1` | `0` | `0x4000` | 9 | `17db37bd6770ac00dd2d1d2828839fd23a7959a3` |
| `armeabi-v7a/libdartjni.so` | B ABI stub | 81,444 | `a1f1f29941a7dbdcf53405af3a6a95db1f53b229343bb886f19ef8be6025e3d0` | `0` | `0x4000` | 313 | `5747f5291d39862e94ed94caa8d65286c2925f77` |
| `armeabi-v7a/libdatastore_shared_counter.so` | E ABI stub | 4,416 | `716c5d8d2cac8ca0edf65da8f139c7886b726ac79d542a14edeb94994ba6d3dc` | `0` | `0x4000` | 9 | `c2c541cc4337a0fc28271329ef238d6bddf6d70d` |
| `x86_64/libdartjni.so` | B ABI stub | 116,640 | `700b41f8bde7bef6f3c52e8d9a10136f2bc53860cca8d7d364759ff05a47114d` | `0` | `0x4000` | 313 | `736794ee0237c33a36a8ad786ab8a1f5f5e53970` |
| `x86_64/libdatastore_shared_counter.so` | E ABI stub | 6,224 | `fb6c9208988c49ae94943bc3236fa763ba584722e044fa4ea9a12d2941027105` | `0` | `0x4000` | 9 | `c6ef0802be059f5f59b23f1205d040dbc5b1ccad` |

## Security-property matrix

Every entry has a non-executable `RW` GNU stack, no writable+executable load
segment, no `DT_TEXTREL`, no `.debug_*`/`.zdebug_*`, and no `.symtab`/`.strtab`.
All load alignments are valid multiples of at least 16 KiB. Dynamic imports use
GNU RELRO plus BIND_NOW. `libapp.so` has no dynamic imports; the Go Core,
launcher, and `gh` static PIEs have only relative relocations, so BIND_NOW is
not applicable. Their GNU RELRO is present. All other imported ELFs have full
RELRO. The only RPATH/RUNPATH in the inventory is the Git HTTP finding below.

| Group | `DT_NEEDED` |
| --- | --- |
| Dart AOT; Go Core/launcher/`gh` | none |
| Dart JNI | `liblog.so`, `libm.so`, `libdl.so`, `libc.so` |
| Datastore | `libm.so`, `libdl.so`, `libc.so` |
| Flutter engine | `libc.so`, `libdl.so`, `libm.so`, `libandroid.so`, `libEGL.so`, `libGLESv2.so`, `liblog.so`, `libjnigraphics.so` |
| curl; Git; Git HTTP | Android platform `libz.so`, `libdl.so`, `libc.so` |
| jq; sqlite3 | Android platform `libm.so`, `libdl.so`, `libc.so` |
| Python | Android platform `libm.so`, `libdl.so`, `liblog.so`, `libz.so`, `libc.so` |
| ripgrep | Android platform `libdl.so`, `libc.so` |

SONAME appears only where a loadable shared-library contract needs it:
`libdartjni.so`, `libdatastore_shared_counter.so`, and `libflutter.so` across
packaged ABIs. Executable payloads have no SONAME.

## Debug, metadata, and path findings

The exact packaged Dart AOT confirms `PC-DEF-008` is non-impacting and resolved:
AGP's output is stripped and exposes only `_kDartSnapshotBuildId`,
`_kDartSnapshotData`, and `_kDartSnapshotText`. It has no source-level DWARF,
source paths, or sampled unobfuscated application names. Its 45-byte
`.eh_frame` is unwind metadata. Adding `gen_snapshot --strip` has no practical
distributed-binary benefit under the verified AGP boundary and risks changing
the external private-symbol/reproducibility contract.

Clang/LLD `.comment` records occur in plugin and C/Rust runtime outputs; `rg`
also names its Rust compiler. These are toolchain provenance rather than
source-level debug data or secrets. Go runtime function/file metadata remains
despite `-s -w`, but `-trimpath` prevents a host-root leak and permits useful Go
panic/stack diagnostics. H5B must not erase either class blindly.

No artifact contains `/home/lordegypt` or the PocketClaw checkout path. Three
payloads retain the neutral build root: curl has ten mbedTLS `__FILE__` paths,
Git HTTP has those ten plus `/tmp/pocketclaw-runtime-build/deps/lib` as
`DT_RUNPATH`, and Python has one CPython build-root string. Intentional
non-build data was classified separately: Python's `/tmp/perf-%jd.map`, Core's
`/tmp/project1`/`project2` documentation, and upstream `gh` strings such as
`/home/runner/work/` and `/tmp/extBrowse-*` do not identify this build machine.

## Export audit

- Core and Core launcher each export only `main.main`; they are static Go PIE
  executables, not JNI libraries or public shared APIs.
- Dart AOT exports exactly the three required Flutter snapshot symbols.
- Arm64 Dart JNI exports 313 names: 214 `globalEnv_*`, 42 Dart DL, seven
  Java/JNI, and 50 other symbols. Eleven exported names also occur as strings in
  current AOT. Dynamic Dart FFI lookup and JNI name binding prevent inferring a
  safe export allowlist from this one build; the automated audit asserts the
  known required boundary rather than hiding the remaining surface.
- Python exports 2,261 names. Its standalone executable and statically linked
  modules suggest narrowing may be possible, but only dependency-specific
  runtime evidence can establish that.
- Datastore exports four Java entry points and five internal C++ helpers.
  Flutter exposes 70 engine API symbols. Both are dependency-owned.
- curl, jq, ripgrep, and sqlite3 export none; Git and Git HTTP export only
  `error`; `gh` exports only `main.main`.

## Category-specific final policy

1. **Dart AOT:** packaged output has no source DWARF or static symbol table;
   retain required snapshot exports, build ID, and unwind metadata. Preserve
   the existing external Dart split-debug file privately.
2. **JNI/plugin libraries:** ship without source DWARF/static symbol tables;
   retain exact JNI, Dart FFI, Flutter, or Android dependency entry points.
   Narrow other exports only with call/reachability and runtime evidence.
3. **Core executables:** derive a private symbol-capable companion and a
   stripped, reproducible shipped executable from the same build input. Keep
   necessary Go runtime/unwind data and the executable-payload packaging model.
4. **Managed Runtime executables:** derive private support material before
   `llvm-strip`; ship stripped, NX/full-RELRO outputs without TEXTREL,
   RPATH/RUNPATH, or build roots. Preserve Python's appended ZIP and catalog
   hash semantics.
5. **Dependency-native payloads:** accept pinned upstream stripping/export
   contracts unless a concrete project requirement supports a source rebuild.
   Record shipped hashes/build IDs and preserve upstream symbols privately when
   available.

## Private native-symbol archive policy

Native support artifacts belong under ignored
`build/private-symbols/native/android-arm64/`, grouped by shipped artifact.
Each release's private manifest records shipped payload name/SHA-256/build ID,
support-file path/size/SHA-256, source/toolchain inputs, and the final APK hash.
The archive is private, untracked, outside APK/AAB files, absent from public
release assets and F-Droid payloads by default, and stored with the exact build
it symbolizes. A hash or build ID associates evidence; a build ID is useful but
not a substitute for byte hashes. Existing `build/` ignore coverage already
protects this root, so no `.gitignore` change was required. No native symbol
file was created in H5A.

## Automated audit and H5B implementation plan

`tool/native_elf_audit.py` performs read-only APK extraction and category-aware
ELF inspection with `readelf`, `file`, and `strings`; independent `nm -D` and
`objdump -p` checks confirmed the key export and RUNPATH results. Its H5A audit
mode reports 184 target-policy PASS, four FAIL,
and zero SKIP: the RUNPATH and three build-path checks fail as recorded in
`PC-DEF-009`/`010`. These expected findings do not fail audit mode; H5B must run
`--enforce-target`. Eleven focused parser/policy tests pass, and the production
source gate runs them as `native.elf_audit_contract`.

H5B must remain a separate milestone and perform these ordered targets:

1. Add deterministic C/C++ prefix maps at the mbedTLS/curl dependency build,
   investigate why Python's existing map leaves one raw root, and prove no
   packaged current-build root remains.
2. Capture the exact Git HTTP link command, remove the RUNPATH at its source
   configuration/link step, and prove dependencies still resolve only to the
   Android platform entries. Do not patch the finished binary.
3. For owned Core and runtime recipes, generate a private unstripped twin or
   separate debug companion before the final strip, then derive the catalogued
   shipped bytes deterministically. Handle Go, Rust/C, and Python separately;
   create Python support data before strip and before its ZIP append.
4. Add an ignored per-build manifest under the native private-symbol root,
   verify at least representative native address symbolization, and bind each
   support hash to each shipped payload and final validation APK.
5. Preserve the current required Dart/JNI/Android entry points. Narrow Python,
   datastore, or Dart-JNI visibility only when dependency-specific call and
   runtime evidence proves the allowlist; otherwise document the retained
   dependency contract.
6. Prove two equivalent native builds produce identical shipped payloads and
   meaningful support artifacts before adopting bytes. Update runtime/Core
   catalogs atomically, run the enforced native audit plus all existing gates,
   and use only a LOCAL TEST / NON-RELEASABLE APK unless H5B explicitly says
   otherwise.

## Golden physical workflow preserved for later use

The durable future-device workflow is recorded in
[`RELEASE_PROCESS.md`](../../RELEASE_PROCESS.md). H5A only documented it and did
not invoke ADB or access the device. Because vc62 and the production certificate
have different signing identities, a future production transition needs a
separately authorized clean-install/migration and data-safeguard plan; an
`adb install -r` across those identities is forbidden.
