import { IconChevronLeft, IconChevronRight } from "@tabler/icons-react"
import { useAtom } from "jotai"
import * as React from "react"
import { useTranslation } from "react-i18next"

import {
  TOUR_STEP_CONFIG,
  type TourPlacement,
  type TourStepConfig,
  clamp,
  isEligibleTarget,
  isRtl,
  resolveTourTarget,
} from "@/components/tour/tour-steps"
import { Button } from "@/components/ui/button"
import { cn } from "@/lib/utils"
import {
  TOUR_VISIBLE_STEPS,
  type TourStep,
  tourAtom,
  tourCurrentStepAtom,
  tourIsActiveAtom,
  useTourActions,
} from "@/store/tour"

/** Gap between target and card. */
const GAP = 12
/** Every edge of the card stays at least this far inside the viewport. */
const SAFE_MARGIN = 8
/** The card's declared size (`w-80` plus typical content), used only when the
 *  live measurement reports zero. */
const CARD_NOMINAL_WIDTH = 320
const CARD_NOMINAL_HEIGHT = 200
/**
 * How many animation frames a step will wait for its target before giving up
 * and rendering centred. A frame budget rather than a timeout: it is tied to
 * the render loop that would produce the target, it is deterministic in tests,
 * and it cannot wait forever.
 */
const TARGET_FRAME_BUDGET = 90

function sameRect(a: DOMRect | null, b: DOMRect): boolean {
  return (
    a !== null &&
    a.top === b.top &&
    a.left === b.left &&
    a.width === b.width &&
    a.height === b.height
  )
}

/**
 * Resolves one step's target and then keeps its geometry current for as long as
 * the step is on screen.
 *
 * The old implementation measured once, during the render that the click
 * produced, and never again. Anything that moved the target afterwards — a
 * scroll, a resize, the header finishing its first data load — left the
 * highlight behind at coordinates that no longer meant anything.
 */
function useAnchoredTarget(selector: string | undefined, step: TourStep) {
  const [rect, setRect] = React.useState<DOMRect | null>(null)

  React.useLayoutEffect(() => {
    if (!selector) {
      setRect(null)
      return
    }

    let cancelled = false
    let frame = 0
    let framesLeft = TARGET_FRAME_BUDGET
    let target: HTMLElement | null = null
    let resizeObserver: ResizeObserver | null = null

    const measure = () => {
      if (cancelled || !target) {
        return
      }
      if (!isEligibleTarget(target)) {
        release()
        hunt()
        return
      }
      const next = target.getBoundingClientRect()
      setRect((previous) => (sameRect(previous, next) ? previous : next))
    }

    const release = () => {
      resizeObserver?.disconnect()
      resizeObserver = null
      target = null
      setRect(null)
    }

    const adopt = (element: HTMLElement) => {
      target = element
      // Bring it into view through its own scroll container rather than
      // scrolling the page: the sidebar and several settings panes scroll
      // independently of the document.
      if (typeof element.scrollIntoView === "function") {
        element.scrollIntoView({ block: "nearest", inline: "nearest" })
      }
      if (typeof ResizeObserver !== "undefined") {
        resizeObserver = new ResizeObserver(measure)
        resizeObserver.observe(element)
      }
      measure()
    }

    const hunt = () => {
      if (cancelled || target) {
        return
      }
      const found = resolveTourTarget(selector)
      if (found) {
        adopt(found)
        return
      }
      if (framesLeft <= 0) {
        // Budget spent. The step degrades to a centred card rather than
        // waiting for an element that is not coming.
        return
      }
      framesLeft -= 1
      frame = requestAnimationFrame(hunt)
    }

    const onViewportChange = () => measure()

    // Capture phase, so a scroll inside any container reaches this too.
    window.addEventListener("resize", onViewportChange)
    window.addEventListener("scroll", onViewportChange, true)

    let mutationObserver: MutationObserver | null = null
    if (typeof MutationObserver !== "undefined") {
      mutationObserver = new MutationObserver(() => {
        if (cancelled) {
          return
        }
        if (target && !target.isConnected) {
          // Replaced or unmounted — a responsive swap, or a re-render. Look
          // again from scratch with a fresh budget.
          release()
          framesLeft = TARGET_FRAME_BUDGET
          hunt()
        } else if (!target) {
          const found = resolveTourTarget(selector)
          if (found) {
            adopt(found)
          }
        }
      })
      mutationObserver.observe(document.body, {
        childList: true,
        subtree: true,
      })
    }

    hunt()

    return () => {
      cancelled = true
      cancelAnimationFrame(frame)
      window.removeEventListener("resize", onViewportChange)
      window.removeEventListener("scroll", onViewportChange, true)
      resizeObserver?.disconnect()
      mutationObserver?.disconnect()
    }
  }, [selector, step])

  return rect
}

/**
 * Places the card and then clamps it inside the viewport.
 *
 * The clamp is the guarantee, not the placement: whatever the layout does, the
 * card — and therefore Next — stays reachable.
 */
function usePopoverPosition(
  rect: DOMRect | null,
  placement: TourPlacement,
  offsetBlock: number,
) {
  const ref = React.useRef<HTMLDivElement | null>(null)
  const [position, setPosition] = React.useState<{
    top: number
    left: number
  } | null>(null)
  const [viewport, setViewport] = React.useState(0)

  // A resize changes the clamp even when the target has not moved an inch, so
  // the card has to be placed again rather than only re-measured.
  React.useEffect(() => {
    const onResize = () => setViewport((tick) => tick + 1)
    window.addEventListener("resize", onResize)
    return () => window.removeEventListener("resize", onResize)
  }, [])

  React.useLayoutEffect(() => {
    const node = ref.current
    if (!node) {
      return
    }
    if (!rect) {
      setPosition((previous) => (previous === null ? previous : null))
      return
    }

    const size = node.getBoundingClientRect()
    // A card that reports no size would be clamped against zero and could then
    // be placed past the edge — the exact failure the clamp exists to prevent.
    // Fall back to its declared dimensions instead of trusting a zero.
    const width = size.width || CARD_NOMINAL_WIDTH
    const height = size.height || CARD_NOMINAL_HEIGHT
    const rtl = isRtl()

    let top: number
    let left: number
    switch (placement) {
      case "end":
        left = rtl ? rect.left - GAP - width : rect.right + GAP
        top = rect.top + rect.height / 2 - height / 2 + offsetBlock
        break
      case "start":
        left = rtl ? rect.right + GAP : rect.left - GAP - width
        top = rect.top + rect.height / 2 - height / 2 + offsetBlock
        break
      case "above":
        left = rect.left + rect.width / 2 - width / 2
        top = rect.top - GAP - height
        break
      default:
        left = rect.left + rect.width / 2 - width / 2
        top = rect.bottom + GAP
        break
    }

    const next = {
      left: clamp(
        left,
        SAFE_MARGIN,
        Math.max(SAFE_MARGIN, window.innerWidth - width - SAFE_MARGIN),
      ),
      top: clamp(
        top,
        SAFE_MARGIN,
        Math.max(SAFE_MARGIN, window.innerHeight - height - SAFE_MARGIN),
      ),
    }
    setPosition((previous) =>
      previous && previous.top === next.top && previous.left === next.left
        ? previous
        : next,
    )
    // Intentionally dependency-free apart from the viewport tick: the card's
    // own measured size is an input, and that is only known after a render.
    // The equality guard above makes the repeat runs settle immediately.
  }, [rect, placement, offsetBlock, viewport])

  return { ref, position }
}

/** Mounted only while the tour is running, so its unmount cleanup is the one
 *  place that has to restore focus — every way out of the tour goes through
 *  it. */
function TourOverlay({
  step,
  config,
  onNext,
  onPrev,
  onSkip,
}: {
  step: Exclude<TourStep, "completed">
  config: TourStepConfig
  onNext: () => void
  onPrev: () => void
  onSkip: () => void
}) {
  const { t } = useTranslation()
  const rect = useAnchoredTarget(config.targetSelector, step)
  const { ref, position } = usePopoverPosition(
    rect,
    config.placement,
    config.offsetBlock ?? 0,
  )

  // Focus ownership. The tour takes focus when it opens and gives it back when
  // it closes, by any route. Without this a user who reached the tour from a
  // focused control was left with that control's focus ring still painted
  // after the tour was gone, which reads as a highlight the tour forgot to
  // clear.
  React.useEffect(() => {
    const previous =
      document.activeElement instanceof HTMLElement &&
      document.activeElement !== document.body
        ? document.activeElement
        : null
    ref.current?.focus()
    return () => {
      if (previous && previous.isConnected) {
        previous.focus()
      } else if (document.activeElement instanceof HTMLElement) {
        document.activeElement.blur()
      }
    }
    // Mount/unmount only: re-running per step would hand focus back mid-tour.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [])

  React.useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === "Escape") {
        event.preventDefault()
        onSkip()
      }
    }
    document.addEventListener("keydown", onKeyDown)
    return () => document.removeEventListener("keydown", onKeyDown)
  }, [onSkip])

  const stepIndex = TOUR_VISIBLE_STEPS.indexOf(step)
  const totalSteps = TOUR_VISIBLE_STEPS.length
  const isLastStep = stepIndex === totalSteps - 1
  const anchored = rect !== null

  return (
    <>
      {/* The dimmer owns dismissal. It is transparent when a spotlight is
          present, because the spotlight's ring shadow does the dimming there. */}
      <div
        data-tour-layer="dimmer"
        className={cn(
          "fixed inset-0 z-[100]",
          !anchored && "bg-black/20 backdrop-blur-[2px]",
        )}
        onClick={onSkip}
      />

      {anchored && (
        <>
          {/* The spotlight blocks the target instead of letting the click fall
              through to it. The tour's own buttons own progression; a click
              that reached the real control used to navigate the app mid-tour
              and leave its focus ring behind. */}
          <div
            data-tour-layer="spotlight"
            className="fixed z-[101] transition-all duration-300"
            style={{
              top: rect.top - 8,
              left: rect.left - 8,
              width: rect.width + 16,
              height: rect.height + 16,
              boxShadow:
                "0 0 0 9999px rgba(0, 0, 0, 0.2), 0 0 2px 9999px rgba(0, 0, 0, 0.1)",
              borderRadius: "12px",
            }}
            onClick={(event) => event.stopPropagation()}
          />
          <div
            data-tour-layer="ring"
            className="ring-primary ring-offset-background pointer-events-none fixed z-[101] rounded-lg ring-2 ring-offset-2 transition-all duration-300"
            style={{
              top: rect.top - 4,
              left: rect.left - 4,
              width: rect.width + 8,
              height: rect.height + 8,
            }}
          />
        </>
      )}

      <div
        ref={ref}
        data-tour-layer="card"
        role="dialog"
        aria-modal="true"
        aria-label={t(config.titleKey)}
        tabIndex={-1}
        className={cn(
          "bg-background fixed z-[102] w-80 rounded-xl border p-4 shadow-2xl outline-hidden",
          !position && "max-w-md",
        )}
        style={
          position
            ? { top: position.top, left: position.left }
            : { top: "50%", left: "50%", transform: "translate(-50%, -50%)" }
        }
      >
        <div className="mb-3 flex items-center gap-2">
          <h3 className="font-semibold">{t(config.titleKey)}</h3>
        </div>

        <p className="text-muted-foreground mb-4 text-sm leading-relaxed">
          {t(config.descriptionKey)}
        </p>

        <div className="flex items-center justify-between">
          <div className="text-muted-foreground text-xs">
            {stepIndex + 1} / {totalSteps}
          </div>

          <div className="flex items-center gap-2">
            {stepIndex > 0 && (
              <Button variant="outline" size="sm" onClick={onPrev}>
                <IconChevronLeft className="size-4 rtl:rotate-180" />
                {t("tour.prev")}
              </Button>
            )}
            <Button size="sm" onClick={onNext}>
              {isLastStep ? t("tour.finish") : t("tour.next")}
              {!isLastStep && (
                <IconChevronRight className="size-4 rtl:rotate-180" />
              )}
            </Button>
          </div>
        </div>

        {!isLastStep && (
          <Button
            variant="link"
            size="sm"
            className="mt-2 h-auto p-0 text-xs"
            onClick={onSkip}
          >
            {t("tour.skip")}
          </Button>
        )}
      </div>
    </>
  )
}

export function TourGuide() {
  const [tourState] = useAtom(tourAtom)
  const [, setCurrentStep] = useAtom(tourCurrentStepAtom)
  const [, setIsActive] = useAtom(tourIsActiveAtom)
  const { goToNextStep, goToPrevStep } = useTourActions()

  const step = tourState.currentStep

  const handleNext = React.useCallback(() => {
    const nextStep = goToNextStep(step)
    setCurrentStep(nextStep)
    if (nextStep === "completed") {
      setIsActive(false)
    }
  }, [goToNextStep, setCurrentStep, setIsActive, step])

  const handlePrev = React.useCallback(() => {
    setCurrentStep(goToPrevStep(step))
  }, [goToPrevStep, setCurrentStep, step])

  const handleSkip = React.useCallback(() => {
    setCurrentStep("completed")
    setIsActive(false)
  }, [setCurrentStep, setIsActive])

  if (!tourState.isActive || step === "completed") {
    return null
  }

  const config = TOUR_STEP_CONFIG[step]
  if (!config) {
    return null
  }

  return (
    <TourOverlay
      key={step}
      step={step}
      config={config}
      onNext={handleNext}
      onPrev={handlePrev}
      onSkip={handleSkip}
    />
  )
}
