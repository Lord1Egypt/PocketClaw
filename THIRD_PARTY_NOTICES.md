# Third-Party Notices

## PicoClaw FUI

The initial Android/Flutter foundation selectively adapts source files from
PicoClaw FUI at commit `d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`.

- Upstream repository: <https://github.com/sipeed/picoclaw_fui>
- Copyright: © 2026 Sipeed
- License: MIT, reproduced at
  [`licenses/sipeed-picoclaw-fui-MIT.txt`](licenses/sipeed-picoclaw-fui-MIT.txt)

## PicoClaw Core

PocketClaw bundles the reviewed PicoClaw Core `v0.3.1` Android arm64 binaries,
rebuilt from source commit `2cf030d2fd3b871d7ec17e3be34c24688aac76da` solely
to preserve the documented Android active-network DNS integration.

- Upstream repository: <https://github.com/sipeed/picoclaw>
- Copyright: © 2026 PicoClaw contributors
- License: MIT, reproduced at
  [`licenses/picoclaw-core-MIT.txt`](licenses/picoclaw-core-MIT.txt).
  The binary provenance and hashes are in
  [`UPSTREAM_BASELINE.md`](UPSTREAM_BASELINE.md).

Flutter and Android dependencies are governed by their own upstream licenses,
as resolved through `pubspec.lock` and Gradle dependency metadata. No final
license has been selected for newly authored PocketClaw code.

## PocketClaw-authored identity assets

`assets/branding/pocketclaw-mark.png` is an original PocketClaw identity asset
generated for this repository from a non-derivative design brief. The selected
second mark is the primary visual direction; its launcher, adaptive, splash,
and monochrome notification variants preserve that identity. It is not a
PicoClaw/FUI image and does not remove or replace the upstream notices above.
