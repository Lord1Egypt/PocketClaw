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

GH_VERSION="2.101.0"
GH_TARBALL="gh-$GH_VERSION.tar.gz"
GH_URL="https://github.com/cli/cli/archive/refs/tags/v$GH_VERSION.tar.gz"
GH_SHA256="a266fe8575c0e061b987920c1831a15f71bf0036a8729a5ebb93c2fb0164899c"

GO_CACHE_ROOT="${GO_CACHE_ROOT:-/home/lordegypt/PocketCLaw/.tooling/go}"
export GOCACHE="${GOCACHE:-$GO_CACHE_ROOT/go-build}"
export GOMODCACHE="${GOMODCACHE:-$GO_CACHE_ROOT/go-mod}"
# Exactly the pinned Go, never "auto" and never the host's own: a newer Go
# would silently produce different bytes. Go fetches it if absent.
export GOTOOLCHAIN="$POCKETCLAW_GH_GO_TOOLCHAIN"

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
# from ConnectivityManager in POCKETCLAW_DNS_SERVER, and pkg/androiddns installs
# a resolver that uses them. Those files are copied in here rather than
# reimplemented, so there is one resolver in the repository and gh cannot drift
# from it.
#
# The resolver used to import nothing outside the standard library. It now reads
# its variable through pkg/canonicalenv, which accepts the canonical
# POCKETCLAW_* name and the legacy PICOCLAW_* one. canonicalenv is itself a
# std-lib-only leaf, so both files are vendored, and the guard below allows that
# one import and nothing else. If either file grows a dependency that is not on
# this list, the build stops rather than producing a gh whose DNS behaviour has
# silently diverged from Core's.
CORE_DNS_SOURCE="$REPO_ROOT/core/src/pkg/androiddns/resolver.go"
CORE_ENV_SOURCE="$REPO_ROOT/core/src/pkg/canonicalenv/canonicalenv.go"
for src in "$CORE_DNS_SOURCE" "$CORE_ENV_SOURCE"; do
    [ -f "$src" ] || { echo "error: missing shared source: $src" >&2; exit 1; }
done

# canonicalenv is the only module import either file may have.
UNEXPECTED="$(grep -hoE '^\s+"github\.com/[^"]+"' "$CORE_DNS_SOURCE" "$CORE_ENV_SOURCE" \
    | tr -d ' \t"' | grep -vFx 'github.com/sipeed/picoclaw/pkg/canonicalenv' || true)"
if [ -n "$UNEXPECTED" ]; then
    echo "error: the vendored Android resolver imports a module package that is" >&2
    echo "       not vendored with it:" >&2
    printf '         %s\n' $UNEXPECTED >&2
    echo "       Vendor it here too, or split the leaf out." >&2
    exit 1
fi

mkdir -p internal/androiddns internal/canonicalenv
cp "$CORE_DNS_SOURCE" internal/androiddns/resolver.go
cp "$CORE_ENV_SOURCE" internal/canonicalenv/canonicalenv.go
# Repoint the vendored copy at the vendored dependency.
sed -i 's|"github.com/sipeed/picoclaw/pkg/canonicalenv"|"github.com/cli/cli/v2/internal/canonicalenv"|' \
    internal/androiddns/resolver.go
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
    -ldflags "-X github.com/cli/cli/v2/internal/build.Version=$GH_VERSION" \
    -o "$BUILD_ROOT/gh-android-arm64" ./cmd/gh

BUILT_GO="$(go version "$BUILD_ROOT/gh-android-arm64" | awk '{print $NF}')"
if [ "$BUILT_GO" != "$POCKETCLAW_GH_GO_TOOLCHAIN" ]; then
    echo "error: gh was built with $BUILT_GO, pinned $POCKETCLAW_GH_GO_TOOLCHAIN" >&2
    exit 1
fi

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
