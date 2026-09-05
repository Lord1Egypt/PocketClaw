import { IconArrowDown, IconArrowUp, IconPlus, IconX } from "@tabler/icons-react"
import { useEffect, useMemo, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import {
  type ModelInfo,
  type ModelProviderOption,
  materializeModel,
  setModelFallbacks,
} from "@/api/models"
import { ConfiguredModelPicker } from "@/components/models/configured-model-picker"
import { Button } from "@/components/ui/button"
import { useConfiguredModels } from "@/hooks/use-configured-models"
import {
  type ConfiguredProviderGroup,
  type SelectableModel,
  modelKey,
} from "@/lib/configured-model-source"
import { saveAndApplyGatewayConfig } from "@/lib/restart-required"

interface FallbackModelsSectionProps {
  models: ModelInfo[]
  fallbacks: string[]
  defaultModelName?: string
  providerOptions?: ModelProviderOption[]
  onSaved: () => Promise<void> | void
}

/**
 * Ordered fallback chain for the default chat model.
 *
 * Each entry references a configured model by name. The referenced model is
 * used with its own provider, credentials and base URL — nothing is copied from
 * the primary — so selecting one here never moves an API key between providers.
 */
export function FallbackModelsSection({
  models,
  fallbacks,
  defaultModelName,
  providerOptions,
  onSaved,
}: FallbackModelsSectionProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<string[]>(fallbacks)
  const [saving, setSaving] = useState(false)
  const [adding, setAdding] = useState(false)

  // The one source every routing selector uses. It is what stops the shipped
  // keyless provider templates — azure, groq, cerebras, ollama and the rest —
  // from appearing here at all.
  const { groups, discover, discoverAll } = useConfiguredModels({
    models,
    providerOptions,
    defaultModelName,
  })

  // Re-sync when the page reloads its model list, so a save elsewhere does not
  // leave this section showing stale entries.
  useEffect(() => {
    setDraft(fallbacks)
  }, [fallbacks])

  // Populate the picker once the configured providers are known, so opening it
  // does not start with an empty list. Each provider is queried independently.
  useEffect(() => {
    void discoverAll()
  }, [discoverAll])

  // A model cannot be a fallback if it is already in the chain or is the
  // primary itself: a primary listed as its own fallback would make the chain
  // retry the candidate that just failed. Everything else about selectability
  // — configured provider, routable entry — is decided by the shared source.
  const takenKeys = useMemo(() => {
    const taken = new Set<string>()
    for (const model of models) {
      const excluded =
        draft.includes(model.model_name) || model.model_name === defaultModelName
      if (excluded) {
        taken.add(modelKey(model.provider, model.api_base, model.model))
      }
    }
    return taken
  }, [models, draft, defaultModelName])

  const hasCandidates = groups.some((group) =>
    group.models.some((model) => !takenKeys.has(model.key)),
  )

  const dirty =
    draft.length !== fallbacks.length ||
    draft.some((name, index) => name !== fallbacks[index])

  const move = (index: number, delta: number) => {
    const target = index + delta
    if (target < 0 || target >= draft.length) return
    const next = [...draft]
    ;[next[index], next[target]] = [next[target], next[index]]
    setDraft(next)
  }

  const handlePick = async (
    model: SelectableModel,
    group: ConfiguredProviderGroup,
  ) => {
    if (model.origin === "configured" && model.modelName) {
      setDraft([...draft, model.modelName])
      return
    }

    // A fallback references model_list by name, and that invariant is not
    // relaxed for discovery — the entry has to exist first. The backend
    // creates it from the provider instance the model was discovered through,
    // inheriting that provider's base URL and stored credential; the key never
    // passes through here.
    setAdding(true)
    try {
      const res = await materializeModel({
        source_index: group.sourceIndex,
        model: model.model,
        // No role: the chain is applied by the single Save below, so a
        // half-applied role is not a state this can reach.
        role: "",
      })
      setDraft((current) =>
        current.includes(res.model_name) ? current : [...current, res.model_name],
      )
      if (res.created) {
        toast.success(
          t("models.discovery.addedModel", { model: res.model_name }),
        )
      }
      await onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setAdding(false)
    }
  }

  const handleSave = async () => {
    setSaving(true)
    try {
      await saveAndApplyGatewayConfig(t, {
        save: () => setModelFallbacks(draft),
        savedMessage: t("models.fallbacks.saved"),
        name: t("models.fallbacks.title"),
      })
      await onSaved()
    } catch (e) {
      toast.error(e instanceof Error ? e.message : t("models.loadError"))
    } finally {
      setSaving(false)
    }
  }

  const describe = (name: string) => {
    const model = models.find((entry) => entry.model_name === name)
    if (!model) {
      // A name with no matching entry means the referenced model was deleted.
      // Showing it rather than hiding it lets the user remove the dead entry.
      return { title: name, detail: t("models.fallbacks.missingModel") }
    }
    return {
      title: model.model_name,
      detail: [model.provider, model.model].filter(Boolean).join(" · "),
    }
  }

  return (
    <section className="border-pc-line bg-pc-surface-1 mt-6 rounded-xl border px-4 py-4">
      <h3 className="text-pc-text text-[1rem] font-semibold">
        {t("models.fallbacks.title")}
      </h3>
      <p className="text-pc-muted mt-1 text-sm">
        {t("models.fallbacks.description")}
      </p>

      {draft.length === 0 && (
        <p className="text-pc-faint mt-3 text-sm">
          {t("models.fallbacks.empty")}
        </p>
      )}

      {draft.length > 0 && (
        <ol className="mt-3 space-y-2">
          {draft.map((name, index) => {
            const { title, detail } = describe(name)
            return (
              <li
                key={name}
                className="border-pc-line bg-pc-surface-2 flex items-center gap-3 rounded-md border px-3 py-2"
              >
                <span className="text-pc-faint pc-mono w-5 shrink-0 text-sm">
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="text-pc-text truncate text-sm">{title}</div>
                  <div className="text-pc-faint pc-mono truncate text-xs" dir="ltr">
                    {detail}
                  </div>
                </div>
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={t("models.fallbacks.moveUp")}
                  disabled={index === 0}
                  onClick={() => move(index, -1)}
                >
                  <IconArrowUp className="size-4" />
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={t("models.fallbacks.moveDown")}
                  disabled={index === draft.length - 1}
                  onClick={() => move(index, 1)}
                >
                  <IconArrowDown className="size-4" />
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  aria-label={t("models.fallbacks.remove")}
                  onClick={() =>
                    setDraft(draft.filter((entry) => entry !== name))
                  }
                >
                  <IconX className="size-4" />
                </Button>
              </li>
            )
          })}
        </ol>
      )}

      <div className="mt-3 flex flex-wrap items-center gap-2">
        <ConfiguredModelPicker
          groups={groups}
          takenKeys={takenKeys}
          triggerLabel={t("models.fallbacks.add")}
          busy={adding}
          onSelect={(model, group) => void handlePick(model, group)}
          onRefresh={(groupKey) => void discover(groupKey)}
        />

        {!hasCandidates && draft.length === 0 && (
          <span className="text-pc-muted text-sm">
            {t("models.fallbacks.noCandidates")}
          </span>
        )}

        <Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
          <IconPlus className="hidden" />
          {saving ? t("models.fallbacks.saving") : t("common.save")}
        </Button>

        {dirty && !saving && (
          <span className="text-pc-muted text-sm">
            {t("models.unsavedPrompt")}
          </span>
        )}
      </div>
    </section>
  )
}
