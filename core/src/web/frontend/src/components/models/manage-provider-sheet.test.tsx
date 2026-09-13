import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ProviderInfo } from "@/api/providers"

const getProvider = vi.fn()
const updateProvider = vi.fn()
const applyGatewayConfigIfRequired = vi.fn()

vi.mock("@/api/providers", () => ({
  getProvider: (...args: unknown[]) => getProvider(...args),
  updateProvider: (...args: unknown[]) => updateProvider(...args),
  deleteProvider: vi.fn(),
}))

vi.mock("@/lib/restart-required", () => ({
  applyGatewayConfigIfRequired: (...args: unknown[]) =>
    applyGatewayConfigIfRequired(...args),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({
  toast: { error: vi.fn(), success: vi.fn() },
}))

import { ManageProviderSheet } from "./manage-provider-sheet"

function info(overrides: Partial<ProviderInfo> = {}): ProviderInfo {
  return {
    provider: "opencode_go",
    model_count: 2,
    credential_state: "shared",
    api_key_masked: "sk-****abcd",
    api_base: "https://opencode.ai/zen/go/v1",
    api_base_mixed: false,
    holds_default_model: true,
    models: [
      {
        index: 0,
        model_name: "opencode-flash",
        model: "deepseek-v4.1-flash",
        enabled: true,
        is_default: true,
        has_api_key: true,
      },
      {
        index: 1,
        model_name: "opencode-coder",
        model: "qwen3-coder",
        enabled: true,
        is_default: false,
        has_api_key: true,
      },
    ],
    ...overrides,
  }
}

function renderSheet(overrides: Partial<ProviderInfo> = {}) {
  getProvider.mockResolvedValue(info(overrides))
  return render(
    <ManageProviderSheet
      provider="opencode_go"
      onClose={vi.fn()}
      onChanged={vi.fn()}
    />,
  )
}

async function saveButton() {
  return (await screen.findByText("models.provider.save")).closest(
    "button",
  ) as HTMLButtonElement
}

describe("ManageProviderSheet", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    updateProvider.mockResolvedValue({
      status: "ok",
      provider: "opencode_go",
      models_updated: 2,
      credential_change: true,
    })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
  })

  it("loads the provider it was opened for", async () => {
    renderSheet()
    await waitFor(() => expect(getProvider).toHaveBeenCalledWith("opencode_go"))
  })

  // The stored credential must never come back to the client in the clear, so
  // the field starts empty and the existing key is described, not filled in.
  it("starts the key field empty rather than prefilling the stored key", async () => {
    renderSheet()
    await screen.findByText("models.provider.credential")

    const field = document.querySelector(
      'input[type="password"]',
    ) as HTMLInputElement
    expect(field, "the credential field must exist").toBeTruthy()
    expect(field.value).toBe("")
  })

  // Nothing typed is nothing to save. A save that fired anyway would blank or
  // rewrite fields the user never touched.
  it("keeps Save disabled until something changes", async () => {
    renderSheet()
    expect((await saveButton()).disabled).toBe(true)
  })

  it("sends only the new key when the key is what changed", async () => {
    renderSheet()
    await screen.findByText("models.provider.credential")

    const field = document.querySelector(
      'input[type="password"]',
    ) as HTMLInputElement
    fireEvent.change(field, { target: { value: "sk-opencode-new" } })

    const save = await saveButton()
    expect(save.disabled).toBe(false)
    fireEvent.click(save)

    await waitFor(() =>
      expect(updateProvider).toHaveBeenCalledWith("opencode_go", {
        api_key: "sk-opencode-new",
      }),
    )
  })

  // The whole point of the defect: the key is only rotated once the gateway is
  // running on it. Saving without applying leaves the process on the old key.
  it("applies the rotation to the gateway after saving it", async () => {
    renderSheet()
    await screen.findByText("models.provider.credential")

    fireEvent.change(
      document.querySelector('input[type="password"]') as HTMLInputElement,
      { target: { value: "sk-opencode-new" } },
    )
    fireEvent.click(await saveButton())

    await waitFor(() => expect(applyGatewayConfigIfRequired).toHaveBeenCalled())
  })

  it("reports a failed save instead of claiming the key was replaced", async () => {
    updateProvider.mockRejectedValue(new Error("provider is not configured"))
    renderSheet()
    await screen.findByText("models.provider.credential")

    fireEvent.change(
      document.querySelector('input[type="password"]') as HTMLInputElement,
      { target: { value: "sk-opencode-new" } },
    )
    fireEvent.click(await saveButton())

    await waitFor(() =>
      expect(screen.getByText("provider is not configured")).toBeTruthy(),
    )
    expect(applyGatewayConfigIfRequired).not.toHaveBeenCalled()
  })

  // Presenting one of several keys as "the provider key" would let a rotation
  // look like it applied to a provider whose models actually disagree.
  it("says so when the provider's models use different keys", async () => {
    renderSheet({ credential_state: "mixed", api_key_masked: undefined })
    expect(
      await screen.findByText("models.provider.credentialMixed"),
    ).toBeTruthy()
  })

  // A base URL that differs per model is not a provider-level field, and
  // offering one editable box for it would silently overwrite the others.
  it("disables the base URL when the models do not agree on one", async () => {
    renderSheet({ api_base_mixed: true, api_base: undefined })
    await screen.findByText("models.provider.apiBaseMixed")

    const inputs = Array.from(
      document.querySelectorAll("input"),
    ) as HTMLInputElement[]
    const base = inputs.find((input) => input.disabled)
    expect(base, "the base URL field must be disabled, not editable").toBeTruthy()
  })

  // A provider that can be seen but not removed is the gap this sheet closes.
  it("offers Delete Provider", async () => {
    renderSheet()
    expect(
      (await screen.findByText("models.provider.delete")).closest("button"),
    ).toBeTruthy()
  })
})
