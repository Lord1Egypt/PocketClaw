import { render, screen } from "@testing-library/react"
import { describe, expect, it } from "vitest"

import { ConfigChangeNotice } from "./config-change-notice"

/**
 * PC-DEF-033. The notice rendered as an empty amber rectangle on every screen
 * that raises it — the Add Model and Edit Model sheets, the settings pages and
 * the channel config page — because it painted `text-pc-warning` on
 * `bg-pc-warning`: one token, used for both the surface and the label.
 *
 * The regression is a colour identity, not a layout, so that is what is
 * asserted: the background token must be the soft variant, and the foreground
 * token must never be the same token as the background.
 */
describe("ConfigChangeNotice", () => {
  const kinds = ["save", "restart"] as const

  it.each(kinds)("renders its title and description for kind=%s", (kind) => {
    render(
      <ConfigChangeNotice
        kind={kind}
        title="Unsaved changes"
        description="Save to write this into the model configuration."
      />,
    )

    expect(screen.getByText("Unsaved changes")).toBeTruthy()
    expect(
      screen.getByText("Save to write this into the model configuration."),
    ).toBeTruthy()
  })

  it.each(kinds)(
    "never paints the label in the background colour for kind=%s",
    (kind) => {
      const { container } = render(
        <ConfigChangeNotice kind={kind} title="Unsaved changes" />,
      )

      const notice = container.firstElementChild
      expect(notice).toBeTruthy()
      const classes = (notice as HTMLElement).className.split(/\s+/)

      expect(classes).toContain("bg-pc-warning-soft")
      expect(classes).toContain("text-pc-warning")
      // The solid token as a background is the defect itself.
      expect(classes).not.toContain("bg-pc-warning")
    },
  )

  it("keeps a caller-supplied class without losing the palette", () => {
    const { container } = render(
      <ConfigChangeNotice kind="save" title="Unsaved changes" className="mb-4" />,
    )

    const classes = (container.firstElementChild as HTMLElement).className
    expect(classes).toContain("mb-4")
    expect(classes).toContain("bg-pc-warning-soft")
  })
})
