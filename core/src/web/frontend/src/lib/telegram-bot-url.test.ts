import { describe, expect, it } from "vitest"

import {
  canonicalTelegramUsername,
  telegramBotChatUrl,
} from "./telegram-bot-url"

/**
 * PC-DEF-075. "Open chat" produced `https://t.me/@name` and Telegram answered
 * "Username not found", while the bot, token and channel were all fine.
 * Telegram's canonical bot link carries no `@` in the path.
 */
describe("the canonical Telegram bot link", () => {
  it("builds the canonical URL from a bare username", () => {
    expect(telegramBotChatUrl("pocketclaw_ab12cd34_bot")).toBe(
      "https://t.me/pocketclaw_ab12cd34_bot",
    )
  })

  it("removes exactly one leading @ — the defect itself", () => {
    expect(telegramBotChatUrl("@pocketclaw_ab12cd34_bot")).toBe(
      "https://t.me/pocketclaw_ab12cd34_bot",
    )
  })

  it("never produces a doubled @", () => {
    for (const raw of [
      "@pocketclaw_ab12cd34_bot",
      " @pocketclaw_ab12cd34_bot ",
      "pocketclaw_ab12cd34_bot",
    ]) {
      expect(telegramBotChatUrl(raw)).not.toContain("@")
    }
  })

  it("refuses @@name rather than quietly repairing it", () => {
    // Already evidence that something upstream is wrong. Turning it into a
    // working link would hide the bug.
    expect(telegramBotChatUrl("@@pocketclaw_ab12cd34_bot")).toBeNull()
  })

  it("trims surrounding whitespace", () => {
    expect(telegramBotChatUrl("  pocketclaw_ab12cd34_bot \n")).toBe(
      "https://t.me/pocketclaw_ab12cd34_bot",
    )
  })

  // RTL display moves the neutral "@" to the visual right, so what is on
  // screen and what belongs in the path are different strings. Display
  // formatting must never become URL identity.
  it("is unchanged by bidirectional display formatting", () => {
    const plain = telegramBotChatUrl("pocketclaw_ab12cd34_bot")
    for (const marked of [
      "‏pocketclaw_ab12cd34_bot",
      "‫pocketclaw_ab12cd34_bot‬",
      "‎@pocketclaw_ab12cd34_bot",
    ]) {
      // A bidi control character is not part of a username, so it is refused
      // outright rather than silently stripped into something that looks right.
      expect(telegramBotChatUrl(marked)).toBeNull()
    }
    expect(plain).toBe("https://t.me/pocketclaw_ab12cd34_bot")
  })

  it("refuses anything that is not a username", () => {
    for (const raw of [
      "",
      "@",
      "bot",
      "https://t.me/pocketclaw_bot",
      "pocketclaw/../evil",
      "pocketclaw bot",
      "pocketclaw.bot",
      "javascript:alert(1)",
      "//evil.example",
      null,
      undefined,
      42,
    ]) {
      expect(telegramBotChatUrl(raw), String(raw)).toBeNull()
    }
  })

  it("exposes the canonical username on its own, for display", () => {
    expect(canonicalTelegramUsername("@pocketclaw_ab12cd34_bot")).toBe(
      "pocketclaw_ab12cd34_bot",
    )
    expect(canonicalTelegramUsername("@@x")).toBeNull()
  })
})
