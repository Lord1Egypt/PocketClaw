import { describe, expect, it } from "vitest"

import {
  isAdvancedFormAlwaysVisible,
  isTelegramManageableWithoutHost,
  isTelegramStartingState,
  resolveTelegramManualReason,
  resolveTelegramSurface,
} from "./telegram-surface"

/**
 * PC-DEF-060. The desktop Dashboard does not omit Telegram management — the manual
 * form is the whole page there — but it used to explain itself with "One-tap bot
 * creation is not available in this build", which on a browser is untrue and sends
 * the user looking for a different build.
 */
describe("telegram surface", () => {
  describe("which surface is shown", () => {
    it("puts a connected channel first whatever the client", () => {
      // The connected summary must not depend on the Android host: the
      // configuration is Core's.
      expect(
        resolveTelegramSurface({ configured: true, onboardingAvailable: false }),
      ).toBe("connected")
      expect(
        resolveTelegramSurface({ configured: true, onboardingAvailable: true }),
      ).toBe("connected")
    })

    it("offers managed onboarding only where it can run", () => {
      expect(
        resolveTelegramSurface({ configured: false, onboardingAvailable: true }),
      ).toBe("managed-onboarding")
      expect(
        resolveTelegramSurface({ configured: false, onboardingAvailable: false }),
      ).toBe("manual-only")
    })

    it("keeps the manual form visible when it is the only path", () => {
      expect(isAdvancedFormAlwaysVisible("manual-only")).toBe(true)
      expect(isAdvancedFormAlwaysVisible("managed-onboarding")).toBe(false)
      expect(isAdvancedFormAlwaysVisible("connected")).toBe(false)
    })
  })

  // The defect: one sentence for two different causes.
  describe("why managed onboarding is unavailable", () => {
    it("says a browser has no host rather than blaming the build", () => {
      expect(
        resolveTelegramManualReason({
          hostPresent: false,
          onboardingConfigured: false,
        }),
      ).toBe("no-host")
    })

    it("blames the build only when a host is present without an endpoint", () => {
      expect(
        resolveTelegramManualReason({
          hostPresent: true,
          onboardingConfigured: false,
        }),
      ).toBe("host-without-endpoint")
    })

    // A host that *is* configured never reaches the manual surface, but if it
    // somehow does, the honest answer is still "not this client".
    it("does not blame the build when the endpoint is configured", () => {
      expect(
        resolveTelegramManualReason({
          hostPresent: true,
          onboardingConfigured: true,
        }),
      ).toBe("no-host")
    })
  })

  // The owner's requirement that a connected configuration stays manageable from
  // desktop, stated as its own rule so it cannot be lost to a host check.
  describe("an existing configuration is manageable without a host", () => {
    it("is manageable when configured", () => {
      expect(isTelegramManageableWithoutHost(true)).toBe(true)
    })

    it("has nothing to manage when not configured", () => {
      expect(isTelegramManageableWithoutHost(false)).toBe(false)
    })
  })

  // PC-DEF-071. The connected card presented anything without a branch of its
  // own as a stage of starting. The membership is stated positively here so a
  // state added later cannot inherit that claim by omission.
  describe("which readiness states are stages of starting", () => {
    it.each(["gateway_starting", "channel_starting", "registering_commands"])(
      "%s is a starting stage",
      (state) => {
        expect(isTelegramStartingState(state)).toBe(true)
      },
    )

    it.each([
      "not_configured",
      "gateway_stopped",
      "authentication_failed",
      "telegram_conflict",
      "setup_required",
      "ready",
      "unknown",
    ])("%s is not a starting stage", (state) => {
      expect(isTelegramStartingState(state)).toBe(false)
    })
  })
})
