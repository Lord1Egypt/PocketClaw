import { beforeEach, describe, expect, it } from "vitest"

import { isAppReady, markAppReady, markAppUnhealthy } from "./app-readiness"

describe("app readiness flag", () => {
  beforeEach(() => {
    delete (window as { __pocketclawReady?: boolean }).__pocketclawReady
  })

  // The Android host's resume probe reads this. An unset flag must mean "not
  // ready", never "assume fine", because a WebView whose renderer was killed
  // never runs the code that would set it.
  it("is not ready before the app marks itself", () => {
    expect(isAppReady()).toBe(false)
  })

  it("is ready once the app has rendered", () => {
    markAppReady()
    expect(isAppReady()).toBe(true)
    expect(window.__pocketclawReady).toBe(true)
  })

  // A crashed React tree leaves a shell that looks mounted but is not a usable
  // console, so it must stop reporting itself healthy.
  it("stops reporting ready after the app is marked unhealthy", () => {
    markAppReady()
    markAppUnhealthy()
    expect(isAppReady()).toBe(false)
  })
})
