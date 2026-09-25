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

command -v rustup >/dev/null || { echo "error: rustup not found; ripgrep needs Rust $POCKETCLAW_RUST_TOOLCHAIN" >&2; exit 1; }
# The pinned compiler, not the host default: rustc decides the payload bytes.
export RUSTUP_TOOLCHAIN="$POCKETCLAW_RUST_TOOLCHAIN"
RUSTC_VERSION="$(rustc --version 2>/dev/null || true)"
case "$RUSTC_VERSION" in
    "rustc $POCKETCLAW_RUST_TOOLCHAIN "*) ;;
    *) echo "error: rustc is '${RUSTC_VERSION:-missing}', pinned $POCKETCLAW_RUST_TOOLCHAIN" >&2
       echo "       Install it: rustup toolchain install $POCKETCLAW_RUST_TOOLCHAIN" >&2
       exit 1 ;;
esac
if ! rustup target list --installed --toolchain "$POCKETCLAW_RUST_TOOLCHAIN" | grep -qx aarch64-linux-android; then
    rustup target add --toolchain "$POCKETCLAW_RUST_TOOLCHAIN" aarch64-linux-android || {
        echo "error: the aarch64-linux-android target is unavailable for Rust $POCKETCLAW_RUST_TOOLCHAIN" >&2
        exit 1
    }
fi

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
cargo build --locked --release --target aarch64-linux-android >/dev/null 2>&1

# The binary records its compiler in .comment; prove it is the pinned one.
# Captured, then matched: `strings | grep -q` under pipefail fails on SIGPIPE.
RG_COMMENT="$("$TOOLCHAIN/bin/llvm-readelf" -p .comment \
    "$BUILD_ROOT/ripgrep-$RG_VERSION/target/aarch64-linux-android/release/rg")"
case "$RG_COMMENT" in
    *"rustc version $POCKETCLAW_RUST_TOOLCHAIN "*) ;;
    *) echo "error: the built rg does not record rustc $POCKETCLAW_RUST_TOOLCHAIN" >&2
       exit 1 ;;
esac

echo
echo "Installed:"
install_payload "$BUILD_ROOT/ripgrep-$RG_VERSION/target/aarch64-linux-android/release/rg" "libpocketclaw-rg.so"
report_catalog_reminder
