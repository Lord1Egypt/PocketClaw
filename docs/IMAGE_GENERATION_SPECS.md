# PocketClaw Visual Generation Specifications

## Selected primary direction

The second original PocketClaw mark is the approved primary visual direction.
All compact Android variants simplify its pocket/container, terminal/claw, and
luminous-node composition. Do not substitute a cartoon lobster or mascot.

## Generated in Milestone B

`assets/branding/pocketclaw-mark.png` was generated with the built-in image generator as an original, text-free 1254×1254 transparent PNG. Concept: a compact pocket/container opens into three precise claw-like terminal forms around a luminous agent node. It avoids animals, mascots, PicoClaw artwork, text, and watermarks. It is used as the launcher/adaptive foreground and splash mark.

## Future production assets

| Asset | Specification | Target |
| --- | --- | --- |
| Main logo | 2048×2048 transparent PNG/SVG-ready source; centered original mark with optional separate PocketClaw wordmark; 20% clear margin; no embedded small text. | `assets/branding/` |
| Launcher icon | 1024×1024 RGBA PNG; mark occupies central 60% adaptive safe zone; opaque #08111F background comes from Android. | launcher generator source |
| Adaptive foreground/background | Foreground 108×108dp-equivalent transparent PNG with 66×66dp safe content; background solid #08111F or texture-free 432×432 PNG. | `mipmap-anydpi-v26/` |
| Splash illustration | 1080×1920 portrait or scalable transparent 1024×1024 mark; dark #08111F field, central lightweight mark, no startup delay or embedded status text. | `drawable*/` |
| Notification icon | 24dp monochrome white vector, simple pocket/claw silhouette, no gradients/raster detail. | `drawable/ic_stat_pocketclaw.xml` |
| Onboarding hero | 1600×900 transparent/opaque PNG; restrained developer-tool scene, clear negative space, no fake UI text or mascot. | future `assets/branding/` |
| Empty-state illustration | 1024×768 transparent PNG; small abstract agent/pocket mark with subtle terminal lines, no provider logos or text. | future `assets/branding/` |

All future prompts must request original, non-derivative art and avoid PicoClaw/Sipeed logos, animal mascots, generic robot heads, watermarks, and embedded text.
