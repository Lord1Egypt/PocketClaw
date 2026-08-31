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

# Android provides no /etc/resolv.conf, so Go's resolver finds no nameservers
# and falls back to [::1]:53 where nothing listens. Every gh request then fails
# as "error connecting to api.github.com" without leaving the device, which is
# indistinguishable from a rejected credential unless you look at GH_DEBUG.
#
# Core and the launcher already solve this: the Android host passes the servers
# from ConnectivityManager in PICOCLAW_DNS_SERVER, and pkg/androiddns installs a
# resolver that uses them. That file is copied in here rather than reimplemented,
# so there is one resolver in the repository and gh cannot drift from it. It
# imports nothing outside the standard library, which is what makes the copy
# safe; the build fails below if that stops being true.
CORE_DNS_SOURCE="$REPO_ROOT/core/src/pkg/androiddns/resolver.go"
[ -f "$CORE_DNS_SOURCE" ] || {
    echo "error: the shared Android resolver is missing: $CORE_DNS_SOURCE" >&2
    exit 1
}
if grep -qE '^\s+"github\.com/' "$CORE_DNS_SOURCE"; then
    echo "error: $CORE_DNS_SOURCE now imports a module package and can no longer" >&2
    echo "       be copied into the gh build. Vendor it or split the leaf out." >&2
    exit 1
fi
mkdir -p internal/androiddns
cp "$CORE_DNS_SOURCE" internal/androiddns/resolver.go
cat > cmd/gh/pocketclaw_android_dns.go <<'SHIM'
package main

// PocketClaw runs this binary as a managed tool on Android, where Go's resolver
// has no /etc/resolv.conf to read. The host supplies the active network's DNS
// servers in PICOCLAW_DNS_SERVER; this installs a resolver that uses them.
//
// Off Android, or when the variable is absent, it does nothing and gh keeps the
// platform's own behaviour.

import "github.com/cli/cli/v2/internal/androiddns"

func init() {
	androiddns.ConfigureDefaultResolverFromEnvironment()
}
SHIM

GOOS=android GOARCH=arm64 CGO_ENABLED=0 go build -trimpath \
    -ldflags "-s -w -X github.com/cli/cli/v2/internal/build.Version=$GH_VERSION" \
    -o "$BUILD_ROOT/gh-android-arm64" ./cmd/gh

# The shim is only useful if it is actually linked in. A silent drop -- a build
# tag, a moved main package -- would restore the exact failure this fixes.
# grep reads the binary directly rather than through `strings`: under pipefail,
# `grep -q` closing the pipe early makes strings die of SIGPIPE and the whole
# pipeline report failure on a successful match.
if ! grep -qa "PICOCLAW_DNS_SERVER" "$BUILD_ROOT/gh-android-arm64"; then
    echo "error: the built gh does not carry the PocketClaw resolver shim" >&2
    exit 1
fi

echo
echo "Installed:"
install_payload "$BUILD_ROOT/gh-android-arm64" "libpocketclaw-gh.so"
report_catalog_reminder
