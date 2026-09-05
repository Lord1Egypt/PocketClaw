import { IconCheck, IconSearch } from "@tabler/icons-react"
import { useMemo, useState } from "react"
import { useTranslation } from "react-i18next"

import type { ModelProviderOption } from "@/api/models"
import { Input } from "@/components/ui/input"

import { ProviderIcon } from "./provider-icon"
import {
  type ProviderCatalogEntry,
  type ProviderCategory,
  searchProviders,
} from "./provider-registry"

// Rendering order of the groups in the picker. Cloud providers come first
// because they are the ones the API-key-only flow was built for.
const CATEGORY_ORDER: ProviderCategory[] = [
  "cloud",
  "local",
  "custom",
  "managed",
  "speech",
]

interface ProviderPickerProps {
  providerOptions?: ModelProviderOption[]
  configuredProviders: Set<string>
  onSelect: (provider: ProviderCatalogEntry) => void
}

export function ProviderPicker({
  providerOptions,
  configuredProviders,
  onSelect,
}: ProviderPickerProps) {
  const { t } = useTranslation()
  const [query, setQuery] = useState("")

  const groups = useMemo(() => {
    const matches = searchProviders(query, providerOptions)
    return CATEGORY_ORDER.map((category) => ({
      category,
      providers: matches.filter((provider) => provider.category === category),
    })).filter((group) => group.providers.length > 0)
  }, [query, providerOptions])

  return (
    <div className="space-y-4">
      <div className="relative">
        <IconSearch className="text-muted-foreground pointer-events-none absolute top-1/2 start-3 size-4 -translate-y-1/2" />
        <Input
          value={query}
          onChange={(event) => setQuery(event.target.value)}
          placeholder={t("models.picker.searchPlaceholder")}
          className="ps-9"
          autoFocus
        />
      </div>

      {groups.length === 0 && (
        <p className="text-muted-foreground py-8 text-center text-sm">
          {t("models.picker.noMatches")}
        </p>
      )}

      <div className="space-y-5">
        {groups.map((group) => (
          <section key={group.category} className="space-y-2">
            <h3 className="text-muted-foreground text-xs font-medium tracking-wide uppercase">
              {t(`models.picker.category.${group.category}`)}
            </h3>
            <div className="grid gap-2 sm:grid-cols-2">
              {group.providers.map((provider) => {
                const configured = configuredProviders.has(provider.key)
                return (
                  <button
                    key={provider.key}
                    type="button"
                    onClick={() => onSelect(provider)}
                    className="hover:border-primary/60 hover:bg-accent/40 focus-visible:ring-ring flex items-center gap-3 rounded-lg border p-3 text-start transition-colors focus-visible:ring-2 focus-visible:outline-none"
                  >
                    <ProviderIcon provider={provider} size="md" />
                    <span className="min-w-0 flex-1">
                      <span className="flex items-center gap-1.5">
                        <span className="truncate text-sm font-medium">
                          {provider.label}
                        </span>
                        {configured && (
                          <IconCheck className="size-3.5 shrink-0 text-pc-success" />
                        )}
                      </span>
                      <span className="text-muted-foreground mt-0.5 block truncate text-xs">
                        {configured
                          ? t("models.picker.statusConfigured")
                          : t(`models.picker.description.${provider.category}`)}
                      </span>
                    </span>
                  </button>
                )
              })}
            </div>
          </section>
        ))}
      </div>
    </div>
  )
}
