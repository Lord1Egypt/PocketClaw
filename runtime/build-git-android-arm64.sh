#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime git payloads for Android ARM64.
#
# Requires build-curl-android-arm64.sh to have run first: git-remote-http links
# against the libcurl and mbedTLS this leaves in $DEPS_PREFIX.
#
# Two ELF payloads ship. Nearly every git command is a builtin of the git binary
# itself; the remaining helpers upstream ships are Perl and shell scripts, which
# NO_PERL excludes and PocketClaw does not need.
#
#   git              -> libpocketclaw-git.so
#   git-remote-http  -> libpocketclaw-git-remote-http.so
#
# git looks its transport helper up as "git-remote-https" inside GIT_EXEC_PATH.
# nativeLibraryDir cannot hold a file under that name, so the runtime builds a
# symlink directory at execution time and points GIT_EXEC_PATH at it. See
# RUNTIME.md, "Helper payloads".
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

GIT_VERSION="2.51.0"
GIT_TARBALL="git-$GIT_VERSION.tar.gz"
GIT_URL="https://github.com/git/git/archive/refs/tags/v$GIT_VERSION.tar.gz"
GIT_SHA256="3524fc5fd81f16f80e1696a8281bd8ad831048b67848015d7b7382bf365ae685"

[ -f "$DEPS_PREFIX/lib/libcurl.a" ] || {
    echo "error: libcurl not found in $DEPS_PREFIX." >&2
    echo "       Run runtime/build-curl-android-arm64.sh first." >&2
    exit 1
}

fetch_pinned "$GIT_URL" "$GIT_TARBALL" "$GIT_SHA256"

echo "Building git $GIT_VERSION"
rm -rf "$BUILD_ROOT/git-$GIT_VERSION"
tar xzf "$CACHE_DIR/$GIT_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/git-$GIT_VERSION"

# Bionic implements no pthread cancellation and declares none of its symbols.
# The shim is a no-op that reports success, which is the correct behaviour on a
# platform where no thread can be cancelled. See the header for the reasoning.
cp "$REPO_ROOT/runtime/patches/git-android-pthread-cancel.h" \
   compat/pocketclaw-android-pthread-cancel.h

export PATH="$TOOLCHAIN/bin:$PATH"

# Android-specific build settings, each for a concrete bionic difference:
#
#   HAVE_SYNC_FILE_RANGE=   bionic has no sync_file_range().
#   CSPRNG_METHOD=arc4random
#                           bionic provides arc4random_buf() natively; its
#                           getrandom() only appears at API 28 and needs a
#                           header git does not include on this path.
#   PTHREAD_LIBS=           pthreads live inside bionic's libc; -lpthread does
#                           not exist and the link fails if it is passed.
#   NO_OPENSSL=1            git needs OpenSSL only for imap-send and SHA-1;
#                           its own SHA-1 is used instead and HTTPS comes
#                           entirely from libcurl.
#   NO_EXPAT=1              expat is only needed for the obsolete dumb-HTTP DAV
#                           push path. Smart HTTP push does not use it.
#   NO_PERL/NO_PYTHON/NO_TCLTK
#                           no interpreter ships with PocketClaw, so the
#                           script-based commands could not run anyway.
make -j"$(nproc)" \
    CC="$TARGET_CC" AR="llvm-ar" \
    CFLAGS="-Os -fPIE -include compat/pocketclaw-android-pthread-cancel.h" \
    LDFLAGS="-pie -L$DEPS_PREFIX/lib" \
    CURLDIR="$DEPS_PREFIX" \
    CURL_LDFLAGS="-lcurl -lmbedtls -lmbedx509 -lmbedcrypto -lz" \
    PTHREAD_LIBS= PTHREAD_CFLAGS= \
    uname_S=Linux \
    HAVE_SYNC_FILE_RANGE= CSPRNG_METHOD=arc4random \
    NO_OPENSSL=1 NO_EXPAT=1 NO_GETTEXT=1 NO_ICONV=1 \
    NO_PERL=1 NO_PYTHON=1 NO_TCLTK=1 NO_INSTALL_HARDLINKS=1 \
    prefix=/pocketclaw-runtime/git \
    git git-remote-http >/dev/null

echo
echo "Installed:"
install_payload "$BUILD_ROOT/git-$GIT_VERSION/git" "libpocketclaw-git.so"
install_payload "$BUILD_ROOT/git-$GIT_VERSION/git-remote-http" "libpocketclaw-git-remote-http.so"
report_catalog_reminder
