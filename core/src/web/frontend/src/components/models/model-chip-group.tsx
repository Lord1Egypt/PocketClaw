import { Badge } from "@/components/ui/badge"

/**
 * Where a model id on this screen came from.
 *
 * PC-DEF-041. The Add and Edit sheets rendered three sources of model ids as
 * three identical rows of chips with no labels: the provider preset's curated
 * `common_models`, whatever a previous Fetch Models saved into
 * `model_catalogs.json`, and this session's live result. A user could not tell
 * a name this build happens to know from one the provider had just confirmed it
 * serves, which is how a model ends up configured against an endpoint that does
 * not serve it.
 */
export type ModelChipOrigin = "suggestion" | "cached" | "verified"

interface ModelChipGroupProps {
  origin: ModelChipOrigin
  label: string
  hint?: string
  models: string[]
  selected: string
  onSelect: (model: string) => void
}

export function ModelChipGroup({
  origin,
  label,
  hint,
  models,
  selected,
  onSelect,
}: ModelChipGroupProps) {
  if (models.length === 0) return null

  return (
    <div className="flex flex-col gap-1" data-origin={origin}>
      <div className="flex flex-wrap items-baseline gap-x-2">
        <span className="text-pc-faint pc-micro">{label}</span>
        {hint && (
          <span className="text-pc-faint min-w-0 text-xs">{hint}</span>
        )}
      </div>
      <div className="flex flex-wrap gap-1.5">
        {models.map((model) => (
          <Badge
            key={model}
            // Only a verified id is shown as a confirmed choice. A suggestion
            // stays visually secondary however it is selected, because
            // selecting one does not make it any more real.
            variant={
              origin === "suggestion"
                ? "secondary"
                : selected === model
                  ? "default"
                  : "outline"
            }
            className="cursor-pointer font-mono text-xs"
            onClick={() => onSelect(model)}
          >
            {model}
          </Badge>
        ))}
      </div>
    </div>
  )
}
