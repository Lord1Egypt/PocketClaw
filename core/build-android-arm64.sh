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
JNI_LIBS="${JNI_LIBS:-$REPO_ROOT/android/app/src/main/jniLibs/arm64-v8a}"
CORE_BUILD_DIR="${CORE_BUILD_DIR:-build}"
NATIVE_SYMBOL_ROOT="${NATIVE_SYMBOL_ROOT:-$REPO_ROOT/build/private-symbols/native/android-arm64}"
# shellcheck source=runtime/toolchains.env
source "$REPO_ROOT/runtime/toolchains.env"
NDK_ROOT="${NDK_ROOT:-/home/lordegypt/PocketCLaw/.tooling/android-sdk/ndk/$POCKETCLAW_NDK_VERSION}"
ELF_TOOLS="$NDK_ROOT/toolchains/llvm/prebuilt/linux-x86_64/bin"
if [[ "$CORE_BUILD_DIR" = /* ]]; then
    CORE_OUTPUT_ROOT="$CORE_BUILD_DIR"
else
    CORE_OUTPUT_ROOT="$CORE_SRC/$CORE_BUILD_DIR"
fi

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
# Exactly the pinned Go (runtime/toolchains.env), resolved into GOMODCACHE if
# the host lacks it. "auto" would let a newer host Go build Core instead.
export GOTOOLCHAIN="$POCKETCLAW_CORE_GO_TOOLCHAIN"
export PATH="$PNPM_BIN_DIR:$PATH"

[ -d "$CORE_SRC" ] || { echo "error: repository-local Core source missing: $CORE_SRC" >&2; exit 1; }
ndk_revision="$(sed -n 's/^Pkg.Revision *= *//p' "$NDK_ROOT/source.properties" 2>/dev/null || true)"
[ "$ndk_revision" = "$POCKETCLAW_NDK_VERSION" ] || {
    echo "error: NDK at $NDK_ROOT is '${ndk_revision:-unknown}', pinned $POCKETCLAW_NDK_VERSION" >&2
    exit 1
}

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
echo "  build time: $BUILD_TIME${POCKETCLAW_BUILD_EPOCH:+ (POCKETCLAW_BUILD_EPOCH=$POCKETCLAW_BUILD_EPOCH)}"
echo

make build-android-arm64          VERSION="$CORE_VERSION" GIT_COMMIT="$CORE_GIT_COMMIT" \
                                  BUILD_TIME="$BUILD_TIME" \
                                  SOURCE_FINGERPRINT="$SOURCE_FINGERPRINT" \
                                  BUILD_DIR="$CORE_BUILD_DIR" STRIP_LDFLAGS=
make build-launcher-android-arm64 VERSION="$CORE_VERSION" GIT_COMMIT="$CORE_GIT_COMMIT" \
                                  BUILD_TIME="$BUILD_TIME" \
                                  SOURCE_FINGERPRINT="$SOURCE_FINGERPRINT" \
                                  BUILD_DIR="$CORE_BUILD_DIR" STRIP_LDFLAGS=

# The upstream build emits picoclaw-android-arm64 and
# picoclaw-launcher-android-arm64; those intermediate names stay as upstream
# writes them. What PocketClaw packages is its own identity, so the install
# destination — the name that reaches nativeLibraryDir, /proc/<pid>/comm and the
# APK payload — is libpocketclaw*.so. Renaming here rather than in the Makefile
# keeps the upstream build recipe untouched.
mkdir -p "$JNI_LIBS"
install -m 0755 "$CORE_OUTPUT_ROOT/picoclaw-android-arm64"          "$JNI_LIBS/libpocketclaw.so"
install -m 0755 "$CORE_OUTPUT_ROOT/picoclaw-launcher-android-arm64" "$JNI_LIBS/libpocketclaw-web.so"
"$ELF_TOOLS/llvm-strip" --strip-unneeded "$JNI_LIBS/libpocketclaw.so"
"$ELF_TOOLS/llvm-strip" --strip-unneeded "$JNI_LIBS/libpocketclaw-web.so"

for lib in libpocketclaw.so libpocketclaw-web.so; do
    built_go="$(go version "$JNI_LIBS/$lib" | awk '{print $NF}')"
    [ "$built_go" = "$POCKETCLAW_CORE_GO_TOOLCHAIN" ] || {
        echo "error: $lib was built with $built_go, pinned $POCKETCLAW_CORE_GO_TOOLCHAIN" >&2
        exit 1
    }
done

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

# Both shipped binaries are stripped, so the only way to read a future crash
# address is the private companion derived here from the same link. It is
# archived last: a build rejected by any check above must not leave symbols
# describing bytes that were never adopted.
echo
for lib in libpocketclaw.so libpocketclaw-web.so; do
    case "$lib" in
        libpocketclaw.so)     source_binary=picoclaw-android-arm64;          category=core ;;
        libpocketclaw-web.so) source_binary=picoclaw-launcher-android-arm64; category=core-launcher ;;
    esac
    python3 "$REPO_ROOT/tool/native_support.py" \
        --source "$CORE_OUTPUT_ROOT/$source_binary" \
        --shipped "$JNI_LIBS/$lib" --output-root "$NATIVE_SYMBOL_ROOT" \
        --logical-name "$lib" --category "$category" --toolchain go \
        --source-id "Core $CORE_VERSION ($CORE_GIT_COMMIT), fingerprint $SOURCE_FINGERPRINT" \
        --probe-symbol main.main --objcopy "$ELF_TOOLS/llvm-objcopy" \
        --readelf "$ELF_TOOLS/llvm-readelf" --nm "$ELF_TOOLS/llvm-nm" \
        --addr2line "$ELF_TOOLS/llvm-addr2line"
done

echo
echo "Core build complete. Package with:"
echo "  core/verify-no-external-source.sh   # optional: prove repo-local build"
echo "  cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64"
