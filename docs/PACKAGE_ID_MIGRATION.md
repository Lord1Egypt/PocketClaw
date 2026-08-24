# Android Package-ID Migration

## Applied migration

| Concern | Before | After |
| --- | --- | --- |
| `applicationId` | `com.sipeed.picoclaw` | `com.lord1egypt.pocketclaw` |
| Android namespace | `com.sipeed.picoclaw` | `com.lord1egypt.pocketclaw` |
| Kotlin source root | `com/sipeed/picoclaw` | `com/lord1egypt/pocketclaw` |
| Flutter MethodChannel | `com.sipeed.picoclaw/picoclaw` | `com.lord1egypt.pocketclaw/picoclaw` |
| Android service action strings | old package prefix | new package prefix |
| ProGuard rules | old namespace | new namespace |

The manifest uses relative component names, so activities, services, and the boot receiver resolve under the new namespace. No FileProvider authority or custom deep-link host is declared. The optional Umeng scheme remains a build placeholder and is not a PocketClaw identity/deep-link feature.

## Data and release consequences

- Android treats the new application ID as a separate application. Its private files, preferences, service state, and Core configuration do not migrate automatically from `com.sipeed.picoclaw`.
- No cross-package private-data copy is attempted in this milestone.
- PicoClaw Core configuration formats and export/import behavior are retained.
- The compatible external workspace path `Downloads/picoclaw/` remains unchanged. A safe, user-approved PocketClaw workspace migration is future technical debt and must not be coupled to this package migration.
- The old reference APK and the new PocketClaw APK can coexist for regression comparison. Play signing/release identity is a future controlled milestone.
