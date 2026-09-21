#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime jq payload for Android ARM64.
#
# The payload is packaged as android/app/src/main/jniLibs/arm64-v8a/libpocketclaw-jq.so.
# The lib*.so name is not cosmetic: Android's package manager only unpacks
# lib/<abi>/*.so entries into nativeLibraryDir, and nativeLibraryDir is the only
# place an app targeting API 29+ may execute a file from. A payload under any
# other name would ship inside the APK and never be runnable.
#
# The build is hash-pinned to an official jq release tarball and runs in a fixed
# neutral directory so no developer-machine path is baked into the binary.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

JQ_VERSION="1.7.1"
JQ_TARBALL="jq-${JQ_VERSION}.tar.gz"
JQ_URL="https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/${JQ_TARBALL}"
JQ_TARBALL_SHA256="478c9ca129fd2e3443fe27314b455e211e0d8c60bc8ff7df703873deeee580c2"

PAYLOAD_NAME="libpocketclaw-jq.so"
fetch_pinned "$JQ_URL" "$JQ_TARBALL" "$JQ_TARBALL_SHA256"

rm -rf "$BUILD_ROOT/jq-$JQ_VERSION"
tar xzf "$CACHE_DIR/$JQ_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/jq-$JQ_VERSION"

export PATH="$TOOLCHAIN/bin:$PATH"
export CC="aarch64-linux-android${ANDROID_API}-clang"
export AR="llvm-ar"
export RANLIB="llvm-ranlib"
# -fPIE/-pie: Android has required position-independent executables since API 21.
export CFLAGS="-Os -fPIE $NATIVE_DEBUG_CFLAGS"
export LDFLAGS="-pie"

./configure \
    --host=aarch64-linux-android \
    --build=x86_64-pc-linux-gnu \
    --disable-shared --enable-static \
    --with-oniguruma=builtin \
    --disable-docs --disable-maintainer-mode --disable-valgrind

# jq exposes its configure command through $JQ_BUILD_CONFIGURATION. Autoconf
# includes the literal CFLAGS there, which would disclose the temporary root
# used by the reproducibility build even though Clang correctly prefix-maps all
# compiled source paths. Replace that generated source input with a stable,
# truthful release description before compilation; the finished ELF is never
# patched.
cat > src/config_opts.inc <<'CONFIG'
#define JQ_CONFIG "PocketClaw Android arm64 reproducible build"
CONFIG

make -j"$(nproc)"

echo
echo "Installed:"
install_payload "$BUILD_ROOT/jq-$JQ_VERSION/jq" "$PAYLOAD_NAME"
report_catalog_reminder
