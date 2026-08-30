#!/usr/bin/env bash
#
# EXPERIMENTAL — PYTHON LITE PHASE A. NOT A SHIPPING PAYLOAD.
#
# Builds a standalone CPython interpreter for Android ARM64 and appends the
# trimmed standard library to it as a zip. The result is a single self-contained
# ELF that runs from nativeLibraryDir under the Managed Runtime execution model.
#
# This script is committed as the reproducible record of the Phase A build. It
# does NOT install a payload and is not referenced by the Gradle build guard or
# the Runtime catalog. Do not wire it in before the Phase B decision.
#
# PROVENANCE GAP (must close in Phase B): bzip2 and xz come from upstream
# CPython's prebuilt Android dependency release. Upstream fetches those with no
# checksum at all; this script at least pins them by SHA-256, but they remain
# third-party *binaries* rather than locally built pinned source. SQLite is
# already built here from PocketClaw's own pinned amalgamation.
#
set -euo pipefail

source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

PYTHON_VERSION="3.14.7"
PYTHON_TGZ="Python-$PYTHON_VERSION.tgz"
PYTHON_URL="https://www.python.org/ftp/python/$PYTHON_VERSION/$PYTHON_TGZ"
PYTHON_SHA256="62859805f6fdf25e2bcbf3fa3217801e1996887ca33e6a2af80674bdfa2dbe07"

SQLITE_VERSION="3500400"
SQLITE_ZIP="sqlite-amalgamation-$SQLITE_VERSION.zip"
SQLITE_URL="https://sqlite.org/2025/$SQLITE_ZIP"
SQLITE_SHA256="1d3049dd0f830a025a53105fc79fd2ab9431aea99e137809d064d8ee8356b032"

DEPS_URL="https://github.com/beeware/cpython-android-source-deps/releases/download"
BZIP2_NAME="bzip2-1.0.8-3"
BZIP2_SHA256="2385f46e173d525f079946957c007000a8ad11d8496ba66bae99129183d74bd9"
XZ_NAME="xz-5.4.6-1"
XZ_SHA256="320b76d45dc3499cf855e5310f875cba61c2608e4a98bb280cc4f1b8f189da1a"

# CPython compiles its own project base into the interpreter, so the build must
# happen at a path that identifies nobody. -ffile-prefix-map handles __FILE__;
# this handles the rest.
NEUTRAL_ROOT="${PYTHON_BUILD_ROOT:-/tmp/pocketclaw-build}"
PREFIX_DIR="$NEUTRAL_ROOT/prefix"
SRC_DIR="$NEUTRAL_ROOT/cpython"

fetch_pinned "$PYTHON_URL" "$PYTHON_TGZ" "$PYTHON_SHA256"
fetch_pinned "$SQLITE_URL" "$SQLITE_ZIP" "$SQLITE_SHA256"
fetch_pinned "$DEPS_URL/$BZIP2_NAME/$BZIP2_NAME-aarch64-linux-android.tar.gz" \
             "$BZIP2_NAME-aarch64-linux-android.tar.gz" "$BZIP2_SHA256"
fetch_pinned "$DEPS_URL/$XZ_NAME/$XZ_NAME-aarch64-linux-android.tar.gz" \
             "$XZ_NAME-aarch64-linux-android.tar.gz" "$XZ_SHA256"

rm -rf "$NEUTRAL_ROOT"
mkdir -p "$PREFIX_DIR/include" "$PREFIX_DIR/lib" "$SRC_DIR"
tar xzf "$CACHE_DIR/$PYTHON_TGZ" -C "$NEUTRAL_ROOT"
mv "$NEUTRAL_ROOT/Python-$PYTHON_VERSION"/* "$SRC_DIR/"

( cd "$PREFIX_DIR" \
  && tar xzf "$CACHE_DIR/$BZIP2_NAME-aarch64-linux-android.tar.gz" \
  && tar xzf "$CACHE_DIR/$XZ_NAME-aarch64-linux-android.tar.gz" )

echo "Building SQLite $SQLITE_VERSION from the pinned amalgamation"
unzip -o -q "$CACHE_DIR/$SQLITE_ZIP" -d "$NEUTRAL_ROOT"
( cd "$NEUTRAL_ROOT/sqlite-amalgamation-$SQLITE_VERSION"
  "$TOOLCHAIN/bin/$TARGET_CC" -c sqlite3.c -o sqlite3.o -O2 -fPIC \
    -DSQLITE_ENABLE_FTS5 -DSQLITE_ENABLE_JSON1 -DSQLITE_ENABLE_RTREE \
    -DSQLITE_ENABLE_COLUMN_METADATA -DSQLITE_THREADSAFE=1 \
    -DSQLITE_OMIT_LOAD_EXTENSION -D__BIONIC_NO_PAGE_SIZE_MACRO
  "$TOOLCHAIN/bin/llvm-ar" rcs "$PREFIX_DIR/lib/libsqlite3.a" sqlite3.o
  cp sqlite3.h sqlite3ext.h "$PREFIX_DIR/include/" )

# Upstream's Android tooling pins NDK 27.3; PocketClaw builds every payload on
# one toolchain, and 28.2 builds CPython 3.14.7 unmodified.
sed -i "s/^ndk_version=.*/ndk_version=${NDK_VERSION:-28.2.13676358}/" \
    "$SRC_DIR/Android/android-env.sh"

echo "Building the host (build) interpreter"
( cd "$SRC_DIR" && python3 Android/android.py configure-build >/dev/null \
                && python3 Android/android.py make-build >/dev/null )
BUILD_PYTHON="$SRC_DIR/cross-build/build/python"

echo "Cross-compiling CPython $PYTHON_VERSION for aarch64-linux-android"
export ANDROID_HOME HOST=aarch64-linux-android ANDROID_API_LEVEL="$ANDROID_API"
export PREFIX="$PREFIX_DIR" MODULE_BUILDTYPE=static
# shellcheck disable=SC1091
source "$SRC_DIR/Android/android-env.sh"
export PATH="$(dirname "$AR"):$PATH"          # configure looks for llvm-ar on PATH
export CFLAGS="$CFLAGS -ffile-prefix-map=$SRC_DIR=/pocketclaw/cpython"
export CXXFLAGS="$CFLAGS"

OBJ="$NEUTRAL_ROOT/obj"
rm -rf "$OBJ"; mkdir -p "$OBJ"; cd "$OBJ"
"$SRC_DIR/configure" \
    --host=aarch64-linux-android \
    --build="$(gcc -dumpmachine)" \
    --with-build-python="$BUILD_PYTHON" \
    --prefix=/pocketclaw/python \
    --without-ensurepip \
    --disable-test-modules \
    --with-lto >/dev/null

# The Lite profile. Network, TLS, FFI and the CJK codecs are deliberately absent:
# see runtime/PYTHON_LITE_PHASE_A.md for why each one is out.
mkdir -p Modules
cat > Modules/Setup.local <<'SETUP'
*disabled*
_socket
_ssl
_hashlib
_ctypes
_zstd
_asyncio
_codecs_cn
_codecs_hk
_codecs_iso2022
_codecs_jp
_codecs_kr
_codecs_tw
_multibytecodec
_interpreters
_interpchannels
_interpqueues
_remote_debugging
_lsprof
_zoneinfo
syslog
termios
SETUP

make -j"$(nproc)"
make install prefix="$NEUTRAL_ROOT/install" >/dev/null

echo "Assembling the payload"
INTERP="$NEUTRAL_ROOT/libpocketclaw-python.so"
cp "$OBJ/python" "$INTERP"
# Strip BEFORE appending: stripping afterwards would discard the archive.
"$TOOLCHAIN/bin/llvm-strip" --strip-unneeded "$INTERP"
"$BUILD_PYTHON" "$(dirname "${BASH_SOURCE[0]}")/python-lite-stdlib.py" \
    "$NEUTRAL_ROOT/install/lib/python3.14" "$NEUTRAL_ROOT/stdlib"
cat "$INTERP" "$NEUTRAL_ROOT/stdlib/stdlib-pyc.zip" > "$NEUTRAL_ROOT/payload.so"

echo "Payload: $NEUTRAL_ROOT/payload.so"
echo "  bytes:  $(stat -c%s "$NEUTRAL_ROOT/payload.so")"
echo "  sha256: $(sha256sum "$NEUTRAL_ROOT/payload.so" | cut -d' ' -f1)"
echo
echo "PHASE A: no payload installed and no catalog entry written, by design."
