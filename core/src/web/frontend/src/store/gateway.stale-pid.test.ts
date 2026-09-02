import { getDefaultStore } from "jotai"
import { beforeEach, describe, expect, it } from "vitest"

import {
  applyGatewayStatusToStore,
  gatewayAtom,
  updateGatewayStore,
} from "./gateway"

const STALE_PID_REASON =
  "Gateway could not start because a stale previous process record was detected."

function readStore() {
  return getDefaultStore().get(gatewayAtom)
}

describe("gateway store last-error handling", () => {
  beforeEach(() => {
    updateGatewayStore({
      status: "unknown",
      canStart: true,
      restartRequired: false,
      lastError: undefined,
    })
  })

  it("keeps the sanitized reason when the gateway reports an error", () => {
    applyGatewayStatusToStore({
      gateway_status: "error",
      gateway_last_error: STALE_PID_REASON,
    })

    expect(readStore().status).toBe("error")
    expect(readStore().lastError).toBe(STALE_PID_REASON)
  })

  it("clears the reason once the gateway starts successfully", () => {
    applyGatewayStatusToStore({
      gateway_status: "error",
      gateway_last_error: STALE_PID_REASON,
    })
    expect(readStore().lastError).toBe(STALE_PID_REASON)

    applyGatewayStatusToStore({ gateway_status: "running" })

    expect(readStore().status).toBe("running")
    expect(readStore().lastError).toBeUndefined()
  })

  it("does not strand the UI in starting when the child exits", () => {
    applyGatewayStatusToStore({ gateway_status: "starting" })
    expect(readStore().status).toBe("starting")

    applyGatewayStatusToStore({
      gateway_status: "error",
      gateway_last_error: STALE_PID_REASON,
    })

    expect(readStore().status).not.toBe("starting")
    expect(readStore().status).toBe("error")
  })
})
