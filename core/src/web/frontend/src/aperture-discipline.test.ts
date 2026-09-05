/**
 * Token discipline across the shipped surfaces.
 *
 * Aperture allows one interactive accent, one live-state colour and four
 * status colours. Every one of them is a token. A component that reaches for
 * `bg-emerald-500` instead is not just off-palette — it has opted out of both
 * themes at once, because a raw Tailwind colour cannot resolve per theme the
 * way a token does. That is what made the console read as a stock template in
 * the first place, so it is worth a guard rather than a review note.
 */
import fsSync from "node:fs"
import path from "node:path"

import { describe, expect, it } from "vitest"

/// shadcn primitives are vendored and already token-driven; they are not ours
/// to re-style, and they are the one place raw utilities are expected.
const VENDORED = path.join("components", "ui")

function shippedSources(): string[] {
  const found: string[] = []
  const walk = (dir: string) => {
    for (const entry of fsSync.readdirSync(dir, { withFileTypes: true })) {
      const full = path.join(dir, entry.name)
      if (entry.isDirectory()) {
        if (entry.name === "node_modules") continue
        walk(full)
        continue
      }
      if (!/\.tsx$/.test(entry.name)) continue
      if (entry.name.includes(".test.")) continue
      if (full.includes(VENDORED)) continue
      found.push(full)
    }
  }
  walk(path.join(process.cwd(), "src", "components"))
  walk(path.join(process.cwd(), "src", "routes"))
  return found
}

const SOURCES = shippedSources().map((file) => ({
  file: path.relative(process.cwd(), file),
  text: fsSync.readFileSync(file, "utf8"),
}))

describe("one palette", () => {
  it("finds the surfaces it is supposed to be checking", () => {
    expect(SOURCES.length).toBeGreaterThan(40)
  })

  // Status families are semantic: success, warning, danger. Naming the hue
  // instead of the meaning is how a "green" ends up meaning three things.
  it.each([
    "emerald",
    "green",
    "amber",
    "yellow",
    "violet",
    "purple",
    "zinc",
    "slate",
  ])("uses no raw %s utility", (family) => {
    const pattern = new RegExp(`\\b(?:bg|text|border|ring|fill|stroke)-${family}-\\d{2,3}\\b`)
    const offenders = SOURCES.filter(({ text }) => pattern.test(text)).map(
      ({ file }) => file,
    )
    expect(offenders).toEqual([])
  })

  it("hard-codes no hex colour in a shipped component", () => {
    const offenders = SOURCES.filter(({ text }) =>
      /className=[^\n]*#[0-9a-fA-F]{6}/.test(text),
    ).map(({ file }) => file)
    expect(offenders).toEqual([])
  })

  // A `dark:` variant on a token is a second source of truth for a value that
  // already resolves per theme.
  it("declares no dark: variant of an Aperture token", () => {
    const offenders = SOURCES.filter(({ text }) =>
      /\bdark:(?:bg|text|border|ring)-pc-/.test(text),
    ).map(({ file }) => file)
    expect(offenders).toEqual([])
  })
})

describe("live state stays reserved", () => {
  // Signal means "this is running right now". Spending it on a button or a
  // heading is what turns a considered dark mode into a gaming aesthetic.
  it("uses Signal only where something is genuinely live", () => {
    const allowed = new Set([
      "src/components/app-header.tsx", // the gateway pill
      "src/components/chat/typing-indicator.tsx", // a response streaming
    ])
    const offenders = SOURCES.filter(
      ({ file, text }) => text.includes("pc-signal") && !allowed.has(file),
    ).map(({ file }) => file)
    expect(offenders).toEqual([])
  })
})
