import {
  IconArrowDown,
  IconArrowUp,
  IconPlus,
  IconX,
} from "@tabler/icons-react"
import { useEffect, useState } from "react"
import { useTranslation } from "react-i18next"
import { toast } from "sonner"

import { type ModelInfo, setModelFallbacks } from "@/api/models"
import { Button } from "@/components/ui/button"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { saveAndApplyGatewayConfig } from "@/lib/restart-required"

interface FallbackModelsSectionProps {
  models: ModelInfo[]
  fallbacks: string[]
  defaultModelName?: string
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
  onSaved,
}: FallbackModelsSectionProps) {
  const { t } = useTranslation()
  const [draft, setDraft] = useState<string[]>(fallbacks)
  const [saving, setSaving] = useState(false)

  // Re-sync when the page reloads its model list, so a save elsewhere does not
  // leave this section showing stale entries.
  useEffect(() => {
    setDraft(fallbacks)
  }, [fallbacks])

  // A model can be a fallback if it is a real chat model, is not already in the
  // chain, and is not the primary itself. The last exclusion matters: a primary
  // listed as its own fallback would make the chain retry the candidate that
  // just failed.
  const selectable = models.filter(
    (model) =>
      !model.is_virtual &&
      model.default_model_allowed !== false &&
      model.model_name !== defaultModelName &&
      !draft.includes(model.model_name),
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
    <section className="mt-6 rounded-lg border px-4 py-4">
      <h3 className="text-sm font-medium">{t("models.fallbacks.title")}</h3>
      <p className="text-muted-foreground mt-1 text-sm">
        {t("models.fallbacks.description")}
      </p>

      {draft.length === 0 && (
        <p className="text-muted-foreground mt-3 text-sm">
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
                className="flex items-center gap-3 rounded-md border px-3 py-2"
              >
                <span className="text-muted-foreground w-5 shrink-0 text-sm tabular-nums">
                  {index + 1}
                </span>
                <div className="min-w-0 flex-1">
                  <div className="truncate text-sm">{title}</div>
                  <div className="text-muted-foreground truncate text-xs">
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
        <Select
          value=""
          onValueChange={(name) => setDraft([...draft, name])}
          disabled={selectable.length === 0}
        >
          <SelectTrigger className="w-64" aria-label={t("models.fallbacks.add")}>
            <SelectValue placeholder={t("models.fallbacks.add")} />
          </SelectTrigger>
          <SelectContent>
            {selectable.map((model) => (
              <SelectItem key={model.model_name} value={model.model_name}>
                {model.model_name}
                <span className="text-muted-foreground ml-2 text-xs">
                  {[model.provider, model.model].filter(Boolean).join(" · ")}
                </span>
              </SelectItem>
            ))}
          </SelectContent>
        </Select>

        {selectable.length === 0 && draft.length === 0 && (
          <span className="text-muted-foreground text-sm">
            {t("models.fallbacks.noCandidates")}
          </span>
        )}

        <Button size="sm" onClick={handleSave} disabled={!dirty || saving}>
          <IconPlus className="hidden" />
          {saving ? t("models.fallbacks.saving") : t("common.save")}
        </Button>

        {dirty && !saving && (
          <span className="text-muted-foreground text-sm">
            {t("models.unsavedPrompt")}
          </span>
        )}
      </div>
    </section>
  )
}
