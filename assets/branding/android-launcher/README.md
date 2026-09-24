# Derived Android launcher artwork

`ic_launcher_master.png` is a **derived artifact**, not a design source. It is
written by `tool/generate_android_launcher_icons.py` from the canonical
APERTURE geometry that lives in
`core/src/web/frontend/scripts/generate-brand-assets.py` — the same geometry the
React mark component and `favicon.svg` carry.

It is the launcher's opaque legacy tile at 1024 px: full-bleed and square, the
shape store listings ask for, where the platform applies its own mask. The
rounded variant used by the README is `../pocketclaw-icon.png`.

Do not hand-edit it, and do not treat it as a second geometry master. To change
the launcher, change the mark in Core and re-run:

    python3 tool/generate_android_launcher_icons.py
