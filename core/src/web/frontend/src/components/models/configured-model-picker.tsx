import {
  IconAlertTriangle,
  IconLoader2,
  IconRefresh,
} from "@tabler/icons-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import type { SelectableModel } from "@/lib/configured-model-source"
import type { ConfiguredProviderGroup } from "@/lib/configured-model-source"
import { Button } from "@/components/ui/button"
import { Input } from "@/components/ui/input"
import {
  Popover,
  PopoverContent,
  PopoverTrigger,
} from "@/components/ui/popover"

interface ConfiguredModelPickerProps {
  groups: ConfiguredProviderGroup[]
  /** Model keys already spoken for, shown but not selectable. */
  takenKeys?: Set<string>
  triggerLabel: string
  busy?: boolean
  onSelect: (model: SelectableModel, group: ConfiguredProviderGroup) => void
  onRefresh: (groupKey: string) => void
  className?: string
}

/**
 * The picker every routing selector opens.
 *
 * It lists only providers the user has configured, and under each one both the
 * models already in `model_list` and whatever that provider's API reports.
 * There is no path from here to the global provider catalog: a provider with
 * no credentials contributes no group at all, which is the whole point.
 *
 * Discovery state is per provider. One provider failing shows a retry on that
 * group and leaves every other group alone.
 */
export function ConfiguredModelPicker({
  groups,
  takenKeys,
  triggerLabel,
  busy = false,
  onSelect,
  onRefresh,
  className,
}: ConfiguredModelPickerProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(false)
  const [filter, setFilter] = useState("")

  const needle = filter.trim().toLowerCase()
  const visible = useMemo(() => {
    if (needle === "") return groups
    return groups
      .map((group) => ({
        ...group,
        models: group.models.filter((model) =>
          model.model.toLowerCase().includes(needle),
        ),
      }))
      .filter(
        (group) => group.models.length > 0 || group.label.toLowerCase().includes(needle),
      )
  }, [groups, needle])

  return (
    <Popover
      open={open}
      onOpenChange={(next) => {
        setOpen(next)
        if (!next) setFilter("")
      }}
    >
      <PopoverTrigger asChild>
        <Button
          variant="outline"
          size="sm"
          className={className}
          disabled={busy}
          aria-label={triggerLabel}
        >
          {busy ? (
            <IconLoader2 className="size-4 animate-spin" />
          ) : (
            <IconRefresh className="hidden" />
          )}
          {busy ? t("models.discovery.adding") : triggerLabel}
        </Button>
      </PopoverTrigger>

      <PopoverContent align="start" className="w-80 p-0">
        <div className="border-b-pc-line border-b p-2">
          <Input
            value={filter}
            onChange={(e) => setFilter(e.target.value)}
            placeholder={t("models.discovery.searchPlaceholder")}
            aria-label={t("models.discovery.searchPlaceholder")}
            className="h-8"
          />
        </div>

        <div className="max-h-[19rem] overflow-y-auto p-1">
          {groups.length === 0 && (
            <p className="text-pc-muted px-3 py-6 text-center text-sm">
              {t("models.discovery.empty")}
            </p>
          )}

          {visible.map((group) => (
            <section key={group.key} className="mb-1">
              <header className="flex items-center justify-between gap-2 px-2 py-1.5">
                <span className="text-pc-faint pc-micro min-w-0 truncate">
                  {group.label}
                </span>
                {group.supportsFetch && (
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    className="size-7 shrink-0"
                    onClick={() => onRefresh(group.key)}
                    disabled={group.discovery.phase === "loading"}
                    aria-label={
                      group.discovery.phase === "error"
                        ? t("models.discovery.retry")
                        : t("models.discovery.refresh")
                    }
                    title={
                      group.discovery.phase === "error"
                        ? t("models.discovery.retry")
                        : t("models.discovery.refresh")
                    }
                  >
                    {group.discovery.phase === "loading" ? (
                      <IconLoader2 className="size-3.5 animate-spin" />
                    ) : (
                      <IconRefresh className="size-3.5" />
                    )}
                  </Button>
                )}
              </header>

              {/* One provider's failure is that provider's row, never an empty
                  picker and never a reason to show unconfigured providers. */}
              {group.discovery.phase === "error" && (
                <p className="text-pc-warning flex items-start gap-1.5 px-2 pb-1.5 text-xs">
                  <IconAlertTriangle className="mt-0.5 size-3.5 shrink-0" />
                  <span className="min-w-0 break-words">
                    {t("models.discovery.failed")}
                  </span>
                </p>
              )}

              {group.models.length === 0 &&
                group.discovery.phase !== "error" && (
                  <p className="text-pc-faint px-2 pb-1.5 text-xs">
                    {group.supportsFetch
                      ? t("models.discovery.noModels")
                      : t("models.discovery.unsupported")}
                  </p>
                )}

              <ul>
                {group.models.map((model) => {
                  const taken = takenKeys?.has(model.key) ?? false
                  return (
                    <li key={model.key}>
                      <button
                        type="button"
                        disabled={taken}
                        onClick={() => {
                          onSelect(model, group)
                          setOpen(false)
                        }}
                        className="hover:bg-pc-surface-2 flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-start transition-colors disabled:opacity-40"
                      >
                        <span
                          className="pc-mono text-pc-text min-w-0 flex-1 truncate text-xs"
                          dir="ltr"
                        >
                          {model.model}
                        </span>
                        {/* Configured and discovered are different things: one
                            is already routable, the other is created on use. */}
                        <span
                          className={`pc-micro shrink-0 rounded-xs px-1.5 py-0.5 ${
                            model.origin === "configured"
                              ? "bg-pc-claw-soft text-pc-claw"
                              : "bg-pc-surface-3 text-pc-muted"
                          }`}
                        >
                          {model.origin === "configured"
                            ? t("models.discovery.configured")
                            : t("models.discovery.discovered")}
                        </span>
                      </button>
                    </li>
                  )
                })}
              </ul>
            </section>
          ))}
        </div>
      </PopoverContent>
    </Popover>
  )
}
