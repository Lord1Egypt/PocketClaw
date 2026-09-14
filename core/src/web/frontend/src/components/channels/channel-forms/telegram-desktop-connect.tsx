import {
  IconBrandTelegram,
  IconCopy,
  IconLoader2,
  IconRefresh,
} from "@tabler/icons-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type TelegramDesktopPairing,
  type TelegramPairingState,
  cancelTelegramPairing,
  completeTelegramPairing,
  createTelegramPairing,
  fetchTelegramPairingStatus,
} from "@/api/telegram-onboarding"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"

/**
 * Managed Telegram pairing from a client with no Android host.
 *
 * PC-DEF-060. The native flow launches Telegram and writes the token itself; a browser
 * cannot. Core runs the pairing, so this component only ever calls same-origin endpoints
 * and never handles a credential: it shows the Telegram link, polls state, and asks Core
 * to finish.
 *
 * The poll interval and the deadline both come from the pairing, so the service decides
 * them rather than this file guessing.
 */

type Phase =
  | { kind: "idle" }
  | { kind: "creating" }
  | { kind: "waiting"; pairing: TelegramDesktopPairing }
  | { kind: "configuring"; pairing: TelegramDesktopPairing }
  | { kind: "failed"; reason: string }

interface TelegramDesktopConnectProps {
  /** Called once Telegram is configured, so the page can reload its config. */
  onConnected: () => void
}

export function TelegramDesktopConnect({
  onConnected,
}: TelegramDesktopConnectProps) {
  const { t } = useTranslation()
  const [phase, setPhase] = useState<Phase>({ kind: "idle" })
  const [state, setState] = useState<TelegramPairingState>("pending")

  // Held in a ref so the polling effect can stop without being re-created, and so an
  // unmount cancels the pairing it started rather than leaving Core holding a token.
  const pairingRef = useRef<TelegramDesktopPairing | null>(null)
  const completingRef = useRef(false)

  useEffect(() => {
    return () => {
      const pairing = pairingRef.current
      // Only an unfinished pairing is cancelled: a completed one is already spent.
      if (pairing && !completingRef.current) {
        void cancelTelegramPairing(pairing.pairing_id)
      }
    }
  }, [])

  const fail = useCallback((reason: string) => {
    pairingRef.current = null
    completingRef.current = false
    setPhase({ kind: "failed", reason })
  }, [])

  const start = useCallback(async () => {
    setPhase({ kind: "creating" })
    setState("pending")
    try {
      const pairing = await createTelegramPairing()
      pairingRef.current = pairing
      completingRef.current = false
      setPhase({ kind: "waiting", pairing })
    } catch (error) {
      fail(
        error instanceof Error && error.message === "rate_limited"
          ? t("channels.telegram.desktop.errorRateLimited")
          : t("channels.telegram.desktop.errorStart"),
      )
    }
  }, [fail, t])

  const cancel = useCallback(async () => {
    const pairing = pairingRef.current
    pairingRef.current = null
    completingRef.current = false
    setPhase({ kind: "idle" })
    if (pairing) await cancelTelegramPairing(pairing.pairing_id)
  }, [])

  // Polls while a pairing is outstanding. The pairing's own expiry ends it, so this
  // cannot poll forever.
  useEffect(() => {
    if (phase.kind !== "waiting") return
    const pairing = phase.pairing
    const intervalMs = Math.max(1, pairing.poll_interval_seconds) * 1000
    const deadline = Date.parse(pairing.expires_at)
    let cancelled = false

    const tick = async () => {
      if (cancelled) return
      if (Number.isFinite(deadline) && Date.now() >= deadline) {
        fail(t("channels.telegram.desktop.errorExpired"))
        return
      }
      try {
        const status = await fetchTelegramPairingStatus(pairing.pairing_id)
        if (cancelled) return
        setState(status.state)
        if (status.state === "expired") {
          fail(t("channels.telegram.desktop.errorExpired"))
          return
        }
        if (status.state === "failed") {
          fail(t("channels.telegram.desktop.errorFailed"))
          return
        }
        if (status.state !== "ready") return

        // Ready means the token is collectable, exactly once, so guard against a
        // second tick racing the first.
        if (completingRef.current) return
        completingRef.current = true
        setPhase({ kind: "configuring", pairing })

        const result = await completeTelegramPairing(pairing.pairing_id)
        if (cancelled) return
        pairingRef.current = null
        if (result.pending) {
          // Saved but not yet live, which is not a failure — PC-DEF-030's rule.
          toast.info(t("channels.telegram.desktop.savedPending"))
        } else {
          toast.success(t("channels.telegram.desktop.connected"))
        }
        setPhase({ kind: "idle" })
        onConnected()
      } catch {
        if (!cancelled) fail(t("channels.telegram.desktop.errorFailed"))
      }
    }

    void tick()
    const timer = setInterval(() => void tick(), intervalMs)
    return () => {
      cancelled = true
      clearInterval(timer)
    }
  }, [phase, fail, onConnected, t])

  const copyLink = useCallback(async (link: string) => {
    try {
      await navigator.clipboard.writeText(link)
      toast.success(t("channels.telegram.desktop.linkCopied"))
    } catch {
      // A browser that refuses clipboard access is not an error worth a dialog; the
      // link is on screen and selectable.
      toast.error(t("channels.telegram.desktop.linkCopyFailed"))
    }
  }, [t])

  return (
    <Card className="shadow-sm" data-testid="telegram-desktop-connect">
      <CardContent className="space-y-4 px-6 py-5">
        <div>
          <p className="text-sm font-medium">
            {t("channels.telegram.desktop.title")}
          </p>
          <p className="text-muted-foreground mt-1 text-sm">
            {t("channels.telegram.desktop.body")}
          </p>
        </div>

        {phase.kind === "idle" && (
          <Button onClick={() => void start()} className="min-h-10">
            <IconBrandTelegram className="size-4" />
            {t("channels.telegram.desktop.connect")}
          </Button>
        )}

        {phase.kind === "creating" && (
          <p className="text-muted-foreground flex items-center gap-2 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            {t("channels.telegram.desktop.creating")}
          </p>
        )}

        {phase.kind === "waiting" && (
          <div className="space-y-3">
            <div>
              <p className="text-muted-foreground text-xs tracking-wide uppercase">
                {t("channels.telegram.desktop.suggestedBot")}
              </p>
              <p className="font-mono text-sm">
                @{phase.pairing.suggested_username}
              </p>
            </div>

            <div className="flex flex-wrap gap-2">
              {/* The link Core returned is a Telegram link by construction — it
                  validates that before returning it — so this never sends the user
                  to a hosting origin. */}
              <Button asChild className="min-h-10">
                <a
                  href={phase.pairing.deep_link}
                  target="_blank"
                  rel="noreferrer noopener"
                >
                  <IconBrandTelegram className="size-4" />
                  {t("channels.telegram.desktop.openTelegram")}
                </a>
              </Button>
              {/* For finishing on a phone instead of this machine. */}
              <Button
                variant="outline"
                className="min-h-10"
                onClick={() => void copyLink(phase.pairing.deep_link)}
              >
                <IconCopy className="size-4" />
                {t("channels.telegram.desktop.copyLink")}
              </Button>
              <Button
                variant="ghost"
                className="min-h-10"
                onClick={() => void cancel()}
              >
                {t("common.cancel")}
              </Button>
            </div>

            <p className="text-muted-foreground text-sm">
              {state === "created"
                ? t("channels.telegram.desktop.botCreated")
                : t("channels.telegram.desktop.waiting")}
            </p>
          </div>
        )}

        {phase.kind === "configuring" && (
          <p className="text-muted-foreground flex items-center gap-2 text-sm">
            <IconLoader2 className="size-4 animate-spin" />
            {t("channels.telegram.desktop.configuring")}
          </p>
        )}

        {phase.kind === "failed" && (
          <div className="space-y-3">
            <p className="text-destructive text-sm" role="alert">
              {phase.reason}
            </p>
            <Button
              variant="outline"
              className="min-h-10"
              onClick={() => void start()}
            >
              <IconRefresh className="size-4" />
              {t("channels.telegram.desktop.retry")}
            </Button>
          </div>
        )}
      </CardContent>
    </Card>
  )
}
