import {
  IconAlertTriangle,
  IconBrandWhatsapp,
  IconCircleCheckFilled,
  IconLoader2,
  IconPencil,
  IconPlugConnected,
  IconPlugConnectedX,
  IconSend,
} from "@tabler/icons-react"
import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import {
  WHATSAPP_SELF_CHAT_TEST_MESSAGE,
  type WhatsAppNumberError,
  normalizeWhatsAppNumber,
} from "@/components/channels/channel-forms/whatsapp-self-chat"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  HOST_READY_EVENT,
  type PocketClawHost,
  getPocketClawHost,
  isWhatsAppSelfChatAvailable,
} from "@/lib/pocketclaw-host"
import type { ApplyOutcome } from "@/lib/restart-required"

interface WhatsAppSelfChatPanelProps {
  config: ChannelConfig
  /**
   * Persists the whole channel block and applies it through the gateway's safe
   * config-apply path. The panel owns its own draft, so the page-level
   * Save/Reset footer never competes with Connect and Disconnect.
   *
   * Resolves with what actually happened, so the card can say it rather than
   * leaving the user to guess — or, worse, claim a state that is not true.
   */
  onPersist: (nextConfig: ChannelConfig) => Promise<ApplyOutcome>
}

function asString(value: unknown): string {
  return typeof value === "string" ? value : ""
}

const NUMBER_ERROR_KEY: Record<WhatsAppNumberError, string> = {
  empty: "channels.whatsappSelfChat.errorEmpty",
  invalidChars: "channels.whatsappSelfChat.errorInvalidChars",
  notInternational: "channels.whatsappSelfChat.errorNotInternational",
  length: "channels.whatsappSelfChat.errorLength",
}

/**
 * Channels → WhatsApp Self-Chat.
 *
 * The only thing this surface stores is the user's own number. It replaced two
 * cards — "WhatsApp" and "WhatsApp Native" — that asked a phone user for a
 * bridge URL and a session store path, neither of which they could supply.
 *
 * Nothing here reads WhatsApp, keeps a session, or presses Send. The host opens
 * a deep link and stops.
 */
export function WhatsAppSelfChatPanel({
  config,
  onPersist,
}: WhatsAppSelfChatPanelProps) {
  const { t } = useTranslation()
  const storedNumber = asString(config.self_number)

  const [host, setHost] = useState<PocketClawHost | null>(() =>
    getPocketClawHost(),
  )
  const [draft, setDraft] = useState(storedNumber)
  const [editing, setEditing] = useState(false)
  const [error, setError] = useState("")
  const [busy, setBusy] = useState(false)
  const [outcome, setOutcome] = useState<ApplyOutcome | null>(null)

  // The host injects itself after the page loads, which can land after React
  // has already rendered. Without this the console would decide "no host" once
  // and hide Test for the rest of the session.
  useEffect(() => {
    const sync = () => setHost(getPocketClawHost())
    sync()
    window.addEventListener(HOST_READY_EVENT, sync)
    return () => window.removeEventListener(HOST_READY_EVENT, sync)
  }, [])

  // A reload — after Connect, or after the page refetched — is the authority on
  // what is stored, so the draft follows it out of edit mode.
  useEffect(() => {
    setDraft(storedNumber)
    setEditing(false)
    setError("")
  }, [storedNumber])

  const configured = useMemo(
    () => normalizeWhatsAppNumber(storedNumber).ok === true,
    [storedNumber],
  )
  const canTest = isWhatsAppSelfChatAvailable(host)

  const persist = useCallback(
    async (selfNumber: string) => {
      setBusy(true)
      setOutcome(null)
      try {
        setOutcome(await onPersist({ ...config, self_number: selfNumber }))
      } finally {
        setBusy(false)
      }
    },
    [config, onPersist],
  )

  /**
   * What the card says about applying the change.
   *
   * The number itself is read from the configuration on every use, so a saved
   * number is in effect whether or not the gateway restarted. What a deferred
   * or failed apply means is that the *rest* of the gateway has not picked the
   * save up yet — which is worth saying, and is never a reason to ask the user
   * to restart something by hand.
   */
  const applyStatus = busy
    ? { key: "channels.whatsappSelfChat.applying", tone: "pending" as const }
    : outcome === "not_applied"
      ? {
          key: "channels.whatsappSelfChat.applyDeferred",
          tone: "pending" as const,
        }
      : outcome === "failed"
        ? {
            key: "channels.whatsappSelfChat.applyFailed",
            tone: "warning" as const,
          }
        : null

  const applyStatusLine = applyStatus && (
    <p
      className={`flex items-center gap-2 text-sm ${
        applyStatus.tone === "warning"
          ? "text-amber-600"
          : "text-muted-foreground"
      }`}
      data-testid={`whatsapp-self-chat-apply-${applyStatus.tone}`}
    >
      {applyStatus.tone === "pending" ? (
        <IconLoader2 className="size-4 animate-spin" />
      ) : (
        <IconAlertTriangle className="size-4" />
      )}
      {t(applyStatus.key)}
    </p>
  )

  const handleConnect = useCallback(async () => {
    const result = normalizeWhatsAppNumber(draft)
    if (!result.ok) {
      setError(t(NUMBER_ERROR_KEY[result.error]))
      return
    }
    setError("")
    await persist(result.number)
  }, [draft, persist, t])

  const handleDisconnect = useCallback(async () => {
    setError("")
    await persist("")
  }, [persist])

  const handleTest = useCallback(() => {
    host?.openWhatsAppSelfChat?.(storedNumber, WHATSAPP_SELF_CHAT_TEST_MESSAGE)
  }, [host, storedNumber])

  const numberField = (
    <div className="space-y-2">
      <Label htmlFor="whatsapp-self-number">
        {t("channels.whatsappSelfChat.numberLabel")}
      </Label>
      <Input
        id="whatsapp-self-number"
        type="tel"
        inputMode="tel"
        autoComplete="tel"
        dir="ltr"
        placeholder="+20 101 234 5678"
        value={draft}
        onChange={(event) => {
          setDraft(event.target.value)
          setError("")
        }}
      />
      <p className="text-muted-foreground text-sm">
        {t("channels.whatsappSelfChat.numberHint")}
      </p>
      {error && <p className="text-destructive text-sm">{error}</p>}
    </div>
  )

  if (!configured || editing) {
    return (
      <div className="space-y-6" data-testid="whatsapp-self-chat-setup">
        <Card className="shadow-sm">
          <CardContent className="space-y-4 px-6 py-5">
            <div>
              <p className="text-base font-semibold">
                {t("channels.whatsappSelfChat.connectTitle")}
              </p>
              <p className="text-muted-foreground mt-1 text-sm">
                {t("channels.whatsappSelfChat.connectBody")}
              </p>
            </div>

            {numberField}

            {applyStatusLine}

            <div className="flex flex-wrap gap-2 pt-1">
              <Button onClick={() => void handleConnect()} disabled={busy}>
                <IconPlugConnected />
                {t("channels.whatsappSelfChat.connect")}
              </Button>
              {editing && (
                <Button
                  variant="outline"
                  disabled={busy}
                  onClick={() => {
                    setDraft(storedNumber)
                    setEditing(false)
                    setError("")
                    setOutcome(null)
                  }}
                >
                  {t("common.cancel")}
                </Button>
              )}
            </div>
          </CardContent>
        </Card>
      </div>
    )
  }

  return (
    <div className="space-y-6" data-testid="whatsapp-self-chat-connected">
      <Card className="shadow-sm">
        <CardContent className="space-y-4 px-6 py-5">
          <div className="flex items-center gap-2">
            <IconCircleCheckFilled className="size-5 text-emerald-500" />
            <p className="text-base font-semibold">
              {t("channels.whatsappSelfChat.connected")}
            </p>
          </div>

          <div>
            <p className="text-muted-foreground text-xs tracking-wide uppercase">
              {t("channels.whatsappSelfChat.numberLabel")}
            </p>
            <p className="font-mono text-sm" dir="ltr">
              {storedNumber}
            </p>
          </div>

          <p className="text-muted-foreground text-sm">
            {t("channels.whatsappSelfChat.neverSends")}
          </p>

          {applyStatusLine}

          <div className="flex flex-wrap gap-2 pt-1">
            {canTest && (
              <Button onClick={handleTest} disabled={busy}>
                <IconSend />
                {t("channels.whatsappSelfChat.test")}
              </Button>
            )}
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => {
                setDraft(storedNumber)
                setEditing(true)
                setOutcome(null)
              }}
            >
              <IconPencil />
              {t("channels.whatsappSelfChat.change")}
            </Button>
            <Button
              variant="outline"
              disabled={busy}
              onClick={() => void handleDisconnect()}
            >
              <IconPlugConnectedX />
              {t("channels.whatsappSelfChat.disconnect")}
            </Button>
          </div>

          {!canTest && (
            <p className="text-muted-foreground flex items-center gap-2 text-sm">
              <IconBrandWhatsapp className="size-4" />
              {t("channels.whatsappSelfChat.testNeedsApp")}
            </p>
          )}
        </CardContent>
      </Card>
    </div>
  )
}
