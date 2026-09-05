/**
 * Token integrity for the Aperture foundation.
 *
 * The dashboard was running unmodified shadcn defaults: every neutral at
 * chroma 0 and, in dark mode, a near-white primary. That is not a matter of
 * taste — chroma-0 grey is the literal default of the framework, which is
 * exactly why it read as one. These assert the arithmetic that fixes it, not
 * anyone's opinion of the result.
 */
import fsSync from "node:fs"

import { describe, expect, it } from "vitest"

const CSS = fsSync.readFileSync(`${process.cwd()}/src/index.css`, "utf8")

/// The declarations inside one top-level block, by custom-property name.
function block(selector: string): Record<string, string> {
  const start = CSS.indexOf(`\n${selector} {`)
  expect(start, `${selector} block is missing`).toBeGreaterThan(-1)
  const body = CSS.slice(start + selector.length + 3, CSS.indexOf("\n}", start))
  const out: Record<string, string> = {}
  for (const [, name, value] of body.matchAll(/(--[\w-]+):\s*([^;]+);/g)) {
    out[name] = value.trim().replace(/\s+/g, " ")
  }
  return out
}

const light = block(":root")
const dark = block(".dark")

/// Every OKLCH literal in a declaration, as [L, C, H].
function oklch(value: string): [number, number, number][] {
  return [...value.matchAll(/oklch\(\s*([\d.]+)\s+([\d.]+)\s+([\d.]+)/g)].map(
    (m) => [Number(m[1]), Number(m[2]), Number(m[3])] as [number, number, number],
  )
}

const NEUTRALS = [
  "--pc-bg",
  "--pc-surface-1",
  "--pc-surface-2",
  "--pc-surface-3",
  "--pc-border",
  "--pc-border-strong",
  "--pc-text",
  "--pc-text-muted",
  "--pc-text-faint",
]

describe.each([
  ["light", light],
  ["dark", dark],
])("the %s palette", (name, palette) => {
  it("defines the whole neutral ramp", () => {
    for (const token of NEUTRALS) {
      expect(palette[token], `${name} is missing ${token}`).toBeTruthy()
    }
  })

  it("carries the brand hue in every tinted neutral", () => {
    for (const token of NEUTRALS) {
      for (const [, chroma, hue] of oklch(palette[token])) {
        // Pure white is the one sanctioned exception: a light-mode card lifts
        // by being lighter than the canvas, and white is the top of that ramp.
        if (chroma === 0) {
          expect(palette[token], `${name} ${token}`).toContain("oklch(1 0 0)")
          continue
        }
        expect(hue, `${name} ${token} hue`).toBe(245)
        expect(chroma, `${name} ${token} chroma`).toBeGreaterThan(0)
        expect(chroma, `${name} ${token} chroma`).toBeLessThan(0.03)
      }
    }
  })

  it("puts the accent and the live-state colour on their own hues", () => {
    expect(oklch(palette["--pc-primary"])[0][2]).toBe(208)
    expect(oklch(palette["--pc-signal"])[0][2]).toBe(195)
  })

  it("distinguishes live state from interactive intent", () => {
    expect(palette["--pc-signal"]).not.toBe(palette["--pc-primary"])
  })

  it("remaps the shadcn variables onto the Aperture layer rather than dropping them", () => {
    for (const shadcn of [
      "--background",
      "--foreground",
      "--card",
      "--popover",
      "--primary",
      "--primary-foreground",
      "--muted-foreground",
      "--border",
      "--ring",
      "--sidebar",
    ]) {
      expect(palette[shadcn], `${name} ${shadcn}`).toMatch(/^var\(--pc-/)
    }
  })

  it("leaves no stock shadcn chroma-0 neutral behind", () => {
    for (const [token, value] of Object.entries(palette)) {
      if (!value.startsWith("oklch(")) continue
      for (const [lightness, chroma] of oklch(value)) {
        if (chroma !== 0) continue
        expect(
          lightness,
          `${name} ${token} is a chroma-0 grey`,
        ).toBe(1)
      }
    }
  })
})

describe("the dark palette specifically", () => {
  it("never paints the canvas pure black", () => {
    expect(oklch(dark["--pc-bg"])[0][0]).toBeGreaterThan(0.1)
  })

  it("raises chroma as the surface darkens", () => {
    const bg = oklch(dark["--pc-bg"])[0]
    const text = oklch(dark["--pc-text"])[0]
    expect(bg[0]).toBeLessThan(text[0])
    expect(bg[1]).toBeGreaterThan(text[1])
  })

  // The accent sits at L 0.70, so white ink on it would fail contrast. The
  // token exists precisely so no component has to think about that.
  it("puts dark ink on the bright accent", () => {
    expect(oklch(dark["--pc-primary"])[0][0]).toBeCloseTo(0.7, 2)
    expect(oklch(dark["--pc-primary-fg"])[0][0]).toBeLessThan(0.3)
  })

  it("inverts the ink rule in light mode, where the accent is dark", () => {
    expect(oklch(light["--pc-primary"])[0][0]).toBeLessThan(0.6)
    expect(oklch(light["--pc-primary-fg"])[0][0]).toBeGreaterThan(0.9)
  })

  it("orders the chart sequence on the hue spine", () => {
    expect(oklch(dark["--chart-1"])[0][2]).toBe(208)
    expect(oklch(dark["--chart-2"])[0][2]).toBe(195)
  })
})

// Phase 1 was reviewed in dark mode. Light mode has to be the same product,
// not a white shadcn default that happens to be legible.
describe("light mode is designed, not defaulted", () => {
  it("separates the canvas from the surfaces that sit on it", () => {
    const canvas = oklch(light["--pc-bg"])[0][0]
    const surface = light["--pc-surface-1"]
    // Cards lift by being lighter than the canvas, which is the inversion of
    // the dark-mode rule and the reason both modes read as deliberate.
    expect(surface).toContain("oklch(1 0 0)")
    expect(canvas).toBeLessThan(1)
    expect(canvas).toBeGreaterThan(0.95)
  })

  it("steps its surfaces monotonically away from white", () => {
    const steps = [
      "--pc-surface-1",
      "--pc-surface-2",
      "--pc-surface-3",
      "--pc-border",
      "--pc-border-strong",
    ].map((token) => oklch(light[token])[0][0])
    for (let i = 1; i < steps.length; i++) {
      expect(steps[i], `${i} is not darker than ${i - 1}`).toBeLessThan(
        steps[i - 1],
      )
    }
  })

  it("keeps every text role clearly darker than every surface", () => {
    const darkestSurface = oklch(light["--pc-surface-3"])[0][0]
    for (const token of ["--pc-text", "--pc-text-muted", "--pc-text-faint"]) {
      expect(
        oklch(light[token])[0][0],
        `${token} is not dark enough for a light surface`,
      ).toBeLessThan(darkestSurface - 0.3)
    }
  })

  it("darkens the accent rather than reusing the dark-mode one", () => {
    // Reusing the dark accent on white is the single most common way a light
    // theme ends up looking unfinished.
    expect(light["--pc-primary"]).not.toBe(dark["--pc-primary"])
    expect(oklch(light["--pc-primary"])[0][0]).toBeLessThan(
      oklch(dark["--pc-primary"])[0][0],
    )
  })

  it("keeps the status colours dark enough to read on a light surface", () => {
    for (const token of [
      "--pc-signal",
      "--pc-success",
      "--pc-warning",
      "--pc-danger",
    ]) {
      const lightness = oklch(light[token])[0][0]
      expect(lightness, `${token} is too pale on white`).toBeLessThan(0.7)
      expect(lightness, `${token} is too dark to read as a status`).toBeGreaterThan(0.4)
    }
  })

  it("carries less chroma than dark mode, as the palette says it should", () => {
    const lightChroma = oklch(light["--pc-surface-3"])[0][1]
    const darkChroma = oklch(dark["--pc-surface-3"])[0][1]
    expect(lightChroma).toBeLessThan(darkChroma)
  })

  it("draws its own hairline shadows rather than reusing the dark ones", () => {
    expect(light["--pc-shadow-3"]).not.toBe(dark["--pc-shadow-3"])
    expect(light["--pc-shadow-3"]).toContain("oklch(0.22 0.015 245")
  })
})

describe("the shared devices", () => {
  // The bracket, the rail and the sheet entry are all logical edges, which is
  // why the whole system mirrors in Arabic with no locale branch anywhere.
  it("draws the bracket and the assistant rail on a logical edge", () => {
    expect(CSS).toMatch(/\.pc-bracket\s*{\s*border-inline-start/)
    expect(CSS).toMatch(/\.pc-rail\s*{\s*border-inline-start/)
    expect(CSS).not.toMatch(/\.pc-(bracket|rail)\s*{\s*border-left/)
  })

  it("keeps a visible focus ring rather than removing the outline", () => {
    expect(CSS).toContain(":focus-visible")
    expect(CSS).toContain("outline: 2px solid var(--pc-focus)")
    expect(CSS).not.toMatch(/outline:\s*none/)
  })

  it("stops the live-state pulse under reduced motion", () => {
    const reduced = CSS.slice(CSS.indexOf("prefers-reduced-motion"))
    expect(reduced).toContain(".pc-signal-pulse")
    expect(reduced).toContain("animation: none")
  })

  it("lands buttons, cards and the composer on the Aperture radii", () => {
    expect(CSS).toContain("--radius-md: 10px")
    expect(CSS).toContain("--radius-xl: 14px")
    expect(CSS).toContain("--radius-2xl: 20px")
  })
})
