#!/usr/bin/env python3
"""Generate the Android launcher resources from the canonical APERTURE mark.

Every file this writes is a DERIVED artifact. The geometry lives in exactly one
place — `core/src/web/frontend/scripts/generate-brand-assets.py`, which is the
same geometry the React component and `favicon.svg` carry — and this script
imports that module read-only rather than restating it. There is deliberately
no `STROKES` table here: a fourth copy would drift the moment the mark is
refined again, which is the failure the Core branding test already guards
against.

    python3 tool/generate_android_launcher_icons.py

Writes, under android/app/src/main/res/:

    drawable-{m,h,xh,xxh,xxxh}dpi/ic_launcher_foreground.png   transparent, claw
    drawable-{m,h,xh,xxh,xxxh}dpi/ic_launcher_monochrome.png   transparent, white
    mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher.png                opaque canvas tile

and one derived raster master under assets/branding/android-launcher/.

The legacy mipmaps are written without an alpha channel. Pre-26 launchers apply
no mask and composite nothing behind the icon, so a transparent-background
launcher icon floats on the wallpaper — which is exactly what shipped before
this script existed. Encoding them as RGB makes the opacity a property of the
file format rather than of its pixels, so the guard in
`test/unit/launcher_icon_test.dart` can prove it cheaply.

Only Pillow is required.
"""

from __future__ import annotations

import importlib.util
import pathlib
import sys

from PIL import Image

REPO = pathlib.Path(__file__).resolve().parent.parent
CANONICAL = REPO / "core/src/web/frontend/scripts/generate-brand-assets.py"
RES = REPO / "android/app/src/main/res"
MASTER_DIR = REPO / "assets/branding/android-launcher"


def canonical_brand():
    """Import the Core brand module by path, read-only.

    Its filename is hyphenated, so it cannot be imported by name. Nothing in
    this script writes to Core; bytecode caching is off so importing the module
    does not drop a __pycache__ into the Core tree either.
    """
    sys.dont_write_bytecode = True
    spec = importlib.util.spec_from_file_location("pocketclaw_brand", CANONICAL)
    if spec is None or spec.loader is None:
        raise SystemExit(f"cannot load the canonical mark from {CANONICAL}")
    module = importlib.util.module_from_spec(spec)
    spec.loader.exec_module(module)
    return module


# Adaptive foreground: the drawable is laid out over the full 108dp canvas and
# the adaptive-icon XML insets it by 16%, so the mark's ink box ends up at
# 108 * 0.68 * 0.65 = 47.7dp. Its farthest ink point — the outer edge of a
# bottom pocket corner — then sits 32.2dp from centre, inside Android's 33dp
# mask-safe radius. `launcher_icon_test.dart` re-measures that from the written
# PNG rather than trusting this comment.
FOREGROUND_SCALE = 0.65

# Legacy icons carry no mask and no inset, so the mark is drawn larger.
LEGACY_SCALE = 0.62

MONOCHROME = (255, 255, 255, 255)

# Density buckets. The adaptive layers are 108dp square, the legacy icons 48dp.
DENSITIES = {"mdpi": 1, "hdpi": 1.5, "xhdpi": 2, "xxhdpi": 3, "xxxhdpi": 4}
ADAPTIVE_DP = 108
LEGACY_DP = 48
MASTER_PX = 1024


def legacy_tile(brand, size: int) -> Image.Image:
    """An opaque canvas tile with the mark on it, as RGB with no alpha."""
    tile = Image.new("RGBA", (size, size), brand.CANVAS)
    tile.alpha_composite(brand.draw_mark(size, LEGACY_SCALE, brand.CLAW))
    return tile.convert("RGB")


def main() -> None:
    brand = canonical_brand()

    written: list[pathlib.Path] = []

    def write(image: Image.Image, path: pathlib.Path) -> None:
        path.parent.mkdir(parents=True, exist_ok=True)
        image.save(path)
        written.append(path)

    for bucket, factor in DENSITIES.items():
        adaptive = round(ADAPTIVE_DP * factor)
        legacy = round(LEGACY_DP * factor)

        write(
            brand.draw_mark(adaptive, FOREGROUND_SCALE, brand.CLAW),
            RES / f"drawable-{bucket}/ic_launcher_foreground.png",
        )
        write(
            brand.draw_mark(adaptive, FOREGROUND_SCALE, MONOCHROME),
            RES / f"drawable-{bucket}/ic_launcher_monochrome.png",
        )
        write(
            legacy_tile(brand, legacy),
            RES / f"mipmap-{bucket}/ic_launcher.png",
        )

    write(legacy_tile(brand, MASTER_PX), MASTER_DIR / "ic_launcher_master.png")

    for path in written:
        print(f"  {path.relative_to(REPO)!s:62s} {path.stat().st_size:8d} bytes")


if __name__ == "__main__":
    sys.exit(main())
