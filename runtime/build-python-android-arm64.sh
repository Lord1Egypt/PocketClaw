#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime python payload for Android ARM64.
#
# The result is a single self-contained PIE executable: CPython with every
# standard-library extension module linked in statically, and the pure-Python
# standard library appended to the ELF as a zip that zipimport reads directly.
# There is no lib-dynload, no second payload and nothing written to app storage.
#
# Every dependency is built here from pinned source. Upstream CPython's Android
# tooling downloads prebuilt dependency binaries with no checksum verification;
# this script uses none of them.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

PYTHON_VERSION="3.14.7"
PYTHON_TGZ="Python-$PYTHON_VERSION.tgz"
PYTHON_URL="https://www.python.org/ftp/python/$PYTHON_VERSION/$PYTHON_TGZ"
PYTHON_SHA256="62859805f6fdf25e2bcbf3fa3217801e1996887ca33e6a2af80674bdfa2dbe07"

SQLITE_VERSION="3500400"
SQLITE_ZIP="sqlite-amalgamation-$SQLITE_VERSION.zip"
SQLITE_URL="https://sqlite.org/2025/$SQLITE_ZIP"
SQLITE_SHA256="1d3049dd0f830a025a53105fc79fd2ab9431aea99e137809d064d8ee8356b032"

BZIP2_VERSION="1.0.8"
BZIP2_TGZ="bzip2-$BZIP2_VERSION.tar.gz"
BZIP2_URL="https://sourceware.org/pub/bzip2/$BZIP2_TGZ"
BZIP2_SHA256="ab5a03176ee106d3f0fa90e381da478ddae405918153cca248e682cd0c4a2269"

# 5.4.7 is the last release on the 5.4 branch. The CVE-2024-3094 backdoor was
# introduced in 5.6.0 and removed after 5.6.1; the 5.4 branch never carried it.
XZ_VERSION="5.4.7"
XZ_TGZ="xz-$XZ_VERSION.tar.gz"
XZ_URL="https://github.com/tukaani-project/xz/releases/download/v$XZ_VERSION/$XZ_TGZ"
XZ_SHA256="8db6664c48ca07908b92baedcfe7f3ba23f49ef2476864518ab5db6723836e71"

PY_ROOT="$BUILD_ROOT/python"
PY_PREFIX="$PY_ROOT/prefix"
PY_SRC="$PY_ROOT/cpython"

fetch_pinned "$PYTHON_URL" "$PYTHON_TGZ" "$PYTHON_SHA256"
fetch_pinned "$SQLITE_URL" "$SQLITE_ZIP" "$SQLITE_SHA256"
fetch_pinned "$BZIP2_URL"  "$BZIP2_TGZ"  "$BZIP2_SHA256"
fetch_pinned "$XZ_URL"     "$XZ_TGZ"     "$XZ_SHA256"

rm -rf "$PY_ROOT"
mkdir -p "$PY_PREFIX/include" "$PY_PREFIX/lib" "$PY_SRC"

export CC="$TOOLCHAIN/bin/$TARGET_CC"
export AR="$TOOLCHAIN/bin/llvm-ar"
export RANLIB="$TOOLCHAIN/bin/llvm-ranlib"
export STRIP="$TOOLCHAIN/bin/llvm-strip"
PY_NATIVE_DEBUG_CFLAGS="-g -ffile-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build -fdebug-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build -fmacro-prefix-map=$BUILD_ROOT=/pocketclaw-runtime/build"

echo "Building bzip2 $BZIP2_VERSION"
tar xzf "$CACHE_DIR/$BZIP2_TGZ" -C "$PY_ROOT"
( cd "$PY_ROOT/bzip2-$BZIP2_VERSION"
  "$CC" -c -O2 -fPIC $PY_NATIVE_DEBUG_CFLAGS -D_FILE_OFFSET_BITS=64 -D__BIONIC_NO_PAGE_SIZE_MACRO \
      blocksort.c huffman.c crctable.c randtable.c compress.c decompress.c bzlib.c
  "$AR" rcs "$PY_PREFIX/lib/libbz2.a" ./*.o
  cp bzlib.h "$PY_PREFIX/include/" )

echo "Building xz $XZ_VERSION (liblzma only)"
tar xzf "$CACHE_DIR/$XZ_TGZ" -C "$PY_ROOT"
( cd "$PY_ROOT/xz-$XZ_VERSION"
  ./configure --host=aarch64-linux-android --prefix="$PY_PREFIX" \
      --enable-static --disable-shared \
      --disable-xz --disable-xzdec --disable-lzmadec --disable-lzmainfo \
      --disable-lzma-links --disable-scripts --disable-doc --disable-nls \
      CFLAGS="-O2 -fPIC $PY_NATIVE_DEBUG_CFLAGS -D__BIONIC_NO_PAGE_SIZE_MACRO" >/dev/null
  make -j"$(nproc)" >/dev/null
  make install >/dev/null )

echo "Building SQLite $SQLITE_VERSION"
unzip -o -q "$CACHE_DIR/$SQLITE_ZIP" -d "$PY_ROOT"
( cd "$PY_ROOT/sqlite-amalgamation-$SQLITE_VERSION"
  "$CC" -c sqlite3.c -o sqlite3.o -O2 -fPIC $PY_NATIVE_DEBUG_CFLAGS \
      -DSQLITE_ENABLE_FTS5 -DSQLITE_ENABLE_JSON1 -DSQLITE_ENABLE_RTREE \
      -DSQLITE_ENABLE_COLUMN_METADATA -DSQLITE_THREADSAFE=1 \
      -DSQLITE_OMIT_LOAD_EXTENSION -D__BIONIC_NO_PAGE_SIZE_MACRO
  "$AR" rcs "$PY_PREFIX/lib/libsqlite3.a" sqlite3.o
  cp sqlite3.h sqlite3ext.h "$PY_PREFIX/include/" )

echo "Unpacking CPython $PYTHON_VERSION"
tar xzf "$CACHE_DIR/$PYTHON_TGZ" -C "$PY_ROOT"
mv "$PY_ROOT/Python-$PYTHON_VERSION"/* "$PY_SRC/"

# Upstream's Android tooling pins NDK 27.3 and installs it if absent. PocketClaw
# builds every payload on one toolchain, and 28.2 builds CPython 3.14.7 unpatched.
sed -i "s/^ndk_version=.*/ndk_version=$(basename "$NDK_ROOT")/" \
    "$PY_SRC/Android/android-env.sh"

export ANDROID_HOME="${ANDROID_HOME:-$(dirname "$(dirname "$NDK_ROOT")")}"

# The dependency builds above needed the cross toolchain in the environment.
# The "build" interpreter is a native host build and must not see it, or its
# configure step fails with "cannot run C compiled programs". The cross tools
# come back from android-env.sh for the host build below.
unset CC AR RANLIB STRIP

echo "Building the host (build) interpreter"
( cd "$PY_SRC"
  python3 Android/android.py configure-build >/dev/null
  python3 Android/android.py make-build >/dev/null )
BUILD_PYTHON="$PY_SRC/cross-build/build/python"

# CPython compiles its configure-time VPATH into getpath.c as a runtime fallback.
# The out-of-tree source directory is useful to make, but meaningless on an
# Android device and would disclose whichever temporary root built the payload.
# Normalize that generated C input only after the native host interpreter is
# complete: that build tool needs its real source tree, while the Android target
# does not. Prefix-map flags cannot rewrite an explicit C string literal.
python3 - "$PY_SRC/Makefile.pre.in" <<'NORMALIZE_VPATH'
import pathlib, sys
path = pathlib.Path(sys.argv[1])
text = path.read_text()
needle = "-DVPATH='\"$(VPATH)\"'"
replacement = "-DVPATH='\"/pocketclaw/cpython\"'"
if text.count(needle) != 1:
    raise SystemExit(f"error: expected exactly one CPython VPATH definition, found {text.count(needle)}")
path.write_text(text.replace(needle, replacement))
NORMALIZE_VPATH

echo "Cross-compiling CPython $PYTHON_VERSION for aarch64-linux-android"
export HOST=aarch64-linux-android
export ANDROID_API_LEVEL="$ANDROID_API"
export PREFIX="$PY_PREFIX"
export MODULE_BUILDTYPE=static           # link stdlib extensions into the binary
# shellcheck disable=SC1091
source "$PY_SRC/Android/android-env.sh"
export PATH="$TOOLCHAIN/bin:$PATH"       # configure looks for llvm-ar on PATH
# __FILE__ from assert() would otherwise name the build machine in the binary.
export CFLAGS="$CFLAGS $PY_NATIVE_DEBUG_CFLAGS"
export CXXFLAGS="$CFLAGS"

OBJ="$PY_ROOT/obj"
mkdir -p "$OBJ"
( cd "$OBJ"
  # Invoke configure relative to the object root. With full LTO, Clang retains
  # the source filename as the bitcode module identifier; an absolute srcdir
  # changes link layout across otherwise equivalent build roots even after
  # debug/file prefix maps have normalized the final paths.
  ../cpython/configure \
      --host=aarch64-linux-android \
      --build="$("$PY_SRC/config.guess")" \
      --with-build-python="$BUILD_PYTHON" \
      --prefix=/pocketclaw/python \
      --without-ensurepip \
      --disable-test-modules \
      --with-lto >/dev/null

  # The Lite profile. Networking, TLS, FFI and the CJK codecs are deliberately
  # absent; see runtime/PYTHON_LITE_PHASE_A.md for why each one is out.
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

  make -j"$(nproc)" >/dev/null
  make install prefix="$PY_ROOT/install" >/dev/null )

echo "Packaging the standard library"
"$BUILD_PYTHON" "$(dirname "${BASH_SOURCE[0]}")/python-lite-stdlib.py" \
    "$PY_ROOT/install/lib/python3.14" "$PY_ROOT/stdlib"

echo "Assembling the payload"
INTERP="$PY_ROOT/interpreter.so"
cp "$OBJ/python" "$INTERP"
# Strip BEFORE appending. llvm-strip rewrites the file and would drop the zip.
"$TOOLCHAIN/bin/llvm-strip" --strip-unneeded "$INTERP"
cat "$INTERP" "$PY_ROOT/stdlib/stdlib-pyc.zip" > "$PY_ROOT/payload.so"

# The interpreter must depend on nothing but the Android platform. A stray
# NEEDED entry means a dependency leaked in and would fail on the device.
EXPECTED_NEEDED="libc.so libdl.so liblog.so libm.so libz.so"
ACTUAL_NEEDED="$("$TOOLCHAIN/bin/llvm-readelf" -dW "$PY_ROOT/payload.so" \
    | sed -n 's/.*(NEEDED).*\[\(.*\)\]/\1/p' | sort -u | tr '\n' ' ' | sed 's/ $//')"
if [ "$ACTUAL_NEEDED" != "$EXPECTED_NEEDED" ]; then
    echo "error: unexpected shared library dependencies" >&2
    echo "  expected: $EXPECTED_NEEDED" >&2
    echo "  actual:   $ACTUAL_NEEDED" >&2
    exit 1
fi

# The bootstrap goes inside the payload, so the checksum the runtime verifies
# covers it. Android's CPython points sys.stdout and sys.stderr at the system
# log rather than at descriptors 1 and 2; without this module every print()
# from a managed run lands in logcat and the caller sees an empty stream.
"$BUILD_PYTHON" "$(dirname "${BASH_SOURCE[0]}")/install-python-bootstrap.py" \
    "$PY_ROOT/payload.so"

# zipimport must still find the appended archive after packaging.
"$BUILD_PYTHON" - "$PY_ROOT/payload.so" <<'VERIFY'
import sys, zipfile
path = sys.argv[1]
with open(path, "rb") as fh:
    if fh.read(4) != b"\x7fELF":
        sys.exit("error: payload does not start with an ELF header")
names = zipfile.ZipFile(path).namelist()
if "json/__init__.pyc" not in names:
    sys.exit("error: appended stdlib zip is missing or unreadable")
print(f"  appended stdlib: {len(names)} modules, ELF header intact")
VERIFY

install_payload "$PY_ROOT/payload.so" "libpocketclaw-python.so" no-strip "$OBJ/python"
report_catalog_reminder
