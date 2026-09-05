#!/usr/bin/env python3
"""Generate the PocketClaw raster brand assets from the vector mark.

The mark is one geometry — an open container with two claw arms rising out of
it, on a 24x24 grid with a 2-unit stroke — and every asset here is that same
geometry drawn at a different size. Keeping the raster pipeline in the tree is
what makes the icons reproducible: nothing is hand-edited, so a change to the
mark is a change to this file and a re-run.

    python3 scripts/generate-brand-assets.py

Writes into public/: favicon.svg, favicon-96x96.png, favicon.ico,
apple-touch-icon.png, web-app-manifest-192x192.png, web-app-manifest-512x512.png.

Colours are the Aperture tokens converted from OKLCH to sRGB:
    canvas  oklch(0.17 0.012 245) -> #0b1014
    claw    oklch(0.70 0.130 208) -> #00b4c8

Only Pillow is required.
"""

from __future__ import annotations

import pathlib

from PIL import Image, ImageDraw

PUBLIC = pathlib.Path(__file__).resolve().parent.parent / "public"

CANVAS = (11, 16, 20, 255)
CLAW = (0, 180, 200, 255)

# Supersampling factor. The mark is drawn large and box-filtered down, which is
# what keeps a 2-unit stroke clean at 16px.
SS = 16

# The mark on its 24x24 grid: three polylines, round caps and joins.
STROKES = [
    [(4, 12), (4, 20), (20, 20), (20, 12)],
    [(8.5, 14), (8.5, 8), (5.5, 3.5)],
    [(15.5, 14), (15.5, 8), (18.5, 3.5)],
]
STROKE_WIDTH = 2.0
# The drawn extent of the mark within the grid, used to centre it optically.
INK_BOX = (4 - 1, 3.5 - 1, 20 + 1, 20 + 1)


def draw_mark(size: int, scale: float, colour: tuple[int, int, int, int]) -> Image.Image:
    """The mark alone, centred on a transparent square of `size` pixels.

    `scale` is the fraction of the square the mark's ink box occupies.
    """
    hi = size * SS
    img = Image.new("RGBA", (hi, hi), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)

    ink_w = INK_BOX[2] - INK_BOX[0]
    ink_h = INK_BOX[3] - INK_BOX[1]
    unit = hi * scale / max(ink_w, ink_h)
    off_x = (hi - ink_w * unit) / 2 - INK_BOX[0] * unit
    off_y = (hi - ink_h * unit) / 2 - INK_BOX[1] * unit

    def px(point: tuple[float, float]) -> tuple[float, float]:
        return (point[0] * unit + off_x, point[1] * unit + off_y)

    width = max(1, round(STROKE_WIDTH * unit))
    radius = width / 2
    for stroke in STROKES:
        points = [px(p) for p in stroke]
        draw.line(points, fill=colour, width=width, joint="curve")
        # Round caps. `joint="curve"` rounds the joins but not the ends.
        for end in (points[0], points[-1]):
            draw.ellipse(
                [end[0] - radius, end[1] - radius, end[0] + radius, end[1] + radius],
                fill=colour,
            )

    return img.resize((size, size), Image.LANCZOS)


def rounded_tile(size: int, radius_fraction: float) -> Image.Image:
    """A canvas-coloured tile with rounded corners, for tab and touch icons."""
    hi = size * SS
    tile = Image.new("RGBA", (hi, hi), (0, 0, 0, 0))
    ImageDraw.Draw(tile).rounded_rectangle(
        [0, 0, hi - 1, hi - 1], radius=hi * radius_fraction, fill=CANVAS
    )
    return tile.resize((size, size), Image.LANCZOS)


def icon(size: int, *, radius_fraction: float, scale: float) -> Image.Image:
    base = rounded_tile(size, radius_fraction)
    base.alpha_composite(draw_mark(size, scale, CLAW))
    return base


SVG = """<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" width="24" height="24">
  <title>PocketClaw</title>
  <style>
    .mark { stroke: #00859a; }
    @media (prefers-color-scheme: dark) { .mark { stroke: #00b4c8; } }
  </style>
  <g class="mark" fill="none" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
    <path d="M4 12v8h16v-8"/>
    <path d="M8.5 14V8L5.5 3.5"/>
    <path d="M15.5 14V8l3-4.5"/>
  </g>
</svg>
"""


def main() -> None:
    (PUBLIC / "favicon.svg").write_text(SVG, encoding="utf-8")

    # Tab icons: a rounded tile so the mark reads on a light or a dark tab bar.
    icon(96, radius_fraction=0.22, scale=0.66).save(PUBLIC / "favicon-96x96.png")
    ico = icon(48, radius_fraction=0.22, scale=0.66)
    ico.save(PUBLIC / "favicon.ico", sizes=[(16, 16), (32, 32), (48, 48)])

    # Apple applies its own mask, so this one is square and full-bleed.
    apple = Image.new("RGBA", (180, 180), CANVAS)
    apple.alpha_composite(draw_mark(180, 0.62, CLAW))
    apple.save(PUBLIC / "apple-touch-icon.png")

    # Maskable PWA icons: the launcher may crop to a circle, so the mark stays
    # inside the central 80% safe zone.
    for size in (192, 512):
        maskable = Image.new("RGBA", (size, size), CANVAS)
        maskable.alpha_composite(draw_mark(size, 0.50, CLAW))
        maskable.save(PUBLIC / f"web-app-manifest-{size}x{size}.png")

    for name in sorted(p.name for p in PUBLIC.iterdir() if p.suffix in {".png", ".ico", ".svg"}):
        print(f"  {name:34s} {(PUBLIC / name).stat().st_size:8d} bytes")


if __name__ == "__main__":
    main()
