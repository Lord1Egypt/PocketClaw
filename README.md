# PocketClaw

PocketClaw is an independent Android AI-agent product repository. This initial
foundation deliberately preserves the physically verified PicoClaw Android
integration while PocketClaw develops its own product architecture and UI.

## Status

Phase 2 Milestone A: independent Android foundation. The Flutter/Android
foundation is selectively adapted from the PicoClaw FUI reference recorded in
[`UPSTREAM_BASELINE.md`](UPSTREAM_BASELINE.md); it is not a Git fork and does
not merge upstream history.

Run from a configured Flutter/Android environment:

```bash
flutter pub get
flutter analyze
flutter test
cd android && ./gradlew :app:assembleRelease -Ptarget-platform=android-arm64
```

Do not commit generated APKs, Core binaries, tool caches, signing material, or
provider/Telegram/Firebase credentials.
