# Python Lite — Phase A feasibility record

Experimental. No payload ships, no catalog entry exists, and nothing here is
wired into the Gradle build guard or the Agent. This file records what was
built, what was measured, and what is still unproven, so Phase B starts from
evidence rather than from the architecture review's estimates.

Reproduce with `runtime/build-python-android-arm64.sh`.

## Target

| | |
|---|---|
| CPython | 3.14.7 (released 2026-08-05, bugfix branch, EOL Oct 2030) |
| Source SHA-256 | `62859805f6fdf25e2bcbf3fa3217801e1996887ca33e6a2af80674bdfa2dbe07` (matches python.org) |
| Triplet / API / ABI | `aarch64-linux-android` / 24 / arm64-v8a |
| NDK | 28.2.13676358 — PocketClaw's own pin, **not** upstream's 27.3.13750724 |
| Build route | upstream `Android/android.py`, one patched line (NDK version) |

CPython 3.14's `Android/android-env.sh` already defaults to API level 24, which
is also Flutter's `minSdkVersion` and the level every other payload is built
at. No compromise was needed.

## Profile

Extension modules are linked **statically** (`MODULE_BUILDTYPE=static`), so
`lib-dynload` is empty and the interpreter is one self-contained ELF. This
removes the extension-module naming problem entirely: there are no
`_ssl.cpython-314-aarch64-linux-android.so` files to package.

Excluded: `_socket` `_ssl` `_hashlib` `_ctypes` `_zstd` `_asyncio`, the six CJK
codecs plus `_multibytecodec`, `_interpreters` `_interpchannels`
`_interpqueues` `_remote_debugging` `_lsprof` `_zoneinfo` `syslog` `termios`,
and all test modules.

Unavailable on Android regardless: `_curses` `_curses_panel` `_dbm` `_gdbm`
**`_multiprocessing`** `_posixshmem` `_tkinter` `_uuid` `grp` `readline`.

`hashlib` still provides sha256, sha3_256, blake2b and hmac without OpenSSL,
via CPython's built-in HACL\* implementations. Verified.

## Measurements

| Artifact | Bytes |
|---|---|
| Interpreter, unstripped | 37,213,856 |
| Interpreter, stripped, LTO | 9,123,056 |
| stdlib `.py` zip (298 modules) | 1,524,950 |
| stdlib `.pyc` zip (298 modules) | 2,468,331 |
| **Payload: interpreter + `.pyc` stdlib** | **11,591,387** |
| Payload: interpreter + `.py` stdlib | 10,648,006 |

APK impact, measured by adding the payload to a copy of the shipped
`app-release.apk` (58,332,647 bytes):

| Variant | In-APK | Projected APK | Delta |
|---|---|---|---|
| `.pyc` stdlib | 5,814,792 | 64,147,589 | +5,814,942 |
| `.py` stdlib | 4,878,684 | 63,211,481 | +4,878,834 |

Gate: APK increase ≤ 7.5 MB and installed ≤ 13.0 MB. Both pass.

LTO is worth keeping: it saved 475,632 bytes (−5.0%) over the same build
without it. `configure --with-lto` needs `llvm-ar` on `PATH`, which
`android-env.sh` does not put there.

### Why `.pyc` despite being larger

The `.pyc` stdlib costs 943,381 more bytes installed but starts far faster,
because `PYTHONDONTWRITEBYTECODE` means a `.py` zip is recompiled on every
launch and nothing is ever cached. Measured on the host (x86_64, five runs,
importing json/re/datetime/hashlib):

| stdlib | min | median | max |
|---|---|---|---|
| `.py` | 0.55 s | 0.74 s | 1.05 s |
| `.pyc` | 0.15 s | 0.16 s | 0.22 s |

Roughly 4.5x. For a tool whose whole point is short utility tasks, that is
worth 0.9 MB. **This is a host figure, not a device figure** — the device
harness measures both and the choice should be re-confirmed there.

## Packaging: zip appended to the ELF

The payload is `[stripped ELF][stdlib zip]` in one file, with `PYTHONPATH`
pointing at the payload itself. Stripping happens **before** appending;
stripping afterwards would discard the archive.

Verified on the host: ELF magic, class, machine and `DYN` type intact; all
three `LOAD` segments still 16 KB aligned; the trailing zip parses; and CPython
3.14 imports 33 stdlib modules straight out of it under
`-P -s -S -B -u` with `PYTHONHOME` set to a path that does not exist. Nothing
depends on directory layout around the file.

Not yet verified on a device: that the Android package manager extracts the
file unchanged and that the loader accepts it. The fallback, if it does not, is
a stdlib zip as non-executable app-private data, which costs the zip twice.

## ELF

ELF64, AArch64, `DYN` (PIE), interpreter `/system/bin/linker64`, all three
`LOAD` segments aligned `0x4000` (16 KB). `NEEDED`: `libm.so`, `libdl.so`,
`liblog.so`, `libz.so`, `libc.so` — Android platform libraries only. No
`RUNPATH`/`RPATH`. No glibc, libpthread, libutil, ncurses, readline or OpenSSL.

## Build-path privacy

CPython bakes its project base into the interpreter, and `assert()` bakes
`__FILE__` into every translation unit. Both leak the build machine.

- Building in the session scratchpad: **42** leaked strings.
- Adding `-ffile-prefix-map`: **1** — CPython's compiled-in project base.
- Adding a neutral build root (`/tmp/pocketclaw-build`): **0**.

The final payload has zero `$HOME` hits, zero developer-root hits, zero
occurrences of the builder's username and zero of the session id, under
`install_payload`'s own two rules. The only absolute paths left are
`/tmp/pocketclaw-build/cpython`, `/tmp/myimport.zip` (a zipimport docstring)
and `/tmp/perf-` (a runtime template) — none of which identify a machine.

`_sysconfigdata__android_aarch64-linux-android.py` embeds build paths including
`$HOME`, so it is excluded from the stdlib zip. Nothing in the Lite profile
imports it. If `sysconfig` is ever needed, Phase B must sanitise that file
rather than ship it.

## Provenance

| Component | Version | Source | SHA-256 verified | Role |
|---|---|---|---|---|
| CPython | 3.14.7 | python.org | yes, matches published | interpreter + stdlib |
| SQLite | 3.50.4 | sqlite.org amalgamation | yes, PocketClaw's existing pin | `_sqlite3`, built here from source |
| libmpdec | bundled in CPython | python.org | via CPython | `_decimal` |
| HACL\* | bundled in CPython | python.org | via CPython | `hashlib`, `hmac` |
| expat | bundled in CPython | python.org | via CPython | `pyexpat`, `xml.etree` |
| bzip2 | 1.0.8-3 | beeware prebuilt | pinned here, **PROVENANCE GAP** | `_bz2` |
| xz | 5.4.6-1 | beeware prebuilt | pinned here, **PROVENANCE GAP** | `_lzma` |
| zlib | platform | Android NDK | n/a | `zlib` |

> **Superseded by Phase B (2026-08-31).** bzip2 1.0.8 and XZ 5.4.7 are now built
> from pinned source by `runtime/build-python-android-arm64.sh`, and no upstream
> prebuilt binary is used. Dependency provenance is PASS. The gap described
> below is retained because it documents what upstream's tooling does, which is
> still true and still the reason PocketClaw does not use it.

**PROVENANCE: GAP.** Upstream `Android/android.py` downloads six prebuilt
dependency tarballs from `github.com/beeware/cpython-android-source-deps` with
`curl -Lf` and **no checksum verification of any kind**. The Phase A build
script pins each by SHA-256, which is strictly better than upstream, but bzip2
and xz remain third-party *binaries* rather than pinned source built locally.

Recorded hashes of everything upstream fetched during Phase A:

```
f446f18d381ed641cb9d58b4768097f9cb7fcce79f7d13be371ee089f2c93e27  sqlite-3.50.4-0-aarch64-linux-android.tar.gz
3d62143ba57f17dfa25816b1ce06256944cb23e5bad1212a419cfd073b1eebab  openssl-3.5.7-0-aarch64-linux-android.tar.gz
c3ae98fbe54b0cef9601d9dc120ed692d79609087e6926c70d8fc30face07fe7  zstd-1.5.7-2-aarch64-linux-android.tar.gz
9f2c0255ce025c177d44db16174ad5158c7560efe3c7ef0c8c0c64b2196e6a9d  libffi-3.4.4-3-aarch64-linux-android.tar.gz
2385f46e173d525f079946957c007000a8ad11d8496ba66bae99129183d74bd9  bzip2-1.0.8-3-aarch64-linux-android.tar.gz
320b76d45dc3499cf855e5310f875cba61c2608e4a98bb280cc4f1b8f189da1a  xz-5.4.6-1-aarch64-linux-android.tar.gz
```

OpenSSL, libffi and zstd are downloaded by upstream but the Lite profile links
none of them. Phase B should build bzip2 and xz from pinned source with
`android-build-env.sh`, as SQLite already is, and then this table has no gap.

## Licensing (for THIRD_PARTY_NOTICES.md in Phase B)

PSF License 2.0 (CPython), public domain (SQLite — a second, embedded copy
beside the CLI's), BSD 2-clause (libmpdec), Apache 2.0 (HACL\*), MIT (expat),
0BSD (xz), BSD-like (bzip2). OpenSSL and libffi do not apply: not linked.

## Physical validation — PASS (2026-08-31)

Samsung SM-A165F, Android 16, API 36, arm64-v8a, 4096-byte page size, via
`runtime/python-lite-device-tests.sh`. **52 checks passed, 0 failed.**

Acceptance gate: payload present in `nativeLibraryDir`; direct execution as the
app uid; `Python 3.14.7`; `PYTHON-PASS` from a stdin script. All four pass, so
**standalone CPython runs on Android under the Managed Runtime execution model**.

getpath resolved exactly as designed, with `PYTHONHOME` pointing at
`/pocketclaw/python`, which does not exist on the device:

```
sys.prefix   = /pocketclaw/python
sys.path[0]  = <nativeLibraryDir>/libpocketclaw-python.so
flags        : no_site=1 no_user_site=1 dont_write_bytecode=1 safe_path=1
```

The appended-zip stdlib is confirmed on hardware: every import came out of the
zip appended to the ELF, and the package manager extracted the file unchanged.
The fallback layout is not needed.

34 modules import and 8 compute assertions pass. `bz2`, `lzma` and `sqlite3`
work. `socket`, `ssl`, `ctypes`, `multiprocessing`, `email` and `http` are
absent. `hashlib`, `hmac` and `secrets` work with no OpenSSL:
`sha256`, `sha3_256`, `blake2b` and HMAC all produce correct digests.

SQLite is 3.50.4 with FTS5 and JSON1; commit and rollback both behave.
Arabic and emoji round-trip through stdout and a file (23 chars, 39 UTF-8 bytes).

| Startup (10 runs) | min | median | max |
|---|---|---|---|
| bare | 76 ms | **90 ms** | 96 ms |
| `import json, re, datetime, hashlib` | 108 ms | **121 ms** | 134 ms |
| sqlite3 in-memory round trip (5 runs) | 100 ms | **112 ms** | 116 ms |

Comfortably inside the 1-second product threshold, and better than the review's
250-600 ms estimate. This is the `.pyc` payload; the `.py` variant was not
re-measured on device, and on this evidence there is no reason to revisit it.

| RSS | |
|---|---|
| bare | 11,200 kB |
| typical imports | 12,736 kB |
| sqlite3, 1k inserts | 13,928 kB |
| json, 20k objects | 20,756 kB |

RSS only. PSS and USS need `dumpsys meminfo` against a live process and were not
collected, so they are not reported.

A runaway `while True: pass` was killed in **25 ms** with **no orphan**, using
the process-group semantics the Runtime already implements.

## Correction: Android does have /bin/sh

The architecture review asserted that Android provides no `/bin/sh` and that
`subprocess(shell=True)` and `os.system()` would therefore be unusable. **On
API 30 and later that is wrong**, and it was measured wrong here:

```
command -v sh  : /system/bin/sh
/bin           : lrw-r--r-- root root 11 -> /system/bin
/bin/sh        : -rwxr-xr-x root shell 334864
/system/bin/sh : mksh, MIRBSD KSH R59 2020/10/31 Android
```

From Python on the device: `shell=False` rc=0, **`shell=True` rc=0**,
**`os.system()` rc=0**. Android 11 added the `/bin` -> `/system/bin` symlink, so
the shell is reachable at the path Python looks for.

PocketClaw's `minSdkVersion` is 24, so this is version-dependent: on API 24-29
`/bin/sh` genuinely is absent and those calls fail. Python Lite must therefore
document shell availability as *conditional*, not absent.

This does not change the security conclusion, it sharpens it. The review already
said Python's `subprocess` is a Runtime bypass and that guidance about it is
advisory rather than a boundary. Shell access being available on modern devices
makes that bypass easier to reach, not different in kind. Nothing was added to
the device and no shell ships with PocketClaw.

The same conditionality applies to the documented git limitation, which was
written from the same assumption. That is recorded but deliberately not acted on
here.

## What Phase A could not answer

Nothing, now. The initial build host had no device, but the physical run above
was completed against a real SM-A165F by pointing the Linux adb client at the
Windows ADB server (`ADB_SERVER_SOCKET=tcp:172.20.64.1:5037`).

The one item still outstanding is not a measurement but a work item: bzip2 and
xz remain third-party prebuilt binaries (see Provenance).

`build/phase-a-python/` holds the payload, both stdlib zips, a signed
throwaway diagnostic APK (`com.pocketclaw.pythonprobe`, targetSdk 36,
debuggable, no code) and `run-device-tests.sh`, which installs it, runs the
interpreter from `nativeLibraryDir`, and answers: standalone execution,
`--version`, stdin execution, getpath values, the module matrix, sqlite3,
hashing, Unicode, shell-free `subprocess`, runaway termination and orphans,
startup latency and RSS. It uninstalls the probe afterwards.
