import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ModelInfo } from "@/api/models"

const setModelFallbacks = vi.fn()
const saveAndApplyGatewayConfig = vi.fn()

vi.mock("@/api/models", () => ({
  setModelFallbacks: (...args: unknown[]) => setModelFallbacks(...args),
}))

vi.mock("@/lib/restart-required", () => ({
  saveAndApplyGatewayConfig: (...args: unknown[]) =>
    saveAndApplyGatewayConfig(...args),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({ toast: { error: vi.fn(), success: vi.fn() } }))

import { FallbackModelsSection } from "./fallback-models-section"

function model(name: string, overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 0,
    model_name: name,
    provider: "openai",
    model: `${name}-id`,
    api_key: "",
    enabled: true,
    available: true,
    status: "available",
    is_default: false,
    is_virtual: false,
    default_model_allowed: true,
    ...overrides,
  }
}

const models = [
  model("DeepSeek", { is_default: true }),
  model("Gemini", { provider: "gemini" }),
  model("Claude", { provider: "anthropic" }),
]

// The Radix Select trigger does not open under jsdom, so selection is exercised
// through the component's own state transitions where a real click is needed.
async function saveWith(fallbacks: string[]) {
  saveAndApplyGatewayConfig.mockImplementation(
    async (_t: unknown, options: { save: () => Promise<unknown> }) =>
      options.save(),
  )
  render(
    <FallbackModelsSection
      models={models}
      fallbacks={fallbacks}
      defaultModelName="DeepSeek"
      onSaved={vi.fn()}
    />,
  )
}

describe("FallbackModelsSection", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    setModelFallbacks.mockResolvedValue({ status: "ok", fallbacks: [] })
  })

  it("shows an explicit empty state when no fallback is configured", async () => {
    await saveWith([])
    expect(screen.getByText("models.fallbacks.empty")).toBeTruthy()
  })

  it("renders configured fallbacks in their declared order", async () => {
    await saveWith(["Claude", "Gemini"])

    const items = screen.getAllByRole("listitem")
    expect(items).toHaveLength(2)
    // Order is the feature: the chain is tried top to bottom.
    expect(items[0].textContent).toContain("Claude")
    expect(items[1].textContent).toContain("Gemini")
  })

  it("moves an entry up and saves the new order", async () => {
    await saveWith(["Claude", "Gemini"])

    // Move "Gemini" (second) above "Claude".
    fireEvent.click(screen.getAllByLabelText("models.fallbacks.moveUp")[1])

    const reordered = screen.getAllByRole("listitem")
    expect(reordered[0].textContent).toContain("Gemini")

    fireEvent.click(screen.getByRole("button", { name: "common.save" }))
    await waitFor(() => expect(setModelFallbacks).toHaveBeenCalled())
    expect(setModelFallbacks).toHaveBeenCalledWith(["Gemini", "Claude"])
  })

  it("removes an entry and saves the shorter chain", async () => {
    await saveWith(["Claude", "Gemini"])

    fireEvent.click(screen.getAllByLabelText("models.fallbacks.remove")[0])

    fireEvent.click(screen.getByRole("button", { name: "common.save" }))
    await waitFor(() => expect(setModelFallbacks).toHaveBeenCalled())
    expect(setModelFallbacks).toHaveBeenCalledWith(["Gemini"])
  })

  it("disables Save until the chain actually changes", async () => {
    await saveWith(["Gemini"])

    const save = screen.getByRole("button", { name: "common.save" })
    expect(save.hasAttribute("disabled")).toBe(true)

    fireEvent.click(screen.getByLabelText("models.fallbacks.remove"))
    expect(
      screen
        .getByRole("button", { name: "common.save" })
        .hasAttribute("disabled"),
    ).toBe(false)
  })

  it("routes the save through the gateway apply helper", async () => {
    await saveWith(["Gemini"])

    fireEvent.click(screen.getAllByLabelText("models.fallbacks.remove")[0])
    fireEvent.click(screen.getByRole("button", { name: "common.save" }))

    // Saving must go through saveAndApplyGatewayConfig, not setModelFallbacks
    // directly, or the gateway would keep serving the previous chain.
    await waitFor(() => expect(saveAndApplyGatewayConfig).toHaveBeenCalled())
  })

  it("still lists a fallback whose model was deleted so it can be removed", async () => {
    await saveWith(["Deleted"])

    const item = screen.getByRole("listitem")
    expect(item.textContent).toContain("Deleted")
    expect(item.textContent).toContain("models.fallbacks.missingModel")
  })

  it("keeps each fallback's own provider visible", async () => {
    await saveWith(["Gemini"])

    // The provider is shown because a fallback runs with its own provider and
    // credentials, never the primary's.
    expect(screen.getByRole("listitem").textContent).toContain("gemini")
  })
})
