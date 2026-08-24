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

## 2026-08-25 — User-facing debranding audit

Scope: everything a normal user can see in the Android product — Flutter UI,
Android resources and notifications, and the embedded web runtime served by
`libpicoclaw-web.so`.

### Result

| Surface | Count |
| --- | --- |
| User-visible PicoClaw occurrences | 0 |
| User-visible Sipeed occurrences | 0 |
| User-facing GitHub links | 0 |

Two deliberate exceptions are listed under "Open items" below.

### Changed

- Embedded web frontend: assistant display name in chat; all five shipped
  locales (`en`, `zh`, `cs`, `pt-br`, `bn-in`), 7 product-name strings each;
  workspace and evolution-path placeholders; removed the `docs.picoclaw.io`
  link from the channel configuration header; removed the upstream docs button
  from the app header; page title.
- Removed the runtime `productize()` string-replacement shim from
  `src/i18n/index.ts`. Locale data is now correct at source, so the audit is a
  plain grep instead of a runtime transform.
- Web backend: OAuth completion page title, desktop autostart entry name and
  comment, launcher tray strings (English and Chinese).
- Core: `/start` reply, and the seeded onboarding workspace identity in
  `workspace/AGENT.md` and `workspace/SOUL.md`. The upstream lobster mascot
  emoji was dropped along with the name.
- Android: workspace default is now `Download/pocketclaw` for fresh installs,
  including the no-permission fallback directory and the log-export directory.
  Existing `Download/picoclaw` data is left untouched and no migration runs at
  startup.
- Android: the missing-binary error label no longer names PicoClaw.
- Removed three stale generated localization files from `lib/l10n/`
  (`app_localizations*.dart`). `l10n.yaml` generates into `lib/src/generated/l10n`,
  so these were dead copies that still carried the `PicoClaw UI` title.

### Retained intentionally — legal and attribution

`LICENSE`, `licenses/`, `THIRD_PARTY_NOTICES.md`, `UPSTREAM_BASELINE.md`, and
the upstream copyright headers. These are attribution obligations and are not
product identity.

### Retained intentionally — internal compatibility

Not user-visible; changing them would break integration contracts.

- Binary names `libpicoclaw.so`, `libpicoclaw-web.so`; `PICOCLAW_DNS_SERVER`,
  `PICOCLAW_HOME`, `PICOCLAW_CONFIG` and other environment keys.
- Dart `PicoClawChannel` and the `com.lord1egypt.pocketclaw/picoclaw`
  MethodChannel name; Kotlin `PicoClawService`, `PicoClawApp`; the
  `picoclaw_foreground` notification channel id; `filesDir/picoclaw`
  (internal config home — renaming it would discard existing configuration).
- Web frontend browser-storage keys (`picoclaw-tour-state`,
  `picoclaw:last-session-id`, `picoclaw:code-block-wrap`,
  `picoclaw:chat-*`), DOM attributes (`data-picoclaw-code-block`,
  `data-picoclaw-highlight-theme`), and the `picoclaw-oauth-result`
  postMessage type.
- Go module path `github.com/sipeed/picoclaw`, which appears in the stripped
  binaries' function-name tables but never in UI.
- CLI help text, log lines, and config-migration messages inside the Core
  binaries. These are not reachable from the Android UI.

### Open items — need a product decision

1. **MQTT topic prefix.** The Core default is `/picoclaw`
   (`pkg/channels/mqtt/mqtt.go`). It is a broker-side protocol identifier, so it
   was not renamed. It is therefore visible in the MQTT channel configuration
   screen only: the topic preview, the input placeholder, and the
   `channels.form.desc.topicPrefix` hint in all five locales. Changing it means
   changing the Go default and the frontend together.
2. **Bundled `picoclaw-agent` skill.** `workspace/skills/picoclaw-agent/SKILL.md`
   is seeded into the onboarding workspace and appears in the skills list. Its
   subject is the upstream engine's own CLI and repository internals, so
   rebranding its text would make it factually wrong; the options are to keep it
   or to drop it from the seeded workspace. The bundled `hardware` skill
   similarly names Sipeed boards (LicheeRV Nano, MaixCAM, NanoKVM), which is a
   factual reference to third-party hardware rather than product branding.
