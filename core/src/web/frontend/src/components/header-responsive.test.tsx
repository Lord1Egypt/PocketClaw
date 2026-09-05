/**
 * The header's responsive hierarchy.
 *
 * At phone width the toolbar was carrying a menu button, a mark, a gateway
 * control and four icon buttons in one row, so nothing in it read as more
 * important than anything else. The utilities now collect behind one control
 * and the gateway keeps its place — nothing is removed, and every action is
 * still reachable and announced.
 */
import fsSync from "node:fs"

import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import userEvent from "@testing-library/user-event"
import { beforeEach, describe, expect, it, vi } from "vitest"

import i18n from "@/i18n"

import { HeaderUtilityMenu } from "./header-utility-menu"

const HEADER = fsSync.readFileSync(
  `${process.cwd()}/src/components/app-header.tsx`,
  "utf8",
)

function renderMenu(overrides: Partial<Parameters<typeof HeaderUtilityMenu>[0]> = {}) {
  const props = {
    theme: "dark",
    onToggleTheme: vi.fn(),
    onLogout: vi.fn(),
    onRestartGateway: vi.fn(),
    restartRequired: false,
    restartDisabled: false,
    ...overrides,
  }
  render(<HeaderUtilityMenu {...props} />)
  return props
}

describe("the overflow menu is a real menu", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en")
  })

  it("names its trigger, so it is not an unlabelled glyph", () => {
    renderMenu()
    expect(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    ).toBeTruthy()
  })

  it("opens from the keyboard and exposes its items", async () => {
    const user = userEvent.setup()
    renderMenu()
    const trigger = screen.getByRole("button", {
      name: i18n.t("header.moreActions"),
    })
    trigger.focus()
    await user.keyboard("{Enter}")

    await waitFor(() => expect(screen.getByRole("menu")).toBeTruthy())
    expect(
      screen.getByRole("menuitem", { name: i18n.t("common.lightMode") }),
    ).toBeTruthy()
    expect(
      screen.getByRole("menuitem", { name: i18n.t("header.logout.tooltip") }),
    ).toBeTruthy()
  })

  it("reaches every language without leaving the menu", async () => {
    const user = userEvent.setup()
    renderMenu()
    await user.click(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    )
    await waitFor(() => expect(screen.getByRole("menu")).toBeTruthy())
    expect(
      screen.getByRole("menuitem", { name: i18n.t("common.language") }),
    ).toBeTruthy()
  })

  it("returns focus to the trigger when it closes", async () => {
    const user = userEvent.setup()
    renderMenu()
    const trigger = screen.getByRole("button", {
      name: i18n.t("header.moreActions"),
    })
    await user.click(trigger)
    await waitFor(() => expect(screen.getByRole("menu")).toBeTruthy())

    fireEvent.keyDown(document.activeElement ?? document.body, {
      key: "Escape",
    })
    await waitFor(() => expect(document.activeElement).toBe(trigger))
  })

  it("offers restart only when a restart is actually required", async () => {
    const user = userEvent.setup()
    const view = render(
      <HeaderUtilityMenu
        theme="dark"
        onToggleTheme={vi.fn()}
        onLogout={vi.fn()}
        onRestartGateway={vi.fn()}
        restartRequired={false}
        restartDisabled={false}
      />,
    )
    await user.click(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    )
    await waitFor(() => expect(screen.getByRole("menu")).toBeTruthy())
    expect(
      screen.queryByRole("menuitem", {
        name: i18n.t("header.gateway.action.restart"),
      }),
    ).toBeNull()
    view.unmount()

    renderMenu({ restartRequired: true })
    await user.click(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    )
    await waitFor(() =>
      expect(
        screen.getByRole("menuitem", {
          name: i18n.t("header.gateway.action.restart"),
        }),
      ).toBeTruthy(),
    )
  })

  it("actually runs the action it names", async () => {
    const user = userEvent.setup()
    const props = renderMenu()
    await user.click(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    )
    await waitFor(() => expect(screen.getByRole("menu")).toBeTruthy())
    await user.click(
      screen.getByRole("menuitem", { name: i18n.t("common.lightMode") }),
    )
    expect(props.onToggleTheme).toHaveBeenCalledTimes(1)
  })

  it.each(["ar", "de", "ja"])("is labelled in %s", async (locale) => {
    await i18n.changeLanguage(locale)
    expect(i18n.t("header.moreActions")).not.toBe("header.moreActions")
    renderMenu()
    expect(
      screen.getByRole("button", { name: i18n.t("header.moreActions") }),
    ).toBeTruthy()
  })
})

describe("the toolbar splits by width, it does not drop actions", () => {
  it("hides the spelled-out utilities only where the overflow appears", () => {
    // The two must be exact complements: one `sm:` boundary, both directions.
    expect(HEADER).toContain('className="hidden items-center gap-1 sm:flex"')
    expect(HEADER).toContain('className="size-11 sm:hidden"')
  })

  it("keeps every utility reachable at both widths", () => {
    for (const control of [
      "LanguageMenu",
      "HeaderUtilityMenu",
      "IconSun",
      "IconLogout",
    ]) {
      expect(HEADER, control).toContain(control)
    }
  })

  it("keeps the gateway control outside the overflow at every width", () => {
    // Runtime state is the one thing the narrow toolbar must still say.
    expect(HEADER).toContain('data-tour="gateway-button"')
    const overflow = HEADER.slice(HEADER.indexOf("<HeaderUtilityMenu"))
    expect(overflow).not.toContain("gateway-button")
  })

  it("drops the gateway label rather than the gateway control when narrow", () => {
    expect(HEADER).toContain('className="hidden text-xs font-semibold sm:inline"')
  })

  it("gives the mobile menu button a 44px hit area", () => {
    const trigger =
      HEADER.match(/<SidebarTrigger[\s\S]*?<\/SidebarTrigger>/)?.[0] ?? ""
    expect(trigger).toContain("size-11")
  })

  it("uses no physical direction in the header chrome", () => {
    expect(HEADER).not.toMatch(/\b(ml|mr|pl|pr)-\d/)
    expect(HEADER).not.toMatch(/\b(left|right)-\d/)
    expect(HEADER).not.toContain("border-l")
    expect(HEADER).not.toContain("border-r")
  })
})
