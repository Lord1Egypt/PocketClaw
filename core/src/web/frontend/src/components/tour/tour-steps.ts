import type { TourStep } from "@/store/tour"

/**
 * Where the card sits relative to its target, in logical terms.
 *
 * `start`/`end` follow the writing direction rather than the screen. The
 * sidebar already moves to the right edge in Arabic, so a hard-coded "right"
 * put the card — and the only Next button — past the viewport edge with no way
 * to reach it. The header makes the same promise: see the "uses no physical
 * direction" assertion in header-responsive.test.tsx.
 */
export type TourPlacement = "start" | "end" | "above" | "below"

export interface TourStepConfig {
  titleKey: string
  descriptionKey: string
  /** Absent means the step is intentionally targetless and renders centred. */
  targetSelector?: string
  placement: TourPlacement
  /** Nudge along the block axis, for targets whose visual centre is not their
   *  geometric one. */
  offsetBlock?: number
}

/**
 * Exported so a structural test can assert that every selector here is carried
 * by a real control. The previous table included a `docs` step pointing at
 * `[data-tour='docs-button']`, an attribute no component has ever rendered; the
 * console has no documentation button for it to point at.
 */
export const TOUR_STEP_CONFIG: Record<
  Exclude<TourStep, "completed">,
  TourStepConfig
> = {
  welcome: {
    titleKey: "tour.welcome.title",
    descriptionKey: "tour.welcome.description",
    placement: "below",
  },
  models: {
    titleKey: "tour.models.title",
    descriptionKey: "tour.models.description",
    targetSelector: "[data-tour='models-nav']",
    placement: "end",
  },
  gateway: {
    titleKey: "tour.gateway.title",
    descriptionKey: "tour.gateway.description",
    targetSelector: "[data-tour='gateway-button']",
    placement: "start",
    offsetBlock: 60,
  },
}

export function isRtl(): boolean {
  return document.documentElement.getAttribute("dir") === "rtl"
}

export function clamp(value: number, min: number, max: number): number {
  return Math.min(Math.max(value, min), max)
}

/**
 * A target is only usable if a user could actually see it. An element that is
 * mounted but unlaid-out reports an all-zero rect, which the previous
 * implementation accepted: the spotlight then drew a 16px box at (-8,-8) and
 * dimmed the whole screen from the corner.
 */
export function isEligibleTarget(
  element: Element | null,
): element is HTMLElement {
  if (!(element instanceof HTMLElement) || !element.isConnected) {
    return false
  }
  const style = window.getComputedStyle(element)
  if (style.display === "none" || style.visibility === "hidden") {
    return false
  }
  const rect = element.getBoundingClientRect()
  return rect.width > 0 && rect.height > 0
}

/** The first *eligible* match, not merely the first match: responsive variants
 *  of one control can both be mounted while only one is laid out. */
export function resolveTourTarget(selector: string): HTMLElement | null {
  for (const candidate of Array.from(document.querySelectorAll(selector))) {
    if (isEligibleTarget(candidate)) {
      return candidate
    }
  }
  return null
}
