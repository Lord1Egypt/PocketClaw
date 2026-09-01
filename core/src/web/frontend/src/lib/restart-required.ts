import type { TFunction } from "i18next"
import { toast } from "sonner"

import { applyGatewayConfig } from "@/api/gateway"
import { refreshGatewayState } from "@/store/gateway"

export function showRestartRequiredToast(t: TFunction, name: string) {
  toast.warning(t("common.restartRequiredTitle"), {
    description: t("common.restartRequiredDesc", { name }),
  })
}

export function showSaveSuccessOrRestartToast(
  t: TFunction,
  savedMessage: string,
  name: string,
  restartRequired: boolean,
) {
  if (restartRequired) {
    showRestartRequiredToast(t, name)
    return
  }
  toast.success(savedMessage)
}

/**
 * How long to wait for the gateway to come back before reporting the restart as
 * failed. A restart that returns HTTP 200 has only been *requested*; readiness
 * is what the user actually cares about.
 */
const READY_TIMEOUT_MS = 30_000
const READY_POLL_INTERVAL_MS = 500

/**
 * Serialises automatic restarts and coalesces bursts of saves.
 *
 * Several restart-requiring saves in quick succession — model, then fallbacks,
 * then streaming — should produce one effective restart, not three. A save that
 * arrives while a restart is already running waits for it and then re-checks
 * whether a further restart is still needed, because the running one may
 * already have picked up the newer config.
 */
let pendingRestart: Promise<ApplyOutcome> | null = null

async function waitForGatewayReady(): Promise<boolean> {
  const deadline = Date.now() + READY_TIMEOUT_MS
  while (Date.now() < deadline) {
    const state = await refreshGatewayState({ force: true })
    // Ready means the process is back *and* the config it booted with matches
    // the saved one. Checking only "running" would report success while the
    // gateway was still serving the previous configuration.
    if (state?.status === "running" && state.restartRequired === false) {
      return true
    }
    if (state?.status === "error") {
      return false
    }
    await new Promise((resolve) => setTimeout(resolve, READY_POLL_INTERVAL_MS))
  }
  return false
}

/**
 * What happened to a saved configuration change.
 *
 * `not_required` means the backend's signature comparison said no restart was
 * needed. `not_applied` means the gateway was deliberately left alone because
 * it was busy — the change is saved, and it is not an error.
 */
export type ApplyOutcome =
  | "not_required"
  | "restarted"
  | "not_applied"
  | "failed"

async function restartAndWait(reason: string): Promise<ApplyOutcome> {
  // apply-config rather than the manual restart: it holds the restart until the
  // gateway is idle, so saving settings never cuts off an answer in progress.
  const response = await applyGatewayConfig(reason)

  // The gateway was busy, or would not say whether it was busy, so it was left
  // alone on purpose. The configuration is saved; it is not live yet. This is a
  // deliberate outcome, not a failure, and must not be reported as one.
  if (response.status === "saved_not_applied") {
    return "not_applied"
  }

  return (await waitForGatewayReady()) ? "restarted" : "failed"
}

export interface SaveAndApplyOptions<T> {
  /** Persists the configuration. Must resolve before any restart is attempted. */
  save: () => Promise<T>
  /** Shown when the save needed no restart, or the restart succeeded. */
  savedMessage: string
  /** Human-readable name of what was saved, used in restart messages. */
  name: string
  /**
   * Reports the outcome instead of the generic toasts.
   *
   * A surface that shows its own state — where "saved, applying automatically"
   * belongs in the card rather than in a toast telling the user to restart
   * something themselves — passes this and renders the outcome itself. Callers
   * that do not pass it keep the toasts exactly as before.
   */
  onOutcome?: (outcome: ApplyOutcome) => void
}

/**
 * Saves a configuration change and, only if the backend says the change
 * requires it, restarts the gateway and waits for it to be ready.
 *
 * The restart decision is not made here. It comes from the backend's existing
 * `gateway_restart_required` signature comparison, so cosmetic edits, unchanged
 * saves and hot-reloadable fields do not trigger one and there is no second
 * decision system to keep in agreement with the first.
 *
 * A failed restart never discards the saved configuration: the config is
 * already persisted before the restart is attempted, and the user is told to
 * use the manual Restart control, which remains available.
 */
/**
 * Applies a configuration change that has already been persisted.
 *
 * Use this where the save happened earlier in the handler and only the "apply
 * it" half is needed. `saveAndApplyGatewayConfig` is the same flow for callers
 * that still have the save to perform.
 */
export async function applyGatewayConfigIfRequired(
  t: TFunction,
  options: { savedMessage: string; name: string },
): Promise<void> {
  await saveAndApplyGatewayConfig(t, {
    save: async () => undefined,
    savedMessage: options.savedMessage,
    name: options.name,
  })
}

export async function saveAndApplyGatewayConfig<T>(
  t: TFunction,
  options: SaveAndApplyOptions<T>,
): Promise<T> {
  const result = await options.save()
  const report = options.onOutcome

  const state = await refreshGatewayState({ force: true })
  if (state?.restartRequired !== true) {
    // Nothing to apply: the gateway is stopped, still starting, or already
    // running the saved configuration. None of those is a failure.
    if (report) {
      report("not_required")
    } else {
      toast.success(options.savedMessage)
    }
    return result
  }

  const restartToast = report
    ? ""
    : toast.loading(t("common.restartingGateway"))

  try {
    // Join an in-flight restart rather than starting a second one.
    if (pendingRestart) {
      await pendingRestart
      const after = await refreshGatewayState({ force: true })
      if (after?.restartRequired !== true) {
        if (report) {
          report("restarted")
        } else {
          toast.success(options.savedMessage, { id: restartToast })
        }
        return result
      }
    }

    pendingRestart = restartAndWait(options.name)
    const outcome = await pendingRestart

    if (report) {
      report(outcome)
    } else if (outcome === "restarted") {
      toast.success(t("common.restartedGateway"), { id: restartToast })
    } else if (outcome === "not_applied") {
      // Saved, but deliberately not applied: something was still running. The
      // manual Restart Gateway control applies it when the user is ready.
      toast.warning(t("common.restartDeferredTitle"), {
        id: restartToast,
        description: t("common.restartDeferredDesc", { name: options.name }),
      })
    } else {
      toast.error(t("common.restartFailedTitle"), {
        id: restartToast,
        description: t("common.restartFailedDesc", { name: options.name }),
      })
    }
  } catch (e) {
    if (report) {
      report("failed")
    } else {
      toast.error(t("common.restartFailedTitle"), {
        id: restartToast,
        description:
          e instanceof Error
            ? e.message
            : t("common.restartFailedDesc", { name: options.name }),
      })
    }
  } finally {
    pendingRestart = null
  }

  return result
}
