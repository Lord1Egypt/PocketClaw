#!/usr/bin/env bash
#
# Shared setup for PocketClaw Managed Runtime payload builds.
# Source this; do not execute it.
#
# Every payload is packaged as android/app/src/main/jniLibs/arm64-v8a/lib*.so.
# The lib*.so name is an Android packaging requirement, not a claim about the
# file format: Android's package manager only unpacks lib/<abi>/*.so entries
# into nativeLibraryDir, and nativeLibraryDir is the only directory an app
# targeting API 29+ may execute from. A payload under any other name would ship
# inside the APK and never be runnable.

set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JNI_LIBS="${JNI_LIBS:-$REPO_ROOT/android/app/src/main/jniLibs/arm64-v8a}"
MANIFEST="$REPO_ROOT/core/src/pkg/pcruntime/manifest.json"
NATIVE_SYMBOL_ROOT="${NATIVE_SYMBOL_ROOT:-$REPO_ROOT/build/private-symbols/native/android-arm64}"

# API 24 is the oldest sysroot these payloads need. It is well below the app's
# minSdk, so nothing is gained by raising it and older devices keep working.
ANDROID_API="${ANDROID_API:-24}"

# shellcheck source=runtime/toolchains.env
source "$REPO_ROOT/runtime/toolchains.env"

NDK_ROOT="${NDK_ROOT:-/home/lordegypt/PocketCLaw/.tooling/android-sdk/ndk/$POCKETCLAW_NDK_VERSION}"
NDK_ROOT="${NDK_ROOT%/}"
TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64"

# require_ndk_version: the NDK decides the bytes of every C payload.
require_ndk_version() {
    local revision
    revision="$(sed -n 's/^Pkg.Revision *= *//p' "$NDK_ROOT/source.properties" 2>/dev/null || true)"
    if [ "$revision" != "$POCKETCLAW_NDK_VERSION" ]; then
        echo "error: NDK at $NDK_ROOT is '${revision:-unknown}', pinned $POCKETCLAW_NDK_VERSION" >&2
        echo "       (runtime/toolchains.env). Point NDK_ROOT at the pinned NDK." >&2
        exit 1
    fi
}

# A fixed build directory keeps the developer's home path out of the binaries:
# autotools records its own configure arguments in the compiled result.
BUILD_ROOT="${BUILD_ROOT:-/tmp/pocketclaw-runtime-build}"
CACHE_DIR="${CACHE_DIR:-$BUILD_ROOT/downloads}"
# Shared install prefix for cross-built dependencies (mbedTLS, libcurl).
DEPS_PREFIX="${DEPS_PREFIX:-$BUILD_ROOT/deps}"

TARGET_CC="aarch64-linux-android${ANDROID_API}-clang"

# CPython and some upstream C sources use __DATE__/__TIME__, so the shipped
# payload bytes depend on this value: libpocketclaw-python.so embeds the UTC
# date of RUNTIME_EPOCH literally.
#
# It is therefore a pinned build input, like the tarball checksums and
# RUNTIME_* versions below it, and deliberately not derived from repository
# state. Deriving it from HEAD -- or from the last commit touching runtime/ --
# would mean that the very commit which records a payload checksum in the Core
# catalog also changes the bytes that checksum describes, so the catalog could
# never be reproduced from the tree that carries it. core/resolve-build-time.sh
# documents the same failure for Core and solves it by path scoping; scoping
# cannot help here because the recipes are their own build input.
#
# Move this value only together with a rebuild and a catalog update.
#
# An inherited SOURCE_DATE_EPOCH is deliberately replaced, not honoured: build
# services export their own (fdroidserver exports the commit time), and letting
# it win produced a payload that no longer matched the catalog.
RUNTIME_EPOCH=1789157892   # 2026-09-11T20:18:12Z, the H5B native-hardening build input
SOURCE_DATE_EPOCH="$RUNTIME_EPOCH"
export SOURCE_DATE_EPOCH

[ -d "$TOOLCHAIN/bin" ] || {
    echo "error: NDK toolchain not found at $TOOLCHAIN; set NDK_ROOT" >&2
    exit 1
}
require_ndk_version
mkdir -p "$CACHE_DIR" "$DEPS_PREFIX" "$JNI_LIBS"

# C/C++ debug companions need source lines, while shipped payloads remain
# stripped. The maps make distinct build roots and distinct NDK installations
# produce the same debug and shipped bytes without rewriting any finished ELF.
# The NDK map matters wherever a GNU build-id is linked: the build-id hashes the
# unstripped ELF, so an NDK path left in its DWARF changes shipped bytes
# (PC-DEF-091).
NATIVE_DEBUG_CFLAGS="-g -ffile-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build -fdebug-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build -fmacro-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build -ffile-prefix-map=$NDK_ROOT=/pocketclaw-ndk -fdebug-prefix-map=$NDK_ROOT=/pocketclaw-ndk -fmacro-prefix-map=$NDK_ROOT=/pocketclaw-ndk"

# fetch_pinned <url> <filename> <sha256>
#
# Downloads once and verifies every time. A checksum mismatch aborts the build
# rather than producing a payload whose provenance cannot be stated.
fetch_pinned() {
    local url="$1" name="$2" sha="$3"
    if [ ! -f "$CACHE_DIR/$name" ]; then
        echo "Fetching $url"
        curl -fsSL -o "$CACHE_DIR/$name" "$url"
    fi
    echo "$sha  $CACHE_DIR/$name" | sha256sum -c - >/dev/null || {
        echo "error: $name does not match its pinned checksum; refusing to build" >&2
        exit 1
    }
}

# install_payload <built-file> <lib*.so name>
#
# Strips, verifies, and installs one payload, then prints its checksum for the
# runtime catalog.
# install_payload <built-file> <lib*.so name> [strip-mode] [symbol-source]
#
# Pass "no-strip" for a payload that carries data after the ELF image, such as
# the Python interpreter with its standard library appended: llvm-strip rewrites
# the file and would discard everything past the last section.
install_payload() {
    local built="$1" payload="$2" strip_mode="${3:-strip}" symbol_source="${4:-$1}"

    install -m 0755 "$built" "$JNI_LIBS/$payload"
    if [ "$strip_mode" != "no-strip" ]; then
        "$TOOLCHAIN/bin/llvm-strip" --strip-unneeded "$JNI_LIBS/$payload"
    fi

    # Build-path privacy, the same rule the Core build enforces: a shipped
    # binary must not identify the machine it was built on.
    #
    # The test is deliberately narrower than a bare /home|/Users|/root search.
    # Go records trimmed module paths such as "pkg/root/trusted_root.go", and
    # upstream sources contain their own CI constants like "/home/runner/work/".
    # Neither identifies this machine, and failing on them would mean either
    # shipping with the check disabled or patching upstream for nothing. What
    # must never ship is this build's own home directory, so that is what is
    # checked, plus generic developer roots anchored at a path boundary.
    local leaked
    leaked="$(strings -a "$JNI_LIBS/$payload" | grep -c -F "$HOME" || true)"
    if [ "$leaked" -ne 0 ]; then
        echo "error: $payload contains this build machine's home directory" >&2
        strings -a "$JNI_LIBS/$payload" | grep -F "$HOME" | head -5 >&2
        exit 1
    fi
    leaked="$(strings -a "$JNI_LIBS/$payload" | grep -c -F "$BUILD_ROOT" || true)"
    if [ "$leaked" -ne 0 ]; then
        echo "error: $payload contains its temporary native build root" >&2
        strings -a "$JNI_LIBS/$payload" | grep -F "$BUILD_ROOT" | head -5 >&2
        exit 1
    fi

    # Executable payloads resolve only Android platform dependencies. A build
    # directory in DT_RPATH/DT_RUNPATH is both non-reproducible and unusable on
    # device, so reject every search path rather than matching one known root.
    local search_path
    search_path="$("$TOOLCHAIN/bin/llvm-readelf" -dW "$JNI_LIBS/$payload" \
        | sed -n '/(RPATH)\|(RUNPATH)/p')"
    if [ -n "$search_path" ]; then
        echo "error: $payload contains an RPATH/RUNPATH" >&2
        printf '%s\n' "$search_path" >&2
        exit 1
    fi
    leaked="$(strings -a "$JNI_LIBS/$payload"         | grep -c -E '(^|[^[:alnum:]_./-])(/home/[a-z]|/Users/[A-Za-z]|/root/[a-z.])' || true)"
    if [ "$leaked" -ne 0 ]; then
        echo "error: $payload contains developer-machine paths" >&2
        strings -a "$JNI_LIBS/$payload"             | grep -E '(^|[^[:alnum:]_./-])(/home/[a-z]|/Users/[A-Za-z]|/root/[a-z.])' | head -5 >&2
        exit 1
    fi

    # An ARM64 payload that is not ARM64 fails on the device with a bare ENOEXEC.
    #
    # The header is captured and then matched, rather than piped into `grep -q`.
    # Under `pipefail` a `grep -q` that exits on its match can leave the reader
    # writing into a closed pipe, and the resulting SIGPIPE fails the pipeline
    # for a reason that has nothing to do with the machine type. That misfired
    # once on the python payload, whose bytes were provably correct.
    local elf_header
    elf_header="$("$TOOLCHAIN/bin/llvm-readelf" -h "$JNI_LIBS/$payload")"
    case "$elf_header" in
        *AArch64*) ;;
        *) echo "error: $payload is not an AArch64 binary" >&2; exit 1 ;;
    esac

    # 16 KB page alignment. Android 15 introduced devices with 16 KB pages, and
    # a payload linked for 4 KB pages will not load there at all.
    #
    # The requirement is "at least 16 KB", not "exactly": a segment aligned to a
    # larger multiple satisfies a 16 KB page too. The NDK links C payloads at
    # 0x4000, while Go links arm64 at 0x10000 -- which is why Core, the launcher
    # and gh are all 0x10000 and have run on device since Phase 1. An equality
    # test rejected them, so it tested the linker's choice rather than the
    # property Android cares about.
    local alignment
    for alignment in $("$TOOLCHAIN/bin/llvm-readelf" -lW "$JNI_LIBS/$payload" \
        | awk '$1 == "LOAD" { print $NF }' | sort -u); do
        if [ $(( alignment )) -lt 16384 ] || [ $(( alignment % 16384 )) -ne 0 ]; then
            echo "error: $payload has a LOAD segment aligned $alignment;" >&2
            echo "       it must be a multiple of 0x4000 to load on a 16 KB-page device" >&2
            exit 1
        fi
    done

    # Archive the private companion only once the payload has passed every
    # release check above: a rejected build must not leave a support file and a
    # manifest entry describing bytes that were never adopted.
    local probe="main" toolchain_id="ndk-clang"
    case "$payload" in
        libpocketclaw-gh.so) probe="main.main"; toolchain_id="go" ;;
        libpocketclaw-rg.so) toolchain_id="rust" ;;
        libpocketclaw-python.so) toolchain_id="cpython-ndk-clang" ;;
    esac
    python3 "$REPO_ROOT/tool/native_support.py" \
        --source "$symbol_source" --shipped "$JNI_LIBS/$payload" \
        --output-root "$NATIVE_SYMBOL_ROOT" --logical-name "$payload" \
        --category managed-runtime --toolchain "$toolchain_id" \
        --source-id "pinned runtime recipe for $payload" --probe-symbol "$probe" \
        --objcopy "$TOOLCHAIN/bin/llvm-objcopy" \
        --readelf "$TOOLCHAIN/bin/llvm-readelf" --nm "$TOOLCHAIN/bin/llvm-nm" \
        --addr2line "$TOOLCHAIN/bin/llvm-addr2line"

    printf '  %-38s %10d bytes  %s\n' "$payload" \
        "$(stat -c%s "$JNI_LIBS/$payload")" \
        "$(sha256sum "$JNI_LIBS/$payload" | cut -d' ' -f1)"
}

report_catalog_reminder() {
    echo
    echo "Record these checksums in the runtime catalog:"
    echo "  $MANIFEST"
    echo
    echo "TestBundledPayloadsMatchTheirPinnedChecksums fails until they match."
}
