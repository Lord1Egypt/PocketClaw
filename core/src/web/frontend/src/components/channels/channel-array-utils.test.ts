/**
 * List-valued channel config fields.
 *
 * Canonical on-disk shape is a JSON array. Older console builds joined entries
 * with "\n" into a single string, which the backend could not read back: Go's
 * FlexibleStringSlice turns "a\nb" into the single entry "a\nb" rather than
 * two, and the plain []string fields rejected a bare string outright. These
 * pin the canonical write and the legacy read that keeps old configs working.
 */
import { describe, expect, it } from "vitest"

import {
  asStringArray,
  normalizeAllowFromValues,
  parseAllowFromInput,
  serializeStringArrayForSubmit,
} from "./channel-array-utils"

describe("reading stored list values", () => {
  it("reads the canonical array", () => {
    expect(asStringArray(["123", "456"])).toEqual(["123", "456"])
  })

  it("reads a legacy single-value string", () => {
    expect(asStringArray("pico-user")).toEqual(["pico-user"])
  })

  it("reads a legacy newline-joined string as separate entries", () => {
    expect(asStringArray("123\n456")).toEqual(["123", "456"])
  })

  it("drops blank lines and surrounding whitespace", () => {
    expect(asStringArray("  123  \n\n\r\n  456 \n   ")).toEqual(["123", "456"])
  })

  it("leaves a stored entry containing a comma intact", () => {
    // Only "\n" was ever a separator in a stored value.
    expect(asStringArray("a,b")).toEqual(["a,b"])
  })

  it("reads an unsupported type as empty rather than throwing", () => {
    expect(asStringArray(undefined)).toEqual([])
    expect(asStringArray(null)).toEqual([])
    expect(asStringArray(42)).toEqual([])
    expect(asStringArray({ a: 1 })).toEqual([])
  })
})

describe("writing list values on save", () => {
  it("emits an array, never a newline-joined string", () => {
    expect(serializeStringArrayForSubmit(["123", "456"])).toEqual([
      "123",
      "456",
    ])
  })

  it("keeps a one-item list a one-item array", () => {
    expect(serializeStringArrayForSubmit(["pico-user"])).toEqual(["pico-user"])
  })

  it("trims entries and drops blanks", () => {
    expect(serializeStringArrayForSubmit(["  a  ", "", "   ", "b"])).toEqual([
      "a",
      "b",
    ])
  })

  it("drops duplicates", () => {
    expect(serializeStringArrayForSubmit(["a", "a", "b"])).toEqual(["a", "b"])
  })

  it("emits an empty array for an emptied list", () => {
    expect(serializeStringArrayForSubmit([])).toEqual([])
  })

  it("leaves non-list values alone", () => {
    expect(serializeStringArrayForSubmit(18790)).toBe(18790)
    expect(serializeStringArrayForSubmit(true)).toBe(true)
    expect(serializeStringArrayForSubmit("irc.example.org")).toBe(
      "irc.example.org",
    )
  })
})

describe("multi-value entry becomes array entries", () => {
  it("splits what the user types into separate entries", () => {
    const typed = parseAllowFromInput("123, 456\n789")
    expect(typed).toEqual(["123", "456", "789"])
    expect(serializeStringArrayForSubmit(typed)).toEqual(["123", "456", "789"])
  })

  it("normalizes an allow-list read from a legacy string", () => {
    expect(normalizeAllowFromValues("123\n456")).toEqual(["123", "456"])
  })
})
