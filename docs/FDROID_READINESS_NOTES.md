# F-Droid readiness notes

Observations recorded during the post-v0.2.1 Android cleanup
(`feature/post-v0.2.1-android-cleanup`, 2026-09-24). **Observations only.**
Nothing here was changed for F-Droid, no metadata was created and no
submission was made. The standing analysis is `docs/FDROID_RELEASE.md`; this
file records what the cleanup learned on top of it.

## Improved by this cleanup

- **Fewer dependencies.** Removed from `pubspec.yaml`: `flutter_background_service`,
  `flutter_local_notifications`, `file_picker`, `bitsdojo_window`,
  `desktop_webview_window`, `tray_manager`, `webview_windows`,
  `window_manager`, `windows_single_instance`, `process_run`,
  `flex_color_scheme`, `local_session_timeout`, `archive`, `args`, `path`,
  `web_socket_channel` and the dev tool `flutter_launcher_icons`. The Android
  plugin set is now `jni`, `package_info_plus`, `path_provider_android`
  (transitive), `share_plus`, `shared_preferences_android`,
  `url_launcher_android`, `webview_flutter_android`.
- **Fewer permissions and components.** The packaged manifest no longer
  requests `FOREGROUND_SERVICE_DATA_SYNC`, `RECEIVE_BOOT_COMPLETED` or
  `VIBRATE`, has no `GET_CONTENT` query, and has no boot or watchdog
  receivers. `tool/release_gate.py` forbids all three permissions.
- **A downloader is gone.** `tools/fetch_core_local.dart` downloaded prebuilt
  Core binaries from upstream GitHub releases. It was unreferenced; it is
  deleted. Nothing in the build fetches a binary.
- **Icons are generated, not drawn.** `tool/generate_android_launcher_icons.py`
  derives every launcher, splash, notification and README image from one
  geometry. `assets/branding/android-launcher/ic_launcher_master.png` (1024 px,
  full-bleed square) is a ready listing icon; `assets/branding/pocketclaw-icon.png`
  (512 px rounded tile) is the README image.

## Blockers and questions for the F-Droid phase

| Area | Observation |
| --- | --- |
| Committed prebuilts | `android/app/src/main/jniLibs/arm64-v8a/` holds the staged Core pair and eight Managed Runtime payloads. F-Droid builds from source; the recipes exist (`core/build-android-arm64.sh`, `runtime/`), but an F-Droid build must run them and must not use the committed binaries. |
| Proprietary SDK in the build files | `com.umeng.umsdk:common` and `:asms` are `compileOnly` in every build and `implementation` only when `POCKETCLAW_ANALYTICS_PROVIDER=umeng`. The shipped APK contains no Umeng code (the gate checks), but `AnalyticsReporter.kt` imports Umeng, so compiling needs the SDK. F-Droid's scanner flags the dependency by name. Options for that phase: a source set without `AnalyticsReporter`, or removing the analytics provider. Not decided here. |
| Empty Umeng manifest entries | `UMENG_APPKEY` / `UMENG_CHANNEL` `<meta-data>` ship with empty values in the default build. Harmless, but reviewers will ask. |
| Network services | Managed Telegram onboarding calls a PocketClaw-hosted service whose URL is compiled in (`android/official-onboarding.properties`); `build_hardened_android.py --onboarding omit` or `--onboarding-url` exists for downstream builds. Model providers are remote by design. Candidate anti-feature: **NonFreeNet** for the hosted onboarding service, depending on how F-Droid classifies it. |
| Toolchains | The recipes need Go 1.25.11 (resolved from `GOMODCACHE`), Android NDK 28.2.13676358, pnpm, Flutter 3.47.1 / Dart 3.13.1, JDK 17, and the Rust/C toolchains of the Managed Runtime recipes. Defaults point at `/home/lordegypt/PocketCLaw/.tooling`; every path is overridable by environment, which an F-Droid recipe must set. |
| Reproducibility | Both Core binaries are byte-reproducible from source (BuildTime from the last build-input commit). Full-APK reproducibility is **not proven**: Dart AOT with obfuscation and R8 output have not been compared across two clean builds. |
| Loose version constraints | `intl: any` in `pubspec.yaml`; the lockfile pins it, but a recipe must use `--enforce-lockfile`. |
| Signing | Production APKs are signed with the owner key `176dca6b…`. F-Droid signs its own builds unless reproducible builds with the upstream signature are adopted; that decision belongs to the F-Droid phase. |

## Permissions a reviewer will ask about

| Permission | Why PocketClaw holds it |
| --- | --- |
| `INTERNET`, `ACCESS_NETWORK_STATE` | Model providers, Telegram, the LAN dashboard. |
| `FOREGROUND_SERVICE`, `FOREGROUND_SERVICE_SPECIAL_USE` | `PocketClawService` hosts Core as a long-running local agent. |
| `WAKE_LOCK` | `PocketClawService` holds a partial wake lock while Core runs. |
| `POST_NOTIFICATIONS` | The foreground-service notification (Android 13+). |
| `MANAGE_EXTERNAL_STORAGE` | The workspace in `Download/pocketclaw`, where the user can reach their files. Without it the workspace falls back to the app's own external folder (see PC-DEF-077, still open). The strongest review question. |
| `READ_/WRITE_EXTERNAL_STORAGE` (maxSdk 32 / 28) | The same workspace on older Android versions. |
| `DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION` | Generated by AndroidX for non-exported dynamic receivers. |

## Not yet present

No `fastlane/` metadata, no screenshots, no F-Droid build recipe. Store
descriptions would need the same truthful scope as README (arm64-v8a only,
Android 7+, API key required).
