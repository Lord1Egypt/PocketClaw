#!/usr/bin/env bash
#
# Proves the PocketClaw Core build does not read application source from any
# checkout outside this repository.
#
# It makes the historical reference checkout unavailable by renaming it, runs
# the canonical repo-local Core build, and restores the rename afterwards. The
# checkout is never deleted.
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
EXTERNAL="${EXTERNAL_CORE_CHECKOUT:-/home/lordegypt/PocketCLaw/.upstream/picoclaw-core-v0.3.1}"
HIDDEN="$EXTERNAL.hidden-for-verification"

restore() {
    if [ -d "$HIDDEN" ]; then
        mv "$HIDDEN" "$EXTERNAL"
        echo "Restored $EXTERNAL"
    fi
}
trap restore EXIT

if [ -d "$EXTERNAL" ]; then
    mv "$EXTERNAL" "$HIDDEN"
    echo "Hid external reference checkout: $EXTERNAL"
else
    echo "External reference checkout not present; nothing to hide."
fi

echo
"$REPO_ROOT/core/build-android-arm64.sh"

echo
echo "PASS: Core built with the external reference checkout unavailable."
