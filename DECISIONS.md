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
