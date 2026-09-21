import { describe, expect, it } from "vitest"

import { resolveLauncherSetupGate } from "./launcher-setup-gate"

describe("launcher setup gate", () => {
  it("offers first-run setup only when nobody owns the dashboard", () => {
    expect(
      resolveLauncherSetupGate({ authenticated: false, initialized: false }),
    ).toEqual({ render: "first-run" })
  })

  it("sends an anonymous visitor to login once an owner exists", () => {
    expect(
      resolveLauncherSetupGate({ authenticated: false, initialized: true }),
    ).toEqual({ render: "leaving", redirectTo: "/launcher-login" })
  })

  it("sends an authenticated owner to the existing password-change surface", () => {
    expect(
      resolveLauncherSetupGate({ authenticated: true, initialized: true }),
    ).toEqual({ render: "leaving", redirectTo: "/config" })
  })

  it("never renders first-run copy for an initialized dashboard", () => {
    for (const authenticated of [true, false]) {
      expect(
        resolveLauncherSetupGate({ authenticated, initialized: true }).render,
      ).not.toBe("first-run")
    }
  })

  it("does not guess first-run when status is unreachable", () => {
    expect(resolveLauncherSetupGate(null)).toEqual({
      render: "leaving",
      redirectTo: "/launcher-login",
    })
  })
})
