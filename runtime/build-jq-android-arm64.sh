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
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
JNI_LIBS="$REPO_ROOT/android/app/src/main/jniLibs/arm64-v8a"
MANIFEST="$REPO_ROOT/core/src/pkg/pcruntime/manifest.json"

JQ_VERSION="1.7.1"
JQ_TARBALL="jq-${JQ_VERSION}.tar.gz"
JQ_URL="https://github.com/jqlang/jq/releases/download/jq-${JQ_VERSION}/${JQ_TARBALL}"
JQ_TARBALL_SHA256="478c9ca129fd2e3443fe27314b455e211e0d8c60bc8ff7df703873deeee580c2"

# Android 5.0 is well below PocketClaw's minSdk; API 24 is chosen because the
# NDK still ships a sysroot for it and jq needs nothing newer.
ANDROID_API="24"
PAYLOAD_NAME="libpocketclaw-jq.so"

NDK_ROOT="${NDK_ROOT:-/home/lordegypt/PocketCLaw/.tooling/android-sdk/ndk/28.2.13676358}"
TOOLCHAIN="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64"

# A fixed build directory keeps the developer's home path out of the binary:
# autotools records its own configure arguments in the compiled result.
BUILD_ROOT="${BUILD_ROOT:-/tmp/pocketclaw-runtime-build}"
CACHE_DIR="${CACHE_DIR:-$BUILD_ROOT/downloads}"

[ -d "$TOOLCHAIN/bin" ] || {
    echo "error: NDK toolchain not found at $TOOLCHAIN; set NDK_ROOT" >&2
    exit 1
}

mkdir -p "$CACHE_DIR"
if [ ! -f "$CACHE_DIR/$JQ_TARBALL" ]; then
    echo "Fetching $JQ_URL"
    curl -fsSL -o "$CACHE_DIR/$JQ_TARBALL" "$JQ_URL"
fi

echo "$JQ_TARBALL_SHA256  $CACHE_DIR/$JQ_TARBALL" | sha256sum -c - >/dev/null || {
    echo "error: $JQ_TARBALL does not match its pinned checksum; refusing to build" >&2
    exit 1
}

rm -rf "$BUILD_ROOT/jq-$JQ_VERSION"
tar xzf "$CACHE_DIR/$JQ_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/jq-$JQ_VERSION"

export PATH="$TOOLCHAIN/bin:$PATH"
export CC="aarch64-linux-android${ANDROID_API}-clang"
export AR="llvm-ar"
export RANLIB="llvm-ranlib"
# -fPIE/-pie: Android has required position-independent executables since API 21.
export CFLAGS="-Os -fPIE"
export LDFLAGS="-pie"

./configure \
    --host=aarch64-linux-android \
    --build=x86_64-pc-linux-gnu \
    --disable-shared --enable-static \
    --with-oniguruma=builtin \
    --disable-docs --disable-maintainer-mode --disable-valgrind

make -j"$(nproc)"

install -m 0755 "$BUILD_ROOT/jq-$JQ_VERSION/jq" "$JNI_LIBS/$PAYLOAD_NAME"
"$TOOLCHAIN/bin/llvm-strip" --strip-unneeded "$JNI_LIBS/$PAYLOAD_NAME"

# Build-path privacy, the same rule the Core build enforces: a shipped binary
# must not carry developer-machine paths.
leaked="$(strings -a "$JNI_LIBS/$PAYLOAD_NAME" | grep -c -E '/home/|/Users/|/root/' || true)"
if [ "$leaked" -ne 0 ]; then
    echo "error: $PAYLOAD_NAME contains developer-machine paths" >&2
    exit 1
fi

# The payload must be an ARM64 PIE executable linked only against bionic, or the
# runtime's ABI verification will reject it on the device.
"$TOOLCHAIN/bin/llvm-readelf" -h "$JNI_LIBS/$PAYLOAD_NAME" | grep -q 'AArch64' || {
    echo "error: $PAYLOAD_NAME is not an AArch64 binary" >&2
    exit 1
}

PAYLOAD_SHA256="$(sha256sum "$JNI_LIBS/$PAYLOAD_NAME" | cut -d' ' -f1)"

echo
printf 'Installed %s\n' "$JNI_LIBS/$PAYLOAD_NAME"
printf '  version:  jq %s\n' "$JQ_VERSION"
printf '  size:     %s bytes\n' "$(stat -c%s "$JNI_LIBS/$PAYLOAD_NAME")"
printf '  sha256:   %s\n' "$PAYLOAD_SHA256"
echo
echo "Record this checksum in the runtime catalog:"
echo "  $MANIFEST  ->  tools[jq].sha256"
echo
echo "The Go test TestBundledPayloadsMatchTheirPinnedChecksums fails until it matches."
