import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import type { ModelInfo } from "@/api/models"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

import { TooltipProvider } from "@/components/ui/tooltip"

import { ModelCard } from "./model-card"

function model(overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 2,
    model_name: "opencode-deepseek",
    provider: "opencode_go",
    model: "deepseek-v4.1-flash",
    api_key: "sk-****abcd",
    enabled: true,
    available: true,
    status: "available",
    is_default: false,
    is_virtual: false,
    default_model_allowed: true,
    ...overrides,
  }
}

function renderCard(overrides: Partial<ModelInfo> = {}) {
  const onEdit = vi.fn()
  const onDelete = vi.fn()
  const onSetDefault = vi.fn()
  render(
    <TooltipProvider>
      <ModelCard
        model={model(overrides)}
        onEdit={onEdit}
        onDelete={onDelete}
        onSetDefault={onSetDefault}
        settingDefault={false}
      />
    </TooltipProvider>,
  )
  return { onEdit, onDelete, onSetDefault }
}

/**
 * PC-DEF-046. The owner reported from the device that there was no obvious way
 * to delete a model. There was one: a 32px grey trash glyph packed beside the
 * edit and set-default glyphs in the card's top-right corner, next to a
 * truncating title, explained only by a tooltip — and a tooltip never opens on
 * a touch screen.
 *
 * These assert the affordance is readable and operable, not merely present in
 * the DOM, because present-in-the-DOM was already true when it was reported
 * missing.
 */
describe("ModelCard actions", () => {
  it("labels Edit and Delete in text, not only as an icon", () => {
    renderCard()

    const edit = screen.getByText("models.action.edit").closest("button")
    const remove = screen.getByText("models.action.delete").closest("button")

    expect(edit, "Edit must carry a visible text label").toBeTruthy()
    expect(remove, "Delete must carry a visible text label").toBeTruthy()
  })

  it("invokes the handlers when the labelled controls are pressed", () => {
    const { onEdit, onDelete } = renderCard()

    fireEvent.click(
      screen.getByText("models.action.edit").closest("button") as HTMLElement,
    )
    expect(onEdit).toHaveBeenCalled()

    fireEvent.click(
      screen.getByText("models.action.delete").closest("button") as HTMLElement,
    )
    expect(onDelete).toHaveBeenCalled()
  })

  // A tooltip is not an explanation on a phone, so the reason a disabled
  // control is disabled has to be on the card itself.
  it("explains a disabled Delete in visible text", () => {
    renderCard({ is_default: true })

    const remove = screen
      .getByText("models.action.delete")
      .closest("button") as HTMLButtonElement
    expect(remove.disabled).toBe(true)
    expect(
      screen.getByText("models.action.deleteDisabled.isDefault"),
      "the reason must be readable without hovering",
    ).toBeTruthy()
  })

  it("keeps both actions comfortably tappable", () => {
    renderCard()

    for (const label of ["models.action.edit", "models.action.delete"]) {
      const button = screen.getByText(label).closest("button") as HTMLElement
      // 32px icon-only buttons are what this replaced.
      expect(
        button.className,
        `${label} must not go back to a cramped target`,
      ).toContain("min-h-10")
    }
  })
})
