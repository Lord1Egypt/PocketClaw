/**
 * The mobile menu button.
 *
 * The header already asked for a hamburger — it passes `<IconMenu2 />` to
 * SidebarTrigger — but the trigger hard-coded `<IconLayoutSidebar />` between
 * its own tags. JSX children written between the tags win over children
 * arriving through a spread, so the icon the header supplied was silently
 * discarded and a panel-layout glyph rendered in its place. The control was
 * authored correctly and swallowed by the component.
 *
 * Desktop is not required to match: the sidebar there genuinely is a
 * collapsible panel, so SidebarRail keeps the panel glyph and the panel
 * wording. Mobile navigates; desktop collapses.
 */
import fsSync from "node:fs"

import { fireEvent, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import i18n from "@/i18n"

const isMobile = vi.fn(() => true)
vi.mock("@/hooks/use-mobile", () => ({ useIsMobile: () => isMobile() }))

const { Sidebar, SidebarProvider, SidebarTrigger } =
  await import("./ui/sidebar")

const HEADER_SOURCE = fsSync.readFileSync(
  `${process.cwd()}/src/components/app-header.tsx`,
  "utf8",
)

describe("SidebarTrigger renders the icon it is handed", () => {
  beforeEach(async () => {
    isMobile.mockReturnValue(true)
    await i18n.changeLanguage("en")
  })

  it("uses the supplied child instead of its own glyph", () => {
    render(
      <SidebarProvider>
        <SidebarTrigger>
          <svg data-testid="supplied-icon" />
        </SidebarTrigger>
      </SidebarProvider>,
    )
    expect(screen.getByTestId("supplied-icon")).toBeTruthy()
  })

  it("still falls back to the panel glyph when handed nothing", () => {
    render(
      <SidebarProvider>
        <SidebarTrigger />
      </SidebarProvider>,
    )
    const button = screen.getByRole("button", {
      name: i18n.t("common.toggleSidebar"),
    })
    expect(button.querySelector("svg")).toBeTruthy()
  })

  it("keeps toggling the drawer regardless of the icon", () => {
    render(
      <SidebarProvider>
        <Sidebar>
          <p>sidebar content</p>
        </Sidebar>
        <SidebarTrigger label="Open menu">
          <svg data-testid="supplied-icon" />
        </SidebarTrigger>
      </SidebarProvider>,
    )
    const openDrawer = () =>
      document.querySelector('[data-slot="sidebar"][data-mobile="true"]')

    expect(openDrawer()).toBeNull()
    fireEvent.click(screen.getByRole("button", { name: "Open menu" }))
    expect(openDrawer()).not.toBeNull()
  })
})

describe("the header's mobile control is a menu", () => {
  it("hands the trigger the hamburger, not a layout glyph", () => {
    expect(HEADER_SOURCE).toContain("<IconMenu2 />")
    expect(HEADER_SOURCE).not.toContain("IconLayoutSidebar")
  })

  // 36px failed the touch-target minimum. The glyph stays 20px; the hit area
  // is what grew.
  it("gives it a 44x44 hit area with a 20px glyph", () => {
    const trigger =
      HEADER_SOURCE.match(
        /<SidebarTrigger[\s\S]*?<\/SidebarTrigger>/,
      )?.[0] ?? ""
    expect(trigger).toContain("size-11")
    expect(trigger).toContain("[&>svg]:size-5")
    expect(trigger).not.toMatch(/\bh-9 w-9\b/)
  })

  it("announces opening and closing a menu, not toggling a panel", () => {
    expect(HEADER_SOURCE).toContain('t("common.openMenu")')
    expect(HEADER_SOURCE).toContain('t("common.closeMenu")')
  })

  it.each(["en", "ar", "de", "ja"])(
    "has a translated menu label in %s",
    async (locale) => {
      await i18n.changeLanguage(locale)
      for (const key of ["common.openMenu", "common.closeMenu"]) {
        expect(i18n.t(key), `${locale} ${key}`).not.toBe(key)
      }
    },
  )
})

describe("the desktop rail keeps its own semantics", () => {
  it("still describes itself as a sidebar toggle", () => {
    const sidebar = fsSync.readFileSync(
      `${process.cwd()}/src/components/ui/sidebar.tsx`,
      "utf8",
    )
    const rail = sidebar.slice(sidebar.indexOf("function SidebarRail"))
    expect(rail).toContain('t("common.toggleSidebar")')
  })
})
