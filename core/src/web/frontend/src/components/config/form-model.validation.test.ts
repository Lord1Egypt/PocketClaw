import { describe, expect, it } from "vitest"

import {
  parseFloatField,
  parseIntField,
  parseJSONObjectField,
} from "./form-model"

/// Stands in for i18next. It records what the validator asked for and returns a
/// recognisable string, so a test can tell a translated message apart from one
/// still built in source.
function translator() {
  const calls: Array<{ key: string; values: Record<string, unknown> }> = []
  const t = (key: string, values: Record<string, string | number>) => {
    calls.push({ key, values })
    const rendered = Object.entries(values)
      .map(([name, value]) => `${name}=${String(value)}`)
      .join(" ")
    return `[${key}] ${rendered}`
  }
  return { t, calls }
}

function rejection(run: () => unknown) {
  try {
    run()
  } catch (error) {
    return error instanceof Error ? error.message : String(error)
  }
  throw new Error("expected the validator to reject")
}

describe("numeric field validation", () => {
  // Localizing the messages must not move where the boundaries are.
  it("accepts and rejects exactly what it did before", () => {
    const { t } = translator()

    expect(parseIntField("5", "Max Tokens", t)).toBe(5)
    expect(parseIntField("0", "Max Tokens", t, { min: 0 })).toBe(0)
    expect(parseIntField("100", "Percent", t, { min: 1, max: 100 })).toBe(100)
    expect(parseFloatField("0.5", "Ratio", t, { min: 0.01, max: 1 })).toBe(0.5)

    for (const run of [
      () => parseIntField("1.5", "Max Tokens", t),
      () => parseIntField("abc", "Max Tokens", t),
      () => parseIntField("0", "Max Tokens", t, { min: 1 }),
      () => parseIntField("101", "Percent", t, { min: 1, max: 100 }),
      () => parseFloatField("x", "Ratio", t),
      () => parseFloatField("0", "Ratio", t, { min: 0.01 }),
      () => parseFloatField("1.5", "Ratio", t, { max: 1 }),
    ]) {
      expect(() => run()).toThrow()
    }
  })

  it("builds every rejection from the bundle, with the field and bound", () => {
    const { t, calls } = translator()

    expect(rejection(() => parseIntField("1.5", "Max Tokens", t))).toBe(
      "[pages.config.validation_integer] label=Max Tokens",
    )
    expect(rejection(() => parseFloatField("x", "Ratio", t))).toBe(
      "[pages.config.validation_number] label=Ratio",
    )
    expect(
      rejection(() => parseIntField("0", "Max Tokens", t, { min: 1 })),
    ).toBe("[pages.config.validation_min] label=Max Tokens min=1")
    expect(
      rejection(() => parseIntField("101", "Percent", t, { min: 1, max: 100 })),
    ).toBe("[pages.config.validation_max] label=Percent max=100")

    expect(calls.map((call) => call.key)).toEqual([
      "pages.config.validation_integer",
      "pages.config.validation_number",
      "pages.config.validation_min",
      "pages.config.validation_max",
    ])
  })
})

describe("MCP env and headers JSON validation", () => {
  it("parses exactly what it did before", () => {
    const { t } = translator()

    expect(parseJSONObjectField("", "headers", t)).toEqual({})
    expect(parseJSONObjectField("   ", "headers", t)).toEqual({})
    expect(
      parseJSONObjectField('{"Accept":"text/plain"}', "headers", t),
    ).toEqual({ Accept: "text/plain" })
    expect(parseJSONObjectField("{}", "headers", t)).toEqual({})
  })

  it("rejects malformed JSON, the wrong shape, and non-string values", () => {
    const { t } = translator()

    expect(rejection(() => parseJSONObjectField("{oops", "headers", t))).toBe(
      "[pages.config.validation_json_invalid] label=headers",
    )
    // An array is JSON, but not an object.
    expect(rejection(() => parseJSONObjectField("[1,2]", "headers", t))).toBe(
      "[pages.config.validation_json_not_object] label=headers",
    )
    expect(rejection(() => parseJSONObjectField("null", "headers", t))).toBe(
      "[pages.config.validation_json_not_object] label=headers",
    )
    expect(rejection(() => parseJSONObjectField('"text"', "headers", t))).toBe(
      "[pages.config.validation_json_not_object] label=headers",
    )
    // A member that is not a string names both the field and the member.
    expect(
      rejection(() => parseJSONObjectField('{"Retries":3}', "headers", t)),
    ).toBe(
      "[pages.config.validation_json_value_not_string] label=headers key=Retries",
    )
  })

  it("names the field the message is about", () => {
    const { t, calls } = translator()

    expect(() =>
      parseJSONObjectField("{oops", "MCP server github env", t),
    ).toThrow()
    expect(calls[0].values.label).toBe("MCP server github env")
  })
})
