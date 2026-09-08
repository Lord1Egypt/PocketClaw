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
#   SOURCE_DATE_EPOCH unset    → the commit timestamp of HEAD, which is a
#                                property of the source rather than of the
#                                machine that happened to build it.
#   neither available          → FAIL. Falling back to the wall clock is what
#                                this script exists to prevent, and a silent
#                                fallback would make the guarantee worthless
#                                exactly when it is hardest to notice.
#
# Prints one timestamp on stdout, in the same format the build has always
# emitted (`%FT%T%z`), fixed to UTC so the output does not depend on the
# builder's timezone.
set -euo pipefail

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
    source_of_truth="git commit timestamp"
    repo_root="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
    epoch="$(git -C "$repo_root" log -1 --format=%ct 2>/dev/null || true)"
    case "$epoch" in
        ''|*[!0-9]*)
            fail "no SOURCE_DATE_EPOCH and no usable git commit timestamp.
  A build outside a git checkout must supply the timestamp explicitly:
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

printf '%s\n' "$formatted"
