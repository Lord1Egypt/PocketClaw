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
    python3 tool/generate_android_launcher_icons.py --check   # regenerate, compare, write nothing

Writes, under android/app/src/main/res/:

    drawable-{m,h,xh,xxh,xxxh}dpi/ic_launcher_foreground.png   transparent, claw
    drawable-{m,h,xh,xxh,xxxh}dpi/ic_launcher_monochrome.png   transparent, white
    mipmap-{m,h,xh,xxh,xxxh}dpi/ic_launcher.png                opaque canvas tile

one derived raster master under assets/branding/android-launcher/, and the two
desktop-identity assets:

    assets/app_icon.png    512px rounded tile, tray icon and macOS icon source
    assets/icon.ico        multi-size ICO, Windows tray and Windows icon source

Those last two were the final PicoClaw lobster artwork in the tree. They are
derived here rather than maintained by hand so they cannot drift from the mark
again, and they use the same rounded-tile composition Core already uses for
`favicon.ico` — one idiom, not a second interpretation of the brand.

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
import io
import pathlib
import sys

from PIL import Image

REPO = pathlib.Path(__file__).resolve().parent.parent
CANONICAL = REPO / "core/src/web/frontend/scripts/generate-brand-assets.py"
RES = REPO / "android/app/src/main/res"
MASTER_DIR = REPO / "assets/branding/android-launcher"
ASSETS = REPO / "assets"


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

# Desktop identity. The mark sits on a rounded canvas tile, which is what Core's
# own favicon.ico does — a tray icon has to read against a light or a dark system
# bar, and a bare glyph reads as neither. The radius and scale are the canonical
# tab-icon values, taken from the brand module's own usage rather than chosen
# here, so there is still exactly one place that decides how the mark is framed.
DESKTOP_RADIUS_FRACTION = 0.22
DESKTOP_SCALE = 0.66
APP_ICON_PX = 512

# Windows reads whichever frame fits the surface it is drawing: 16px in a tray,
# 32px in a title bar, 256px in the shell. A single-frame 256px ICO — which is
# what the lobster file was — leaves the small sizes to be downscaled at draw
# time by whatever is asking, and they blur. Rendering the frames here is the
# only way they are anti-aliased from the geometry rather than from a bitmap.
ICO_PX = 256
ICO_SIZES = [(16, 16), (24, 24), (32, 32), (48, 48), (64, 64), (128, 128), (256, 256)]

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
    # --check regenerates every artifact into memory and compares it against
    # what is tracked, writing nothing. It is how a test proves the committed
    # rasters are the ones this script produces, without the test needing to
    # decode images or compare them perceptually: either the bytes match or the
    # tree is stale.
    check = "--check" in sys.argv[1:]

    brand = canonical_brand()

    written: list[pathlib.Path] = []
    stale: list[str] = []

    def record(path: pathlib.Path, data: bytes) -> None:
        if check:
            current = path.read_bytes() if path.exists() else b""
            if current != data:
                what = "missing" if not current else f"{len(current)} tracked vs {len(data)} generated"
                stale.append(f"{path.relative_to(REPO)} ({what})")
            return
        path.parent.mkdir(parents=True, exist_ok=True)
        path.write_bytes(data)
        written.append(path)

    def encode(image: Image.Image, path: pathlib.Path, **save_kwargs) -> bytes:
        buffer = io.BytesIO()
        image.save(buffer, format=path.suffix.lstrip(".").upper(), **save_kwargs)
        return buffer.getvalue()

    def write(image: Image.Image, path: pathlib.Path, **save_kwargs) -> None:
        record(path, encode(image, path, **save_kwargs))

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

    # Desktop identity.
    write(
        brand.icon(
            APP_ICON_PX,
            radius_fraction=DESKTOP_RADIUS_FRACTION,
            scale=DESKTOP_SCALE,
        ),
        ASSETS / "app_icon.png",
    )

    write(
        brand.icon(
            ICO_PX, radius_fraction=DESKTOP_RADIUS_FRACTION, scale=DESKTOP_SCALE
        ),
        ASSETS / "icon.ico",
        sizes=ICO_SIZES,
    )

    if check:
        if stale:
            print("These tracked assets are not what the generator produces:")
            for entry in stale:
                print(f"  {entry}")
            print("\nRun: python3 tool/generate_android_launcher_icons.py")
            raise SystemExit(1)
        print("All generated branding assets match the canonical APERTURE mark.")
        return

    for path in written:
        print(f"  {path.relative_to(REPO)!s:62s} {path.stat().st_size:8d} bytes")


if __name__ == "__main__":
    sys.exit(main())
