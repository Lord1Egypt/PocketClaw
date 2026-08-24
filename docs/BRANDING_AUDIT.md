# PocketClaw Branding Audit

Reviewed on 2026-08-24 for Milestone B. Product-facing identity is PocketClaw; PicoClaw remains the attributed engine.

| Important location | Current/baseline value | Classification | Change | Compatibility reason |
| --- | --- | --- | --- | --- |
| `android/app/src/main/AndroidManifest.xml` | `PicoClaw` label | USER-FACING BRANDING | `PocketClaw` | Launcher and Android settings show the product name. |
| `lib/main.dart`, `lib/l10n/*.arb` | PicoClaw UI/title/hints | USER-FACING BRANDING | PocketClaw | Flutter titles and visible guidance identify the product. |
| About dialog | PicoClaw application text | USER-FACING BRANDING / UPSTREAM ATTRIBUTION | PocketClaw app; PicoClaw Core project/Sipeed retained | Separates product from engine provenance. |
| Android namespace, `applicationId`, Kotlin packages | `com.sipeed.picoclaw` | ANDROID PACKAGE IDENTITY | `com.lord1egypt.pocketclaw` | Independent install identity; permits side-by-side validation. |
| Dart package/import prefix | `picoclaw_flutter_ui` | SOURCE / TEST | `pocketclaw` | Independent application source identity. |
| Flutter MethodChannel and Android action strings | `com.sipeed.picoclaw/...` | ANDROID PACKAGE IDENTITY | `com.lord1egypt.pocketclaw/...` | Both Dart and Kotlin move together; no Core protocol change. |
| `PicoClawService`, Core version labels, Core URLs | PicoClaw | CORE INTERNAL IDENTIFIER / UPSTREAM ATTRIBUTION | retained | Product is powered by PicoClaw Core. |
| `libpicoclaw.so`, `libpicoclaw-web.so` | PicoClaw binary names | CORE BINARY NAME | retained | Required binary packaging/runtime contract. |
| `PICOCLAW_DNS_SERVER`, `PICOCLAW_*` configuration | PicoClaw variables | ENVIRONMENT VARIABLE / CONFIG COMPATIBILITY | retained | Required Core/feedback contract. |
| `Downloads/picoclaw`, app-internal `picoclaw` | legacy workspace/config paths | WORKSPACE PATH / CONFIG COMPATIBILITY | retained | Safe workspace migration is deliberately deferred. |
| `THIRD_PARTY_NOTICES.md`, licenses, upstream docs | PicoClaw/Sipeed | UPSTREAM ATTRIBUTION / DOCUMENTATION | retained | Required MIT attribution and provenance. |
| Core-oriented tests | PicoClaw names | TEST | retained or package-prefix migrated | Tests keep exercising engine integration. |

No wholesale branding replacement is permitted. References not listed here use the same product-versus-engine distinction.
