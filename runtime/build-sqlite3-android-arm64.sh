#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime sqlite3 payload for Android ARM64.
#
# Built from the official amalgamation, which is a single translation unit and
# needs no dependency beyond libc and libm.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

SQLITE_VERSION="3500400"
SQLITE_LABEL="3.50.4"
SQLITE_ZIP="sqlite-amalgamation-$SQLITE_VERSION.zip"
SQLITE_URL="https://sqlite.org/2025/$SQLITE_ZIP"
SQLITE_SHA256="1d3049dd0f830a025a53105fc79fd2ab9431aea99e137809d064d8ee8356b032"

fetch_pinned "$SQLITE_URL" "$SQLITE_ZIP" "$SQLITE_SHA256"

echo "Building sqlite3 $SQLITE_LABEL"
rm -rf "$BUILD_ROOT/sqlite-amalgamation-$SQLITE_VERSION"
unzip -o -q "$CACHE_DIR/$SQLITE_ZIP" -d "$BUILD_ROOT"
cd "$BUILD_ROOT/sqlite-amalgamation-$SQLITE_VERSION"

# SQLITE_OMIT_LOAD_EXTENSION matters for more than size: it removes the ability
# to load a shared library at runtime, which would be a way to execute code the
# runtime never verified.
"$TOOLCHAIN/bin/$TARGET_CC" -Os -fPIE -pie $NATIVE_DEBUG_CFLAGS \
    -DSQLITE_ENABLE_FTS5 -DSQLITE_ENABLE_JSON1 -DSQLITE_ENABLE_RTREE \
    -DSQLITE_THREADSAFE=1 -DSQLITE_OMIT_LOAD_EXTENSION \
    -o sqlite3.bin shell.c sqlite3.c -lm

echo
echo "Installed:"
install_payload "$BUILD_ROOT/sqlite-amalgamation-$SQLITE_VERSION/sqlite3.bin" "libpocketclaw-sqlite3.so"
report_catalog_reminder
