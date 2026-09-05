import { IconChevronDown } from "@tabler/icons-react"
import { useState } from "react"

import type { ModelInfo } from "@/api/models"

import { ModelCard } from "./model-card"
import { ProviderIcon } from "./provider-icon"
import type { ProviderCatalogEntry } from "./provider-registry"

interface ProviderSectionProps {
  provider: Pick<ProviderCatalogEntry, "key" | "label" | "iconSlug" | "domain">
  models: ModelInfo[]
  onEdit: (model: ModelInfo) => void
  onSetDefault: (model: ModelInfo) => void
  onDelete: (model: ModelInfo) => void
  settingDefaultIndex: number | null
}

export function ProviderSection({
  provider,
  models,
  onEdit,
  onSetDefault,
  onDelete,
  settingDefaultIndex,
}: ProviderSectionProps) {
  const [open, setOpen] = useState(true)

  return (
    <section className="my-7">
      <button
        type="button"
        onClick={() => setOpen((v) => !v)}
        className="mb-3 grid w-full grid-cols-[1fr_auto_1fr_auto] items-center gap-2 px-1 py-1.5 text-left"
        aria-expanded={open}
      >
        <div className="border-pc-line border-t" />
        <span className="text-pc-faint pc-micro text-center">
          <span className="bg-pc-canvas inline-flex items-center gap-1.5 px-2">
            <ProviderIcon provider={provider} />
            {provider.label}
          </span>
        </span>
        <div className="border-pc-line border-t" />
        <span className="flex justify-end">
          <IconChevronDown
            className={[
              "text-muted-foreground size-4 transition-transform",
              open ? "rotate-180" : "",
            ].join(" ")}
          />
        </span>
      </button>

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
