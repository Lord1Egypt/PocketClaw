/**
 * PC-DEF-089. On a 411 dp phone the Models header — title, "Saved Catalogs"
 * and "Add Provider" — sat in one fixed-height row that could not wrap, and
 * "Add Provider" was clipped at the edge. jsdom has no layout, so this pins
 * the contract that prevents it: the header and its actions wrap, nothing
 * fixes the row height, and the actions align to the logical end so RTL
 * mirrors them.
 */
import fsSync from "node:fs"

import { render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

// matchMedia does not exist in jsdom, and the breakpoint is not under test:
// this is the phone layout.
vi.mock("@/hooks/use-mobile", () => ({ useIsMobile: () => true }))

const { SidebarProvider } = await import("@/components/ui/sidebar")
const { PageHeader } = await import("./page-header")

function renderHeader(dir: "ltr" | "rtl" = "ltr") {
  render(
    <div dir={dir}>
      <SidebarProvider>
        <PageHeader title="Models">
          <button type="button">Saved Catalogs</button>
          <button type="button">Add Provider</button>
        </PageHeader>
      </SidebarProvider>
    </div>,
  )
  const title = screen.getByRole("heading", { name: "Models" })
  const header = title.closest("div.z-40") as HTMLElement
  const actions = screen.getByRole("button", { name: "Add Provider" })
    .parentElement as HTMLElement
  return { header, actions }
}

describe("the page header at phone width", () => {
  it("wraps instead of clipping, and has no fixed height", () => {
    const { header } = renderHeader()
    const classes = header.className.split(/\s+/)
    expect(classes).toContain("flex-wrap")
    expect(classes).toContain("min-h-14")
    expect(classes).not.toContain("h-14")
  })

  it("keeps every action rendered and lets the action row wrap too", () => {
    const { actions } = renderHeader()
    expect(actions.className.split(/\s+/)).toContain("flex-wrap")
    expect(screen.getByRole("button", { name: "Saved Catalogs" })).toBeTruthy()
    expect(screen.getByRole("button", { name: "Add Provider" })).toBeTruthy()
  })

  it("aligns a wrapped action row to the logical end, so RTL mirrors it", () => {
    const { actions } = renderHeader("rtl")
    const classes = actions.className.split(/\s+/)
    expect(classes).toContain("ms-auto")
    expect(classes.some((c) => /^(ml|mr)-auto$/.test(c))).toBe(false)
  })

  it("the Models page's own action group wraps as well", () => {
    const source = fsSync.readFileSync(
      `${process.cwd()}/src/components/models/models-page.tsx`,
      "utf8",
    )
    expect(source).toMatch(
      /<PageHeader title=\{t\("navigation\.models"\)\}>\s*<div className="flex flex-wrap items-center gap-3">/,
    )
  })
})
