# F-Droid release readiness

Official F-Droid distribution is a confirmed requirement alongside direct APK
and Google Play. This is the audit of what stands between PocketClaw and an
F-Droid submission, and what must be true before one is made.

> **Nothing has been submitted.** No `fdroiddata` metadata exists yet. This
> document is the preparation, not the application.

---

## 1. The blocker, found by audit

**PocketClaw cannot be accepted into official F-Droid as it is built today.**

The shipped `0.2.0+62` APK contains Google Play Services and Firebase. Verified
by inspecting the artifact, not by reading the dependency list:

```
firebase-analytics.properties
play-services-ads-identifier.properties        (+ 10 further GMS entries)
com.google.android.gms.ads.APPLICATION_ID
com.google.android.gms.measurement.AppMeasurementService
com.google.firebase.analytics.connector.internal.AnalyticsConnectorRegistrar
```

These come from `firebase_analytics` and `firebase_core` in `pubspec.yaml`.
F-Droid's inclusion policy requires that an app build from free source with free
dependencies; Google Play Services and Firebase are proprietary binaries, and
analytics plus an advertising identifier additionally attract the *Tracking* and
*NonFreeDep* anti-features.

**This is a dependency problem, not a behaviour problem.** Firebase is already
disabled at runtime unless `POCKETCLAW_FIREBASE_APP_ID` is supplied at build
time — but F-Droid judges what is *in the binary*, and the AAR is packaged
unconditionally because a Flutter plugin listed in `pubspec.yaml` is contributed
to the Android build by the Flutter Gradle plugin whether or not any Dart code
calls it.

### Recommended resolution — and why it is not a flavor

The instinctive fix is an `fdroid` product flavor. **That is not the
recommendation.** A flavor would mean two build configurations, two things to
verify, and a reproducibility story that has to be told twice.

The better fix already exists in this repository as a precedent. Umeng
analytics was once an unconditional dependency and is now `compileOnly` by
default, promoted to a real `implementation` only when
`-PanalyticsProvider=umeng` asks for it, so the default build carries neither
its classes nor its manifest contributions. The same shape applies to Firebase,
with one extra step: because it arrives as a Flutter plugin rather than a Gradle
coordinate, it must first come *out* of `pubspec.yaml` and be reached through a
narrow platform interface instead — the reporter in
`lib/src/core/firebase_device_reporter.dart` is already the only caller, which
makes that boundary small.

The outcome is **one canonical source and build configuration** that serves all
three channels, with the proprietary path absent by default and requested
explicitly for the Play build if it is wanted there at all.

That work is a hardening phase of its own. It is not H1, and it must not be done
by half.

### Other dependencies reviewed

| Dependency | Status |
| --- | --- |
| `firebase_analytics`, `firebase_core` | **Blocker** — proprietary, packaged unconditionally. |
| Umeng analytics | Clear. `compileOnly` by default; absent from the default APK (verified: 0 entries). |
| `google_fonts` | Review. Fetches fonts over the network at runtime unless bundled; F-Droid dislikes runtime downloads of non-packaged assets. |
| Managed Runtime payloads (`python`, `git`, `gh`, `curl`, `rg`, `jq`, `sqlite3`) | Review, and likely the second real discussion. They are prebuilt binaries committed to the repository. F-Droid requires building from source, so either they are built in the F-Droid pipeline or their presence must be declared and justified. |
| Vendored Core (`core/src`, Go) | Clear. Source is in-repo, MIT, and built by `core/build-android-arm64.sh`. |
| Everything else in `pubspec.yaml` | No known proprietary component. |

> The Managed Runtime is the item most likely to need a conversation with
> F-Droid rather than a code change. Committed prebuilt binaries are exactly what
> a source-building distribution exists to avoid, even when their sources are
> free.

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
| Is a separate F-Droid flavor necessary? | **No** — and it should be avoided. Make the proprietary dependency optional instead, following the Umeng precedent. |
| Is there a concrete blocker? | **Yes** — Firebase and Google Play Services are packaged unconditionally. |
| Is the build reproducible today? | The Core and frontend, yes and proven. The full APK, not yet verified. |
| Can direct and F-Droid share a signature? | Yes, if reproducibility holds. That is the design target. |
| Can Play share it? | No, under Play App Signing. Accepted. |
