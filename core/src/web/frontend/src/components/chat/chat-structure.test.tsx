/**
 * The accepted Aperture chat architecture, held in place.
 *
 * The user's turn is a contained bubble; the assistant's is an editorial block
 * on a logical-start rail. Two symmetrical bubbles facing each other is a
 * messaging app, and reverting to one is the specific regression these guard
 * against. Everything here is presentation: no test touches message
 * semantics, tool behaviour, reasoning visibility or streaming.
 */
import fsSync from "node:fs"

import { render, screen } from "@testing-library/react"
import { beforeEach, describe, expect, it } from "vitest"

import i18n from "@/i18n"

import { AssistantMessage } from "./assistant-message"
import { UserMessage } from "./user-message"

const read = (file: string) =>
  fsSync.readFileSync(`${process.cwd()}/src/components/chat/${file}`, "utf8")

describe("the two turns are not styled symmetrically", () => {
  beforeEach(async () => {
    await i18n.changeLanguage("en")
  })

  it("gives the user a bounded bubble", () => {
    const { container } = render(<UserMessage content="hello" />)
    const bubble = container.querySelector('[class*="rounded-2xl"]')
    expect(bubble, "the user turn lost its bubble").toBeTruthy()
    expect(container.querySelector('[class*="max-w-"]')).toBeTruthy()
    // Contained, not full measure.
    expect(container.innerHTML).toContain("items-end")
  })

  it("gives the assistant a rail and no bubble", () => {
    const { container } = render(
      <AssistantMessage content="here is the answer" />,
    )
    expect(
      container.querySelector(".pc-rail"),
      "the assistant turn lost its rail",
    ).toBeTruthy()
    // A bubble would mean a bounded, filled container around the prose.
    const body = container.querySelector(".pc-rail")
    expect(body?.className).not.toContain("rounded-2xl")
    expect(body?.className).not.toContain("bg-pc-surface-2")
  })

  it("draws the rail on a logical edge so Arabic mirrors it", () => {
    const css = fsSync.readFileSync(`${process.cwd()}/src/index.css`, "utf8")
    expect(css).toMatch(/\.pc-rail\s*{\s*border-inline-start/)
    expect(read("assistant-message.tsx")).not.toMatch(/\bborder-l\b/)
  })

  it("keeps machine values in mono and left-to-right", () => {
    render(
      <AssistantMessage content="ok" modelName="claude-opus-5" />,
    )
    const model = screen.getByText("claude-opus-5")
    expect(model.className).toContain("pc-mono")
    expect(model.getAttribute("dir")).toBe("ltr")
  })

  it("still renders a slash command as a readout, not as speech", () => {
    const { container } = render(<UserMessage content="/context" />)
    expect(container.innerHTML).toContain("pc-mono")
  })
})

describe("reasoning and tool visibility are untouched", () => {
  // Aperture restyles the control and nothing else. A collapsed block must
  // stay collapsible, and no hidden content may be rendered by default.
  it("keeps the collapsed block's expand affordance", () => {
    const { container } = render(
      <AssistantMessage content="thinking" kind="thought" />,
    )
    expect(container.querySelector("svg")).toBeTruthy()
  })

  it("does not reach into detail-visibility from the message component", () => {
    expect(read("assistant-message.tsx")).not.toContain("detail-visibility")
    expect(read("assistant-message.tsx")).not.toContain(
      "assistantDetailVisibility",
    )
  })
})

describe("the composer is one control well", () => {
  const composer = read("chat-composer.tsx")

  it("responds to focus as a whole, not just at the textarea", () => {
    expect(composer).toContain("focus-within:border-pc-claw")
    expect(composer).toContain("focus-within:ring-2")
  })

  it("keeps send in one slot rather than unmounting it", () => {
    // Unmounting the button collapsed the footer mid-conversation. It is
    // disabled now, so the row keeps its height.
    expect(composer).toContain("disabled={!canInput || !canSend}")
    expect(composer).not.toContain("{canInput ? (")
  })

  it("gives the attachment and send controls real hit areas", () => {
    expect(composer.match(/size-10/g)?.length).toBeGreaterThanOrEqual(2)
  })

  it("keeps the keyboard and safe area in mind on mobile", () => {
    expect(composer).toContain("env(safe-area-inset-bottom)")
  })

  it("uses logical edges throughout", () => {
    expect(composer).not.toMatch(/\b(ml|mr|pl|pr)-\d/)
    expect(composer).not.toMatch(/\b(top|bottom)-1 (left|right)-1\b/)
    expect(composer).toContain("end-1")
  })

  it("shares the thread's measure so the well lines up with the messages", () => {
    expect(composer).toContain("max-w-[78ch]")
    expect(read("chat-page.tsx")).toContain("max-w-[78ch]")
  })
})

describe("streaming state", () => {
  it("wears Signal and the assistant rail, so nothing shifts when it resolves", () => {
    const indicator = read("typing-indicator.tsx")
    expect(indicator).toContain("pc-rail")
    expect(indicator).toContain("bg-pc-signal")
    expect(indicator).toContain('role="status"')
    expect(indicator).toContain('aria-live="polite"')
  })
})

describe("no palette leaks back into chat", () => {
  it.each([
    "assistant-message.tsx",
    "user-message.tsx",
    "chat-composer.tsx",
    "chat-empty-state.tsx",
    "typing-indicator.tsx",
    "message-code-block.tsx",
    "context-usage-ring.tsx",
  ])("%s uses tokens, not one-off colours", (file) => {
    const source = read(file)
    expect(source).not.toMatch(/\b(violet|zinc|emerald|slate)-\d{2,3}\b/)
    expect(source).not.toMatch(/#[0-9a-fA-F]{6}\b/)
  })
})
