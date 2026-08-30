import { beforeEach, describe, expect, it, vi } from "vitest"

const applyGatewayConfig = vi.fn()
const refreshGatewayState = vi.fn()
const toastSuccess = vi.fn()
const toastError = vi.fn()
const toastLoading: (...a: unknown[]) => string = vi.fn(() => "toast-id")

vi.mock("@/api/gateway", () => ({
  applyGatewayConfig: (...args: unknown[]) => applyGatewayConfig(...args),
}))
vi.mock("@/store/gateway", () => ({
  refreshGatewayState: (...args: unknown[]) => refreshGatewayState(...args),
}))
vi.mock("sonner", () => ({
  toast: {
    success: (...a: unknown[]) => toastSuccess(...a),
    error: (...a: unknown[]) => toastError(...a),
    warning: vi.fn(),
    loading: (...a: unknown[]) => toastLoading(...a),
  },
}))

import { saveAndApplyGatewayConfig } from "./restart-required"

const t = ((key: string) => key) as never

const running = { status: "running", canStart: true, restartRequired: false }
const needsRestart = { status: "running", canStart: true, restartRequired: true }

describe("saveAndApplyGatewayConfig", () => {
  beforeEach(() => {
    vi.clearAllMocks()
    applyGatewayConfig.mockResolvedValue({ status: "ok" })
  })

  // The restart decision comes from the backend signature, not from this
  // helper. A save that changes nothing restart-worthy must not restart.
  it("does not restart when the backend says none is required", async () => {
    refreshGatewayState.mockResolvedValue(running)
    const save = vi.fn().mockResolvedValue("saved")

    const result = await saveAndApplyGatewayConfig(t, {
      save,
      savedMessage: "saved.message",
      name: "Model",
    })

    expect(result).toBe("saved")
    expect(applyGatewayConfig).not.toHaveBeenCalled()
    expect(toastSuccess).toHaveBeenCalledWith("saved.message")
  })

  it("restarts exactly once when the backend says it is required", async () => {
    refreshGatewayState
      .mockResolvedValueOnce(needsRestart) // after save
      .mockResolvedValue(running) // readiness poll
    const save = vi.fn().mockResolvedValue("saved")

    await saveAndApplyGatewayConfig(t, {
      save,
      savedMessage: "saved.message",
      name: "Model",
    })

    expect(applyGatewayConfig).toHaveBeenCalledTimes(1)
    expect(toastSuccess).toHaveBeenCalledWith(
      "common.restartedGateway",
      expect.anything(),
    )
  })

  // Readiness, not the HTTP 200 from the restart call, is what success means.
  it("waits for the gateway to be ready before reporting success", async () => {
    refreshGatewayState
      .mockResolvedValueOnce(needsRestart)
      .mockResolvedValueOnce({ status: "starting", canStart: false, restartRequired: true })
      .mockResolvedValue(running)

    await saveAndApplyGatewayConfig(t, {
      save: vi.fn().mockResolvedValue("saved"),
      savedMessage: "saved.message",
      name: "Model",
    })

    // Polled past the not-yet-ready state rather than declaring success.
    expect(refreshGatewayState.mock.calls.length).toBeGreaterThan(2)
    expect(toastSuccess).toHaveBeenCalledWith(
      "common.restartedGateway",
      expect.anything(),
    )
  })

  // A failed restart must never look like a failed save: the config is already
  // persisted, and the manual control stays available.
  it("keeps the saved config and reports failure when the gateway errors", async () => {
    refreshGatewayState
      .mockResolvedValueOnce(needsRestart)
      .mockResolvedValue({ status: "error", canStart: true, restartRequired: true })
    const save = vi.fn().mockResolvedValue("saved")

    const result = await saveAndApplyGatewayConfig(t, {
      save,
      savedMessage: "saved.message",
      name: "Model",
    })

    expect(result).toBe("saved")
    expect(save).toHaveBeenCalledTimes(1)
    expect(toastError).toHaveBeenCalled()
  })

  it("does not loop when the restart request itself throws", async () => {
    refreshGatewayState.mockResolvedValueOnce(needsRestart).mockResolvedValue(running)
    applyGatewayConfig.mockRejectedValue(new Error("gateway refused"))

    await saveAndApplyGatewayConfig(t, {
      save: vi.fn().mockResolvedValue("saved"),
      savedMessage: "saved.message",
      name: "Model",
    })

    expect(applyGatewayConfig).toHaveBeenCalledTimes(1)
    expect(toastError).toHaveBeenCalled()
  })

  // Several restart-requiring saves in quick succession should produce one
  // effective restart, not one per save.
  it("coalesces concurrent restart-requiring saves into a single restart", async () => {
    let resolveRestart: (v: unknown) => void = () => {}
    applyGatewayConfig.mockImplementation(
      () => new Promise((resolve) => (resolveRestart = resolve)),
    )
    refreshGatewayState.mockImplementation(async () => needsRestart)

    const first = saveAndApplyGatewayConfig(t, {
      save: vi.fn().mockResolvedValue("a"),
      savedMessage: "saved.a",
      name: "A",
    })
    await Promise.resolve()
    await Promise.resolve()

    refreshGatewayState.mockImplementation(async () => running)
    const second = saveAndApplyGatewayConfig(t, {
      save: vi.fn().mockResolvedValue("b"),
      savedMessage: "saved.b",
      name: "B",
    })

    resolveRestart({ status: "ok" })
    await Promise.all([first, second])

    // The second save saw the first restart already in flight and did not
    // start another one.
    expect(applyGatewayConfig).toHaveBeenCalledTimes(1)
  })

  it("persists the configuration before attempting any restart", async () => {
    const order: string[] = []
    refreshGatewayState.mockImplementation(async () => {
      order.push("refresh")
      return needsRestart
    })
    applyGatewayConfig.mockImplementation(async () => {
      order.push("restart")
      refreshGatewayState.mockImplementation(async () => running)
      return { status: "ok" }
    })

    await saveAndApplyGatewayConfig(t, {
      save: vi.fn().mockImplementation(async () => {
        order.push("save")
        return "saved"
      }),
      savedMessage: "saved.message",
      name: "Model",
    })

    // Save must come first: a restart that preceded persistence would boot the
    // old configuration and discard the user's change.
    expect(order[0]).toBe("save")
    expect(order.indexOf("restart")).toBeGreaterThan(order.indexOf("save"))
  })
})
