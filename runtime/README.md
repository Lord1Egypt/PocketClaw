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
runtime/build-jq-android-arm64.sh
```

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

## Toolchain

Android NDK 28.2.13676358, overridable with `NDK_ROOT`.
