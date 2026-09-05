import {
  IconLoader2,
  IconLogout,
  IconMenu2,
  IconMoon,
  IconPlayerPlay,
  IconPower,
  IconRefresh,
  IconSun,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import * as React from "react"
import { useTranslation } from "react-i18next"

import { postLauncherDashboardLogout } from "@/api/launcher-auth"
import { PocketClawMark } from "@/components/brand/pocketclaw-mark"
import { HeaderUtilityMenu } from "@/components/header-utility-menu"
import { LanguageMenu } from "@/components/language-menu"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog.tsx"
import { Button } from "@/components/ui/button.tsx"
import { Separator } from "@/components/ui/separator.tsx"
import { SidebarTrigger, useSidebar } from "@/components/ui/sidebar"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"
import { useGateway } from "@/hooks/use-gateway.ts"
import { useTheme } from "@/hooks/use-theme.ts"

export function AppHeader() {
  const { t } = useTranslation()
  const { theme, toggleTheme } = useTheme()
  const { openMobile } = useSidebar()
  const {
    state: gwState,
    loading: gwLoading,
    canStart,
    startReason,
    restartRequired,
    start,
    restart,
    stop,
    error: gwError,
  } = useGateway()

  const isRunning = gwState === "running"
  const isStarting = gwState === "starting"
  const isRestarting = gwState === "restarting"
  const isStopping = gwState === "stopping"
  const showNotConnectedHint =
    !isRestarting &&
    !isStopping &&
    canStart &&
    (gwState === "stopped" || gwState === "error")

  const [showStopDialog, setShowStopDialog] = React.useState(false)
  const [showLogoutDialog, setShowLogoutDialog] = React.useState(false)

  const handleLogout = async () => {
    await postLauncherDashboardLogout()
    globalThis.location.assign("/launcher-login")
  }

  const handleGatewayToggle = () => {
    if (gwLoading || isRestarting || isStopping || (!isRunning && !canStart)) {
      return
    }
    if (isRunning) {
      setShowStopDialog(true)
    } else {
      void start()
    }
  }

  const handleGatewayRestart = () => {
    if (gwLoading || isRestarting || !restartRequired || !canStart) return
    void restart()
  }

  const confirmStop = () => {
    setShowStopDialog(false)
    stop()
  }

  return (
    <header className="bg-pc-surface-1 border-b-pc-line sticky top-0 z-50 flex h-14 shrink-0 items-center justify-between border-b px-3 md:px-4">
      <div className="flex min-w-0 items-center gap-2">
        {/* A navigation drawer, so it says menu and looks like one. 44x44 is
            the hit area; the glyph stays 20px. */}
        <SidebarTrigger
          className="text-muted-foreground hover:bg-accent hover:text-foreground flex size-11 shrink-0 items-center justify-center rounded-md sm:hidden [&>svg]:size-5"
          label={openMobile ? t("common.closeMenu") : t("common.openMenu")}
        >
          <IconMenu2 />
        </SidebarTrigger>
        {/* Mark only, phone only. On desktop the sidebar header carries the
            full lockup and repeating it here said the same thing twice. */}
        <Link
          to="/"
          className="text-foreground flex shrink-0 items-center rounded-md sm:hidden"
          aria-label={t("header.logoAlt")}
        >
          <PocketClawMark className="text-pc-claw size-6" />
        </Link>

        {/* Status belongs in the leading zone the brand vacated, in normal
            flow rather than absolutely centred — an absolutely positioned
            pill has no way to yield when the two ends grow. */}
        {showNotConnectedHint && (
          <div className="text-pc-muted border-pc-danger/40 ms-1 hidden min-w-0 items-center gap-2 rounded-full border border-dashed px-3 py-1.5 text-xs md:flex">
            <span
              aria-hidden="true"
              className="bg-pc-danger size-2 shrink-0 rounded-full"
            />
            <span className="truncate">{t("chat.notConnected")}</span>
          </div>
        )}
      </div>

      <AlertDialog open={showStopDialog} onOpenChange={setShowStopDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>
              {t("header.gateway.stopDialog.title")}
            </AlertDialogTitle>
            <AlertDialogDescription>
              {t("header.gateway.stopDialog.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction
              onClick={confirmStop}
              className="bg-pc-danger text-white hover:bg-pc-danger/90"
            >
              {t("header.gateway.stopDialog.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <AlertDialog open={showLogoutDialog} onOpenChange={setShowLogoutDialog}>
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t("header.logout.tooltip")}</AlertDialogTitle>
            <AlertDialogDescription>
              {t("header.logout.description")}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{t("common.cancel")}</AlertDialogCancel>
            <AlertDialogAction onClick={() => void handleLogout()}>
              {t("header.logout.confirm")}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>

      <div className="text-muted-foreground flex shrink-0 items-center gap-1 text-sm font-medium">
        {restartRequired && (
          <Tooltip delayDuration={700}>
            <TooltipTrigger asChild>
              <Button
                variant="secondary"
                size="icon-sm"
                className="bg-pc-warning-soft text-pc-warning hover:bg-pc-warning-soft hover:text-pc-warning hidden size-9 rounded-full sm:inline-flex"
                onClick={handleGatewayRestart}
                disabled={gwLoading || isRestarting || isStopping || !canStart}
                aria-label={t("header.gateway.action.restart")}
              >
                <IconRefresh className="size-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {t("header.gateway.restartRequired")}
            </TooltipContent>
          </Tooltip>
        )}

        {/* Gateway Start/Stop */}
        {isRunning ? (
          <Tooltip delayDuration={700}>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="sm"
                className="border-pc-line bg-pc-surface-1 hover:border-pc-danger/50 hover:text-pc-danger h-9 gap-2 rounded-full border px-3"
                data-tour="gateway-button"
                onClick={handleGatewayToggle}
                disabled={gwLoading}
                aria-label={t("header.gateway.action.stop")}
              >
                <span
                  aria-hidden="true"
                  className="bg-pc-signal pc-signal-pulse size-2 shrink-0 rounded-full"
                />
                <span className="hidden text-xs font-semibold sm:inline">
                  {t("header.gateway.status.running")}
                </span>
                <IconPower className="size-4 opacity-70" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>
              {gwError ?? t("header.gateway.action.stop")}
            </TooltipContent>
          </Tooltip>
        ) : (
          <Tooltip
            delayDuration={gwError || (!canStart && startReason) ? 0 : 700}
          >
            <TooltipTrigger asChild>
              {/* Wrap in span so the tooltip still fires when the button is disabled */}
              <span
                className={
                  !canStart && startReason ? "cursor-not-allowed" : undefined
                }
                tabIndex={!canStart && startReason ? 0 : undefined}
              >
                <Button
                  variant={
                    isStarting || isRestarting || isStopping
                      ? "secondary"
                      : "default"
                  }
                  size="sm"
                  data-tour="gateway-button"
                  className={`h-9 gap-2 rounded-full px-3 sm:px-3.5 ${
                    !canStart ? "pointer-events-none" : ""
                  }`}
                  onClick={handleGatewayToggle}
                  disabled={
                    gwLoading ||
                    isStarting ||
                    isRestarting ||
                    isStopping ||
                    !canStart
                  }
                >
                  {gwLoading || isStarting || isRestarting || isStopping ? (
                    <IconLoader2 className="h-4 w-4 animate-spin opacity-70" />
                  ) : (
                    <IconPlayerPlay className="h-4 w-4 opacity-80" />
                  )}
                  <span className="hidden text-xs font-semibold sm:inline">
                    {isStopping
                      ? t("header.gateway.status.stopping")
                      : isRestarting
                        ? t("header.gateway.status.restarting")
                        : isStarting
                          ? t("header.gateway.status.starting")
                          : t("header.gateway.action.start")}
                  </span>
                </Button>
              </span>
            </TooltipTrigger>
            {gwError || (!canStart && startReason) ? (
              <TooltipContent>{gwError ?? startReason}</TooltipContent>
            ) : null}
          </Tooltip>
        )}

        <Separator
          className="mx-2 my-2 hidden sm:block"
          orientation="vertical"
        />

        {/* Desktop: the utilities are cheap to show, so they stay one tap
            away. Phone: they move into the overflow below. */}
        <div className="hidden items-center gap-1 sm:flex">
          <LanguageMenu className="size-10" />

          <Button
            variant="ghost"
            size="icon"
            className="size-10"
            onClick={toggleTheme}
            aria-label={
              theme === "dark" ? t("common.lightMode") : t("common.darkMode")
            }
          >
            {theme === "dark" ? (
              <IconSun className="size-4.5" />
            ) : (
              <IconMoon className="size-4.5" />
            )}
          </Button>

          <Tooltip delayDuration={700}>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="size-10"
                onClick={() => setShowLogoutDialog(true)}
                aria-label={t("header.logout.tooltip")}
              >
                <IconLogout className="size-4.5" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>{t("header.logout.tooltip")}</TooltipContent>
          </Tooltip>
        </div>

        <HeaderUtilityMenu
          className="size-11 sm:hidden"
          theme={theme}
          onToggleTheme={toggleTheme}
          onLogout={() => setShowLogoutDialog(true)}
          onRestartGateway={handleGatewayRestart}
          restartRequired={restartRequired}
          restartDisabled={
            gwLoading || isRestarting || isStopping || !canStart
          }
        />
      </div>
    </header>
  )
}
