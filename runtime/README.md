# Managed Runtime payload builds

Scripts here cross-build the executables the PocketClaw Managed Runtime ships
inside the APK. They are release-engineering tools, not part of the app build.

See [`../RUNTIME.md`](../RUNTIME.md) for the runtime architecture and
[`../THIRD_PARTY_NOTICES.md`](../THIRD_PARTY_NOTICES.md) for payload licensing.

## Why payloads are packaged as `lib*.so`

PocketClaw targets Android SDK 36. Since API 29 an app may not `execve()` a file
in its own writable data directory, whatever the file's mode bits say. The only
directory the app may execute from is `nativeLibraryDir`, and the package
manager fills that directory by unpacking `lib/<abi>/*.so` entries from the APK
at install time.

So a payload must be named `lib<something>.so` or the installer never unpacks it,
and it can never run. `libpocketclaw-jq.so` is an ARM64 PIE executable; the name
is a packaging requirement, not a claim about the file format. The agent still
sees the logical name `jq`.

## Building

```sh
runtime/build-curl-android-arm64.sh      # run first: git links its libcurl
runtime/build-git-android-arm64.sh
runtime/build-gh-android-arm64.sh
runtime/build-ripgrep-android-arm64.sh
runtime/build-sqlite3-android-arm64.sh
runtime/build-jq-android-arm64.sh
```

`build-curl` also builds the mbedTLS and libcurl that `git-remote-http` links
against, so it must run before `build-git`. The others are independent.

Each script pins its upstream release tarball by SHA-256, builds in a fixed
neutral directory so no developer-machine path is baked into the binary, refuses
to install a payload that carries one, and prints the payload's checksum.

## After building

1. Record the printed checksum in `core/src/pkg/pcruntime/manifest.json`.
2. Run `go test ./pkg/pcruntime/` in `core/src`.
   `TestBundledPayloadsMatchTheirPinnedChecksums` fails while the catalog and the
   packaged payload disagree — that drift would otherwise reach a device as a
   `checksum_mismatch` that looks like a corrupt install.
3. If the payload is new, add it to both `requiredArm64NativeLibraries` and
   `keepDebugSymbols` in `android/app/build.gradle.kts`. Without the second,
   Gradle strips the executable during packaging, which changes its bytes and
   breaks the pinned checksum on every device.

## Helper payloads

git needs a second executable, `git-remote-https`, which Android cannot package
under that name. It ships as `libpocketclaw-git-remote-http.so` and the catalog
declares it as a *helper* with its logical name; the runtime presents it through
a symlink directory at execution time. See `RUNTIME.md`, "Helper payloads".

A helper's checksum goes in the catalog's `helpers` array, not just the tool's
own `sha256`. `TestBundledPayloadsMatchTheirPinnedChecksums` checks both.

## Build-path privacy

`install_payload` refuses to install a binary containing this build machine's
home directory, or a developer root at a path boundary. Two toolchains needed
help to pass it:

- **Rust** bakes source paths into panic-location strings that stripping cannot
  remove. The ripgrep script sets a neutral `CARGO_HOME` and
  `--remap-path-prefix`.
- **autotools** records its own configure arguments in the compiled result, which
  is why every build runs in a fixed neutral directory rather than wherever the
  developer happens to be.

The check is deliberately narrower than a bare `/home|/Users|/root` search: Go
records trimmed module paths such as `pkg/root/trusted_root.go`, and upstream
sources carry their own CI constants. Neither identifies this machine, and
failing on them would mean either disabling the check or patching upstream for
nothing.

## Toolchain

Android NDK 28.2.13676358, overridable with `NDK_ROOT`. ripgrep additionally
needs a Rust toolchain with the `aarch64-linux-android` target; gh needs Go.
