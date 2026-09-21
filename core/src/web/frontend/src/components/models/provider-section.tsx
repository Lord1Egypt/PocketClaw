import { IconChevronDown, IconSettings } from "@tabler/icons-react"
import { useState } from "react"
import { useTranslation } from "react-i18next"

import type { ModelInfo } from "@/api/models"
import { Button } from "@/components/ui/button"

import { ModelCard } from "./model-card"
import { ProviderIcon } from "./provider-icon"
import type { ProviderCatalogEntry } from "./provider-registry"

interface ProviderSectionProps {
  provider: Pick<ProviderCatalogEntry, "key" | "label" | "iconSlug" | "domain">
  models: ModelInfo[]
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  /** Opens provider-scoped management for this provider. */
  onManageProvider: (providerKey: string) => void
  settingDefaultIndex: number | null
}

export function ProviderSection({
  provider,
  models,
  onEdit,
  onSetDefault,
  onDelete,
  onManageProvider,
  settingDefaultIndex,
}: ProviderSectionProps) {
  const { t } = useTranslation()
  const [open, setOpen] = useState(true)

  return (
    <section className="my-7">
      {/* The provider heading is also where the provider itself is managed.
          It used to be a divider and a label with no action on it at all, so a
          configured provider could be seen but not edited, re-keyed or removed.
          Manage is a labelled control rather than an icon with a tooltip: on a
          touch screen there is no hover, so a tooltip explains nothing. */}
      <div className="mb-3 flex items-center gap-2 px-1">
        <button
          type="button"
          onClick={() => setOpen((v) => !v)}
          className="flex min-h-10 min-w-0 flex-1 items-center gap-2 text-left"
          aria-expanded={open}
        >
          <span className="text-pc-faint pc-micro inline-flex min-w-0 items-center gap-1.5">
            <ProviderIcon provider={provider} />
            <span className="truncate">{provider.label}</span>
          </span>
          <IconChevronDown
            className={[
              "text-muted-foreground size-4 shrink-0 transition-transform",
              open ? "rotate-180" : "",
            ].join(" ")}
          />
          <span className="border-pc-line min-w-4 flex-1 border-t" />
        </button>
        <Button
          variant="outline"
          size="sm"
          className="min-h-10 shrink-0"
          onClick={() => onManageProvider(provider.key)}
        >
          <IconSettings className="size-4" />
          {t("models.provider.manage")}
        </Button>
      </div>

      {open && (
        <div className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {models.map((model) => (
            <ModelCard
              key={model.model_name}
              model={model}
              onEdit={onEdit}
              onSetDefault={onSetDefault}
              onDelete={onDelete}
              settingDefault={settingDefaultIndex === model.index}
            />
          ))}
        </div>
      )}
    </section>
  )
}
