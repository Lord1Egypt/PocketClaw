import { launcherFetch } from "@/api/http"

// API client for the managed PocketClaw realtime channel.

interface PocketClawInfoResponse {
  ws_url: string
  enabled: boolean
  configured?: boolean
}

interface PocketClawSetupResponse {
  ws_url: string
  enabled: boolean
  configured?: boolean
  changed: boolean
}

const BASE_URL = ""

async function request<T>(path: string, options?: RequestInit): Promise<T> {
  const res = await launcherFetch(`${BASE_URL}${path}`, options)
  if (!res.ok) {
    throw new Error(`API error: ${res.status} ${res.statusText}`)
  }
  return res.json() as Promise<T>
}

export async function getPocketClawInfo(): Promise<PocketClawInfoResponse> {
  return request<PocketClawInfoResponse>("/api/pocketclaw/info")
}

export async function regenPocketClawToken(): Promise<PocketClawInfoResponse> {
  return request<PocketClawInfoResponse>("/api/pocketclaw/token", { method: "POST" })
}

export async function setupPocketClaw(): Promise<PocketClawSetupResponse> {
  return request<PocketClawSetupResponse>("/api/pocketclaw/setup", { method: "POST" })
}

export type { PocketClawInfoResponse, PocketClawSetupResponse }
