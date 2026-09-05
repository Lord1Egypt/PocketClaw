/**
 * Live discovery across configured providers.
 *
 * Two properties matter more than the happy path: one provider failing must not
 * take the others with it, and no API key may cross into the browser. The
 * second is structural — the request carries a `model_index`, and the backend
 * resolves the credential from stored config on its own side.
 */
import { renderHook, waitFor } from "@testing-library/react"
import { beforeEach, describe, expect, it, vi } from "vitest"

import type { ModelInfo, ModelProviderOption } from "@/api/models"
import { providerInstanceKey } from "@/lib/configured-model-source"

const fetchUpstreamModels = vi.fn()
vi.mock("@/api/models", () => ({
  fetchUpstreamModels: (...args: unknown[]) => fetchUpstreamModels(...args),
}))

const { useConfiguredModels } = await import("./use-configured-models")

function entry(overrides: Partial<ModelInfo> = {}): ModelInfo {
  return {
    index: 0,
    model_name: "m",
    provider: "openai",
    model: "m-id",
    api_key: "",
    enabled: true,
    available: true,
    status: "available",
    is_default: false,
    is_virtual: false,
    default_model_allowed: true,
    ...overrides,
  }
}

const models: ModelInfo[] = [
  entry({
    model_name: "opencode-primary",
    provider: "opencode",
    model: "deepseek-v4-flash-free",
    api_base: "https://opencode.example/v1",
  }),
  entry({
    model_name: "gemini-primary",
    provider: "gemini",
    model: "gemini-3.7-flash",
    api_base: "https://gemini.example/v1beta",
  }),
  entry({
    model_name: "azure-gpt5",
    provider: "azure",
    model: "deployment",
    status: "unconfigured",
    available: false,
  }),
]

const providerOptions: ModelProviderOption[] = [
  {
    id: "opencode",
    display_name: "OpenCode",
    default_api_base: "https://opencode.example/v1",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
  {
    id: "gemini",
    display_name: "Google Gemini",
    default_api_base: "https://gemini.example/v1beta",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
  {
    id: "azure",
    display_name: "Azure",
    default_api_base: "",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
  },
]

const OPENCODE = providerInstanceKey("opencode", "https://opencode.example/v1")
const GEMINI = providerInstanceKey("gemini", "https://gemini.example/v1beta")

describe("discovery", () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it("queries every configured provider and no unconfigured one", async () => {
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    renderHook(() =>
      useConfiguredModels({ models, providerOptions, autoDiscover: true }),
    )

    await waitFor(() => expect(fetchUpstreamModels).toHaveBeenCalledTimes(2))
    const queried = fetchUpstreamModels.mock.calls.map(
      (call) => (call[0] as { provider: string }).provider,
    )
    expect(queried.sort()).toEqual(["gemini", "opencode"])
    // Azure advertises supports_fetch but is unconfigured, so it is never hit.
    expect(queried).not.toContain("azure")
  })

  it("never puts a credential in the request", async () => {
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    const { result } = renderHook(() =>
      useConfiguredModels({ models, providerOptions, autoDiscover: true }),
    )
    await waitFor(() => expect(fetchUpstreamModels).toHaveBeenCalled())

    for (const call of fetchUpstreamModels.mock.calls) {
      const req = call[0] as { api_key?: string; model_index?: number }
      expect(req.api_key).toBeUndefined()
      // An index is what the backend resolves the stored key from.
      expect(typeof req.model_index).toBe("number")
    }
    expect(result.current.groups.length).toBe(2)
  })

  it("isolates one provider's failure from the others", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) => {
        if (provider === "gemini") throw new Error("502 Bad Gateway")
        return { models: [{ id: "discovered-one" }], total: 1 }
      },
    )

    const { result } = renderHook(() =>
      useConfiguredModels({ models, providerOptions, autoDiscover: true }),
    )

    await waitFor(() => {
      const gemini = result.current.groups.find((g) => g.key === GEMINI)!
      expect(gemini.discovery.phase).toBe("error")
    })

    const gemini = result.current.groups.find((g) => g.key === GEMINI)!
    const opencode = result.current.groups.find((g) => g.key === OPENCODE)!

    // The working provider still got its discovered model...
    expect(opencode.discovery.phase).toBe("ready")
    expect(opencode.models.map((m) => m.model)).toContain("discovered-one")
    // ...and the failing one kept its configured entry rather than emptying.
    expect(gemini.models.map((m) => m.model)).toEqual(["gemini-3.7-flash"])
  })

  it("retries a single provider without disturbing the rest", async () => {
    fetchUpstreamModels.mockImplementation(
      async ({ provider }: { provider: string }) => {
        if (provider === "gemini") throw new Error("502")
        return { models: [{ id: "opencode-model" }], total: 1 }
      },
    )
    const { result } = renderHook(() =>
      useConfiguredModels({ models, providerOptions, autoDiscover: true }),
    )
    await waitFor(() =>
      expect(
        result.current.groups.find((g) => g.key === GEMINI)!.discovery.phase,
      ).toBe("error"),
    )

    fetchUpstreamModels.mockResolvedValue({
      models: [{ id: "gemini-3.7-pro" }],
      total: 1,
    })
    await result.current.discover(GEMINI)

    await waitFor(() => {
      const gemini = result.current.groups.find((g) => g.key === GEMINI)!
      expect(gemini.discovery.phase).toBe("ready")
      expect(gemini.models.map((m) => m.model)).toContain("gemini-3.7-pro")
    })
    // The other provider's result was not refetched or lost.
    const opencode = result.current.groups.find((g) => g.key === OPENCODE)!
    expect(opencode.models.map((m) => m.model)).toContain("opencode-model")
  })

  it("does not discover until asked when auto-discovery is off", async () => {
    fetchUpstreamModels.mockResolvedValue({ models: [], total: 0 })
    const { result } = renderHook(() =>
      useConfiguredModels({ models, providerOptions }),
    )
    await waitFor(() => expect(result.current.groups.length).toBe(2))
    expect(fetchUpstreamModels).not.toHaveBeenCalled()

    await result.current.discoverAll()
    await waitFor(() => expect(fetchUpstreamModels).toHaveBeenCalledTimes(2))
  })

  it("never falls back to the global provider catalog on failure", async () => {
    fetchUpstreamModels.mockRejectedValue(new Error("offline"))
    const { result } = renderHook(() =>
      useConfiguredModels({ models, providerOptions, autoDiscover: true }),
    )
    await waitFor(() =>
      expect(
        result.current.groups.every((g) => g.discovery.phase === "error"),
      ).toBe(true),
    )
    // Still exactly the two configured providers — azure and the rest of the
    // shipped templates do not appear as a consolation list.
    expect(result.current.groups.map((g) => g.provider).sort()).toEqual([
      "gemini",
      "opencode",
    ])
  })
})
