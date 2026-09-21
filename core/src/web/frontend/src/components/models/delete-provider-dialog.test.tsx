import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ProviderInfo } from "@/api/providers"

const deleteProvider = vi.fn()
const applyGatewayConfigIfRequired = vi.fn()
const toastError = vi.fn()

vi.mock("@/api/providers", () => ({
  deleteProvider: (...args: unknown[]) => deleteProvider(...args),
}))

vi.mock("@/lib/restart-required", () => ({
  applyGatewayConfigIfRequired: (...args: unknown[]) =>
    applyGatewayConfigIfRequired(...args),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({
  toast: { error: (...a: unknown[]) => toastError(...a) },
}))

import { DeleteProviderDialog } from "./delete-provider-dialog"

function provider(overrides: Partial<ProviderInfo> = {}): ProviderInfo {
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

describe("DeleteProviderDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    deleteProvider.mockResolvedValue({
      status: "ok",
      provider: "opencode_go",
      models_removed: ["opencode-flash", "opencode-coder"],
      cleared_sites: ["agents.defaults.model_name"],
      models_remaining: 0,
    })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
  })

  // "This provider has 2 configured models" without saying which two asks the
  // user to remember what they configured.
  it("names every dependent model before the delete is confirmed", () => {
    render(
      <DeleteProviderDialog
        provider={provider()}
        label="OpenCode Go"
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    )

    expect(
      screen.getByText("opencode-flash"),
      "the dependent model must be named, not only counted",
    ).toBeTruthy()
    expect(screen.getByText("opencode-coder")).toBeTruthy()
  })

  // Losing the default model means Chat has no model until another is chosen,
  // which is worth saying before it happens rather than after.
  it("calls out the default chat model among the dependents", () => {
    render(
      <DeleteProviderDialog
        provider={provider()}
        label="OpenCode Go"
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    )

    expect(
      screen.getByText("models.provider.deleteDefaultNote"),
      "losing the default chat model must be called out before it happens",
    ).toBeTruthy()
  })

  it("deletes the provider and applies the change to the gateway", async () => {
    const onDeleted = vi.fn()
    render(
      <DeleteProviderDialog
        provider={provider()}
        label="OpenCode Go"
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    )

    fireEvent.click(screen.getByText("models.provider.deleteConfirm"))

    await waitFor(() => expect(deleteProvider).toHaveBeenCalledWith("opencode_go"))
    expect(onDeleted).toHaveBeenCalled()
    // A running gateway still holds the deleted models and their credentials.
    await waitFor(() =>
      expect(applyGatewayConfigIfRequired).toHaveBeenCalled(),
    )
  })

  // A swallowed failure would leave the provider listed with no explanation and
  // no way to tell a failed delete from a stale list.
  it("reports a failed delete instead of closing silently", async () => {
    deleteProvider.mockRejectedValue(new Error("config is read-only"))
    const onDeleted = vi.fn()

    render(
      <DeleteProviderDialog
        provider={provider()}
        label="OpenCode Go"
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    )

    fireEvent.click(screen.getByText("models.provider.deleteConfirm"))

    await waitFor(() => expect(toastError).toHaveBeenCalledWith("config is read-only"))
    expect(onDeleted).not.toHaveBeenCalled()
    expect(applyGatewayConfigIfRequired).not.toHaveBeenCalled()
  })
})
