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

  it("renders the brand as inline SVG, not as an image file", () => {
    const sidebar = fsSync.readFileSync(
      `${process.cwd()}/src/components/app-sidebar.tsx`,
      "utf8",
    )
    expect(sidebar).not.toContain("logo_with_text")
    expect(sidebar).not.toMatch(/<img\b/)
    expect(sidebar).toContain("PocketClawLockup")
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
    // The lipped mouth: the optical fix that stops the mark reading as a
    // crown below 24px. If this path changes, the rasters must be
    // regenerated from the same geometry in the same commit.
    expect(svg).toContain("M7.5 12H4v8h16v-8h-3.5")
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

describe("one geometry, three copies", () => {
  // The component, the SVG favicon and the raster pipeline are three copies of
  // one geometry. They drifted the moment the mark was refined, so hold them
  // to the same three paths.
  it("keeps the component, the favicon and the raster script on one geometry", () => {
    const paths = ["M7.5 12H4v8h16v-8h-3.5", "M10 13.5V10", "M14 13.5V10"]
    const component = fsSync.readFileSync(
      `${process.cwd()}/src/components/brand/pocketclaw-mark.tsx`,
      "utf8",
    )
    const svg = fsSync.readFileSync(`${PUBLIC}/favicon.svg`, "utf8")
    const script = fsSync.readFileSync(
      `${process.cwd()}/scripts/generate-brand-assets.py`,
      "utf8",
    )
    for (const path of paths) {
      expect(component, `component is missing ${path}`).toContain(path)
      expect(svg, `favicon.svg is missing ${path}`).toContain(path)
    }
    // The script draws polylines rather than path data, so assert the two
    // numbers the refinement actually moved.
    expect(script).toContain("(7.5, 12)")
    expect(script).toContain("(10, 13.5)")
  })
})

describe("the brand is anchored once", () => {
  const header = fsSync.readFileSync(
    `${process.cwd()}/src/components/app-header.tsx`,
    "utf8",
  )
  const sidebar = fsSync.readFileSync(
    `${process.cwd()}/src/components/app-sidebar.tsx`,
    "utf8",
  )

  // Desktop showed PocketClaw twice: once in the top chrome and again in the
  // sidebar header directly beneath it. The sidebar is the persistent anchor;
  // the top bar is utility chrome.
  it("puts the full lockup in the sidebar and nowhere else in the shell", () => {
    expect(sidebar).toContain("PocketClawLockup")
    expect(header).not.toContain("PocketClawLockup")
  })

  // A mark-only presence in the header is allowed exactly where it earns its
  // place: phone width, where the sidebar holding the lockup is off-canvas.
  it("keeps a mark in the header only at phone width", () => {
    const brandLink = header.match(
      /<Link[\s\S]*?PocketClawMark[\s\S]*?<\/Link>/,
    )?.[0]
    expect(brandLink, "the header no longer renders the mark").toBeTruthy()
    expect(brandLink).toContain("sm:hidden")
  })

  it("stops offsetting the sidebar beneath a full-width header", () => {
    const css = fsSync.readFileSync(`${process.cwd()}/src/index.css`, "utf8")
    expect(css).not.toContain('[data-slot="sidebar-container"]')
    expect(css).not.toContain("100svh - 3.5rem")
  })

  it("nests the header inside the content column, beside the sidebar", () => {
    const layout = fsSync.readFileSync(
      `${process.cwd()}/src/components/app-layout.tsx`,
      "utf8",
    )
    expect(layout.indexOf("<AppSidebar />")).toBeLessThan(
      layout.indexOf("<AppHeader />"),
    )
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
