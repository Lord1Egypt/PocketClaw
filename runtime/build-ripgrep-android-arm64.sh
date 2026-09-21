#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime ripgrep payload for Android ARM64.
#
# ripgrep is the cheapest high-value tool in the pack: recursive regex search
# with gitignore awareness, for a little over 4 MB. Android's toybox grep cannot
# do it.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

RG_VERSION="14.1.1"
RG_TARBALL="ripgrep-$RG_VERSION.tar.gz"
RG_URL="https://github.com/BurntSushi/ripgrep/archive/refs/tags/$RG_VERSION.tar.gz"
RG_SHA256="4dad02a2f9c8c3c8d89434e47337aa654cb0e2aa50e806589132f186bf5c2b66"

command -v cargo >/dev/null || { echo "error: cargo not found; ripgrep needs a Rust toolchain" >&2; exit 1; }
rustup target add aarch64-linux-android >/dev/null 2>&1 || true

fetch_pinned "$RG_URL" "$RG_TARBALL" "$RG_SHA256"

echo "Building ripgrep $RG_VERSION"
rm -rf "$BUILD_ROOT/ripgrep-$RG_VERSION"
tar xzf "$CACHE_DIR/$RG_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/ripgrep-$RG_VERSION"

# Rust bakes source paths into panic-location strings, which stripping cannot
# remove. A neutral CARGO_HOME keeps the dependency registry out of them and
# --remap-path-prefix rewrites what is left, so the shipped payload carries no
# developer-machine path. install_payload fails the build if any survives.
export CARGO_HOME="$BUILD_ROOT/cargo"
export RUSTFLAGS="-C debuginfo=2 --remap-path-prefix=$CARGO_HOME=/pocketclaw-runtime/cargo --remap-path-prefix=$BUILD_ROOT=/pocketclaw-runtime/build"

CARGO_TARGET_AARCH64_LINUX_ANDROID_LINKER="$TOOLCHAIN/bin/$TARGET_CC" \
CC_aarch64_linux_android="$TOOLCHAIN/bin/$TARGET_CC" \
AR_aarch64_linux_android="$TOOLCHAIN/bin/llvm-ar" \
cargo build --release --target aarch64-linux-android >/dev/null 2>&1

echo
echo "Installed:"
install_payload "$BUILD_ROOT/ripgrep-$RG_VERSION/target/aarch64-linux-android/release/rg" "libpocketclaw-rg.so"
report_catalog_reminder
