import type { ModelProviderOption } from "@/api/models"

export type ProviderCategory =
  | "cloud"
  | "local"
  | "managed"
  | "custom"
  | "speech"

export interface ProviderCatalogEntry {
  key: string
  label: string
  category: ProviderCategory
  iconSlug?: string
  domain?: string
  documentationUrl?: string
  priority: number
  isLocal: boolean
  defaultApiBase?: string
  requiresApiKey: boolean
  createAllowed: boolean
  defaultModelAllowed: boolean
  supportsFetch: boolean
  defaultAuthMethod?: string
  authMethodLocked?: boolean
  emptyApiKeyAllowed?: boolean
  commonModels: string[]
  aliases: string[]
}

// Frontend still needs the same trim/lower normalization as the backend
// NormalizeProvider before it can look up canonical IDs in provider_options.
// This helper does not define provider semantics; aliases and canonical IDs
// still come entirely from the backend payload.
function normalizeProvider(provider?: string): string {
  return provider?.trim().toLowerCase() || ""
}

const PROVIDER_CATEGORIES: ProviderCategory[] = [
  "cloud",
  "local",
  "managed",
  "custom",
  "speech",
]

// An unrecognized category from a newer backend is grouped with the cloud
// providers rather than dropped, so a provider can never become unreachable in
// the picker because the frontend has not been rebuilt yet.
function normalizeCategory(category?: string): ProviderCategory {
  const value = (category || "").trim().toLowerCase() as ProviderCategory
  return PROVIDER_CATEGORIES.includes(value) ? value : "cloud"
}

function toCatalogEntry(option: ModelProviderOption): ProviderCatalogEntry {
  const defaultApiBase = option.default_api_base || undefined
  return {
    key: option.id,
    label: option.display_name || option.id,
    category: normalizeCategory(option.category),
    iconSlug: option.icon_slug || undefined,
    domain: option.domain || undefined,
    documentationUrl: option.documentation_url || undefined,
    priority: option.priority ?? 0,
    isLocal: option.local === true,
    defaultApiBase,
    requiresApiKey: !option.empty_api_key_allowed,
    createAllowed: option.create_allowed,
    defaultModelAllowed: option.default_model_allowed,
    supportsFetch: option.supports_fetch === true,
    defaultAuthMethod: option.default_auth_method || undefined,
    authMethodLocked: option.auth_method_locked,
    emptyApiKeyAllowed: option.empty_api_key_allowed,
    commonModels: option.common_models || [],
    aliases: option.aliases || [],
  }
}

function buildAliasMap(
  backendOptions?: ModelProviderOption[],
): Record<string, string> {
  const aliases: Record<string, string> = {}
  for (const option of backendOptions || []) {
    const key = normalizeProvider(option.id)
    if (!key) continue
    aliases[key] = option.id
    for (const alias of option.aliases || []) {
      const normalized = normalizeProvider(alias)
      if (normalized) {
        aliases[normalized] = option.id
      }
    }
  }
  return aliases
}

export function getProviderAliasMap(
  backendOptions?: ModelProviderOption[],
): Record<string, string> {
  return buildAliasMap(backendOptions)
}

export function getCanonicalProviderKey(
  provider?: string,
  backendOptions?: ModelProviderOption[],
): string {
  const normalized = normalizeProvider(provider)
  if (!normalized) return ""
  return getProviderAliasMap(backendOptions)[normalized] ?? normalized
}

export function getKnownProviderKeys(
  backendOptions?: ModelProviderOption[],
): Set<string> {
  return new Set(getProviderCatalog(backendOptions).map((p) => p.key))
}

export function getProviderCatalog(
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry[] {
  if (!backendOptions || backendOptions.length === 0) {
    return []
  }

  return [...backendOptions]
    .map(toCatalogEntry)
    .sort((a, b) => b.priority - a.priority)
}

export function getProviderCatalogMap(
  backendOptions?: ModelProviderOption[],
): Map<string, ProviderCatalogEntry> {
  return new Map(getProviderCatalog(backendOptions).map((p) => [p.key, p]))
}

export function getProviderCatalogEntry(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry | undefined {
  const key = getCanonicalProviderKey(provider, backendOptions)
  if (!key) return undefined
  return getProviderCatalogMap(backendOptions).get(key)
}

export function getProviderDefaultAPIBase(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): string {
  return getProviderCatalogEntry(provider, backendOptions)?.defaultApiBase ?? ""
}

export function getProviderDefaultAuthMethod(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): string {
  return getProviderCatalogEntry(provider, backendOptions)?.defaultAuthMethod ?? ""
}

export function isProviderAuthMethodLocked(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  return getProviderCatalogEntry(provider, backendOptions)?.authMethodLocked === true
}

export function providerSupportsFetch(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  const key = getCanonicalProviderKey(provider, backendOptions)
  if (!key) return false
  return getProviderCatalogMap(backendOptions).get(key)?.supportsFetch === true
}

/**
 * Find the closest known provider key by edit distance.
 * Returns the key if distance <= 2, otherwise undefined.
 */
export function findClosestProvider(
  input: string,
  backendOptions?: ModelProviderOption[],
): string | undefined {
  const lower = input.toLowerCase()
  let best: string | undefined
  let bestDist = 3

  for (const key of getKnownProviderKeys(backendOptions)) {
    const dist = editDistance(lower, key)
    if (dist < bestDist) {
      bestDist = dist
      best = key
    }
  }

  for (const alias of Object.keys(getProviderAliasMap(backendOptions))) {
    const dist = editDistance(lower, alias)
    if (dist < bestDist) {
      bestDist = dist
      best = getProviderAliasMap(backendOptions)[alias]
    }
  }
  return best
}

function editDistance(a: string, b: string): number {
  const m = a.length
  const n = b.length
  const dp: number[][] = Array.from({ length: m + 1 }, () =>
    new Array(n + 1).fill(0),
  )
  for (let i = 0; i <= m; i++) dp[i][0] = i
  for (let j = 0; j <= n; j++) dp[0][j] = j
  for (let i = 1; i <= m; i++) {
    for (let j = 1; j <= n; j++) {
      dp[i][j] =
        a[i - 1] === b[j - 1]
          ? dp[i - 1][j - 1]
          : 1 + Math.min(dp[i - 1][j], dp[i][j - 1], dp[i - 1][j - 1])
    }
  }
  return dp[m][n]
}

/** The provider ID reserved for arbitrary OpenAI-compatible endpoints. */
export const CUSTOM_OPENAI_PROVIDER = "custom-openai"

/**
 * Providers offered in the simplified "Add Provider" flow: every creatable
 * provider except the speech-only entries, which cannot drive a chat model.
 */
export function getSelectableProviders(
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry[] {
  return getProviderCatalog(backendOptions).filter(
    (provider) => provider.createAllowed && provider.category !== "speech",
  )
}

export function searchProviders(
  query: string,
  backendOptions?: ModelProviderOption[],
): ProviderCatalogEntry[] {
  const needle = query.trim().toLowerCase()
  const providers = getSelectableProviders(backendOptions)
  if (!needle) return providers
  return providers.filter(
    (provider) =>
      provider.key.includes(needle) ||
      provider.label.toLowerCase().includes(needle) ||
      provider.aliases.some((alias) => alias.toLowerCase().includes(needle)),
  )
}

/**
 * Whether the simplified flow can configure this provider with nothing but an
 * API key. Managed providers (AWS credential chain, Entra ID, OAuth, local CLI
 * bridges) and custom endpoints need more than a key, so they keep the full
 * form.
 */
export function isApiKeyOnlyProvider(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  const entry = getProviderCatalogEntry(provider, backendOptions)
  if (!entry) return false
  return entry.category === "cloud" && !!entry.defaultApiBase
}

/**
 * Whether the normal (non-advanced) form must show the Base URL field. Local
 * servers and custom endpoints are addressable at a host the user chooses, so
 * hiding the base URL there would make them unusable.
 */
export function requiresVisibleApiBase(
  provider: string | undefined,
  backendOptions?: ModelProviderOption[],
): boolean {
  const entry = getProviderCatalogEntry(provider, backendOptions)
  if (!entry) return true
  return entry.category === "local" || entry.category === "custom"
}

/**
 * Derive a unique model alias from the chosen model ID, so a normal user never
 * has to invent one. The model ID is kept intact as the alias when it is free;
 * otherwise a numeric suffix is appended.
 */
export function deriveModelAlias(
  modelId: string,
  existingNames: string[],
): string {
  const base = modelId.trim().replace(/^.*\//, "") || "model"
  const taken = new Set(existingNames.map((name) => name.trim()))
  if (!taken.has(base)) return base
  for (let suffix = 2; suffix < 1000; suffix++) {
    const candidate = `${base}-${suffix}`
    if (!taken.has(candidate)) return candidate
  }
  return `${base}-${Date.now()}`
}

/**
 * Recognize which preset an already-saved model belongs to. A stored entry that
 * uses a preset ID but overrides the base URL is reported as a preset with an
 * override, never rewritten back to the preset default.
 */
export function matchStoredProvider(
  provider: string | undefined,
  apiBase: string | undefined,
  backendOptions?: ModelProviderOption[],
): {
  entry?: ProviderCatalogEntry
  hasApiBaseOverride: boolean
} {
  const entry = getProviderCatalogEntry(provider, backendOptions)
  const stored = (apiBase || "").trim().replace(/\/+$/, "")
  if (!entry) {
    return { entry: undefined, hasApiBaseOverride: stored !== "" }
  }
  const preset = (entry.defaultApiBase || "").trim().replace(/\/+$/, "")
  return {
    entry,
    hasApiBaseOverride: stored !== "" && stored !== preset,
  }
}
