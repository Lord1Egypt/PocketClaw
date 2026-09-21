import { launcherFetch } from "@/api/http"

// API client for provider-level management.
//
// A provider is not a stored record: the configuration schema has no provider
// object, only a flat `model_list` whose entries each carry a provider name, an
// endpoint and a credential. These endpoints manage the set of models sharing
// one provider key, which is what "the provider" means in this system.

/**
 * What the models of one provider agree on about their credential.
 *
 * `mixed` is reported rather than resolved. Per-model overrides are legitimate,
 * and presenting one of several keys as "the provider key" would let a rotation
 * update one model while its siblings kept an old credential.
 */
export type ProviderCredentialState = "unset" | "shared" | "mixed"

export interface ProviderModelSummary {
  index: number
  model_name: string
  model: string
  enabled: boolean
  is_default: boolean
  has_api_key: boolean
}

export interface ProviderInfo {
  provider: string
  model_count: number
  credential_state: ProviderCredentialState
  /** Masked. The raw credential never leaves the backend. */
  api_key_masked?: string
  /** The endpoint every model agrees on; absent when `api_base_mixed`. */
  api_base?: string
  api_base_mixed: boolean
  auth_method?: string
  holds_default_model: boolean
  /** Configuration sites that currently name a model of this provider. */
  reference_sites?: string[]
  models: ProviderModelSummary[]
}

export interface ProviderUpdateRequest {
  /** Omit to leave the stored credential alone. */
  api_key?: string
  api_base?: string
  proxy?: string
}

export interface ProviderUpdateResponse {
  status: string
  provider: string
  models_updated: number
  credential_change: boolean
}

export interface ProviderDeleteResponse {
  status: string
  provider: string
  models_removed: string[]
  cleared_sites: string[]
  models_remaining: number
}

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(path, options)
  if (!res.ok) {
    let detail = ""
    try {
      detail = await res.text()
    } catch {
      // The status line is the only thing left to report.
    }
    throw new Error(detail || `API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getProviders(): Promise<ProviderInfo[]> {
  const body = await request<{ providers: ProviderInfo[] }>("/api/providers")
  return body.providers ?? []
}

export async function getProvider(provider: string): Promise<ProviderInfo> {
  return request<ProviderInfo>(`/api/providers/${encodeURIComponent(provider)}`)
}

/**
 * Applies a provider-scoped change to every model of the provider.
 *
 * Sending `api_key` rotates the credential for all of them, which is the point:
 * a rotation that reached only the edited model would leave its siblings using
 * the key the user just replaced.
 */
export async function updateProvider(
  provider: string,
  update: ProviderUpdateRequest,
): Promise<ProviderUpdateResponse> {
  return request<ProviderUpdateResponse>(
    `/api/providers/${encodeURIComponent(provider)}`,
    {
      method: "PUT",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(update),
    },
  )
}

/**
 * Removes a provider, its models, and every reference to them.
 *
 * The response reports what was removed so the caller can confirm the delete
 * reconciled rather than assume it.
 */
export async function deleteProvider(
  provider: string,
): Promise<ProviderDeleteResponse> {
  return request<ProviderDeleteResponse>(
    `/api/providers/${encodeURIComponent(provider)}`,
    { method: "DELETE" },
  )
}
