import {
  IconAlertTriangle,
  IconFlask,
  IconLoader2,
  IconPlugConnected,
  IconPlugConnectedX,
  IconQrcode,
} from "@tabler/icons-react"
import { useCallback, useEffect, useRef, useState } from "react"
import { useTranslation } from "react-i18next"

import {
  type ChannelConfig,
  WHATSAPP_AGENT_QR_URL,
  type WhatsAppAgentStatus,
  forgetWhatsAppAgentSession,
  getChannelConfig,
  getWhatsAppAgentPairCode,
  getWhatsAppAgentStatus,
} from "@/api/channels"
import { normalizeWhatsAppNumber } from "@/components/channels/channel-forms/whatsapp-self-chat"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import type { ApplyOutcome } from "@/lib/restart-required"

interface WhatsAppAgentPanelProps {
  config: ChannelConfig
  /** Persists the channel block and applies it, with an explicit enabled flag. */
  onPersist: (
    nextConfig: ChannelConfig,
    nextEnabled: boolean,
  ) => Promise<ApplyOutcome>
}

/** While a code is live or a session is coming up, state changes are worth
 * following closely; otherwise a slow poll is enough to notice a drop. */
const FAST_POLL_MS = 2000
const IDLE_POLL_MS = 10000

/**
 * WhatsApp shows companion codes in two groups of four. The value is displayed
 * grouped and copied whole, so the dash is presentation only and never reaches
 * the pairing state.
 */
function formatPairCode(code: string): string {
  const trimmed = code.trim()
  if (trimmed.length !== 8) return trimmed
  return `${trimmed.slice(0, 4)}-${trimmed.slice(4)}`
}

const STATE_KEY: Record<WhatsAppAgentStatus["state"], string> = {
  unavailable: "channels.whatsappAgent.stateUnavailable",
  not_paired: "channels.whatsappAgent.stateNotPaired",
  pairing: "channels.whatsappAgent.statePairing",
  connecting: "channels.whatsappAgent.stateConnecting",
  connected: "channels.whatsappAgent.stateConnected",
  disconnected: "channels.whatsappAgent.stateDisconnected",
  logged_out: "channels.whatsappAgent.stateLoggedOut",
}

/**
 * Channels → WhatsApp Agent Channel (experimental).
 *
 * A real transport on the user's personal WhatsApp account, over the
 * multi-device protocol. It is deliberately not the old "WhatsApp" and
 * "WhatsApp Native" cards: it asks for no bridge URL and no session store path,
 * because neither is the user's to choose. The session database is owned by
 * Android, and the pairing code is rendered server-side so it never becomes a
 * string this page could leak.
 */
export function WhatsAppAgentPanel({
  config,
  onPersist,
}: WhatsAppAgentPanelProps) {
  const { t } = useTranslation()
  const [status, setStatus] = useState<WhatsAppAgentStatus | null>(null)
  const [selfNumber, setSelfNumber] = useState<string>("")
  const [busy, setBusy] = useState(false)
  const [error, setError] = useState("")
  const [outcome, setOutcome] = useState<ApplyOutcome | null>(null)
  // Cache-busts the QR image so a new code replaces the old one in the tab.
  const [qrNonce, setQrNonce] = useState(0)
  const previousHasQR = useRef(false)
  // The companion code, held only for as long as it is live. It is fetched
  // separately from the polled status so the polled response carries no
  // credential, and dropped the moment pairing leaves the pairing state.
  const [pairCode, setPairCode] = useState("")
  const [showQrFallback, setShowQrFallback] = useState(false)

  const refresh = useCallback(async () => {
    try {
      const next = await getWhatsAppAgentStatus()
      setStatus(next)
      if (next.has_qr && !previousHasQR.current) {
        setQrNonce((n) => n + 1)
      }
      previousHasQR.current = next.has_qr

      if (next.has_code) {
        try {
          setPairCode((await getWhatsAppAgentPairCode()).code)
        } catch {
          setPairCode("")
        }
      } else {
        // Paired, cancelled, timed out or disconnected — the code is dead and
        // must not stay on screen.
        setPairCode("")
      }
    } catch {
      // A failed poll is not worth a visible error: the next tick retries, and
      // the panel keeps showing the last state it actually knew.
    }
  }, [])

  useEffect(() => {
    void refresh()
  }, [refresh])

  // The self number is what seeds the allow-list. This channel denies every
  // sender that is not on it, so without a number there is nobody to allow.
  useEffect(() => {
    let cancelled = false
    void (async () => {
      try {
        const response = await getChannelConfig("whatsapp_self_chat")
        if (cancelled) return
        const stored = response.config.self_number
        const parsed = normalizeWhatsAppNumber(
          typeof stored === "string" ? stored : "",
        )
        setSelfNumber(parsed.ok ? parsed.number : "")
      } catch {
        if (!cancelled) setSelfNumber("")
      }
    })()
    return () => {
      cancelled = true
    }
  }, [])

  const state = status?.state ?? "unavailable"
  const active = state === "pairing" || state === "connecting"

  useEffect(() => {
    const interval = window.setInterval(
      () => void refresh(),
      active ? FAST_POLL_MS : IDLE_POLL_MS,
    )
    return () => window.clearInterval(interval)
  }, [active, refresh])

  const handlePair = useCallback(async () => {
    if (selfNumber === "") {
      setError(t("channels.whatsappAgent.errorNoSelfNumber"))
      return
    }
    setBusy(true)
    setError("")
    setOutcome(null)
    try {
      // use_native is written here rather than shown as a control: bridge mode
      // is not on offer, so the checkbox would have exactly one valid value.
      // allow_from carries the user's own number and nothing else — this
      // channel reaches an agent with shell tools, so it must start closed.
      setOutcome(
        await onPersist(
          { ...config, use_native: true, allow_from: [selfNumber] },
          true,
        ),
      )
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : t("channels.page.saveError"))
    } finally {
      setBusy(false)
    }
  }, [config, onPersist, refresh, selfNumber, t])

  const handleDisconnect = useCallback(async () => {
    setBusy(true)
    setError("")
    setOutcome(null)
    try {
      // Disable and restart first, so nothing holds the session database open
      // when it is erased.
      setOutcome(await onPersist({ ...config, use_native: true }, false))
      await forgetWhatsAppAgentSession()
      await refresh()
    } catch (e) {
      setError(e instanceof Error ? e.message : t("channels.page.saveError"))
    } finally {
      setBusy(false)
    }
  }, [config, onPersist, refresh, t])

  if (status && !status.available) {
    return (
      <Card>
        <CardContent className="text-muted-foreground py-6 text-sm">
          {t("channels.whatsappAgent.unavailable")}
        </CardContent>
      </Card>
    )
  }

  const paired =
    state === "connected" || state === "connecting" || state === "disconnected"

  return (
    <div className="flex flex-col gap-4">
      <Card className="border-amber-500/40 bg-amber-500/5">
        <CardContent className="flex gap-3 py-4">
          <IconFlask className="mt-0.5 shrink-0 text-amber-600" size={18} />
          <div className="flex flex-col gap-1">
            <p className="text-sm font-medium">
              {t("channels.whatsappAgent.experimentalTitle")}
            </p>
            <p className="text-muted-foreground text-sm">
              {t("channels.whatsappAgent.experimentalBody")}
            </p>
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="flex flex-col gap-4 py-4">
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium">
              {t("channels.whatsappAgent.statusLabel")}
            </span>
            <span className="text-muted-foreground text-sm" data-testid="wa-agent-state">
              {t(STATE_KEY[state])}
            </span>
            {busy && (
              <IconLoader2
                className="text-muted-foreground animate-spin"
                size={16}
              />
            )}
          </div>

          {pairCode !== "" && (
            <div className="flex flex-col items-center gap-3 py-2">
              <p className="text-muted-foreground text-center text-sm">
                {t("channels.whatsappAgent.codeHint")}
              </p>
              <div
                data-testid="wa-agent-pair-code"
                className="bg-muted rounded-lg px-6 py-4 font-mono text-3xl tracking-[0.3em] select-all"
              >
                {formatPairCode(pairCode)}
              </div>
              <p className="text-muted-foreground text-center text-xs">
                {t("channels.whatsappAgent.codeExpiry")}
              </p>
            </div>
          )}

          {status?.has_qr && (
            <div className="flex flex-col items-center gap-2 py-1">
              <button
                type="button"
                onClick={() => setShowQrFallback((shown) => !shown)}
                className="text-muted-foreground hover:text-foreground flex items-center gap-2 text-sm underline-offset-4 hover:underline"
              >
                <IconQrcode size={16} />
                {t("channels.whatsappAgent.qrFallbackToggle")}
              </button>
              {showQrFallback && (
                <div className="flex flex-col items-center gap-2 pt-2">
                  <p className="text-muted-foreground max-w-sm text-center text-xs">
                    {t("channels.whatsappAgent.qrFallbackHint")}
                  </p>
                  <img
                    // Served as an image so the pairing payload is never a
                    // string in this page. Single-use and short-lived, so it is
                    // re-fetched rather than cached.
                    src={`${WHATSAPP_AGENT_QR_URL}?v=${qrNonce}`}
                    alt={t("channels.whatsappAgent.qrAlt")}
                    className="h-64 w-64 rounded border bg-white p-2"
                  />
                </div>
              )}
            </div>
          )}

          {selfNumber === "" && (
            <div className="flex gap-2 text-amber-600">
              <IconAlertTriangle className="mt-0.5 shrink-0" size={16} />
              <p className="text-sm">
                {t("channels.whatsappAgent.errorNoSelfNumber")}
              </p>
            </div>
          )}

          {selfNumber !== "" && (
            <p className="text-muted-foreground text-sm">
              {t("channels.whatsappAgent.allowedSender", { number: selfNumber })}
            </p>
          )}

          {error !== "" && <p className="text-destructive text-sm">{error}</p>}

          {outcome === "not_applied" && (
            <p className="text-muted-foreground text-sm">
              {t("channels.whatsappAgent.applyDeferred")}
            </p>
          )}

          <div className="flex gap-2">
            <Button
              onClick={() => void handlePair()}
              disabled={busy || selfNumber === ""}
              className="gap-2"
            >
              {paired ? (
                <IconPlugConnected size={16} />
              ) : (
                <IconQrcode size={16} />
              )}
              {t("channels.whatsappAgent.pair")}
            </Button>
            <Button
              variant="outline"
              onClick={() => void handleDisconnect()}
              disabled={busy || (!paired && !status?.enabled)}
              className="gap-2"
            >
              <IconPlugConnectedX size={16} />
              {t("channels.whatsappAgent.disconnect")}
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}
