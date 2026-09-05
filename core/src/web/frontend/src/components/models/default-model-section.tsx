import { IconStar } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ModelInfo,
  type ModelProviderOption,
  materializeModel,
} from "@/api/models"
import { ConfiguredModelPicker } from "@/components/models/configured-model-picker"
import { useConfiguredModels } from "@/hooks/use-configured-models"
import {
  type ConfiguredProviderGroup,
  type SelectableModel,
} from "@/lib/configured-model-source"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"
import { setDefaultModel } from "@/api/models"

interface DefaultModelSectionProps {
  models: ModelInfo[]
  defaultModelName?: string
  providerOptions?: ModelProviderOption[]
  onSaved: () => Promise<void> | void
}

/**
 * The default chat model, chosen from the same source as the fallback chain.
 *
 * Picking an already-configured model is a plain set-default. Picking a
 * discovered one is a single backend call that creates the entry and assigns
 * the role together — the reference must name a `model_list` entry, and doing
 * both halves in one operation is what stops a rejected assignment leaving a
 * stray model behind.
 */
export function DefaultModelSection({
  models,
  defaultModelName,
  providerOptions,
  onSaved,
}: DefaultModelSectionProps) {
  const { t } = useTranslation()
  const [busy, setBusy] = useState(false)

  const { groups, discover, discoverAll } = useConfiguredModels({
    models,
    providerOptions,
    defaultModelName,
  })

  useEffect(() => {
    void discoverAll()
  }, [discoverAll])

  // The model already holding the role cannot be selected again.
  const takenKeys = useMemo(() => {
    const taken = new Set<string>()
    for (const group of groups) {
      for (const model of group.models) {
        if (model.isDefault) taken.add(model.key)
      }
    }
    return taken
  }, [groups])

  const current = models.find((model) => model.model_name === defaultModelName)

  const handlePick = async (
    model: SelectableModel,
    group: ConfiguredProviderGroup,
  ) => {
    setBusy(true)
    try {
      if (model.origin === "configured" && model.modelName) {
        await setDefaultModel(model.modelName)
      } else {
        // One call: create the entry from the provider instance it was
        // discovered through, then take the role. The credential is read from
        // stored config on the backend; only an index travels from here.
        await materializeModel({
          source_index: group.sourceIndex,
          model: model.model,
          role: "default",
        })
      }
      await applyGatewayConfigIfRequired(t, {
        savedMessage: t("models.defaultChangeSuccess"),
        name: model.model,
      })
      await onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setBusy(false)
    }
  }

  return (
    <section className="border-pc-line bg-pc-surface-1 mt-6 rounded-xl border px-4 py-4">
      <h3 className="text-pc-text text-[1rem] font-semibold">
        {t("models.defaultModel.title")}
      </h3>
      <p className="text-pc-muted mt-1 text-sm">
        {t("models.defaultModel.description")}
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        {current ? (
          <span className="border-pc-line bg-pc-surface-2 flex min-w-0 items-center gap-2 rounded-md border px-3 py-2">
            <IconStar className="text-pc-claw size-4 shrink-0" />
            <span className="min-w-0">
              <span className="text-pc-text block truncate text-sm">
                {current.model_name}
              </span>
              <span
                className="text-pc-faint pc-mono block truncate text-xs"
                dir="ltr"
              >
                {[current.provider, current.model].filter(Boolean).join(" · ")}
              </span>
            </span>
          </span>
        ) : (
          <span className="text-pc-muted text-sm">
            {t("models.defaultModel.none")}
          </span>
        )}

        <ConfiguredModelPicker
          groups={groups}
          takenKeys={takenKeys}
          triggerLabel={t("models.defaultModel.change")}
          busy={busy}
          onSelect={(model, group) => void handlePick(model, group)}
          onRefresh={(groupKey) => void discover(groupKey)}
        />
      </div>
    </section>
  )
}
