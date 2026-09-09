#!/usr/bin/env bash
#
# Canonical PocketClaw Core build.
#
# Builds both Android arm64 runtime binaries from the repository-local Core
# source in core/src and installs them into the Android jniLibs payload.
#
# This script reads no source outside this repository. Everything it needs from
# outside is a build tool (Go toolchain cache, pnpm), never application source.
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_SRC="$REPO_ROOT/core/src"
JNI_LIBS="$REPO_ROOT/android/app/src/main/jniLibs/arm64-v8a"

# Upstream provenance stamped into the binaries. These are fixed to the pinned
# PicoClaw baseline and must never be derived from PocketClaw's own git state,
# which would stamp a PocketClaw tag onto upstream-versioned Core.
CORE_VERSION="${CORE_VERSION:-v0.3.1}"
CORE_GIT_COMMIT="${CORE_GIT_COMMIT:-2cf030d2}"

# Build tools. Override these if your toolchain lives elsewhere; see
# core/README.md, "External toolchain prerequisites".
GO_CACHE_ROOT="${GO_CACHE_ROOT:-/home/lordegypt/PocketCLaw/.tooling/go}"
PNPM_BIN_DIR="${PNPM_BIN_DIR:-/home/lordegypt/PocketCLaw/.tooling/pnpm/node_modules/.bin}"

export GOCACHE="${GOCACHE:-$GO_CACHE_ROOT/go-build}"
export GOMODCACHE="${GOMODCACHE:-$GO_CACHE_ROOT/go-mod}"
# The Makefile pins GOTOOLCHAIN=local; go.mod requires a newer Go than the
# system one, and the required toolchain is resolved from GOMODCACHE.
export GOTOOLCHAIN="${GOTOOLCHAIN:-auto}"
export PATH="$PNPM_BIN_DIR:$PATH"

[ -d "$CORE_SRC" ] || { echo "error: repository-local Core source missing: $CORE_SRC" >&2; exit 1; }

cd "$CORE_SRC"

# Fingerprint of the Core source this build consumes, stamped into the binaries
# so the test gate can tell a Core built from the current source from one built
# before it. The catalog guard cannot see a Go-only change: manifest.json is
# unchanged, so a Core built yesterday still embeds today's catalog.
#
# Computed by the same code the gate recomputes with, so the two values cannot
# drift through two implementations of one hash. GOOS/GOARCH are cleared because
# this runs on the build host, not on the Android target.
SOURCE_FINGERPRINT="$(GOOS= GOARCH= go run ./cmd/corefingerprint .)"

# The build timestamp is resolved once, here, and passed explicitly into both
# make invocations. Resolving it in the Makefile instead would let the two
# binaries carry different timestamps, and exporting it would not work at all:
# a `:=` assignment in Make ignores the environment, so only a command-line
# assignment overrides. See core/resolve-build-time.sh for where the value
# comes from; it fails rather than falling back to the wall clock, and `set -e`
# means that failure stops the build here.
BUILD_TIME="$("$REPO_ROOT/core/resolve-build-time.sh")"

echo "PocketClaw Core build"
echo "  source:  $CORE_SRC"
echo "  version: $CORE_VERSION ($CORE_GIT_COMMIT)"
echo "  source fingerprint: $SOURCE_FINGERPRINT"
echo "  build time: $BUILD_TIME${SOURCE_DATE_EPOCH:+ (SOURCE_DATE_EPOCH=$SOURCE_DATE_EPOCH)}"
echo

make build-android-arm64          VERSION="$CORE_VERSION" GIT_COMMIT="$CORE_GIT_COMMIT" \
                                  BUILD_TIME="$BUILD_TIME" \
                                  SOURCE_FINGERPRINT="$SOURCE_FINGERPRINT"
make build-launcher-android-arm64 VERSION="$CORE_VERSION" GIT_COMMIT="$CORE_GIT_COMMIT" \
                                  BUILD_TIME="$BUILD_TIME" \
                                  SOURCE_FINGERPRINT="$SOURCE_FINGERPRINT"

# The upstream build emits picoclaw-android-arm64 and
# picoclaw-launcher-android-arm64; those intermediate names stay as upstream
# writes them. What PocketClaw packages is its own identity, so the install
# destination — the name that reaches nativeLibraryDir, /proc/<pid>/comm and the
# APK payload — is libpocketclaw*.so. Renaming here rather than in the Makefile
# keeps the upstream build recipe untouched.
install -m 0755 "$CORE_SRC/build/picoclaw-android-arm64"          "$JNI_LIBS/libpocketclaw.so"
install -m 0755 "$CORE_SRC/build/picoclaw-launcher-android-arm64" "$JNI_LIBS/libpocketclaw-web.so"

echo
echo "Installed into $JNI_LIBS:"
for lib in libpocketclaw.so libpocketclaw-web.so; do
    printf '  %-20s %12d bytes  %s\n' "$lib" \
        "$(stat -c%s "$JNI_LIBS/$lib")" \
        "$(sha256sum "$JNI_LIBS/$lib" | cut -d' ' -f1)"
done

# Build-path privacy: a release binary must not carry developer-machine paths.
# -trimpath is what keeps this true; this check is what proves it stayed true.
echo
leaked=0
for lib in libpocketclaw.so libpocketclaw-web.so; do
    hits="$(strings -a "$JNI_LIBS/$lib" | grep -c -E '/home/|/Users/|/root/' || true)"
    printf '  %-20s developer paths: %s\n' "$lib" "$hits"
    [ "$hits" -eq 0 ] || leaked=1
done
if [ "$leaked" -ne 0 ]; then
    echo "error: release binaries contain developer-machine paths; -trimpath regressed" >&2
    exit 1
fi

# The stamp is what the test gate greps for. A build that produced a binary
# without it would pass here and fail the gate with a confusing message, so
# check it where the cause is still obvious.
#
# Both binaries, because both ship. Checking only libpocketclaw.so is what let a
# dashboard change escape provenance until N4K-A: the -X flag reaches the
# launcher build through the LDFLAGS passed above, but a linker drops an -X
# target that no live code reads, so the flag succeeding says nothing about the
# binary carrying it. web/backend/main.go reads it back at startup; this is what
# proves that stayed true.
for lib in libpocketclaw.so libpocketclaw-web.so; do
    if ! grep -qa "$SOURCE_FINGERPRINT" "$JNI_LIBS/$lib"; then
        echo "error: staged $lib does not carry its source fingerprint" >&2
        exit 1
    fi
done
echo
echo "  source fingerprint stamped in both binaries: $SOURCE_FINGERPRINT"

echo
echo "Core build complete. Package with:"
echo "  core/verify-no-external-source.sh   # optional: prove repo-local build"
echo "  cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64"
