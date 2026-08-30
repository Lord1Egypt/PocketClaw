import { launcherFetch } from "@/api/http"

// API client for gateway process management.

interface GatewayStatusResponse {
  gateway_status: "running" | "starting" | "restarting" | "stopped" | "error"
  gateway_start_allowed?: boolean
  gateway_start_reason?: string
  gateway_restart_required?: boolean
  pid?: number
  boot_default_model?: string
  config_default_model?: string
  [key: string]: unknown
}

interface GatewayLogsResponse {
  logs?: string[]
  log_total?: number
  log_run_id?: number
}

interface GatewayActionResponse {
  status: string
  pid?: number
  log_total?: number
  log_run_id?: number
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getGatewayStatus(): Promise<GatewayStatusResponse> {
  return request<GatewayStatusResponse>("/api/gateway/status")
}

export async function getGatewayLogs(options?: {
  log_offset?: number
  log_run_id?: number
}): Promise<GatewayLogsResponse> {
  const params = new URLSearchParams()
  if (options?.log_offset !== undefined) {
    params.set("log_offset", options.log_offset.toString())
  }
  if (options?.log_run_id !== undefined) {
    params.set("log_run_id", options.log_run_id.toString())
  }
  const queryString = params.toString() ? `?${params.toString()}` : ""
  return request<GatewayLogsResponse>(`/api/gateway/logs${queryString}`)
}

export async function startGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/start", {
    method: "POST",
  })
}

export async function stopGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/stop", {
    method: "POST",
  })
}

export async function restartGateway(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/restart", {
    method: "POST",
  })
}

/**
 * Restarts the gateway so a saved configuration change takes effect.
 *
 * Distinct from restartGateway: this waits for in-flight turns to finish first,
 * so saving settings never cuts off an answer that is still being produced, and
 * concurrent saves coalesce into one restart.
 */
export interface ApplyGatewayConfigResponse {
  /**
   * "ok" when the gateway restarted, "saved_not_applied" when it was not safe
   * to restart. The second is not a failure: the configuration is persisted, it
   * simply is not live yet.
   */
  status: string
  /** Why the restart was withheld: busy_timeout or unverified. */
  outcome?: string
  message?: string
  deferred?: boolean
  pid?: number
}

export async function applyGatewayConfig(
  reason: string,
): Promise<ApplyGatewayConfigResponse> {
  return request<ApplyGatewayConfigResponse>("/api/gateway/apply-config", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ reason }),
  })
}

export async function clearGatewayLogs(): Promise<GatewayActionResponse> {
  return request<GatewayActionResponse>("/api/gateway/logs/clear", {
    method: "POST",
  })
}

export type {
  GatewayStatusResponse,
  GatewayLogsResponse,
  GatewayActionResponse,
}
