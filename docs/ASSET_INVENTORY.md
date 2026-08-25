# PocketClaw Asset Inventory

| Path | Dimensions / format | Used by | PicoClaw branding | Provenance | Milestone B decision |
| --- | --- | --- | --- | --- | --- |
| `assets/app_icon.png` | 512×512 PNG | legacy Flutter launcher config | Yes | PicoClaw FUI MIT | Retain only as attribution-era source; not used by PocketClaw launcher. |
| `assets/icon.ico` | ICO | desktop tray fallback | Yes | PicoClaw FUI MIT | Reference-only for Android milestone; replace in later desktop identity work. |
| `android/app/src/main/res/mipmap-*/ic_launcher.png` | 48–192 PNG | legacy Android launcher | Yes | PicoClaw FUI MIT | Replaced by generated PocketClaw launcher output. |
| `android/app/src/main/res/drawable*/launch_background.xml` | XML | Android startup splash | No artwork | Flutter template | Updated to PocketClaw dark splash with original mark. |
| `assets/branding/pocketclaw-mark.png` | 1254×1254 RGBA PNG | launcher/adaptive foreground/source mark | No | Original, generated for PocketClaw | Keep. |
| `android/app/src/main/res/drawable/pocketclaw_mark.png` | 1254×1254 RGBA PNG | lightweight Android splash | No | Derived from PocketClaw mark | Keep. |
| `android/app/src/main/res/drawable/ic_stat_pocketclaw.xml` | 24dp vector XML | foreground-service notification | No | PocketClaw-authored | Keep; intentionally simple monochrome glyph. |
| Web/Desktop/iOS assets | not copied into Android foundation | none in Milestone B Android APK | n/a | n/a | Reference-only / future platform work. |

There are no embedded branded illustrations, custom fonts, JPEGs, WebP files, or SVG logos in the Android foundation. No upstream raster is edited to create the PocketClaw mark.
