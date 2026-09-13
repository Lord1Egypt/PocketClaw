import {
  IconBrandTelegram,
  IconCircleCheckFilled,
  IconChevronDown,
  IconRefresh,
} from "@tabler/icons-react"
import { useCallback, useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ChannelConfig } from "@/api/channels"
import { type ArrayFieldFlusher } from "@/components/channels/channel-array-list-field"
import { TelegramForm } from "@/components/channels/channel-forms/telegram-form"
import {
  type TelegramSurface,
  isAdvancedFormAlwaysVisible,
  resolveTelegramManualReason,
  resolveTelegramSurface,
} from "@/components/channels/channel-forms/telegram-surface"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  HOST_READY_EVENT,
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
  const surface: TelegramSurface = useMemo(
    () => resolveTelegramSurface({ configured, onboardingAvailable }),
    [configured, onboardingAvailable],
  )
  const advancedAlwaysVisible = isAdvancedFormAlwaysVisible(surface)

  const botUsername = host?.telegramBotUsername ?? null
  const ownerConfigured = asStringArray(config.allow_from).length > 0

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
              <IconCircleCheckFilled className="size-5 text-pc-success" />
              <p className="text-base font-semibold">
                {t("channels.telegram.connected")}
              </p>
            </div>

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
              {botUsername && host && (
                <Button onClick={openChat}>
                  <IconBrandTelegram />
                  {t("channels.telegram.openChat")}
                </Button>
              )}
              {onboardingAvailable && (
                <Button variant="outline" onClick={connect}>
                  <IconRefresh />
                  {t("channels.telegram.reconnect")}
                </Button>
              )}
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
