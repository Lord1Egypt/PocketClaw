import { fireEvent, render, screen, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ModelInfo } from "@/api/models"

const deleteModel = vi.fn()
const applyGatewayConfigIfRequired = vi.fn()
const toastError = vi.fn()

vi.mock("@/api/models", () => ({
  deleteModel: (...args: unknown[]) => deleteModel(...args),
}))

vi.mock("@/lib/restart-required", () => ({
  applyGatewayConfigIfRequired: (...args: unknown[]) =>
    applyGatewayConfigIfRequired(...args),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({ toast: { error: (...a: unknown[]) => toastError(...a) } }))

import { DeleteModelDialog } from "./delete-model-dialog"

function model(overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 3,
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

describe("DeleteModelDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    deleteModel.mockResolvedValue({ status: "ok" })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
  })

  it("removes the model and applies the change to the gateway", async () => {
    const onDeleted = vi.fn()
    render(
      <DeleteModelDialog
        model={model()}
        onClose={vi.fn()}
        onDeleted={onDeleted}
      />,
    )

    fireEvent.click(screen.getByText("models.delete.confirm"))

    await waitFor(() => expect(deleteModel).toHaveBeenCalledWith(3))
    expect(onDeleted).toHaveBeenCalled()
    // A gateway still holding the deleted entry would go on routing to it.
    await waitFor(() => expect(applyGatewayConfigIfRequired).toHaveBeenCalled())
  })

  // The dialog used to close and do nothing at all for the default model: the
  // user pressed Delete, the dialog went away and the model was still there.
  it("refuses the default model out loud rather than silently", () => {
    render(
      <DeleteModelDialog
        model={model({ is_default: true })}
        onClose={vi.fn()}
        onDeleted={vi.fn()}
      />,
    )

    expect(screen.getByText("models.delete.defaultBlocked")).toBeTruthy()
    const confirm = screen
      .getByText("models.delete.confirm")
      .closest("button") as HTMLButtonElement
    expect(confirm.disabled).toBe(true)

    fireEvent.click(confirm)
    expect(deleteModel).not.toHaveBeenCalled()
  })

  // A swallowed failure left the model in the list with no explanation.
  it("reports a failed delete instead of swallowing it", async () => {
    deleteModel.mockRejectedValue(new Error("Index 3 out of range"))
    const onClose = vi.fn()

    render(
      <DeleteModelDialog
        model={model()}
        onClose={onClose}
        onDeleted={vi.fn()}
      />,
    )

    fireEvent.click(screen.getByText("models.delete.confirm"))

    await waitFor(() => expect(toastError).toHaveBeenCalled())
    // The dialog stays open so the user can retry or cancel deliberately.
    expect(onClose).not.toHaveBeenCalled()
  })
})
