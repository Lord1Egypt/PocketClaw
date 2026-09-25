# F-Droid readiness notes

Where PocketClaw stands against official F-Droid inclusion, measured against the
**current** policy documents (read 2026-09-24): the Inclusion Policy, the
Anti-Features list, the Build Metadata Reference, the fdroiddata "App inclusion"
merge-request template and the reviewers' wiki. The standing background analysis
is [`FDROID_RELEASE.md`](FDROID_RELEASE.md).

> **Golden #3 (2026-09-25):** PocketClaw v0.2.2, source `80c9dc0`, owner-signed
> APK `320368ea…`, minSdk 26, arm64-v8a — published as the GitHub release
> `v0.2.2` (tag at `e535fca`). F-Droid
> will build and sign its own APK from the tagged source in Phase C.
>
> **Nothing has been submitted.** No fdroiddata metadata exists, no merge
> request is open, and no release was made for F-Droid. Phase A audited;
> Phase B (branch `feature/fdroid-phase-b`) fixed the source. Phase C — the
> F-Droid-shaped build under `fdroid build`, the scanners and the metadata
> draft — has not started.

## Phase A findings and their Phase B state

| # | Finding (Phase A) | Phase B |
| --- | --- | --- |
| B1 | Umeng resolved in `releaseCompileClasspath` (compileOnly). The policy forbids proprietary analytics in the build, not only in the APK. | **Fixed.** Removed outright with the device-feedback feature; `tool/dependency_graph.py` fails the gate on any proprietary SDK group in the resolved graph. |
| B2 | `kagi-openapi-golang`, which has no licence, was linked into Core. | **Fixed.** Replaced by a standard-library request with the same wire contract; gone from `go.mod`/`go.sum` and from both Core binaries. |
| B3 | No unsigned release path. | **Fixed.** `-PpocketclawUnsignedRelease=true` / `build_hardened_android.py --signing unsigned`, refusing any signing material, verified unsigned, classified UNSIGNED / REPOSITORY-SIGNABLE. Gate class `repository`. |
| B4 | The APK advertised armeabi-v7a and x86_64 through plugin stubs. | **Fixed.** `abiFilters` arm64-v8a; the gate fails on any other ABI. |
| B5 | MANAGE_EXTERNAL_STORAGE, requested by a Settings redirect on every cold launch. | **Fixed; physically tested 2026-09-25** (PC-DEF-077: detector and import pass; cancel/hide not physically exercised). No storage permission; app-specific workspace; explicit SAF import of an old `Download/pocketclaw`, offered only when that folder holds real workspace evidence (an empty folder, like the one Samsung My Files created on the owner's phone, is ignored). Detection only stats paths. |
| B6 | Payload toolchains not pinned (ripgrep's Rust in particular), while Core rejects any payload whose bytes change. | **Fixed.** `runtime/toolchains.env`; recipes select exact versions and fail closed. gh and ripgrep rebuild byte-identical under the pins. |
| B7 | The proprietary-SDK gate checked Firebase names only. | **Fixed.** Resolved-graph check plus Umeng in the source and DEX markers. |

Also done in Phase B: the upstream self-updater is not linked into the Android
Core; the Remix Icon font (non-FLOSS licence) was replaced with Material Icons;
`LICENSE` is standard MIT again with attribution in `THIRD_PARTY_NOTICES.md`;
the Gradle wrapper pins its distribution checksum; `DebugProbesKt.bin` is no
longer packaged; the Firebase debug-manifest residue is gone; stray CJK
developer comments were removed and `tool/cjk_hygiene.py` keeps them out;
PC-DEF-084 is fixed in source; and upstream Fastlane metadata exists
(`fastlane/metadata/android/en-US`) — without screenshots, which need a device
(see *Fastlane metadata and screenshots* below).

## Components that can start PocketClaw (merged release manifest)

Read from the release merged manifest, 2026-09-25; held by
`test/unit/android_lifecycle_contract_test.dart`.

| Component | Origin | Exported | What starts it |
| --- | --- | --- | --- |
| `MainActivity` | app | yes (MAIN/LAUNCHER only) | the owner tapping the icon |
| `service.PocketClawService` | app | no | the app itself, `specialUse` foreground service; START_NOT_STICKY, stops on an OS re-creation |
| `androidx.profileinstaller.ProfileInstallReceiver` | AndroidX | yes, `android.permission.DUMP` | shell/system tooling only (profile install) |
| `share.SharePlusPendingIntent` | share_plus | no | the system share sheet's result, after the owner shares |
| `share.ShareFileProvider` | share_plus | no | a share the owner started |
| `urllauncher.WebViewActivity` | url_launcher | no | the app opening a link in-app |
| `androidx.startup.InitializationProvider` (ProcessLifecycle, ProfileInstaller initializers) | AndroidX | no | runs inside an already-starting process; starts nothing |

No boot, locked-boot, package-replaced or direct-boot component; no alarm,
job, WorkManager or sync adapter; `RECEIVE_BOOT_COMPLETED` is forbidden by
the release gate and background-scheduler libraries by the dependency gate.
PocketClaw does not start on boot, by design.

## Fastlane metadata and screenshots

`fastlane/metadata/android/en-US` has the title, short and full descriptions,
the 512 px icon and `changelogs/64.txt` for the version being built.
`test/unit/fastlane_metadata_test.dart` holds the length limits, the scope
statements, the changelog for the current versionCode and the icon size.

**Four real screenshots are committed** (2026-09-25, build `b4011015…`,
English UI, demo-mode status bar): `1.png` Status, `2.png` Dashboard chat with
a Python tool call, `3.png` Settings, `4.png` What's New. Private screens (the
Telegram bot handle, GitHub account, logs, existing chats, the Models page's
masked key) were deliberately left out; the Models page also has a layout
defect (PC-DEF-089). The rules below still apply to any replacement.

F-Droid shows
`images/phoneScreenshots/*.png|jpg` in file-name order. They must be captured
by the owner from the installed app on the phone — no mockups, renders or
placeholders; the test refuses any file there that is not a portrait PNG or
JPEG at least 320 px wide, and an empty directory. Suggested set, 4–6 images:
Dashboard home, Chat, Models, Channels → Telegram (connected), Settings. Before
committing, check each one for API keys, bot tokens, passwords, chat content,
phone numbers and notification-shade content. `featureGraphic.png` (1024×500)
is optional and not planned.

## Fresh-install network capture (owed; must not touch the owner's install)

The owner's `com.lord1egypt.pocketclaw` install holds real data and must never
be uninstalled or cleared for this. Non-destructive options, safest first:

1. **`fdroid build` output in a disposable environment (Phase C).** The
   F-Droid-built APK is signed by F-Droid, so it cannot be installed over the
   owner's production-signed app anyway; run it on a second device or an arm64
   emulator image and capture there. This is the artifact reviewers will judge.
2. **A second Android user or profile on the same phone.** An app installed
   for another user or in Samsung Secure Folder gets its own empty data
   directory under the same package name; the owner's user-0 data is untouched.
   Requires the owner to create the user/profile; secondary users can be
   disabled on some Samsung builds.
3. **A side-by-side capture build** with a distinct `applicationId` (for example
   a `.capture` suffix on a local-test build): installs next to the owner's app
   with fresh data. The package name differs from the shipped one, so record
   the result as indicative, not as the reviewed artifact.

Capture from first launch, before any configuration: PCAPdroid (non-root VPN,
per-app filter) gives hostnames via DNS/SNI; the `/proc/net` UID sampling used
on 2026-09-25 is a fallback that can miss very short connections. Record
host/IP, trigger and whether it was user-initiated; redact keys, tokens and
message text. Expected result: no connection until a provider or channel is
configured.

## Where current policy and the ThothTerm precedent differ

- Prebuilt build tools from Go, Rust/Rustup and Node.js are explicitly allowed
  by the current Inclusion Policy. PocketClaw needs all three (gh's Go
  toolchain, ripgrep's Rust, the dashboard's pnpm tree with its esbuild,
  rolldown, lightningcss and Tailwind oxide binaries).
- Reproducible builds are requested, not required: the merge-request template
  says to enable them or give the reason. They decide whether F-Droid can
  publish the upstream signature, not whether the app is included.
- The reviewers' wiki checks explicitly for unnecessary MANAGE_EXTERNAL_STORAGE
  and for permissions requested at startup — the reason B5 was treated as a
  blocker.

## Anti-features

- **NonFreeNet — applies.** PocketClaw promotes and integrates proprietary
  network services: hosted model providers, Telegram and other chat platforms,
  Sogou web search by default. It does not depend entirely on any of them.
- **TetheredNet — does not apply.** Model endpoints are configurable (self-hosted
  OpenAI-compatible servers work), Telegram's Bot API base URL is editable, and
  managed onboarding is optional beside a manual token form. The onboarding
  service is MIT-licensed; its URL is fixed at build time.
- **Tracking — does not apply**, by source audit: no analytics, crash reporting
  or update check, and no network contact before the owner configures a
  provider or channel.
  **Physical observation (2026-09-25, SM-A165F, build `8f6364e8…`):** from a
  cold start, with no interaction for 120 s, every socket owned by
  PocketClaw's UID was sampled every 0.2 s from `/proc/net/{tcp,tcp6,udp,udp6}`
  (no root, so no packet capture; connections shorter than the sample interval
  could be missed). The only external endpoint was `149.154.166.110:443` =
  `api.telegram.org`, 4 s after start: the owner's configured Telegram channel
  long-polling, started by the owner's gateway auto-start preference. No
  analytics, update or other host. Model providers are contacted only on a
  chat turn. One finding: the system WebView, loaded into the app process,
  initialises Google's metrics client (`FilePhenotypeFlags …
  clearcut_client#com.lord1egypt.pocketclaw` in logcat); PocketClaw's own dex
  has no GMS or Clearcut code. The manifest now opts out with
  `android.webkit.WebView.MetricsOptOut` (`b39862c`); on the next build
  (`b4011015…`) the WebView loaded three times with no metrics-client log line.
  A capture on a fresh, unconfigured install is still owed for the "before
  configuration" claim — see *Fresh-install network capture* below.

## Build from source (Phase C recipe outline)

1. `rm:` the committed `android/app/src/main/jniLibs/arm64-v8a/lib*.so`.
2. Install the pinned toolchains from `runtime/toolchains.env`: NDK
   28.2.13676358 (`ndk:`), Go (go1.25.11 for Core; go1.24.6 is fetched by gh's
   recipe), Rust 1.94.1 with `aarch64-linux-android`, pnpm 10.33 on Node, and a
   CPython 3.14 host interpreter for the python payload.
3. Run the runtime recipes (curl before git), then `go test ./pkg/pcruntime/`,
   which fails unless every rebuilt payload matches the catalog Core embeds.
4. `./core/build-android-arm64.sh`.
5. `python3 tool/build_hardened_android.py --signing unsigned`.

Not yet run under `fdroid build`; the two-hour default `timeout:` may not be
enough.

## Reproducibility

See the Phase B section of `PROJECT_STATE.md` for the byte-level evidence. In
short: Core, gh, ripgrep, jq and sqlite3 are reproducible from source; the APK
is reproducible at a fixed path; across checkout paths the Dart AOT snapshot
still differs. `libdartjni.so`'s path-dependent build ID was fixed in Phase B.
The leading explanation for the Dart difference — the kernel records the app's
own libraries by absolute file URI (`package:pocketclaw` resolves to the
checkout) and pub-cache packages under `$HOME` — is consistent with the
evidence but not proven. Until it is fixed, a reproducible build requires
upstream to build at F-Droid's build path and pub-cache location.
