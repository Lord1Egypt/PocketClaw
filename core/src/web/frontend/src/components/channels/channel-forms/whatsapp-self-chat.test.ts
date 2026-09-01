import { describe, expect, it } from "vitest"

import {
  isWhatsAppSelfChatConfigured,
  normalizeWhatsAppNumber,
} from "@/components/channels/channel-forms/whatsapp-self-chat"

describe("normalizeWhatsAppNumber", () => {
  it.each([
    ["+20 101 234 5678", "+201012345678"],
    ["+201012345678", "+201012345678"],
    ["+90 (532) 123-4567", "+905321234567"],
    ["  +905321234567  ", "+905321234567"],
    ["201012345678", "+201012345678"],
    ["0020 101 234 5678", "+201012345678"],
    ["+1.415.555.0123", "+14155550123"],
  ])("normalises %s to %s", (input, expected) => {
    expect(normalizeWhatsAppNumber(input)).toEqual({
      ok: true,
      number: expected,
    })
  })

  it.each([
    ["", "empty"],
    ["   ", "empty"],
    ["+", "empty"],
    ["01012345678", "notInternational"],
    ["0101 234 5678", "notInternational"],
    // "00" is only an access code when the user did not already write "+".
    ["+0020101234567", "notInternational"],
    ["+201234", "length"],
    ["+2010123456789012", "length"],
    ["+20 101 CALL ME", "invalidChars"],
    ["20+1012345678", "invalidChars"],
    ["++201012345678", "invalidChars"],
    ["+20101234567;8", "invalidChars"],
  ])("rejects %s as %s", (input, error) => {
    expect(normalizeWhatsAppNumber(input)).toEqual({ ok: false, error })
  })
})

describe("isWhatsAppSelfChatConfigured", () => {
  it("is true only for a stored number that still validates", () => {
    expect(isWhatsAppSelfChatConfigured("+201012345678")).toBe(true)
    expect(isWhatsAppSelfChatConfigured("")).toBe(false)
    expect(isWhatsAppSelfChatConfigured("01012345678")).toBe(false)
    expect(isWhatsAppSelfChatConfigured(undefined)).toBe(false)
    expect(isWhatsAppSelfChatConfigured(201012345678)).toBe(false)
  })
})
