#!/usr/bin/env bash
#
# Regenerates core/pocketclaw-core-v0.3.1.patch: the difference between
# upstream PicoClaw v0.3.1 and PocketClaw's repository-local Core baseline.
#
# This is a provenance/maintenance operation, not a build step. It is the only
# script here that reaches upstream, and it only ever reads. Pass an existing
# local PicoClaw checkout as $1 to work without network access.
#
set -euo pipefail

REPO_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
CORE_SRC="$REPO_ROOT/core/src"
PATCH_OUT="$REPO_ROOT/core/pocketclaw-core-v0.3.1.patch"

UPSTREAM_URL="https://github.com/sipeed/picoclaw.git"
UPSTREAM_COMMIT="2cf030d2fd3b871d7ec17e3be34c24688aac76da"

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
WORK="$TMP/tree"
FILE_LIST="$TMP/files.txt"
mkdir -p "$WORK"

if [ "${1:-}" != "" ]; then
    echo "Using local upstream checkout: $1"
    git -C "$1" archive "$UPSTREAM_COMMIT" | tar -x -C "$WORK"
else
    echo "Fetching upstream $UPSTREAM_COMMIT for review only..."
    git -C "$WORK" init -q
    git -C "$WORK" remote add origin "$UPSTREAM_URL"
    git -C "$WORK" fetch -q --depth 1 origin "$UPSTREAM_COMMIT"
    git -C "$WORK" checkout -q FETCH_HEAD
    rm -rf "$WORK/.git"
fi

# Paths PocketClaw deliberately does not vendor. Dropping them from the
# baseline too keeps them out of the patch instead of reporting them as
# PocketClaw deletions. See core/README.md, "Deliberate omissions".
rm -rf "$WORK/assets"          # upstream README media, not a build input
rm -rf "$WORK/pkg/seahorse/.omc"   # an upstream developer's tool-state file

# --force throughout: upstream's own .gitignore excludes paths it nonetheless
# tracks (notably its bare `onboard` rule, which covers
# cmd/picoclaw/internal/onboard/). Without --force those files never enter the
# baseline, and the patch then misreports unmodified upstream source as
# PocketClaw-authored additions.
git -C "$WORK" init -q
git -C "$WORK" add -A --force
git -C "$WORK" -c user.email=core@pocketclaw -c user.name=PocketClaw \
    commit -q -m "upstream picoclaw $UPSTREAM_COMMIT"

# Replace the worktree with PocketClaw's baseline. The file list comes from git
# so build outputs, node_modules, and caches under core/src can never leak into
# the patch. Rebuilding the tree from empty is what makes deletions show up.
git -C "$REPO_ROOT" ls-files -co --exclude-standard -z -- core/src \
    | while IFS= read -r -d '' path; do
        # `git ls-files` includes tracked paths deleted in the worktree. They
        # must stay absent so the generated patch records the deletion.
        if [ -e "$REPO_ROOT/$path" ]; then
            printf '%s\0' "${path#core/src/}"
        fi
    done > "$FILE_LIST"
find "$WORK" -mindepth 1 -maxdepth 1 ! -name .git -exec rm -rf {} +
rsync -a --from0 --files-from="$FILE_LIST" "$CORE_SRC/" "$WORK/"

git -C "$WORK" add -A --force
git -C "$WORK" diff --cached --binary > "$PATCH_OUT"

echo
echo "Wrote $PATCH_OUT"
echo "  changed files: $(git -C "$WORK" diff --cached --name-only | wc -l)"
echo "  patch bytes:   $(stat -c%s "$PATCH_OUT")"
