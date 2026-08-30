#!/usr/bin/env bash
#
# Builds the PocketClaw Managed Runtime curl payload, and the mbedTLS + libcurl
# stack that git-remote-http also links against.
#
# mbedTLS rather than OpenSSL: the whole TLS stack costs about 1.3 MB in the
# finished curl binary this way, against several megabytes for OpenSSL, and
# zlib comes from the Android system image. Run this before build-git.
#
source "$(dirname "${BASH_SOURCE[0]}")/android-build-env.sh"

MBEDTLS_VERSION="3.6.4"
MBEDTLS_TARBALL="mbedtls-$MBEDTLS_VERSION.tar.bz2"
MBEDTLS_URL="https://github.com/Mbed-TLS/mbedtls/releases/download/mbedtls-$MBEDTLS_VERSION/$MBEDTLS_TARBALL"
MBEDTLS_SHA256="ec35b18a6c593cf98c3e30db8b98ff93e8940a8c4e690e66b41dfc011d678110"

CURL_VERSION="8.11.1"
CURL_TARBALL="curl-$CURL_VERSION.tar.xz"
CURL_URL="https://github.com/curl/curl/releases/download/curl-${CURL_VERSION//./_}/$CURL_TARBALL"
CURL_SHA256="c7ca7db48b0909743eaef34250da02c19bc61d4f1dcedd6603f109409536ab56"

fetch_pinned "$MBEDTLS_URL" "$MBEDTLS_TARBALL" "$MBEDTLS_SHA256"
fetch_pinned "$CURL_URL" "$CURL_TARBALL" "$CURL_SHA256"

echo "Building mbedTLS $MBEDTLS_VERSION"
rm -rf "$BUILD_ROOT/mbedtls-$MBEDTLS_VERSION"
tar xjf "$CACHE_DIR/$MBEDTLS_TARBALL" -C "$BUILD_ROOT"
cmake -S "$BUILD_ROOT/mbedtls-$MBEDTLS_VERSION" -B "$BUILD_ROOT/mbedtls-$MBEDTLS_VERSION/build-android" \
    -DCMAKE_TOOLCHAIN_FILE="$NDK_ROOT/build/cmake/android.toolchain.cmake" \
    -DANDROID_ABI=arm64-v8a -DANDROID_PLATFORM="android-$ANDROID_API" \
    -DCMAKE_BUILD_TYPE=MinSizeRel -DCMAKE_INSTALL_PREFIX="$DEPS_PREFIX" \
    -DENABLE_TESTING=OFF -DENABLE_PROGRAMS=OFF \
    -DUSE_SHARED_MBEDTLS_LIBRARY=OFF -DUSE_STATIC_MBEDTLS_LIBRARY=ON >/dev/null
cmake --build "$BUILD_ROOT/mbedtls-$MBEDTLS_VERSION/build-android" -j"$(nproc)" >/dev/null
cmake --install "$BUILD_ROOT/mbedtls-$MBEDTLS_VERSION/build-android" >/dev/null

echo "Building curl $CURL_VERSION"
rm -rf "$BUILD_ROOT/curl-$CURL_VERSION"
tar xJf "$CACHE_DIR/$CURL_TARBALL" -C "$BUILD_ROOT"
cd "$BUILD_ROOT/curl-$CURL_VERSION"

export PATH="$TOOLCHAIN/bin:$PATH"
export CC="$TARGET_CC" AR="llvm-ar" RANLIB="llvm-ranlib"
export CFLAGS="-Os -fPIE" LDFLAGS="-pie"
export CPPFLAGS="-I$DEPS_PREFIX/include" LIBS="-L$DEPS_PREFIX/lib"

# Protocols PocketClaw's agent actually needs. Everything else is disabled to
# keep the payload small; a protocol can be enabled later if a real use appears.
./configure \
    --host=aarch64-linux-android --build=x86_64-pc-linux-gnu \
    --prefix="$DEPS_PREFIX" --with-mbedtls="$DEPS_PREFIX" \
    --enable-static --disable-shared \
    --with-zlib --without-libpsl --without-libidn2 \
    --without-brotli --without-zstd --without-nghttp2 \
    --disable-ldap --disable-ldaps --disable-rtsp --disable-dict --disable-telnet \
    --disable-tftp --disable-pop3 --disable-imap --disable-smb --disable-smtp \
    --disable-gopher --disable-mqtt --disable-manual --disable-ntlm \
    --with-ca-path=/system/etc/security/cacerts >/dev/null

make -j"$(nproc)" >/dev/null
make install >/dev/null

echo
echo "Installed:"
install_payload "$BUILD_ROOT/curl-$CURL_VERSION/src/curl" "libpocketclaw-curl.so"
report_catalog_reminder
