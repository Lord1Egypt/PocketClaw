import { render, screen, within } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import { ModelChipGroup } from "./model-chip-group"

/**
 * PC-DEF-041. Three sources of model ids were rendered as three identical,
 * unlabelled rows of chips: the provider preset's curated names, a cached
 * result from an earlier Fetch Models, and this session's live result. A user
 * could not tell a name this build happens to know from one the provider had
 * just confirmed it serves.
 */
describe("ModelChipGroup", () => {
  it("labels every row so its provenance is readable", () => {
    const { container } = render(
      <ModelChipGroup
        origin="suggestion"
        label="Suggestions"
        hint="Not confirmed against your account."
        models={["deepseek-v4-flash"]}
        selected=""
        onSelect={vi.fn()}
      />,
    )

    expect(screen.getByText("Suggestions")).toBeTruthy()
    expect(screen.getByText("Not confirmed against your account.")).toBeTruthy()
    expect(
      container.querySelector('[data-origin="suggestion"]'),
      "the row must declare where its ids came from",
    ).toBeTruthy()
  })

  it("renders nothing rather than an empty labelled row", () => {
    const { container } = render(
      <ModelChipGroup
        origin="verified"
        label="Verified available"
        models={[]}
        selected=""
        onSelect={vi.fn()}
      />,
    )
    expect(container.firstChild).toBeNull()
  })

  it("does not present a selected suggestion as a confirmed choice", () => {
    const { container } = render(
      <ModelChipGroup
        origin="suggestion"
        label="Suggestions"
        models={["deepseek-v4-flash"]}
        selected="deepseek-v4-flash"
        onSelect={vi.fn()}
      />,
    )

    const chip = within(container as HTMLElement).getByText("deepseek-v4-flash")
    // The "default" badge variant is the confirmed-selection look, reserved
    // for ids a provider actually returned.
    expect(chip.className).not.toContain("bg-primary")
  })

  it("passes the exact id back, never a normalised one", () => {
    const onSelect = vi.fn()
    render(
      <ModelChipGroup
        origin="verified"
        label="Verified available"
        models={["deepseek-v4.1-flash"]}
        selected=""
        onSelect={onSelect}
      />,
    )

    screen.getByText("deepseek-v4.1-flash").click()
    expect(onSelect).toHaveBeenCalledWith("deepseek-v4.1-flash")
  })
})
