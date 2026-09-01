/**
 * WhatsApp Self-Chat: the console's half of the number contract.
 *
 * The same rules live in Go (`pkg/whatsapp/selfchat`), because the console
 * validates what the user types and Core re-validates what it reads back off
 * disk. Kept as a pure module so the rules are asserted directly rather than
 * inferred from a rendered form.
 */

/** E.164 caps a number at 15 digits; below 8 it cannot carry a country code
 * plus a plausible subscriber part. */
const MIN_NUMBER_DIGITS = 8
const MAX_NUMBER_DIGITS = 15

/** Separators a person plausibly types into, or pastes into, a phone field. */
const NUMBER_SEPARATORS = new Set([" ", "\t", "-", ".", "(", ")", " "])

export type WhatsAppNumberError =
  | "empty"
  | "invalidChars"
  | "notInternational"
  | "length"

export type WhatsAppNumberResult =
  | { ok: true; number: string }
  | { ok: false; error: WhatsAppNumberError }

/**
 * Converts user input into canonical international form: a leading "+" and
 * digits only.
 *
 * "+20 101 234 5678", "0020-101-234-5678" and "201012345678" all name their
 * country and are accepted. "0101 234 5678" is refused: the leading trunk zero
 * belongs to a national plan, and guessing the country would quietly point the
 * feature at a stranger's number.
 */
export function normalizeWhatsAppNumber(input: string): WhatsAppNumberResult {
  let digits = ""
  let plusSeen = false

  for (const character of input) {
    if (character >= "0" && character <= "9") {
      digits += character
      continue
    }
    if (character === "+") {
      // A "+" only means anything before the first digit.
      if (plusSeen || digits.length > 0) {
        return { ok: false, error: "invalidChars" }
      }
      plusSeen = true
      continue
    }
    if (NUMBER_SEPARATORS.has(character)) {
      continue
    }
    return { ok: false, error: "invalidChars" }
  }

  if (digits === "") {
    return { ok: false, error: "empty" }
  }

  // "00" is the international access code across most of the world, so
  // dropping it reads the country code the user already wrote rather than
  // inferring one. It is only an access code when no "+" was given.
  if (!plusSeen && digits.startsWith("00")) {
    digits = digits.slice(2)
  }

  if (digits.startsWith("0")) {
    return { ok: false, error: "notInternational" }
  }
  if (digits.length < MIN_NUMBER_DIGITS || digits.length > MAX_NUMBER_DIGITS) {
    return { ok: false, error: "length" }
  }

  return { ok: true, number: `+${digits}` }
}

/** Whether a stored config already holds a usable self number. */
export function isWhatsAppSelfChatConfigured(selfNumber: unknown): boolean {
  return (
    typeof selfNumber === "string" &&
    normalizeWhatsAppNumber(selfNumber).ok === true
  )
}

/**
 * The message the Settings Test button prepares. Deliberately not translated:
 * it is a marker a tester looks for verbatim in WhatsApp's compose box.
 */
export const WHATSAPP_SELF_CHAT_TEST_MESSAGE = "PocketClaw WhatsApp test"
