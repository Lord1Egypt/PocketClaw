/**
 * The visible brand.
 *
 * The desktop dashboard shipped the PicoClaw lobster and a PICOCLAW wordmark
 * as `public/logo_with_text.png`, rendered in the header under an `alt` that
 * already claimed to be PocketClaw — the image and its own description
 * disagreed. Every favicon was the same lobster and the manifest had never
 * been branded at all, still carrying the "MyWebSite" boilerplate.
 *
 * These assert the user-visible surfaces only. The internal names — the
 * `picoclaw-web` package id, the localStorage keys, the Go paths, the `.so`
 * filenames — are load-bearing and deliberately untouched; renaming them
 * discards saved preferences and needs its own migration.
 */
import fsSync from "node:fs"

import { describe, expect, it } from "vitest"

const PUBLIC = `${process.cwd()}/public`

describe("the PicoClaw lobster is gone from the shipped assets", () => {
  it("no longer ships the PICOCLAW wordmark image", () => {
    expect(fsSync.existsSync(`${PUBLIC}/logo_with_text.png`)).toBe(false)
  })

  it("renders the header brand as inline SVG, not as an image file", () => {
    const header = fsSync.readFileSync(
      `${process.cwd()}/src/components/app-header.tsx`,
      "utf8",
    )
    expect(header).not.toContain("logo_with_text")
    expect(header).not.toMatch(/<img\b/)
    expect(header).toContain("PocketClawLockup")
  })

  it("keeps the whole shipped source free of the old wordmark reference", () => {
    const hits: string[] = []
    const walk = (dir: string) => {
      for (const entry of fsSync.readdirSync(dir, { withFileTypes: true })) {
        const path = `${dir}/${entry.name}`
        if (entry.isDirectory()) {
          if (entry.name === "node_modules") continue
          walk(path)
          continue
        }
        if (!/\.(tsx?|css|html)$/.test(entry.name)) continue
        if (entry.name.endsWith(".test.tsx")) continue
        if (fsSync.readFileSync(path, "utf8").includes("logo_with_text")) {
          hits.push(path)
        }
      }
    }
    walk(`${process.cwd()}/src`)
    expect(hits).toEqual([])
  })

  // The Lark/Feishu glyph is a platform icon on a channel row, not branding.
  it("leaves the Lark platform icon in place", () => {
    expect(fsSync.existsSync(`${PUBLIC}/lark.svg`)).toBe(true)
  })
})

describe("the icon family is the PocketClaw mark", () => {
  it.each([
    "favicon.svg",
    "favicon.ico",
    "favicon-96x96.png",
    "apple-touch-icon.png",
    "web-app-manifest-192x192.png",
    "web-app-manifest-512x512.png",
  ])("ships %s", (name) => {
    expect(fsSync.existsSync(`${PUBLIC}/${name}`)).toBe(true)
  })

  // The lobster favicon was a 90 KB SVG. A mark that cannot be drawn small and
  // flat cannot function as a favicon, which was the underlying defect.
  it("draws the SVG favicon as the flat vector mark", () => {
    const svg = fsSync.readFileSync(`${PUBLIC}/favicon.svg`, "utf8")
    expect(svg.length).toBeLessThan(2048)
    expect(svg).toContain("<title>PocketClaw</title>")
    expect(svg).toContain("M4 12v8h16v-8")
    expect(svg).not.toMatch(/<image\b|base64/)
  })

  it("keeps the raster pipeline reproducible from the same geometry", () => {
    const script = fsSync.readFileSync(
      `${process.cwd()}/scripts/generate-brand-assets.py`,
      "utf8",
    )
    expect(script).toContain("favicon.ico")
    expect(script).toContain("web-app-manifest-512x512.png")
  })
})

describe("the web manifest identifies the product", () => {
  const manifest = JSON.parse(
    fsSync.readFileSync(`${PUBLIC}/site.webmanifest`, "utf8"),
  ) as {
    name: string
    short_name: string
    theme_color: string
    background_color: string
    icons: { src: string }[]
  }

  it("is named PocketClaw rather than the stale boilerplate", () => {
    expect(manifest.name).toBe("PocketClaw")
    expect(manifest.short_name).toBe("PocketClaw")
    expect(JSON.stringify(manifest)).not.toMatch(/MyWebSite|MySite|PicoClaw/i)
  })

  it("paints the install surface with the Aperture canvas, not white", () => {
    expect(manifest.theme_color).toBe("#0b1014")
    expect(manifest.background_color).toBe("#0b1014")
  })

  it("points every icon at a file that exists", () => {
    for (const icon of manifest.icons) {
      expect(
        fsSync.existsSync(`${PUBLIC}${icon.src}`),
        `${icon.src} is missing`,
      ).toBe(true)
    }
  })
})

describe("the document shell", () => {
  const html = fsSync.readFileSync(`${process.cwd()}/index.html`, "utf8")

  it("titles the page PocketClaw", () => {
    expect(html).toContain("<title>PocketClaw</title>")
  })

  it("declares a theme colour for each scheme", () => {
    expect(html).toContain('content="#0b1014"')
    expect(html).toContain('content="#f8fafc"')
  })
})
