#!/usr/bin/env bash
#
# Resolves the build timestamp stamped into Core's version metadata.
#
# This is the one place that decides what BuildTime is. It used to be derived
# independently by the Makefile from `date`, which meant identical source
# produced different binaries purely because the clock had moved — the source
# fingerprint stayed correct, but nobody could reproduce a released artifact
# byte for byte from its source.
#
# Contract:
#
#   SOURCE_DATE_EPOCH set      → validated and used. This is the cross-ecosystem
#                                convention, so a downstream reproducer already
#                                knows to pass it.
#   SOURCE_DATE_EPOCH unset    → the timestamp of the most recent commit that
#                                touched a canonical Core build input. Not HEAD:
#                                HEAD moves for documentation, staged binaries
#                                and unrelated application changes, which would
#                                give the same Core source a different timestamp
#                                and quietly undo the whole guarantee.
#   neither available          → FAIL. Falling back to the wall clock is what
#                                this script exists to prevent, and a silent
#                                fallback would make the guarantee worthless
#                                exactly when it is hardest to notice.
#
# Prints one timestamp on stdout, in the same format the build has always
# emitted (`%FT%T%z`), fixed to UTC so the output does not depend on the
# builder's timezone.
#
#   --print-epoch     print the resolved Unix seconds instead
#   --print-commit    print the canonical build-input commit instead
set -euo pipefail

# BUILD_INPUTS is the path set whose history decides the default timestamp:
# everything a canonical `./core/build-android-arm64.sh` actually consumes.
#
# Deliberately broader than the Core *source fingerprint*, which names only
# cmd/, pkg/, workspace/, go.mod, go.sum and the Makefile because those are what
# reach the Core gateway compiler. The canonical build also produces
# libpicoclaw-web.so from core/src/web, and stamps both binaries with one
# timestamp, so web/ materially affects the bytes this script is dating. The two
# sets answer different questions and are allowed to differ.
#
# Deliberately excludes the staged JNI binaries (build *output*, and folding
# them in would make every staging commit move the timestamp), documentation,
# the acceptance baseline, and the Flutter application.
#
# Over-inclusion here is safe and under-inclusion is not: an extra commit moves
# the timestamp slightly more often than strictly necessary, whereas a missing
# path means a real build-input change that does not move it at all.
BUILD_INPUTS=(
    "core/src"
    "core/build-android-arm64.sh"
    "core/resolve-build-time.sh"
)

fail() {
    echo "resolve-build-time: $1" >&2
    exit 1
}

epoch="${SOURCE_DATE_EPOCH:-}"
source_of_truth="SOURCE_DATE_EPOCH"

if [ -n "$epoch" ]; then
    # Digits only, and non-zero: a malformed value must be an error rather
    # than something `date` quietly reinterprets.
    case "$epoch" in
        ''|*[!0-9]*) fail "SOURCE_DATE_EPOCH must be Unix seconds, got: $epoch" ;;
    esac
    [ "$epoch" -gt 0 ] 2>/dev/null || fail "SOURCE_DATE_EPOCH must be a positive integer, got: $epoch"
else
    source_of_truth="canonical Core build-input commit"
    repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    # Path-scoped: a documentation or staged-binary commit must not move this.
    epoch="$(git -C "$repo_root" log -1 --format=%ct -- "${BUILD_INPUTS[@]}" 2>/dev/null || true)"
    build_input_commit="$(git -C "$repo_root" log -1 --format=%H -- "${BUILD_INPUTS[@]}" 2>/dev/null || true)"
    case "$epoch" in
        ''|*[!0-9]*)
            fail "no SOURCE_DATE_EPOCH and no usable canonical Core build-input commit.
  A build outside a git checkout, or one whose history contains no commit
  touching ${BUILD_INPUTS[*]}, must supply the timestamp explicitly:
    SOURCE_DATE_EPOCH=\$(date +%s) ./core/build-android-arm64.sh
  The wall clock is deliberately not used: it would make the build
  unreproducible without saying so."
            ;;
    esac
fi

# GNU date takes -d @<epoch>; BSD/macOS date takes -r <epoch>.
if formatted="$(date -u -d "@$epoch" +%FT%T%z 2>/dev/null)"; then
    :
elif formatted="$(date -u -r "$epoch" +%FT%T%z 2>/dev/null)"; then
    :
else
    fail "could not format epoch $epoch with this date(1)"
fi

if [ "${RESOLVE_BUILD_TIME_VERBOSE:-}" = "1" ]; then
    echo "resolve-build-time: $formatted (from $source_of_truth, epoch $epoch)" >&2
fi

case "${1:-}" in
    --print-epoch)  printf '%s\n' "$epoch" ;;
    --print-commit) printf '%s\n' "${build_input_commit:-}" ;;
    ''|--print-time) printf '%s\n' "$formatted" ;;
    *) fail "unknown argument: $1" ;;
esac
