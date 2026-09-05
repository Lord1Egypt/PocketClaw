/**
 * Which edge the sidebar drawer anchors to.
 *
 * Physically observed on vc34: Arabic rendered right-to-left correctly and the
 * trigger sat at the top right, but tapping it slid the drawer in from the
 * left — away from the finger that opened it. The side was a hard-coded
 * `"left"` default on the shared Sidebar, so no page could have been at fault
 * and no page needed patching.
 *
 * These drive the real components the way a user does: tap the trigger, then
 * read the side the drawer reports.
 */
import fsSync from "node:fs"

import { fireEvent, render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import i18n from "@/i18n"

// The reported failure is the mobile drawer. matchMedia does not exist in
// jsdom, and the breakpoint is not what is under test.
const isMobile = vi.fn(() => true)
vi.mock("@/hooks/use-mobile", () => ({ useIsMobile: () => isMobile() }))

const { Sidebar, SidebarProvider, SidebarTrigger } =
  await import("./ui/sidebar")

function renderSidebar() {
  return render(
    <SidebarProvider>
      <Sidebar>
        <p>sidebar content</p>
      </Sidebar>
      <SidebarTrigger />
    </SidebarProvider>,
  )
}

/// Taps the trigger the app header renders.
function tapTrigger() {
  fireEvent.click(
    screen.getByRole("button", { name: i18n.t("common.toggleSidebar") }),
  )
}

/// The edge the open mobile drawer anchored to, read off the rendered node.
function drawerSide() {
  const drawer = document.querySelector(
    '[data-slot="sidebar"][data-mobile="true"]',
  )
  return drawer?.getAttribute("data-side") ?? null
}

describe("sidebar drawer side follows the writing direction", () => {
  beforeEach(async () => {
    isMobile.mockReturnValue(true)
    await i18n.changeLanguage("en")
  })

  it("opens from the right in Arabic", async () => {
    await i18n.changeLanguage("ar")
    renderSidebar()
    tapTrigger()

    expect(i18n.dir()).toBe("rtl")
    expect(drawerSide()).toBe("right")
  })

  it.each(["de", "en"])("opens from the left in %s", async (locale) => {
    await i18n.changeLanguage(locale)
    renderSidebar()
    tapTrigger()

    expect(i18n.dir()).toBe("ltr")
    expect(drawerSide()).toBe("left")
  })

  // The drawer is open while the language changes: the anchor has to move with
  // it, not stay where it was when the drawer mounted.
  it("moves back to the left when Arabic is switched to German", async () => {
    await i18n.changeLanguage("ar")
    renderSidebar()
    tapTrigger()
    expect(drawerSide()).toBe("right")

    await i18n.changeLanguage("de")
    expect(drawerSide()).toBe("left")
  })

  it("moves to the right when German is switched to Arabic", async () => {
    await i18n.changeLanguage("de")
    renderSidebar()
    tapTrigger()
    expect(drawerSide()).toBe("left")

    await i18n.changeLanguage("ar")
    expect(drawerSide()).toBe("right")
  })

  it("still lets a caller pin the side explicitly", async () => {
    await i18n.changeLanguage("ar")
    render(
      <SidebarProvider>
        <Sidebar side="left">
          <p>sidebar content</p>
        </Sidebar>
        <SidebarTrigger />
      </SidebarProvider>,
    )
    tapTrigger()

    expect(drawerSide()).toBe("left")
  })
})

describe("the desktop rail follows the same direction", () => {
  beforeEach(async () => {
    isMobile.mockReturnValue(false)
    await i18n.changeLanguage("en")
  })

  function containerSide() {
    return document
      .querySelector('[data-slot="sidebar-container"]')
      ?.getAttribute("data-side")
  }

  it("anchors right in Arabic and left in German", async () => {
    await i18n.changeLanguage("ar")
    const view = renderSidebar()
    expect(containerSide()).toBe("right")

    await i18n.changeLanguage("de")
    expect(containerSide()).toBe("left")
    view.unmount()
  })
})

describe("sidebar chrome uses a logical border edge", () => {
  beforeEach(async () => {
    isMobile.mockReturnValue(false)
    await i18n.changeLanguage("en")
  })

  // The border faces the content: the sidebar's right edge in LTR, its left
  // edge in RTL. Both are the inline-end edge, so one logical class is correct
  // in both directions and no language check is needed. A physical border-r
  // would sit on the outer screen edge once the sidebar moves right.
  it("declares the border on the inline-end edge, not a physical one", () => {
    const source = fsSync.readFileSync(
      `${process.cwd()}/src/components/app-sidebar.tsx`,
      "utf8",
    )
    // Read the chrome off the <Sidebar> element itself rather than off one
    // colour utility, so restyling the surface cannot quietly empty the match
    // and turn this into an assertion about nothing.
    const chrome =
      source.match(/<Sidebar\b[^>]*className="([^"]*)"/s)?.[1] ?? ""
    expect(chrome).not.toBe("")
    expect(chrome).toContain("border-e")
    expect(chrome).not.toMatch(/border-[rl]\b/)
    expect(chrome).not.toMatch(/border-[rl]-/)
  })

  // The chrome class rides on the same node that carries the side, so a change
  // to one must not disturb the other.
  it.each([
    ["ar", "right"],
    ["de", "left"],
  ])(
    "keeps placement and chrome together in %s",
    async (locale, expectedSide) => {
      await i18n.changeLanguage(locale)
      render(
        <SidebarProvider>
          <Sidebar className="bg-pc-surface-1 border-e-border border-e">
            <p>sidebar content</p>
          </Sidebar>
        </SidebarProvider>,
      )

      const container = document.querySelector(
        '[data-slot="sidebar-container"]',
      )
      expect(container?.getAttribute("data-side")).toBe(expectedSide)
      expect(container?.className).toContain("border-e")
      expect(container?.className).not.toMatch(/\bborder-r\b/)
    },
  )
})

// The Phase 2 header keeps the drawer contract it inherited: the trigger sits
// at the inline-start edge and the drawer enters from the side the writing
// direction puts it. A hamburger is direction-neutral and needs no mirroring;
// what must not drift is the placement around it.
describe("the mobile header stays RTL-safe", () => {
  beforeEach(async () => {
    isMobile.mockReturnValue(true)
    await i18n.changeLanguage("en")
  })

  const header = fsSync.readFileSync(
    `${process.cwd()}/src/components/app-header.tsx`,
    "utf8",
  )

  it("declares no physical direction in the header chrome", () => {
    expect(header).not.toMatch(/\b(ml|mr|pl|pr)-\d/)
    expect(header).not.toMatch(/\bmx-4\b.*orientation="vertical"/)
    expect(header).not.toContain("left-1/2")
  })

  it("keeps the drawer on the writing direction's side with the new header", async () => {
    for (const [locale, side] of [
      ["ar", "right"],
      ["en", "left"],
    ] as const) {
      await i18n.changeLanguage(locale)
      const view = render(
        <SidebarProvider>
          <Sidebar>
            <p>sidebar content</p>
          </Sidebar>
          <SidebarTrigger label="menu" />
        </SidebarProvider>,
      )
      fireEvent.click(screen.getByRole("button", { name: "menu" }))
      expect(drawerSide(), locale).toBe(side)
      view.unmount()
    }
  })

  // The overflow menu anchors to the inline-end edge in both directions,
  // which Radix resolves from the document dir rather than from a locale test.
  it("anchors the utility menu to an edge, not to a hard-coded side", () => {
    const menu = fsSync.readFileSync(
      `${process.cwd()}/src/components/header-utility-menu.tsx`,
      "utf8",
    )
    expect(menu).toContain('align="end"')
    expect(menu).not.toMatch(/align="(left|right)"/)
  })
})

// Changing the sidebar's own chrome must not reach the sheets that share the
// primitive but anchor themselves.
describe("unrelated sheets are untouched", () => {
  it.each([
    "src/components/agent/skills/detail-sheet.tsx",
    "src/components/credentials/device-code-sheet.tsx",
    "src/components/models/edit-model-sheet.tsx",
    "src/components/models/add-model-sheet.tsx",
  ])("leaves %s deciding its own side", (file) => {
    const source = fsSync.readFileSync(`${process.cwd()}/${file}`, "utf8")
    // They must not have grown a dependency on the sidebar's direction logic.
    expect(source).not.toContain("resolvedSide")
    expect(source).not.toContain("@/components/ui/sidebar")
  })
})
