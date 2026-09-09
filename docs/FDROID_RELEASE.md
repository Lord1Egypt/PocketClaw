# F-Droid release readiness

Official F-Droid distribution is a confirmed requirement alongside direct APK
and Google Play. This is the audit of what stands between PocketClaw and an
F-Droid submission, and what must be true before one is made.

> **Nothing has been submitted.** No `fdroiddata` metadata exists yet. This
> document is the preparation, not the application.

---

## 1. Firebase and Google Play Services — REMOVED

H1 found Firebase Analytics and Google Play Services packaged unconditionally in
the shipping APK. **H1.5 removed them from the canonical build.**

They came from `firebase_analytics` and `firebase_core` in `pubspec.yaml`, plus
two `<meta-data>` entries in the app's own `AndroidManifest.xml` referencing
`@string/google_app_id`, and a Gradle task that generated those Firebase string
resources from dart-defines.

A contributing root cause was on the Dart side: `DeviceFeedbackProvider`
fell through to **Firebase for any unrecognised value**, so a build that simply
did not pass `POCKETCLAW_ANALYTICS_PROVIDER` selected an analytics provider by
accident. The default is now `none`.

### What was removed

| Removed | Why |
| --- | --- |
| `firebase_analytics`, `firebase_core` (pubspec) | Proprietary SDK; pulled in Google Play Services. |
| `lib/src/core/firebase_device_reporter.dart` | Its only purpose was uploading device reports to Firebase. |
| The `firebase` arm of `DeviceFeedbackProvider` | No SDK left to reach. |
| 5 `POCKETCLAW_FIREBASE_*` dart-defines | Configured a dependency that no longer exists. |
| `generateFirebaseResources` / `cleanupFirebaseResources` Gradle tasks | Generated `google_app_id` and friends. |
| Two GMS `<meta-data>` manifest entries | Referenced the generated resources. |

**Nothing replaced it.** Optional device feedback still exists with the Umeng
provider (`compileOnly` and absent by default) and `none`. No tracking SDK was
substituted: PocketClaw does not need one to work, and swapping one for another
would have missed the point.

### Verified in the artifact, not in the dependency list

A release-shaped local-test APK was rebuilt and inspected:

```
DEX class scan  com/google/firebase        0 classes
                com/google/android/gms     0 classes
packaged entries  firebase* 0   play-services* 0   gms* 0
                  measurement* 0   admob/ads-identifier* 0   umeng* 0
merged manifest   com.google.android.gms.* / com.google.firebase.*   none
```

The APK also shrank from 63,493,154 to 62,887,336 bytes. Package, ABI policy,
the Core pair, all eight Managed Runtime payloads and the permission set are
unchanged.

Gate checks now enforce this on every artifact run:
`artifact.fdroid_no_firebase`, `..._no_gms`, `..._no_admob`,
`..._no_measurement`, plus `source.fdroid_no_proprietary_sdk` on the source side
so a removed dependency cannot return via `pubspec.yaml`.

### No F-Droid flavor is necessary

There is now **one canonical build** for direct APK, Google Play and F-Droid.
The proprietary dependency was removed rather than hidden behind a flavor, a
build type or a scanignore entry.

## 1b. Remaining item — google_fonts

**Open, and the one thing standing between this build and an F-Droid
submission.**

`google_fonts` 8.2.1 is used across the UI (Inter, Fira Code) with **no bundled
font files**. Verified in the package source: `allowRuntimeFetching` defaults to
`true` and PocketClaw never sets it, so the fonts are fetched from Google's
servers at first use and cached on device.

That is a runtime download of non-packaged assets, a network dependency for
ordinary UI rendering, and a call to a Google server from an app that describes
itself as private-first.

**Remedy** — bundle the fonts, do not drop the design:

1. Add Inter and Fira Code (both SIL OFL 1.1, redistributable) as tracked
   assets, with their licence text.
2. Declare them in `pubspec.yaml` under `flutter: fonts:`.
3. Set `GoogleFonts.config.allowRuntimeFetching = false` at startup so a missing
   font is a loud failure rather than a silent network call.

Not done in H1.5 because fetching the font files needs network access this phase
did not have, and doing half of it — disabling fetching without bundling — would
degrade the UI to fallback fonts. It is a small, bounded task with a known
shape.

## 1c. Managed Runtime — provenance audit

All eight tools have **in-repo, from-source cross-compilation recipes** with
checksum verification. This is a considerably stronger position than H1 assumed.

| Tool | Version | Licence | Upstream source | Recipe | Class |
| --- | --- | --- | --- | --- | --- |
| `python` | 3.14.7 | PSF-2.0 | python.org/ftp | `runtime/build-python-android-arm64.sh` | A |
| `git` | 2.51.0 | GPL-2.0-only | github.com/git/git | `runtime/build-git-android-arm64.sh` | A |
| `git-remote-http` | (with git) | GPL-2.0-only | github.com/git/git | same recipe | A |
| `gh` | 2.82.1 | MIT | github.com/cli/cli | `runtime/build-gh-android-arm64.sh` | A |
| `curl` | 8.11.1 | curl (MIT-like) | github.com/curl/curl | `runtime/build-curl-android-arm64.sh` (+ mbedTLS) | A |
| `rg` | 14.1.1 | MIT OR Unlicense | github.com/BurntSushi/ripgrep | `runtime/build-ripgrep-android-arm64.sh` | A |
| `jq` | 1.7.1 | MIT | github.com/jqlang/jq | `runtime/build-jq-android-arm64.sh` | A |
| `sqlite3` | 3.50.4 | Public domain | sqlite.org | `runtime/build-sqlite3-android-arm64.sh` | A |

**Classification: all eight are class A — reproducibly source-buildable now**,
from checksum-verified upstream tarballs, by scripts already in this repository.

The open question is not *can they be built from source* but **whether F-Droid's
build server will run those recipes**, since they need the Android NDK, plus Rust
for ripgrep and Go for gh. That is a conversation with F-Droid about build
requirements, not a code change, and it should happen before submission rather
than during review.

The binaries are currently **committed** to the repository as
`android/app/src/main/jniLibs/arm64-v8a/libpocketclaw-*.so`. Committed prebuilts
are exactly what a source-building distribution exists to avoid, even when the
sources are free and the recipes are present. Expect to be asked to build them
in-pipeline.

## 1d. Post-install executable downloads — none, structurally

**PocketClaw cannot download and execute a binary.** This is not a policy it
follows; it is a thing the platform will not permit, and the runtime documents
it as the reason no download path exists to audit:

> Since API 29 an app may not `execve()` a file in its own writable data
> directory, and `File.setExecutable(true)` does not change that: the
> restriction is enforced on the app's SELinux domain, not by the file mode.

The only two delivery routes are therefore:

- `system` — binaries the OS itself ships in `/system/bin` (46 catalog entries).
- `bundled` — `lib*.so` in the APK, unpacked by the package manager into
  `nativeLibraryDir` at install time (the 7 manifest entries covering the 8
  shipped tools).

`EnsureTool` reports capability-unavailable for anything outside the catalog
rather than acquiring it. There is no fetch, no checksum-on-download step to
review, and no update channel for executables.

## 1e. Providers and Anti-Features

PocketClaw connects to whatever AI provider the user configures. Declared
honestly rather than concealed:

| Service | Status |
| --- | --- |
| Anthropic, OpenAI-compatible, Azure OpenAI, AWS Bedrock | **Optional, user-selected.** Proprietary network services. |
| OpenAI-compatible endpoints generally | Covers self-hosted and FLOSS servers — this is the escape hatch that keeps the app usable without any proprietary service. |
| Telegram, GitHub | Optional integrations the user turns on. |
| Managed-bot onboarding service | Optional convenience for Telegram setup; manual token entry is the alternative. |

**No provider is required for the app to function**, and none is promoted as a
default that must be accepted. The likely F-Droid anti-feature declaration is
`NonFreeNet` (optional use of proprietary network services). `Tracking` and
`NonFreeDep` no longer apply after the Firebase removal, and `NonFreeAdd` does
not apply.

## 2. Package identity

```
com.lord1egypt.pocketclaw
```

Fixed, and shared across all three channels. It never varies by distribution.

## 3. Build model F-Droid expects

F-Droid builds from source in its own environment and compares. That imposes
requirements PocketClaw partly already meets:

| Requirement | Status |
| --- | --- |
| Source-available build | Met — the app, the Core and the frontend all build from this tree. |
| Pinned toolchain | Met in practice: Flutter 3.47.1, Dart 3.13.1, JDK 17, Go 1.25.11, AGP 8.11.1, Gradle 8.14. To be declared explicitly in metadata. |
| Deterministic timestamp | Met — `core/resolve-build-time.sh` derives it from the last commit touching a build input, never the wall clock, and honours `SOURCE_DATE_EPOCH`. |
| Reproducible native binaries | Met and proven — both Core binaries rebuild byte-identical from the same source and epoch. |
| Deterministic frontend bundle | Met — verified identical across two independent builds. |
| Version extraction | `pubspec.yaml` is the tracked source; the Gradle build refuses a version in `local.properties`. |
| ABI policy | `arm64-v8a` only for the product payload. Declare this; it is deliberate, not an oversight. |
| No committed prebuilt binaries | **Not met** — the staged Core and Managed Runtime payloads are committed. See §1. |

## 4. Reproducibility is the whole strategy

Signing continuity between the direct APK and F-Droid depends on F-Droid
publishing the **developer-signed** artifact, which it will only do for a build
it can reproduce bit-for-bit. So reproducibility is not a quality goal here; it
is the thing that decides whether a user can move between channels without
uninstalling.

### Hardening must not destroy it

Every remaining hardening step is a candidate for introducing nondeterminism,
and each must be assessed before it lands:

| Step | Reproducibility risk |
| --- | --- |
| R8 / ProGuard | Generally deterministic for fixed inputs and rules, but the mapping file is an output that must not feed back into inputs. Verify. |
| Resource shrinking | Usually deterministic. Verify alongside R8. |
| Dart obfuscation | **Highest risk.** Symbol renaming must be deterministic for identical inputs; confirm empirically before adopting. |
| `--split-debug-info` | Produces a separate symbols artifact. Keep it out of the APK and out of the repository; confirm the APK itself is unchanged by its presence. |
| Native stripping | Already deterministic here — `-trimpath`, `-buildvcs=false`, `-s -w`, and a fixed BuildTime. Do not regress it. |
| AAB generation | Bundletool output has historically been less reproducible than APK output. Play uses AAB; F-Droid uses APK. Do not let the AAB path dictate the APK path. |

**This is a release-gate objective:** any hardening change must be shown not to
break byte-identical rebuilds, by the same bounded two-build proof already used
for the Core.

## 5. Signing

The developer app-signing key is described in
[`RELEASE_SIGNING.md`](RELEASE_SIGNING.md) §8. For F-Droid specifically:

- The target is a **developer-signed, reproducible** build, so the F-Droid APK
  carries the same certificate as the direct APK.
- The certificate fingerprint enrolled in
  `android/release-signing-cert.sha256` is the same value that would be
  configured as `AllowedAPKSigningKeys` in F-Droid metadata.
- The fallback, if reproducibility fails, is an F-Droid-signed build under a
  different certificate — not mutually updatable with the direct APK. Treat that
  as a last resort.

## 6. Verifying reproducibility locally

The Core half is already proven and can be re-run:

```bash
SOURCE_DATE_EPOCH=$(./core/resolve-build-time.sh --print-epoch) \
  ./core/build-android-arm64.sh
# then compare sha256 of both staged binaries against the previous build
```

The APK half needs the same treatment once hardening is in place: two builds
from identical source and epoch, compared entry by entry, with any differing
entry explained rather than tolerated.

## 7. Still to be created

None of this exists yet and none of it should be created before §1 is resolved:

- `fdroiddata` metadata (`metadata/com.lord1egypt.pocketclaw.yml`) — build
  recipe, versions, `AllowedAPKSigningKeys`, anti-feature declarations.
- A disclosure of the network services the app contacts: model providers
  (Anthropic, OpenAI-compatible, Azure, Bedrock), Telegram, GitHub, and the
  managed-onboarding service. F-Droid expects users to be told what an app talks
  to, and PocketClaw talks to whatever the user configures.
- A statement on the Managed Runtime payloads (§1).

## 8. Summary

| Question | Answer |
| --- | --- |
| Is a separate F-Droid flavor necessary? | **No.** The proprietary dependency was removed, not hidden. One canonical build serves all three channels. |
| Firebase / Google Play Services? | **Removed and verified absent** from the DEX, the manifest and the packaged entries. |
| Remaining blocker? | **`google_fonts` runtime fetching** (§1b). Small, bounded, needs network to bundle the font files. |
| Managed Runtime buildable from source? | **Yes, all eight**, by recipes already in `runtime/`. Open question is whether F-Droid's builders will run them. |
| Does it download executables after install? | **No — structurally impossible** on targetSdk 36 (§1d). |
| Anti-features to declare? | `NonFreeNet` for optional proprietary providers. Not `Tracking`, not `NonFreeDep`. |
| Is the build reproducible today? | Core and frontend, yes and proven. Full APK, not yet verified. |
| Can direct and F-Droid share a signature? | Yes, if APK reproducibility holds. That is the design target. |
| Can Play share it? | No, under Play App Signing. Accepted. |
