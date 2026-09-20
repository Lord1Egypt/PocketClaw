import {
  IconAlertTriangle,
  IconBrandTelegram,
  IconCircleCheckFilled,
  IconChevronDown,
  IconLoader2,
  IconRefresh,
} from "@tabler/icons-react"
import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { type ArrayFieldFlusher } from "@/components/channels/channel-array-list-field"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import { getTelegramOnboardingAvailability } from "@/api/telegram-onboarding"
import { TelegramDesktopConnect } from "@/components/channels/channel-forms/telegram-desktop-connect"
import { TelegramDisconnectDialog } from "@/components/channels/channel-forms/telegram-disconnect-dialog"
import {
  readinessLabelKey,
  useTelegramReadiness,
} from "@/components/channels/channel-forms/use-telegram-readiness"
import {
  type TelegramSurface,
  isAdvancedFormAlwaysVisible,
  isTelegramStartingState,
  resolveTelegramManualReason,
  resolveTelegramSurface,
} from "@/components/channels/channel-forms/telegram-surface"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  HOST_READY_EVENT,
  TELEGRAM_UPDATED_EVENT,
  type PocketClawHost,
  getPocketClawHost,
  isTelegramOnboardingAvailable,
} from "@/lib/pocketclaw-host"

interface TelegramPanelProps {
  config: ChannelConfig
  onChange: (key: string, value: unknown) => void
  configuredSecrets: string[]
  configured: boolean
  fieldErrors?: Record<string, string>
  registerArrayFieldFlusher?: (
    fieldPath: string,
    flusher: ArrayFieldFlusher | null,
  ) => void
  arrayFieldResetVersion?: number
}

function asStringArray(value: unknown): string[] {
  return Array.isArray(value)
    ? value.filter((item): item is string => typeof item === "string")
    : []
}

/**
 * Channels → Telegram.
 *
 * This is the page a user actually lands on, and until now it opened straight
 * onto Bot Token / API Base URL. The managed-bot flow existed but was wired
 * only to the native settings list, so nobody reached it. The raw form is
 * unchanged and still here — it has just stopped being the first thing a new
 * user sees.
 */
export function TelegramPanel({
  config,
  onChange,
  configuredSecrets,
  configured,
  fieldErrors,
  registerArrayFieldFlusher,
  arrayFieldResetVersion,
}: TelegramPanelProps) {
  const { t } = useTranslation()
  const [host, setHost] = useState<PocketClawHost | null>(() =>
    getPocketClawHost(),
  )
  const [advancedOpen, setAdvancedOpen] = useState(false)

  // The host injects itself after the page loads, which can land after React
  // has already rendered. Without this the console would decide "no host" once
  // and keep showing manual setup for the rest of the session.
  useEffect(() => {
    const sync = () => setHost(getPocketClawHost())
    sync()
    window.addEventListener(HOST_READY_EVENT, sync)
    return () => window.removeEventListener(HOST_READY_EVENT, sync)
  }, [])

  const onboardingAvailable = isTelegramOnboardingAvailable(host)

  // PC-DEF-060. With no Android host, Core can still run the pairing — ask it.
  // Independent of the host check: this is about what the backend can do, not what
  // this client can.
  const [coreOnboardingAvailable, setCoreOnboardingAvailable] = useState(false)

  // Reuses the event the page already reloads on, rather than threading a second
  // refresh path down from the parent.
  const onManagedConnected = useCallback(() => {
    window.dispatchEvent(new Event(TELEGRAM_UPDATED_EVENT))
  }, [])

  // Asked unconditionally: the connected card needs the answer too, to offer
  // Replace bot where Core can pair one.
  useEffect(() => {
    let cancelled = false
    void getTelegramOnboardingAvailability().then((available) => {
      if (!cancelled) setCoreOnboardingAvailable(available)
    })
    return () => {
      cancelled = true
    }
  }, [])
  const surface: TelegramSurface = useMemo(
    () => resolveTelegramSurface({ configured, onboardingAvailable }),
    [configured, onboardingAvailable],
  )
  const advancedAlwaysVisible = isAdvancedFormAlwaysVisible(surface)

  const botUsername = host?.telegramBotUsername ?? null
  const ownerConfigured = asStringArray(config.allow_from).length > 0

  // PC-DEF-061. A page opened while the gateway is still starting must not
  // simply say Connected: the configuration being present is not the channel
  // being able to receive. Watched only while configured, and it stops once
  // ready, so a settled page makes no requests.
  const { readiness, recheck } = useTelegramReadiness(surface === "connected")
  const receiving = readiness === null || readiness.ready
  // "unknown" means the gateway would not say, which is neither connected nor
  // starting. Claiming either would be the dishonest half of this fix.
  const statusUnreadable = readiness?.state === "unknown"
  // A valid token with no owner is an explicit incomplete state, never
  // "Connected": the bot answers private senders with setup guidance and grants
  // no agent access until an owner is configured.
  const setupIncomplete = readiness?.state === "setup_required"
  // A bot owned by another service -- an active webhook or another long poller
  // -- is terminal for this configuration. It is never "Connected", and the
  // actionable explanation depends on which ownership the other service holds.
  const conflict = readiness?.state === "telegram_conflict"
  const conflictBodyKey =
    readiness?.detail === "webhook_active"
      ? "channels.telegram.conflictWebhook"
      : "channels.telegram.conflictInUse"
  // PC-DEF-071. Anything that is neither ready nor a stage of starting must not
  // be dressed as one. Its readiness sentence is already a complete, accurate
  // statement, so it becomes the heading rather than sitting underneath a
  // spinner that contradicts it.
  const stalled =
    readiness !== null &&
    !receiving &&
    !conflict &&
    !setupIncomplete &&
    !statusUnreadable &&
    !isTelegramStartingState(readiness.state)

  // Route the user straight to the owner field: opening the advanced form is
  // the direct path to Allowed From, and it is the same form on every client.
  useEffect(() => {
    if (setupIncomplete) setAdvancedOpen(true)
  }, [setupIncomplete])

  // PC-DEF-062. Replacing a bot is pairing a new one over the old; the managed
  // flow already does exactly that, so it is revealed rather than reimplemented.
  const [replaceOpen, setReplaceOpen] = useState(false)

  const connect = useCallback(() => {
    host?.openTelegramOnboarding()
  }, [host])

  const openChat = useCallback(() => {
    if (!host || !botUsername) return
    host.openExternal(`https://t.me/${botUsername}`)
  }, [botUsername, host])

  const advancedForm = (
    <TelegramForm
      config={config}
      onChange={onChange}
      configuredSecrets={configuredSecrets}
      fieldErrors={fieldErrors}
      registerArrayFieldFlusher={registerArrayFieldFlusher}
      arrayFieldResetVersion={arrayFieldResetVersion}
    />
  )

  if (advancedAlwaysVisible) {
    // PC-DEF-060. The reason decides the wording. A desktop browser is not a build
    // without the feature, and saying so sent the user looking for a different
    // build instead of telling them where one-tap setup actually lives.
    const manualReason = resolveTelegramManualReason({
      hostPresent: host !== null,
      onboardingConfigured: host?.onboardingConfigured === true,
    })
    return (
      <div
        className="space-y-6"
        data-testid="telegram-surface-manual-only"
        data-manual-reason={manualReason}
      >
        <Card className="shadow-sm">
          <CardContent className="px-6 py-5">
            <p className="text-sm font-medium">
              {t("channels.telegram.manualOnlyTitle")}
            </p>
            <p className="text-muted-foreground mt-1 text-sm">
              {manualReason === "no-host"
                ? t("channels.telegram.manualOnlyDesktopBody")
                : t("channels.telegram.manualOnlyBody")}
            </p>
            {/* An external link, because BotFather is where a token comes from and
                the manual path is otherwise a form with no starting point. */}
            <a
              className="text-primary mt-3 inline-block text-sm underline hover:no-underline"
              href="https://t.me/BotFather"
              target="_blank"
              rel="noreferrer noopener"
            >
              {t("channels.telegram.openBotFather")}
            </a>
          </CardContent>
        </Card>
        {/* Core can run the managed pairing even where this client cannot, so
            Connect is offered first and the manual form stays below it rather than
            being replaced. */}
        {coreOnboardingAvailable && (
          <TelegramDesktopConnect onConnected={onManagedConnected} />
        )}
        {advancedForm}
      </div>
    )
  }

  return (
    <div
      className="space-y-6"
      data-testid={
        surface === "connected"
          ? "telegram-surface-connected"
          : "telegram-surface-managed-onboarding"
      }
    >
      {surface === "connected" ? (
        <Card className="shadow-sm">
          <CardContent className="space-y-4 px-6 py-5">
            <div className="flex items-center gap-2">
              {setupIncomplete || conflict || stalled ? (
                <IconAlertTriangle className="size-5 text-pc-warning" />
              ) : receiving ? (
                <IconCircleCheckFilled className="size-5 text-pc-success" />
              ) : statusUnreadable ? (
                <IconAlertTriangle className="text-muted-foreground size-5" />
              ) : (
                <IconLoader2 className="text-muted-foreground size-5 animate-spin" />
              )}
              <p
                className="text-base font-semibold"
                data-telegram-heading={stalled ? "stalled" : "stage"}
              >
                {conflict
                  ? t("channels.telegram.conflictTitle")
                  : setupIncomplete
                    ? t("channels.telegram.setupIncompleteTitle")
                    : receiving
                      ? t("channels.telegram.connected")
                      : statusUnreadable
                        ? t("channels.telegram.statusUnreadableTitle")
                        : stalled && readiness
                          ? t(readinessLabelKey(readiness.state))
                          : t("channels.telegram.startingTitle")}
              </p>
            </div>

            {/* Named so the user knows why not to send a message yet, rather
                than being told Connected while the channel is still starting.
                A stalled state has already said it in the heading; repeating it
                is what made the card read as two different answers at once. */}
            {!receiving && readiness && !conflict && !stalled && (
              <p
                className="text-muted-foreground text-sm"
                data-readiness-state={readiness.state}
              >
                {t(readinessLabelKey(readiness.state))}
              </p>
            )}
            {/* A bot owned elsewhere needs its own actionable sentence: an
                active webhook and another long poller need different fixes. */}
            {conflict && (
              <p
                className="text-muted-foreground text-sm"
                data-readiness-state="telegram_conflict"
              >
                {t(conflictBodyKey)}
              </p>
            )}

            {botUsername && (
              <div>
                <p className="text-muted-foreground text-xs tracking-wide uppercase">
                  {t("channels.telegram.botLabel")}
                </p>
                <p className="font-mono text-sm">@{botUsername}</p>
              </div>
            )}

            <div>
              <p className="text-muted-foreground text-xs tracking-wide uppercase">
                {t("channels.telegram.ownerLabel")}
              </p>
              <p className="text-sm">
                {ownerConfigured
                  ? t("channels.telegram.ownerConfigured")
                  : t("channels.telegram.ownerAnyone")}
              </p>
            </div>

            <div className="flex flex-wrap gap-2 pt-1">
              {/* A conflict is terminal; the user resolves it elsewhere and
                  then asks again. Re-polling is the honest action. */}
              {conflict && (
                <Button
                  variant="outline"
                  className="min-h-10"
                  onClick={recheck}
                >
                  <IconRefresh />
                  {t("channels.telegram.desktop.retry")}
                </Button>
              )}
              {botUsername && host && (
                <Button onClick={openChat} className="min-h-10">
                  <IconBrandTelegram />
                  {t("channels.telegram.openChat")}
                </Button>
              )}
              {onboardingAvailable && (
                <Button
                  variant="outline"
                  className="min-h-10"
                  onClick={connect}
                >
                  <IconRefresh />
                  {t("channels.telegram.reconnect")}
                </Button>
              )}
              {/* PC-DEF-062. Where Core can pair — which includes a plain
                  browser — replacing the bot is offered here rather than only
                  in the app. */}
              {!onboardingAvailable && coreOnboardingAvailable && (
                <Button
                  variant="outline"
                  className="min-h-10"
                  aria-expanded={replaceOpen}
                  onClick={() => setReplaceOpen((open) => !open)}
                >
                  <IconRefresh />
                  {t("channels.telegram.replaceBot")}
                </Button>
              )}
              <TelegramDisconnectDialog onDisconnected={onManagedConnected} />
            </div>
          </CardContent>
        </Card>
      ) : (
        <Card className="shadow-sm">
          <CardContent className="space-y-4 px-6 py-5">
            <div>
              <p className="text-base font-semibold">
                {t("channels.telegram.connectTitle")}
              </p>
              <p className="text-muted-foreground mt-1 text-sm">
                {t("channels.telegram.connectBody")}
              </p>
            </div>

            <Button onClick={connect}>
              <IconBrandTelegram />
              {t("channels.telegram.openTelegram")}
            </Button>

            <p className="text-muted-foreground text-sm">
              {t("channels.telegram.statusLabel")}{" "}
              <span className="text-foreground font-medium">
                {t("channels.telegram.statusReady")}
              </span>
            </p>
          </CardContent>
        </Card>
      )}

      {/* Pairing a new bot over the old one. The same managed flow, so the owner
          contract and the authoritative writer are the ones already verified. */}
      {surface === "connected" && replaceOpen && (
        <TelegramDesktopConnect
          onConnected={() => {
            setReplaceOpen(false)
            onManagedConnected()
          }}
        />
      )}

      <div className="space-y-4">
        <div className="flex items-center gap-3">
          {surface === "managed-onboarding" && (
            <p className="text-muted-foreground text-sm">
              {t("channels.telegram.troubleLabel")}
            </p>
          )}
          <Button
            variant="outline"
            aria-expanded={advancedOpen}
            onClick={() => setAdvancedOpen((open) => !open)}
          >
            <IconChevronDown
              className={advancedOpen ? "rotate-180 transition" : "transition"}
            />
            {surface === "connected"
              ? t("channels.telegram.advancedSettings")
              : t("channels.telegram.advancedManual")}
          </Button>
        </div>

        {advancedOpen && advancedForm}
      </div>
    </div>
  )
}
