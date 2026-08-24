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

## Fresh installs use `Download/pocketclaw`; old data is left alone

- Date: 2026-08-25
- Decision: The Android workspace directory for fresh PocketClaw installs is
  `Download/pocketclaw`, including the no-permission fallback and the
  log-export directory. This supersedes the earlier decision to keep the
  compatible `Downloads/picoclaw` path.
- Consequence: Any existing `Download/picoclaw` tree is left untouched and no
  migration runs at startup. A migration, if it is ever wanted, is a separate
  designed feature and must not block the first frame.

## Debrand the embedded web runtime at source, never by binary patching

- Date: 2026-08-25
- Decision: PocketClaw wording in the embedded web console comes from edits to
  the Core web frontend/backend source, rebuilt through the Core Makefile
  targets. The compiled `.so` files are never patched.
- Consequence: `core/pocketclaw-core-v0.3.1.patch` and `core/README.md` carry
  the exact source diff and build commands. Build only via
  `make build-launcher-android-arm64`; calling `make -C web build-android-arm64`
  directly drops the root `LDFLAGS` and yields an unstripped binary, which is
  what produced the earlier oversized `libpicoclaw-web.so`.
- Consequence: use `pnpm lint` on the frontend, not `pnpm check` — the latter
  runs `prettier --write` across the whole tree and rewrites unrelated files.

## The release build fails closed on an incomplete arm64 payload

- Date: 2026-08-25
- Decision: `packageRelease` verifies that the APK contains
  `lib/arm64-v8a/libdartjni.so`, `libpicoclaw.so`, and `libpicoclaw-web.so`,
  and fails the build otherwise.
- Reason: the black-screen regression shipped for days because a missing
  `libdartjni.so` is silent at build time and fatal at launch.
- Consequence: `./gradlew :app:assembleRelease -Ptarget-platform=android-arm64`
  is the canonical PocketClaw release path. Do not use a universal
  `flutter build apk --release` for releases.

## The black-screen incident is closed on physical evidence

- Date: 2026-08-25
- Decision: Treat the black-screen incident as RESOLVED and treat APK
  `2717f32e9580cd5b5ea5da70b2cb9fcf13f6f14451423addcb5686e0278a1de4` as the
  reference physically verified PocketClaw artifact.
- Evidence: a physical Android device passed install, app launch, Flutter first
  frame, Gateway/Core startup, navigation, PocketClaw branding, PocketClaw
  workspace path, and the QR/access page, with no abnormal device slowdown.
  This confirms the missing `lib/arm64-v8a/libdartjni.so` diagnosis and the
  build-pipeline fix.
- Consequence: the release guard and the canonical arm64 Gradle command are
  permanent parts of the release process, not temporary debugging aids. Removing
  either reopens the failure mode that caused this incident. Any future
  regression should be compared against this artifact before new hypotheses are
  formed.

## MQTT defaults to `/pocketclaw`, but never rewrites a configured prefix

- Date: 2026-08-25
- Decision: `mqtt.DefaultTopicPrefix` is `/pocketclaw`. `topicPrefix()`
  substitutes it only when the configured value is empty, so any explicitly
  configured prefix — including the legacy `/picoclaw` — is returned unchanged.
- Reason: the topic prefix is a broker-side contract shared with every other
  subscriber. A fresh PocketClaw install should not advertise the upstream name,
  and an existing deployment must not have its topics moved underneath it.
- Consequence: the Go default and the frontend preview/placeholder/hint must be
  changed together or the preview will misreport the topic an unconfigured
  channel publishes on. Because `topic_prefix` is `omitempty`, a config that
  relied on the *implicit* old default is indistinguishable from a fresh one and
  will move to `/pocketclaw`; pinning it would need a config migration, which is
  deliberately out of scope. Set `topic_prefix` explicitly to keep the old topic.

## The upstream agent skill is not seeded, and not renamed

- Date: 2026-08-25
- Decision: `skills/picoclaw-agent` is excluded from a freshly seeded workspace
  through the `unseededTemplates` list in
  `cmd/picoclaw/internal/onboard/helpers.go`.
- Reason: the skill documents the real upstream `picoclaw` CLI and repository
  internals. Rebranding its text would make its instructions technically wrong,
  so the choice was to seed it or not, and PocketClaw does not.
- Consequence: seeding only ever writes files, so a user who already has that
  skill keeps it, and it can still be installed later from a registry or by
  hand. The general skill loading/install mechanism is untouched.

## Factual third-party hardware references are kept accurate

- Date: 2026-08-25
- Decision: Sipeed, LicheeRV Nano, MaixCAM, and NanoKVM stay in the `hardware`
  skill, as do the real `picoclaw` binary name and `~/.picoclaw/workspace` path
  in shell examples.
- Reason: these are factual references to third-party hardware and to the real
  executable, not PocketClaw product branding.
- Consequence: do not rewrite documentation to reach a superficial zero string
  count. The branding audit classifies occurrences rather than merely counting
  them.
