import { describe, expect, it } from "vitest"

import type { ModelProviderOption } from "@/api/models"

import {
  CUSTOM_OPENAI_PROVIDER,
  deriveModelAlias,
  getCanonicalProviderKey,
  getProviderCatalogEntry,
  getProviderDefaultAPIBase,
  getSelectableProviders,
  isApiKeyOnlyProvider,
  matchStoredProvider,
  providerSupportsFetch,
  requiresVisibleApiBase,
  searchProviders,
} from "./provider-registry"

// A trimmed stand-in for the backend payload. Provider semantics always come
// from the backend, so the fixture mirrors its shape rather than inventing one.
const options: ModelProviderOption[] = [
  {
    id: "openai",
    display_name: "OpenAI",
    category: "cloud",
    default_api_base: "https://api.openai.com/v1",
    documentation_url: "https://platform.openai.com/api-keys",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    priority: 100,
    aliases: ["gpt"],
  },
  {
    id: "gemini",
    display_name: "Google Gemini",
    category: "cloud",
    default_api_base: "https://generativelanguage.googleapis.com/v1beta",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    priority: 90,
    aliases: ["google"],
  },
  {
    id: "ollama",
    display_name: "Ollama",
    category: "local",
    default_api_base: "http://localhost:11434/v1",
    empty_api_key_allowed: true,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    local: true,
    priority: 50,
  },
  {
    id: CUSTOM_OPENAI_PROVIDER,
    display_name: "Custom OpenAI-Compatible",
    category: "custom",
    default_api_base: "",
    empty_api_key_allowed: true,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    priority: 30,
    aliases: ["custom", "openai-compatible"],
  },
  {
    id: "opencode_zen",
    display_name: "OpenCode Zen",
    category: "cloud",
    default_api_base: "https://opencode.ai/zen/v1",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    priority: 72,
    aliases: ["opencode-zen", "zen"],
  },
  {
    id: "opencode_go",
    display_name: "OpenCode Go",
    category: "cloud",
    default_api_base: "https://opencode.ai/zen/go/v1",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: true,
    supports_fetch: true,
    priority: 71,
    aliases: ["opencode-go"],
  },
  {
    id: "bedrock",
    display_name: "AWS Bedrock",
    category: "managed",
    default_api_base: "",
    empty_api_key_allowed: true,
    create_allowed: true,
    default_model_allowed: true,
    priority: 60,
  },
  {
    id: "elevenlabs",
    display_name: "ElevenLabs ASR",
    category: "speech",
    default_api_base: "https://api.elevenlabs.io",
    empty_api_key_allowed: false,
    create_allowed: true,
    default_model_allowed: false,
    priority: 47,
  },
]

describe("provider catalog lookup", () => {
  it("finds a known provider and its metadata", () => {
    const entry = getProviderCatalogEntry("openai", options)
    expect(entry?.key).toBe("openai")
    expect(entry?.label).toBe("OpenAI")
    expect(entry?.category).toBe("cloud")
    expect(entry?.defaultApiBase).toBe("https://api.openai.com/v1")
    expect(entry?.documentationUrl).toBe("https://platform.openai.com/api-keys")
  })

  it("resolves aliases to the canonical id, never the display label", () => {
    expect(getCanonicalProviderKey("GPT", options)).toBe("openai")
    expect(getCanonicalProviderKey("Google", options)).toBe("gemini")
    expect(getCanonicalProviderKey("custom", options)).toBe(
      CUSTOM_OPENAI_PROVIDER,
    )
  })

  it("returns nothing for an unknown provider", () => {
    expect(getProviderCatalogEntry("not-a-provider", options)).toBeUndefined()
    expect(getProviderDefaultAPIBase("not-a-provider", options)).toBe("")
  })

  it("returns an empty catalog when the backend payload is missing", () => {
    expect(getSelectableProviders(undefined)).toEqual([])
    expect(getSelectableProviders([])).toEqual([])
  })
})

describe("api key requirements", () => {
  it("marks cloud providers as requiring a key", () => {
    expect(getProviderCatalogEntry("openai", options)?.requiresApiKey).toBe(true)
    expect(isApiKeyOnlyProvider("openai", options)).toBe(true)
  })

  it("marks local and custom providers as keyless", () => {
    expect(getProviderCatalogEntry("ollama", options)?.requiresApiKey).toBe(
      false,
    )
    expect(
      getProviderCatalogEntry(CUSTOM_OPENAI_PROVIDER, options)?.requiresApiKey,
    ).toBe(false)
  })

  it("keeps managed and custom providers out of the api-key-only flow", () => {
    expect(isApiKeyOnlyProvider("bedrock", options)).toBe(false)
    expect(isApiKeyOnlyProvider(CUSTOM_OPENAI_PROVIDER, options)).toBe(false)
    expect(isApiKeyOnlyProvider("ollama", options)).toBe(false)
  })
})

describe("model discovery flag", () => {
  it("reports the backend supports_fetch flag", () => {
    expect(providerSupportsFetch("gemini", options)).toBe(true)
    expect(providerSupportsFetch("openai", options)).toBe(true)
    expect(providerSupportsFetch("bedrock", options)).toBe(false)
    expect(providerSupportsFetch("not-a-provider", options)).toBe(false)
  })
})

describe("base url visibility", () => {
  it("hides the base url for cloud presets and shows it where the host is the user's choice", () => {
    expect(requiresVisibleApiBase("openai", options)).toBe(false)
    expect(requiresVisibleApiBase("gemini", options)).toBe(false)
    expect(requiresVisibleApiBase("ollama", options)).toBe(true)
    expect(requiresVisibleApiBase(CUSTOM_OPENAI_PROVIDER, options)).toBe(true)
  })

  it("shows the base url for an unrecognized provider rather than hiding it", () => {
    expect(requiresVisibleApiBase("not-a-provider", options)).toBe(true)
  })
})

describe("picker listing", () => {
  it("excludes speech-only providers from the chat provider picker", () => {
    const keys = getSelectableProviders(options).map((p) => p.key)
    expect(keys).not.toContain("elevenlabs")
    expect(keys).toContain("openai")
  })

  it("sorts by descending priority", () => {
    const keys = getSelectableProviders(options).map((p) => p.key)
    expect(keys[0]).toBe("openai")
  })

  it("searches by id, label, and alias", () => {
    expect(searchProviders("gpt", options).map((p) => p.key)).toEqual(["openai"])
    expect(searchProviders("Gemini", options).map((p) => p.key)).toEqual([
      "gemini",
    ])
    expect(searchProviders("openai-compatible", options).map((p) => p.key)).toEqual(
      [CUSTOM_OPENAI_PROVIDER],
    )
    expect(searchProviders("zzzz", options)).toEqual([])
  })
})

describe("model alias derivation", () => {
  it("uses the model id when it is free", () => {
    expect(deriveModelAlias("gpt-5.4", [])).toBe("gpt-5.4")
  })

  it("strips a provider prefix from a namespaced model id", () => {
    expect(deriveModelAlias("openai/gpt-oss-120b", [])).toBe("gpt-oss-120b")
  })

  it("suffixes to avoid colliding with an existing alias", () => {
    expect(deriveModelAlias("gpt-5.4", ["gpt-5.4"])).toBe("gpt-5.4-2")
    expect(deriveModelAlias("gpt-5.4", ["gpt-5.4", "gpt-5.4-2"])).toBe(
      "gpt-5.4-3",
    )
  })

  it("falls back to a usable alias for an empty model id", () => {
    expect(deriveModelAlias("", [])).toBe("model")
  })
})

describe("existing configuration compatibility", () => {
  it("recognizes a stored entry that matches its preset default", () => {
    const result = matchStoredProvider(
      "openai",
      "https://api.openai.com/v1",
      options,
    )
    expect(result.entry?.key).toBe("openai")
    expect(result.hasApiBaseOverride).toBe(false)
  })

  it("treats a trailing slash as the same endpoint, not an override", () => {
    expect(
      matchStoredProvider("openai", "https://api.openai.com/v1/", options)
        .hasApiBaseOverride,
    ).toBe(false)
  })

  it("reports a genuinely custom base url as an override", () => {
    const result = matchStoredProvider(
      "openai",
      "https://gateway.internal.example/v1",
      options,
    )
    expect(result.entry?.key).toBe("openai")
    expect(result.hasApiBaseOverride).toBe(true)
  })

  it("treats an empty stored base as inheriting the preset default", () => {
    expect(
      matchStoredProvider("openai", "", options).hasApiBaseOverride,
    ).toBe(false)
    expect(
      matchStoredProvider("openai", undefined, options).hasApiBaseOverride,
    ).toBe(false)
  })

  it("keeps an unknown provider's custom base visible as an override", () => {
    const result = matchStoredProvider(
      "some-retired-provider",
      "https://legacy.example/v1",
      options,
    )
    expect(result.entry).toBeUndefined()
    expect(result.hasApiBaseOverride).toBe(true)
  })
})

describe("opencode presets", () => {
  it("exposes both gateways with their official base URLs", () => {
    expect(getProviderDefaultAPIBase("opencode_zen", options)).toBe(
      "https://opencode.ai/zen/v1",
    )
    expect(getProviderDefaultAPIBase("opencode_go", options)).toBe(
      "https://opencode.ai/zen/go/v1",
    )
  })

  it("keeps the base URL out of the normal flow and asks only for a key", () => {
    for (const id of ["opencode_zen", "opencode_go"]) {
      expect(requiresVisibleApiBase(id, options)).toBe(false)
      expect(isApiKeyOnlyProvider(id, options)).toBe(true)
      expect(getProviderCatalogEntry(id, options)?.requiresApiKey).toBe(true)
    }
  })

  it("offers model discovery for both", () => {
    expect(providerSupportsFetch("opencode_zen", options)).toBe(true)
    expect(providerSupportsFetch("opencode_go", options)).toBe(true)
  })

  it("resolves hyphenated aliases to the underscored canonical ids", () => {
    expect(getCanonicalProviderKey("opencode-zen", options)).toBe("opencode_zen")
    expect(getCanonicalProviderKey("opencode-go", options)).toBe("opencode_go")
  })

  it("is findable in the picker by name and by alias", () => {
    expect(searchProviders("opencode", options).map((p) => p.key)).toEqual([
      "opencode_zen",
      "opencode_go",
    ])
    expect(searchProviders("zen", options).map((p) => p.key)).toContain(
      "opencode_zen",
    )
  })

  it("derives a clean alias from a namespaced opencode model id", () => {
    expect(deriveModelAlias("opencode-go/kimi-k3", [])).toBe("kimi-k3")
  })
})
