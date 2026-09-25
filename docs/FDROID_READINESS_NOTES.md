# F-Droid readiness notes

Where PocketClaw stands against official F-Droid inclusion, measured against the
**current** policy documents (read 2026-09-24): the Inclusion Policy, the
Anti-Features list, the Build Metadata Reference, the fdroiddata "App inclusion"
merge-request template and the reviewers' wiki. The standing background analysis
is [`FDROID_RELEASE.md`](FDROID_RELEASE.md).

> **Golden #3 (2026-09-25):** PocketClaw v0.2.2, source `80c9dc0`, owner-signed
> APK `320368ea…`, minSdk 26, arm64-v8a — published as the GitHub release
> `v0.2.2` (tag at `e535fcabebed3ac977559994fad597c62d6345f7`).
>
> **Nothing has been submitted.** No merge request is open and no release was
> made for F-Droid. Phase A audited; Phase B (branch `feature/fdroid-phase-b`)
> fixed the source; Phase C (2026-09-25, below) ran the real `fdroid build`,
> lint and scanner locally against a metadata draft kept in a separate
> fdroiddata checkout outside this repository.

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

## Network before configuration (Phase C, 2026-09-25)

The owner's install was not uninstalled, cleared or reconfigured for any of
this.

- **Fresh Core on the phone.** The Golden #3 Core and Dashboard binaries, run
  from `adb shell` out of the installed app's `nativeLibraryDir` against a
  throwaway workspace in `/data/local/tmp` (gateway moved to 18791 to avoid
  the owner's running instance): in 180 s, sampling the shell UID's
  `/proc/net/{tcp,tcp6,udp,udp6}` every 0.2 s, **no external endpoint**; the
  only listeners were `127.0.0.1`/`::1` on 18791 and 18877. No outbound URL
  in either log.
- **Fresh app on an emulator.** Golden #3 on an Android 16 x86_64 emulator
  (docker, KVM), first launch, no interaction for 270 s: the app UID owned no
  socket; the kernel's per-UID counters showed one packet each way (60/40
  bytes, a loopback probe of the Core port) and the pcap held only the
  system's boot connectivity checks. The arm64 Go Core cannot run under the
  emulator's ARM translation (SIGSEGV), and the emulator's own internet
  reachability was not verified, so this covers the Flutter layer only.
- **Configured phone (earlier, 2026-09-25):** only `api.telegram.org:443`,
  the owner's configured channel.

Not done: a capture of the F-Droid-built APK itself on a disposable arm64
device (none is available; a second Android user on the owner's phone would
need the owner to create it).

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
  The fresh, unconfigured observations are under *Network before
  configuration* above.
- **NonFreeDep, NonFreeAssets, NonFreeAdd, Ads — do not apply.** Every linked
  Go module, npm production dependency and Dart package has a FLOSS licence
  (Phase C licence sweep; paho.mqtt is EPL-2.0/EDL-1.0); fonts are OFL; no
  proprietary SDK in the resolved Gradle graph (release gate); no ads.
- **KnownVuln — not flagged by F-Droid's scanner.** govulncheck does report
  reachable Go advisories (see *Vulnerability scans* below); they are
  disclosed in the reviewer notes and fixed upstream in 0.2.3 rather than
  hidden.

Draft metadata reason (the fdroiddata draft carries it verbatim):
*"Integrates proprietary network services: hosted AI model providers you
choose to configure (a self-hosted OpenAI-compatible endpoint works instead),
messaging channels such as Telegram, the agent's default web search (Sogou;
SearXNG and others are selectable) and the ClawHub skill registry. Nothing is
contacted until you configure a model or a channel."*

## Phase C — local F-Droid dry run (2026-09-25)

**Environment.** fdroidserver 2.4.2 (git `a35fdfdd`, 2026-09-14) and
fdroiddata `786a8c38`, in F-Droid's buildserver image
(`registry.gitlab.com/fdroid/fdroidserver:buildserver`, `sha256:9cb68105…`:
Debian 13, Python 3.13.5, OpenJDK 21), run with `fdroid fetch_srclibs` then
`fdroid build --on-server --no-tarball`. The fdroiddata checkout and all
evidence live in a separate workspace outside this repository; nothing from it
is committed here.

**Metadata draft** (`metadata/com.lord1egypt.pocketclaw.yml`, not submitted):
`commit:` the full `v0.2.2` hash; srclibs `flutter@3.47.1` and
`rustup@1.29.1`; `ndk: r28c`; `rm:` all ten committed `lib*.so`; Node 25.8.1
by SHA-256 and pnpm 10.33.0 in prebuild; the seven runtime recipes, the
payload checksum test, `core/build-android-arm64.sh` and
`build_hardened_android.py --signing unsigned` in build. It depends on no
developer path, secret or signing material. `fdroid readmeta`, `rewritemeta`
and `lint` pass. Three recipe details the runs forced:

| Recipe line | Why |
| --- | --- |
| `unset SOURCE_DATE_EPOCH` | fdroidserver exports the commit time, which overrides the payloads' pinned `RUNTIME_EPOCH` (zip timestamps, CPython's `__DATE__`) |
| `sed` of one catalog hash (prebuild) | PC-DEF-091: the buildserver's CPython payload differs from upstream's only in its build-id; the recipe pins the buildserver's deterministic `0cc0755f…` (identical in two full runs) instead of upstream's `8b52e36d…` |
| a two-line `android/gradlew` that runs `gradle` | fdroidserver deletes `gradlew`, `gradlew.bat` and the wrapper jar; its own `gradle` reads Gradle 8.14 and its checksum from `gradle-wrapper.properties` |

**Build runs.** 1–4 fixed invocation and cache-mount mistakes and the scanner
findings; 5 found the `SOURCE_DATE_EPOCH` override; 6 isolated the Python
build-id; 7 and a Python-only rerun proved its cause (PC-DEF-091); 8 found the
deleted `gradlew`; **9 succeeded**: "Successfully built
com.lord1egypt.pocketclaw:64 from e535fcab…", about 23 minutes on this
machine.

**Scanner.** Passes. Two `scandelete` globs, both inside the in-tree pub
cache, cover every finding: pub packages' `example/` Android projects
(unknown Maven repositories) and the DevTools extension builds shipped inside
`shared_preferences` and `provider` (wasm, `AssetManifest.bin`). Nothing in
PocketClaw's own source is flagged; the committed `lib*.so` are removed by
`rm:` and rebuilt.

**The F-Droid APK** (`com.lord1egypt.pocketclaw_64.apk`, 61,338,235 bytes,
`3aad94d4…`, unsigned): `com.lord1egypt.pocketclaw` 0.2.2 (64), minSdk 26,
targetSdk 36, `native-code: 'arm64-v8a'`, the same seven permissions and the
same exported components as the table above; apksigner: DOES NOT VERIFY (no
signature). `release_gate.py --verify-artifact --release-class repository
--artifact-class non-publish-audit` from this repository: 34 PASS, 2 FAIL —
both Core provenance rows, because the recipe's catalog edit gives F-Droid's
Core a different source fingerprint (`a543e437…`); run from the F-Droid build
tree the same two rows PASS (fingerprint stamped in both binaries, identical
to what that tree staged). Native ELF audit `--enforce-target`: 146 PASS / 0
FAIL. Against Golden #3, 426 of 430 entries are byte-identical — including
`classes.dex`, resources, the baseline profile and seven payloads, so JDK 21
versus upstream's JDK 17 changes nothing. The four that differ:
`libapp.so` (PC-DEF-092), `libpocketclaw-python.so` (PC-DEF-091) and the
Core pair. Core built on the owner's machine from the same edited catalog is
byte-identical to F-Droid's pair (`eccd6458…` / `89af23be…`).

## Reproducibility

- **Payloads:** curl, git, git-remote-http, gh, ripgrep, sqlite3 and jq
  rebuild on the buildserver byte-identical to Golden #3. CPython differs only
  by PC-DEF-091, and matches exactly when the NDK sits at upstream's path.
- **Core:** reproducible across machines for the same source.
- **APK:** two clean clones of `v0.2.2` at different checkout paths, with
  upstream's Flutter SDK, pub cache and JDK 17, give the identical unsigned
  APK `3efb0081…`; `apksigcopier compare` against Golden #3 matches, and the
  signature copied onto it gives a file byte-identical to `320368ea…`. On the
  buildserver the Dart snapshot differs because its paths differ
  (PC-DEF-092: the checkout path's length, the Flutter SDK location and the
  pub cache location all reach `libapp.so`; a different checkout path of the
  same length does not).

## Signing strategy

**Track A — F-Droid signs its own build — for v0.2.2.** It needs nothing
reproducible and is what the draft does. Consequence to state to users: the
F-Droid APK and the GitHub APK (signer `176dca6b…`) cannot update each other;
switching channels means uninstalling, which deletes app data unless it was
exported first.

**Track B — F-Droid publishes the upstream signature — not possible for
v0.2.2.** It needs a byte-identical APK. Proven remaining gaps: PC-DEF-091
(fixable upstream by mapping the NDK path) and PC-DEF-092 (upstream would
have to build at F-Droid's paths — most simply inside the buildserver image
with this recipe; the Flutter SDK and pub cache locations change the snapshot
too). Then the metadata gains `Binaries:` and `AllowedAPKSigningKeys:
176dca6b198b9552fb4d9ad3ca18da8d6f23c0a3f5ed4bd6b75a0700f9f0efcf`, and the
catalog `sed` goes away.

## Vulnerability scans

govulncheck v1.1.4: see PC-DEF-093 (8 reachable standard-library advisories
in go1.25.11; 29 in the staged Core binary including x/crypto v0.51; 93 in gh
2.82.1). `pnpm audit --prod`: 54 advisories, all in build-time tooling
(`@tailwindcss/vite`, jotai's babel peer, the shadcn CLI) — none present in
the shipped Dashboard bundle. OSV over the 85 locked Dart packages: none.
Secret scan: the source hits are fake fixtures in
`core/pocketclaw-core-v0.3.1.patch`; the APK hits are mbedTLS PEM header
constants.

## Resource use (Golden #3 on the SM-A165F, Android 16)

`dumpsys meminfo` total PSS (includes swapped PSS), KB:

| State | Whole app (Flutter + WebView) | Core `libpocketclaw.so` | Dashboard `libpocketclaw-web.so` | WebView renderer |
| --- | --- | --- | --- | --- |
| Long idle | ~93,000 | ~19,500 | ~15,000 | ~50,000 |
| Dashboard open | ~343,000 | ~21,400 | ~27,400 | ~101,000 |
| Chat turn | ~397,000 | 26,000–27,400 | ~27,900 | ~110,000 |
| Tool work | ~384,000 | ~28,600 | ~27,300 | ~113,000 |

CPU, share of one core: idle — app 0.3 %, Core 0.6 %, Dashboard 0.1 %; during
a chat turn — app ~120 % (UI and WebView rendering), Core 1.5 %, Dashboard
3 %. **What the numbers support:** "Core uses about 20 MB idle and under
30 MB while working." They do not support a lightweight claim for the whole
app, whose memory is dominated by the WebView; never quote a whole-app figure
of 24 MB.

## Fastlane and store text (audit)

Four real screenshots, the 512 px icon and `changelogs/64.txt` are in place.
Two wording issues: the description's "No package manager, no downloads after
install" reads as "the app never downloads anything", while the agent can
fetch skills from ClawHub and files on request; and skills / the ClawHub
registry are not mentioned at all. Suggested for 0.2.3; not a blocker.

## Reviewer-visible residue (low)

**Correction (0.2.3 work, 2026-09-25):** the claim first written here — that
the Android Dashboard binary embeds PicoClaw's unused tray icon — was wrong.
`web/backend/systray_icon_nonwindows.go` did compile for Android, but nothing
there references the variable and the Go linker drops it: the v0.2.2 binary
contains nothing of the 104,580-byte PNG beyond its first 32 bytes (the PNG
signature and image header), and Golden #3's debug info has no symbol for it. 0.2.3 adds the `!android` constraint
anyway (`04c3a52`) so the intent is explicit. Core's CLI onboarding still
prints `🦞 picoclaw is ready!` to the log on first start. `public/lark.svg` is vendored from PicoClaw, whose
authorship upstream does not record (already stated in
`THIRD_PARTY_NOTICES.md`).

## Smallest upstream changes for 0.2.3 (not made)

1. PC-DEF-091: map `$NDK_ROOT` out of payload debug info; re-pin; restage
   Core. Drops the recipe's `sed`.
2. PC-DEF-093: go1.25.13, x/crypto ≥ v0.56 and the other module bumps, a
   current gh.
3. `build_hardened_android.py`: run `gradle` when `android/gradlew` is absent,
   dropping the recipe's shim.
4. Optional, for Track B: build the release in the buildserver image
   (PC-DEF-092).
5. The residue and Fastlane wording above.
