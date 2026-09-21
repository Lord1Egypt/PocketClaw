/**
 * Guided tour regression suite.
 *
 * Every case here comes from a defect the read-only audit proved against the
 * previous implementation, so each test names the failure it exists to stop
 * coming back rather than restating the code.
 */
import fsSync from "node:fs"
import path from "node:path"

import { act, fireEvent, render } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import i18n from "@/i18n"

const isMobile = vi.fn(() => false)
vi.mock("@/hooks/use-mobile", () => ({ useIsMobile: () => isMobile() }))

const { TourGuide } = await import("./tour-guide")
const { TOUR_STEP_CONFIG, isEligibleTarget, resolveTourTarget } =
  await import("./tour-steps")
const { migrateTourState, TOUR_VERSION, TOUR_VISIBLE_STEPS } =
  await import("@/store/tour")

const STORAGE_KEY = "pocketclaw-tour-state"
const VIEWPORT = { width: 1440, height: 900 }

function layout(
  element: HTMLElement,
  box: { top: number; left: number; width: number; height: number },
) {
  element.getBoundingClientRect = () =>
    ({
      ...box,
      right: box.left + box.width,
      bottom: box.top + box.height,
      x: box.left,
      y: box.top,
      toJSON: () => ({}),
    }) as DOMRect
}

/** The card has no layout in jsdom, so give it the size Tailwind's `w-80`
 *  plus its content produce in a browser. Without this the clamp has nothing
 *  to clamp. */
function sizeCard(width = 320, height = 200) {
  const card = layer("card")
  if (card) {
    layout(card, { top: 0, left: 0, width, height })
  }
  return card
}

function layer(name: "dimmer" | "spotlight" | "ring" | "card") {
  return document.querySelector(
    `[data-tour-layer="${name}"]`,
  ) as HTMLElement | null
}
function tourNodeCount() {
  return document.querySelectorAll("[data-tour-layer]").length
}
function cardBox() {
  const card = layer("card")
  if (!card) return null
  return { left: parseFloat(card.style.left), top: parseFloat(card.style.top) }
}
function control(pattern: RegExp) {
  return Array.from(document.querySelectorAll("button")).find((button) =>
    pattern.test(button.textContent ?? ""),
  )
}
function press(pattern: RegExp) {
  const button = control(pattern)
  expect(button, `no control matching ${pattern}`).toBeTruthy()
  act(() => {
    fireEvent.click(button!)
  })
}
/** Presses a control by translation key, so a test can drive the tour in a
 *  language whose labels are not English. */
function pressKey(key: "tour.next" | "tour.finish" | "tour.skip") {
  const label = i18n.t(key)
  const button = Array.from(
    document.querySelectorAll('[data-tour-layer="card"] button'),
  ).find((candidate) => (candidate.textContent ?? "").includes(label))
  expect(button, `no control labelled ${label}`).toBeTruthy()
  act(() => {
    fireEvent.click(button!)
  })
}
function mountSidebarTarget(side: "left" | "right") {
  document.body.innerHTML = `
    <nav><a data-tour="models-nav" href="/models">Models</a></nav>
    <header><button data-tour="gateway-button">Start</button></header>`
  const nav = document.querySelector("[data-tour='models-nav']") as HTMLElement
  const gateway = document.querySelector(
    "[data-tour='gateway-button']",
  ) as HTMLElement
  layout(nav, {
    top: 120,
    left: side === "left" ? 8 : VIEWPORT.width - 248,
    width: 240,
    height: 36,
  })
  layout(gateway, {
    top: 12,
    left: side === "left" ? VIEWPORT.width - 260 : 140,
    width: 120,
    height: 36,
  })
  return { nav, gateway }
}

function expectInsideViewport(label: string, anchored: boolean) {
  const card = layer("card")!
  // Without this the containment assertion would pass vacuously whenever the
  // target failed to resolve and the card fell back to centre.
  expect(
    layer("spotlight") !== null,
    `${label}: expected anchored=${anchored}`,
  ).toBe(anchored)
  const box = cardBox()!
  const width = card.getBoundingClientRect().width
  const height = card.getBoundingClientRect().height
  expect(box.left, `${label}: left edge`).toBeGreaterThanOrEqual(0)
  expect(box.top, `${label}: top edge`).toBeGreaterThanOrEqual(0)
  expect(box.left + width, `${label}: right edge`).toBeLessThanOrEqual(
    VIEWPORT.width,
  )
  expect(box.top + height, `${label}: bottom edge`).toBeLessThanOrEqual(
    VIEWPORT.height,
  )
}

beforeEach(async () => {
  localStorage.clear()
  document.body.innerHTML = ""
  document.documentElement.setAttribute("dir", "ltr")
  isMobile.mockReturnValue(false)
  window.innerWidth = VIEWPORT.width
  window.innerHeight = VIEWPORT.height
  await i18n.changeLanguage("en")
})

describe("placement stays inside the viewport in both directions", () => {
  /**
   * The reported desktop hang. The sidebar moves to the right edge in Arabic,
   * and the old `position: "right"` put the card at left: 1444px on a 1440px
   * viewport — the ring was visible but Next was off-screen and unreachable.
   */
  it("Arabic RTL keeps every step's card reachable", async () => {
    await i18n.changeLanguage("ar")
    document.documentElement.setAttribute("dir", "rtl")
    mountSidebarTarget("right")
    render(<TourGuide />)

    for (const step of TOUR_VISIBLE_STEPS) {
      sizeCard()
      act(() => {
        window.dispatchEvent(new Event("resize"))
      })
      expectInsideViewport(`rtl ${step}`, step !== "welcome")
      if (step !== TOUR_VISIBLE_STEPS[TOUR_VISIBLE_STEPS.length - 1]) {
        pressKey("tour.next")
      }
    }
  })

  it("English LTR keeps every step's card reachable", () => {
    mountSidebarTarget("left")
    render(<TourGuide />)

    for (const step of TOUR_VISIBLE_STEPS) {
      sizeCard()
      act(() => {
        window.dispatchEvent(new Event("resize"))
      })
      expectInsideViewport(`ltr ${step}`, step !== "welcome")
      if (step !== TOUR_VISIBLE_STEPS[TOUR_VISIBLE_STEPS.length - 1]) {
        press(/next/i)
      }
    }
  })

  it("a card larger than the viewport is still pinned to the safe margin", () => {
    mountSidebarTarget("left")
    render(<TourGuide />)
    press(/next/i)
    sizeCard(4000, 4000)
    act(() => {
      window.dispatchEvent(new Event("resize"))
    })
    const box = cardBox()!
    expect(box.left).toBeGreaterThanOrEqual(0)
    expect(box.top).toBeGreaterThanOrEqual(0)
  })
})

describe("the target is never activated through the tour", () => {
  it("a click on the spotlight does not reach the highlighted control", () => {
    const { nav } = mountSidebarTarget("left")
    let navClicks = 0
    nav.addEventListener("click", () => {
      navClicks += 1
    })
    render(<TourGuide />)
    press(/next/i)
    expect(layer("spotlight")).toBeTruthy()

    act(() => {
      fireEvent.click(layer("spotlight")!)
    })
    expect(navClicks, "the spotlight must swallow the click").toBe(0)
    expect(layer("card"), "and must not dismiss the tour").toBeTruthy()
  })

  it("the spotlight and ring follow the target rather than decorating it", () => {
    const { nav } = mountSidebarTarget("left")
    const before = nav.getAttribute("class")
    render(<TourGuide />)
    press(/next/i)
    expect(nav.getAttribute("class")).toBe(before)
    expect(nav.getAttribute("style")).toBeNull()
  })
})

describe("focus ownership", () => {
  /**
   * The reported "stuck highlight": the previous spotlight was
   * pointer-events-none, the click reached the real link, the anchor kept DOM
   * focus, and SidebarMenuButton's focus-visible ring stayed painted after the
   * tour was gone.
   */
  it("takes focus on open and gives it back on close", () => {
    document.body.innerHTML = `<button id="origin">origin</button>`
    mountSidebarTarget("left")
    const origin = document.createElement("button")
    origin.id = "origin"
    document.body.appendChild(origin)
    origin.focus()
    expect(document.activeElement).toBe(origin)

    const view = render(<TourGuide />)
    expect(document.activeElement).toBe(layer("card"))

    press(/skip/i)
    view.rerender(<TourGuide />)
    expect(document.activeElement, "focus returns to the opener").toBe(origin)
  })

  it("blurs rather than guessing when the opener is gone", () => {
    mountSidebarTarget("left")
    const view = render(<TourGuide />)
    press(/skip/i)
    view.rerender(<TourGuide />)
    expect(document.activeElement).toBe(document.body)
  })

  it("leaves no tour node behind on any termination path", () => {
    for (const path of ["skip", "finish"] as const) {
      localStorage.clear()
      mountSidebarTarget("left")
      const view = render(<TourGuide />)
      expect(tourNodeCount()).toBeGreaterThan(0)
      if (path === "finish") {
        TOUR_VISIBLE_STEPS.forEach(() => press(/next|finish/i))
      } else {
        press(/skip/i)
      }
      view.rerender(<TourGuide />)
      expect(tourNodeCount(), `after ${path}`).toBe(0)
      view.unmount()
    }
  })
})

describe("dismissal always has an exit", () => {
  it("Escape closes the tour and records completion", () => {
    mountSidebarTarget("left")
    render(<TourGuide />)
    act(() => {
      fireEvent.keyDown(document, { key: "Escape" })
    })
    expect(tourNodeCount()).toBe(0)
    expect(JSON.parse(localStorage.getItem(STORAGE_KEY)!)).toMatchObject({
      currentStep: "completed",
      isActive: false,
    })
  })

  it("a click on the dimmer closes the tour", () => {
    mountSidebarTarget("left")
    render(<TourGuide />)
    act(() => {
      fireEvent.click(layer("dimmer")!)
    })
    expect(tourNodeCount()).toBe(0)
  })

  it("the centred fallback is still dismissible, so it cannot deaden the app", () => {
    document.body.innerHTML = ""
    render(<TourGuide />)
    expect(layer("card")).toBeTruthy()
    expect(layer("spotlight"), "no target, so no spotlight").toBeNull()
    act(() => {
      fireEvent.click(layer("dimmer")!)
    })
    expect(tourNodeCount()).toBe(0)
  })
})

describe("target eligibility", () => {
  it("rejects an element that is mounted but has no layout", () => {
    document.body.innerHTML = `<a data-tour="models-nav" href="/models">Models</a>`
    const nav = document.querySelector(
      "[data-tour='models-nav']",
    ) as HTMLElement
    expect(isEligibleTarget(nav), "zero rect is not a usable target").toBe(
      false,
    )
    expect(resolveTourTarget("[data-tour='models-nav']")).toBeNull()
  })

  it("skips a hidden duplicate and takes the laid-out one", () => {
    document.body.innerHTML = `
      <a id="hidden" data-tour="models-nav" href="/models">Models</a>
      <a id="shown" data-tour="models-nav" href="/models">Models</a>`
    const hidden = document.getElementById("hidden") as HTMLElement
    const shown = document.getElementById("shown") as HTMLElement
    hidden.style.display = "none"
    layout(shown, { top: 40, left: 10, width: 200, height: 30 })
    expect(resolveTourTarget("[data-tour='models-nav']")).toBe(shown)
  })

  it("a step whose target never appears degrades to a centred card", () => {
    document.body.innerHTML = `<header><button data-tour="gateway-button">Start</button></header>`
    const gateway = document.querySelector(
      "[data-tour='gateway-button']",
    ) as HTMLElement
    layout(gateway, { top: 12, left: 100, width: 120, height: 36 })
    render(<TourGuide />)
    press(/next/i) // models, whose drawer target is absent in this layout
    expect(layer("card"), "the tour still renders").toBeTruthy()
    expect(layer("spotlight"), "with no spotlight to anchor").toBeNull()
    press(/next/i)
    expect(layer("card"), "and it can still be advanced").toBeTruthy()
  })
})

describe("geometry stays synchronized with the layout", () => {
  it("follows the target when it moves and the viewport reports it", () => {
    const { nav } = mountSidebarTarget("left")
    render(<TourGuide />)
    press(/next/i)
    const first = layer("ring")!.style.top

    layout(nav, { top: 620, left: 8, width: 240, height: 36 })
    act(() => {
      window.dispatchEvent(new Event("scroll"))
    })
    expect(layer("ring")!.style.top).not.toBe(first)
    expect(layer("ring")!.style.top).toBe("616px")
  })

  it("re-measures on resize", () => {
    const { nav } = mountSidebarTarget("left")
    render(<TourGuide />)
    press(/next/i)
    layout(nav, { top: 300, left: 40, width: 200, height: 36 })
    act(() => {
      window.dispatchEvent(new Event("resize"))
    })
    expect(layer("ring")!.style.left).toBe("36px")
  })

  it("recovers when the target is removed from the DOM", async () => {
    const { nav } = mountSidebarTarget("left")
    render(<TourGuide />)
    press(/next/i)
    expect(layer("spotlight")).toBeTruthy()

    await act(async () => {
      nav.remove()
      await Promise.resolve()
    })
    expect(layer("card"), "the tour survives").toBeTruthy()
    expect(layer("spotlight"), "and stops pointing at nothing").toBeNull()
  })

  it("re-resolves when a breakpoint swap replaces the target", async () => {
    mountSidebarTarget("left")
    render(<TourGuide />)
    press(/next/i)
    const original = document.querySelector(
      "[data-tour='models-nav']",
    ) as HTMLElement

    await act(async () => {
      original.remove()
      const replacement = document.createElement("a")
      replacement.setAttribute("data-tour", "models-nav")
      document.body.appendChild(replacement)
      layout(replacement, { top: 500, left: 60, width: 180, height: 40 })
      await Promise.resolve()
    })
    expect(layer("card")).toBeTruthy()
  })

  it("removes its listeners and observers when it closes", () => {
    const live = new Map<string, number>()
    const bump = (type: string, delta: number) =>
      live.set(type, (live.get(type) ?? 0) + delta)

    type AddArgs = Parameters<typeof window.addEventListener>
    const originalAdd = window.addEventListener.bind(window)
    const originalRemove = window.removeEventListener.bind(window)
    const addSpy = vi
      .spyOn(window, "addEventListener")
      .mockImplementation((...args: AddArgs) => {
        bump(String(args[0]), 1)
        originalAdd(...args)
      })
    const removeSpy = vi
      .spyOn(window, "removeEventListener")
      .mockImplementation((...args: AddArgs) => {
        bump(String(args[0]), -1)
        originalRemove(...args)
      })

    mountSidebarTarget("left")
    const view = render(<TourGuide />)
    press(/next/i)
    expect(live.get("scroll") ?? 0).toBeGreaterThan(0)
    press(/skip/i)
    view.rerender(<TourGuide />)

    expect(live.get("scroll") ?? 0, "scroll listeners balanced").toBe(0)
    expect(live.get("resize") ?? 0, "resize listeners balanced").toBe(0)

    addSpy.mockRestore()
    removeSpy.mockRestore()
  })
})

describe("step table is structurally honest", () => {
  /**
   * The `docs` step pointed at `[data-tour='docs-button']` for the whole life
   * of the tour. No component ever rendered that attribute, so the step
   * highlighted nothing in every layout and every language.
   */
  it("every declared selector is carried by a real control", () => {
    const componentRoot = path.resolve(__dirname, "..")
    const sources: string[] = []
    const walk = (directory: string) => {
      for (const entry of fsSync.readdirSync(directory, {
        withFileTypes: true,
      })) {
        const full = path.join(directory, entry.name)
        if (entry.isDirectory()) {
          walk(full)
        } else if (
          /\.tsx?$/.test(entry.name) &&
          !/\.test\.tsx?$/.test(entry.name)
        ) {
          sources.push(fsSync.readFileSync(full, "utf8"))
        }
      }
    }
    walk(componentRoot)
    const corpus = sources.join("\n")

    for (const [step, config] of Object.entries(TOUR_STEP_CONFIG)) {
      if (!config.targetSelector) {
        continue
      }
      const name = config.targetSelector.match(/data-tour='([^']+)'/)?.[1]
      expect(
        name,
        `${step}: selector must be a data-tour attribute`,
      ).toBeTruthy()
      expect(
        corpus.includes(`data-tour="${name}"`) || corpus.includes(`"${name}"`),
        `${step}: no component renders data-tour="${name}"`,
      ).toBe(true)
    }
  })

  it("declares no step for a removed docs control", () => {
    expect(Object.keys(TOUR_STEP_CONFIG)).not.toContain("docs")
    expect(JSON.stringify(TOUR_STEP_CONFIG)).not.toContain("docs-button")
  })
})

describe("persisted state migration", () => {
  it("keeps a finished user finished", () => {
    expect(
      migrateTourState({ currentStep: "completed", isActive: false }),
    ).toEqual({
      version: TOUR_VERSION,
      currentStep: "completed",
      isActive: false,
    })
  })

  it("resumes an unversioned mid-tour step that still exists", () => {
    expect(
      migrateTourState({ currentStep: "gateway", isActive: true }),
    ).toEqual({
      version: TOUR_VERSION,
      currentStep: "welcome",
      isActive: true,
    })
  })

  it("completes rather than stranding a user on a removed step", () => {
    expect(migrateTourState({ currentStep: "docs", isActive: true })).toEqual({
      version: TOUR_VERSION,
      currentStep: "completed",
      isActive: false,
    })
  })

  it("survives a corrupt or absent value", () => {
    for (const broken of [null, 7, "nonsense", []]) {
      expect(() => migrateTourState(broken)).not.toThrow()
    }
    expect(migrateTourState(null).currentStep).toBe("welcome")
  })

  it("does not downgrade state written by a newer build", () => {
    const future = {
      version: TOUR_VERSION + 1,
      currentStep: "models",
      isActive: true,
    }
    expect(migrateTourState(future)).toEqual(future)
  })

  it("a completed user is not re-toured on reload", () => {
    localStorage.setItem(
      STORAGE_KEY,
      JSON.stringify({ currentStep: "completed", isActive: false }),
    )
    mountSidebarTarget("left")
    render(<TourGuide />)
    expect(tourNodeCount()).toBe(0)
  })
})
