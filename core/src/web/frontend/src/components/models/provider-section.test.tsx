import { fireEvent, render, screen } from "@testing-library/react"
import { describe, expect, it, vi } from "vitest"

import type { ModelInfo } from "@/api/models"
import { TooltipProvider } from "@/components/ui/tooltip"

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

import { ProviderSection } from "./provider-section"

function model(overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 0,
    model_name: "opencode-flash",
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

function renderSection(onManageProvider = vi.fn()) {
  render(
    <TooltipProvider>
      <ProviderSection
        provider={{ key: "opencode_go", label: "OpenCode Go" }}
        models={[model()]}
        onEdit={vi.fn()}
        onSetDefault={vi.fn()}
        onDelete={vi.fn()}
        onManageProvider={onManageProvider}
        settingDefaultIndex={null}
      />
    </TooltipProvider>,
  )
  return onManageProvider
}

describe("ProviderSection", () => {
  // The provider heading used to be a divider and a label with no action on it,
  // so a configured provider could be seen but not edited, re-keyed or removed.
  it("offers a provider-level Manage control", () => {
    renderSection()

    const manage = screen
      .getByText("models.provider.manage")
      .closest("button")
    expect(manage, "Manage must be a real control on the provider heading")
      .toBeTruthy()
  })

  it("opens management for the provider it belongs to", () => {
    const onManageProvider = renderSection()

    fireEvent.click(
      screen.getByText("models.provider.manage").closest("button")!,
    )

    expect(onManageProvider).toHaveBeenCalledWith("opencode_go")
  })

  // A tooltip explains nothing on a touch screen, and a 32px icon beside a
  // truncating label is what made the model controls unfindable on the device.
  it("labels Manage with text and gives it a full-height touch target", () => {
    renderSection()

    const manage = screen
      .getByText("models.provider.manage")
      .closest("button") as HTMLElement
    expect(manage.className).toContain("min-h-10")
  })

  // Collapsing the section and managing the provider are different intentions.
  // One control doing both would make Manage unreachable while collapsed.
  it("keeps the collapse toggle separate from Manage", () => {
    const onManageProvider = renderSection()

    const toggle = screen
      .getByText("OpenCode Go")
      .closest("button") as HTMLElement
    expect(toggle.getAttribute("aria-expanded")).toBe("true")

    fireEvent.click(toggle)
    expect(onManageProvider).not.toHaveBeenCalled()
    expect(toggle.getAttribute("aria-expanded")).toBe("false")

    // Still reachable with the model list collapsed.
    expect(
      screen.getByText("models.provider.manage").closest("button"),
    ).toBeTruthy()
  })
})
