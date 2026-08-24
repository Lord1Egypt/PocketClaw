# PocketClaw Decisions

## Independent repository, not a permanent FUI fork

- Date: 2026-08-24
- Decision: PocketClaw keeps an independent Git history and uses PicoClaw FUI
  as a manually reviewed reference.
- Consequence: No `git merge` maintenance path from FUI; upstream changes must
  be classified as security, bug fix, Android runtime, Core compatibility,
  Telegram, provider, MCP, performance, useful feature, UI-only, or not
  relevant before adoption.

## Preserve the physical-device-proven runtime behavior

- Date: 2026-08-24
- Decision: Keep the Android active-network DNS bridge, pinned Core `v0.3.1`,
  and optional feedback defaults in the independent foundation.
- Consequence: Never revert to `[::1]:53`, hardcode public DNS, include
  Firebase credentials, or make analytics mandatory.

## Delay package identity and product branding

- Date: 2026-08-24
- Decision: Keep the inherited technical package identity until the independent
  foundation builds and is smoke-tested.
- Consequence: Product branding and package-ID changes are a later isolated
  milestone.

## No final license for PocketClaw-authored code yet

- Date: 2026-08-24
- Decision: Preserve upstream MIT notices and defer choosing a license for new
  PocketClaw code until product-owner approval.

## Bound the local Gradle JVM for reproducible foundation builds

- Date: 2026-08-24
- Decision: Set the inherited Gradle JVM maximum heap to 4 GiB and metaspace to
  2 GiB.
- Reason: The inherited 8 GiB/4 GiB settings exceed the available headroom in
  the supported local build environment and can terminate the daemon before
  APK assembly.
- Consequence: This changes only build-process memory limits; Android runtime
  behavior and the pinned Core binaries are unaffected.

## Product identity is PocketClaw; PicoClaw remains the credited engine

- Date: 2026-08-24
- Decision: Use PocketClaw for all product-visible identity and
  `com.lord1egypt.pocketclaw` for the independent Android/Dart integration
  identity. Retain PicoClaw Core names, binaries, environment variables,
  protocol identifiers, and attribution where they are integration contracts.
- Consequence: The PocketClaw and reference APKs can coexist; Android private
  app data is intentionally not migrated across package identities. The
  compatible `Downloads/picoclaw` external workspace remains until a separately
  designed migration exists.

## Skill Hub regression is DNS-resolved, not a separate product defect

- Date: 2026-08-24
- Decision: Keep the existing Skill Hub/ClawHub implementation unchanged.
- Evidence: On a physical device after the active-network DNS fix, Skill Hub
  opened and `Crypto` search returned 20 results with metadata, URLs, and
  visible install actions.
- Consequence: Treat Skill Hub as regression-sensitive; do not install skills
  automatically and do not change its registry configuration without new
  source/runtime evidence.

## Use the selected original PocketClaw mark consistently

- Date: 2026-08-24
- Decision: The second generated PocketClaw mark is the primary visual
  direction. Launcher, adaptive, splash, and monochrome notification variants
  must simplify that identity rather than introduce a cartoon lobster/mascot.
- Consequence: Preserve a restrained professional Android/developer-tool
  appearance and keep future visual work non-derivative of PicoClaw artwork.

## Android splash layers must contain drawables

- Date: 2026-08-24
- Decision: Keep the branded PocketClaw launch mark, but express the splash
  background as an explicit Android color drawable reference rather than
  `android:color` on a `layer-list` item.
- Evidence: `LayerDrawableItem` accepts `android:drawable`; the Milestone B
  resource supplied neither a drawable attribute nor a child drawable, blocking
  launch-theme inflation before Flutter's first frame on the physical device.
- Consequence: `launch_background.xml` and its v21 variant reference
  `@color/pocketclaw_splash_background`; a Flutter regression test guards both
  files. This is not a package-ID migration or Core integration change.

## The `jni` CMake configure cache lives outside the project and can poison builds

- Date: 2026-08-25
- Decision: Treat `~/.pub-cache/hosted/pub.dev/jni-<version>/android/.cxx/` as a
  build input that must be purged when an Android native library goes missing
  from a release APK. Verify `lib/arm64-v8a/libdartjni.so` is packaged before
  releasing any PocketClaw APK.
- Evidence: On 2026-08-24 02:52 the CMake configure of the `jni` package for
  `arm64-v8a` failed inside this workspace with a transient filesystem error
  (`unable to open output file '...CMakeCCompilerABI.c.o': No such file or
  directory`). CMake concluded the C compiler was unusable for that ABI and
  cached a configure result with an empty target list
  (`"libraries": {}`, `"cFileExtensions": []`). Every later build reported the
  JSON "up-to-date", built no arm64 target, and shipped an APK with no
  `lib/arm64-v8a/libdartjni.so`.
- Consequence: `com.github.dart_lang.jni.JniPlugin` calls
  `System.loadLibrary("dartjni")` from a **static initializer**, and
  `GeneratedPluginRegistrant.registerWith` only catches `Exception`. The
  resulting `ExceptionInInitializerError`/`UnsatisfiedLinkError` is an `Error`,
  so it escapes the registrant during `FlutterActivity.onCreate` and the
  Flutter view never attaches — a permanent black screen. `path_provider_android`
  pulls in `jni`/`jni_flutter`, so this affects every build of this app.
- Why it survived cleanups: the poisoned cache is in `~/.pub-cache`, not in the
  project `build/` tree, so deleting build intermediates never cleared it.
