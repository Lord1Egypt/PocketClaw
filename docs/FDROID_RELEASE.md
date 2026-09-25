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

**Nothing replaced it.** No tracking SDK was substituted: PocketClaw does not
need one to work, and swapping one for another would have missed the point.

**Umeng followed in F-Droid Phase B (2026-09-24).** It had stayed as a
`compileOnly` dependency, absent from the APK but still downloaded and compiled
against — which F-Droid's inclusion policy forbids as surely as packaging it.
The SDK, `AnalyticsReporter.kt`, the device-feedback feature and its manifest
metadata were removed outright, and `tool/dependency_graph.py` now fails the
gate if any proprietary SDK group appears in the resolved Gradle graph.

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

## 1b. Fonts — BUNDLED, no runtime fetching

**Closed.** `google_fonts` is gone from `pubspec.yaml` and from the lockfile.
Inter and Fira Code are now bundled application assets and resolve entirely
offline.

| Family | Weights bundled | Upstream | Licence |
| --- | --- | --- | --- |
| Inter | 400, 500, 600, 700, 800, 900 | [rsms/inter](https://github.com/rsms/inter) `v4.1` | OFL-1.1 |
| Fira Code | 400, 600 | [tonsky/FiraCode](https://github.com/tonsky/FiraCode) `6.2` | OFL-1.1 |

Per-file digests and the source archive hashes are recorded in
[`assets/fonts/README.md`](../assets/fonts/README.md). Both OFL texts are
committed **and packaged into the APK** — OFL-1.1 asks the licence to travel
with the font software, and an APK is a redistribution.

`lib/src/core/app_fonts.dart` replaced the `GoogleFonts.*` call sites with
`TextStyle(fontFamily: ...)` against the declared families, so there is no fetch
path left to disable rather than a fetch path that happens to be turned off.
`test/unit/bundled_fonts_contract_test.dart` holds the line: no
`fonts.googleapis.com` or `fonts.gstatic.com` anywhere in `lib/`, no
`google_fonts` in the lockfile, every declared asset present and a real
TrueType file, every weight the code asks for actually bundled, and the OFL
texts both committed and declared as packaged assets.

## 1c. Managed Runtime — source build demonstrated end to end

All eight tools were **rebuilt from source on a tree with the committed
payloads physically removed**, so nothing could be silently reused.

### Demonstrated ordering

```
source checkout
  → quarantine the eight committed lib*.so out of jniLibs/
  → confirm only the Core pair remains
  → run runtime/build-<tool>-android-arm64.sh for each
  → verify arch / format / stripped / size / SHA-256
  → package into the canonical APK
  → verify the artifact
```

### Toolchain used

NDK 28.2.13676358 (API 24 sysroot), Rust 1.94.1, the repository Go toolchain,
Python 3.13 host interpreter, autoconf 2.71, make 4.3, cmake 3.28.3. Sources are
pinned upstream tarballs, checksum-verified by each recipe.

### Result — all eight built; six byte-identical

| Tool | Version | Built | vs committed |
| --- | --- | --- | --- |
| `curl` | 8.11.1 | yes | **identical** |
| `git` | 2.51.0 | yes | **identical** |
| `git-remote-http` | 2.51.0 | yes | **identical** |
| `jq` | 1.7.1 | yes | **identical** |
| `rg` | 14.1.1 | yes | **identical** |
| `sqlite3` | 3.50.4 | yes | **identical** |
| `gh` | 2.82.1 | yes, after a recipe fix | differed — adopted in H1.5D |
| `python` | 3.14.7 | yes, after a determinism fix | differed — adopted in H1.5D |

Six rebuilding **bit-for-bit** from files that had been deleted first is the
strongest available evidence that they come from source and that the recipes are
deterministic.

### Two defects this exercise found

**`gh` could no longer be built at all.** Its recipe copies
`core/src/pkg/androiddns/resolver.go` into the gh tree so gh resolves DNS on
Android, and guards that the copied file imports only the standard library. The
Zero-Pico work gave that file a `pkg/canonicalenv` import, so the guard fired
and the build stopped — correctly. It means the *committed* gh binary cannot be
reproduced from current source. Fixed by vendoring `canonicalenv` alongside the
resolver, since it is itself a std-lib-only leaf; the guard now allows that one
import and nothing else. The rebuilt gh therefore differs from the committed one
because its input genuinely changed.

**`python` was not reproducible.** The appended stdlib zip embedded wall-clock
timestamps, so every build produced a different binary. Fixed in
`runtime/python-lite-stdlib.py`: entries are written with a fixed timestamp from
`SOURCE_DATE_EPOCH` (falling back to the zip epoch), fixed permissions and a
sorted walk. **Proven** — two consecutive builds with the epoch pinned produced
byte-identical output, `96b34067…`.

### Adopted — H1.5D, 2026-09-10

The owner took the two corrected payloads. Both were rebuilt from the current
recipes and installed; the other six were not rebuilt and not touched, which is
what makes "only these two changed" a `git diff` fact rather than a claim.

| Payload | Was | Is |
| --- | --- | --- |
| `libpocketclaw-gh.so` | `3f56431f…` | `fe97fb29…` |
| `libpocketclaw-python.so` | `dfa19e41…` | `4f98d0e3…` |

Two consecutive gh builds from the current recipe produced identical bytes, so
the adopted gh is reproducible as well as current.

`manifest.json` is a canonical Core build input, so the two checksum edits moved
the Core source fingerprint `6e2382ae…` → `876b87f5…` and both Core binaries
were rebuilt and restaged. That is not bookkeeping: Core embeds the catalog it
verifies payloads against, so a Core built before the edit would have rejected
both adopted payloads at resolve time as corrupt.

The chain is verified end to end for each of the two — source-built output, the
committed payload, the manifest checksum and the payload unpacked from the APK
are one value.

The accepted physical baseline stays **vc62**. The H1.5D APK is a local test
build, was not installed, and does not supersede it.

### Remaining F-Droid question

Not *can these be built from source* — that is now demonstrated — but whether
F-Droid's build server will run recipes needing the Android NDK, Rust and Go.
That is a conversation to have before submission. The payloads also remain
committed, which a source-building distribution will likely challenge.

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
| Pinned toolchain | Met in practice: Flutter 3.47.1, Dart 3.13.1, JDK 17, Go 1.26.8 (Core; gh builds with go1.27.1), AGP 8.11.1, Gradle 8.14. To be declared explicitly in metadata. |
| Deterministic timestamp | Met — `core/resolve-build-time.sh` derives it from the last commit touching a build input, never the wall clock; an explicit `POCKETCLAW_BUILD_EPOCH` overrides it, and `SOURCE_DATE_EPOCH` applies only without usable git history (an inherited one, such as fdroidserver's, would otherwise stamp the checked-out commit's time). |
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
| Dart obfuscation | H3A adopted it after two clean equivalent builds produced byte-identical `libapp.so`. This is scoped Dart evidence; full-APK proof remains open. |
| `--split-debug-info` | H3A produced byte-identical private DWARF from two different output roots and verified it stayed outside the APK/Git. The two debug-signed APK hashes differed, so this does not claim the full APK is unchanged or reproducible. |
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
./core/build-android-arm64.sh   # the epoch comes from the Core build-input commit
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
| Firebase / Google Play Services? | **Removed**, verified absent from DEX, manifest and packaged entries. |
| Runtime font fetching? | **Closed.** Inter and Fira Code are bundled; `google_fonts` is gone from the lockfile. |
| Managed Runtime buildable from source? | **Demonstrated for all eight**, with the committed payloads quarantined first. Six rebuild byte-identical; the other two were corrected and adopted in H1.5D. |
| Are the recipes deterministic? | Six proven identical; `python` made reproducible and proven over two runs; `gh` proven over two runs at adoption. |
| Does it download executables after install? | **No — structurally impossible** on targetSdk 36 (§1d). |
| Anti-features to declare? | `NonFreeNet` for optional proprietary providers. Not `Tracking`, not `NonFreeDep`. |
| Outstanding before submission | Whether F-Droid's builders will run NDK/Rust/Go recipes; committed prebuilts; full-APK reproducibility. |
| Can direct and F-Droid share a signature? | Yes, if APK reproducibility holds. That is the design target. |
| Can Play share it? | No, under Play App Signing. Accepted. |
