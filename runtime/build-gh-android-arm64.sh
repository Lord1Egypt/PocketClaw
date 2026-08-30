#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime GitHub CLI payload for Android ARM64.
#
# gh is by far the largest payload in the pack, at roughly 56 MB installed. It
# ships as a deliberate strategic exception because GitHub capability is core to
# the agent; it is not a precedent for bundling other large tools. Anything else
# above roughly 10 MB installed needs its own justification.
#
# Pure Go with CGO disabled, so there is no dependency to cross-build.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

GH_VERSION="2.82.1"
GH_TARBALL="gh-$GH_VERSION.tar.gz"
GH_URL="https://github.com/cli/cli/archive/refs/tags/v$GH_VERSION.tar.gz"
GH_SHA256="999bdea5c8baf3d03fe0314127c2c393d6c0f7a504a573ad0c107072973af973"

GO_CACHE_ROOT="${GO_CACHE_ROOT:-/home/lordegypt/PocketCLaw/.tooling/go}"
export GOCACHE="${GOCACHE:-$GO_CACHE_ROOT/go-build}"
export GOMODCACHE="${GOMODCACHE:-$GO_CACHE_ROOT/go-mod}"
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"

fetch_pinned "$GH_URL" "$GH_TARBALL" "$GH_SHA256"

echo "Building gh $GH_VERSION"
rm -rf "$BUILD_ROOT/cli-$GH_VERSION"
tar xzf "$CACHE_DIR/$GH_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/cli-$GH_VERSION"

GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/cli/cli/v2/internal/build.Version=$GH_VERSION" \
    -o "$BUILD_ROOT/gh-android-arm64" ./cmd/gh

echo
echo "Installed:"
install_payload "$BUILD_ROOT/gh-android-arm64" "libpocketclaw-gh.so"
report_catalog_reminder
