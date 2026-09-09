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
#                                touched a canonical Core build input, ignoring
#                                Core *_test.go. Not HEAD:
#                                HEAD moves for documentation, staged binaries
#                                and unrelated application changes, which would
#                                give the same Core source a different timestamp
#                                and quietly undo the whole guarantee.
#   neither available          → FAIL. Falling back to the wall clock is what
#                                this script exists to prevent, and a silent
#                                fallback would make the guarantee worthless
#                                exactly when it is hardest to notice. A shallow
#                                clone counts as unavailable: it has no history
#                                to scope the query against.
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
# This set and the Core *source fingerprint* now describe the same production
# universe. They did not until N4K-A: the fingerprint named only cmd/, pkg/,
# workspace/, go.mod, go.sum and the Makefile — what reaches the Core gateway
# compiler — while this script already covered all of core/src because the
# canonical build also produces libpocketclaw-web.so from core/src/web and
# stamps both binaries with one timestamp. That gap meant a dashboard change
# moved the timestamp but not the fingerprint, so staged freshness stayed green
# against a binary that provably differed. The fingerprint now covers web/ too;
# keep the two in step, and prefer widening both to narrowing either.
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

# Go test files under core/src are excluded, for the same reason the Core source
# fingerprint excludes them (see pkg/coresource/fingerprint.go): a test edit
# cannot change the shipped binary, so demanding a rebuild for one teaches people
# to ignore the guard rather than to trust it. Before this exclusion the two
# rules disagreed — a four-line test edit left the fingerprint identical and
# staged freshness green while core.staged_build_time went red against binaries
# that provably could not differ.
#
# The dashboard frontend has tests too, and they are Vitest files rather than Go
# ones: *.test.ts / *.test.tsx under core/src/web/frontend. `vite build` does not
# emit them, so they cannot reach libpocketclaw-web.so, and the same reasoning
# that excludes *_test.go excludes these. pkg/coresource excludes exactly this
# set; a test holds the two rules together.
#
# The exclusion is exactly and only test files under core/src. Everything that
# takes part in producing the binaries stays in, including this script and the
# build script: a change to how the build is defined is a change to the build,
# so editing either still moves the timestamp and still requires a rebuild.
# That self-provenance is deliberate and is covered by a test.
BUILD_INPUT_EXCLUDES=(
    ":(exclude,glob)core/src/**/*_test.go"
    ":(exclude,glob)core/src/web/frontend/**/*.test.ts"
    ":(exclude,glob)core/src/web/frontend/**/*.test.tsx"
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
    # A shallow clone has no history to scope against. Git treats the graft
    # boundary as a root commit, so every path looks introduced by the tip and
    # the query below returns the tip's timestamp — silently restoring the
    # unscoped-HEAD behaviour this scoping exists to remove, and dating a build
    # by whatever documentation or merge commit happens to be checked out.
    # Wrong-but-plausible is the worst outcome here, so refuse.
    if [ "$(git -C "$repo_root" rev-parse --is-shallow-repository 2>/dev/null)" = "true" ]; then
        fail "refusing to derive the build timestamp from a shallow clone.
  Every path appears to originate at the tip commit, so the timestamp would be
  the tip's rather than the canonical Core build input's. Either fetch the full
  history (actions/checkout with fetch-depth: 0) or supply the timestamp:
    SOURCE_DATE_EPOCH=<seconds> ./core/build-android-arm64.sh"
    fi
    # Path-scoped: a documentation or staged-binary commit must not move this.
    epoch="$(git -C "$repo_root" log -1 --format=%ct -- \
        "${BUILD_INPUTS[@]}" "${BUILD_INPUT_EXCLUDES[@]}" 2>/dev/null || true)"
    build_input_commit="$(git -C "$repo_root" log -1 --format=%H -- \
        "${BUILD_INPUTS[@]}" "${BUILD_INPUT_EXCLUDES[@]}" 2>/dev/null || true)"
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
