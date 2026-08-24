# FUI Component Classification

Reviewed baseline: PicoClaw FUI
`d689c94c1b67f625f70ec4111a9aa3f01be9cbb3`.

| Classification | Components | Milestone A treatment |
| --- | --- | --- |
| REQUIRED FOUNDATION | Flutter bootstrap, Android app/service/method channel, Core binary packaging, service manager, configuration, provider/model flow, Telegram-capable Core configuration, logs, workspace behavior, platform adapters, unit/widget tests | Selectively adapted to retain the proven Android/Core runtime path. |
| REUSABLE WITH ATTRIBUTION | Dashboard, chat, config, log, WebView UI, localization resources, application icons, assets, and build tooling | Included under the preserved FUI MIT notice; UI remains reference-like until a later approved product milestone. |
| REFERENCE ONLY | FUI README translations, screenshots, CI documents, release process, desktop/iOS/web platform runners | Not copied into this Android foundation. Consult only during manual upstream review. |
| REIMPLEMENT FOR POCKETCLAW | Product information architecture, branding, package identity, Telegram easy-linking UX, provider catalog, custom-provider UX, MCP/workspace product surfaces | Explicitly deferred until after the independent foundation APK passes physical-device smoke testing. |
| NOT NEEDED | Upstream Git history, build outputs, Gradle/Dart caches, generated APKs, upstream credentials, signing material, desktop-only binaries | Excluded from this repository. |

This classification is a maintenance boundary: upstream changes are reviewed and
adapted selectively; PocketClaw never automatically merges the FUI repository.
