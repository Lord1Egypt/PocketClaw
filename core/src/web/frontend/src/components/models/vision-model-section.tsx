import { IconPhoto, IconX } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ModelInfo,
  type ModelProviderOption,
  materializeModel,
  setVisionModel,
} from "@/api/models"
import { ConfiguredModelPicker } from "@/components/models/configured-model-picker"
import { Button } from "@/components/ui/button"
import { useConfiguredModels } from "@/hooks/use-configured-models"
import type {
  ConfiguredProviderGroup,
  SelectableModel,
} from "@/lib/configured-model-source"
import { applyGatewayConfigIfRequired } from "@/lib/restart-required"

interface VisionModelSectionProps {
  models: ModelInfo[]
  visionModelName?: string
  defaultModelName?: string
  providerOptions?: ModelProviderOption[]
  onSaved: () => Promise<void> | void
}

/**
 * The dedicated model for turns that carry an image.
 *
 * Unset is a first-class state, not a missing setting: with no vision model
 * configured an image turn goes to the default model, which is exactly what
 * every install did before this control existed. That is why the empty state
 * reads as "Auto" rather than as something unconfigured.
 *
 * Only the model for that turn changes. The session, the rolling summary, the
 * tool set and the system prompt are untouched, and a text turn following an
 * image turn goes back to the default — the routing decision reads the current
 * turn only.
 *
 * The fallback chain is the one thing that is *not* shared. An image turn walks
 * `image_model` plus `image_model_fallbacks`, not the Fallback Models list
 * below; the two chains are separate in `instance.go` and `routeMediaTurn`
 * replaces the active candidate set outright. Since this control writes only
 * `image_model`, a vision model configured here has no fallback behind it — if
 * it is unavailable the image turn fails rather than falling through to the
 * text chain. `image_model_fallbacks` is real and already honoured, but nothing
 * in the Dashboard writes it yet.
 *
 * Selection comes from the same configured-provider source as the Default and
 * Fallback selectors, so an unconfigured provider template can no more appear
 * here than it can there.
 */
export function VisionModelSection({
  models,
  visionModelName,
  defaultModelName,
  providerOptions,
  onSaved,
}: VisionModelSectionProps) {
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

  const current = models.find((model) => model.model_name === visionModelName)

  // The model already holding the role cannot be picked again.
  const takenKeys = useMemo(() => {
    const taken = new Set<string>()
    if (!current) return taken
    for (const group of groups) {
      for (const model of group.models) {
        if (model.modelName === current.model_name) taken.add(model.key)
      }
    }
    return taken
  }, [groups, current])

  const apply = async (run: () => Promise<unknown>, name: string) => {
    setBusy(true)
    try {
      await run()
      await applyGatewayConfigIfRequired(t, {
        savedMessage: t("models.defaultChangeSuccess"),
        name,
      })
      await onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setBusy(false)
    }
  }

  const handlePick = (
    model: SelectableModel,
    group: ConfiguredProviderGroup,
  ) =>
    apply(
      () =>
        model.origin === "configured" && model.modelName
          ? setVisionModel(model.modelName)
          : // One call: create the entry from the provider instance it was
            // discovered through, then take the role. A rejected assignment
            // writes nothing rather than leaving a stray model behind.
            materializeModel({
              source_index: group.sourceIndex,
              model: model.model,
              role: "vision",
            }),
      model.model,
    )

  return (
    <section className="border-pc-line bg-pc-surface-1 mt-6 rounded-xl border px-4 py-4">
      <h3 className="text-pc-text text-[1rem] font-semibold">
        {t("models.visionModel.title")}
      </h3>
      <p className="text-pc-muted mt-1 text-sm">
        {t("models.visionModel.description")}
      </p>

      <div className="mt-3 flex flex-wrap items-center gap-2">
        {current ? (
          <span className="border-pc-line bg-pc-surface-2 flex min-w-0 items-center gap-2 rounded-md border px-3 py-2">
            <IconPhoto className="text-pc-claw size-4 shrink-0" />
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
          <span className="border-pc-line bg-pc-surface-2 text-pc-muted flex items-center gap-2 rounded-md border border-dashed px-3 py-2 text-sm">
            {t("models.visionModel.auto")}
          </span>
        )}

        <ConfiguredModelPicker
          groups={groups}
          takenKeys={takenKeys}
          triggerLabel={t("models.visionModel.change")}
          busy={busy}
          onSelect={(model, group) => void handlePick(model, group)}
          onRefresh={(groupKey) => void discover(groupKey)}
        />

        {current && (
          <Button
            variant="ghost"
            size="sm"
            disabled={busy}
            onClick={() => void apply(() => setVisionModel(""), "")}
          >
            <IconX className="size-4" />
            {t("models.visionModel.clear")}
          </Button>
        )}
      </div>

      <p className="text-pc-faint mt-3 text-xs">
        {current
          ? t("models.visionModel.note")
          : t("models.visionModel.autoHint")}
      </p>

      {/*
        No per-model capability badge is shown, because no provider reports
        capabilities today — the discovery response carries only an id and an
        owner. Guessing from the name would put a badge on a model that may not
        accept images, so the honest statement is that support is unknown and
        the choice is the user's.
      */}
      <p className="text-pc-faint mt-1 text-xs">
        {t("models.visionModel.capabilityUnknown")}
      </p>
    </section>
  )
}
