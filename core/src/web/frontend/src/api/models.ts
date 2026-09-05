import { launcherFetch } from "@/api/http"
import { refreshGatewayState } from "@/store/gateway"

// API client for model list management.

export interface ModelInfo {
  index: number
  model_name: string
  provider?: string
  model: string
  api_base?: string
  api_key: string
  proxy?: string
  auth_method?: string
  // Advanced fields
  connect_mode?: string
  workspace?: string
  rpm?: number
  max_tokens_field?: string
  request_timeout?: number
  thinking_level?: string
  tool_schema_transform?: string
  streaming?: {
    enabled?: boolean
  }
  extra_body?: Record<string, unknown>
  custom_headers?: Record<string, string>
  // Meta
  enabled: boolean
  available: boolean
  status: "available" | "unconfigured" | "unreachable"
  is_default: boolean
  is_virtual: boolean
  default_model_allowed?: boolean
}

export interface ModelProviderOption {
  id: string
  display_name?: string
  category?: string
  icon_slug?: string
  domain?: string
  documentation_url?: string
  default_api_base: string
  empty_api_key_allowed: boolean
  create_allowed: boolean
  default_model_allowed: boolean
  supports_fetch?: boolean
  default_auth_method?: string
  auth_method_locked?: boolean
  local?: boolean
  priority?: number
  common_models?: string[]
  aliases?: string[]
}

interface ModelsListResponse {
  models: ModelInfo[]
  total: number
  default_model: string
  /**
   * Ordered chain tried when the default model is unavailable, as model_name
   * references into `models`. Always present, so an empty array means "none
   * configured" rather than "unsupported by this build".
   */
  model_fallbacks: string[]
  /**
   * The dedicated model for turns that carry an image. Empty means no
   * dedicated model is configured and image turns go to the default, which is
   * what every install did before this field existed.
   */
  image_model: string
  provider_options: ModelProviderOption[]
}

interface ModelActionResponse {
  status: string
  index?: number
  default_model?: string
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    let detail = ""
    try {
      detail = await res.text()
    } catch {
      // ignore
    }
    throw new Error(detail || `API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getModels(): Promise<ModelsListResponse> {
  return request<ModelsListResponse>("/api/models")
}

export async function addModel(
  model: Partial<ModelInfo>,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>("/api/models", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function updateModel(
  index: number,
  model: Partial<ModelInfo>,
): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "PUT",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(model),
  })
}

export async function deleteModel(index: number): Promise<ModelActionResponse> {
  return request<ModelActionResponse>(`/api/models/${index}`, {
    method: "DELETE",
  })
}

export async function setDefaultModel(
  modelName: string,
): Promise<ModelActionResponse> {
  const response = await request<ModelActionResponse>("/api/models/default", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ model_name: modelName }),
  })

  await refreshGatewayState()
  return response
}

/**
 * Replaces the ordered fallback chain.
 *
 * Each entry is a reference to a configured model, not a copy of one: the
 * referenced model keeps its own provider, credentials, base URL and headers.
 * The primary's API key is never sent to a fallback provider.
 */
export async function setModelFallbacks(
  fallbacks: string[],
): Promise<{ status: string; fallbacks: string[] }> {
  const response = await request<{ status: string; fallbacks: string[] }>(
    "/api/models/fallbacks",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ fallbacks }),
    },
  )

  await refreshGatewayState()
  return response
}

export interface TestModelResponse {
  success: boolean
  latency_ms: number
  status: string
  error?: string
}

export async function testModel(index: number): Promise<TestModelResponse> {
  return request<TestModelResponse>(`/api/models/${index}/test`, {
    method: "POST",
  })
}

export interface TestModelInlineRequest {
  provider: string
  model: string
  api_base?: string
  api_key?: string
  auth_method?: string
  model_index?: number
}

export async function testModelInline(
  params: TestModelInlineRequest,
): Promise<TestModelResponse> {
  return request<TestModelResponse>("/api/models/test-inline", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(params),
  })
}

export interface UpstreamModel {
  id: string
  owned_by?: string
  extra?: Record<string, unknown>
}

export interface FetchModelsRequest {
  provider: string
  api_key?: string
  api_base?: string
  model_index?: number
}

export interface FetchModelsResponse {
  models: UpstreamModel[]
  total: number
}

export async function fetchUpstreamModels(
  req: FetchModelsRequest,
): Promise<FetchModelsResponse> {
  return request<FetchModelsResponse>("/api/models/fetch", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
}

// --- Model Catalog API ---

export interface CatalogModel {
  id: string
  owned_by?: string
  extra?: Record<string, unknown>
}

export interface CatalogEntry {
  id: string
  provider: string
  api_base: string
  api_key_mask: string
  models: CatalogModel[]
  fetched_at: string
}

interface CatalogListResponse {
  entries: CatalogEntry[]
  total: number
}

export async function getCatalogs(): Promise<CatalogListResponse> {
  return request<CatalogListResponse>("/api/models/catalog")
}

export async function deleteCatalog(id: string): Promise<void> {
  await request<Record<string, never>>(
    `/api/models/catalog/${encodeURIComponent(id)}`,
    {
      method: "DELETE",
    },
  )
}

/** A role a materialized model can be given in the same operation. */
export type ModelRole = "" | "default" | "vision" | "fallback"

export interface MaterializeModelRequest {
  /**
   * The configured model_list entry whose provider identity and credential the
   * new entry inherits. An index, never a key: the secret is read from stored
   * config on the backend and never crosses this boundary.
   */
  source_index: number
  model: string
  role: ModelRole
  model_name?: string
}

export interface MaterializeModelResponse {
  status: string
  model_name: string
  index: number
  /** False when an entry for this provider instance and id already existed. */
  created: boolean
  role: ModelRole
  default_model: string
  image_model: string
  model_fallbacks: string[]
}

/**
 * Makes a discovered model routable and applies a role to it, in one call.
 *
 * Default and fallback references must name a model_list entry — that invariant
 * is enforced by the backend and is not relaxed. Both halves happen in one
 * request so a rejected role cannot leave a stray entry behind.
 */
export async function materializeModel(
  req: MaterializeModelRequest,
): Promise<MaterializeModelResponse> {
  return request<MaterializeModelResponse>("/api/models/materialize", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(req),
  })
}

/**
 * Points the dedicated vision model at an existing entry, or clears it.
 *
 * An empty name is a real state, not an error: with no vision model configured
 * an image turn goes to the default model, exactly as it did before this
 * setting existed. Selecting a *discovered* model instead goes through
 * `materializeModel` with role "vision", which creates the entry and assigns
 * the role in one write.
 */
export async function setVisionModel(
  modelName: string,
): Promise<{ status: string; image_model: string }> {
  const res = await request<{ status: string; image_model: string }>(
    "/api/models/vision",
    {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ model_name: modelName }),
    },
  )
  await refreshGatewayState()
  return res
}

export type { ModelsListResponse, ModelActionResponse }
