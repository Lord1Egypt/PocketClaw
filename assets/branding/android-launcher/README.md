# Derived Android launcher artwork

`ic_launcher_master.png` is a **derived artifact**, not a design source. It is
written by `tool/generate_android_launcher_icons.py` from the canonical
APERTURE geometry that lives in
`core/src/web/frontend/scripts/generate-brand-assets.py` — the same geometry the
React mark component and `favicon.svg` carry.

Do not hand-edit it, and do not treat it as a second geometry master. To change
the launcher, change the mark in Core and re-run:

    python3 tool/generate_android_launcher_icons.py

It exists so `flutter_launcher_icons` has an APERTURE-derived `image_path`
rather than a pointer back to the glossy 3D mark. Android generation is
disabled in that package; see the comment above the block in `pubspec.yaml`.
