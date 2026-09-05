/**
 * The one source every model selector draws from.
 *
 * The shipped default config seeds `model_list` with thirty keyless provider
 * templates — zhipu, openai, anthropic, groq, cerebras, volcengine, ollama,
 * azure and the rest. The backend already marks them `unconfigured`, but the
 * Fallback picker never read that flag, so a user who had configured OpenCode
 * and Gemini was offered twenty-eight providers they had never touched.
 *
 * This module is the single place that answers "what may be selected", so the
 * Default, Fallback and any future routing selector cannot drift apart into
 * three catalogs with three different filters.
 *
 * Nothing here reaches for the global provider preset list. A provider the user
 * has not configured contributes nothing at all.
 */
import type { ModelInfo, ModelProviderOption } from "@/api/models"

/// Separator for composite keys. A NUL cannot occur in a provider id, an
/// endpoint or a model id, so no combination of parts can collide.
const SEP = "\u0000"

/** Where a selectable model came from. */
export type ModelOrigin = "configured" | "discovered"

export interface SelectableModel {
  /**
   * Provider instance + model id. Two configured providers can expose the same
   * model id, and they are different models: deduplicating on the id alone
   * would route one provider's model through the other's credential.
   */
  key: string
  /** The upstream model identifier, without a provider prefix. */
  model: string
  origin: ModelOrigin
  /** Present only for `configured`: the model_list alias a role references. */
  modelName?: string
  /** Present only for `configured`: the entry's index in model_list. */
  index?: number
  /** The configured entry whose credential a discovered model would inherit. */
  sourceIndex: number
  available: boolean
  status?: ModelInfo["status"]
  isDefault: boolean
  /**
   * Whether the provider reported that this model accepts images.
   *
   * `undefined` means the provider did not say, which today is every model:
   * the discovery response carries only `id` and `owned_by`. It is deliberately
   * not inferred from the name — a model called `-vision` may not be one, and a
   * routing decision made from a substring is a routing bug. When a provider
   * starts reporting capabilities this is where they land.
   */
  visionCapable?: boolean
}

/** Whether a provider instance's live discovery has run, and how it went. */
export type DiscoveryState =
  | { phase: "unsupported" }
  | { phase: "idle" }
  | { phase: "loading" }
  | { phase: "ready"; fetchedAt: number }
  | { phase: "error"; message: string }

export interface ConfiguredProviderGroup {
  /** Provider instance identity: normalized provider + normalized API base. */
  key: string
  provider: string
  apiBase: string
  label: string
  /** The model_list entry used to authenticate discovery for this instance. */
  sourceIndex: number
  supportsFetch: boolean
  discovery: DiscoveryState
  models: SelectableModel[]
}

/**
 * A trailing slash, a default port or a case difference is the same endpoint.
 * The backend compares bases the same way, so an entry stored without a base
 * and one stored with the provider default have to match here too.
 */
export function normalizeApiBase(raw: string | undefined): string {
  const trimmed = (raw ?? "").trim().replace(/\/+$/, "")
  if (trimmed === "") return ""
  try {
    const url = new URL(trimmed)
    const redundantPort =
      (url.protocol === "https:" && url.port === "443") ||
      (url.protocol === "http:" && url.port === "80")
    const port = redundantPort ? "" : url.port
    const path = url.pathname.replace(/\/+$/, "")
    const host = url.hostname.toLowerCase()
    return `${url.protocol}//${host}${port ? `:${port}` : ""}${path}`
  } catch {
    return trimmed.toLowerCase()
  }
}

export function normalizeProvider(raw: string | undefined): string {
  return (raw ?? "").trim().toLowerCase()
}

/** The identity two selectable models must share to be the same model. */
export function modelKey(
  provider: string | undefined,
  apiBase: string | undefined,
  model: string,
): string {
  return [
    normalizeProvider(provider),
    normalizeApiBase(apiBase),
    model.trim(),
  ].join(SEP)
}

/** The identity two model entries must share to be the same provider instance. */
export function providerInstanceKey(
  provider: string | undefined,
  apiBase: string | undefined,
): string {
  return normalizeProvider(provider) + SEP + normalizeApiBase(apiBase)
}

/**
 * A discovered id may arrive provider-prefixed. The bare id is what a
 * model_list entry stores, with the provider carried in its own field.
 */
export function stripProviderPrefix(
  model: string,
  provider: string | undefined,
): string {
  const trimmed = model.trim()
  const normalized = normalizeProvider(provider)
  if (normalized === "") return trimmed
  const prefix = `${normalized}/`
  return trimmed.toLowerCase().startsWith(prefix)
    ? trimmed.slice(prefix.length)
    : trimmed
}

/**
 * A model entry counts as configured when the backend says its credentials
 * resolve. `unconfigured` is exactly the shipped-template case.
 */
export function isConfiguredEntry(model: ModelInfo): boolean {
  return model.status !== "unconfigured"
}

/** Entries that can hold a chat routing role at all. */
export function isRoutableEntry(model: ModelInfo): boolean {
  return model.is_virtual !== true && model.default_model_allowed !== false
}

function providerOptionFor(
  provider: string,
  options: ModelProviderOption[] | undefined,
): ModelProviderOption | undefined {
  return options?.find(
    (option) => normalizeProvider(option.id) === normalizeProvider(provider),
  )
}

export interface BuildGroupsInput {
  models: ModelInfo[]
  providerOptions?: ModelProviderOption[]
  /** Live discovery results, keyed by provider instance. */
  discovered?: Record<string, string[]>
  /**
   * Provider-reported vision capability, keyed by model key. Only populated
   * from what a provider actually says; never inferred.
   */
  visionCapable?: Record<string, boolean>
  /** Discovery state, keyed by provider instance. */
  discoveryState?: Record<string, DiscoveryState>
  defaultModelName?: string
}

/**
 * Builds the grouped, deduplicated selection list.
 *
 * Configured entries always appear. Discovered ids appear beneath them, minus
 * any already configured for the same provider instance — the same model
 * offered twice is one row, and the configured one wins because it already has
 * an alias a role can reference.
 */
export function buildConfiguredProviderGroups({
  models,
  providerOptions,
  discovered = {},
  discoveryState = {},
  visionCapable = {},
  defaultModelName,
}: BuildGroupsInput): ConfiguredProviderGroup[] {
  const groups = new Map<string, ConfiguredProviderGroup>()

  for (const model of models) {
    if (!isConfiguredEntry(model) || !isRoutableEntry(model)) continue
    // The backend's own index, not the array position: a caller that filters
    // this list must not silently shift which credential is resolved.
    const index = model.index

    const provider = model.provider ?? ""
    const key = providerInstanceKey(provider, model.api_base)
    let group = groups.get(key)
    if (!group) {
      const option = providerOptionFor(provider, providerOptions)
      const fetchable = option?.supports_fetch === true
      group = {
        key,
        provider,
        apiBase: model.api_base ?? "",
        label: option?.display_name?.trim() || provider,
        sourceIndex: index,
        supportsFetch: fetchable,
        discovery:
          discoveryState[key] ??
          (fetchable ? { phase: "idle" } : { phase: "unsupported" }),
        models: [],
      }
      groups.set(key, group)
    }

    const bare = stripProviderPrefix(model.model, provider)
    group.models.push({
      key: modelKey(provider, model.api_base, bare),
      model: bare,
      origin: "configured",
      modelName: model.model_name,
      index,
      sourceIndex: group.sourceIndex,
      available: model.available,
      status: model.status,
      isDefault: model.model_name === defaultModelName,
      visionCapable: visionCapable[modelKey(provider, model.api_base, bare)],
    })
  }

  for (const group of groups.values()) {
    const ids = discovered[group.key]
    if (!ids?.length) continue

    const seen = new Set(group.models.map((entry) => entry.key))
    for (const raw of ids) {
      const bare = stripProviderPrefix(raw, group.provider)
      if (bare === "") continue
      const key = modelKey(group.provider, group.apiBase, bare)
      if (seen.has(key)) continue
      seen.add(key)
      group.models.push({
        key,
        model: bare,
        origin: "discovered",
        sourceIndex: group.sourceIndex,
        available: true,
        isDefault: false,
        visionCapable: visionCapable[key],
      })
    }
  }

  for (const group of groups.values()) {
    group.models.sort((a, b) => {
      // Configured first: those are the ones already routable.
      if (a.origin !== b.origin) return a.origin === "configured" ? -1 : 1
      return a.model.localeCompare(b.model)
    })
  }

  return [...groups.values()].sort((a, b) => a.label.localeCompare(b.label))
}
