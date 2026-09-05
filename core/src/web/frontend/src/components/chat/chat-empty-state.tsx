import {
  IconPlugConnectedX,
  IconRobotOff,
  IconStar,
} from "@tabler/icons-react"
import { Link } from "@tanstack/react-router"
import type { ReactNode } from "react"
import { useTranslation } from "react-i18next"

import { PocketClawMark } from "@/components/brand/pocketclaw-mark"
import { Button } from "@/components/ui/button"

interface ChatEmptyStateProps {
  hasAvailableModels: boolean
  defaultModelName: string
  isConnected: boolean
}

/**
 * The three blocked variants and the ready one.
 *
 * The whole block used to sit under `opacity-70`, which dragged the heading
 * and the body text below comfortable contrast. Hierarchy comes from the
 * colour tokens instead: the same visual softness, without the accessibility
 * cost.
 */
export function ChatEmptyState({
  hasAvailableModels,
  defaultModelName,
  isConnected,
}: ChatEmptyStateProps) {
  const { t } = useTranslation()

  if (!hasAvailableModels) {
    return (
      <Blocked
        icon={<IconRobotOff className="h-7 w-7" />}
        title={t("chat.empty.noConfiguredModel")}
        description={t("chat.empty.noConfiguredModelDescription")}
        action={
          <Button asChild variant="outline" size="sm" className="px-4">
            <Link to="/models">{t("chat.empty.goToModels")}</Link>
          </Button>
        }
      />
    )
  }

  if (!defaultModelName) {
    return (
      <Blocked
        icon={<IconStar className="h-7 w-7" />}
        title={t("chat.empty.noSelectedModel")}
        description={t("chat.empty.noSelectedModelDescription")}
      />
    )
  }

  if (!isConnected) {
    return (
      <Blocked
        icon={<IconPlugConnectedX className="h-7 w-7" />}
        title={t("chat.empty.notRunning")}
        description={t("chat.empty.notRunningDescription")}
      />
    )
  }

  return (
    <div className="flex flex-col items-center justify-center px-4 py-20 text-center">
      <PocketClawMark className="text-pc-faint size-12" strokeWidth={1.6} />
      <h3 className="text-pc-text mt-4 text-[1.375rem] font-semibold tracking-[-0.01em]">
        {t("chat.welcome")}
      </h3>
      <p className="text-pc-muted mt-2 max-w-prose text-sm">
        {t("chat.welcomeDesc")}
      </p>
      {/* The active model is a machine identifier, so it is set in mono and
          kept LTR: a reversed model id is a bug, not localization. */}
      <p className="text-pc-faint pc-mono mt-3 text-xs" dir="ltr">
        {defaultModelName}
      </p>
    </div>
  )
}

function Blocked({
  icon,
  title,
  description,
  action,
}: {
  icon: ReactNode
  title: string
  description: string
  action?: ReactNode
}) {
  return (
    <div className="flex flex-col items-center justify-center px-4 py-20 text-center">
      <div className="bg-pc-warning-soft text-pc-warning border-pc-warning/30 flex h-14 w-14 items-center justify-center rounded-xl border">
        {icon}
      </div>
      <h3 className="text-pc-text mt-5 text-[1.125rem] font-semibold">
        {title}
      </h3>
      <p className="text-pc-muted mt-2 max-w-prose text-sm">{description}</p>
      {action ? <div className="mt-4">{action}</div> : null}
    </div>
  )
}
