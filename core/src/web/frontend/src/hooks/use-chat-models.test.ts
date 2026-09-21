import { act, renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ModelInfo } from "@/api/models"

const getModels = vi.fn()
const setDefaultModel = vi.fn()
const applyGatewayConfigIfRequired = vi.fn()
const toastError = vi.fn()

vi.mock("@/api/models", () => ({
  getModels: (...args: unknown[]) => getModels(...args),
  setDefaultModel: (...args: unknown[]) => setDefaultModel(...args),
}))

vi.mock("@/lib/restart-required", () => ({
  applyGatewayConfigIfRequired: (...args: unknown[]) =>
    applyGatewayConfigIfRequired(...args),
}))

vi.mock("react-i18next", () => ({
  useTranslation: () => ({ t: (key: string) => key }),
}))

vi.mock("sonner", () => ({ toast: { error: (...a: unknown[]) => toastError(...a) } }))

import { useChatModels } from "./use-chat-models"

function model(name: string, overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 0,
    model_name: name,
    provider: "opencode_go",
    model: name,
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

const GEMINI = model("gemini", { index: 0, is_default: true })
const DEEPSEEK = model("deepseek-v4.1-flash", { index: 1 })

/**
 * PC-DEF-042. Selecting a model in the Chat header did not appear to do
 * anything: the trigger is controlled by server state, and that state was only
 * written after the save, a reload and a gateway restart had all resolved. The
 * owner went to Settings and back — which remounts this hook — and only then
 * saw the model they had already chosen.
 */
describe("useChatModels", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    getModels.mockResolvedValue({
      models: [GEMINI, DEEPSEEK],
      total: 2,
      default_model: "gemini",
      model_fallbacks: [],
      provider_options: [],
    })
    setDefaultModel.mockResolvedValue({ status: "ok" })
    applyGatewayConfigIfRequired.mockResolvedValue(undefined)
  })

  it("shows the picked model before the save round-trip resolves", async () => {
    let releaseSave: () => void = () => {}
    setDefaultModel.mockImplementation(
      () =>
        new Promise<void>((resolve) => {
          releaseSave = () => {
            // The backend persists before the reload reads it back, so the
            // stub has to as well: the point of the test is that the trigger
            // was already correct long before this happened.
            getModels.mockResolvedValue({
              models: [GEMINI, DEEPSEEK],
              total: 2,
              default_model: "deepseek-v4.1-flash",
              model_fallbacks: [],
              provider_options: [],
            })
            resolve()
          }
        }),
    )

    const { result } = renderHook(() => useChatModels({ isConnected: true }))
    await waitFor(() => expect(result.current.defaultModelName).toBe("gemini"))

    act(() => {
      void result.current.handleSetDefault("deepseek-v4.1-flash")
    })

    await waitFor(() =>
      expect(result.current.defaultModelName).toBe("deepseek-v4.1-flash"),
    )
    expect(result.current.settingDefault).toBe(true)

    await act(async () => {
      releaseSave()
    })
    await waitFor(() => expect(result.current.settingDefault).toBe(false))
    expect(result.current.defaultModelName).toBe("deepseek-v4.1-flash")
  })

  // The selection must survive the gateway restart the save triggers, which is
  // the slowest part and the one that used to leave the header looking wrong.
  it("keeps the selection while the gateway is being restarted", async () => {
    let releaseApply: () => void = () => {}
    applyGatewayConfigIfRequired.mockImplementation(
      () => new Promise<void>((resolve) => (releaseApply = () => resolve())),
    )
    getModels.mockResolvedValue({
      models: [GEMINI, DEEPSEEK],
      total: 2,
      default_model: "deepseek-v4.1-flash",
      model_fallbacks: [],
      provider_options: [],
    })

    const { result } = renderHook(() => useChatModels({ isConnected: true }))
    await waitFor(() =>
      expect(result.current.defaultModelName).toBe("deepseek-v4.1-flash"),
    )

    act(() => {
      void result.current.handleSetDefault("gemini")
    })
    await waitFor(() => expect(applyGatewayConfigIfRequired).toHaveBeenCalled())

    await act(async () => {
      releaseApply()
    })
    await waitFor(() => expect(result.current.settingDefault).toBe(false))
  })

  // "If persistence fails: show error, do not pretend the selection succeeded."
  it("reverts and reports when the save fails", async () => {
    setDefaultModel.mockRejectedValue(new Error("model_name is required"))

    const { result } = renderHook(() => useChatModels({ isConnected: true }))
    await waitFor(() => expect(result.current.defaultModelName).toBe("gemini"))

    await act(async () => {
      await result.current.handleSetDefault("deepseek-v4.1-flash")
    })

    expect(result.current.defaultModelName).toBe("gemini")
    expect(result.current.settingDefault).toBe(false)
    expect(toastError).toHaveBeenCalled()
    expect(applyGatewayConfigIfRequired).not.toHaveBeenCalled()
  })

  // A default that is no longer in model_list was deleted, or its provider was
  // removed. Chat clears it rather than going on pointing at it.
  it("clears a default that is no longer configured", async () => {
    getModels.mockResolvedValue({
      models: [DEEPSEEK],
      total: 1,
      default_model: "gemini",
      model_fallbacks: [],
      provider_options: [],
    })

    const { result } = renderHook(() => useChatModels({ isConnected: true }))
    await waitFor(() => expect(getModels).toHaveBeenCalled())
    await waitFor(() => expect(result.current.defaultModelName).toBe(""))
  })
})
