import { render, screen } from "@testing-library/react"
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest"

import { isAppReady, markAppReady } from "@/lib/app-readiness"

import { AppErrorBoundary } from "./app-error-boundary"

function Boom(): never {
  throw new Error("render exploded")
}

describe("AppErrorBoundary", () => {
  let consoleError: ReturnType<typeof vi.spyOn>

  beforeEach(() => {
    // React logs the caught error itself; silencing keeps the test output
    // readable without hiding the assertions below.
    consoleError = vi.spyOn(console, "error").mockImplementation(() => {})
    markAppReady()
  })

  afterEach(() => {
    consoleError.mockRestore()
  })

  it("renders children when nothing throws", () => {
    render(
      <AppErrorBoundary>
        <p>console content</p>
      </AppErrorBoundary>,
    )
    expect(screen.getByText("console content")).toBeTruthy()
  })

  // Without a boundary an uncaught render error empties #root, which looks
  // exactly like a dead WebView renderer and is just as unexplained.
  it("shows a recoverable message instead of an empty page", () => {
    render(
      <AppErrorBoundary>
        <Boom />
      </AppErrorBoundary>,
    )
    expect(screen.getByText(/stopped responding/i)).toBeTruthy()
    expect(screen.getByRole("button", { name: /reload/i })).toBeTruthy()
  })

  it("marks the app unhealthy so the host probe can recover it", () => {
    render(
      <AppErrorBoundary>
        <Boom />
      </AppErrorBoundary>,
    )
    expect(isAppReady()).toBe(false)
  })
})
