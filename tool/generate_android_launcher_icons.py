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

the two other places Android draws the mark:

    drawable-nodpi/pocketclaw_mark.png   the pre-Android-12 launch splash
    drawable/ic_stat_pocketclaw.xml      the status-bar notification icon, as a
                                         vector built from the same strokes

one derived raster master under assets/branding/android-launcher/, and the
README and web derivative:

    assets/branding/pocketclaw-icon.png   512px rounded tile

and the store-listing icon:

    fastlane/metadata/android/en-US/images/icon.png   512px full-bleed tile,
                                                      the launcher tile at the
                                                      size F-Droid lists

That tile is the installed app's identity as a picture: the launcher's canvas
colour and mark, in the same rounded-tile composition Core already uses for
`favicon.ico` — one idiom, not a second interpretation of the brand. README
shows it, so the repository page cannot drift from the icon on the phone. It
replaced the desktop tray and Windows icons this script used to write
(`assets/app_icon.png`, `assets/icon.ico`); PocketClaw has no desktop build.

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
BRANDING = REPO / "assets/branding"
MASTER_DIR = BRANDING / "android-launcher"
# The store-listing icon F-Droid reads from the upstream Fastlane metadata.
FASTLANE_IMAGES = REPO / "fastlane/metadata/android/en-US/images"
FASTLANE_ICON_PX = 512


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

# README and web identity. The mark sits on a rounded canvas tile, which is what
# Core's own favicon.ico does, so it reads on GitHub's light and dark themes
# alike. The radius and scale are the canonical tab-icon values, taken from the
# brand module's own usage rather than chosen here, so there is still exactly
# one place that decides how the mark is framed.
TILE_RADIUS_FRACTION = 0.22
TILE_SCALE = 0.66
TILE_PX = 512

# The pre-Android-12 splash draws this bitmap inside a 160dp box on the splash
# background (launch_background.xml). 640px is that box at xxxhdpi; nodpi so it
# is scaled once, into the box, rather than per density bucket.
SPLASH_PX = 640
SPLASH_SCALE = 0.75


def status_icon_vector(brand) -> bytes:
    """The notification small icon as a vector drawable, built from the strokes.

    Android draws a small icon as a white silhouette, so it is the same
    polylines stroked in white — not a raster, and not a second drawing.
    """
    def path(points) -> str:
        head, *rest = points
        return f"M{head[0]:g},{head[1]:g} " + " ".join(f"L{x:g},{y:g}" for x, y in rest)

    paths = "\n".join(
        "    <path\n"
        f'        android:pathData="{path(stroke)}"\n'
        '        android:fillColor="#00000000"\n'
        '        android:strokeColor="#FFFFFFFF"\n'
        f'        android:strokeWidth="{brand.STROKE_WIDTH:g}"\n'
        '        android:strokeLineCap="round"\n'
        '        android:strokeLineJoin="round" />'
        for stroke in brand.STROKES
    )
    return (
        "<!-- Generated by tool/generate_android_launcher_icons.py from the canonical\n"
        "     APERTURE strokes. Do not edit by hand. -->\n"
        '<vector xmlns:android="http://schemas.android.com/apk/res/android"\n'
        '    android:width="24dp"\n'
        '    android:height="24dp"\n'
        '    android:viewportWidth="24"\n'
        '    android:viewportHeight="24">\n'
        f"{paths}\n"
        "</vector>\n"
    ).encode("utf-8")

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
    write(legacy_tile(brand, FASTLANE_ICON_PX), FASTLANE_IMAGES / "icon.png")

    write(
        brand.draw_mark(SPLASH_PX, SPLASH_SCALE, brand.CLAW),
        RES / "drawable-nodpi/pocketclaw_mark.png",
    )
    record(RES / "drawable/ic_stat_pocketclaw.xml", status_icon_vector(brand))

    write(
        brand.icon(TILE_PX, radius_fraction=TILE_RADIUS_FRACTION, scale=TILE_SCALE),
        BRANDING / "pocketclaw-icon.png",
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
