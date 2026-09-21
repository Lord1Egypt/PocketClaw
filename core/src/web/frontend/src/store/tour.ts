import { atom } from "jotai"
import { atomWithStorage, createJSONStorage } from "jotai/utils"

/**
 * Bump only when the tour itself changes in a way existing users should see
 * again. Changing it replays the tour for everyone, including people who
 * already finished it, so it is deliberately not tied to the app version.
 */
export const TOUR_VERSION = 1

export type TourStep = "welcome" | "models" | "gateway" | "completed"

/** The whole sequence, terminator included. This is the single source of truth:
 *  the component derives its step table and its counter from it. */
export const TOUR_SEQUENCE: readonly TourStep[] = [
  "welcome",
  "models",
  "gateway",
  "completed",
] as const

/** The steps that actually render a card. */
export const TOUR_VISIBLE_STEPS = TOUR_SEQUENCE.filter(
  (step): step is Exclude<TourStep, "completed"> => step !== "completed",
)

export interface TourState {
  version: number
  currentStep: TourStep
  isActive: boolean
}

const STORAGE_KEY = "pocketclaw-tour-state"

const DEFAULT_TOUR_STATE: TourState = {
  version: TOUR_VERSION,
  currentStep: "welcome",
  isActive: true,
}

const COMPLETED_TOUR_STATE: TourState = {
  version: TOUR_VERSION,
  currentStep: "completed",
  isActive: false,
}

function isKnownStep(value: unknown): value is TourStep {
  return typeof value === "string" && TOUR_SEQUENCE.includes(value as TourStep)
}

/**
 * Brings any previously stored value onto the current schema.
 *
 * The first release of the tour stored `{currentStep, isActive}` with no
 * version, so an absent `version` means 0 rather than corruption. Three rules,
 * in this order, decide what a stored value becomes:
 *
 *   1. Anyone who finished stays finished. Replaying onboarding for a user who
 *      already dismissed it is the one outcome a migration must never cause.
 *   2. A step this build no longer defines cannot be resumed, so the tour is
 *      marked finished rather than left pointing at nothing. A user parked on
 *      the removed `docs` step had already seen every other step.
 *   3. A newer TOUR_VERSION deliberately replays the tour; that is what the
 *      constant is for.
 *
 * Anything unreadable falls back to the default rather than throwing: a corrupt
 * preference must not stop the console from rendering.
 */
export function migrateTourState(raw: unknown): TourState {
  if (raw === null || typeof raw !== "object") {
    return DEFAULT_TOUR_STATE
  }
  const value = raw as Partial<Record<keyof TourState, unknown>>
  const isActive = typeof value.isActive === "boolean" ? value.isActive : true
  const storedVersion = typeof value.version === "number" ? value.version : 0

  if (!isActive || value.currentStep === "completed") {
    return COMPLETED_TOUR_STATE
  }
  if (!isKnownStep(value.currentStep)) {
    return COMPLETED_TOUR_STATE
  }
  if (storedVersion > TOUR_VERSION) {
    // Written by a newer build. Leave that build's state alone rather than
    // downgrading it into something it did not mean.
    return { version: storedVersion, currentStep: value.currentStep, isActive }
  }
  if (storedVersion < TOUR_VERSION) {
    return { ...DEFAULT_TOUR_STATE }
  }
  return { version: TOUR_VERSION, currentStep: value.currentStep, isActive }
}

/** localStorage is unavailable in a few real contexts — a privacy-mode window,
 *  a sandboxed frame. The tour is a nicety, so it degrades to memory rather
 *  than taking the page down with it. */
function resolveStorage(): Storage {
  try {
    const probe = "__pocketclaw_tour_probe__"
    window.localStorage.setItem(probe, probe)
    window.localStorage.removeItem(probe)
    return window.localStorage
  } catch {
    const memory = new Map<string, string>()
    return {
      get length() {
        return memory.size
      },
      clear: () => memory.clear(),
      getItem: (key) => memory.get(key) ?? null,
      key: (index) => Array.from(memory.keys())[index] ?? null,
      removeItem: (key) => void memory.delete(key),
      setItem: (key, value) => void memory.set(key, value),
    }
  }
}

const jsonStorage = createJSONStorage<TourState>(resolveStorage)

const migratingStorage = {
  ...jsonStorage,
  getItem: (key: string, initialValue: TourState) =>
    migrateTourState(jsonStorage.getItem(key, initialValue)),
}

export const tourAtom = atomWithStorage<TourState>(
  STORAGE_KEY,
  DEFAULT_TOUR_STATE,
  migratingStorage,
  { getOnInit: true },
)

export const tourIsActiveAtom = atom(
  (get) => get(tourAtom).isActive,
  (get, set, isActive: boolean) => {
    set(tourAtom, { ...get(tourAtom), isActive })
  },
)

export const tourCurrentStepAtom = atom(
  (get) => get(tourAtom).currentStep,
  (get, set, step: TourStep) => {
    set(tourAtom, { ...get(tourAtom), currentStep: step })
  },
)

export function useTourActions() {
  const goToNextStep = (currentStep: TourStep): TourStep => {
    const currentIndex = TOUR_SEQUENCE.indexOf(currentStep)
    if (currentIndex >= 0 && currentIndex < TOUR_SEQUENCE.length - 1) {
      return TOUR_SEQUENCE[currentIndex + 1]
    }
    return "completed"
  }

  const goToPrevStep = (currentStep: TourStep): TourStep => {
    const currentIndex = TOUR_SEQUENCE.indexOf(currentStep)
    if (currentIndex > 0) {
      return TOUR_SEQUENCE[currentIndex - 1]
    }
    return currentStep
  }

  return { goToNextStep, goToPrevStep }
}
