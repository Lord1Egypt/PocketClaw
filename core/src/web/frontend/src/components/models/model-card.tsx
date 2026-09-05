import {
  IconEdit,
  IconKey,
  IconLoader2,
  IconStar,
  IconStarFilled,
  IconTrash,
} from "@tabler/icons-react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from "@/components/ui/tooltip"

interface ModelCardProps {
  model: ModelInfo
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  settingDefault: boolean
}

export function ModelCard({
  model,
  onEdit,
  onSetDefault,
  onDelete,
  settingDefault,
}: ModelCardProps) {
  const { t } = useTranslation()
  const isOAuth = model.auth_method === "oauth"
  const status = model.status
  const statusLabel = t(`models.status.${status}`)
  const canSetDefault =
    model.available &&
    !model.is_default &&
    !model.is_virtual &&
    model.default_model_allowed !== false

  const setDefaultLabel = t("models.action.setDefault")
  const setDefaultDisabledReason = (() => {
    if (settingDefault) return t("models.action.setDefaultDisabled.setting")
    if (!model.available)
      return t("models.action.setDefaultDisabled.unavailable")
    if (model.is_default) return t("models.action.setDefaultDisabled.isDefault")
    if (model.is_virtual) return t("models.action.setDefaultDisabled.isVirtual")
    if (model.default_model_allowed === false) {
      return t("models.action.setDefaultDisabled.unsupportedProvider")
    }
    return setDefaultLabel
  })()

  const editLabel = t("models.action.edit")
  const deleteLabel = t("models.action.delete")
  const deleteDisabledReason = model.is_default
    ? t("models.action.deleteDisabled.isDefault")
    : deleteLabel
  const deleteDisabled = model.is_default

  return (
    <div
      className={[
        "group/card border-pc-line bg-pc-surface-1 hover:bg-pc-surface-2 relative flex w-full flex-col gap-3 justify-self-stretch rounded-xl border border-s-2 p-4 transition-colors",
        model.is_default ? "border-s-pc-claw" : "border-s-pc-line",
        model.available ? "" : "opacity-75",
      ].join(" ")}
    >
      <div className="flex items-start justify-between gap-2">
        <div className="flex min-w-0 items-center gap-2">
          <span
            className={[
              "mt-0.5 h-2 w-2 shrink-0 rounded-full",
              status === "available"
                ? "bg-pc-success"
                : status === "unreachable"
                  ? "bg-pc-warning"
                  : "bg-pc-faint",
            ].join(" ")}
            title={statusLabel}
          />
          <span className="text-pc-text truncate text-sm font-semibold">
            {model.model_name}
          </span>
          {/* Role badges. More than one role can apply to a model, so this
              is a wrapping row rather than a single slot — a later routing
              role lands here as a sibling, not as a redesign. */}
          <div className="flex shrink-0 flex-wrap items-center gap-1">
            {model.is_default && (
              <span className="bg-pc-claw-soft text-pc-claw pc-micro shrink-0 rounded-xs px-1.5 py-0.5">
                {t("models.badge.default")}
              </span>
            )}
            {model.is_virtual && (
              <span className="bg-pc-surface-3 text-pc-muted pc-micro shrink-0 rounded-xs px-1.5 py-0.5">
                {t("models.badge.virtual")}
              </span>
            )}
          </div>
        </div>

        <div className="flex shrink-0 items-center gap-0.5">
          {model.is_default ? (
            <span
              className="text-pc-claw p-1"
              title={t("models.badge.default")}
            >
              <IconStarFilled className="size-3.5" />
            </span>
          ) : (
            <Tooltip delayDuration={!canSetDefault || settingDefault ? 0 : 700}>
              <TooltipTrigger asChild>
                <span
                  className={
                    !canSetDefault || settingDefault
                      ? "cursor-not-allowed"
                      : undefined
                  }
                  tabIndex={!canSetDefault || settingDefault ? 0 : undefined}
                  role={!canSetDefault || settingDefault ? "button" : undefined}
                  aria-disabled={
                    !canSetDefault || settingDefault ? true : undefined
                  }
                  aria-label={
                    !canSetDefault || settingDefault
                      ? setDefaultLabel
                      : undefined
                  }
                  title={
                    !canSetDefault || settingDefault
                      ? setDefaultLabel
                      : undefined
                  }
                >
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    onClick={() => onSetDefault(model)}
                    disabled={settingDefault || !canSetDefault}
                    aria-label={setDefaultLabel}
                    title={setDefaultLabel}
                  >
                    {settingDefault ? (
                      <IconLoader2 className="size-3.5 animate-spin" />
                    ) : (
                      <IconStar className="size-3.5" />
                    )}
                  </Button>
                </span>
              </TooltipTrigger>
              <TooltipContent>{setDefaultDisabledReason}</TooltipContent>
            </Tooltip>
          )}

          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() => onEdit(model)}
            aria-label={editLabel}
            title={editLabel}
          >
            <IconEdit className="size-3.5" />
          </Button>

          <Tooltip delayDuration={deleteDisabled ? 0 : 700}>
            <TooltipTrigger asChild>
              <span
                className={deleteDisabled ? "cursor-not-allowed" : undefined}
                tabIndex={deleteDisabled ? 0 : undefined}
                role={deleteDisabled ? "button" : undefined}
                aria-disabled={deleteDisabled ? true : undefined}
                aria-label={deleteDisabled ? deleteLabel : undefined}
                title={deleteDisabled ? deleteLabel : undefined}
              >
                <Button
                  variant="ghost"
                  size="icon-sm"
                  onClick={() => onDelete(model)}
                  disabled={deleteDisabled}
                  aria-label={deleteLabel}
                  title={deleteLabel}
                  className="text-pc-muted hover:text-pc-danger hover:bg-pc-danger-soft"
                >
                  <IconTrash className="size-3.5" />
                </Button>
              </span>
            </TooltipTrigger>
            <TooltipContent>{deleteDisabledReason}</TooltipContent>
          </Tooltip>
        </div>
      </div>

      <p
        className="text-pc-muted pc-mono truncate text-xs leading-snug"
        dir="ltr"
      >
        {model.model}
      </p>

      <div className="flex items-center gap-2">
        {isOAuth ? (
          <span className="text-pc-muted bg-pc-surface-3 pc-micro rounded-xs px-1.5 py-0.5">
            OAuth
          </span>
        ) : status === "available" && model.api_key ? (
          <span className="text-pc-faint pc-mono flex items-center gap-1 text-[11px]">
            <IconKey className="size-3 shrink-0" />
            <span className="truncate" dir="ltr">
              {model.api_key}
            </span>
          </span>
        ) : (
          <span className="text-pc-faint text-[11px]">{statusLabel}</span>
        )}
      </div>
    </div>
  )
}
