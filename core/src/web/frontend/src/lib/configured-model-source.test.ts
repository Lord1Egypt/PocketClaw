/**
 * What a model selector is allowed to offer.
 *
 * The shipped default config seeds `model_list` with thirty keyless provider
 * templates. A user who had configured OpenCode and Gemini was still offered
 * Azure, Cerebras, Anthropic, Groq, Ollama and Volcengine in the Fallback
 * picker, because that picker filtered on virtual/duplicate/self and never on
 * whether the provider was configured at all.
 */
import { describe, expect, it } from "vitest"

import type { ModelInfo, ModelProviderOption } from "@/api/models"
import {
  buildConfiguredProviderGroups,
  isConfiguredEntry,
  modelKey,
  normalizeApiBase,
  providerInstanceKey,
  stripProviderPrefix,
} from "./configured-model-source"

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

/// The shape the shipped default config actually has: two providers the user
/// configured, and a pile of keyless templates they never touched.
const SHIPPED: ModelInfo[] = [
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
    model: "my-gpt5-deployment",
    status: "unconfigured",
    available: false,
  }),
  entry({
    model_name: "cerebras-llama-3.3-70b",
    provider: "cerebras",
    model: "llama-3.3-70b",
    status: "unconfigured",
    available: false,
  }),
  entry({
    model_name: "llama3",
    provider: "ollama",
    model: "llama3",
    status: "unconfigured",
    available: false,
  }),
  entry({
    model_name: "ark-code-latest",
    provider: "volcengine",
    model: "ark-code",
    status: "unconfigured",
    available: false,
  }),
]

const OPTIONS: ModelProviderOption[] = [
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
    supports_fetch: false,
  },
]

describe("only configured providers can contribute", () => {
  it("offers the two providers the user configured and nothing else", () => {
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
    })
    expect(groups.map((group) => group.provider).sort()).toEqual([
      "gemini",
      "opencode",
    ])
  })

  it.each(["azure", "cerebras", "ollama", "volcengine"])(
    "never surfaces the unconfigured %s template",
    (provider) => {
      const groups = buildConfiguredProviderGroups({
        models: SHIPPED,
        providerOptions: OPTIONS,
      })
      expect(groups.some((group) => group.provider === provider)).toBe(false)
    },
  )

  it("reads configured-ness from the backend's own status field", () => {
    expect(isConfiguredEntry(entry({ status: "unconfigured" }))).toBe(false)
    expect(isConfiguredEntry(entry({ status: "available" }))).toBe(true)
    // Unreachable is configured but down. Hiding it would make a temporary
    // outage look like a deleted model.
    expect(isConfiguredEntry(entry({ status: "unreachable" }))).toBe(true)
  })

  it("excludes entries that cannot hold a chat role", () => {
    const groups = buildConfiguredProviderGroups({
      models: [
        entry({ model_name: "virtual", is_virtual: true }),
        entry({ model_name: "not-chat", default_model_allowed: false }),
      ],
    })
    expect(groups).toEqual([])
  })

  it("returns nothing at all when no provider is configured", () => {
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED.filter((m) => m.status === "unconfigured"),
      providerOptions: OPTIONS,
    })
    expect(groups).toEqual([])
  })
})

describe("discovery is merged per provider instance", () => {
  it("adds discovered ids under the provider that reported them", () => {
    const key = providerInstanceKey("opencode", "https://opencode.example/v1")
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
      discovered: {
        [key]: ["deepseek-v4-flash-vision-exp", "qwen3-coder-480b"],
      },
    })
    const opencode = groups.find((group) => group.provider === "opencode")!
    expect(opencode.models.map((m) => m.model)).toEqual([
      "deepseek-v4-flash-free",
      "deepseek-v4-flash-vision-exp",
      "qwen3-coder-480b",
    ])
    expect(opencode.models[0].origin).toBe("configured")
    expect(opencode.models[1].origin).toBe("discovered")
  })

  it("does not list a model twice when discovery repeats a configured one", () => {
    const key = providerInstanceKey("opencode", "https://opencode.example/v1")
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
      discovered: { [key]: ["deepseek-v4-flash-free"] },
    })
    const opencode = groups.find((group) => group.provider === "opencode")!
    expect(opencode.models).toHaveLength(1)
    // The configured one wins: it is the one that already has an alias a role
    // can reference.
    expect(opencode.models[0].origin).toBe("configured")
    expect(opencode.models[0].modelName).toBe("opencode-primary")
  })

  it("keeps a failing provider's own models and leaves the others alone", () => {
    const openKey = providerInstanceKey(
      "opencode",
      "https://opencode.example/v1",
    )
    const geminiKey = providerInstanceKey("gemini", "https://gemini.example/v1beta")
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
      discovered: { [openKey]: ["extra-model"] },
      discoveryState: {
        [openKey]: { phase: "ready", fetchedAt: 1 },
        [geminiKey]: { phase: "error", message: "502" },
      },
    })
    const gemini = groups.find((group) => group.provider === "gemini")!
    const opencode = groups.find((group) => group.provider === "opencode")!

    expect(gemini.discovery.phase).toBe("error")
    // Its configured entry survives the failure — the list is never replaced.
    expect(gemini.models.map((m) => m.model)).toEqual(["gemini-3.7-flash"])
    expect(opencode.models).toHaveLength(2)
  })

  it("marks a provider with no listing endpoint as unsupported", () => {
    const groups = buildConfiguredProviderGroups({
      models: [
        entry({
          model_name: "azure-real",
          provider: "azure",
          model: "gpt5",
          api_base: "https://azure.example/v1",
        }),
      ],
      providerOptions: OPTIONS,
    })
    expect(groups[0].supportsFetch).toBe(false)
    expect(groups[0].discovery.phase).toBe("unsupported")
    // It still offers what the user explicitly configured.
    expect(groups[0].models.map((m) => m.model)).toEqual(["gpt5"])
  })
})

describe("model identity is provider-scoped", () => {
  it("treats the same id from two providers as two models", () => {
    const models = [
      entry({
        model_name: "a-chat",
        provider: "openai",
        model: "deepseek-chat",
        api_base: "https://provider-a.example/v1",
      }),
      entry({
        model_name: "b-chat",
        provider: "openai",
        model: "deepseek-chat",
        api_base: "https://provider-b.example/v1",
      }),
    ]
    const groups = buildConfiguredProviderGroups({ models })
    expect(groups).toHaveLength(2)
    expect(groups[0].models[0].key).not.toBe(groups[1].models[0].key)
  })

  it("does not collapse a model id onto another provider's entry", () => {
    const a = modelKey("openai", "https://a.example/v1", "deepseek-chat")
    const b = modelKey("openai", "https://b.example/v1", "deepseek-chat")
    expect(a).not.toBe(b)
  })

  it("treats cosmetic endpoint differences as the same instance", () => {
    expect(normalizeApiBase("https://API.Example.com/v1/")).toBe(
      normalizeApiBase("https://api.example.com/v1"),
    )
    expect(normalizeApiBase("https://api.example.com:443/v1")).toBe(
      normalizeApiBase("https://api.example.com/v1"),
    )
    expect(normalizeApiBase(undefined)).toBe("")
  })

  it("survives an api_base that is not a URL", () => {
    expect(normalizeApiBase("not a url")).toBe("not a url")
  })

  it("strips a provider prefix so ids compare against stored bare ids", () => {
    expect(stripProviderPrefix("gemini/gemini-3.7-pro", "gemini")).toBe(
      "gemini-3.7-pro",
    )
    expect(stripProviderPrefix("gemini-3.7-pro", "gemini")).toBe("gemini-3.7-pro")
    // A slash that is part of the id, not a provider prefix, is left alone.
    expect(stripProviderPrefix("Qwen/Qwen3-235B", "modelscope")).toBe(
      "Qwen/Qwen3-235B",
    )
  })
})

describe("presentation", () => {
  it("names a group by the provider's display name when there is one", () => {
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
    })
    expect(groups.map((group) => group.label)).toEqual([
      "Google Gemini",
      "OpenCode",
    ])
  })

  it("falls back to the raw provider id when there is no display name", () => {
    const groups = buildConfiguredProviderGroups({
      models: [entry({ provider: "mystery" })],
    })
    expect(groups[0].label).toBe("mystery")
  })

  it("marks the current default so a selector can show it", () => {
    const groups = buildConfiguredProviderGroups({
      models: SHIPPED,
      providerOptions: OPTIONS,
      defaultModelName: "gemini-primary",
    })
    const gemini = groups.find((group) => group.provider === "gemini")!
    expect(gemini.models[0].isDefault).toBe(true)
  })
})
