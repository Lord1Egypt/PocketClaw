import { describe, expect, it, vi } from "vitest"

import { type ClipboardEnvironment, copyText } from "./clipboard"

/**
 * PC-DEF-052 follow-up. The desktop "Copy link" failed on a plain-HTTP LAN
 * origin because it called navigator.clipboard directly. copyText is the shared
 * helper that tries the async Clipboard API and falls back to execCommand, and
 * these pin exactly that contract, including the paths jsdom cannot produce.
 */

function environment(options: {
  writeText?: (text: string) => Promise<void>
  execCommand?: () => boolean
  document?: boolean
}): ClipboardEnvironment {
  const textarea = {
    value: "",
    style: { position: "", left: "" },
    setAttribute: vi.fn(),
    select: vi.fn(),
  }
  const documentLike = {
    body: { appendChild: vi.fn(), removeChild: vi.fn() },
    createElement: vi.fn(() => textarea),
    execCommand: vi.fn(options.execCommand ?? (() => true)),
  }
  const env: ClipboardEnvironment = {
    navigator: options.writeText
      ? { clipboard: { writeText: options.writeText } }
      : {},
  }
  if (options.document !== false) {
    env.document = documentLike as unknown as ClipboardEnvironment["document"]
  }
  return env
}

describe("copyText", () => {
  it("uses the Clipboard API and does not touch the fallback", async () => {
    const writeText = vi.fn().mockResolvedValue(undefined)
    const env = environment({ writeText })

    await expect(copyText("https://t.me/x", env)).resolves.toBe(true)
    expect(writeText).toHaveBeenCalledWith("https://t.me/x")
    expect(env.document?.execCommand).not.toHaveBeenCalled()
  })

  it("falls back to execCommand when the Clipboard write rejects", async () => {
    const writeText = vi.fn().mockRejectedValue(new Error("NotAllowedError"))
    const env = environment({ writeText, execCommand: () => true })

    await expect(copyText("https://t.me/x", env)).resolves.toBe(true)
    expect(env.document?.execCommand).toHaveBeenCalledWith("copy")
  })

  it("falls back when the Clipboard API is unavailable", async () => {
    // This is the plain-HTTP LAN origin: navigator.clipboard is undefined.
    const env = environment({ execCommand: () => true })

    await expect(copyText("https://t.me/x", env)).resolves.toBe(true)
    expect(env.document?.execCommand).toHaveBeenCalledWith("copy")
  })

  it("reports failure when both paths are unavailable", async () => {
    const env: ClipboardEnvironment = { navigator: {} }

    await expect(copyText("https://t.me/x", env)).resolves.toBe(false)
  })

  it("reports failure when execCommand refuses", async () => {
    const env = environment({ execCommand: () => false })

    await expect(copyText("https://t.me/x", env)).resolves.toBe(false)
  })

  it("reports failure rather than throwing when execCommand is missing", async () => {
    const env = environment({})
    // Simulate jsdom, which has no execCommand at all.
    ;(env.document as unknown as { execCommand?: unknown }).execCommand =
      undefined

    await expect(copyText("https://t.me/x", env)).resolves.toBe(false)
  })
})
